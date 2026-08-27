package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

type DiskStore struct {
	root     string
	snapDir  string
	logPath  string
	mu       sync.RWMutex
	projects map[string]SnapshotEnvelope
	closed   bool
}

// ErrClosed 表示存储生命周期已经结束。
var ErrClosed = errors.New("存储已关闭")

func Open(root string) (*DiskStore, error) {
	if root == "" {
		return nil, errors.New("存储目录不能为空")
	}
	s := &DiskStore{
		root: root, snapDir: filepath.Join(root, "snapshots"), logPath: filepath.Join(root, "operations.jsonl"),
		projects: make(map[string]SnapshotEnvelope),
	}
	if err := os.MkdirAll(s.snapDir, 0o750); err != nil {
		return nil, fmt.Errorf("创建存储目录: %w", err)
	}
	if err := s.loadSnapshots(); err != nil {
		return nil, err
	}
	if err := s.replayConfirmedLog(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *DiskStore) Create(ctx context.Context, aggregate domain.Aggregate, key, operation string) (domain.Aggregate, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.Aggregate{}, false, err
	}
	if key == "" {
		return domain.Aggregate{}, false, domain.NewRuleError("idempotencyKey", "幂等键不能为空")
	}
	id := aggregate.Production.ID
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return domain.Aggregate{}, false, ErrClosed
	}
	if existing, ok := s.projects[id]; ok {
		if saved, found := existing.Idempotency[key]; found && saved.Operation == operation {
			return domain.CloneAggregate(saved.Aggregate), true, nil
		}
		return domain.Aggregate{}, false, &domain.ConflictError{Expected: 0, Actual: existing.Aggregate.Production.Revision}
	}
	env := SnapshotEnvelope{Aggregate: domain.CloneAggregate(aggregate), Idempotency: map[string]SavedResult{}}
	env.Idempotency[key] = SavedResult{Operation: operation, Aggregate: domain.CloneAggregate(aggregate)}
	if err := s.commitLocked(id, env, operation); err != nil {
		return domain.Aggregate{}, false, err
	}
	s.projects[id] = env
	return domain.CloneAggregate(aggregate), false, nil
}

func (s *DiskStore) Get(ctx context.Context, id string) (domain.Aggregate, error) {
	if err := ctx.Err(); err != nil {
		return domain.Aggregate{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	env, ok := s.projects[id]
	if !ok {
		return domain.Aggregate{}, &domain.NotFoundError{Resource: "制作项目"}
	}
	return domain.CloneAggregate(env.Aggregate), nil
}

func (s *DiskStore) Mutate(ctx context.Context, id string, expected int64, key, operation string, fn Mutation) (domain.Aggregate, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.Aggregate{}, false, err
	}
	if key == "" {
		return domain.Aggregate{}, false, domain.NewRuleError("idempotencyKey", "幂等键不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return domain.Aggregate{}, false, ErrClosed
	}
	env, ok := s.projects[id]
	if !ok {
		return domain.Aggregate{}, false, &domain.NotFoundError{Resource: "制作项目"}
	}
	if saved, found := env.Idempotency[key]; found {
		if saved.Operation != operation {
			return domain.Aggregate{}, false, domain.NewRuleError("idempotencyKey", "幂等键已用于其他操作")
		}
		return domain.CloneAggregate(saved.Aggregate), true, nil
	}
	if env.Aggregate.Production.Revision != expected {
		return domain.Aggregate{}, false, &domain.ConflictError{Expected: expected, Actual: env.Aggregate.Production.Revision}
	}
	next := domain.CloneAggregate(env.Aggregate)
	if err := fn(&next); err != nil {
		return domain.Aggregate{}, false, err
	}
	if next.Production.Revision != expected {
		return domain.Aggregate{}, false, errors.New("mutation 不得自行修改 revision")
	}
	next.Touch(time.Now())
	if env.Idempotency == nil {
		env.Idempotency = make(map[string]SavedResult)
	}
	env.Aggregate = next
	env.Idempotency[key] = SavedResult{Operation: operation, Aggregate: domain.CloneAggregate(next)}
	if err := s.commitLocked(id, env, operation); err != nil {
		return domain.Aggregate{}, false, err
	}
	s.projects[id] = env
	return domain.CloneAggregate(next), false, nil
}

func (s *DiskStore) List(ctx context.Context) ([]domain.Production, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Production, 0, len(s.projects))
	for _, env := range s.projects {
		out = append(out, env.Aggregate.Production)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (s *DiskStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *DiskStore) loadSnapshots() error {
	entries, err := os.ReadDir(s.snapDir)
	if err != nil {
		return fmt.Errorf("读取快照目录: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.snapDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取快照 %s: %w", entry.Name(), err)
		}
		var env SnapshotEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("解析快照 %s: %w", entry.Name(), err)
		}
		if env.Aggregate.Production.ID == "" {
			return fmt.Errorf("快照 %s 缺少项目 ID", entry.Name())
		}
		if env.Idempotency == nil {
			env.Idempotency = make(map[string]SavedResult)
		}
		s.projects[env.Aggregate.Production.ID] = env
	}
	return nil
}

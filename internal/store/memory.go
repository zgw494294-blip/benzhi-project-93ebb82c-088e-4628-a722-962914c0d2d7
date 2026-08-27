package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

// MemoryStore 用于领域测试；生产服务始终装配 DiskStore。
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]SnapshotEnvelope
}

func NewMemory() *MemoryStore { return &MemoryStore{items: make(map[string]SnapshotEnvelope)} }

func (s *MemoryStore) Create(_ context.Context, agg domain.Aggregate, key, op string) (domain.Aggregate, bool, error) {
	if key == "" {
		return domain.Aggregate{}, false, domain.NewRuleError("idempotencyKey", "幂等键不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if env, ok := s.items[agg.Production.ID]; ok {
		if saved, found := env.Idempotency[key]; found {
			if saved.Operation != op {
				return domain.Aggregate{}, false, domain.NewRuleError("idempotencyKey", "幂等键已用于其他操作")
			}
			return domain.CloneAggregate(saved.Aggregate), true, nil
		}
		return domain.Aggregate{}, false, &domain.ConflictError{Expected: 0, Actual: env.Aggregate.Production.Revision}
	}
	env := SnapshotEnvelope{Aggregate: domain.CloneAggregate(agg), Idempotency: map[string]SavedResult{key: {Operation: op, Aggregate: domain.CloneAggregate(agg)}}}
	s.items[agg.Production.ID] = env
	return domain.CloneAggregate(agg), false, nil
}

func (s *MemoryStore) Get(id string) (domain.Aggregate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	env, ok := s.items[id]
	if !ok {
		return domain.Aggregate{}, &domain.NotFoundError{Resource: "制作项目"}
	}
	return domain.CloneAggregate(env.Aggregate), nil
}

func (s *MemoryStore) Mutate(_ context.Context, id string, expected int64, key, op string, fn Mutation) (domain.Aggregate, bool, error) {
	if key == "" {
		return domain.Aggregate{}, false, domain.NewRuleError("idempotencyKey", "幂等键不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	env, ok := s.items[id]
	if !ok {
		return domain.Aggregate{}, false, &domain.NotFoundError{Resource: "制作项目"}
	}
	if saved, found := env.Idempotency[key]; found {
		if saved.Operation != op {
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
	next.Touch(time.Now())
	env.Aggregate = next
	env.Idempotency[key] = SavedResult{Operation: op, Aggregate: domain.CloneAggregate(next)}
	s.items[id] = env
	return domain.CloneAggregate(next), false, nil
}

func (s *MemoryStore) List(_ context.Context) ([]domain.Production, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Production, 0, len(s.items))
	for _, env := range s.items {
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

func (s *MemoryStore) Close() error { return nil }

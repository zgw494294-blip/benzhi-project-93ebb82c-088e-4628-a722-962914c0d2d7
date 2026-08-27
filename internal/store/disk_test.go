package store

import (
	"context"
	"testing"
	"time"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

func storedAggregate(t *testing.T) domain.Aggregate {
	t.Helper()
	a, err := domain.NewProduction("p1", "影片", "zh-CN", 20000, 25, []domain.Participant{{Name: "编剧", Role: domain.RoleWriter}, {Name: "排演", Role: domain.RolePerformer}, {Name: "审校", Role: domain.RoleReviewer}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestDiskStorePersistsAndReusesIdempotentResult(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	created, duplicate, err := s.Create(context.Background(), storedAggregate(t), "create-key", "create")
	if err != nil || duplicate {
		t.Fatalf("create duplicate=%v err=%v", duplicate, err)
	}
	updated, duplicate, err := s.Mutate(context.Background(), "p1", created.Production.Revision, "change-key", "change", func(a *domain.Aggregate) error { a.Production.Title = "改名影片"; return nil })
	if err != nil || duplicate {
		t.Fatalf("mutate duplicate=%v err=%v", duplicate, err)
	}
	replayed, duplicate, err := s.Mutate(context.Background(), "p1", created.Production.Revision, "change-key", "change", func(a *domain.Aggregate) error { return domain.NewRuleError("never", "不应再次执行") })
	if err != nil || !duplicate {
		t.Fatalf("replay duplicate=%v err=%v", duplicate, err)
	}
	if replayed.Production.Revision != updated.Production.Revision || replayed.Production.Title != "改名影片" {
		t.Fatalf("wrong replay: %+v", replayed.Production)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	loaded, err := reopened.Get(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Production.Title != "改名影片" {
		t.Fatalf("not recovered: %+v", loaded.Production)
	}
}

func TestDiskStoreRejectsStaleRevision(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	created, _, err := s.Create(context.Background(), storedAggregate(t), "create", "create")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Mutate(context.Background(), "p1", created.Production.Revision+1, "stale", "change", func(a *domain.Aggregate) error { return nil })
	if err == nil {
		t.Fatal("expected conflict")
	}
	if _, ok := err.(*domain.ConflictError); !ok {
		t.Fatalf("wrong error type: %T", err)
	}
}

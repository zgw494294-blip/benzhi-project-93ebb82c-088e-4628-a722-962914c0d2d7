package store

import (
	"context"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

type Mutation func(*domain.Aggregate) error

type Repository interface {
	Create(context.Context, domain.Aggregate, string, string) (domain.Aggregate, bool, error)
	Get(string) (domain.Aggregate, error)
	Mutate(context.Context, string, int64, string, string, Mutation) (domain.Aggregate, bool, error)
	List(context.Context) ([]domain.Production, error)
	Close() error
}

type SnapshotEnvelope struct {
	Aggregate   domain.Aggregate       `json:"aggregate"`
	Idempotency map[string]SavedResult `json:"idempotency"`
}

type SavedResult struct {
	Operation string           `json:"operation"`
	Aggregate domain.Aggregate `json:"aggregate"`
}

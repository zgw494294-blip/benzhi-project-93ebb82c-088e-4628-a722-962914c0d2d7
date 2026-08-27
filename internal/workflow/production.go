package workflow

import (
	"context"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

func (s *Service) CreateProduction(ctx context.Context, cmd CreateProductionCommand) (Result, error) {
	if cmd.ID == "" {
		cmd.ID = newID("production")
	}
	aggregate, err := domain.NewProduction(cmd.ID, cmd.Title, cmd.Language, cmd.DurationMS, cmd.FrameRate, cmd.Participants, s.now())
	if err != nil {
		return Result{}, err
	}
	saved, duplicate, err := s.repo.Create(ctx, aggregate, cmd.IdempotencyKey, "create-production")
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

func (s *Service) UpdateProduction(ctx context.Context, id string, cmd UpdateProductionCommand) (Result, error) {
	saved, duplicate, err := s.repo.Mutate(ctx, id, cmd.ExpectedRevision, cmd.IdempotencyKey, "update-production", func(a *domain.Aggregate) error {
		return a.UpdateMetadata(cmd.Title, cmd.Language, cmd.DurationMS, cmd.FrameRate, cmd.Participants)
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

func (s *Service) GetProduction(_ context.Context, id string) (domain.Aggregate, error) {
	return s.repo.Get(id)
}

func (s *Service) ListProductions(ctx context.Context) ([]domain.Production, error) {
	return s.repo.List(ctx)
}

package workflow

import (
	"context"
	"fmt"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

func (s *Service) Review(ctx context.Context, productionID string, cmd ReviewCommand) (Result, error) {
	if cmd.ID == "" {
		cmd.ID = newID("decision")
	}
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, cmd.ExpectedRevision, cmd.IdempotencyKey, "review", func(a *domain.Aggregate) error {
		decision := domain.ReviewDecision{ID: cmd.ID, FindingID: cmd.FindingID, CueID: cmd.CueID, Action: cmd.Action, Reason: cmd.Reason, Reviewer: cmd.Reviewer}
		return a.AddDecision(decision, s.now())
	})
	if err != nil {
		return Result{}, fmt.Errorf("提交审校决定失败: %v", err)
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

func (s *Service) Approve(ctx context.Context, productionID string, meta MutationMeta) (Result, error) {
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, meta.ExpectedRevision, meta.IdempotencyKey, "approve", func(a *domain.Aggregate) error {
		return a.Approve()
	})
	if err != nil {
		return Result{}, fmt.Errorf("批准制作项目失败: %v", err)
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

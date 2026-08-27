package workflow

import (
	"context"
	"fmt"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/timeline"
)

func (s *Service) SaveCue(ctx context.Context, productionID string, cmd SaveCueCommand) (Result, error) {
	if cmd.CueID == "" {
		cmd.CueID = newID("cue")
	}
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, cmd.ExpectedRevision, cmd.IdempotencyKey, "save-cue", func(a *domain.Aggregate) error {
		windows, conflicts, err := timeline.CandidateWindows(a.Segments, a.Production.DurationMS, s.minimumGapMS)
		if err != nil {
			return err
		}
		if len(conflicts) > 0 {
			return domain.NewRuleError("segments", "时间轴有冲突，不能写稿")
		}
		contained := false
		for _, window := range windows {
			if cmd.WindowStartMS >= window.StartMS && cmd.WindowEndMS <= window.EndMS {
				contained = true
				break
			}
		}
		if !contained {
			// 保留与占用区间重叠的草稿，让确定性校验报告指出并解释阻断区间。
			overlapsOccupied := false
			for _, segment := range a.Segments {
				if segment.Kind != domain.SegmentScene && timeline.Overlaps(cmd.WindowStartMS, cmd.WindowEndMS, segment.StartMS, segment.EndMS) {
					overlapsOccupied = true
					break
				}
			}
			if !overlapsOccupied {
				return domain.NewRuleError("window", "提示必须完整落在候选窗口中")
			}
		}
		for _, existing := range a.LatestCues() {
			if existing.ID == cmd.CueID {
				continue
			}
			if timeline.Overlaps(cmd.WindowStartMS, cmd.WindowEndMS, existing.WindowStartMS, existing.WindowEndMS) {
				start, end := cmd.WindowStartMS, cmd.WindowEndMS
				if existing.WindowStartMS > start {
					start = existing.WindowStartMS
				}
				if existing.WindowEndMS < end {
					end = existing.WindowEndMS
				}
				rule := domain.NewRuleError("cue_overlap", fmt.Sprintf("提示与 %s 重叠于 %d 至 %d 毫秒", existing.ID, start, end)).(*domain.RuleError)
				rule.RelatedID, rule.StartMS, rule.EndMS = existing.ID, start, end
				return rule
			}
		}
		cue := domain.NarrationCue{ID: cmd.CueID, WindowStartMS: cmd.WindowStartMS, WindowEndMS: cmd.WindowEndMS, Text: cmd.Text, Intent: cmd.Intent, PlannedCharsPerSecond: cmd.PlannedCharsPerSecond, PauseBudgetMS: cmd.PauseBudgetMS}
		return a.AddCueVersion(cue, s.now())
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

func (s *Service) WithdrawCue(ctx context.Context, productionID, cueID string, meta MutationMeta) (Result, error) {
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, meta.ExpectedRevision, meta.IdempotencyKey, "withdraw-cue", func(a *domain.Aggregate) error {
		return a.WithdrawCue(cueID, s.now())
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

func (s *Service) ValidateForRehearsal(ctx context.Context, productionID string, meta MutationMeta) (Result, error) {
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, meta.ExpectedRevision, meta.IdempotencyKey, "validate-rehearsal", func(a *domain.Aggregate) error {
		if a.Production.State != domain.StateWriting && a.Production.State != domain.StateRevising {
			return domain.NewRuleError("state", "仅 WRITING 或 REVISING 状态可以提交校验")
		}
		latest := a.LatestCues()
		if len(latest) == 0 {
			return domain.NewRuleError("cues", "至少需要一条提示")
		}
		issues := timeline.ValidateCues(latest, a.Segments, a.Production.DurationMS, s.minimumMargin)
		a.Validation = issues
		a.ValidationBlockingCount = 0
		for _, issue := range issues {
			if issue.Severity == domain.SeverityBlocking {
				a.ValidationBlockingCount++
			}
		}
		a.ValidationCueVersionSetHash = domain.CueSetHash(latest)
		if timeline.HasBlocking(issues) {
			return nil
		}
		for i := range a.Cues {
			for _, cue := range latest {
				if a.Cues[i].ID == cue.ID && a.Cues[i].Version == cue.Version {
					a.Cues[i].Status = domain.CueValidated
				}
			}
		}
		return a.Production.Transition(domain.StateRehearsing)
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

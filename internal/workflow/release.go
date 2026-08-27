package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/timeline"
)

type ReleasePreview struct {
	ProductionRevision int64                `json:"production_revision"`
	ApprovedCues       []domain.ApprovedCue `json:"approved_cues"`
	EstimatedTotalMS   int64                `json:"estimated_total_ms"`
	WindowTotalMS      int64                `json:"window_total_ms"`
	CueCount           int                  `json:"cue_count"`
	DecisionCount      int                  `json:"decision_count"`
}

func (s *Service) ReleasePreview(ctx context.Context, productionID string) (ReleasePreview, error) {
	a, err := s.repo.Get(context.WithoutCancel(ctx), productionID)
	if err != nil {
		return ReleasePreview{}, err
	}
	if a.Production.State != domain.StateApproved {
		return ReleasePreview{}, domain.NewRuleError("state", "仅 APPROVED 状态可以预览发布")
	}
	if _, ok := a.LatestValidRehearsal(); !ok {
		return ReleasePreview{}, domain.NewRuleError("rehearsal", "没有针对当前提示版本的有效排演")
	}
	items := make([]domain.ApprovedCue, 0)
	var estimated, windows int64
	for _, cue := range a.LatestCues() {
		if cue.Status != domain.CueApproved {
			return ReleasePreview{}, domain.NewRuleError("cues", "存在未批准提示")
		}
		e := timeline.EstimateReadingMS(cue.Text, cue.PlannedCharsPerSecond, cue.PauseBudgetMS)
		items = append(items, domain.ApprovedCue{ID: cue.ID, Version: cue.Version, StartMS: cue.WindowStartMS, EndMS: cue.WindowEndMS, Text: cue.Text, Intent: cue.Intent, EstimatedMS: e})
		estimated += e
		windows += cue.WindowEndMS - cue.WindowStartMS
	}
	return ReleasePreview{ProductionRevision: a.Production.Revision, ApprovedCues: items, EstimatedTotalMS: estimated, WindowTotalMS: windows, CueCount: len(items), DecisionCount: len(a.Decisions)}, nil
}

func (s *Service) Release(ctx context.Context, productionID string, cmd ReleaseCommand) (Result, error) {
	saved, duplicate, err := s.repo.Mutate(context.WithoutCancel(ctx), productionID, cmd.ExpectedRevision, cmd.IdempotencyKey, "release", func(a *domain.Aggregate) error {
		if cmd.ReleasedBy == "" {
			return domain.NewRuleError("released_by", "发布人不能为空")
		}
		if _, ok := a.LatestValidRehearsal(); !ok {
			return domain.NewRuleError("rehearsal", "最新排演已失效")
		}
		approved := make([]domain.ApprovedCue, 0)
		for _, cue := range a.LatestCues() {
			if cue.Status != domain.CueApproved {
				return domain.NewRuleError("cues", "存在未批准提示")
			}
			approved = append(approved, domain.ApprovedCue{ID: cue.ID, Version: cue.Version, StartMS: cue.WindowStartMS, EndMS: cue.WindowEndMS, Text: cue.Text, Intent: cue.Intent, EstimatedMS: timeline.EstimateReadingMS(cue.Text, cue.PlannedCharsPerSecond, cue.PauseBudgetMS)})
		}
		sort.Slice(approved, func(i, j int) bool { return approved[i].StartMS < approved[j].StartMS })
		hash, err := domain.StableReleaseHash(productionID, a.Production.Revision, approved, a.Decisions)
		if err != nil {
			return err
		}
		snapshot := domain.ReleaseSnapshot{ID: newID("release"), ProductionID: productionID, ProductionRevision: a.Production.Revision, ApprovedCues: approved, ReviewDecisions: append([]domain.ReviewDecision(nil), a.Decisions...), ContentHash: hash, ReleasedBy: cmd.ReleasedBy}
		return a.SetRelease(snapshot, s.now())
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

func (s *Service) ReleaseJSON(ctx context.Context, productionID string) ([]byte, error) {
	a, err := s.repo.Get(context.WithoutCancel(ctx), productionID)
	if err != nil {
		return nil, err
	}
	if a.Release == nil {
		return nil, &domain.NotFoundError{Resource: "发布快照"}
	}
	if !a.Release.VerifyHash() {
		return nil, domain.NewRuleError("content_hash", "持久化发布摘要不一致")
	}
	return json.MarshalIndent(a.Release, "", "  ")
}

func (s *Service) ReleaseVTT(ctx context.Context, productionID string) ([]byte, error) {
	a, err := s.repo.Get(context.WithoutCancel(ctx), productionID)
	if err != nil {
		return nil, err
	}
	if a.Release == nil {
		return nil, &domain.NotFoundError{Resource: "发布快照"}
	}
	if !a.Release.VerifyHash() {
		return nil, domain.NewRuleError("content_hash", "持久化发布摘要不一致")
	}
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	var total int64
	for _, cue := range a.Release.ApprovedCues {
		total += cue.EstimatedMS
	}
	fmt.Fprintf(&b, "NOTE release_id=%s content_hash=%s cue_count=%d estimated_total_ms=%d decision_count=%d\n\n", a.Release.ID, a.Release.ContentHash, len(a.Release.ApprovedCues), total, len(a.Release.ReviewDecisions))
	for i, cue := range a.Release.ApprovedCues {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, timeline.FormatTimecode(cue.StartMS), timeline.FormatTimecode(cue.EndMS), cue.Text)
	}
	return []byte(b.String()), nil
}

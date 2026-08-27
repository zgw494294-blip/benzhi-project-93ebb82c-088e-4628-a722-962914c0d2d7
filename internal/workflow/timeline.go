package workflow

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/timeline"
)

type WindowsResult struct {
	Windows   []timeline.Window          `json:"windows"`
	Conflicts []timeline.SegmentConflict `json:"conflicts"`
}

func (s *Service) AddSegment(ctx context.Context, productionID string, cmd AddSegmentCommand) (Result, error) {
	if cmd.ID == "" {
		cmd.ID = newID("segment")
	}
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, cmd.ExpectedRevision, cmd.IdempotencyKey, "add-segment", func(a *domain.Aggregate) error {
		if !a.Production.EditableTimeline() {
			return domain.NewRuleError("state", "当前状态不允许修改时间轴")
		}
		if cmd.Kind != domain.SegmentScene && cmd.Kind != domain.SegmentDialogue && cmd.Kind != domain.SegmentMusic && cmd.Kind != domain.SegmentOccupied {
			return domain.NewRuleError("kind", "区间类型无效")
		}
		if strings.TrimSpace(cmd.Label) == "" {
			return domain.NewRuleError("label", "区间标签不能为空")
		}
		if err := domain.ValidateInterval(cmd.StartMS, cmd.EndMS, a.Production.DurationMS); err != nil {
			return err
		}
		if cmd.Kind != domain.SegmentScene {
			if err := assignScene(a, &cmd.SceneID, cmd.StartMS, cmd.EndMS); err != nil {
				return err
			}
		}
		for _, existing := range a.Segments {
			if existing.ID == cmd.ID {
				return domain.NewRuleError("id", "区间 ID 已存在")
			}
		}
		segment := domain.TimelineSegment{ID: cmd.ID, ProductionID: productionID, SceneID: cmd.SceneID, Kind: cmd.Kind, StartMS: cmd.StartMS, EndMS: cmd.EndMS, Label: strings.TrimSpace(cmd.Label), Revision: a.Production.Revision + 1}
		a.Segments = append(a.Segments, segment)
		sortSegments(a.Segments)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

func (s *Service) UpdateSegment(ctx context.Context, productionID, segmentID string, cmd UpdateSegmentCommand) (Result, error) {
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, cmd.ExpectedRevision, cmd.IdempotencyKey, "update-segment", func(a *domain.Aggregate) error {
		if !a.Production.EditableTimeline() {
			return domain.NewRuleError("state", "当前状态不允许修改时间轴")
		}
		if cmd.Kind != domain.SegmentScene && cmd.Kind != domain.SegmentDialogue && cmd.Kind != domain.SegmentMusic && cmd.Kind != domain.SegmentOccupied {
			return domain.NewRuleError("kind", "区间类型无效")
		}
		if strings.TrimSpace(cmd.Label) == "" {
			return domain.NewRuleError("label", "区间标签不能为空")
		}
		if err := domain.ValidateInterval(cmd.StartMS, cmd.EndMS, a.Production.DurationMS); err != nil {
			return err
		}
		if cmd.Kind != domain.SegmentScene {
			if err := assignScene(a, &cmd.SceneID, cmd.StartMS, cmd.EndMS); err != nil {
				return err
			}
		}
		for i := range a.Segments {
			if a.Segments[i].ID == segmentID {
				a.Segments[i].SceneID = cmd.SceneID
				a.Segments[i].Kind = cmd.Kind
				a.Segments[i].StartMS = cmd.StartMS
				a.Segments[i].EndMS = cmd.EndMS
				a.Segments[i].Label = strings.TrimSpace(cmd.Label)
				a.Segments[i].Revision = a.Production.Revision + 1
				sortSegments(a.Segments)
				return nil
			}
		}
		return &domain.NotFoundError{Resource: "时间轴区间"}
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

func sortSegments(items []domain.TimelineSegment) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].StartMS != items[j].StartMS {
			return items[i].StartMS < items[j].StartMS
		}
		if items[i].EndMS != items[j].EndMS {
			return items[i].EndMS < items[j].EndMS
		}
		return items[i].ID < items[j].ID
	})
}

func assignScene(a *domain.Aggregate, sceneID *string, start, end int64) error {
	if *sceneID != "" {
		for _, s := range a.Segments {
			if s.Kind == domain.SegmentScene && s.ID == *sceneID {
				if start < s.StartMS || end > s.EndMS {
					rule := domain.NewRuleError("scene_id", "占用区间必须完整落在指定场景内").(*domain.RuleError)
					rule.RelatedID = s.ID
					rule.StartMS = max64(start, s.StartMS)
					rule.EndMS = min64(end, s.EndMS)
					return rule
				}
				return nil
			}
		}
		rule := domain.NewRuleError("scene_id", "占用区间引用的场景不存在").(*domain.RuleError)
		rule.RelatedID = *sceneID
		rule.StartMS, rule.EndMS = start, end
		return rule
	}
	matched := ""
	for _, s := range a.Segments {
		if s.Kind == domain.SegmentScene && start >= s.StartMS && end <= s.EndMS {
			if matched != "" {
				return domain.NewRuleError("scene_id", "占用区间必须明确选择场景")
			}
			matched = s.ID
		}
	}
	if matched == "" {
		rule := domain.NewRuleError("scene_id", "占用区间必须完整落在场景内").(*domain.RuleError)
		rule.StartMS, rule.EndMS = start, end
		return rule
	}
	*sceneID = matched
	return nil
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (s *Service) DeleteSegment(ctx context.Context, productionID, segmentID string, meta MutationMeta) (Result, error) {
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, meta.ExpectedRevision, meta.IdempotencyKey, "delete-segment", func(a *domain.Aggregate) error {
		if !a.Production.EditableTimeline() {
			return domain.NewRuleError("state", "当前状态不允许修改时间轴")
		}
		for i := range a.Segments {
			if a.Segments[i].ID == segmentID {
				a.Segments = append(a.Segments[:i], a.Segments[i+1:]...)
				return nil
			}
		}
		return &domain.NotFoundError{Resource: "时间轴区间"}
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

func (s *Service) Windows(ctx context.Context, productionID string) (WindowsResult, error) {
	a, err := s.repo.Get(ctx, productionID)
	if err != nil {
		return WindowsResult{}, fmt.Errorf("读取候选窗口项目 %s: %v", productionID, err)
	}
	windows, conflicts, err := timeline.CandidateWindows(a.Segments, a.Production.DurationMS, s.minimumGapMS)
	return WindowsResult{Windows: windows, Conflicts: conflicts}, err
}

func (s *Service) FinalizeTimeline(ctx context.Context, productionID string, meta MutationMeta) (Result, error) {
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, meta.ExpectedRevision, meta.IdempotencyKey, "finalize-timeline", func(a *domain.Aggregate) error {
		if a.Production.State != domain.StateDraft && a.Production.State != domain.StateTimelined {
			return domain.NewRuleError("state", "当前状态不能完成时间轴")
		}
		windows, conflicts, err := timeline.CandidateWindows(a.Segments, a.Production.DurationMS, s.minimumGapMS)
		if err != nil {
			return err
		}
		if len(conflicts) > 0 {
			return domain.NewRuleError("segments", "时间轴仍有区间冲突")
		}
		if len(windows) == 0 {
			return domain.NewRuleError("segments", "没有满足最小留白的候选窗口")
		}
		if a.Production.State == domain.StateDraft {
			return a.Production.Transition(domain.StateTimelined)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

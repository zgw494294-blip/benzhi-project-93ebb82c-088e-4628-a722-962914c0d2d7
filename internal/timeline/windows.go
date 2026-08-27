package timeline

import (
	"fmt"
	"sort"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

type Window struct {
	SceneID  string `json:"scene_id"`
	StartMS  int64  `json:"start_ms"`
	EndMS    int64  `json:"end_ms"`
	UsableMS int64  `json:"usable_ms"`
}

type SegmentConflict struct {
	FirstID  string `json:"first_id"`
	SecondID string `json:"second_id"`
	StartMS  int64  `json:"start_ms,omitempty"`
	EndMS    int64  `json:"end_ms,omitempty"`
	Message  string `json:"message"`
}

func CandidateWindows(segments []domain.TimelineSegment, durationMS, minGapMS int64) ([]Window, []SegmentConflict, error) {
	if durationMS <= 0 || minGapMS < 0 {
		return nil, nil, domain.NewRuleError("duration_ms", "成片时长和最小留白无效")
	}
	scenes := make([]domain.TimelineSegment, 0)
	occupied := make([]domain.TimelineSegment, 0)
	for _, s := range segments {
		if err := domain.ValidateInterval(s.StartMS, s.EndMS, durationMS); err != nil {
			return nil, nil, err
		}
		if s.Kind == domain.SegmentScene {
			scenes = append(scenes, s)
		} else {
			occupied = append(occupied, s)
		}
	}
	if len(scenes) == 0 {
		return nil, nil, domain.NewRuleError("segments", "至少需要一个场景边界")
	}
	sort.Slice(scenes, func(i, j int) bool {
		if scenes[i].StartMS != scenes[j].StartMS {
			return scenes[i].StartMS < scenes[j].StartMS
		}
		if scenes[i].EndMS != scenes[j].EndMS {
			return scenes[i].EndMS < scenes[j].EndMS
		}
		return scenes[i].ID < scenes[j].ID
	})
	sort.Slice(occupied, func(i, j int) bool {
		if occupied[i].StartMS != occupied[j].StartMS {
			return occupied[i].StartMS < occupied[j].StartMS
		}
		if occupied[i].EndMS != occupied[j].EndMS {
			return occupied[i].EndMS < occupied[j].EndMS
		}
		return occupied[i].ID < occupied[j].ID
	})
	conflicts := detectConflicts(scenes, occupied)
	windows := make([]Window, 0)
	for _, scene := range scenes {
		blocked := make([]Interval, 0)
		for _, item := range occupied {
			if Overlaps(scene.StartMS, scene.EndMS, item.StartMS, item.EndMS) {
				start := max64(scene.StartMS, item.StartMS)
				end := min64(scene.EndMS, item.EndMS)
				blocked = append(blocked, Interval{StartMS: start, EndMS: end, Labels: []string{item.Label}})
			}
		}
		merged, err := Normalize(blocked)
		if err != nil {
			return nil, nil, err
		}
		cursor := scene.StartMS
		for _, b := range merged {
			if b.StartMS-cursor >= minGapMS {
				windows = append(windows, Window{SceneID: scene.ID, StartMS: cursor, EndMS: b.StartMS, UsableMS: b.StartMS - cursor})
			}
			if b.EndMS > cursor {
				cursor = b.EndMS
			}
		}
		if scene.EndMS-cursor >= minGapMS {
			windows = append(windows, Window{SceneID: scene.ID, StartMS: cursor, EndMS: scene.EndMS, UsableMS: scene.EndMS - cursor})
		}
	}
	return windows, conflicts, nil
}

func detectConflicts(scenes, occupied []domain.TimelineSegment) []SegmentConflict {
	var out []SegmentConflict
	for i := range scenes {
		for j := i + 1; j < len(scenes); j++ {
			if Overlaps(scenes[i].StartMS, scenes[i].EndMS, scenes[j].StartMS, scenes[j].EndMS) {
				out = append(out, SegmentConflict{FirstID: scenes[i].ID, SecondID: scenes[j].ID, StartMS: max64(scenes[i].StartMS, scenes[j].StartMS), EndMS: min64(scenes[i].EndMS, scenes[j].EndMS), Message: "场景边界重叠"})
			}
		}
	}
	for _, item := range occupied {
		if item.SceneID != "" {
			matched := false
			for _, scene := range scenes {
				if scene.ID != item.SceneID {
					continue
				}
				matched = true
				if item.StartMS >= scene.StartMS && item.EndMS <= scene.EndMS {
					break
				}
				out = append(out, SegmentConflict{
					FirstID: item.ID, SecondID: item.SceneID,
					StartMS: item.StartMS, EndMS: item.EndMS,
					Message: fmt.Sprintf("占用区间 %s 越过场景 %s 边界", item.Label, item.SceneID),
				})
				break
			}
			if !matched {
				out = append(out, SegmentConflict{
					FirstID: item.ID, SecondID: item.SceneID,
					StartMS: item.StartMS, EndMS: item.EndMS,
					Message: fmt.Sprintf("占用区间 %s 引用了未知场景 %s", item.Label, item.SceneID),
				})
			}
			continue
		}
		contained := false
		for _, scene := range scenes {
			if item.StartMS >= scene.StartMS && item.EndMS <= scene.EndMS {
				contained = true
				break
			}
		}
		if !contained {
			second := item.SceneID
			if second == "" {
				for _, scene := range scenes {
					if Overlaps(item.StartMS, item.EndMS, scene.StartMS, scene.EndMS) {
						second = scene.ID
						break
					}
				}
			}
			message := fmt.Sprintf("占用区间 %s 未完整落在场景 %s 内", item.Label, second)
			out = append(out, SegmentConflict{FirstID: item.ID, SecondID: second, StartMS: item.StartMS, EndMS: item.EndMS, Message: message})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartMS != out[j].StartMS {
			return out[i].StartMS < out[j].StartMS
		}
		if out[i].FirstID != out[j].FirstID {
			return out[i].FirstID < out[j].FirstID
		}
		return out[i].SecondID < out[j].SecondID
	})
	return out
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

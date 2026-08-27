package domain

import (
	"sort"
	"strings"
	"time"
)

func (a *Aggregate) LatestCues() []NarrationCue {
	byID := make(map[string]NarrationCue)
	for _, cue := range a.Cues {
		if old, ok := byID[cue.ID]; !ok || cue.Version > old.Version {
			byID[cue.ID] = cue
		}
	}
	out := make([]NarrationCue, 0, len(byID))
	for _, cue := range byID {
		if cue.Status == CueWithdrawn || cue.WithdrawnAt != nil {
			continue
		}
		out = append(out, cue)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].WindowStartMS == out[j].WindowStartMS {
			return out[i].ID < out[j].ID
		}
		return out[i].WindowStartMS < out[j].WindowStartMS
	})
	return out
}

// WithdrawCue 保留全部历史版本，同时将最新版本排除在校验、排演和发布输入之外。
func (a *Aggregate) WithdrawCue(id string, now time.Time) error {
	if !a.Production.EditableCues() {
		return NewRuleError("state", "当前状态不允许撤回提示稿")
	}
	latest := 0
	index := -1
	for i := range a.Cues {
		if a.Cues[i].ID == id && a.Cues[i].Version > latest {
			latest = a.Cues[i].Version
			index = i
		}
	}
	if index < 0 {
		return &NotFoundError{Resource: "提示"}
	}
	when := now.UTC()
	a.Cues[index].Status = CueWithdrawn
	a.Cues[index].WithdrawnAt = &when
	a.Validation = nil
	a.ValidationBlockingCount = 0
	a.ValidationCueVersionSetHash = ""
	if a.Production.State == StateRevising {
		for i := range a.Rehearsals {
			if a.Rehearsals[i].InvalidatedAt == nil {
				a.Rehearsals[i].InvalidatedAt = &when
			}
		}
		a.Decisions = nil
		a.Release = nil
		return a.Production.Transition(StateWriting)
	}
	if a.Production.State == StateTimelined {
		return a.Production.Transition(StateWriting)
	}
	return nil
}

func (a *Aggregate) AddCueVersion(cue NarrationCue, now time.Time) error {
	if !a.Production.EditableCues() {
		return NewRuleError("state", "当前状态不允许修改提示稿")
	}
	if strings.TrimSpace(cue.ID) == "" || strings.TrimSpace(cue.Text) == "" {
		return NewRuleError("text", "提示 ID 和文本不能为空")
	}
	if err := ValidateInterval(cue.WindowStartMS, cue.WindowEndMS, a.Production.DurationMS); err != nil {
		return err
	}
	if cue.PlannedCharsPerSecond <= 0 || cue.PlannedCharsPerSecond > 20 {
		return NewRuleError("planned_chars_per_second", "预计语速必须在 0 到 20 字/秒之间")
	}
	if cue.PauseBudgetMS < 0 {
		return NewRuleError("pause_budget_ms", "停顿预算不能为负数")
	}
	for _, existing := range a.LatestCues() {
		if existing.ID != cue.ID && cue.WindowStartMS < existing.WindowEndMS && existing.WindowStartMS < cue.WindowEndMS {
			return NewRuleError("cue_overlap", "提示窗口与已有提示重叠")
		}
	}
	latest := 0
	for _, existing := range a.Cues {
		if existing.ID == cue.ID && existing.Version > latest {
			latest = existing.Version
		}
	}
	if latest == 0 {
		cue.Version = 1
		cue.SupersedesVersion = 0
	} else {
		cue.Version = latest + 1
		cue.SupersedesVersion = latest
	}
	if a.Production.State == StateRevising {
		nowUTC := now.UTC()
		for i := range a.Rehearsals {
			if a.Rehearsals[i].InvalidatedAt == nil {
				a.Rehearsals[i].InvalidatedAt = &nowUTC
			}
		}
		a.Decisions = nil
		a.Release = nil
		if err := a.Production.Transition(StateWriting); err != nil {
			return err
		}
	} else if a.Production.State == StateTimelined {
		if err := a.Production.Transition(StateWriting); err != nil {
			return err
		}
	}
	cue.ProductionID = a.Production.ID
	cue.Status = CueDraft
	cue.WithdrawnAt = nil
	cue.CreatedAt = now.UTC()
	a.Cues = append(a.Cues, cue)
	a.Validation = nil
	a.ValidationBlockingCount = 0
	a.ValidationCueVersionSetHash = ""
	return nil
}

func (a *Aggregate) CueVersion(id string, version int) (NarrationCue, bool) {
	for _, cue := range a.Cues {
		if cue.ID == id && cue.Version == version {
			return cue, true
		}
	}
	return NarrationCue{}, false
}

package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

func CueSetHash(cues []NarrationCue) string {
	parts := make([]string, 0, len(cues))
	for _, cue := range cues {
		parts = append(parts, fmt.Sprintf("%s:%d", cue.ID, cue.Version))
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

func (a *Aggregate) AddRehearsal(take RehearsalTake, now time.Time) error {
	if a.Production.State != StateRehearsing {
		return NewRuleError("state", "仅 REHEARSING 状态可以登记排演")
	}
	latest := a.LatestCues()
	latestVersions := make(map[string]int, len(latest))
	for _, cue := range latest {
		latestVersions[cue.ID] = cue.Version
	}
	if len(latest) == 0 || take.CueVersionSetHash != CueSetHash(latest) {
		return NewRuleError("cue_version_set_hash", "排演提示版本集合不是当前固定版本")
	}
	if len(take.Measurements) != len(latest) {
		return NewRuleError("measurements", "每个当前提示都必须有实读测量")
	}
	seen := map[string]bool{}
	for _, m := range take.Measurements {
		if seen[m.CueID] {
			return NewRuleError("measurements", "提示测量不能重复")
		}
		seen[m.CueID] = true
		cue, ok := a.CueVersion(m.CueID, m.CueVersion)
		if !ok || cue.Version == 0 || latestVersions[m.CueID] != m.CueVersion {
			return NewRuleError("measurements", "测量引用了未知提示版本")
		}
		if err := ValidateInterval(m.ActualStartMS, m.ActualEndMS, a.Production.DurationMS); err != nil {
			return err
		}
		if m.SpokenDurationMS <= 0 || m.PauseMS < 0 {
			return NewRuleError("measurements", "实读时长必须为正且停顿不能为负")
		}
		if m.SpokenDurationMS+m.PauseMS > m.ActualEndMS-m.ActualStartMS {
			return NewRuleError("measurements", "实读时长与停顿之和不能超过实际区间")
		}
	}
	for _, cue := range latest {
		if !seen[cue.ID] {
			return NewRuleError("measurements", "缺少提示 "+cue.ID+" 的实读测量")
		}
	}
	for i := range take.Findings {
		finding := &take.Findings[i]
		if finding.Severity == "" {
			finding.Severity = SeverityAdvisory
		}
		if !seen[finding.CueID] {
			return NewRuleError("findings", "排演问题引用了未知提示 "+finding.CueID)
		}
		if strings.TrimSpace(finding.Code) == "" || strings.TrimSpace(finding.Message) == "" {
			return NewRuleError("findings", "排演问题必须包含代码和说明")
		}
		if finding.Severity != SeverityBlocking && finding.Severity != SeverityAdvisory {
			return NewRuleError("findings", "排演问题严重级别无效")
		}
	}
	for i := range take.Findings {
		take.Findings[i].Resolved = false
	}
	take.ProductionID = a.Production.ID
	take.Round = len(a.Rehearsals) + 1
	take.StartedAt = now.UTC()
	take.CompletedAt = now.UTC()
	if len(a.Rehearsals) > 0 {
		previous := a.Rehearsals[len(a.Rehearsals)-1]
		previousMeasurements := make(map[string]CueMeasurement, len(previous.Measurements))
		for _, m := range previous.Measurements {
			previousMeasurements[m.CueID] = m
		}
		previousFindings := make(map[string]int)
		for _, f := range previous.Findings {
			previousFindings[f.CueID]++
		}
		currentFindings := make(map[string]int)
		for _, f := range take.Findings {
			currentFindings[f.CueID]++
		}
		for _, m := range take.Measurements {
			cmp := RehearsalComparison{CueID: m.CueID, PreviousRound: previous.Round, CurrentRound: take.Round}
			old, ok := previousMeasurements[m.CueID]
			_, cueOK := a.CueVersion(m.CueID, m.CueVersion)
			oldCueVersion := 0
			for _, pm := range previous.Measurements {
				if pm.CueID == m.CueID {
					oldCueVersion = pm.CueVersion
					break
				}
			}
			if ok && cueOK && oldCueVersion == m.CueVersion {
				cmp.Comparable = true
				cmp.SpokenDurationDeltaMS = m.SpokenDurationMS - old.SpokenDurationMS
				cmp.WindowDeltaDeltaMS = m.WindowDeltaMS - old.WindowDeltaMS
				cmp.WindowStartDeltaDeltaMS = m.WindowStartDeltaMS - old.WindowStartDeltaMS
				cmp.WindowEndDeltaDeltaMS = m.WindowEndDeltaMS - old.WindowEndDeltaMS
				cmp.FindingCountDelta = currentFindings[m.CueID] - previousFindings[m.CueID]
			} else {
				cmp.Note = "提示版本不同或上一轮缺少该提示，不能直接比较"
			}
			take.Comparisons = append(take.Comparisons, cmp)
		}
	}
	a.Rehearsals = append(a.Rehearsals, take)
	for i := range a.Cues {
		for _, latestCue := range latest {
			if a.Cues[i].ID == latestCue.ID && a.Cues[i].Version == latestCue.Version {
				a.Cues[i].Status = CueRehearsed
			}
		}
	}
	return a.Production.Transition(StateReviewing)
}

func (a Aggregate) LatestValidRehearsal() (RehearsalTake, bool) {
	for i := len(a.Rehearsals) - 1; i >= 0; i-- {
		if a.Rehearsals[i].InvalidatedAt == nil && a.Rehearsals[i].CueVersionSetHash == CueSetHash(a.LatestCues()) {
			return a.Rehearsals[i], true
		}
	}
	return RehearsalTake{}, false
}

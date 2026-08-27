package domain

import (
	"strings"
	"time"
)

func (a *Aggregate) AddDecision(decision ReviewDecision, now time.Time) error {
	if a.Production.State != StateReviewing {
		return NewRuleError("state", "仅 REVIEWING 状态可以审校")
	}
	if decision.Action != ReviewAccept && decision.Action != ReviewRevise && decision.Action != ReviewReject {
		return NewRuleError("action", "审校动作无效")
	}
	if (decision.Action == ReviewRevise || decision.Action == ReviewReject) && strings.TrimSpace(decision.Reason) == "" {
		return NewRuleError("reason", "要求修订或驳回时必须说明理由")
	}
	if strings.TrimSpace(decision.Reviewer) == "" {
		return NewRuleError("reviewer", "审校员不能为空")
	}
	registered := false
	for _, p := range a.Production.Participants {
		if p.Role == RoleReviewer && strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(decision.Reviewer)) {
			registered = true
			break
		}
	}
	if !registered {
		return NewRuleError("reviewer", "审校员必须是项目登记的 REVIEWER")
	}
	known := false
	for _, cue := range a.LatestCues() {
		if cue.ID == decision.CueID {
			known = true
			break
		}
	}
	if !known {
		return NewRuleError("cue_id", "审校决定引用未知提示")
	}
	if decision.FindingID != "" {
		found := false
		if take, ok := a.LatestValidRehearsal(); ok {
			for _, f := range take.Findings {
				if f.ID == decision.FindingID && f.CueID == decision.CueID {
					found = true
				}
			}
		}
		if !found {
			return NewRuleError("finding_id", "审校决定引用未知排演问题")
		}
	}
	for _, existing := range a.Decisions {
		if existing.CueID == decision.CueID && existing.FindingID == decision.FindingID {
			return NewRuleError("decision", "同一提示或问题已有当前审校决定")
		}
	}
	decision.CreatedAt = now.UTC()
	a.Decisions = append(a.Decisions, decision)
	if decision.Action == ReviewRevise || decision.Action == ReviewReject {
		return a.Production.Transition(StateRevising)
	}
	return nil
}

func (a Aggregate) CanApprove() error {
	if a.Production.State != StateReviewing {
		return NewRuleError("state", "仅 REVIEWING 状态可以批准")
	}
	take, ok := a.LatestValidRehearsal()
	if !ok {
		return NewRuleError("rehearsal", "没有针对当前提示版本的有效排演")
	}
	acceptedCues := map[string]bool{}
	acceptedFindings := map[string]bool{}
	for _, d := range a.Decisions {
		if d.Action == ReviewAccept {
			if d.FindingID == "" {
				acceptedCues[d.CueID] = true
			} else {
				acceptedFindings[d.FindingID] = true
			}
		}
	}
	missing := make([]string, 0)
	for _, cue := range a.LatestCues() {
		if !acceptedCues[cue.ID] {
			missing = append(missing, cue.ID)
		}
	}
	missingCues := append([]string(nil), missing...)
	missing = nil
	for _, f := range take.Findings {
		if f.Severity == SeverityBlocking && !f.Resolved && !acceptedFindings[f.ID] {
			missing = append(missing, f.ID)
		}
	}
	if len(missingCues) > 0 || len(missing) > 0 {
		parts := make([]string, 0, 2)
		if len(missingCues) > 0 {
			parts = append(parts, "未接受提示: "+strings.Join(missingCues, ","))
		}
		if len(missing) > 0 {
			parts = append(parts, "未处置阻断问题: "+strings.Join(missing, ","))
		}
		return NewRuleError("decisions", strings.Join(parts, "; "))
	}
	return nil
}

func (a *Aggregate) Approve() error {
	if err := a.CanApprove(); err != nil {
		return err
	}
	for i := range a.Cues {
		for _, latest := range a.LatestCues() {
			if a.Cues[i].ID == latest.ID && a.Cues[i].Version == latest.Version {
				a.Cues[i].Status = CueApproved
			}
		}
	}
	return a.Production.Transition(StateApproved)
}

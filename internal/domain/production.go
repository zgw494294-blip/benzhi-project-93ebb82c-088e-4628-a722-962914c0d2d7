package domain

import (
	"strings"
	"time"
)

func NewProduction(id, title, language string, durationMS int64, frameRate float64, participants []Participant, now time.Time) (Aggregate, error) {
	if strings.TrimSpace(id) == "" {
		return Aggregate{}, NewRuleError("id", "项目 ID 不能为空")
	}
	if strings.TrimSpace(title) == "" {
		return Aggregate{}, NewRuleError("title", "片名不能为空")
	}
	if strings.TrimSpace(language) == "" {
		return Aggregate{}, NewRuleError("language", "语言不能为空")
	}
	if durationMS <= 0 {
		return Aggregate{}, NewRuleError("duration_ms", "成片时长必须大于零")
	}
	if frameRate <= 0 || frameRate > 240 {
		return Aggregate{}, NewRuleError("frame_rate", "帧率必须在 0 到 240 之间")
	}
	if err := ValidateParticipants(participants); err != nil {
		return Aggregate{}, err
	}
	return Aggregate{Production: Production{
		ID: id, Title: strings.TrimSpace(title), Language: strings.TrimSpace(language),
		DurationMS: durationMS, FrameRate: frameRate, State: StateDraft, Revision: 1,
		Participants: append([]Participant(nil), participants...), CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}}, nil
}

func ValidateParticipants(items []Participant) error {
	if len(items) == 0 {
		return NewRuleError("participants", "至少登记一名参与者")
	}
	roles := map[Role]bool{}
	for i, p := range items {
		if strings.TrimSpace(p.Name) == "" {
			return NewRuleError("participants", "参与者姓名不能为空")
		}
		if p.Role != RoleWriter && p.Role != RolePerformer && p.Role != RoleReviewer {
			return NewRuleError("participants", "参与者角色无效")
		}
		roles[p.Role] = true
		for j := 0; j < i; j++ {
			if strings.EqualFold(items[j].Name, p.Name) && items[j].Role == p.Role {
				return NewRuleError("participants", "同一角色下参与者不能重复")
			}
		}
	}
	for _, role := range []Role{RoleWriter, RolePerformer, RoleReviewer} {
		if !roles[role] {
			return NewRuleError("participants", "编剧、排演员和审校员角色必须齐全")
		}
	}
	return nil
}

func (a *Aggregate) Touch(now time.Time) {
	a.Production.Revision++
	a.Production.UpdatedAt = now.UTC()
}

func (a *Aggregate) UpdateMetadata(title, language string, durationMS int64, frameRate float64, participants []Participant) error {
	if !a.Production.EditableMetadata() {
		return NewRuleError("state", "当前状态不允许修改项目资料")
	}
	if strings.TrimSpace(title) == "" || strings.TrimSpace(language) == "" {
		return NewRuleError("title", "片名和语言不能为空")
	}
	if durationMS <= 0 || frameRate <= 0 || frameRate > 240 {
		return NewRuleError("duration_ms", "成片时长或帧率无效")
	}
	if err := ValidateParticipants(participants); err != nil {
		return err
	}
	for _, s := range a.Segments {
		if s.EndMS > durationMS {
			return NewRuleError("duration_ms", "新时长不能截断已有时间轴")
		}
	}
	a.Production.Title = strings.TrimSpace(title)
	a.Production.Language = strings.TrimSpace(language)
	a.Production.DurationMS = durationMS
	a.Production.FrameRate = frameRate
	a.Production.Participants = append([]Participant(nil), participants...)
	return nil
}

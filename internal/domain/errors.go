package domain

import "fmt"

type RuleError struct {
	Field     string `json:"field,omitempty"`
	Message   string `json:"message"`
	RelatedID string `json:"related_id,omitempty"`
	StartMS   int64  `json:"start_ms,omitempty"`
	EndMS     int64  `json:"end_ms,omitempty"`
}

func (e *RuleError) Error() string { return e.Message }

func NewRuleError(field, message string) error {
	return &RuleError{Field: field, Message: message}
}

type ConflictError struct {
	Expected int64 `json:"expected"`
	Actual   int64 `json:"actual"`
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("版本冲突：期望 %d，实际 %d", e.Expected, e.Actual)
}

type NotFoundError struct{ Resource string }

func (e *NotFoundError) Error() string { return e.Resource + "不存在" }

func ValidateInterval(start, end, limit int64) error {
	if start < 0 {
		return NewRuleError("start_ms", "开始时间不能为负数")
	}
	if end <= start {
		return NewRuleError("end_ms", "结束时间必须晚于开始时间")
	}
	if limit > 0 && end > limit {
		return NewRuleError("end_ms", "结束时间超过成片时长")
	}
	return nil
}

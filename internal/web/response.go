package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Field     string `json:"field,omitempty"`
	Expected  int64  `json:"expected,omitempty"`
	Actual    int64  `json:"actual,omitempty"`
	RelatedID string `json:"related_id,omitempty"`
	StartMS   int64  `json:"start_ms,omitempty"`
	EndMS     int64  `json:"end_ms,omitempty"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, domain.NewRuleError("body", "JSON 请求无效："+err.Error()))
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, domain.NewRuleError("body", "请求体只能包含一个 JSON 对象"))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	var rule *domain.RuleError
	var conflict *domain.ConflictError
	var missing *domain.NotFoundError
	detail := errorDetail{Code: "internal_error", Message: "服务器处理失败"}
	status := http.StatusInternalServerError
	switch {
	case errors.As(err, &rule):
		status, detail = http.StatusUnprocessableEntity, errorDetail{Code: "rule_violation", Message: rule.Message, Field: rule.Field, RelatedID: rule.RelatedID, StartMS: rule.StartMS, EndMS: rule.EndMS}
	case errors.As(err, &conflict):
		status, detail = http.StatusConflict, errorDetail{Code: "revision_conflict", Message: conflict.Error(), Expected: conflict.Expected, Actual: conflict.Actual}
	case errors.As(err, &missing):
		status, detail = http.StatusNotFound, errorDetail{Code: "not_found", Message: missing.Error()}
	}
	writeJSON(w, status, errorBody{Error: detail})
}

func queryInt(r *http.Request, name string) (int, error) {
	value := r.URL.Query().Get(name)
	var out int
	if value == "" {
		return 0, domain.NewRuleError(name, "缺少查询参数 "+name)
	}
	if _, err := fmt.Sscanf(value, "%d", &out); err != nil || out <= 0 {
		return 0, domain.NewRuleError(name, name+" 必须为正整数")
	}
	return out, nil
}

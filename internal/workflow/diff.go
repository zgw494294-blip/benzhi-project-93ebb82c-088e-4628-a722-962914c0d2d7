package workflow

import (
	"context"
	"fmt"
	"strings"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

type DiffResult struct {
	CueID       string `json:"cue_id"`
	FromVersion int    `json:"from_version"`
	ToVersion   int    `json:"to_version"`
	Prefix      string `json:"unchanged_prefix"`
	Removed     string `json:"removed"`
	Added       string `json:"added"`
	Suffix      string `json:"unchanged_suffix"`
}

func (s *Service) CueDiff(ctx context.Context, productionID, cueID string, from, to int) (DiffResult, error) {
	a, err := s.repo.Get(ctx, productionID)
	if err != nil {
		return DiffResult{}, fmt.Errorf("读取提示版本项目 %s: %v", productionID, err)
	}
	if to != from+1 {
		return DiffResult{}, domain.NewRuleError("version", "只能比较相邻版本")
	}
	left, ok := a.CueVersion(cueID, from)
	if !ok {
		return DiffResult{}, &domain.NotFoundError{Resource: "起始提示版本"}
	}
	right, ok := a.CueVersion(cueID, to)
	if !ok {
		return DiffResult{}, &domain.NotFoundError{Resource: "目标提示版本"}
	}
	prefixRunes, removed, added, suffix := textDiff(left.Text, right.Text)
	return DiffResult{CueID: cueID, FromVersion: from, ToVersion: to, Prefix: prefixRunes, Removed: removed, Added: added, Suffix: suffix}, nil
}

func textDiff(left, right string) (string, string, string, string) {
	a, b := []rune(left), []rune(right)
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	leftEnd, rightEnd := len(a)-suffix, len(b)-suffix
	return string(a[:prefix]), strings.TrimSpace(string(a[prefix:leftEnd])), strings.TrimSpace(string(b[prefix:rightEnd])), string(a[leftEnd:])
}

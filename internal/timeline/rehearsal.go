package timeline

import (
	"fmt"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

func EvaluateRehearsal(cues []domain.NarrationCue, measurements []domain.CueMeasurement, reported []domain.Finding) []domain.Finding {
	byCue := make(map[string]domain.NarrationCue)
	for _, cue := range cues {
		byCue[cue.ID] = cue
	}
	out := append([]domain.Finding(nil), reported...)
	sequence := len(out)
	for _, m := range measurements {
		cue, ok := byCue[m.CueID]
		if !ok {
			continue
		}
		if m.ActualStartMS < cue.WindowStartMS || m.ActualEndMS > cue.WindowEndMS {
			sequence++
			out = append(out, domain.Finding{
				ID: fmt.Sprintf("auto-overflow-%d", sequence), CueID: cue.ID,
				Code: "ACTUAL_OUTSIDE_WINDOW", Message: "实读起止时间超出计划窗口", Severity: domain.SeverityBlocking,
			})
		}
		elapsed := m.ActualEndMS - m.ActualStartMS
		if m.SpokenDurationMS+m.PauseMS > elapsed {
			sequence++
			out = append(out, domain.Finding{
				ID: fmt.Sprintf("auto-measure-%d", sequence), CueID: cue.ID,
				Code: "MEASUREMENT_INCONSISTENT", Message: "实读时长与停顿之和超过实际区间", Severity: domain.SeverityBlocking,
			})
		}
		planned := EstimateReadingMS(cue.Text, cue.PlannedCharsPerSecond, cue.PauseBudgetMS)
		if m.SpokenDurationMS > planned+1000 {
			sequence++
			out = append(out, domain.Finding{
				ID: fmt.Sprintf("auto-pace-%d", sequence), CueID: cue.ID,
				Code: "PACE_SLOW", Message: "实读明显慢于预计时长，请审校可懂度", Severity: domain.SeverityAdvisory,
			})
		}
	}
	return out
}

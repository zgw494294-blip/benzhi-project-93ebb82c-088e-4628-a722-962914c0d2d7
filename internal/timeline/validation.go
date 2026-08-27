package timeline

import (
	"fmt"
	"sort"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

func ValidateCues(cues []domain.NarrationCue, segments []domain.TimelineSegment, durationMS, minimumMarginMS int64) []domain.ValidationIssue {
	issues := make([]domain.ValidationIssue, 0)
	starts := make(map[string]int64, len(cues))
	versions := make(map[string]int, len(cues))
	ends := make(map[string]int64, len(cues))
	estimates := make(map[string]int64, len(cues))
	capacities := make(map[string]int64, len(cues))
	for _, cue := range cues {
		starts[cue.ID] = cue.WindowStartMS
		versions[cue.ID] = cue.Version
		ends[cue.ID] = cue.WindowEndMS
		estimates[cue.ID] = EstimateReadingMS(cue.Text, cue.PlannedCharsPerSecond, cue.PauseBudgetMS)
		capacities[cue.ID] = cue.WindowEndMS - cue.WindowStartMS - 2*minimumMarginMS
	}
	for _, cue := range cues {
		if err := domain.ValidateInterval(cue.WindowStartMS, cue.WindowEndMS, durationMS); err != nil {
			item := issue(cue.ID, "TIME_BOUNDARY", err.Error(), domain.SeverityBlocking)
			item.EstimatedMS, item.UsableMS = estimates[cue.ID], capacities[cue.ID]
			issues = append(issues, item)
		}
		for _, segment := range segments {
			if segment.Kind != domain.SegmentScene && Overlaps(cue.WindowStartMS, cue.WindowEndMS, segment.StartMS, segment.EndMS) {
				item := issue(cue.ID, "OCCUPIED_OVERLAP", "提示窗口遮盖了“"+segment.Label+"”", domain.SeverityBlocking)
				item.OccupiedStartMS = max64(cue.WindowStartMS, segment.StartMS)
				item.OccupiedEndMS = min64(cue.WindowEndMS, segment.EndMS)
				issues = append(issues, item)
			}
		}
		estimated := EstimateReadingMS(cue.Text, cue.PlannedCharsPerSecond, cue.PauseBudgetMS)
		capacity := cue.WindowEndMS - cue.WindowStartMS - 2*minimumMarginMS
		if capacity <= 0 {
			item := issue(cue.ID, "MINIMUM_MARGIN", "窗口无法提供两侧最小留白", domain.SeverityBlocking)
			item.EstimatedMS, item.UsableMS = estimated, capacity
			issues = append(issues, item)
		} else if estimated > capacity {
			item := issue(cue.ID, "READING_OVERFLOW", fmt.Sprintf("预计需要 %dms，含留白可用 %dms", estimated, capacity), domain.SeverityBlocking)
			item.EstimatedMS, item.UsableMS = estimated, capacity
			issues = append(issues, item)
		}
		if CountReadableCharacters(cue.Text) < 2 {
			issues = append(issues, issue(cue.ID, "TEXT_TOO_SHORT", "提示文本过短，无法用于排演", domain.SeverityBlocking))
		}
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if starts[issues[i].CueID] != starts[issues[j].CueID] {
			return starts[issues[i].CueID] < starts[issues[j].CueID]
		}
		if issues[i].CueID != issues[j].CueID {
			return issues[i].CueID < issues[j].CueID
		}
		if issues[i].Code != issues[j].Code {
			return issues[i].Code < issues[j].Code
		}
		return issues[i].Message < issues[j].Message
	})
	for i := range issues {
		issues[i].CueVersion = versions[issues[i].CueID]
		issues[i].WindowStartMS = starts[issues[i].CueID]
		issues[i].WindowEndMS = ends[issues[i].CueID]
		if issues[i].EstimatedMS == 0 {
			issues[i].EstimatedMS = estimates[issues[i].CueID]
		}
		if issues[i].UsableMS == 0 {
			issues[i].UsableMS = capacities[issues[i].CueID]
		}
	}
	return issues
}

func issue(cueID, code, message string, severity domain.FindingSeverity) domain.ValidationIssue {
	return domain.ValidationIssue{CueID: cueID, Code: code, Message: message, Severity: severity}
}

func HasBlocking(issues []domain.ValidationIssue) bool {
	for _, item := range issues {
		if item.Severity == domain.SeverityBlocking {
			return true
		}
	}
	return false
}

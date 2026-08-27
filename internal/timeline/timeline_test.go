package timeline

import (
	"testing"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

func TestCandidateWindowsMergesOccupiedIntervals(t *testing.T) {
	segments := []domain.TimelineSegment{
		{ID: "scene", Kind: domain.SegmentScene, StartMS: 0, EndMS: 20000},
		{ID: "d1", Kind: domain.SegmentDialogue, StartMS: 3000, EndMS: 6000, Label: "对白一"},
		{ID: "d2", Kind: domain.SegmentMusic, StartMS: 5000, EndMS: 8000, Label: "音乐"},
	}
	windows, conflicts, err := CandidateWindows(segments, 20000, 1200)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", conflicts)
	}
	if len(windows) != 2 {
		t.Fatalf("want 2 windows, got %+v", windows)
	}
	if windows[0].StartMS != 0 || windows[0].EndMS != 3000 {
		t.Fatalf("first window: %+v", windows[0])
	}
	if windows[1].StartMS != 8000 || windows[1].EndMS != 20000 {
		t.Fatalf("second window: %+v", windows[1])
	}
}

func TestValidationDetectsReadingOverflowAndOverlap(t *testing.T) {
	cues := []domain.NarrationCue{{ID: "cue", WindowStartMS: 1000, WindowEndMS: 3000, Text: "这是一段明显无法在狭小窗口中读完的口述影像旁白", PlannedCharsPerSecond: 2}}
	segments := []domain.TimelineSegment{{ID: "dialogue", Kind: domain.SegmentDialogue, StartMS: 2000, EndMS: 2500, Label: "对白"}}
	issues := ValidateCues(cues, segments, 10000, 250)
	codes := map[string]bool{}
	for _, issue := range issues {
		codes[issue.Code] = true
	}
	if !codes["OCCUPIED_OVERLAP"] || !codes["READING_OVERFLOW"] {
		t.Fatalf("missing expected issues: %+v", issues)
	}
}

func TestEvaluateRehearsalSeparatesSeverity(t *testing.T) {
	cues := []domain.NarrationCue{{ID: "cue", Version: 1, WindowStartMS: 1000, WindowEndMS: 5000, Text: "灯塔亮起", PlannedCharsPerSecond: 6}}
	measurements := []domain.CueMeasurement{{CueID: "cue", CueVersion: 1, ActualStartMS: 900, ActualEndMS: 5200, SpokenDurationMS: 4000, PauseMS: 500}}
	findings := EvaluateRehearsal(cues, measurements, nil)
	if len(findings) < 2 {
		t.Fatalf("expected overflow and pace findings, got %+v", findings)
	}
	if findings[0].Severity != domain.SeverityBlocking {
		t.Fatalf("overflow should block: %+v", findings[0])
	}
}

func TestReadingHelpers(t *testing.T) {
	if got := CountReadableCharacters("晨雾，散开。 "); got != 4 {
		t.Fatalf("got %d", got)
	}
	if got := EstimateReadingMS("晨雾散开", 4, 250); got != 1250 {
		t.Fatalf("got %d", got)
	}
	if got := FormatTimecode(3723456); got != "01:02:03.456" {
		t.Fatalf("got %s", got)
	}
}

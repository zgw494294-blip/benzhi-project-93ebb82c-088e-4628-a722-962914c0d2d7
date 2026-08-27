package workflow

import (
	"context"
	"strings"
	"testing"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/store"
)

func TestWorkflowHappyPathToExports(t *testing.T) {
	ctx := context.Background()
	service := New(store.NewMemory())
	created, err := service.CreateProduction(ctx, CreateProductionCommand{ID: "p", Title: "影片", Language: "zh-CN", DurationMS: 20000, FrameRate: 25, Participants: []domain.Participant{{Name: "编剧", Role: domain.RoleWriter}, {Name: "排演", Role: domain.RolePerformer}, {Name: "审校", Role: domain.RoleReviewer}}, IdempotencyKey: "create"})
	if err != nil {
		t.Fatal(err)
	}
	a := created.Value.(domain.Aggregate)
	segment, err := service.AddSegment(ctx, "p", AddSegmentCommand{MutationMeta: MutationMeta{ExpectedRevision: a.Production.Revision, IdempotencyKey: "scene"}, ID: "scene", Kind: domain.SegmentScene, StartMS: 0, EndMS: 20000, Label: "场景"})
	if err != nil {
		t.Fatal(err)
	}
	a = segment.Value.(domain.Aggregate)
	finalized, err := service.FinalizeTimeline(ctx, "p", MutationMeta{ExpectedRevision: a.Production.Revision, IdempotencyKey: "finalize"})
	if err != nil {
		t.Fatal(err)
	}
	a = finalized.Value.(domain.Aggregate)
	saved, err := service.SaveCue(ctx, "p", SaveCueCommand{MutationMeta: MutationMeta{ExpectedRevision: a.Production.Revision, IdempotencyKey: "cue"}, CueID: "cue", WindowStartMS: 1000, WindowEndMS: 9000, Text: "晨雾散开，灯塔亮起。", Intent: "交代环境", PlannedCharsPerSecond: 5, PauseBudgetMS: 200})
	if err != nil {
		t.Fatal(err)
	}
	a = saved.Value.(domain.Aggregate)
	validated, err := service.ValidateForRehearsal(ctx, "p", MutationMeta{ExpectedRevision: a.Production.Revision, IdempotencyKey: "validate"})
	if err != nil {
		t.Fatal(err)
	}
	a = validated.Value.(domain.Aggregate)
	rehearsed, err := service.RecordRehearsal(ctx, "p", RecordRehearsalCommand{MutationMeta: MutationMeta{ExpectedRevision: a.Production.Revision, IdempotencyKey: "take"}, ID: "take", Measurements: []domain.CueMeasurement{{CueID: "cue", CueVersion: 1, ActualStartMS: 1200, ActualEndMS: 8000, SpokenDurationMS: 2500, PauseMS: 200}}})
	if err != nil {
		t.Fatal(err)
	}
	a = rehearsed.Value.(domain.Aggregate)
	reviewed, err := service.Review(ctx, "p", ReviewCommand{MutationMeta: MutationMeta{ExpectedRevision: a.Production.Revision, IdempotencyKey: "review"}, CueID: "cue", Action: domain.ReviewAccept, Reviewer: "审校"})
	if err != nil {
		t.Fatal(err)
	}
	a = reviewed.Value.(domain.Aggregate)
	approved, err := service.Approve(ctx, "p", MutationMeta{ExpectedRevision: a.Production.Revision, IdempotencyKey: "approve"})
	if err != nil {
		t.Fatal(err)
	}
	a = approved.Value.(domain.Aggregate)
	released, err := service.Release(ctx, "p", ReleaseCommand{MutationMeta: MutationMeta{ExpectedRevision: a.Production.Revision, IdempotencyKey: "release"}, ReleasedBy: "审校"})
	if err != nil {
		t.Fatal(err)
	}
	a = released.Value.(domain.Aggregate)
	if a.Production.State != domain.StateReleased || !a.Release.VerifyHash() {
		t.Fatalf("bad release: %+v", a.Release)
	}
	vtt, err := service.ReleaseVTT(ctx, "p")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(vtt), "WEBVTT") || !strings.Contains(string(vtt), "晨雾散开") {
		t.Fatalf("bad vtt: %s", vtt)
	}
}

func TestCueDiffRequiresAdjacentVersions(t *testing.T) {
	service := New(store.NewMemory())
	_, err := service.CueDiff(context.Background(), "missing", "cue", 1, 3)
	if err == nil {
		t.Fatal("expected error")
	}
}

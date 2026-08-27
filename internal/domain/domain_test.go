package domain

import (
	"testing"
	"time"
)

func testAggregate(t *testing.T) Aggregate {
	t.Helper()
	a, err := NewProduction("p1", "影片", "zh-CN", 10000, 25, []Participant{{Name: "编剧", Role: RoleWriter}, {Name: "排演", Role: RolePerformer}, {Name: "审校", Role: RoleReviewer}}, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestProductionRequiresAllRoles(t *testing.T) {
	_, err := NewProduction("p", "影片", "zh-CN", 1000, 25, []Participant{{Name: "编剧", Role: RoleWriter}}, time.Now())
	if err == nil {
		t.Fatal("expected missing roles error")
	}
}

func TestCueRevisionInvalidatesRehearsalAndDecision(t *testing.T) {
	a := testAggregate(t)
	if err := a.Production.Transition(StateTimelined); err != nil {
		t.Fatal(err)
	}
	if err := a.AddCueVersion(NarrationCue{ID: "cue", WindowStartMS: 0, WindowEndMS: 8000, Text: "灯塔亮起", PlannedCharsPerSecond: 5}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := a.Production.Transition(StateRehearsing); err != nil {
		t.Fatal(err)
	}
	cue := a.LatestCues()[0]
	take := RehearsalTake{ID: "take", CueVersionSetHash: CueSetHash([]NarrationCue{cue}), Measurements: []CueMeasurement{{CueID: "cue", CueVersion: 1, ActualStartMS: 0, ActualEndMS: 2000, SpokenDurationMS: 1000}}}
	if err := a.AddRehearsal(take, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := a.AddDecision(ReviewDecision{ID: "revise", CueID: "cue", Action: ReviewRevise, Reason: "补充时段", Reviewer: "审校"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := a.AddCueVersion(NarrationCue{ID: "cue", WindowStartMS: 0, WindowEndMS: 8000, Text: "清晨灯塔亮起", PlannedCharsPerSecond: 5}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if a.Rehearsals[0].InvalidatedAt == nil {
		t.Fatal("old rehearsal should be invalidated")
	}
	if len(a.Decisions) != 0 {
		t.Fatal("old decisions should be cleared")
	}
	if a.Production.State != StateWriting {
		t.Fatalf("state=%s", a.Production.State)
	}
}

func TestStableReleaseHashIgnoresInputOrdering(t *testing.T) {
	cuesA := []ApprovedCue{{ID: "b", StartMS: 2000}, {ID: "a", StartMS: 1000}}
	cuesB := []ApprovedCue{{ID: "a", StartMS: 1000}, {ID: "b", StartMS: 2000}}
	hashA, err := StableReleaseHash("p", 7, cuesA, nil)
	if err != nil {
		t.Fatal(err)
	}
	hashB, err := StableReleaseHash("p", 7, cuesB, nil)
	if err != nil {
		t.Fatal(err)
	}
	if hashA != hashB {
		t.Fatalf("hashes differ: %s %s", hashA, hashB)
	}
}

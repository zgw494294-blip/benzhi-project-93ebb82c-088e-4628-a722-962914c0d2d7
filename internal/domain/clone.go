package domain

func CloneAggregate(in Aggregate) Aggregate {
	out := in
	out.Production.Participants = append([]Participant(nil), in.Production.Participants...)
	out.Segments = append([]TimelineSegment(nil), in.Segments...)
	out.Cues = append([]NarrationCue(nil), in.Cues...)
	out.Rehearsals = append([]RehearsalTake(nil), in.Rehearsals...)
	out.Decisions = append([]ReviewDecision(nil), in.Decisions...)
	out.Validation = append([]ValidationIssue(nil), in.Validation...)
	if in.Release != nil {
		release := *in.Release
		release.ApprovedCues = append([]ApprovedCue(nil), in.Release.ApprovedCues...)
		release.ReviewDecisions = append([]ReviewDecision(nil), in.Release.ReviewDecisions...)
		out.Release = &release
	}
	return out
}

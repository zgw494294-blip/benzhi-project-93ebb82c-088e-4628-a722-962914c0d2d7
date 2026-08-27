package domain

import "time"

type Role string

const (
	RoleWriter    Role = "WRITER"
	RolePerformer Role = "PERFORMER"
	RoleReviewer  Role = "REVIEWER"
)

type Participant struct {
	Name string `json:"name"`
	Role Role   `json:"role"`
}

type Production struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Language     string          `json:"language"`
	DurationMS   int64           `json:"duration_ms"`
	FrameRate    float64         `json:"frame_rate"`
	State        ProductionState `json:"state"`
	Revision     int64           `json:"revision"`
	Participants []Participant   `json:"participants"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type SegmentKind string

const (
	SegmentScene    SegmentKind = "SCENE"
	SegmentDialogue SegmentKind = "DIALOGUE"
	SegmentMusic    SegmentKind = "MUSIC_EMPHASIS"
	SegmentOccupied SegmentKind = "OCCUPIED"
)

type TimelineSegment struct {
	ID           string      `json:"id"`
	ProductionID string      `json:"production_id"`
	SceneID      string      `json:"scene_id,omitempty"`
	Kind         SegmentKind `json:"kind"`
	StartMS      int64       `json:"start_ms"`
	EndMS        int64       `json:"end_ms"`
	Label        string      `json:"label"`
	Revision     int64       `json:"revision"`
}

type CueStatus string

const (
	CueDraft     CueStatus = "DRAFT"
	CueValidated CueStatus = "VALIDATED"
	CueRehearsed CueStatus = "REHEARSED"
	CueApproved  CueStatus = "APPROVED"
	CueWithdrawn CueStatus = "WITHDRAWN"
)

type NarrationCue struct {
	ID                    string     `json:"id"`
	ProductionID          string     `json:"production_id"`
	WindowStartMS         int64      `json:"window_start_ms"`
	WindowEndMS           int64      `json:"window_end_ms"`
	Version               int        `json:"version"`
	Text                  string     `json:"text"`
	Intent                string     `json:"intent"`
	PlannedCharsPerSecond float64    `json:"planned_chars_per_second"`
	PauseBudgetMS         int64      `json:"pause_budget_ms"`
	Status                CueStatus  `json:"status"`
	WithdrawnAt           *time.Time `json:"withdrawn_at,omitempty"`
	SupersedesVersion     int        `json:"supersedes_version,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}

type CueMeasurement struct {
	CueID                string  `json:"cue_id"`
	CueVersion           int     `json:"cue_version"`
	ActualStartMS        int64   `json:"actual_start_ms"`
	ActualEndMS          int64   `json:"actual_end_ms"`
	SpokenDurationMS     int64   `json:"spoken_duration_ms"`
	PauseMS              int64   `json:"pause_ms"`
	WindowDeltaMS        int64   `json:"window_delta_ms"`
	WindowStartDeltaMS   int64   `json:"window_start_delta_ms"`
	WindowEndDeltaMS     int64   `json:"window_end_delta_ms"`
	ActualCharsPerSecond float64 `json:"actual_chars_per_second"`
	PauseRatio           float64 `json:"pause_ratio"`
}

type FindingSeverity string

const (
	SeverityBlocking FindingSeverity = "BLOCKING"
	SeverityAdvisory FindingSeverity = "ADVISORY"
)

type Finding struct {
	ID       string          `json:"id"`
	CueID    string          `json:"cue_id"`
	Code     string          `json:"code"`
	Message  string          `json:"message"`
	Severity FindingSeverity `json:"severity"`
	Resolved bool            `json:"resolved"`
}

type RehearsalTake struct {
	ID                string                `json:"id"`
	ProductionID      string                `json:"production_id"`
	Round             int                   `json:"round"`
	CueVersionSetHash string                `json:"cue_version_set_hash"`
	StartedAt         time.Time             `json:"started_at"`
	CompletedAt       time.Time             `json:"completed_at"`
	Measurements      []CueMeasurement      `json:"measurements"`
	Findings          []Finding             `json:"findings"`
	Comparisons       []RehearsalComparison `json:"comparisons,omitempty"`
	InvalidatedAt     *time.Time            `json:"invalidated_at,omitempty"`
}

type RehearsalComparison struct {
	CueID                   string `json:"cue_id"`
	PreviousRound           int    `json:"previous_round"`
	CurrentRound            int    `json:"current_round"`
	Comparable              bool   `json:"comparable"`
	SpokenDurationDeltaMS   int64  `json:"spoken_duration_delta_ms,omitempty"`
	WindowDeltaDeltaMS      int64  `json:"window_delta_delta_ms,omitempty"`
	WindowStartDeltaDeltaMS int64  `json:"window_start_delta_delta_ms,omitempty"`
	WindowEndDeltaDeltaMS   int64  `json:"window_end_delta_delta_ms,omitempty"`
	FindingCountDelta       int    `json:"finding_count_delta,omitempty"`
	Note                    string `json:"note,omitempty"`
}

type ReviewAction string

const (
	ReviewAccept ReviewAction = "ACCEPT"
	ReviewRevise ReviewAction = "REVISE"
	ReviewReject ReviewAction = "REJECT"
)

type ReviewDecision struct {
	ID        string       `json:"id"`
	FindingID string       `json:"finding_id,omitempty"`
	CueID     string       `json:"cue_id"`
	Action    ReviewAction `json:"action"`
	Reason    string       `json:"reason,omitempty"`
	Reviewer  string       `json:"reviewer"`
	CreatedAt time.Time    `json:"created_at"`
}

type ValidationIssue struct {
	CueID           string          `json:"cue_id,omitempty"`
	CueVersion      int             `json:"cue_version,omitempty"`
	WindowStartMS   int64           `json:"window_start_ms"`
	WindowEndMS     int64           `json:"window_end_ms"`
	Code            string          `json:"code"`
	Message         string          `json:"message"`
	Severity        FindingSeverity `json:"severity"`
	EstimatedMS     int64           `json:"estimated_ms"`
	UsableMS        int64           `json:"usable_ms"`
	OccupiedStartMS int64           `json:"occupied_start_ms"`
	OccupiedEndMS   int64           `json:"occupied_end_ms"`
}

type ApprovedCue struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	StartMS     int64  `json:"start_ms"`
	EndMS       int64  `json:"end_ms"`
	Text        string `json:"text"`
	Intent      string `json:"intent"`
	EstimatedMS int64  `json:"estimated_ms"`
}

type ReleaseSnapshot struct {
	ID                 string           `json:"id"`
	ProductionID       string           `json:"production_id"`
	ProductionRevision int64            `json:"production_revision"`
	ApprovedCues       []ApprovedCue    `json:"approved_cues"`
	ReviewDecisions    []ReviewDecision `json:"review_decisions"`
	ContentHash        string           `json:"content_hash"`
	ReleasedBy         string           `json:"released_by"`
	ReleasedAt         time.Time        `json:"released_at"`
}

type Aggregate struct {
	Production                  Production        `json:"production"`
	Segments                    []TimelineSegment `json:"segments"`
	Cues                        []NarrationCue    `json:"cues"`
	Rehearsals                  []RehearsalTake   `json:"rehearsals"`
	Decisions                   []ReviewDecision  `json:"review_decisions"`
	Validation                  []ValidationIssue `json:"validation_issues"`
	ValidationBlockingCount     int               `json:"validation_blocking_count"`
	ValidationCueVersionSetHash string            `json:"validation_cue_version_set_hash,omitempty"`
	Release                     *ReleaseSnapshot  `json:"release,omitempty"`
}

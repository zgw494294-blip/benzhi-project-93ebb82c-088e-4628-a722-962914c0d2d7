package workflow

import "benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"

type CreateProductionCommand struct {
	ID             string               `json:"id,omitempty"`
	Title          string               `json:"title"`
	Language       string               `json:"language"`
	DurationMS     int64                `json:"duration_ms"`
	FrameRate      float64              `json:"frame_rate"`
	Participants   []domain.Participant `json:"participants"`
	IdempotencyKey string               `json:"idempotencyKey"`
}

type UpdateProductionCommand struct {
	MutationMeta
	Title        string               `json:"title"`
	Language     string               `json:"language"`
	DurationMS   int64                `json:"duration_ms"`
	FrameRate    float64              `json:"frame_rate"`
	Participants []domain.Participant `json:"participants"`
}

type AddSegmentCommand struct {
	MutationMeta
	ID      string             `json:"id,omitempty"`
	SceneID string             `json:"scene_id,omitempty"`
	Kind    domain.SegmentKind `json:"kind"`
	StartMS int64              `json:"start_ms"`
	EndMS   int64              `json:"end_ms"`
	Label   string             `json:"label"`
}

type UpdateSegmentCommand struct {
	MutationMeta
	SceneID string             `json:"scene_id,omitempty"`
	Kind    domain.SegmentKind `json:"kind"`
	StartMS int64              `json:"start_ms"`
	EndMS   int64              `json:"end_ms"`
	Label   string             `json:"label"`
}

type SaveCueCommand struct {
	MutationMeta
	CueID                 string  `json:"cue_id,omitempty"`
	WindowStartMS         int64   `json:"window_start_ms"`
	WindowEndMS           int64   `json:"window_end_ms"`
	Text                  string  `json:"text"`
	Intent                string  `json:"intent"`
	PlannedCharsPerSecond float64 `json:"planned_chars_per_second"`
	PauseBudgetMS         int64   `json:"pause_budget_ms"`
}

type WithdrawCueCommand struct{ MutationMeta }

type RecordRehearsalCommand struct {
	MutationMeta
	ID                string                  `json:"id,omitempty"`
	CueVersionSetHash string                  `json:"cue_version_set_hash,omitempty"`
	Measurements      []domain.CueMeasurement `json:"measurements"`
	Findings          []domain.Finding        `json:"findings"`
}

type ReviewCommand struct {
	MutationMeta
	ID        string              `json:"id,omitempty"`
	FindingID string              `json:"finding_id,omitempty"`
	CueID     string              `json:"cue_id"`
	Action    domain.ReviewAction `json:"action"`
	Reason    string              `json:"reason,omitempty"`
	Reviewer  string              `json:"reviewer"`
}

type ReleaseCommand struct {
	MutationMeta
	ReleasedBy string `json:"released_by"`
}

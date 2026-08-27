package workflow

import (
	"context"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/timeline"
)

func (s *Service) RecordRehearsal(ctx context.Context, productionID string, cmd RecordRehearsalCommand) (Result, error) {
	if cmd.ID == "" {
		cmd.ID = newID("take")
	}
	saved, duplicate, err := s.repo.Mutate(ctx, productionID, cmd.ExpectedRevision, cmd.IdempotencyKey, "record-rehearsal", func(a *domain.Aggregate) error {
		latest := a.LatestCues()
		measurements := append([]domain.CueMeasurement(nil), cmd.Measurements...)
		findingsInput := append([]domain.Finding(nil), cmd.Findings...)
		for i := range findingsInput {
			if findingsInput[i].ID == "" {
				findingsInput[i].ID = newID("finding")
			}
			if findingsInput[i].Severity == "" {
				findingsInput[i].Severity = domain.SeverityAdvisory
			}
		}
		findings := timeline.EvaluateRehearsal(latest, measurements, findingsInput)
		for i := range measurements {
			m := &measurements[i]
			for _, cue := range latest {
				if cue.ID != m.CueID {
					continue
				}
				m.WindowDeltaMS = m.ActualEndMS - m.ActualStartMS - (cue.WindowEndMS - cue.WindowStartMS)
				m.WindowStartDeltaMS = m.ActualStartMS - cue.WindowStartMS
				m.WindowEndDeltaMS = m.ActualEndMS - cue.WindowEndMS
				chars := timeline.CountReadableCharacters(cue.Text)
				if m.SpokenDurationMS > 0 {
					m.ActualCharsPerSecond = float64(chars) / (float64(m.SpokenDurationMS) / 1000)
				}
				if m.ActualEndMS > m.ActualStartMS {
					m.PauseRatio = float64(m.PauseMS) / float64(m.ActualEndMS-m.ActualStartMS)
				}
				break
			}
		}
		versionHash := cmd.CueVersionSetHash
		if versionHash == "" {
			versionHash = domain.CueSetHash(latest)
		}
		take := domain.RehearsalTake{ID: cmd.ID, CueVersionSetHash: versionHash, Measurements: measurements, Findings: findings}
		return a.AddRehearsal(take, s.now())
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Value: saved, Idempotent: duplicate}, nil
}

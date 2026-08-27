package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

type releaseDigest struct {
	ProductionID string           `json:"production_id"`
	Revision     int64            `json:"production_revision"`
	Cues         []ApprovedCue    `json:"approved_cues"`
	Decisions    []ReviewDecision `json:"review_decisions"`
}

func StableReleaseHash(productionID string, revision int64, cues []ApprovedCue, decisions []ReviewDecision) (string, error) {
	cueCopy := append([]ApprovedCue(nil), cues...)
	decisionCopy := append([]ReviewDecision(nil), decisions...)
	sort.Slice(cueCopy, func(i, j int) bool {
		if cueCopy[i].StartMS == cueCopy[j].StartMS {
			return cueCopy[i].ID < cueCopy[j].ID
		}
		return cueCopy[i].StartMS < cueCopy[j].StartMS
	})
	sort.Slice(decisionCopy, func(i, j int) bool {
		if decisionCopy[i].CueID == decisionCopy[j].CueID {
			return decisionCopy[i].ID < decisionCopy[j].ID
		}
		return decisionCopy[i].CueID < decisionCopy[j].CueID
	})
	b, err := json.Marshal(releaseDigest{ProductionID: productionID, Revision: revision, Cues: cueCopy, Decisions: decisionCopy})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (s ReleaseSnapshot) VerifyHash() bool {
	hash, err := StableReleaseHash(s.ProductionID, s.ProductionRevision, s.ApprovedCues, s.ReviewDecisions)
	return err == nil && hash == s.ContentHash
}

func (a *Aggregate) SetRelease(snapshot ReleaseSnapshot, now time.Time) error {
	if a.Production.State != StateApproved {
		return NewRuleError("state", "仅 APPROVED 状态可以发布")
	}
	if a.Release != nil {
		return NewRuleError("release", "发布快照不可替换")
	}
	if !snapshot.VerifyHash() {
		return NewRuleError("content_hash", "发布摘要校验失败")
	}
	snapshot.ReleasedAt = now.UTC()
	a.Release = &snapshot
	return a.Production.Transition(StateReleased)
}

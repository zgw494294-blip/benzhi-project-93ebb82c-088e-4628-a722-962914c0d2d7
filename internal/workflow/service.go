package workflow

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/store"
)

type Service struct {
	repo          store.Repository
	now           func() time.Time
	minimumGapMS  int64
	minimumMargin int64
	windowsMu     sync.RWMutex
	windowsCache  map[string]cachedWindowsResult
}

func New(repo store.Repository) *Service {
	return &Service{
		repo:          repo,
		now:           time.Now,
		minimumGapMS:  1200,
		minimumMargin: 250,
		windowsCache:  make(map[string]cachedWindowsResult),
	}
}

func newID(prefix string) string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return prefix + "-" + hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return prefix + "-" + hex.EncodeToString(raw)
}

type MutationMeta struct {
	ExpectedRevision int64  `json:"expectedRevision"`
	IdempotencyKey   string `json:"idempotencyKey"`
}

type Result struct {
	Value      any  `json:"value"`
	Idempotent bool `json:"idempotent"`
}

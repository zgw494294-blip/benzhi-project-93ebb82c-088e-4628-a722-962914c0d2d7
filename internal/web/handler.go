package web

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/workflow"
)

//go:embed assets/*
var assets embed.FS

type Handler struct {
	service *workflow.Service
	mux     *http.ServeMux
	started time.Time
}

func New(service *workflow.Service) *Handler {
	h := &Handler{service: service, mux: http.NewServeMux(), started: time.Now().UTC()}
	h.routes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Cache-Control", "no-store")
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) routes() {
	staticRoot, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	h.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(staticRoot))))
	h.mux.HandleFunc("GET /", h.HandleIndex)
	h.mux.HandleFunc("GET /healthz", h.HandleHealth)
	h.mux.HandleFunc("GET /api/productions", h.HandleListProductions)
	h.mux.HandleFunc("POST /api/productions", h.HandleCreateProduction)
	h.mux.HandleFunc("GET /api/productions/{id}", h.HandleGetProduction)
	h.mux.HandleFunc("PUT /api/productions/{id}", h.HandleUpdateProduction)
	h.mux.HandleFunc("POST /api/productions/{id}/segments", h.HandleAddSegment)
	h.mux.HandleFunc("PUT /api/productions/{id}/segments/{segmentID}", h.HandleUpdateSegment)
	h.mux.HandleFunc("DELETE /api/productions/{id}/segments/{segmentID}", h.HandleDeleteSegment)
	h.mux.HandleFunc("GET /api/productions/{id}/windows", h.HandleWindows)
	h.mux.HandleFunc("POST /api/productions/{id}/timeline/finalize", h.HandleFinalizeTimeline)
	h.mux.HandleFunc("POST /api/productions/{id}/cues", h.HandleSaveCue)
	h.mux.HandleFunc("DELETE /api/productions/{id}/cues/{cueID}", h.HandleWithdrawCue)
	h.mux.HandleFunc("GET /api/productions/{id}/cues/{cueID}/diff", h.HandleCueDiff)
	h.mux.HandleFunc("POST /api/productions/{id}/validation", h.HandleValidate)
	h.mux.HandleFunc("POST /api/productions/{id}/rehearsals", h.HandleRehearsal)
	h.mux.HandleFunc("POST /api/productions/{id}/reviews", h.HandleReview)
	h.mux.HandleFunc("POST /api/productions/{id}/approve", h.HandleApprove)
	h.mux.HandleFunc("POST /api/productions/{id}/release", h.HandleRelease)
	h.mux.HandleFunc("GET /api/productions/{id}/release/preview", h.HandleReleasePreview)
	h.mux.HandleFunc("GET /api/productions/{id}/release.json", h.HandleReleaseJSON)
	h.mux.HandleFunc("GET /api/productions/{id}/release.vtt", h.HandleReleaseVTT)
}

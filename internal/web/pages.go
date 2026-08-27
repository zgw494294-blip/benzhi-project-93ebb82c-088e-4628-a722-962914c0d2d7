package web

import (
	"net/http"
)

func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := assets.ReadFile("assets/index.html")
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "started_at": h.started})
}

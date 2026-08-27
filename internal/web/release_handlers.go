package web

import (
	"fmt"
	"net/http"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/workflow"
)

func (h *Handler) HandleRehearsal(w http.ResponseWriter, r *http.Request) {
	var cmd workflow.RecordRehearsalCommand
	if !decodeJSON(w, r, &cmd) {
		return
	}
	result, err := h.service.RecordRehearsal(r.Context(), r.PathValue("id"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) HandleReview(w http.ResponseWriter, r *http.Request) {
	var cmd workflow.ReviewCommand
	if !decodeJSON(w, r, &cmd) {
		return
	}
	result, err := h.service.Review(r.Context(), r.PathValue("id"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) HandleApprove(w http.ResponseWriter, r *http.Request) {
	var meta workflow.MutationMeta
	if !decodeJSON(w, r, &meta) {
		return
	}
	result, err := h.service.Approve(r.Context(), r.PathValue("id"), meta)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleRelease(w http.ResponseWriter, r *http.Request) {
	var cmd workflow.ReleaseCommand
	if !decodeJSON(w, r, &cmd) {
		return
	}
	result, err := h.service.Release(r.Context(), r.PathValue("id"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) HandleReleasePreview(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ReleasePreview(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleReleaseJSON(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.ReleaseJSON(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", r.PathValue("id")+".json"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) HandleReleaseVTT(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.ReleaseVTT(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", r.PathValue("id")+".vtt"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

package web

import (
	"net/http"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/workflow"
)

func (h *Handler) HandleAddSegment(w http.ResponseWriter, r *http.Request) {
	var cmd workflow.AddSegmentCommand
	if !decodeJSON(w, r, &cmd) {
		return
	}
	result, err := h.service.AddSegment(r.Context(), r.PathValue("id"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) HandleUpdateSegment(w http.ResponseWriter, r *http.Request) {
	var cmd workflow.UpdateSegmentCommand
	if !decodeJSON(w, r, &cmd) {
		return
	}
	result, err := h.service.UpdateSegment(r.Context(), r.PathValue("id"), r.PathValue("segmentID"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleDeleteSegment(w http.ResponseWriter, r *http.Request) {
	var meta workflow.MutationMeta
	if !decodeJSON(w, r, &meta) {
		return
	}
	result, err := h.service.DeleteSegment(r.Context(), r.PathValue("id"), r.PathValue("segmentID"), meta)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleWindows(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Windows(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleFinalizeTimeline(w http.ResponseWriter, r *http.Request) {
	var meta workflow.MutationMeta
	if !decodeJSON(w, r, &meta) {
		return
	}
	result, err := h.service.FinalizeTimeline(r.Context(), r.PathValue("id"), meta)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleSaveCue(w http.ResponseWriter, r *http.Request) {
	var cmd workflow.SaveCueCommand
	if !decodeJSON(w, r, &cmd) {
		return
	}
	result, err := h.service.SaveCue(r.Context(), r.PathValue("id"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) HandleWithdrawCue(w http.ResponseWriter, r *http.Request) {
	var meta workflow.MutationMeta
	if !decodeJSON(w, r, &meta) {
		return
	}
	result, err := h.service.WithdrawCue(r.Context(), r.PathValue("id"), r.PathValue("cueID"), meta)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleCueDiff(w http.ResponseWriter, r *http.Request) {
	from, err := queryInt(r, "from")
	if err != nil {
		writeError(w, err)
		return
	}
	to, err := queryInt(r, "to")
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.CueDiff(r.Context(), r.PathValue("id"), r.PathValue("cueID"), from, to)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleValidate(w http.ResponseWriter, r *http.Request) {
	var meta workflow.MutationMeta
	if !decodeJSON(w, r, &meta) {
		return
	}
	result, err := h.service.ValidateForRehearsal(r.Context(), r.PathValue("id"), meta)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

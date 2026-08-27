package web

import (
	"net/http"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/workflow"
)

func (h *Handler) HandleListProductions(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListProductions(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"productions": items})
}

func (h *Handler) HandleCreateProduction(w http.ResponseWriter, r *http.Request) {
	var cmd workflow.CreateProductionCommand
	if !decodeJSON(w, r, &cmd) {
		return
	}
	result, err := h.service.CreateProduction(r.Context(), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) HandleGetProduction(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetProduction(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleUpdateProduction(w http.ResponseWriter, r *http.Request) {
	var cmd workflow.UpdateProductionCommand
	if !decodeJSON(w, r, &cmd) {
		return
	}
	result, err := h.service.UpdateProduction(r.Context(), r.PathValue("id"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

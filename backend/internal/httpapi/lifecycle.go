package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
)

type lifecycleRequest struct {
	SeasonID int64  `json:"season_id"`
	Name     string `json:"name"`
}

func readLifecycleRequest(w http.ResponseWriter, r *http.Request) (lifecycleRequest, bool) {
	var req lifecycleRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be a valid JSON object.")
		return req, false
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON object.")
		return req, false
	}
	if req.SeasonID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_season_id", "Season id must be a positive integer.")
		return req, false
	}
	return req, true
}

func (h SeasonHandler) handleSeasonClose(w http.ResponseWriter, r *http.Request) {
	req, ok := readLifecycleRequest(w, r)
	if !ok {
		return
	}
	summary, err := h.seasons.CloseSeason(r.Context(), req.SeasonID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toSeasonSummaryResponse(summary))
}

func (h SeasonHandler) handleNextSeason(w http.ResponseWriter, r *http.Request) {
	req, ok := readLifecycleRequest(w, r)
	if !ok {
		return
	}
	summary, err := h.seasons.CreateNextSeason(r.Context(), req.SeasonID, req.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, h.toSeasonSummaryResponse(summary))
}

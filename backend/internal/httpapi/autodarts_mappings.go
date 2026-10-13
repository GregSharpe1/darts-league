package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

func (h PendingResultHandler) handleMappings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	view, err := h.pending.AutodartsMappings(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h PendingResultHandler) handleMappingWrite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	id, err := strconv.ParseInt(r.PathValue("playerID"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, 400, "invalid_player_id", "Player id must be a positive integer.")
		return
	}
	var body struct {
		SeasonID int64   `json:"season_id"`
		Username *string `json:"username"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	decoder.DisallowUnknownFields()
	if r.Method == http.MethodDelete {
		var deletion struct {
			SeasonID int64 `json:"season_id"`
		}
		err = decoder.Decode(&deletion)
		body.SeasonID = deletion.SeasonID
	} else {
		err = decoder.Decode(&body)
	}
	if err != nil || decoder.Decode(new(json.RawMessage)) != io.EOF || body.SeasonID <= 0 || (r.Method == http.MethodPut && body.Username == nil) {
		writeError(w, 400, "invalid_json", "Supply one valid mapping request with the current season_id.")
		return
	}
	if r.Method == http.MethodDelete {
		if err := h.pending.ClearAutodartsMapping(r.Context(), body.SeasonID, id); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	mapping, err := h.pending.SaveAutodartsMapping(r.Context(), body.SeasonID, id, *body.Username)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapping)
}

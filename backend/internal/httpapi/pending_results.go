package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/greg/darts-league/backend/internal/league"
)

type PendingResultHandler struct {
	pending league.PendingResultService
}

func NewPendingResultHandler(pending league.PendingResultService) PendingResultHandler {
	return PendingResultHandler{pending: pending}
}

func (h PendingResultHandler) RegisterRoutes(mux *http.ServeMux, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/admin/pending-results", requireAdmin(h.handleList))
	mux.HandleFunc("GET /api/admin/pending-results/{pendingID}", requireAdmin(h.handleDetail))
	mux.HandleFunc("POST /api/admin/pending-results/{pendingID}/confirm", requireAdmin(h.handleConfirm))
	mux.HandleFunc("POST /api/admin/pending-results/{pendingID}/reject", requireAdmin(h.handleReject))
}

func pendingResultResponse(pending league.PendingResult) map[string]any {
	response := map[string]any{
		"id":                 pending.ID,
		"external_match_id":  pending.ExternalMatchID,
		"player_one_name":    pending.PlayerOneName,
		"player_one_legs":    pending.PlayerOneLegs,
		"player_one_average": pending.PlayerOneAverage,
		"player_two_name":    pending.PlayerTwoName,
		"player_two_legs":    pending.PlayerTwoLegs,
		"player_two_average": pending.PlayerTwoAverage,
		"status":             pending.Status,
		"received_at":        pending.ReceivedAt.UTC().Format(http.TimeFormat),
	}
	if pending.ConfirmedAt != nil {
		response["confirmed_at"] = pending.ConfirmedAt.UTC().Format(http.TimeFormat)
	}
	if pending.ConfirmedBy != "" {
		response["confirmed_by"] = pending.ConfirmedBy
	}
	return response
}

func (h PendingResultHandler) handleList(w http.ResponseWriter, r *http.Request) {
	pending, err := h.pending.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	response := make([]map[string]any, 0, len(pending))
	for _, entry := range pending {
		response = append(response, pendingResultResponse(entry))
	}
	writeJSON(w, http.StatusOK, map[string]any{"pending_results": response})
}

func (h PendingResultHandler) handleConfirm(w http.ResponseWriter, r *http.Request) {
	pendingID, err := strconv.ParseInt(r.PathValue("pendingID"), 10, 64)
	if err != nil || pendingID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_pending_id", "Pending result id must be a positive integer.")
		return
	}
	var body map[string]json.RawMessage
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}
	if _, ok := body["expected_result"]; !ok {
		writeError(w, http.StatusBadRequest, "expected_result_required", "Supply expected_result, using null for an unscored fixture.")
		return
	}
	if decoder.Decode(new(json.RawMessage)) != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", "Supply one JSON object.")
		return
	}
	b, err := json.Marshal(body)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	var req league.ApprovalRequest
	strict := json.NewDecoder(bytes.NewReader(b))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid approval request.")
		return
	}

	actor := adminActorFromContext(r.Context())
	if actor == "" {
		actor = "admin"
	}

	req.PendingID, req.Actor = pendingID, actor
	result, err := h.pending.Approve(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                 result.ID,
		"fixture_id":         result.FixtureID,
		"player_one_legs":    result.PlayerOneLegs,
		"player_two_legs":    result.PlayerTwoLegs,
		"player_one_average": result.PlayerOneAverage,
		"player_two_average": result.PlayerTwoAverage,
		"winner_id":          result.WinnerID,
	})
}

func (h PendingResultHandler) handleReject(w http.ResponseWriter, r *http.Request) {
	pendingID, err := strconv.ParseInt(r.PathValue("pendingID"), 10, 64)
	if err != nil || pendingID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_pending_id", "Pending result id must be a positive integer.")
		return
	}
	actor := adminActorFromContext(r.Context())
	if actor == "" {
		actor = "admin"
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}
	if err := h.pending.RejectWithReason(r.Context(), pendingID, actor, req.Reason); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

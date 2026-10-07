package httpapi

import (
	"net/http"
	"strconv"
)

func (h PendingResultHandler) handleDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	id, err := strconv.ParseInt(r.PathValue("pendingID"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_pending_id", "Pending result id must be a positive integer.")
		return
	}
	record, err := h.pending.ImportDetail(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pending_result": pendingResultResponse(record.Pending),
		"source":         record.Import.Source, "digest": record.Import.Digest, "changed_import": record.Changed,
		"settings_evidence": record.Import.SettingsEvidence, "review_reason": record.Import.ReviewReason,
		"played_at": record.Import.PlayedAt, "players": record.Import.Players, "detail": record.Import.Detail,
		"original_payload": record.Import.Payload, "season_id": record.SeasonID, "fixture_id": record.FixtureID,
		"result_id": record.ResultID, "source_active": record.Active, "mapping": record.Mapping, "approval": record.Approval,
	})
}

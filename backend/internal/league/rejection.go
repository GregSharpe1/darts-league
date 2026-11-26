package league

import (
	"context"
	"strings"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

func (s PendingResultService) RejectWithReason(ctx context.Context, id int64, actor, reason string) error {
	return transact(ctx, s.store, func(tx Store) error {
		imports, ok := tx.(ApprovalStore)
		if !ok {
			return ErrImportStoreRequired
		}
		r, err := imports.GetImport(ctx, id)
		if err != nil {
			return err
		}
		if r.Pending.Status != PendingResultStatusPending && r.Pending.Status != PendingResultStatusReviewBlocked {
			return ErrPendingResultNotPending
		}
		if strings.TrimSpace(actor) == "" {
			return ErrApprovalConflict
		}
		now := s.now().UTC()
		r.Pending.Status, r.Pending.ConfirmedAt, r.Pending.ConfirmedBy = PendingResultStatusRejected, &now, actor
		meta := &ImportAudit{PendingID: id, Source: r.Import.Source, ExternalMatchID: r.Import.ExternalMatchID, Digest: r.Import.Digest, Reason: strings.TrimSpace(reason)}
		r.Approval = meta
		if r.Import.CompetitionType == autodarts.TournamentMatch {
			return imports.SaveImportApproval(ctx, r)
		}
		season, err := tx.GetActiveSeason(ctx)
		if err != nil {
			return err
		}
		if season.Status == SeasonStatusCompleted || (r.SeasonID != nil && *r.SeasonID != season.ID) {
			return ErrSeasonCompleted
		}
		r.SeasonID = &season.ID
		if err := imports.SaveImportApproval(ctx, r); err != nil {
			return err
		}
		_, err = tx.CreateAuditLog(ctx, AuditLogEntry{SeasonID: season.ID, Action: "import_rejected", Actor: actor, Import: meta, CreatedAt: now})
		return err
	})
}

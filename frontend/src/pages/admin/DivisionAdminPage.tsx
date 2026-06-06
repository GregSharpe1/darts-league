import { useParams } from 'react-router-dom'
import { useAdminDivisionFixtures, useAuditLog, useDivisions, useSaveResult, useSeasonSummary, useUndoResult, formatWhen } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'
import { AdminFixtureCard } from './AdminFixtureCard'
import { AuditEntryCard } from './AuditEntryCard'

export function DivisionAdminPage() {
  const { slug = '' } = useParams()
  const seasonQuery = useSeasonSummary()
  const divisionsQuery = useDivisions()
  const division = divisionsQuery.data?.find((item) => item.slug === slug)
  const fixturesQuery = useAdminDivisionFixtures(slug, true)
  const auditQuery = useAuditLog(slug, true)
  const saveResultMutation = useSaveResult()
  const undoResultMutation = useUndoResult()

  return (
    <>
      <section className="page-intro">
        <span className="eyebrow">Division scoring</span>
        <h1>{division?.name ?? 'Division admin'}</h1>
        <p className="fixture-meta">{seasonQuery.data?.name ?? 'Active season'}</p>
        <p>Scores, audit history, and weekly progress on this page affect only the selected division.</p>
      </section>

      <section className="admin-grid admin-grid-wide">
        <article className="admin-card">
          <h2>Fixtures</h2>
          {fixturesQuery.isLoading ? <StateNotice message="Loading division fixtures..." compact /> : null}
          {fixturesQuery.error ? <StateNotice tone="error" message={readError(fixturesQuery.error)} compact /> : null}
          {fixturesQuery.data?.length === 0 ? <StateNotice message="This division has no fixtures yet." compact /> : null}
          {(fixturesQuery.data ?? []).map((week) => (
            <div className="admin-week" key={week.week_number}>
              <div className="admin-week-header">
                <span className="admin-week-title-group">
                  <strong className="admin-week-title">Week {week.week_number}</strong>
                  <span className="fixture-meta">Reveals {formatWhen(week.reveal_at)}</span>
                </span>
              </div>
              <div className="admin-fixtures">
                {week.fixtures.map((fixture) => (
                  <AdminFixtureCard
                    key={fixture.id}
                    fixture={fixture}
                    onSave={(payload) => saveResultMutation.mutateAsync(payload)}
                    onUndo={(fixtureId) => undoResultMutation.mutateAsync(fixtureId)}
                    isSaving={saveResultMutation.isPending}
                    isUndoing={undoResultMutation.isPending}
                  />
                ))}
              </div>
            </div>
          ))}
        </article>

        <article className="admin-card">
          <h2>Audit log</h2>
          {auditQuery.isLoading ? <StateNotice message="Loading audit history..." compact /> : null}
          {auditQuery.error ? <StateNotice tone="error" message={readError(auditQuery.error)} compact /> : null}
          {auditQuery.data?.length === 0 ? <StateNotice message="No score edits recorded for this division yet." compact /> : null}
          {(auditQuery.data ?? []).map((entry) => <AuditEntryCard key={entry.id} entry={entry} />)}
        </article>
      </section>
    </>
  )
}

import type { FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { ApiError } from '../../lib/api'
import {
  type Tournament,
  type TournamentMatchFormat,
  useAdminTournaments,
  useCreateTournament,
  useDeleteTournament,
  useRemoveTournamentEntry,
  useSetTournamentRegistration,
  useStartTournament,
  useTournamentEntries,
  useUpdateTournamentFormat,
} from '../../lib/tournamentsApi'
import { readError } from '../../lib/utils'
import { TournamentDate } from './TournamentDate'
import { TournamentList } from './TournamentList'
import { TournamentFormat } from './TournamentFormat'

export function AdminTournamentsPage() {
  const { slug } = useParams()
  const tournaments = useAdminTournaments()
  const event = tournaments.data?.find((item) => item.slug === slug)
  const unauthorized = tournaments.error instanceof ApiError && tournaments.error.status === 401

  return (
    <>
      <section className="page-intro">
        <span className="eyebrow">Tournament admin</span>
        <h1>{event?.name ?? 'Tournaments'}</h1>
        {slug ? <Link className="tournament-link" to="/admin/tournaments">All tournaments</Link> : null}
      </section>
      {tournaments.isPending ? <StateNotice message="Loading tournaments..." /> : null}
      {unauthorized ? <Link to="/admin" className="button-link cta-link">Log in to manage tournaments</Link> : null}
      {tournaments.error && !unauthorized ? <StateNotice tone="error" message={readError(tournaments.error)} /> : null}
      {tournaments.isSuccess && slug && !event ? <StateNotice tone="error" message="Tournament was not found." /> : null}
      {event && tournaments.isSuccess ? <TournamentRoster key={event.slug} event={event} /> : null}
      {!slug && tournaments.isSuccess ? (
        <section className="admin-grid">
          <CreateTournament />
          <article className="admin-card">
            <h2>Manage tournaments</h2>
            <TournamentList admin events={tournaments.data.map((item) => ({ ...item, started: Boolean(item.bracket?.length) }))} />
          </article>
        </section>
      ) : null}
    </>
  )
}

function CreateTournament() {
  const navigate = useNavigate()
  const create = useCreateTournament()

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    create.mutate({
      name: String(form.get('name')),
      slug: String(form.get('slug')),
      starts_on: String(form.get('starts-on')),
      ...readMatchFormat(form),
    }, {
      onSuccess: (tournament) => navigate(`/admin/tournaments/${tournament.slug}`),
    })
  }

  return (
    <article className="admin-card">
      <h2>Create tournament</h2>
      <form className="form-preview" onSubmit={submit}>
        <div className="field">
          <label htmlFor="tournament-name">Tournament name</label>
          <input id="tournament-name" name="name" required disabled={create.isPending} />
        </div>
        <div className="field">
          <label htmlFor="tournament-date">Start date</label>
          <input id="tournament-date" name="starts-on" type="date" required disabled={create.isPending} />
        </div>
        <div className="field">
          <label htmlFor="tournament-slug">Event link slug</label>
          <input id="tournament-slug" name="slug" required pattern="[a-z0-9]+(-[a-z0-9]+)*" maxLength={60} placeholder="friday-knockout" aria-describedby="tournament-slug-help" disabled={create.isPending} />
          <span className="fixture-meta" id="tournament-slug-help">Use lowercase letters, digits, and hyphens. This becomes the end of the public link.</span>
        </div>
        <MatchFormatFields disabled={create.isPending} />
        <button type="submit" disabled={create.isPending}>{create.isPending ? 'Creating...' : 'Create tournament'}</button>
      </form>
      {create.error ? <StateNotice tone="error" message={readError(create.error)} compact /> : null}
    </article>
  )
}

function readMatchFormat(form: FormData): TournamentMatchFormat {
  return {
    game_variant: String(form.get('game_variant')),
    legs_to_win: Number(form.get('legs_to_win')),
  }
}

function MatchFormatFields({ format, disabled }: { format?: TournamentMatchFormat; disabled: boolean }) {
  return (
    <>
      <div className="field">
        <label htmlFor="tournament-game-variant">Game variant</label>
        <select id="tournament-game-variant" name="game_variant" defaultValue={format?.game_variant ?? '501'} disabled={disabled}>
          <option value="501">501</option>
          <option value="301">301</option>
        </select>
      </div>
      <div className="field">
        <label htmlFor="tournament-legs-to-win">First to (legs)</label>
        <input id="tournament-legs-to-win" name="legs_to_win" type="number" min={1} max={2_147_483_647} step={1} defaultValue={format?.legs_to_win ?? 3} required disabled={disabled} />
      </div>
    </>
  )
}

function TournamentRoster({ event }: { event: Tournament }) {
  const navigate = useNavigate()
  const entries = useTournamentEntries(event.slug)
  const registration = useSetTournamentRegistration(event.slug)
  const remove = useRemoveTournamentEntry(event.slug)
  const start = useStartTournament(event.slug)
  const deleteTournament = useDeleteTournament(event.slug)
  const updateFormat = useUpdateTournamentFormat(event.slug)
  const started = Boolean(event.bracket)
  const canStart = !event.registration_open && (entries.data?.length ?? 0) >= 2
  const canDelete = event.registration_open && !started
  const tournamentBusy = updateFormat.isPending || start.isPending || deleteTournament.isPending

  async function confirmDelete() {
    if (!window.confirm(`Delete ${event.name} and all its entries? This cannot be undone.`)) return
    try {
      await deleteTournament.mutateAsync()
      navigate('/admin/tournaments', { replace: true })
    } catch {
      // The mutation error is shown below the delete action.
    }
  }

  return (
    <section className="admin-grid">
      <article className="admin-card">
        <h2>{started ? 'Tournament started' : event.registration_open ? 'Registration open' : 'Registration closed'}</h2>
        <p>{started ? 'Event date' : 'Starts'} <TournamentDate startsOn={event.starts_on} /></p>
        {!started && !event.registration_open && entries.isSuccess && !canStart ? <p>At least two entrants are needed to draw the bracket.</p> : null}
        <div className="toolbar-actions tournament-registration-actions">
          <Link className="button-link secondary-button" to={`/tournaments/${event.slug}`}>{started ? 'View bracket' : 'Public sign-up page'}</Link>
          {!started ? (
            <button className={canStart ? 'secondary-button' : undefined} type="button" disabled={registration.isPending || tournamentBusy} onClick={() => registration.mutate(!event.registration_open)}>
              {event.registration_open ? 'Close registration' : 'Open registration'}
            </button>
          ) : null}
          {!started && canStart ? (
            <button type="button" disabled={tournamentBusy || registration.isPending || remove.isPending} onClick={() => {
              if (window.confirm('Start this tournament? This locks the roster and match format and publishes the knockout draw.')) start.mutate()
            }}>
              {start.isPending ? 'Drawing bracket...' : 'Start tournament'}
            </button>
          ) : null}
        </div>
        {registration.error ? <StateNotice tone="error" message={readError(registration.error)} compact /> : null}
        {start.error ? <StateNotice tone="error" message={readError(start.error)} compact /> : null}
        {canDelete ? (
          <div className="tournament-registration-actions">
            <button className="ghost-button" type="button" disabled={tournamentBusy || registration.isPending || remove.isPending} onClick={() => void confirmDelete()}>
              {deleteTournament.isPending ? 'Deleting...' : 'Delete tournament'}
            </button>
          </div>
        ) : null}
        {deleteTournament.error ? <StateNotice tone="error" message={readError(deleteTournament.error)} compact /> : null}
        <div className="tournament-registration-actions">
          <h3>Match format</h3>
          {started ? <TournamentFormat format={event} /> : (
            <form className="form-preview" onSubmit={(submission) => {
              submission.preventDefault()
              updateFormat.mutate(readMatchFormat(new FormData(submission.currentTarget)))
            }}>
              <MatchFormatFields key={`${event.game_variant}-${event.legs_to_win}`} format={event} disabled={tournamentBusy} />
              <button type="submit" disabled={tournamentBusy}>{updateFormat.isPending ? 'Saving...' : 'Save match format'}</button>
            </form>
          )}
          {updateFormat.error ? <StateNotice tone="error" message={readError(updateFormat.error)} compact /> : null}
          {updateFormat.isSuccess ? <StateNotice message="Match format saved." compact /> : null}
        </div>
      </article>
      <article className="admin-card">
        <h2>Entrants{entries.data ? ` (${entries.data.length})` : ''}</h2>
        {entries.isPending ? <StateNotice message="Loading entrants..." compact /> : null}
        {entries.error ? <StateNotice tone="error" message={readError(entries.error)} compact /> : null}
        {remove.error ? <StateNotice tone="error" message={readError(remove.error)} compact /> : null}
        {entries.isSuccess && entries.data.length === 0 ? <p>No entries yet.</p> : null}
        {entries.isSuccess && entries.data.length > 0 ? (
          <ul className="check-list">
            {entries.data.map((entry) => (
              <li key={entry.id}>
                <div className="tournament-list-label">
                  <strong>{entry.display_name}</strong>
                  {entry.nickname ? <div className="fixture-meta">{entry.nickname}</div> : null}
                </div>
                {!started ? <button
                  className="ghost-button tournament-remove-entry"
                  type="button"
                  aria-label={`Remove ${entry.display_name}`}
                  disabled={remove.isPending || start.isPending || deleteTournament.isPending}
                  onClick={() => {
                    if (window.confirm(`Remove ${entry.display_name} from ${event.name}?`)) remove.mutate(entry.id)
                  }}
                >
                  Remove
                </button> : null}
              </li>
            ))}
          </ul>
        ) : null}
      </article>
    </section>
  )
}

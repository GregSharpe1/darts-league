import type { FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { ApiError } from '../../lib/api'
import {
  type Tournament,
  useAdminTournaments,
  useCreateTournament,
  useRemoveTournamentEntry,
  useSetTournamentRegistration,
  useTournamentEntries,
} from '../../lib/tournamentsApi'
import { readError } from '../../lib/utils'
import { TournamentDate } from './TournamentDate'

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
            {tournaments.data.length === 0 ? <p>No tournaments yet.</p> : (
              <ul className="check-list">
                {tournaments.data.map((item) => (
                  <li key={item.slug}>
                    <div className="tournament-list-label">
                      <Link className="tournament-link" to={`/admin/tournaments/${item.slug}`}>{item.name}</Link>
                      <div className="fixture-meta">
                        <TournamentDate startsOn={item.starts_on} /> | {item.registration_open ? 'Registration open' : 'Registration closed'}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            )}
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
        <button type="submit" disabled={create.isPending}>{create.isPending ? 'Creating...' : 'Create tournament'}</button>
      </form>
      {create.error ? <StateNotice tone="error" message={readError(create.error)} compact /> : null}
    </article>
  )
}

function TournamentRoster({ event }: { event: Tournament }) {
  const entries = useTournamentEntries(event.slug)
  const registration = useSetTournamentRegistration(event.slug)
  const remove = useRemoveTournamentEntry(event.slug)

  return (
    <section className="admin-grid">
      <article className="admin-card">
        <h2>{event.registration_open ? 'Registration open' : 'Registration closed'}</h2>
        <p>Starts <TournamentDate startsOn={event.starts_on} /></p>
        <div className="toolbar-actions tournament-registration-actions">
          <Link className="button-link secondary-button" to={`/tournaments/${event.slug}`}>Public sign-up page</Link>
          <button type="button" disabled={registration.isPending} onClick={() => registration.mutate(!event.registration_open)}>
            {event.registration_open ? 'Close registration' : 'Open registration'}
          </button>
        </div>
        {registration.error ? <StateNotice tone="error" message={readError(registration.error)} compact /> : null}
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
                <button
                  className="ghost-button tournament-remove-entry"
                  type="button"
                  aria-label={`Remove ${entry.display_name}`}
                  disabled={remove.isPending}
                  onClick={() => {
                    if (window.confirm(`Remove ${entry.display_name} from ${event.name}?`)) remove.mutate(entry.id)
                  }}
                >
                  Remove
                </button>
              </li>
            ))}
          </ul>
        ) : null}
      </article>
    </section>
  )
}

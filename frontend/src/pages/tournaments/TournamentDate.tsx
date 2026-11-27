const dateFormat = new Intl.DateTimeFormat('en-GB', { dateStyle: 'long', timeZone: 'UTC' })

export function TournamentDate({ date }: { date: string }) {
  return <time dateTime={date}>{dateFormat.format(new Date(`${date}T00:00:00Z`))}</time>
}

export function TournamentDateRange({ startsOn, endsOn }: { startsOn: string; endsOn?: string }) {
  return <>
    <TournamentDate date={startsOn} />
    {endsOn && endsOn !== startsOn ? <> - <TournamentDate date={endsOn} /></> : null}
  </>
}

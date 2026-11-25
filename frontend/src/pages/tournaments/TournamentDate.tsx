const dateFormat = new Intl.DateTimeFormat('en-GB', { dateStyle: 'long', timeZone: 'UTC' })

export function TournamentDate({ startsOn }: { startsOn: string }) {
  return <time dateTime={startsOn}>{dateFormat.format(new Date(`${startsOn}T00:00:00Z`))}</time>
}

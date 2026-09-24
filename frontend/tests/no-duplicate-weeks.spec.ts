import { expect, test } from '@playwright/test'

const players: [string, string][] = [
  ['Luke Humphries', 'The Freeze'],
  ['Michael Smith', 'Bully Boy'],
  ['Gerwyn Price', 'The Iceman'],
  ['Peter Wright', 'Snakebite'],
  ['Nathan Aspinall', 'The Asp'],
  ['Rob Cross', 'Voltage'],
  ['Jonny Clayton', 'The Ferret'],
  ['Damon Heta', 'The Heat'],
  ['Luke Littler', 'The Nuke'],
  ['Gary Anderson', 'The Flying Scotsman'],
  ['Dave Chisnall', 'Chizzy'],
  ['James Wade', 'The Machine'],
  ['Michael van Gerwen', 'Mighty Mike'],
  ['Dimitri Van den Bergh', 'The DreamMaker'],
  ['Joe Cullen', 'The Rockstar'],
  ['Danny Noppert', 'The Freeze NL'],
  ['Josh Rock', 'Rocky'],
  ['Chris Dobey', 'Hollywood'],
  ['Stephen Bunting', 'The Bullet'],
  ['Callan Rydz', 'The Riot'],
  ['Ryan Searle', 'Heavy Metal'],
  ['Andrew Gilding', 'Goldfinger'],
  ['Brendan Dolan', 'The History Maker'],
  ['Kim Huybrechts', 'The Hurricane'],
  ['Krzysztof Ratajski', 'The Polish Eagle'],
  ['Ross Smith', 'Smudger'],
  ['Martin Schindler', 'The Wall'],
  ['Raymond van Barneveld', 'Barney'],
  ['Dirk van Duijvenbode', 'The Titan'],
  ['Ricardo Pietreczko', 'Pikachu'],
  ['Gian van Veen', 'The Giant'],
  ['Mike De Decker', 'The Real Deal'],
]

test('32 players in one division: no match pairing appears in more than one week', async ({ page }) => {
  test.setTimeout(180_000)

  for (const [displayName, nickname] of players) {
    await page.goto('/register')
    await expect(page.getByLabel('Display name')).toBeEnabled()
    await page.getByLabel('Display name').fill(displayName)
    await page.getByLabel('Nickname').fill(nickname)
    await page.getByRole('button', { name: /register for the league/i }).click()
    await expect(page.getByText(new RegExp(`${nickname.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')} is registered and waiting for division assignment`, 'i'))).toBeVisible()
  }

  await page.goto('/admin')
  await page.getByLabel('Username').fill('admin')
  await page.getByLabel('Password').fill('change-me')
  await page.getByRole('button', { name: /unlock admin tools/i }).click()
  await expect(page.getByRole('heading', { name: /divisions/i })).toBeVisible()

  await page.getByLabel('Division count').fill('1')
  await page.getByRole('button', { name: /create divisions/i }).click()
  await expect(page.getByLabel('Division name').first()).toBeVisible()

  const divisionSelects = page.locator('select[aria-label$=" division"]')
  for (let index = 0; index < players.length; index++) {
    await divisionSelects.nth(index).selectOption({ label: 'Division 1' })
  }

  await page.getByRole('button', { name: /start season/i }).click()
  await expect(page.getByText(/registration is locked/i)).toBeVisible()

  const fixturesData = await page.evaluate(async () => {
    const response = await fetch('/api/admin/divisions/division-1/fixtures', { credentials: 'include' })
    return response.json() as Promise<{
      weeks: {
        week_number: number
        fixtures: { player_one: string; player_two: string }[]
      }[]
    }>
  })

  const pairToWeek = new Map<string, number>()
  let totalFixtures = 0
  const duplicates: string[] = []

  for (const week of fixturesData.weeks) {
    const playersThisWeek = new Set<string>()

    for (const fixture of week.fixtures) {
      totalFixtures++
      const pair = [fixture.player_one, fixture.player_two].sort().join(' vs ')
      if (pairToWeek.has(pair)) {
        duplicates.push(`"${pair}" appears in week ${pairToWeek.get(pair)} AND week ${week.week_number}`)
      } else {
        pairToWeek.set(pair, week.week_number)
      }

      if (playersThisWeek.has(fixture.player_one)) {
        duplicates.push(`${fixture.player_one} plays twice in week ${week.week_number}`)
      }
      playersThisWeek.add(fixture.player_one)

      if (playersThisWeek.has(fixture.player_two)) {
        duplicates.push(`${fixture.player_two} plays twice in week ${week.week_number}`)
      }
      playersThisWeek.add(fixture.player_two)
    }
  }

  const expectedFixtures = (players.length * (players.length - 1)) / 2
  expect(totalFixtures).toBe(expectedFixtures)
  expect(fixturesData.weeks.length).toBe(players.length - 1)
  expect(duplicates).toEqual([])
})

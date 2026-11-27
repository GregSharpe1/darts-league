import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import { execSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

import { expect, test, type Page } from '@playwright/test'
import type { Tournament } from '../src/lib/tournamentsApi'
import { expectAdminActionLayout, expectMatchingAdminLayout, expectMatchingControlHeights, expectMatchingDivisionWidths, expectSingleRowNavigation } from './form-control-checks'

const captureUiScreenshots = process.env.CAPTURE_UI_SCREENSHOTS === '1'
const currentDirectory = path.dirname(fileURLToPath(import.meta.url))

function sanitizePathSegment(value: string): string {
  return value.replace(/[^a-zA-Z0-9._-]+/g, '-').replace(/^-+|-+$/g, '') || 'local'
}

function getBranchName(): string {
  try {
    const branchName = execSync('git rev-parse --abbrev-ref HEAD', {
      cwd: path.resolve(currentDirectory, '..', '..'),
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore'],
    }).trim()

    if (!branchName || branchName === 'HEAD') {
      return 'local'
    }

    return sanitizePathSegment(branchName)
  } catch {
    return 'local'
  }
}

const screenshotDirectory = path.resolve(currentDirectory, '..', '..', 'docs', 'pr-screenshots', getBranchName())

async function captureScreenshot(page: Page, fileName: string) {
  if (!captureUiScreenshots) {
    return
  }

  await mkdir(screenshotDirectory, { recursive: true })
  await page.screenshot({ path: path.join(screenshotDirectory, fileName), fullPage: true })
}

test('register, assign divisions, start season, enter result, and view division standings', async ({ page }) => {
  const leagueName = 'Cardiff Premier League'
  await page.goto('/')
  const brandIcon = page.getByRole('link', { name: 'Go to home page' }).locator('img')
  await expect(brandIcon).toHaveJSProperty('naturalWidth', 96)
  await expect(page.locator('link[rel="icon"]')).toHaveAttribute('href', '/favicon.png')
  const homeLeague = page.getByRole('region', { name: 'League matches' })
  const homeTournaments = page.getByRole('region', { name: 'Tournaments' })
  const homeToolbar = page.getByRole('navigation', { name: 'Primary' })
  await expect(homeToolbar.getByRole('link')).toHaveText(['Records', 'Player stats', 'Admin'])
  await expect(homeToolbar.getByRole('link', { name: 'Records' })).toHaveAttribute('href', '/players/records')
  await expect(homeToolbar.getByRole('link', { name: 'Admin' })).toHaveAttribute('href', '/admin')
  await expect(homeTournaments.getByRole('link', { name: 'Browse tournaments' })).toHaveAttribute('href', '/tournaments')
  await expect(homeLeague.getByRole('link', { name: 'Browse divisions' })).toHaveAttribute('href', '/divisions')
  await expect(homeLeague.getByRole('link', { name: 'Register for the league' })).toBeVisible()
  await expect(homeTournaments.getByText('No tournaments are accepting entries right now.')).toBeVisible()
  await captureScreenshot(page, 'home-competitions.png')
  await page.setViewportSize({ width: 320, height: 850 })
  await expectSingleRowNavigation(page, 3)
  await page.setViewportSize({ width: 375, height: 850 })
  await expectSingleRowNavigation(page, 3)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await captureScreenshot(page, 'home-competitions-mobile.png')
  await page.setViewportSize({ width: 1280, height: 720 })
  await homeLeague.getByRole('link', { name: 'Browse divisions' }).click()
  await expect(page).toHaveURL(/\/divisions$/)
  await expect(page.getByRole('heading', { name: 'League divisions', level: 1 })).toBeVisible()
  await captureScreenshot(page, 'league-divisions.png')
  const leagueNavigation = page.getByRole('navigation', { name: 'Primary' })
  await expect(leagueNavigation.getByRole('link', { name: 'Register', exact: true })).toHaveAttribute('href', '/register')
  await expect(leagueNavigation.getByRole('link', { name: 'Tournaments', exact: true })).toHaveAttribute('href', '/tournaments')
  const registerAction = page.locator('.hero-actions').first().getByRole('link', { name: 'Register', exact: true })
  await expect(registerAction).toHaveAttribute('href', '/register')
  await expect(registerAction).toHaveCSS('background-image', /linear-gradient/)
  const players = [
    ['Luke Humphries', 'The Freeze'],
    ['Michael Smith', 'Bully Boy'],
    ['Gerwyn Price', 'The Iceman'],
    ['Peter Wright', 'Snakebite'],
    ['Nathan Aspinall', 'The Asp'],
    ['Rob Cross', 'Voltage'],
    ['Jonny Clayton', 'The Ferret'],
    ['Damon Heta', 'The Heat'],
  ]

  await page.goto('/register')
  const registrationRequests: string[] = []
  page.on('request', (request) => {
    if (request.url().endsWith('/api/players/register')) registrationRequests.push(request.url())
  })
  await expect(page.getByLabel('Display name')).toBeEnabled()
  await page.getByLabel('Display name').fill('a'.repeat(61))
  await page.getByLabel('Nickname').fill('b'.repeat(31))
  await page.getByRole('button', { name: /register for the league/i }).click()
  await expect(page.getByLabel('Display name')).toHaveAttribute('aria-invalid', 'true')
  await expect(page.getByLabel('Nickname')).toHaveAttribute('aria-invalid', 'true')
  expect(registrationRequests).toHaveLength(0)

  for (const [displayName, nickname] of players) {
    await page.goto('/register')
    await expect(page.getByLabel('Display name')).toBeEnabled()
    await page.getByLabel('Display name').fill(displayName)
    await page.getByLabel('Nickname').fill(nickname)
    await page.getByRole('button', { name: /register for the league/i }).click()
    await expect(page.getByText(new RegExp(`${nickname} is registered and waiting for division assignment`, 'i'))).toBeVisible()
  }

  await captureScreenshot(page, 'register-open.png')
  await page.setViewportSize({ width: 375, height: 850 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await captureScreenshot(page, 'league-register-mobile.png')
  await page.setViewportSize({ width: 1280, height: 720 })
  await expectMatchingControlHeights(page)

  await page.goto('/admin')
  await page.getByLabel('Username').fill('admin')
  await page.getByLabel('Password').fill('change-me')
  await expectMatchingControlHeights(page)
  await page.getByRole('button', { name: /unlock admin tools/i }).click()

  await expect(page.getByRole('heading', { name: /league settings/i })).toBeVisible()
  await expectMatchingControlHeights(page)
  await expectMatchingAdminLayout(page)
  await page.getByLabel('League name').fill(leagueName)
  await page.getByRole('button', { name: /save config/i }).click()
  await expect(page.getByText(leagueName, { exact: true })).toBeVisible()

  await page.getByLabel('Division count').fill('2')
  await page.getByRole('button', { name: /create divisions/i }).click()

  const divisionNameInputs = page.getByLabel('Division name')
  await expect(divisionNameInputs.first()).toBeVisible()
  await divisionNameInputs.nth(0).fill('Premier Division')
  await page.getByLabel('Slack public channel').nth(0).fill('CPREMIER')
  await page.getByRole('button', { name: /save division/i }).nth(0).click()

  await divisionNameInputs.nth(1).fill('Challenger Division')
  await page.getByLabel('Slack public channel').nth(1).fill('CCHALLENGER')
  await page.getByRole('button', { name: /save division/i }).nth(1).click()

  const divisionSelects = page.locator('select[aria-label$=" division"]')
  for (let index = 0; index < 4; index++) {
    await divisionSelects.nth(index).selectOption({ label: 'Premier Division' })
  }
  for (let index = 4; index < 8; index++) {
    await divisionSelects.nth(index).selectOption({ label: 'Challenger Division' })
  }

  await captureScreenshot(page, 'admin-pre-start.png')
  await expectMatchingControlHeights(page)
  await expectAdminActionLayout(page)
  await expectMatchingDivisionWidths(page)

  await page.getByRole('button', { name: /start season/i }).click()
  await expect(page.getByText(/registration is locked/i)).toBeVisible()
  await expect(page.getByRole('button', { name: /start season/i })).toHaveCount(0)
  const unavailableClose = page.getByRole('button', { name: 'Close league', exact: true })
  await expect(unavailableClose).toBeDisabled()
  await expect(unavailableClose).toHaveCSS('background-image', 'none')
  await expect(unavailableClose).toHaveCSS('cursor', 'not-allowed')
  const closeWidth = await unavailableClose.evaluate((button) => button.getBoundingClientRect().width / parseFloat(getComputedStyle(document.documentElement).fontSize))
  expect(closeWidth).toBeGreaterThanOrEqual(12)
  const adminCards = page.locator('.admin-settings-grid > article').evaluateAll((cards) => cards.slice(0, 2).map(card => card.getBoundingClientRect().height))
  const [settingsHeight, divisionsHeight] = await adminCards
  expect(settingsHeight).toBeCloseTo(divisionsHeight)
  const mappings = page.getByRole('region', { name: 'Autodarts mappings' })
  const mappingPlayer = mappings.getByRole('combobox', { name: 'League player' })
  const mappingUsername = mappings.getByLabel('Autodarts username')
  const mappingOption = mappingPlayer.getByRole('option', { name: /Luke Humphries/ })
  const mappingLabel = (await mappingOption.textContent())!
  await mappingPlayer.selectOption({ label: mappingLabel })
  await mappingUsername.fill('smoke-luke')
  await mappings.getByRole('button', { name: 'Save mapping' }).click()
  const savedMappings = mappings.getByRole('list', { name: 'Mapped players' })
  await expect(savedMappings.getByText('smoke-luke', { exact: true })).toBeVisible()
  await expect(mappingOption).toHaveCount(0)
  await savedMappings.getByRole('button', { name: /Edit mapping for .*Luke Humphries/ }).click()
  await expect(mappingUsername).toBeFocused()
  await expect(mappingUsername).toHaveValue('smoke-luke')
  await mappingUsername.fill('smoke-luke-edited')
  await mappings.getByRole('button', { name: 'Save mapping' }).click()
  await expect(savedMappings.getByText('smoke-luke-edited', { exact: true })).toBeVisible()
  await savedMappings.getByRole('button', { name: /Remove mapping for .*Luke Humphries/ }).click()
  await expect(mappingOption).toHaveCount(1)
  await captureScreenshot(page, 'admin-post-start.png')

  await page.goto('/divisions')
  const deadline = page.getByRole('region', { name: 'Final week and catch-up deadline' })
  await expect(deadline.getByText('Finish outstanding fixtures by')).toBeVisible()
  await expect(deadline.locator('time')).toHaveText([/09:00 UK time/, /09:00 UK time/])
  await expect(deadline.getByText(/autodarts bots to represent both players/i)).toBeVisible()
  await page.setViewportSize({ width: 375, height: 650 })
  await page.evaluate(() => window.scrollTo({ top: document.body.scrollHeight, behavior: 'instant' }))
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBeGreaterThan(0)
  await page.getByRole('link', { name: 'View fixtures', exact: true }).last().click()
  await expect(page).toHaveURL(/\/divisions\/division-2$/)
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0)
  await page.setViewportSize({ width: 1280, height: 720 })


  await page.goto('/admin/divisions/division-1')
  await expect(page.getByRole('heading', { name: /premier division/i })).toBeVisible()
  const editedFixture = page.locator('#p1-1').locator('xpath=ancestor::article[1]')
  await page.locator('#p1-1').fill('3')
  await page.locator('#p2-1').fill('1')
  await page.locator('#a1-1').fill('96.4')
  await page.locator('#a2-1').fill('89.1')
  await editedFixture.getByRole('button', { name: /save score/i }).click()
  await expect(page.getByText(/score saved/i)).toBeVisible()

  await page.route('**/api/divisions/division-1/fixtures', async (route) => {
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        current_week: 1,
        weeks: [
          {
            week_number: 1,
            status: 'unlocked',
            reveal_at: '2026-03-23T09:00:00Z',
            fixtures: [
              {
                id: 1,
                player_one: 'The Freeze (Luke Humphries)',
                player_two: 'Bully Boy (Michael Smith)',
                scheduled_at: '2026-03-24T19:30:00Z',
                game_variant: '501',
                legs_to_win: 3,
                result: { player_one_legs: 3, player_two_legs: 1, player_one_average: 96.4, player_two_average: 89.1, winner_id: 1 },
              },
            ],
          },
          {
            week_number: 2,
            status: 'locked',
            reveal_at: '2026-03-30T09:00:00Z',
            fixtures: [{ id: 2, player_one: 'I knew you\'d look', player_two: 'Nothing to see here' }],
          },
        ],
      }),
    })
  })

  await page.goto('/divisions/division-1')
  await expect(page.getByRole('heading', { name: 'Premier Division' })).toBeVisible()
  await expect(page.getByText(/every unlocked fixture in this division has been played so far/i)).toBeVisible()
  const recordedResults = page.getByRole('region', { name: 'Recorded results' })
  const recordedWeek = recordedResults.getByRole('button', { name: 'Week 1', exact: true })
  await expect(recordedWeek).toHaveAttribute('aria-expanded', 'false')
  const fixtureSearch = page.getByRole('combobox', { name: "Find a player's games" })
  await fixtureSearch.fill('luke')
  await expect(recordedWeek).toHaveAttribute('aria-expanded', 'true')
  await expect(recordedResults.getByRole('link', { name: /view match: the freeze/i })).toBeVisible()
  await fixtureSearch.clear()
  await fixtureSearch.blur()
  await expect(recordedWeek).toHaveAttribute('aria-expanded', 'false')
  await captureScreenshot(page, 'public-post-start.png')

  await page.goto('/divisions/division-1/standings')
  await expect(leagueNavigation.getByRole('link', { name: 'Tournaments', exact: true })).toBeVisible()
  await expect(leagueNavigation.getByRole('link', { name: 'Register', exact: true })).toHaveCount(0)
  await expect(page.getByText('The Freeze')).toBeVisible()
  await expect(page.getByText('Luke Humphries')).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'LW' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'LL' })).toBeVisible()
  await captureScreenshot(page, 'standings-post-start.png')

  await page.getByRole('link', { name: 'Go to home page' }).click()
  await page.getByRole('link', { name: 'Player stats' }).click()
  await expect(page.getByRole('heading', { name: 'Player stats', exact: true })).toBeVisible()
  await expect(page.getByRole('status')).toHaveText('8 profiles')
  await captureScreenshot(page, 'player-directory.png')
  const playerSearch = page.getByRole('searchbox', { name: 'Search by player name' })
  await playerSearch.fill('lUkE   hUm')
  await expect(page.getByRole('status')).toHaveText('1 profile')
  for (const width of [375, 768, 1280]) {
    await page.setViewportSize({ width, height: 850 })
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    if (width === 375) await captureScreenshot(page, 'player-directory-mobile.png')
  }
  await playerSearch.fill('no such player')
  await expect(page.getByRole('status')).toHaveText('No players match your search.')
  await playerSearch.fill('freeze')
  await page.getByRole('link', { name: /The Freeze/ }).click()
  await expect(page.getByRole('heading', { name: 'The Freeze', exact: true })).toBeVisible()
  const bestPlacements = page.getByRole('region', { name: 'Best placements' })
  await expect(bestPlacements.getByText('No final league placement yet.')).toBeVisible()
  await expect(bestPlacements.getByText('No recorded tournament finish yet.')).toBeVisible()
  await captureScreenshot(page, 'player-placements.png')

  await page.goto('/register')
  await expect(page.getByText(/registration closed/i)).toBeVisible()
  await expect(page.getByText(/the active season has already started/i)).toBeVisible()

  await page.unroute('**/api/divisions/division-1/fixtures')
  const standingsBefore: string[] = []
  for (const division of ['division-1', 'division-2']) {
    await page.goto(`/admin/divisions/${division}`)
    await expect(page.locator('.admin-fixture-card')).toHaveCount(6)
    const remaining = page.locator('.admin-fixture-card:not(.recorded)')
    while (await remaining.count() > 0) {
      const count = await remaining.count()
      await remaining.first().locator('input[id^="p1-"]').fill('3')
      await remaining.first().locator('input[id^="p2-"]').fill('1')
      await remaining.first().getByRole('button', { name: 'Save score' }).click()
      await expect(remaining).toHaveCount(count - 1)
    }
    await page.goto(`/divisions/${division}/standings`)
    await expect(page.locator('tbody tr')).toHaveCount(4)
    standingsBefore.push(await page.locator('tbody').innerText())
    if (division === 'division-1') {
      await page.goto('/admin')
      await expect(page.getByRole('button', { name: 'Close league' })).toBeDisabled()
    }
  }
  await page.goto('/admin')
  await expect(page.getByRole('button', { name: 'Close league' })).toBeEnabled()
  await expect(page.getByRole('button', { name: 'Close league' })).toHaveCSS('background-image', /linear-gradient/)
  const closeBounds = await page.getByRole('button', { name: 'Close league' }).boundingBox()
  const helpBounds = await page.locator('#close-league-help').boundingBox()
  if (!closeBounds || !helpBounds) throw new Error('Close controls must be visible')
  expect(closeBounds.x).toBeGreaterThanOrEqual(helpBounds.x + helpBounds.width)
  const lifecycleGaps = await page.getByRole('region', { name: 'Season lifecycle' }).evaluate((card) => {
    const previous = card.previousElementSibling
    const next = card.nextElementSibling
    if (!previous || !next) throw new Error('Expected surrounding admin sections')
    return {
      above: card.getBoundingClientRect().top - previous.getBoundingClientRect().bottom,
      below: next.getBoundingClientRect().top - card.getBoundingClientRect().bottom,
    }
  })
  expect(lifecycleGaps.below).toBeGreaterThan(0)
  expect(lifecycleGaps.above).toBeCloseTo(lifecycleGaps.below)
  page.once('dialog', (dialog) => dialog.dismiss())
  await page.getByRole('button', { name: 'Close league' }).click()
  await expect(page.getByRole('button', { name: 'Close league' })).toBeEnabled()
  page.once('dialog', (dialog) => dialog.accept())
  await page.getByRole('button', { name: 'Close league' }).click()
  await expect(page.getByRole('heading', { name: 'League completed' })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('heading', { name: 'League completed' })).toBeVisible()
  const lockedRoster = page.locator('.locked-roster')
  await expect(lockedRoster).not.toHaveAttribute('open', '')
  await expect(lockedRoster.getByText('The Freeze (Luke Humphries)', { exact: true })).toBeHidden()
  await lockedRoster.locator('summary').click()
  await expect(lockedRoster.getByText('The Freeze (Luke Humphries)', { exact: true })).toBeVisible()
  await lockedRoster.locator('summary').click()
  for (const width of [375, 768, 1280]) {
    await page.setViewportSize({ width, height: 900 })
    const nextName = await page.getByLabel('Next league name').boundingBox()
    const openRegistration = await page.getByRole('button', { name: 'Open next season registration' }).boundingBox()
    if (!nextName || !openRegistration) throw new Error('Next-season controls must be visible')
    expect(Math.abs(openRegistration.height - nextName.height)).toBeLessThan(1)
    expect(Math.abs(openRegistration.width - nextName.width)).toBeLessThan(1)
    await captureScreenshot(page, `admin-completed-${width}.png`)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  }
  for (const [index, division] of ['division-1', 'division-2'].entries()) {
    await page.goto(`/admin/divisions/${division}`)
    await expect(page.locator('.admin-fixture-card')).toHaveCount(6)
    await expect(page.getByRole('button', { name: 'Save score' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Undo result' })).toHaveCount(0)
    await expect(page.locator('input[id^="p1-"]').first()).toHaveAttribute('readonly', '')
    await page.goto(`/divisions/${division}/standings`)
    await expect(page.locator('tbody')).toHaveText(standingsBefore[index].replace(/\s+/g, ' ').trim(), { useInnerText: true })
    if (index === 0) {
      for (const width of [375, 768, 1280]) {
        await page.setViewportSize({ width, height: 900 })
        await captureScreenshot(page, `standings-completed-${width}.png`)
      }
    }
  }
  await page.goto('/divisions')
  await expect(page.getByText('League completed. Final results remain available.')).toBeVisible()
  await expect(page.getByRole('heading', { name: "View the previous league's scores here" })).toBeVisible()
  await expect(page.getByText('Monday 09:00 unlocks')).toHaveCount(0)
  await page.goto('/admin')
  await page.getByLabel('Next league name').fill('Next League')
  page.once('dialog', (dialog) => dialog.accept())
  await page.getByRole('button', { name: 'Open next season registration' }).click()
  await expect(page.getByRole('button', { name: 'Start season' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Start season' })).toBeDisabled()
  await expect(page.getByRole('button', { name: 'Start season' })).toHaveCSS('background-image', 'none')
  await expect(page.getByRole('button', { name: 'Start season' })).toHaveCSS('cursor', 'not-allowed')
  await expect(page.getByText(/no players registered yet/i)).toBeVisible()
  await expect(page.getByRole('link', { name: 'Open scoring page' })).toHaveCount(0)
  await page.goto('/register')
  const returningLeague = page.getByRole('link', { name: 'Returning player? Select your profile' })
  await expect(returningLeague).toHaveCSS('color', 'rgb(246, 241, 235)')
  await returningLeague.hover()
  await expect(returningLeague).toHaveCSS('color', 'rgb(246, 241, 235)')
  await returningLeague.click()
  await expect(page.getByRole('button', { name: 'Register for the league' })).toBeDisabled()
  const leagueSearch = page.getByLabel('Search previous players by first name or nickname')
  await leagueSearch.fill('LuKe')
  await expect(page.getByLabel('Previous player profile').getByRole('option', { name: /The Freeze/ })).toHaveCount(1)
  await leagueSearch.fill('freeze')
  await page.getByLabel('Previous player profile').selectOption({ label: `The Freeze / ${leagueName}` })
  const returningIdentity = Number(await page.getByLabel('Previous player profile').inputValue())
  await captureScreenshot(page, 'returning-league.png')
  await page.setViewportSize({ width: 375, height: 900 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await captureScreenshot(page, 'returning-league-mobile.png')
  await page.setViewportSize({ width: 1280, height: 900 })
  await page.getByRole('button', { name: 'Register for the league' }).click()
  await expect(page.getByText('The Freeze is registered and waiting for division assignment.')).toBeVisible()
  await page.reload()
  const eligibleLeaguePlayers = page.getByLabel('Previous player profile')
  await expect(eligibleLeaguePlayers.getByRole('option', { name: /The Iceman/ })).toHaveCount(1)
  await expect(eligibleLeaguePlayers.getByRole('option', { name: /The Freeze/ })).toHaveCount(0)
  await page.goto('/divisions')
  await expect(page.getByText(/no divisions have been created yet/i)).toBeVisible()
  await page.goto('/register')
  await page.getByLabel('Display name').fill('Michael Smith')
  await page.getByRole('button', { name: 'Register for the league' }).click()
  await expect(page.getByText('Michael Smith is registered and waiting for division assignment.')).toBeVisible()
  await page.goto('/admin')
  await page.getByLabel('Division count').fill('1')
  await page.getByRole('button', { name: 'Create divisions' }).click()
  await expect(page.getByLabel('Division name')).toHaveValue('Division 1')
  const newAssignments = page.locator('select[aria-label$=" division"]')
  await expect(newAssignments).toHaveCount(2)
  for (let index = 0; index < 2; index++) await newAssignments.nth(index).selectOption({ label: 'Division 1' })
  await page.getByRole('button', { name: 'Start season' }).click()
  await expect(page.getByRole('button', { name: 'Close league' })).toBeDisabled()
  await page.goto('/admin/divisions/division-1')
  await expect(page.locator('.admin-fixture-card')).toHaveCount(1)
  await expect(page.locator('.admin-fixture-card.recorded')).toHaveCount(0)
  await expect(page.locator('input[id^="p1-"]')).toHaveValue('')
  const profiles = await (await page.request.get('/api/players')).json()
  const freeze = profiles.filter((profile: { name: string }) => profile.name === 'The Freeze')
  expect(freeze).toEqual([expect.objectContaining({ identity_id: returningIdentity, competition: 'Next League' })])
  await page.goto('/players?q=freeze')
  await page.getByRole('link', { name: /The Freeze/ }).click()
  const seasons = page.getByRole('navigation', { name: 'League seasons' })
  const current = seasons.getByRole('link', { name: 'Next League' })
  const earlier = seasons.getByRole('link', { name: leagueName })
  await expect(seasons.getByRole('link')).toHaveText([leagueName, 'Next League'])
  await expect(current).toHaveAttribute('aria-current', 'page')
  await captureScreenshot(page, 'player-season-navigation.png')
  await earlier.click()
  await expect(earlier).toHaveAttribute('aria-current', 'page')
  await expect(page).toHaveURL(/\/seasons\/1\/players\//)
  await expect(page.getByRole('region', { name: 'Match history' })).toBeVisible()
  await page.setViewportSize({ width: 375, height: 900 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await captureScreenshot(page, 'player-season-navigation-mobile.png')
  await current.click()
  await expect(current).toHaveAttribute('aria-current', 'page')

  const leader = { profile: freeze[0], rank: 1, value: 3, played: 4, known_matches: 2 }
  await page.route('**/api/players/records', route => route.fulfill({
    json: {
      total_180: [leader],
      highest_finish: [{ ...leader, value: 170 }],
      bullseyes: [{ ...leader, value: 1 }],
      match_average: [{ ...leader, value: 96.4 }],
      best_leg: [],
      wins: [{ ...leader, known_matches: 4 }],
    },
  }))
  await page.getByRole('link', { name: 'Player stats', exact: true }).click()
  await expect(page.getByRole('link', { name: 'Overall records' })).toHaveAttribute('href', '/players/records')
  await page.getByRole('link', { name: 'Go to home page' }).click()
  await homeToolbar.getByRole('link', { name: 'Records', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Overall records' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Highest checkout' }).getByText('170')).toBeVisible()
  await expect(page.getByRole('region', { name: 'Highest match average' }).getByText('96.40')).toBeVisible()
  await expect(page.getByRole('region', { name: 'Recorded bullseyes' }).getByText('2 of 4 matches with data')).toBeVisible()
  await expect(page.getByText('No complete winning legs have been recorded.')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await captureScreenshot(page, 'player-records-mobile.png')
  await page.setViewportSize({ width: 1280, height: 900 })
  await captureScreenshot(page, 'player-records.png')
  await page.getByRole('region', { name: 'Most 180s' }).getByRole('link', { name: 'The Freeze' }).click()
  await expect(page.getByRole('heading', { name: 'The Freeze' })).toBeVisible()
})

test('all five division navigation actions fit on one line without scrolling', async ({ page }) => {
  await page.route('**/api/season', route => route.fulfill({ json: { registration_open: true, status: 'registration_open' } }))
  await page.goto('/divisions/premier')
  for (const width of [320, 375, 480, 481, 640, 768, 1181, 1280]) {
    await page.setViewportSize({ width, height: 850 })
    await expectSingleRowNavigation(page, 5)
  }
})

test('admins open a tournament and manage public entries independently of the league', async ({ page, browser }) => {
  const eventName = 'Cardiff Winter Knockout'
  const eventPath = '/tournaments/cardiff-winter-knockout'
  const leagueBefore = await (await page.request.get('/api/season')).json()

  await page.goto('/admin/tournaments')
  await expect(page.getByRole('link', { name: 'Log in to manage tournaments' })).toBeVisible()
  await page.getByRole('link', { name: 'Log in to manage tournaments' }).click()
  await page.getByLabel('Username').fill('admin')
  await page.getByLabel('Password').fill('change-me')
  await page.getByRole('button', { name: 'Unlock admin tools' }).click()
  await page.getByRole('link', { name: 'Tournaments', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Create tournament' })).toBeVisible()
  await captureScreenshot(page, 'tournament-admin-create.png')
  await page.getByLabel('Tournament name').fill(eventName)
  await page.getByLabel('Start date').fill('2027-02-06')
  await page.getByLabel('End date (optional)').fill('2027-02-16')
  await expect(page.getByLabel('Event link slug')).toHaveCount(0)
  await page.getByLabel('Game variant').selectOption('301')
  await page.getByLabel('First to (legs)').fill('5')
  await page.getByRole('button', { name: 'Create tournament' }).click()
  await expect(page).toHaveURL(/\/admin\/tournaments\/cardiff-winter-knockout$/)
  await expect(page.getByRole('heading', { name: eventName })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Public sign-up page' })).toHaveAttribute('href', eventPath)
  await expect(page.getByRole('heading', { name: 'Registration closed' })).toBeVisible()

  const publicPage = await browser.newPage({ baseURL: new URL(page.url()).origin })
  try {
    await publicPage.goto('/')
    const navigation = publicPage.getByRole('navigation', { name: 'Primary' })
    await publicPage.getByRole('link', { name: 'Browse tournaments', exact: true }).click()
    const directory = publicPage.getByRole('region', { name: 'Tournament directory' })
    const signupLink = directory.getByRole('link', { name: `Sign up for ${eventName}`, exact: true })
    await expect(navigation.getByRole('link')).toHaveText(['League'])
    await navigation.getByRole('link', { name: 'League', exact: true }).click()
    await expect(publicPage).toHaveURL(/\/divisions$/)
    await navigation.getByRole('link', { name: 'Tournaments', exact: true }).click()
    await expect(directory.getByRole('heading', { name: 'Public tournament boards' })).toBeVisible()
    await expect(directory.getByText(/registration closed/i)).toBeVisible()
    await expect(directory.getByText('16 February 2027')).toBeVisible()
    await expect(signupLink).toHaveCount(0)
    await directory.getByRole('link', { name: eventName, exact: true }).click()
    await expect(publicPage.getByRole('heading', { name: 'Registration closed' })).toBeVisible()
    await expect(navigation.getByRole('link')).toHaveText(['League'])
    const publicFormat = publicPage.getByRole('region', { name: 'Match format' })
    await expect(publicFormat.getByText('301, first to 5 legs', { exact: true })).toBeVisible()
    const rules = publicPage.getByRole('region', { name: 'Tournament rules' })
    await expect(rules.getByText('16 February 2027')).toBeVisible()
    await expect(rules.getByText(/organiser arranges Autodarts simulations/)).toBeVisible()
    await expect(rules.getByRole('list')).toHaveCSS('border-left-width', '2px')
    await expect(rules.getByRole('listitem').last()).toHaveCSS('border-left-width', '0px')

    await page.getByLabel('End date (optional)').fill('2027-02-18')
    await page.getByRole('button', { name: 'Save end date' }).click()
    await expect(page.getByText('End date saved.')).toBeVisible()
    await publicPage.reload()
    await expect(rules.getByText('18 February 2027')).toBeVisible()
    await page.getByLabel('Game variant').selectOption('501')
    await page.getByLabel('First to (legs)').fill('7')
    await page.getByRole('button', { name: 'Save match format' }).click()
    await expect(page.getByText('Match format saved.')).toBeVisible()
    await page.getByRole('button', { name: 'Open registration' }).click()
    await expect(page.getByRole('heading', { name: 'Registration open' })).toBeVisible()

    await publicPage.goto('/')
    const tournamentOverview = publicPage.getByRole('region', { name: 'Tournaments' })
    await expect(tournamentOverview.getByRole('link', { name: `Sign up for ${eventName}` })).toHaveAttribute('href', eventPath)
    await expect(tournamentOverview.getByRole('link', { name: eventName, exact: true })).toHaveCSS('color', 'rgb(255, 255, 255)')
    await captureScreenshot(publicPage, 'home-open-tournaments.png')
    await tournamentOverview.getByRole('link', { name: 'Browse tournaments', exact: true }).click()
    await expect(directory.getByText(/registration open/i)).toBeVisible()
    await expect(signupLink).toBeVisible()
    await captureScreenshot(publicPage, 'tournaments-directory.png')
    await publicPage.setViewportSize({ width: 375, height: 850 })
    expect(await publicPage.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    await signupLink.click()
    await expect(publicPage.getByText('6 February 2027')).toBeVisible()
    await expect(publicFormat.getByText('501, first to 7 legs', { exact: true })).toBeVisible()
    await expect(publicPage.getByRole('button', { name: 'Enter tournament' })).toBeEnabled()
    await publicPage.setViewportSize({ width: 1280, height: 720 })
    await captureScreenshot(publicPage, 'tournament-signup-open.png')
    await publicPage.setViewportSize({ width: 375, height: 850 })
    expect(await publicPage.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    await expect(publicPage.getByRole('region', { name: 'Player sign-up' }).getByRole('button', { name: 'Enter tournament' })).toBeVisible()
    await captureScreenshot(publicPage, 'tournament-signup-mobile.png')
    await publicPage.getByLabel('Display name').fill('Luke Humphries')
    await publicPage.getByLabel('Nickname').fill('The Freeze')
    await publicPage.getByRole('button', { name: 'Enter tournament' }).click()
    await expect(publicPage.getByText('The Freeze is registered for Cardiff Winter Knockout.')).toBeVisible()

    await publicPage.getByLabel('Display name').fill('luke  HUMPHRIES')
    await publicPage.getByRole('button', { name: 'Enter tournament' }).click()
    await expect(publicPage.getByText('Display name is already registered for this tournament.')).toBeVisible()
    await page.reload()
    await expect(page.getByRole('heading', { name: 'Entrants (1)' })).toBeVisible()
    await expect(page.getByText('The Freeze', { exact: true })).toBeVisible()
    await captureScreenshot(page, 'tournament-admin-roster.png')
    await page.setViewportSize({ width: 375, height: 850 })
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)

    await page.getByRole('button', { name: 'Close registration' }).click()
    await expect(page.getByRole('heading', { name: 'Registration closed' })).toBeVisible()
    await publicPage.getByLabel('Display name').fill('Maya Evans')
    await publicPage.getByRole('button', { name: 'Enter tournament' }).click()
    await expect(publicPage.getByRole('heading', { name: 'Registration closed' })).toBeVisible()
    await publicPage.setViewportSize({ width: 1280, height: 720 })
    await captureScreenshot(publicPage, 'tournament-signup-closed.png')

    page.once('dialog', (dialog) => dialog.accept())
    await page.getByRole('button', { name: 'Remove Luke Humphries' }).click()
    await expect(page.getByText('No entries yet.')).toBeVisible()
    await page.getByRole('button', { name: 'Open registration' }).click()
    await expect(page.getByRole('heading', { name: 'Registration open' })).toBeVisible()
    for (const [display_name, nickname] of [['Alex Morgan', ''], ['Sam Patel', ''], ['Maya Evans', 'Sharp Shot']]) {
      const response = await publicPage.request.post(`/api${eventPath}/entries`, { data: { display_name, nickname } })
      expect(response.status()).toBe(201)
    }
    await page.reload()
    await expect(page.getByRole('heading', { name: 'Entrants (3)' })).toBeVisible()
    await page.getByRole('button', { name: 'Close registration' }).click()
    page.once('dialog', (dialog) => dialog.accept())
    await page.getByRole('button', { name: 'Start tournament' }).click()
    await expect(page.getByRole('heading', { name: 'Tournament started' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Open registration' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Save match format' })).toHaveCount(0)
    await page.getByRole('link', { name: 'View bracket' }).click()
    await expect(page.getByRole('heading', { name: 'Knockout bracket' })).toBeVisible()
    await expect(page.getByRole('region', { name: 'Match format' }).getByText('501, first to 7 legs', { exact: true })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Final', exact: true })).toBeVisible()
    await expect(page.getByText('Bye', { exact: true })).toBeVisible()
    await captureScreenshot(page, 'tournament-bracket.png')
    await publicPage.getByRole('link', { name: 'All tournaments', exact: true }).click()
    await expect(directory.getByText(/tournament started/i)).toBeVisible()
    await expect(signupLink).toHaveCount(0)
    await directory.getByRole('link', { name: `View bracket for ${eventName}`, exact: true }).click()
    await expect(publicPage.getByRole('heading', { name: 'Knockout bracket' })).toBeVisible()
    await publicPage.goto('/players?q=Sharp')
    await publicPage.getByRole('link', { name: /Sharp Shot/ }).click()
    await expect(publicPage.getByRole('heading', { name: 'Sharp Shot' })).toBeVisible()
    await expect(publicPage.getByRole('region', { name: 'Best placements' })).toBeVisible()
    const leagueAfter = await (await page.request.get('/api/season')).json()
    expect(leagueAfter).toEqual(leagueBefore)

    expect((await publicPage.request.get('/api/admin/player-identities')).status()).toBe(401)
    const secondCup = await page.request.post('/api/admin/tournaments', {
      data: { name: 'Cardiff Spring Knockout', starts_on: '2027-04-10', game_variant: '501', legs_to_win: 3 },
    })
    expect(secondCup.status()).toBe(201)
    const { slug: secondSlug } = await secondCup.json()
    expect((await page.request.put(`/api/admin/tournaments/${secondSlug}/registration`, { data: { registration_open: true } })).ok()).toBe(true)
    for (const display_name of ['Alex Morgan', 'Sam Patel']) {
      expect((await publicPage.request.post(`/api/tournaments/${secondSlug}/entries`, { data: { display_name } })).status()).toBe(201)
    }
    await publicPage.goto(`/tournaments/${secondSlug}`)
    const returningCup = publicPage.getByRole('link', { name: 'Returning player? Select your profile' })
    await expect(returningCup).toHaveCSS('color', 'rgb(246, 241, 235)')
    await returningCup.click()
    const selection = publicPage.getByLabel('Previous player profile')
    await expect(selection.getByRole('option', { name: /Sharp Shot/ })).toHaveCount(1)
    await expect(selection.getByRole('option', { name: /Alex Morgan|Sam Patel/ })).toHaveCount(0)
    const tournamentSearch = publicPage.getByLabel('Search previous players by first name or nickname')
    await tournamentSearch.fill('MaYa')
    await expect(selection.getByRole('option', { name: /Sharp Shot/ })).toHaveCount(1)
    await expect(selection.getByRole('option', { name: 'Choose a player' })).toHaveCount(1)
    await expect(selection.getByRole('option')).toHaveCount(2)
    await tournamentSearch.fill('sharp shot')
    await selection.selectOption({ label: `Sharp Shot / ${eventName}` })
    await captureScreenshot(publicPage, 'returning-tournament.png')
    const identity_id = Number(await selection.inputValue())
    await publicPage.getByRole('button', { name: 'Enter tournament' }).click()
    await expect(publicPage.getByText('Sharp Shot is registered for Cardiff Spring Knockout.')).toBeVisible()
    expect((await publicPage.request.post(`/api/tournaments/${secondSlug}/returning-entry`, { data: { identity_id } })).status()).toBe(409)
    const eligible = await (await publicPage.request.get(`/api/tournaments/${secondSlug}/returning-entry`)).json()
    expect(eligible.map((profile: { identity_id: number }) => profile.identity_id)).not.toContain(identity_id)
    await publicPage.reload()
    await expect(publicPage.getByText('Loading previous players...')).toHaveCount(0)
    await expect(selection.getByRole('option', { name: /Sharp Shot/ })).toHaveCount(0)
    await page.goto('/admin')
    await page.getByRole('link', { name: 'Player identity review' }).click()
    const suggestions = page.getByRole('region', { name: 'Possible matches' })
    const alex = suggestions.getByRole('listitem').filter({ has: page.getByRole('button', { name: 'Link profiles' }) })
      .filter({ hasText: 'Alex Morgan' })
    const sam = suggestions.getByRole('listitem').filter({ has: page.getByRole('button', { name: 'Keep separate' }) })
      .filter({ hasText: 'Sam Patel' })
    const review = await (await page.request.get('/api/admin/player-identities')).json()
    expect(review.entries).toContainEqual(expect.objectContaining({
      identity_id, display_name: 'Maya Evans', competition: 'Cardiff Spring Knockout',
    }))
    await expect(suggestions.getByRole('listitem').filter({ hasText: 'Maya Evans' })).toHaveCount(0)
    await expect(alex).toContainText('Same display name')
    await expect(sam).toContainText(eventName)
    for (const width of [375, 768, 1280]) {
      await page.setViewportSize({ width, height: 900 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
      if (width !== 768) await captureScreenshot(page, `player-identity-review-${width}.png`)
    }
    page.once('dialog', dialog => dialog.accept())
    await alex.getByRole('button', { name: 'Link profiles' }).click()
    await expect(page.getByText('Player profiles linked.')).toBeVisible()
    await sam.getByRole('button', { name: 'Keep separate' }).click()
    await expect(page.getByText('Profiles marked as different people.')).toBeVisible()
    await page.reload()
    await expect(suggestions.getByRole('heading', { name: 'Possible matches' })).toBeVisible()
    await expect(alex).toHaveCount(0)
    await expect(sam).toHaveCount(0)

    const finished: Tournament = await (await page.request.get(`/api${eventPath}`)).json()
    const rounds = finished.bracket!
    const semiFinal = rounds[0].find(match => !match.slots.some(slot => slot.bye))!
    const final = rounds[1][0]
    const advanced = semiFinal.slots[0]
    semiFinal.legs = [7, 2]
    semiFinal.winner_id = advanced.entry_id
    final.slots = final.slots.map(slot => slot.winner_of === semiFinal.number ? { ...slot, ...advanced } : slot) as typeof final.slots
    await page.route('**/api/admin/tournaments', route => route.fulfill({ json: [finished] }))
    await page.goto(`/admin${eventPath}`)
    await expect(page.getByRole('heading', { name: 'Tournament started' })).toBeVisible()
    await expect(page.getByText('Event dates 6 February 2027 - 18 February 2027')).toBeVisible()
    final.legs = [7, 4]
    final.winner_id = final.slots[0].entry_id
    await page.reload()
    await expect(page.getByRole('heading', { name: 'Tournament finished' })).toBeVisible()
    await expect(page.getByText('Event dates 6 February 2027 - 18 February 2027')).toBeVisible()
    await captureScreenshot(page, 'tournament-admin-finished.png')
    await page.getByRole('link', { name: 'All tournaments' }).click()
    const tournamentRow = page.getByRole('listitem').filter({ hasText: eventName })
    await expect(tournamentRow).toContainText('6 February 2027 - 18 February 2027 | Tournament finished')
    await captureScreenshot(page, 'tournament-admin-directory.png')
  } finally {
    await publicPage.close()
  }
})

test('an admin can confirm deletion of an open tournament and its entries', async ({ page }) => {
  const name = 'Short-lived Cup'
  const slug = 'short-lived-cup'
  const publicPath = `/tournaments/${slug}`

  await page.goto('/admin')
  await page.getByLabel('Username').fill('admin')
  await page.getByLabel('Password').fill('change-me')
  await page.getByRole('button', { name: 'Unlock admin tools' }).click()
  await page.getByRole('link', { name: 'Tournaments', exact: true }).click()
  await page.getByLabel('Tournament name').fill(name)
  await page.getByLabel('Start date').fill('2027-02-06')
  await page.getByRole('button', { name: 'Create tournament' }).click()
  await expect(page.getByRole('heading', { name })).toBeVisible()
  const deleteButton = page.getByRole('button', { name: 'Delete tournament' })
  await expect(deleteButton).toHaveCount(0)

  await page.getByRole('button', { name: 'Open registration' }).click()
  await expect(deleteButton).toBeVisible()
  const entry = await page.request.post(`/api${publicPath}/entries`, { data: { display_name: 'Temporary Entrant' } })
  expect(entry.status()).toBe(201)
  await page.reload()
  await expect(page.getByText('Temporary Entrant')).toBeVisible()
  await captureScreenshot(page, 'tournament-admin-delete.png')

  page.once('dialog', (dialog) => dialog.dismiss())
  await deleteButton.click()
  await expect(page.getByText('Temporary Entrant')).toBeVisible()

  page.once('dialog', (dialog) => dialog.accept())
  await deleteButton.click()
  await expect(page).toHaveURL(/\/admin\/tournaments$/)
  await expect(page.getByRole('heading', { name: 'Manage tournaments' })).toBeVisible()
  await expect(page.getByRole('link', { name, exact: true })).toHaveCount(0)
  await page.goto('/tournaments')
  const directory = page.getByRole('region', { name: 'Tournament directory' })
  await expect(directory).toBeVisible()
  await expect(directory.getByRole('link', { name, exact: true })).toHaveCount(0)
  await page.goto(publicPath)
  await expect(page.getByText('Tournament was not found.')).toBeVisible()
})

import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import { execSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

import { expect, test, type Page } from '@playwright/test'
import { expectAdminActionLayout, expectMatchingControlHeights, expectMatchingDivisionWidths } from './form-control-checks'

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

  for (const [displayName, nickname] of players) {
    await page.goto('/register')
    await expect(page.getByLabel('Display name')).toBeEnabled()
    await page.getByLabel('Display name').fill(displayName)
    await page.getByLabel('Nickname').fill(nickname)
    await page.getByRole('button', { name: /register for the league/i }).click()
    await expect(page.getByText(new RegExp(`${nickname} is registered and waiting for division assignment`, 'i'))).toBeVisible()
  }

  await captureScreenshot(page, 'register-open.png')
  await expectMatchingControlHeights(page)

  await page.goto('/admin')
  await page.getByLabel('Username').fill('admin')
  await page.getByLabel('Password').fill('change-me')
  await expectMatchingControlHeights(page)
  await page.getByRole('button', { name: /unlock admin tools/i }).click()

  await expect(page.getByRole('heading', { name: /league settings/i })).toBeVisible()
  await expectMatchingControlHeights(page)
  await page.getByLabel('League name').fill(leagueName)
  await page.getByRole('button', { name: /save config/i }).click()
  await expect(page.getByText(leagueName)).toBeVisible()

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
  await captureScreenshot(page, 'admin-post-start.png')

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
  await captureScreenshot(page, 'public-post-start.png')

  await page.goto('/divisions/division-1/standings')
  await expect(page.getByText('The Freeze')).toBeVisible()
  await expect(page.getByText('Luke Humphries')).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'LW' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'LL' })).toBeVisible()
  await captureScreenshot(page, 'standings-post-start.png')

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
  await page.goto('/')
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
  await page.getByLabel('Display name').fill('Luke Humphries')
  await page.getByRole('button', { name: 'Register for the league' }).click()
  await expect(page.getByText('Luke Humphries is registered and waiting for division assignment.')).toBeVisible()
  await page.goto('/')
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
})

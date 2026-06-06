import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import { execSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

import { expect, test, type Page } from '@playwright/test'

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

  await page.goto('/admin')
  await page.getByLabel('Username').fill('admin')
  await page.getByLabel('Password').fill('change-me')
  await page.getByRole('button', { name: /unlock admin tools/i }).click()

  await expect(page.getByRole('heading', { name: /league settings/i })).toBeVisible()
  await page.getByLabel('League name').fill(leagueName)
  await page.getByRole('button', { name: /save config/i }).click()
  await expect(page.getByText(leagueName)).toBeVisible()

  await page.getByLabel('Division count').fill('2')
  await page.getByRole('button', { name: /create divisions/i }).click()

  const divisionNameInputs = page.getByLabel('Division name')
  await expect(divisionNameInputs.first()).toBeVisible()
  await divisionNameInputs.nth(0).fill('Premier Division')
  await page.getByLabel('Division slug').nth(0).fill('premier')
  await page.getByLabel('Slack public channel').nth(0).fill('CPREMIER')
  await page.getByRole('button', { name: /save division/i }).nth(0).click()

  await divisionNameInputs.nth(1).fill('Challenger Division')
  await page.getByLabel('Division slug').nth(1).fill('challenger')
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

  await page.getByRole('button', { name: /start season/i }).click()
  await expect(page.getByText(/registration is locked, division names are frozen/i)).toBeVisible()
  await expect(page.getByRole('button', { name: /start season/i })).toBeDisabled()
  await captureScreenshot(page, 'admin-post-start.png')

  await page.goto('/admin/divisions/premier')
  await expect(page.getByRole('heading', { name: /premier division/i })).toBeVisible()
  const editedFixture = page.locator('#p1-1').locator('xpath=ancestor::article[1]')
  await page.locator('#p1-1').fill('3')
  await page.locator('#p2-1').fill('1')
  await page.locator('#a1-1').fill('96.4')
  await page.locator('#a2-1').fill('89.1')
  await editedFixture.getByRole('button', { name: /save score/i }).click()
  await expect(page.getByText(/score saved/i)).toBeVisible()

  await page.route('**/api/divisions/premier/fixtures', async (route) => {
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

  await page.goto('/divisions/premier')
  await expect(page.getByText(/premier division/i)).toBeVisible()
  await expect(page.getByText(/every unlocked fixture in this division has been played so far/i)).toBeVisible()
  await captureScreenshot(page, 'public-post-start.png')

  await page.goto('/divisions/premier/standings')
  await expect(page.getByText('The Freeze')).toBeVisible()
  await expect(page.getByText('Luke Humphries')).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'LW' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'LL' })).toBeVisible()
  await captureScreenshot(page, 'standings-post-start.png')

  await page.goto('/register')
  await expect(page.getByText(/registration closed/i)).toBeVisible()
  await expect(page.getByText(/the active season has already started/i)).toBeVisible()
})

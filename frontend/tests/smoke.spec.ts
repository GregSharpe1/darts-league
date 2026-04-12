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

const screenshotDirectory = path.resolve(
  currentDirectory,
  '..',
  '..',
  'docs',
  'pr-screenshots',
  getBranchName(),
)

async function captureScreenshot(page: Page, fileName: string) {
  if (!captureUiScreenshots) {
    return
  }

  await mkdir(screenshotDirectory, { recursive: true })
  await page.screenshot({ path: path.join(screenshotDirectory, fileName), fullPage: true })
}

test('register, start season, enter result, and view standings', async ({ page }) => {
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
    await page.getByRole('button', { name: /register for this season/i }).click()
    await expect(page.getByText(new RegExp(`${nickname} is in for the active season`, 'i'))).toBeVisible()
  }

  await page.goto('/admin')
  await page.getByLabel('Username').fill('admin')
  await page.getByLabel('Password').fill('change-me')
  await page.getByRole('button', { name: /unlock admin tools/i }).click()

  await expect(page.getByRole('heading', { name: /league settings/i })).toBeVisible()
  await expect(page.getByLabel('League name')).toHaveValue('MVP Season')
  await page.getByLabel('League name').fill(leagueName)
  await page.getByRole('button', { name: /save league name/i }).click()
  await expect(page.getByText(leagueName)).toBeVisible()
  await expect(page.getByLabel('League name')).toHaveValue(leagueName)

  await expect(page.getByRole('heading', { name: /registered players/i })).toBeVisible()
  await expect(page.getByRole('button', { name: /^delete$/i }).first()).toBeVisible()

  await page.goto('/')
  await expect(page.getByText(leagueName)).toBeVisible()

  await page.goto('/register')
  await expect(page.getByText(leagueName)).toBeVisible()
  await captureScreenshot(page, 'register-open.png')

  await page.goto('/admin')
  await captureScreenshot(page, 'admin-pre-start.png')
  await page.getByRole('button', { name: /start season/i }).click()
  await page.getByRole('button', { name: /^start season$/i }).last().click()
  await expect(page.getByText(/registration is locked and player deletion is now disabled/i)).toBeVisible()
  await expect(page.getByText(/roster locked/i).first()).toBeVisible()
  await expect(page.getByRole('button', { name: /start season/i })).toBeDisabled()
  await expect(page.getByLabel('League name')).toBeDisabled()
  await expect(page.getByText(/all settings are locked once the season has started/i)).toBeVisible()
  await captureScreenshot(page, 'admin-post-start.png')

  await expect(page.getByText(/week 1/i)).toBeVisible()
  await page.locator('#p1-1').fill('3')
  await page.locator('#p2-1').fill('1')
  await page.locator('#a1-1').fill('96.4')
  await page.locator('#a2-1').fill('89.1')
  await page.getByRole('button', { name: /save score/i }).first().click()
  await expect(page.getByText(/score saved/i)).toBeVisible()
  await page.getByRole('button', { name: /undo result/i }).first().click()
  await expect(page.getByText(/recorded result removed/i)).toBeVisible()
  await page.locator('#p1-1').fill('3')
  await page.locator('#p2-1').fill('1')
  await page.getByRole('button', { name: /save score/i }).first().click()
  await expect(page.getByText(/score saved/i)).toBeVisible()

  await page.route('**/api/fixtures', async (route) => {
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        current_week: 2,
        weeks: [
          {
            week_number: 1,
            status: 'unlocked',
            reveal_at: '2026-03-23T09:00:00Z',
            fixtures: [
              {
                id: 101,
                player_one: 'The Freeze (Luke Humphries)',
                player_two: 'The Ferret (Jonny Clayton)',
                scheduled_at: '2026-03-24T19:30:00Z',
                game_variant: '501',
                legs_to_win: 3,
              },
            ],
          },
          {
            week_number: 2,
            status: 'unlocked',
            reveal_at: '2026-03-30T09:00:00Z',
            fixtures: [
              {
                id: 102,
                player_one: 'Voltage (Rob Cross)',
                player_two: 'Snakebite (Peter Wright)',
                scheduled_at: '2026-03-31T19:30:00Z',
                game_variant: '501',
                legs_to_win: 3,
              },
            ],
          },
          {
            week_number: 3,
            status: 'locked',
            reveal_at: '2026-04-06T09:00:00Z',
            fixtures: [
              { id: 103, player_one: 'I knew you\'d look', player_two: 'Nothing to see here' },
            ],
          },
        ],
      }),
    })
  })

  await page.goto('/')
  await expect(page.getByRole('button', { name: /week 2/i })).toBeVisible()
  await expect(page.getByRole('button', { name: /week 2/i })).toHaveAttribute('aria-expanded', 'true')
  await expect(page.getByText(/voltage \(rob cross\) vs snakebite \(peter wright\)/i)).toBeVisible()
  await page.getByRole('button', { name: /week 1/i }).click()
  await expect(page.getByText(/the freeze \(luke humphries\) vs the ferret \(jonny clayton\)/i)).toBeVisible()
  await captureScreenshot(page, 'public-post-start.png')

  await page.goto('/standings')
  await expect(page.getByText('The Freeze')).toBeVisible()
  await expect(page.getByText('Luke Humphries')).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'LW' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'LL' })).toBeVisible()
  await captureScreenshot(page, 'standings-post-start.png')

  await expect(page.getByRole('link', { name: /^register$/i })).toHaveCount(0)
  await page.goto('/register')
  await expect(page.getByText(/registration closed/i)).toBeVisible()
  await expect(page.getByText(/the active season has already started/i)).toBeVisible()
})

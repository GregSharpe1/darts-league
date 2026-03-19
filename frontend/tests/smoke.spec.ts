import { expect, test } from '@playwright/test'

test('register, start season, enter result, and view standings', async ({ page }) => {
  const players = [
    ['Luke Humphries', 'The Freeze'],
    ['Michael Smith', 'Bully Boy'],
    ['Gerwyn Price', 'The Iceman'],
    ['Peter Wright', 'Snakebite'],
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

  await expect(page.getByRole('heading', { name: /registered players/i })).toBeVisible()
  await expect(page.getByRole('button', { name: /^delete$/i }).first()).toBeVisible()
  await page.getByRole('button', { name: /start season/i }).click()
  await expect(page.getByText(/registration is locked and player deletion is now disabled/i)).toBeVisible()
  await expect(page.getByText(/roster locked/i).first()).toBeVisible()
  await expect(page.getByRole('button', { name: /start season/i })).toBeDisabled()

  await expect(page.getByText(/week 1/i)).toBeVisible()
  await page.locator('#p1-1').fill('3')
  await page.locator('#p2-1').fill('1')
  await page.locator('#a1-1').fill('96.4')
  await page.locator('#a2-1').fill('89.1')
  await page.getByRole('button', { name: /save score/i }).first().click()
  await expect(page.getByText(/score saved/i)).toBeVisible()
  await page.getByRole('button', { name: /undo result/i }).first().click()
  await expect(page.getByText(/recorded result removed/i)).toBeVisible()
  await page.getByRole('button', { name: /save score/i }).first().click()
  await expect(page.getByText(/score saved/i)).toBeVisible()

  await page.goto('/standings')
  await expect(page.getByText('The Freeze')).toBeVisible()
  await expect(page.getByText('Luke Humphries')).toBeVisible()

  await expect(page.getByRole('link', { name: /^register$/i })).toHaveCount(0)
  await page.goto('/register')
  await expect(page.getByText(/registration closed/i)).toBeVisible()
  await expect(page.getByText(/the active season has already started/i)).toBeVisible()
})

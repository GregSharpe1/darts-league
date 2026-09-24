import { expect, type Page } from '@playwright/test'

export async function expectMatchingControlHeights(page: Page) {
  for (const width of [375, 768, 1280]) {
    await page.setViewportSize({ width, height: 900 })
    const controls = await page.locator('form.admin-login, form.form-preview, .toolbar-actions:has(select)').evaluateAll((groups) => groups.flatMap((group) => {
      const input = group.querySelector('input, select')
      if (!input) return []
      return [...group.querySelectorAll('button')].map((button) => ({
        label: button.textContent?.trim(),
        inputHeight: input.getBoundingClientRect().height,
        buttonHeight: button.getBoundingClientRect().height,
      }))
    }))
    expect(controls.length).toBeGreaterThan(0)
    for (const control of controls) {
      expect(Math.abs(control.inputHeight - control.buttonHeight), `${control.label} at ${width}px`).toBeLessThan(1)
    }
  }
}

export async function expectAdminActionLayout(page: Page) {
  const saveRightGap = await page.getByRole('button', { name: 'Save config' }).evaluate((button) => {
    const form = button.closest('form')
    if (!form) throw new Error('Expected settings form')
    return form.getBoundingClientRect().right - button.getBoundingClientRect().right
  })
  expect(Math.abs(saveRightGap)).toBeLessThan(1)
  const startWidth = await page.getByRole('button', { name: 'Start season' }).evaluate((button) => button.getBoundingClientRect().width / parseFloat(getComputedStyle(document.documentElement).fontSize))
  expect(startWidth).toBeGreaterThanOrEqual(12)
  const divisionWidths = await page.getByRole('link', { name: 'Open scoring page' }).evaluateAll((links) => links.map((link) => {
    const actions = link.parentElement
    const form = link.closest('form')
    if (!actions || !form) throw new Error('Expected division action row')
    return { row: actions.getBoundingClientRect().width, form: form.getBoundingClientRect().width, filled: [...actions.children].reduce((width, action) => width + action.getBoundingClientRect().width, 0) + parseFloat(getComputedStyle(actions).columnGap) }
  }))
  expect(divisionWidths.length).toBeGreaterThan(0)
  for (const widths of divisionWidths) {
    expect(Math.abs(widths.row - widths.form)).toBeLessThan(1)
    expect(Math.abs(widths.filled - widths.row)).toBeLessThan(1)
  }
}

export async function expectMatchingDivisionWidths(page: Page) {
  for (const width of [375, 768, 1024, 1280]) {
    await page.setViewportSize({ width, height: 900 })
    const widths = await page.locator('#division-count, input[id^="division-name-"], input[id^="division-slack-"]').evaluateAll((inputs) => inputs.map((input) => input.getBoundingClientRect().width))
    expect(widths.length).toBeGreaterThanOrEqual(5)
    expect(Math.max(...widths) - Math.min(...widths), `Division field widths at ${width}px`).toBeLessThan(1)
  }
}

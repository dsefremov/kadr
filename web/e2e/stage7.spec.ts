import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { join } from 'node:path'
import { registerVerifyLogin } from './helpers'

test('upload queue page', async ({ page, request }) => {
  await registerVerifyLogin(page, request)
  await page.goto('/uploads')
  await expect(page.getByTestId('upload-queue')).toBeVisible()
  const a11y = await new AxeBuilder({ page }).analyze()
  expect(a11y.violations).toEqual([])
})

test('timeline scrub and viewer', async ({ page, request }) => {
  test.setTimeout(180_000)
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  await registerVerifyLogin(page, request)
  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('timeline-item')).toBeVisible({ timeout: 120_000 })
  await expect(page.getByTestId('timeline-scrub')).toBeVisible()
  await page.getByTestId(/open-viewer-/).first().click()
  await expect(page.getByTestId('viewer')).toBeVisible()
  await page.getByTestId('viewer-info').click()
  await expect(page.getByTestId('viewer-info-panel')).toBeVisible()
  await page.getByTestId('viewer-change-date').click()
  await expect(page.getByTestId('viewer-date-dialog')).toBeVisible()
  const a11y = await new AxeBuilder({ page }).analyze()
  expect(a11y.violations).toEqual([])
})

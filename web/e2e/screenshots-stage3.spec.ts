import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { join } from 'node:path'
import { registerVerifyLogin, screenshotDir } from './helpers'

test('stage 3 screenshots', async ({ page, request }, testInfo) => {
  test.setTimeout(120_000)
  const slug = testInfo.project.name
  const out = screenshotDir('stage-3')
  await registerVerifyLogin(page, request)
  await page.getByTestId('visibility-toggle').click()
  await expect(page.getByTestId('public-feed-confirm')).toBeVisible()
  await page.screenshot({ path: `${out}/public-confirm-${slug}.png`, fullPage: true })
  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations).toEqual([])
  await page.getByTestId('public-feed-confirm-confirm').click()
  await expect(page.getByTestId('visibility-toggle')).toHaveText('публичная')
  await page.screenshot({ path: `${out}/home-public-${slug}.png`, fullPage: true })

  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('upload-progress')).toBeVisible({ timeout: 15_000 })
  await page.screenshot({ path: `${out}/upload-progress-${slug}.png`, fullPage: true })
  await expect(page.getByTestId('upload-message')).toContainText('принята', { timeout: 30_000 })
  await page.screenshot({ path: `${out}/upload-done-${slug}.png`, fullPage: true })
})

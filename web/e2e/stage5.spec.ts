import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { join } from 'node:path'
import { registerVerifyLogin, waitForTimelineReady } from './helpers'

test('duplicate upload returns link to existing media', async ({ page, request }) => {
  test.setTimeout(180_000)
  await registerVerifyLogin(page, request)
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('upload-message')).toContainText('принята', { timeout: 90_000 })
  await waitForTimelineReady(page, 1)

  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('upload-duplicate')).toBeVisible({ timeout: 120_000 })
  await expect(page.getByTestId('upload-duplicate-link')).toHaveAttribute('href', /\/m\/.*preview-640/)
  const count = await page.getByTestId('timeline-item').count()
  expect(count).toBe(1)

  const a11y = await new AxeBuilder({ page }).analyze()
  expect(a11y.violations).toEqual([])
})

test('timeline thumb exposes preview srcset', async ({ page, request }) => {
  test.setTimeout(180_000)
  await registerVerifyLogin(page, request)
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('upload-message')).toContainText('принята', { timeout: 90_000 })
  await waitForTimelineReady(page, 1)
  await expect(page.getByTestId('timeline-thumb')).toBeVisible()
  const srcset = await page.getByTestId('timeline-thumb').getAttribute('srcset')
  expect(srcset).toMatch(/320w/)
  expect(srcset).toMatch(/1280w/)
})

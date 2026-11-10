import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { join } from 'node:path'
import { registerVerifyLogin } from './helpers'

test('trash and restore media', async ({ page, request }) => {
  test.setTimeout(180_000)
  await registerVerifyLogin(page, request)
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('timeline-item')).toBeVisible({ timeout: 120_000 })

  await page.getByTestId(/trash-item-/).click()
  await expect(page.getByTestId('timeline-empty')).toBeVisible({ timeout: 30_000 })

  await page.getByTestId('open-trash').click()
  await expect(page.getByTestId('trash-item')).toBeVisible()
  await page.getByTestId(/restore-/).click()
  await expect(page.getByTestId('trash-message')).toContainText('восстановлен', { timeout: 15_000 })

  await page.getByTestId('trash-back-home').click()
  await expect(page.getByTestId('timeline-item')).toBeVisible({ timeout: 30_000 })

  const a11y = await new AxeBuilder({ page }).analyze()
  expect(a11y.violations).toEqual([])
})

test('duplicate upload when file is in trash offers restore', async ({ page, request }) => {
  test.setTimeout(180_000)
  await registerVerifyLogin(page, request)
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('timeline-item')).toBeVisible({ timeout: 120_000 })
  await page.getByTestId(/trash-item-/).click()
  await expect(page.getByTestId('timeline-empty')).toBeVisible({ timeout: 30_000 })

  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('upload-duplicate')).toContainText('корзине', { timeout: 120_000 })
  await expect(page.getByTestId('upload-restore-trash')).toBeVisible()
  await page.getByTestId('upload-restore-trash').click()
  await expect(page.getByTestId('timeline-item')).toBeVisible({ timeout: 30_000 })
})

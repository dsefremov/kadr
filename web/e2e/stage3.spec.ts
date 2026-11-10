import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { registerVerifyLogin } from './helpers'

test('public visibility confirmation modal', async ({ page, request }) => {
  test.setTimeout(90_000)
  await registerVerifyLogin(page, request)
  await expect(page.getByTestId('visibility-toggle')).toHaveText('личная')
  await page.getByTestId('visibility-toggle').click()
  await expect(page.getByTestId('public-feed-confirm')).toBeVisible()
  await expect(page.getByText('Ваша лента станет видна всем по ссылке')).toBeVisible()
  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations).toEqual([])
  await page.getByTestId('public-feed-confirm-cancel').click()
  await expect(page.getByTestId('public-feed-confirm')).not.toBeVisible()
  await expect(page.getByTestId('visibility-toggle')).toHaveText('личная')
  await page.getByTestId('visibility-toggle').click()
  await page.keyboard.press('Escape')
  await expect(page.getByTestId('public-feed-confirm')).not.toBeVisible()
  await page.getByTestId('visibility-toggle').click()
  await page.getByTestId('public-feed-confirm-confirm').click()
  await expect(page.getByTestId('visibility-toggle')).toHaveText('публичная', { timeout: 10_000 })
})

test('tus upload resumes after interrupted chunk', async ({ page, request }) => {
  test.setTimeout(120_000)
  await registerVerifyLogin(page, request)
  let failOnce = true
  await page.route('**/api/uploads/**', async (route) => {
    const req = route.request()
    if (failOnce && req.method() === 'PATCH') {
      failOnce = false
      await route.abort('failed')
      return
    }
    await route.continue()
  })
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  const payload = Buffer.concat([readFileSync(fixture), readFileSync(fixture), readFileSync(fixture)])
  const tmp = test.info().outputPath('resume-large.png')
  await import('node:fs/promises').then((fs) => fs.writeFile(tmp, payload))
  await page.getByTestId('upload-input').setInputFiles(tmp)
  await expect(page.getByTestId('upload-message')).toContainText('принята', { timeout: 60_000 })
  await expect(page.getByTestId('upload-progress')).toBeVisible()
})

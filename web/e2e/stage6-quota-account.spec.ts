import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { join } from 'node:path'
import { registerVerifyLogin, screenshotDir } from './helpers'

test('quota warning banner when usage is high', async ({ page, request }) => {
  await registerVerifyLogin(page, request)
  await page.route('**/api/me', async (route) => {
    const res = await route.fetch()
    const json = await res.json()
    await route.fulfill({
      response: res,
      json: {
        ...json,
        quota_limit: 5_000_000_000,
        quota_used: 4_600_000_000,
        quota_trash: 200_000_000,
        quota_reserved: 0,
        quota_free: 200_000_000,
        quota_status: 'warning',
        email_verified: true,
      },
    })
  })
  await page.reload()
  await expect(page.getByTestId('quota-warning')).toBeVisible()
  const a11y = await new AxeBuilder({ page }).analyze()
  expect(a11y.violations).toEqual([])
})

test('quota exceeded banner on upload', async ({ page, request }) => {
  await registerVerifyLogin(page, request)
  await page.route('**/api/uploads', (route) => {
    if (route.request().method() === 'POST') {
      return route.fulfill({
        status: 403,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'quota_exceeded', quota_free: 100_000, need_bytes: 500_000 }),
      })
    }
    return route.continue()
  })
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('quota-exceeded')).toBeVisible({ timeout: 30_000 })
})

for (const { slug, width, height } of [
  { slug: 'desktop-1440', width: 1440, height: 900 },
  { slug: 'mobile-390', width: 390, height: 844 },
]) {
  test(`stage 6 quota screenshots ${slug}`, async ({ page, request }) => {
    const out = screenshotDir('stage-6')
    await page.setViewportSize({ width, height })
    await registerVerifyLogin(page, request)
    await page.route('**/api/me', async (route) => {
      const res = await route.fetch()
      const json = await res.json()
      await route.fulfill({
        response: res,
        json: {
          ...json,
          quota_limit: 5_000_000_000,
          quota_used: 4_600_000_000,
          quota_trash: 200_000_000,
          quota_reserved: 0,
          quota_free: 200_000_000,
          quota_status: 'warning',
          email_verified: true,
        },
      })
    })
    await page.reload()
    await expect(page.getByTestId('quota-warning')).toBeVisible()
    await page.screenshot({ path: `${out}/quota-warning-${slug}.png`, fullPage: true })
    const a11y = await new AxeBuilder({ page }).analyze()
    expect(a11y.violations).toEqual([])
  })
}

test('account delete flow', async ({ page, request }) => {
  const { password } = await registerVerifyLogin(page, request)
  await page.goto('/account')
  await page.getByTestId('account-delete-open').click()
  await page.getByTestId('account-delete-password').fill(password)
  await page.getByTestId('account-delete-confirm-confirm').click()
  await expect(page.getByTestId('account-delete-progress')).toBeVisible()
})

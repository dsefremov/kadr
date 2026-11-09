import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { join } from 'node:path'
import { latestToken, registerVerifyLogin, screenshotDir } from './helpers'

test('forgot and invalid-link screens', async ({ page }, testInfo) => {
  const slug = testInfo.project.name
  const dir = screenshotDir()

  await page.goto('/forgot')
  await expect(page.getByTestId('forgot-submit')).toBeVisible()
  await page.screenshot({ path: `${dir}/forgot-${slug}.png`, fullPage: true })

  await page.goto('/verify?token=stage2-screenshot-invalid')
  await expect(page.getByTestId('invalid-link')).toBeVisible()
  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations).toEqual([])
  await page.screenshot({ path: `${dir}/invalid-link-${slug}.png`, fullPage: true })
})

test('reset password form', async ({ page, request }, testInfo) => {
  test.setTimeout(120_000)
  const slug = testInfo.project.name
  const dir = screenshotDir()
  const { email } = await registerVerifyLogin(page, request)
  await page.goto('/forgot')
  await page.getByTestId('forgot-email').fill(email)
  await page.getByTestId('forgot-submit').click()
  let token: string | null = null
  for (let i = 0; i < 30; i++) {
    token = await latestToken(request, 'сброс', email)
    if (token) break
    await page.waitForTimeout(1000)
  }
  expect(token).toBeTruthy()
  await page.goto(`/reset?token=${token}`)
  await expect(page.getByTestId('reset-password')).toBeVisible()
  await page.screenshot({ path: `${dir}/reset-${slug}.png`, fullPage: true })
})

test('home upload zone and upload accepted', async ({ page, request }, testInfo) => {
  test.setTimeout(120_000)
  const slug = testInfo.project.name
  const dir = screenshotDir()
  await registerVerifyLogin(page, request)
  await expect(page.getByTestId('upload-zone')).toBeVisible()
  await expect(page.getByTestId('visibility-toggle')).toBeVisible()
  await page.screenshot({ path: `${dir}/home-${slug}.png`, fullPage: true })

  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('upload-message')).toContainText('принята', { timeout: 30_000 })
  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations).toEqual([])
  await page.screenshot({ path: `${dir}/upload-done-${slug}.png`, fullPage: true })
})

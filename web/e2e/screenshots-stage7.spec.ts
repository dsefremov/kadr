import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { join } from 'node:path'
import { registerVerifyLogin, screenshotDir } from './helpers'

for (const { slug, width, height } of [
  { slug: 'desktop-1440', width: 1440, height: 900 },
  { slug: 'mobile-390', width: 390, height: 844 },
]) {
  test(`stage 7 screenshots ${slug}`, async ({ page, request }) => {
    test.setTimeout(180_000)
    const out = screenshotDir('stage-7')
    await page.setViewportSize({ width, height })
    await registerVerifyLogin(page, request)
    const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
    await page.getByTestId('upload-input').setInputFiles(fixture)
    await expect(page.getByTestId('timeline-item')).toBeVisible({ timeout: 120_000 })
    await page.screenshot({ path: `${out}/feed-grouped-${slug}.png`, fullPage: true })
    await page.goto('/uploads')
    await expect(page.getByTestId('upload-queue')).toBeVisible()
    await page.screenshot({ path: `${out}/upload-queue-${slug}.png`, fullPage: true })
    await page.goto('/')
    await expect(page.getByTestId('timeline-item')).toBeVisible()
    await page.getByTestId(/open-viewer-/).first().click()
    await expect(page.getByTestId('viewer')).toBeVisible()
    await page.screenshot({ path: `${out}/viewer-photo-${slug}.png`, fullPage: true })
    const a11y = await new AxeBuilder({ page }).analyze()
    expect(a11y.violations).toEqual([])
  })
}

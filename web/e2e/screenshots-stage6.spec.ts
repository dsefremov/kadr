import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { join } from 'node:path'
import { registerVerifyLogin, screenshotDir, waitForTimelineReady } from './helpers'

for (const { slug, width, height } of [
  { slug: 'desktop-1440', width: 1440, height: 900 },
  { slug: 'mobile-390', width: 390, height: 844 },
]) {
  test(`stage 6 screenshots ${slug}`, async ({ page, request }) => {
    test.setTimeout(180_000)
    const out = screenshotDir('stage-6')
    await page.setViewportSize({ width, height })
    await registerVerifyLogin(page, request)
    const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
    await page.getByTestId('upload-input').setInputFiles(fixture)
    await expect(page.getByTestId('upload-message')).toContainText('принята', { timeout: 90_000 })
    await waitForTimelineReady(page, 1)
    await page.getByTestId(/trash-item-/).click()
    await page.getByTestId('open-trash').click()
    await expect(page.getByTestId('trash-item')).toBeVisible({ timeout: 30_000 })
    await page.screenshot({ path: `${out}/trash-${slug}.png`, fullPage: true })
    const a11y = await new AxeBuilder({ page }).analyze()
    expect(a11y.violations).toEqual([])
  })
}

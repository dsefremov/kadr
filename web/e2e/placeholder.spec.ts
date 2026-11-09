import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'

test('placeholder page renders', async ({ page }, testInfo) => {
  await page.goto('/')
  await expect(page.getByTestId('kadr-app')).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Кадр' })).toBeVisible()

  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations, JSON.stringify(accessibility.violations, null, 2)).toEqual([])

  const slug = testInfo.project.name
  await page.screenshot({
    path: `../docs/stage-1/screenshots/placeholder-${slug}.png`,
    fullPage: true,
  })
})

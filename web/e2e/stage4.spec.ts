import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { registerVerifyLogin, waitForTimelineReady } from './helpers'

test('upload processes and shows in timeline', async ({ page, request }) => {
  test.setTimeout(180_000)
  await registerVerifyLogin(page, request)
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  await page.getByTestId('upload-input').setInputFiles(fixture)
  await expect(page.getByTestId('upload-message')).toContainText('принята', { timeout: 90_000 })
  await waitForTimelineReady(page, 1)
  await expect(page.getByTestId('timeline-count')).toHaveText('Готово: 1')
  const a11y = await new AxeBuilder({ page }).analyze()
  expect(a11y.violations).toEqual([])
})

test('animated GIF is rejected with visible error', async ({ page, request }) => {
  test.setTimeout(120_000)
  await registerVerifyLogin(page, request)
  const gif = Buffer.from('GIF89a\x00\x00\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00!\xff\x0bNETSCAPE2.0\x03\x01\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;')
  const tmp = test.info().outputPath('anim.gif')
  await import('node:fs/promises').then((fs) => fs.writeFile(tmp, gif))
  await page.getByTestId('upload-input').setInputFiles(tmp)
  await expect(page.getByTestId('upload-message')).toContainText('принята', { timeout: 90_000 })
  await expect
    .poll(
      async () => {
        const res = await page.request.get('/api/me/media?status=failed', { credentials: 'include' })
        const data = (await res.json()) as { media?: { error?: string }[] }
        return (data.media ?? []).filter((m) => m.error).length
      },
      { timeout: 120_000 },
    )
    .toBeGreaterThan(0)
})

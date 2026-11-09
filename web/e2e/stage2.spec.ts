import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { latestToken, registerVerifyLogin } from './helpers'

test('password reset flow', async ({ page, request }) => {
  test.setTimeout(120_000)
  const { email } = await registerVerifyLogin(page, request)
  await page.goto('/forgot')
  await page.getByTestId('forgot-email').fill(email)
  await page.getByTestId('forgot-submit').click()
  await expect(page.getByTestId('forgot-sent')).toBeVisible()

  let token: string | null = null
  for (let i = 0; i < 30; i++) {
    token = await latestToken(request, 'сброс', email)
    if (token) break
    await page.waitForTimeout(1000)
  }
  expect(token).toBeTruthy()
  const newPassword = 'newlongpassword9'
  await page.goto(`/reset?token=${token}`)
  await page.getByTestId('reset-password').fill(newPassword)
  await page.getByTestId('reset-submit').click()
  await expect(page.getByRole('heading', { name: 'Пароль обновлён' })).toBeVisible()

  await page.goto('/login')
  await page.getByTestId('login-email').fill(email)
  await page.getByTestId('login-password').fill(newPassword)
  await page.getByTestId('login-submit').click()
  await expect(page.getByTestId('home-status')).toContainText(email)

  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations).toEqual([])
})

test('tus photo upload', async ({ page, request }) => {
  test.setTimeout(120_000)
  await registerVerifyLogin(page, request)
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  const fileBuffer = readFileSync(fixture)
  const meta = `filetype ${Buffer.from('photo').toString('base64')}`
  const create = await page.request.post('/api/uploads', {
    headers: {
      'Tus-Resumable': '1.0.0',
      'Upload-Length': String(fileBuffer.length),
      'Upload-Metadata': meta,
    },
  })
  expect(create.status(), await create.text()).toBe(201)
  const loc = create.headers()['location']
  expect(loc).toBeTruthy()
  const url = loc!.startsWith('http') ? loc! : new URL(loc!, 'http://127.0.0.1').pathname
  const patch = await page.request.patch(url, {
    headers: {
      'Tus-Resumable': '1.0.0',
      'Upload-Offset': '0',
      'Content-Type': 'application/offset+octet-stream',
    },
    data: fileBuffer,
  })
  expect(patch.status()).toBe(204)
  await page.goto('/')
  await expect(page.getByTestId('upload-zone')).toBeVisible({ timeout: 30_000 })

  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations).toEqual([])
})

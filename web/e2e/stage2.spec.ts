import AxeBuilder from '@axe-core/playwright'
import { APIRequestContext, expect, test } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const mailpit = process.env.MAILPIT_URL ?? 'http://127.0.0.1:8025'

async function latestToken(
  request: APIRequestContext,
  subjectPart: string,
  toEmail?: string,
): Promise<string | null> {
  const resp = await request.get(`${mailpit}/api/v1/messages`)
  const data = await resp.json()
  const msgs = (data.messages as { ID: string; Subject: string; To?: { Address: string }[] }[]).filter(
    (m) => m.Subject.includes(subjectPart) && (!toEmail || m.To?.some((t) => t.Address === toEmail)),
  )
  const msg = msgs[0]
  if (!msg) return null
  const body = await request.get(`${mailpit}/api/v1/message/${msg.ID}`)
  const full = await body.json()
  const text: string = full.Text ?? ''
  const match = text.match(/token=([A-Za-z0-9_-]+)/)
  return match ? match[1] : null
}

async function registerVerifyLogin(page: import('@playwright/test').Page, request: APIRequestContext) {
  const email = `s2-${Date.now()}@example.test`
  const password = 'longpassword1'
  await page.goto('/register')
  await page.getByTestId('register-email').fill(email)
  await page.getByTestId('register-password').fill(password)
  await page.getByTestId('register-submit').click()
  let token: string | null = null
  for (let i = 0; i < 30; i++) {
    token = await latestToken(request, 'подтвердите')
    if (token) break
    await page.waitForTimeout(1000)
  }
  expect(token).toBeTruthy()
  await page.goto(`/verify?token=${token}`)
  await expect(page.getByRole('heading', { name: 'Почта подтверждена' })).toBeVisible()
  await page.goto('/login')
  await page.getByTestId('login-email').fill(email)
  await page.getByTestId('login-password').fill(password)
  await page.getByTestId('login-submit').click()
  await expect(page.getByTestId('home-status')).toContainText(email)
  await expect.poll(async () => {
    return page.evaluate(async () => {
      const r = await fetch('/api/me', { credentials: 'include' })
      if (!r.ok) return false
      const j = await r.json()
      return j.email_verified === true
    })
  }).toBe(true)
  return { email, password }
}

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

test('tus photo upload', async ({ page, request }, testInfo) => {
  test.setTimeout(120_000)
  await registerVerifyLogin(page, request)
  await expect(page.getByTestId('upload-zone')).toBeVisible()
  const fixture = join(process.cwd(), '..', 'fixtures', 'in', 'sample.png')
  const fileBuffer = readFileSync(fixture)
  await page.getByTestId('upload-picker').click()
  await page.getByTestId('upload-input').setInputFiles({
    name: 'sample.png',
    mimeType: 'image/png',
    buffer: fileBuffer,
  })
  await expect(page.getByTestId('upload-message')).toContainText('принята', { timeout: 15000 })

  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations).toEqual([])

  const slug = testInfo.project.name
  await page.screenshot({
    path: `../docs/stage-2/screenshots/home-${slug}.png`,
    fullPage: true,
  })
})

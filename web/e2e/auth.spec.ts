import AxeBuilder from '@axe-core/playwright'
import { APIRequestContext, expect, test } from '@playwright/test'

const mailpit = process.env.MAILPIT_URL ?? 'http://127.0.0.1:8025'

async function latestVerifyToken(request: APIRequestContext): Promise<string | null> {
  const resp = await request.get(`${mailpit}/api/v1/messages`)
  const data = await resp.json()
  const msgs = data.messages as { ID: string; Subject: string }[]
  const msg = msgs.find((m) => m.Subject.includes('подтвердите'))
  if (!msg) return null
  const body = await request.get(`${mailpit}/api/v1/message/${msg.ID}`)
  const full = await body.json()
  const text: string = full.Text ?? ''
  const match = text.match(/token=([A-Za-z0-9_-]+)/)
  return match ? match[1] : null
}

test('registration verify and login flow', async ({ page, request }, testInfo) => {
  test.setTimeout(90_000)
  const email = `user-${Date.now()}@example.test`
  const password = 'longpassword1'

  await page.goto('/register')
  await page.getByTestId('register-email').fill(email)
  await page.getByTestId('register-password').fill(password)
  await page.getByTestId('register-submit').click()
  await expect(page.getByTestId('email-sent-message')).toBeVisible()

  let token: string | null = null
  for (let i = 0; i < 30; i++) {
    token = await latestVerifyToken(request)
    if (token) break
    await page.waitForTimeout(1000)
  }
  expect(token, 'verify email in Mailpit').toBeTruthy()

  const verifyRes = await request.post('/api/auth/verify', { data: { token } })
  expect(verifyRes.ok()).toBeTruthy()

  await page.goto('/login')
  await page.getByTestId('login-email').fill(email)
  await page.getByTestId('login-password').fill(password)
  await page.getByTestId('login-submit').click()
  await expect(page.getByTestId('home-status')).toContainText(email)

  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations).toEqual([])

  const slug = testInfo.project.name
  await page.screenshot({
    path: `../docs/stage-2/screenshots/auth-${slug}.png`,
    fullPage: true,
  })
})

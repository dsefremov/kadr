import { APIRequestContext, expect, Page } from '@playwright/test'

const mailpit = process.env.MAILPIT_URL ?? 'http://127.0.0.1:8025'

export async function latestToken(
  request: APIRequestContext,
  subjectPart: string,
  toEmail?: string,
): Promise<string | null> {
  const resp = await request.get(`${mailpit}/api/v1/messages`)
  const data = await resp.json()
  const want = toEmail?.toLowerCase()
  const msgs = (data.messages as { ID: string; Subject: string; To?: { Address: string }[] }[]).filter(
    (m) =>
      m.Subject.includes(subjectPart) &&
      (!want || m.To?.some((t) => t.Address.toLowerCase() === want)),
  )
  const msg = msgs[0]
  if (!msg) return null
  const body = await request.get(`${mailpit}/api/v1/message/${msg.ID}`)
  const full = await body.json()
  const text: string = full.Text ?? ''
  const match = text.match(/token=([A-Za-z0-9_-]+)/)
  return match ? match[1] : null
}

export async function registerVerifyLogin(page: Page, request: APIRequestContext) {
  const email = `s2-${Date.now()}@example.test`
  const password = 'longpassword1'
  await page.goto('/register')
  await page.getByTestId('register-email').fill(email)
  await page.getByTestId('register-password').fill(password)
  await page.getByTestId('register-submit').click()
  let token: string | null = null
  for (let i = 0; i < 30; i++) {
    token = await latestToken(request, 'подтвердите', email)
    if (token) break
    await page.waitForTimeout(1000)
  }
  expect(token).toBeTruthy()
  const verify = await page.request.post('/api/auth/verify', { data: { token } })
  expect(verify.ok()).toBeTruthy()
  const login = await page.request.post('/api/auth/login', { data: { email, password } })
  expect(login.ok()).toBeTruthy()
  await page.goto('/')
  await expect(page.getByTestId('home-status')).toContainText(email)
  return { email, password }
}

export function screenshotDir(): string {
  return '../docs/stage-2/screenshots'
}

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
  await page.goto('/login')
  await page.getByTestId('login-email').fill(email)
  await page.getByTestId('login-password').fill(password)
  await page.getByTestId('login-submit').click()
  await expect(page.getByTestId('home-status')).toContainText(email, { timeout: 30_000 })
  await expect
    .poll(async () =>
      page.evaluate(async () => {
        const me = await fetch('/api/me', { credentials: 'include' }).then((r) => r.json())
        return me.feed_id as string | undefined
      }),
    )
    .toBeTruthy()
  await expect(page.getByTestId('visibility-toggle')).toBeVisible({ timeout: 30_000 })
  return { email, password }
}

/** Poll timeline API, then reload so React state matches (dev/local e2e). */
export async function openTrash(page: Page) {
  await page.goto('/trash')
}

export async function trashFirstTimelineItem(page: Page) {
  const btn = page.locator('[data-testid^=trash-item-]').first()
  await btn.scrollIntoViewIfNeeded()
  await btn.evaluate((el) => {
    ;(el as HTMLButtonElement).click()
  })
  await expect
    .poll(
      async () =>
        page.evaluate(async () => {
          const me = await fetch('/api/me', { credentials: 'include' }).then((r) => r.json())
          if (!me.feed_id) return -1
          const t = await fetch(`/api/feeds/${me.feed_id}/timeline?limit=5`, { credentials: 'include' }).then((r) =>
            r.json(),
          )
          return Array.isArray(t.items) ? t.items.length : 0
        }),
      { timeout: 30_000 },
    )
    .toBe(0)
  await page.reload()
}

export async function waitForTimelineReady(page: Page, minItems = 1) {
  await expect
    .poll(
      async () =>
        page.evaluate(async () => {
          const me = await fetch('/api/me', { credentials: 'include' }).then((r) => r.json())
          if (!me.feed_id) return 0
          const t = await fetch(`/api/feeds/${me.feed_id}/timeline?limit=10`, {
            credentials: 'include',
          }).then((r) => r.json())
          return Array.isArray(t.items) ? t.items.length : 0
        }),
      { timeout: 120_000 },
    )
    .toBeGreaterThanOrEqual(minItems)
  await page.reload()
  await expect(page.getByTestId('timeline-item')).toHaveCount(minItems, { timeout: 30_000 })
}

export function screenshotDir(
  stage: 'stage-2' | 'stage-3' | 'stage-4' | 'stage-5' | 'stage-6' | 'stage-7' = 'stage-2',
): string {
  return `../docs/${stage}/screenshots`
}

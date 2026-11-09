import { describe, expect, it, vi } from 'vitest'
import { tusUpload } from './tusUpload'

describe('tusUpload', () => {
  it('creates and patches', async () => {
    const file = new File(['hello'], 'a.png', { type: 'image/png' })
    vi.stubGlobal(
      'fetch',
      vi.fn()
        .mockResolvedValueOnce({
          ok: true,
          headers: { get: () => '/api/uploads/abc123' },
        })
        .mockResolvedValueOnce({ ok: true, headers: { get: () => null } }),
    )
    const id = await tusUpload(file, 'photo')
    expect(id).toBe('abc123')
    vi.unstubAllGlobals()
  })
})

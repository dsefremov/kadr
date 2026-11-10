import { beforeEach, describe, expect, it, vi } from 'vitest'

const start = vi.fn()
const abort = vi.fn()

vi.mock('tus-js-client', () => ({
  Upload: vi.fn(function UploadMock(this: { url?: string; start: typeof start; abort: typeof abort }, _file: File, opts: { onSuccess?: () => void }) {
    this.url = '/api/uploads/testid'
    this.start = start
    this.abort = abort
    start.mockImplementation(() => {
      queueMicrotask(() => opts.onSuccess?.())
    })
  }),
}))

import { tusUpload } from './tusUpload'

describe('tusUpload', () => {
  beforeEach(() => {
    start.mockClear()
  })

  it('uses tus-js-client and returns upload id', async () => {
    const file = new File(['hello'], 'a.png', { type: 'image/png' })
    const id = await tusUpload(file, 'photo')
    expect(start).toHaveBeenCalled()
    expect(id).toBe('testid')
  })
})

import * as tus from 'tus-js-client'
import type { DetailedError } from 'tus-js-client'

export type DuplicateUploadBody = {
  error: string
  media_id: string
  media_state?: string
  preview_url: string
  timeline_url?: string
  restore_url?: string
}

export class DuplicateUploadError extends Error {
  body: DuplicateUploadBody
  constructor(body: DuplicateUploadBody) {
    super('duplicate upload')
    this.body = body
  }
}

export type TusUploadHandlers = {
  onProgress?: (bytesUploaded: number, bytesTotal: number) => void
  onError?: (err: Error) => void
}

function duplicateResponseText(err: Error): string {
  const res = (err as DetailedError).originalResponse
  const body = res?.getBody?.() ?? ''
  if (body) return body
  const m = err.message.match(/response text: (\{.*\}), request id/)
  return m?.[1] ?? ''
}

function tryParseDuplicate(err: Error): DuplicateUploadError | null {
  const status = (err as DetailedError).originalResponse?.getStatus() ?? 0
  const text = duplicateResponseText(err)
  if (status !== 409 && !text.includes('"duplicate')) return null
  try {
    const body = JSON.parse(text) as DuplicateUploadBody
    if ((body?.error === 'duplicate' || body?.error === 'duplicate_in_trash') && body.media_id) {
      return new DuplicateUploadError(body)
    }
  } catch {
    return null
  }
  return null
}

export type TusUploadControl = {
  upload: tus.Upload
  promise: Promise<string>
  abort: (terminate?: boolean) => void
}

const chunkSize = Number(import.meta.env.VITE_TUS_CHUNK_SIZE) || 256 * 1024

function tusEndpoint(): string {
  const base = import.meta.env.VITE_TUS_ENDPOINT ?? '/api/uploads'
  return base.startsWith('http') ? base : `${window.location.origin}${base}`
}

function kindMetadata(kind: 'photo' | 'video'): Record<string, string> {
  return { filetype: kind }
}

export function createTusUpload(
  file: File,
  kind: 'photo' | 'video',
  handlers: TusUploadHandlers = {},
  options?: { uploadUrl?: string },
): TusUploadControl {
  let resolve!: (id: string) => void
  let reject!: (err: Error) => void
  const promise = new Promise<string>((res, rej) => {
    resolve = res
    reject = rej
  })

  const upload = new tus.Upload(file, {
    endpoint: tusEndpoint(),
    uploadUrl: options?.uploadUrl,
    chunkSize,
    retryDelays: [0, 1000, 3000, 5000],
    metadata: kindMetadata(kind),
    headers: {},
    storeFingerprintForResuming: true,
    removeFingerprintOnSuccess: true,
    onShouldRetry: (err) => tryParseDuplicate(err) === null,
    onProgress: (uploaded, total) => handlers.onProgress?.(uploaded, total),
    onSuccess: () => {
      const url = upload.url ?? ''
      const id = url.split('/').filter(Boolean).pop() ?? ''
      if (!id) {
        reject(new Error('no upload id'))
        return
      }
      resolve(id)
    },
    onError: (err) => {
      const dup = tryParseDuplicate(err)
      if (dup) {
        handlers.onError?.(dup)
        reject(dup)
        return
      }
      handlers.onError?.(err)
      reject(err)
    },
  })

  if (import.meta.env.DEV || import.meta.env.VITE_E2E_HOOKS === '1') {
    ;(window as unknown as { __kadrActiveTusUpload?: tus.Upload }).__kadrActiveTusUpload = upload
  }

  void upload.start()

  return {
    upload,
    promise,
    abort: (terminate = false) => upload.abort(terminate),
  }
}

/** Resumes or starts an upload; returns the public upload id when complete. */
export async function tusUpload(
  file: File,
  kind: 'photo' | 'video',
  handlers?: TusUploadHandlers,
  uploadUrl?: string,
): Promise<string> {
  const full = uploadUrl
    ? uploadUrl.startsWith('http')
      ? uploadUrl
      : `${window.location.origin}${uploadUrl}`
    : undefined
  const { promise } = createTusUpload(file, kind, handlers, full ? { uploadUrl: full } : undefined)
  return promise
}

/** Resume a server-side partial upload (owner must supply the same file). */
export async function tusResumeFromUrl(
  file: File,
  kind: 'photo' | 'video',
  uploadUrl: string,
  handlers?: TusUploadHandlers,
): Promise<string> {
  return tusUpload(file, kind, handlers, uploadUrl)
}

export { chunkSize }

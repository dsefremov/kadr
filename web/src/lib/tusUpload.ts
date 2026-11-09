import * as tus from 'tus-js-client'

export type TusUploadHandlers = {
  onProgress?: (bytesUploaded: number, bytesTotal: number) => void
  onError?: (err: Error) => void
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

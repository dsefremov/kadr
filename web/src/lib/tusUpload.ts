/** Minimal tus 1.0 client for single-chunk uploads (stage 2). */
export async function tusUpload(file: File, kind: 'photo' | 'video'): Promise<string> {
  const meta = `filetype ${btoa(kind)}`
  const create = await fetch('/api/uploads', {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Tus-Resumable': '1.0.0',
      'Upload-Length': String(file.size),
      'Upload-Metadata': meta,
    },
  })
  if (!create.ok) {
    throw new Error(`create ${create.status}`)
  }
  const loc = create.headers.get('Location')
  if (!loc) throw new Error('no location')
  const url = loc.startsWith('http') ? loc : new URL(loc, window.location.origin).pathname
  const patch = await fetch(url, {
    method: 'PATCH',
    credentials: 'include',
    headers: {
      'Tus-Resumable': '1.0.0',
      'Upload-Offset': '0',
      'Content-Type': 'application/offset+octet-stream',
    },
    body: file,
  })
  if (!patch.ok) {
    throw new Error(`patch ${patch.status}`)
  }
  return url.split('/').pop() ?? ''
}

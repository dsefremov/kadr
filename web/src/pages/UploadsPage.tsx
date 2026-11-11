import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { QuotaBar } from '../components/QuotaBar'
import { createTusUpload, type TusUploadControl } from '../lib/tusUpload'

type Me = {
  email_verified: boolean
  feed_id?: string
  quota_limit?: number
  quota_used?: number
  quota_trash?: number
  quota_reserved?: number
  quota_status?: string
}

type UploadRow = {
  id: string
  kind: string
  length: number
  offset: number
  state: string
}

type ProcessingRow = { id: string; kind: string; state: string; error?: string }

function formatSize(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)} МБ`
  return `${Math.round(n / 1000)} КБ`
}

function uploadLabel(u: UploadRow): string {
  const pct = u.length ? Math.round((100 * u.offset) / u.length) : 0
  if (u.offset >= u.length && u.length > 0) return 'загружено · обработка'
  if (u.offset > 0) return `${formatSize(u.length)} · загружается ${pct}%`
  return `${formatSize(u.length)} · в очереди`
}

export function UploadsPage() {
  const navigate = useNavigate()
  const [me, setMe] = useState<Me | null>(null)
  const [uploads, setUploads] = useState<UploadRow[]>([])
  const [processing, setProcessing] = useState<ProcessingRow[]>([])
  const [uploadPct, setUploadPct] = useState<number | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const activeRef = useRef<TusUploadControl | null>(null)

  const refresh = useCallback(() => {
    if (!me?.email_verified) return
    fetch('/api/me/uploads', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : { uploads: [] }))
      .then((d) => setUploads((d.uploads ?? []) as UploadRow[]))
      .catch(() => undefined)
    fetch('/api/me/media?status=processing', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : { media: [] }))
      .then((d) => setProcessing((d.media ?? []) as ProcessingRow[]))
      .catch(() => undefined)
  }, [me?.email_verified])

  useEffect(() => {
    fetch('/api/me', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : null))
      .then(setMe)
      .catch(() => undefined)
  }, [])

  useEffect(() => {
    refresh()
    if (!me?.email_verified) return
    const id = window.setInterval(refresh, 2000)
    return () => window.clearInterval(id)
  }, [me?.email_verified, refresh])

  async function cancelUpload(id: string) {
    const res = await fetch(`/api/uploads/${id}`, { method: 'DELETE', credentials: 'include' })
    if (res.ok || res.status === 404) {
      setUploads((prev) => prev.filter((u) => u.id !== id))
    }
    refresh()
  }

  async function onFile(file: File) {
    const kind = file.type.startsWith('video/') ? 'video' : 'photo'
    setUploadPct(0)
    if (activeRef.current) activeRef.current.abort(false)
    const control = createTusUpload(file, kind, {
      onProgress: (done, total) => setUploadPct(total ? Math.round((100 * done) / total) : 0),
    })
    activeRef.current = control
    try {
      await control.promise
      setUploadPct(100)
      refresh()
    } finally {
      activeRef.current = null
      setUploadPct(null)
    }
  }

  if (!me?.email_verified) {
    return (
      <AppShell>
        <p className="note">Войдите и подтвердите почту, чтобы видеть очередь загрузок.</p>
        <Link className="btn pri" to="/login">
          Войти
        </Link>
      </AppShell>
    )
  }

  const quota =
    me.quota_limit != null
      ? {
          limit: me.quota_limit,
          used: me.quota_used ?? 0,
          trash: me.quota_trash ?? 0,
          reserved: me.quota_reserved ?? 0,
          status: me.quota_status,
        }
      : null

  return (
    <AppShell quota={quota} onUpload={() => inputRef.current?.click()}>
      <input
        ref={inputRef}
        type="file"
        accept="image/*,video/*"
        style={{ display: 'none' }}
        data-testid="uploads-file-input"
        onChange={(e) => {
          const f = e.target.files?.[0]
          if (f) void onFile(f)
          e.target.value = ''
        }}
      />
      <div className="feedhead">
        <div>
          <h1>Загрузки</h1>
          <p>Файлы попадают в ленту после обработки.</p>
        </div>
        <button type="button" className="btn sm" onClick={() => navigate('/')}>
          К ленте
        </button>
      </div>
      <div className="grid2" data-testid="upload-queue">
        <div className="panel">
          {uploads.length === 0 && processing.length === 0 ? (
            <p className="note">Очередь пуста.</p>
          ) : null}
          {uploads.map((u) => {
            const pct = u.length ? Math.round((100 * u.offset) / u.length) : 0
            return (
              <div key={u.id} className="qi" data-testid="upload-queue-item">
                <div className="th">{u.kind === 'video' ? 'VID' : 'IMG'}</div>
                <div className="meta">
                  <div className="nm">Загрузка {u.id.slice(0, 8)}</div>
                  <div className="sm">{uploadLabel(u)}</div>
                  <div className="bar">
                    <b style={{ width: `${pct}%` }} />
                  </div>
                </div>
                <div className="row">
                  <button type="button" className="btn sm ghost" onClick={() => void cancelUpload(u.id)}>
                    Отменить
                  </button>
                  {u.offset > 0 && u.offset < u.length ? (
                    <button
                      type="button"
                      className="btn sm"
                      onClick={() => {
                        inputRef.current?.click()
                      }}
                    >
                      Продолжить
                    </button>
                  ) : null}
                </div>
              </div>
            )
          })}
          {processing.map((p) => (
            <div key={p.id} className="qi" data-testid="upload-queue-processing">
              <div className="th">{p.kind === 'video' ? 'VID' : 'IMG'}</div>
              <div className="meta">
                <div className="nm">Медиа {p.id.slice(0, 8)}</div>
                <div className="sm">{p.state === 'failed' ? `ошибка: ${p.error}` : 'обрабатывается…'}</div>
                <div className="bar">
                  <b style={{ width: '40%' }} />
                </div>
              </div>
            </div>
          ))}
        </div>
        <div>
          {quota ? (
            <div className="panel">
              <h3>Место</h3>
              <QuotaBar {...quota} />
              <p className="note" style={{ marginTop: 10 }}>
                Загрузки в очереди заранее резервируют место.
              </p>
            </div>
          ) : null}
          {uploadPct !== null ? (
            <p className="note" data-testid="uploads-active-pct">
              Текущая загрузка: {uploadPct}%
            </p>
          ) : null}
        </div>
      </div>
    </AppShell>
  )
}

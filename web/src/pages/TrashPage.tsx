import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { createTusUpload, type TusUploadControl } from '../lib/tusUpload'

type Me = {
  email_verified: boolean
  quota_limit?: number
  quota_used?: number
  quota_trash?: number
  quota_reserved?: number
  quota_status?: string
}

type TrashRow = {
  id: string
  kind: string
  trashed_at: string
  purge_after: string
  days_left: number
  preview_url: string
}

export function TrashPage() {
  const navigate = useNavigate()
  const [me, setMe] = useState<Me | null>(null)
  const [items, setItems] = useState<TrashRow[]>([])
  const [msg, setMsg] = useState('')
  const [denied, setDenied] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const activeRef = useRef<TusUploadControl | null>(null)

  const load = useCallback(() => {
    fetch('/api/me/trash', { credentials: 'include' })
      .then((r) => {
        if (r.status === 401) {
          setDenied(true)
          return { items: [] }
        }
        setDenied(false)
        return r.json()
      })
      .then((data) => setItems((data.items ?? []) as TrashRow[]))
      .catch(() => setMsg('Не удалось загрузить корзину'))
  }, [])

  useEffect(() => {
    fetch('/api/me', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => setMe(d))
      .catch(() => undefined)
  }, [])

  useEffect(() => {
    load()
  }, [load])

  async function restore(id: string) {
    setMsg('')
    const res = await fetch(`/api/media/${id}/restore`, { method: 'POST', credentials: 'include' })
    if (res.ok) {
      load()
      setMsg('Элемент восстановлен в ленту.')
      // refresh quota after restore
      fetch('/api/me', { credentials: 'include' })
        .then((r) => (r.ok ? r.json() : null))
        .then((d) => {
          if (d) setMe(d)
        })
        .catch(() => undefined)
    } else if (res.status === 403) {
      setMsg('Недостаточно места в квоте для восстановления.')
    } else {
      setMsg('Не удалось восстановить.')
    }
  }

  async function purgeOne(id: string) {
    setMsg('')
    const res = await fetch(`/api/media/${id}/trash`, { method: 'DELETE', credentials: 'include' })
    if (res.ok) {
      load()
      setMsg('Элемент удалён навсегда.')
      fetch('/api/me', { credentials: 'include' })
        .then((r) => (r.ok ? r.json() : null))
        .then((d) => {
          if (d) setMe(d)
        })
        .catch(() => undefined)
    } else {
      setMsg('Не удалось удалить.')
    }
  }

  async function emptyTrash() {
    setMsg('')
    const res = await fetch('/api/me/trash', { method: 'DELETE', credentials: 'include' })
    if (res.ok) {
      load()
      setMsg('Корзина очищена.')
      fetch('/api/me', { credentials: 'include' })
        .then((r) => (r.ok ? r.json() : null))
        .then((d) => {
          if (d) setMe(d)
        })
        .catch(() => undefined)
    } else {
      setMsg('Не удалось очистить корзину.')
    }
  }

  async function onFile(file: File) {
    const kind = file.type.startsWith('video/') ? 'video' : 'photo'
    if (activeRef.current) activeRef.current.abort(false)
    const control = createTusUpload(file, kind, {})
    activeRef.current = control
    try {
      await control.promise
      navigate('/')
    } finally {
      activeRef.current = null
    }
  }

  const quota =
    me?.quota_limit != null
      ? {
          limit: me.quota_limit,
          used: me.quota_used ?? 0,
          trash: me.quota_trash ?? 0,
          reserved: me.quota_reserved ?? 0,
          status: me.quota_status,
        }
      : null

  if (denied || (me && !me.email_verified)) {
    return (
      <AppShell>
        <div className="feedhead">
          <div>
            <h1>Корзина</h1>
          </div>
        </div>
        <p className="note">
          <Link to="/login">Войдите</Link>, чтобы открыть корзину.
        </p>
      </AppShell>
    )
  }

  return (
    <AppShell quota={quota} onUpload={() => inputRef.current?.click()}>
      <input
        ref={inputRef}
        type="file"
        accept="image/*,video/*"
        style={{ display: 'none' }}
        data-testid="trash-file-input"
        onChange={(e) => {
          const f = e.target.files?.[0]
          if (f) void onFile(f)
          e.target.value = ''
        }}
      />
      <div className="feedhead">
        <div>
          <h1>Корзина</h1>
          <p className="note">Элементы хранятся 30 дней и учитываются в квоте.</p>
        </div>
        <Link className="btn" to="/" data-testid="trash-back-home">
          К ленте
        </Link>
      </div>
      {items.length > 0 ? (
        <button type="button" className="btn danger" data-testid="empty-trash" onClick={() => void emptyTrash()}>
          Очистить корзину
        </button>
      ) : null}
      {msg ? (
        <p className="note" data-testid="trash-message">
          {msg}
        </p>
      ) : null}
      {items.length === 0 ? (
        <p className="note" data-testid="trash-empty">
          Корзина пуста.
        </p>
      ) : null}
      <ul className="trash-list" data-testid="trash-list">
        {items.map((it) => (
          <li key={it.id} className="trash-row" data-testid="trash-item">
            <img src={it.preview_url} alt="" className="trash-thumb" loading="lazy" />
            <div>
              <div className="t1">{it.kind === 'video' ? 'Видео' : 'Фото'}</div>
              <p className="note">
                Удалено {new Date(it.trashed_at).toLocaleString('ru-RU')}; осталось ~{it.days_left} дн.
              </p>
              <div className="trash-actions">
                <button
                  type="button"
                  className="btn sm pri"
                  data-testid={`restore-${it.id}`}
                  onClick={() => void restore(it.id)}
                >
                  Восстановить
                </button>
                <button
                  type="button"
                  className="btn sm danger"
                  data-testid={`purge-${it.id}`}
                  onClick={() => void purgeOne(it.id)}
                >
                  Удалить навсегда
                </button>
              </div>
            </div>
          </li>
        ))}
      </ul>
    </AppShell>
  )
}

import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ConfirmModal } from '../components/ConfirmModal'
import { QuotaBar } from '../components/QuotaBar'
import { AppShell } from '../components/AppShell'
import { createTusUpload, type TusUploadControl } from '../lib/tusUpload'

type Me = {
  email: string
  email_verified?: boolean
  handle?: string
  quota_limit?: number
  quota_used?: number
  quota_trash?: number
  quota_reserved?: number
  quota_status?: string
}

export function AccountPage() {
  const navigate = useNavigate()
  const [me, setMe] = useState<Me | null>(null)
  const [guest, setGuest] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [deleting, setDeleting] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const activeRef = useRef<TusUploadControl | null>(null)

  useEffect(() => {
    fetch('/api/me', { credentials: 'include' })
      .then((r) => {
        if (r.status === 401) {
          setGuest(true)
          return null
        }
        return r.ok ? r.json() : null
      })
      .then((d) => setMe(d))
      .catch(() => setGuest(true))
  }, [])

  async function confirmDelete() {
    setErr('')
    const res = await fetch('/api/me/delete-account', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password }),
    })
    if (res.ok) {
      setDeleting(true)
      setDeleteOpen(false)
      return
    }
    const body = (await res.json().catch(() => ({}))) as { error?: string }
    if (body.error === 'invalid_password') {
      setErr('Неверный пароль')
    } else {
      setErr('Не удалось удалить аккаунт')
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

  if (deleting) {
    return (
      <AppShell>
        <div className="panel" data-testid="account-delete-progress">
          <h2>Удаление аккаунта</h2>
          <p>Сессии завершены, данные удаляются в фоне. Эту страницу можно закрыть.</p>
          <Link className="btn" to="/register">
            Зарегистрироваться снова
          </Link>
        </div>
      </AppShell>
    )
  }

  if (guest) {
    return (
      <AppShell>
        <div className="feedhead">
          <div>
            <h1>Аккаунт</h1>
          </div>
        </div>
        <p className="note">
          <Link to="/login">Войдите</Link>, чтобы открыть настройки аккаунта.
        </p>
      </AppShell>
    )
  }

  return (
    <AppShell quota={quota} onUpload={me?.email_verified ? () => inputRef.current?.click() : undefined}>
      <input
        ref={inputRef}
        type="file"
        accept="image/*,video/*"
        style={{ display: 'none' }}
        data-testid="account-file-input"
        onChange={(e) => {
          const f = e.target.files?.[0]
          if (f) void onFile(f)
          e.target.value = ''
        }}
      />
      <div className="feedhead">
        <div>
          <h1>Аккаунт</h1>
          <p className="note" data-testid="account-email">
            {me?.email ?? '…'}
            {me?.handle ? ` (@${me.handle})` : ''}
          </p>
        </div>
        <Link className="btn sm" to="/">
          К ленте
        </Link>
      </div>
      {me ? (
        <div className="panel account-panel" data-testid="account-panel">
          <h3>Место</h3>
          <QuotaBar
            used={me.quota_used ?? 0}
            trash={me.quota_trash ?? 0}
            reserved={me.quota_reserved ?? 0}
            limit={me.quota_limit ?? 0}
            status={me.quota_status}
          />
          <p className="note">Квота по умолчанию — 1 ГБ (1 ГБ = 1000 МБ). Корзина входит в квоту.</p>
        </div>
      ) : null}
      <div className="panel panel-danger" data-testid="account-delete-section">
        <h3>Удаление аккаунта</h3>
        <p className="note">Лента и все файлы будут удалены. Действие необратимо.</p>
        <button type="button" className="btn danger sm" data-testid="account-delete-open" onClick={() => setDeleteOpen(true)}>
          Удалить аккаунт
        </button>
      </div>
      <ConfirmModal
        open={deleteOpen}
        title="Удалить аккаунт навсегда?"
        testId="account-delete-confirm"
        confirmLabel="Удалить аккаунт"
        cancelLabel="Отмена"
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => void confirmDelete()}
      >
        <p>Лента и все медиа будут удалены без восстановления.</p>
        <label className="field">
          <span>Введите пароль, чтобы подтвердить</span>
          <input
            className="inp"
            type="password"
            data-testid="account-delete-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        {err ? <p className="hint bad">{err}</p> : null}
      </ConfirmModal>
    </AppShell>
  )
}

import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ConfirmModal } from '../components/ConfirmModal'
import { QuotaBar } from '../components/QuotaBar'
import { Shell } from '../components/Shell'

type Me = {
  email: string
  handle?: string
  quota_limit?: number
  quota_used?: number
  quota_trash?: number
  quota_reserved?: number
  quota_status?: string
}

export function AccountPage() {
  const [me, setMe] = useState<Me | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    fetch('/api/me', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => setMe(d))
      .catch(() => undefined)
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

  if (deleting) {
    return (
      <Shell>
        <div className="panel" data-testid="account-delete-progress">
          <h2>Удаление аккаунта</h2>
          <p>Сессии завершены, данные удаляются в фоне. Эту страницу можно закрыть.</p>
          <Link className="btn" to="/register">
            Зарегистрироваться снова
          </Link>
        </div>
      </Shell>
    )
  }

  return (
    <Shell>
      <div className="feedhead">
        <div>
          <h1>Аккаунт</h1>
          <p className="note" data-testid="account-email">
            {me?.email ?? '…'}
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
          <p className="note">Квота по умолчанию — 5 ГБ (1 ГБ = 1000 МБ). Корзина входит в квоту.</p>
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
    </Shell>
  )
}

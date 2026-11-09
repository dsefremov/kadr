import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Shell } from '../components/Shell'

type Me = {
  email: string
  email_verified: boolean
}

export function HomePage() {
  const [me, setMe] = useState<Me | null>(null)
  const [guest, setGuest] = useState(false)

  useEffect(() => {
    fetch('/api/me', { credentials: 'include' })
      .then((r) => {
        if (r.status === 401) {
          setGuest(true)
          return null
        }
        return r.json()
      })
      .then((data) => {
        if (data) setMe(data)
      })
      .catch(() => setGuest(true))
  }, [])

  return (
    <Shell>
      {me && !me.email_verified ? (
        <div className="banner warn" role="status" data-testid="unverified-banner">
          <div>
            <div className="t1">Почта не подтверждена</div>
            <p className="note" style={{ margin: 0 }}>
              Загрузка и публикация ленты будут доступны после подтверждения.
            </p>
          </div>
        </div>
      ) : null}
      <div className="feedhead">
        <div>
          <h1>Кадр</h1>
          <p>
            {me
              ? `Вы вошли как ${me.email}`
              : guest
                ? 'Гость — войдите или зарегистрируйтесь'
                : 'Загрузка…'}
          </p>
        </div>
        {me ? (
          <button
            type="button"
            className="btn"
            onClick={() => fetch('/api/auth/logout', { method: 'POST', credentials: 'include' }).then(() => location.reload())}
          >
            Выйти
          </button>
        ) : (
          <Link className="btn pri" to="/login">
            Войти
          </Link>
        )}
      </div>
      <div className="drop">
        <b>Лента появится на следующих этапах</b>
        <p className="note">Этап 1: аккаунт, почта и очередь писем готовы.</p>
      </div>
    </Shell>
  )
}

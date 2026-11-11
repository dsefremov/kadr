import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Shell } from '../components/Shell'

type TrashRow = {
  id: string
  kind: string
  trashed_at: string
  purge_after: string
  days_left: number
  preview_url: string
}

export function TrashPage() {
  const [items, setItems] = useState<TrashRow[]>([])
  const [msg, setMsg] = useState('')
  const [denied, setDenied] = useState(false)

  const load = useCallback(() => {
    fetch('/api/me/trash', { credentials: 'include' })
      .then((r) => {
        if (r.status === 401) {
          setDenied(true)
          return { items: [] }
        }
        return r.json()
      })
      .then((data) => setItems((data.items ?? []) as TrashRow[]))
      .catch(() => setMsg('Не удалось загрузить корзину'))
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
    } else {
      setMsg('Не удалось очистить корзину.')
    }
  }

  return (
    <Shell>
      <div className="feedhead">
        <div>
          <h1>Корзина</h1>
          <p className="note">Элементы хранятся 30 дней и учитываются в квоте.</p>
        </div>
        <Link className="btn" to="/" data-testid="trash-back-home">
          К ленте
        </Link>
      </div>
      {denied ? (
        <p className="note">
          <Link to="/login">Войдите</Link>, чтобы открыть корзину.
        </p>
      ) : null}
      {!denied && items.length > 0 ? (
        <button type="button" className="btn danger" data-testid="empty-trash" onClick={() => void emptyTrash()}>
          Очистить корзину
        </button>
      ) : null}
      {msg ? (
        <p className="note" data-testid="trash-message">
          {msg}
        </p>
      ) : null}
      {!denied && items.length === 0 ? (
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
    </Shell>
  )
}

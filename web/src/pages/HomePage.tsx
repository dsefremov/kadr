import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Shell } from '../components/Shell'
import { tusUpload } from '../lib/tusUpload'

type Me = {
  email: string
  email_verified: boolean
  handle?: string
  feed_id?: string
  feed_visibility?: string
  quota_limit?: number
  quota_reserved?: number
  quota_used?: number
}

export function HomePage() {
  const [me, setMe] = useState<Me | null>(null)
  const [guest, setGuest] = useState(false)
  const [uploadMsg, setUploadMsg] = useState('')
  const [timelineEmpty, setTimelineEmpty] = useState(true)
  const inputRef = useRef<HTMLInputElement>(null)

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
        if (data) {
          setMe(data)
          if (data.feed_id) {
            fetch(`/api/feeds/${data.feed_id}/timeline`, { credentials: 'include' })
              .then((r) => r.json())
              .then((t) => setTimelineEmpty((t.items ?? []).length === 0))
              .catch(() => undefined)
          }
        }
      })
      .catch(() => setGuest(true))
  }, [])

  async function toggleVisibility() {
    if (!me?.feed_visibility) return
    const next = me.feed_visibility === 'public' ? 'private' : 'public'
    await fetch('/api/me/feed', {
      method: 'PATCH',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ visibility: next }),
    })
    setMe({ ...me, feed_visibility: next })
  }

  async function onFilePicked(file: File) {
    setUploadMsg('')
    const kind = file.type.startsWith('video/') ? 'video' : 'photo'
    try {
      const id = await tusUpload(file, kind)
      setUploadMsg(`Загрузка ${id} принята (обработка — этап 4).`)
    } catch {
      setUploadMsg('Не удалось загрузить файл')
    }
  }

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
          <p data-testid="home-status">
            {me
              ? `Вы вошли как ${me.email}${me.handle ? ` (@${me.handle})` : ''}`
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
      {me?.email_verified ? (
        <>
          <div className="drop" data-testid="upload-zone">
            <b>Загрузить фото или видео</b>
            <p className="note">tus, до 50 МБ фото / 2 ГБ видео; резерв квоты сразу.</p>
            <input
              ref={inputRef}
              type="file"
              accept="image/*,video/*"
              style={{ display: 'none' }}
              data-testid="upload-input"
              onChange={(e) => {
                const f = e.target.files?.[0]
                if (f) void onFilePicked(f)
              }}
            />
            <button type="button" className="btn pri" data-testid="upload-picker" onClick={() => inputRef.current?.click()}>
              Выбрать файл
            </button>
            {uploadMsg ? <p className="note" data-testid="upload-message">{uploadMsg}</p> : null}
          </div>
          {me.handle ? (
            <p className="note">
              Лента{' '}
              <button type="button" className="btn sm" data-testid="visibility-toggle" onClick={() => void toggleVisibility()}>
                {me.feed_visibility === 'public' ? 'публичная' : 'личная'}
              </button>
              {timelineEmpty ? ' — пока пусто, ждём обработку медиа.' : ''}
            </p>
          ) : null}
        </>
      ) : (
        <div className="drop">
          <b>Лента и загрузки</b>
          <p className="note">Этап 2: tus и лента; медиа появится после этапа 4.</p>
        </div>
      )}
    </Shell>
  )
}

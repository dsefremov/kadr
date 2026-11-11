import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { dayHeading } from '../lib/timelineFormat'

type MediaMeta = {
  id: string
  kind: string
  width?: number
  height?: number
  duration_ms?: number
  taken_date: string
  urls?: Record<string, string>
  owner?: boolean
}

export function ViewerPage() {
  const { mediaId } = useParams<{ mediaId: string }>()
  const navigate = useNavigate()
  const [meta, setMeta] = useState<MediaMeta | null>(null)
  const [neighbors, setNeighbors] = useState<string[]>([])
  const [activeId, setActiveId] = useState(mediaId ?? '')
  const [zoom, setZoom] = useState(1)
  const [infoOpen, setInfoOpen] = useState(false)
  const [dateOpen, setDateOpen] = useState(false)
  const [dateValue, setDateValue] = useState('')
  const [dateErr, setDateErr] = useState('')

  const loadMeta = useCallback((id: string) => {
    fetch(`/api/media/${id}`, { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => {
        if (d) {
          setMeta(d)
          setDateValue(d.taken_date)
        }
      })
      .catch(() => undefined)
  }, [])

  useEffect(() => {
    if (!mediaId) return
    setActiveId(mediaId)
    loadMeta(mediaId)
    void (async () => {
      const me = await fetch('/api/me', { credentials: 'include' }).then((r) => (r.ok ? r.json() : null))
      if (!me?.feed_id) return
      const items: { id: string }[] = []
      let cursor: string | undefined
      for (let page = 0; page < 5; page++) {
        const q = new URLSearchParams({ limit: '100' })
        if (cursor) q.set('cursor', cursor)
        const t = await fetch(`/api/feeds/${me.feed_id}/timeline?${q}`, { credentials: 'include' }).then((r) =>
          r.json(),
        )
        items.push(...((t.items ?? []) as { id: string }[]))
        cursor = t.cursor
        if (!cursor) break
      }
      setNeighbors(items.map((i) => i.id))
    })()
  }, [mediaId, loadMeta])

  const idx = neighbors.indexOf(activeId)
  const title = useMemo(() => (meta?.taken_date ? dayHeading(meta.taken_date) : 'Просмотр'), [meta?.taken_date])

  function step(delta: number) {
    if (!neighbors.length) return
    const next = Math.min(neighbors.length - 1, Math.max(0, idx + delta))
    const id = neighbors[next]
    setZoom(1)
    setActiveId(id)
    loadMeta(id)
    navigate(`/view/${id}`, { replace: true })
  }

  async function saveDate() {
    if (!meta?.id) return
    setDateErr('')
    const res = await fetch(`/api/media/${meta.id}/taken-date`, {
      method: 'PATCH',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ taken_date: dateValue }),
    })
    if (res.ok) {
      setDateOpen(false)
      loadMeta(meta.id)
      return
    }
    setDateErr('Не удалось сохранить дату')
  }

  if (!meta) {
    return (
      <div className="viewer" data-testid="viewer-loading">
        <p className="note">Загрузка…</p>
      </div>
    )
  }

  const imageSrc = meta.urls?.preview_1280 ?? meta.urls?.master ?? meta.urls?.poster
  const videoSrc = meta.urls?.video

  return (
    <div className={`viewer${infoOpen ? ' info' : ''}`} data-testid="viewer">
      <div className="vtop">
        <button type="button" className="ic" aria-label="Закрыть" data-testid="viewer-close" onClick={() => navigate('/')}>
          ✕
        </button>
        <div>
          <b data-testid="viewer-title">{title}</b>
          <div className="viewer-sub" data-testid="viewer-position">
            {neighbors.length > 0 ? `${idx + 1} из ${neighbors.length}` : meta.id.slice(0, 8)}
          </div>
        </div>
        <span className="sp" />
        {meta.kind !== 'video' ? (
          <>
            <button type="button" className="ic mh" aria-label="Уменьшить" onClick={() => setZoom((z) => Math.max(1, z - 0.25))}>
              －
            </button>
            <button type="button" className="ic mh" aria-label="Увеличить" onClick={() => setZoom((z) => Math.min(3, z + 0.25))}>
              ＋
            </button>
          </>
        ) : null}
        <button
          type="button"
          className={`ic${infoOpen ? ' on' : ''}`}
          aria-label="Сведения"
          data-testid="viewer-info"
          onClick={() => setInfoOpen((v) => !v)}
        >
          ⓘ
        </button>
        {meta.owner ? (
          <button type="button" className="ic" data-testid="viewer-change-date" onClick={() => setDateOpen(true)}>
            📅
          </button>
        ) : null}
      </div>
      <div className="vstage">
        {meta.kind === 'video' && videoSrc ? (
          <video src={videoSrc} controls playsInline className="viewer-video" data-testid="viewer-video" />
        ) : imageSrc ? (
          <img
            src={imageSrc}
            alt=""
            className={zoom > 1 ? 'zoom' : ''}
            style={zoom > 1 ? { transform: `scale(${zoom})` } : undefined}
            data-testid="viewer-photo"
          />
        ) : null}
        <button type="button" className="navarr" style={{ left: 20 }} aria-label="Предыдущее" onClick={() => step(-1)}>
          ‹
        </button>
        <button type="button" className="navarr" style={{ right: 20 }} aria-label="Следующее" onClick={() => step(1)}>
          ›
        </button>
        {zoom > 1 ? <span className="zoomhud">{Math.round(zoom * 100)}%</span> : null}
      </div>
      {infoOpen ? (
        <aside className="side" data-testid="viewer-info-panel">
          <h4>Сведения</h4>
          <dl>
            <dt>Тип</dt>
            <dd>{meta.kind === 'video' ? 'Видео' : 'Фото'}</dd>
            <dt>Дата</dt>
            <dd data-testid="viewer-taken-date">{meta.taken_date}</dd>
            {meta.width && meta.height ? (
              <>
                <dt>Размер</dt>
                <dd>
                  {meta.width}×{meta.height}
                </dd>
              </>
            ) : null}
          </dl>
          {meta.owner ? (
            <button type="button" className="btn sm" onClick={() => setDateOpen(true)}>
              Сменить дату
            </button>
          ) : null}
          <Link className="btn sm ghost" to="/">
            К ленте
          </Link>
        </aside>
      ) : null}
      <div className="vbot">
        <span>← → соседние</span>
      </div>
      {dateOpen ? (
        <div className="scrim" role="dialog" aria-modal="true" data-testid="viewer-date-dialog">
          <div className="dlg">
            <h3>Сменить дату</h3>
            <p className="note">Дата влияет на группировку в ленте.</p>
            <label className="field">
              <span>Дата</span>
              <input
                type="date"
                className="inp"
                value={dateValue}
                onChange={(e) => setDateValue(e.target.value)}
                data-testid="viewer-date-input"
              />
            </label>
            {dateErr ? <p className="hint bad">{dateErr}</p> : null}
            <div className="acts">
              <button type="button" className="btn" onClick={() => setDateOpen(false)}>
                Отмена
              </button>
              <button type="button" className="btn pri" data-testid="viewer-date-save" onClick={() => void saveDate()}>
                Сохранить
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}

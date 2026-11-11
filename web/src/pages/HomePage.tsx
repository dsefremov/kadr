import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ConfirmModal } from '../components/ConfirmModal'
import { QuotaExceededBanner, QuotaWarningBanner } from '../components/QuotaBar'
import { AppShell } from '../components/AppShell'
import { Timeline } from '../components/Timeline'
import type { Bucket } from '../components/TimelineScrub'
import {
  createTusUpload,
  DuplicateUploadError,
  QuotaExceededError,
  tusResumeFromUrl,
  type TusUploadControl,
} from '../lib/tusUpload'

type Me = {
  email: string
  email_verified: boolean
  handle?: string
  feed_id?: string
  feed_visibility?: string
  quota_limit?: number
  quota_reserved?: number
  quota_used?: number
  quota_trash?: number
  quota_free?: number
  quota_status?: string
}

type PendingUpload = {
  id: string
  kind: string
  length: number
  offset: number
  state: string
}

export function HomePage() {
  const navigate = useNavigate()
  const [me, setMe] = useState<Me | null>(null)
  const [guest, setGuest] = useState(false)
  const [uploadMsg, setUploadMsg] = useState('')
  const [uploadDupUrl, setUploadDupUrl] = useState<string | null>(null)
  const [uploadDupTrash, setUploadDupTrash] = useState<string | null>(null)
  const [quotaExceeded, setQuotaExceeded] = useState<{ need?: number; free: number } | null>(null)
  const [uploadPct, setUploadPct] = useState<number | null>(null)
  const [timelineItems, setTimelineItems] = useState<
    {
      id: string
      kind: string
      width?: number
      height?: number
      duration_ms?: number
      taken_date?: string
      urls?: Record<string, string>
    }[]
  >([])
  const [buckets, setBuckets] = useState<Bucket[]>([])
  const [activeMonth, setActiveMonth] = useState<string | undefined>()
  const [mediaProcessing, setMediaProcessing] = useState<
    { id: string; kind: string; state: string; error?: string }[]
  >([])
  const [publicConfirm, setPublicConfirm] = useState(false)
  const [pendingUploads, setPendingUploads] = useState<PendingUpload[]>([])
  const inputRef = useRef<HTMLInputElement>(null)
  const resumeRef = useRef<PendingUpload | null>(null)
  const activeRef = useRef<TusUploadControl | null>(null)

  const loadPending = useCallback(() => {
    if (!me?.email_verified) return
    fetch('/api/me/uploads', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : { uploads: [] }))
      .then((data) => {
        const list = (data.uploads ?? []) as PendingUpload[]
        setPendingUploads(list.filter((u) => u.state === 'uploading' && u.offset < u.length))
      })
      .catch(() => undefined)
  }, [me?.email_verified])

  const applyTimelinePayload = useCallback((payload: { items?: unknown }) => {
    const next = Array.isArray(payload.items) ? (payload.items as typeof timelineItems) : []
    setTimelineItems((prev) => (next.length === 0 && prev.length > 0 ? prev : next))
  }, [])

  const refreshTimeline = useCallback(() => {
    if (!me?.feed_id) return
    fetch(`/api/feeds/${me.feed_id}/timeline?limit=100`, { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : { items: [] }))
      .then((t) => applyTimelinePayload(t))
      .catch(() => undefined)
    fetch(`/api/feeds/${me.feed_id}/timeline/buckets`, { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : { buckets: [] }))
      .then((b) => setBuckets((b.buckets ?? []) as Bucket[]))
      .catch(() => undefined)
    // Only in-flight processing — sticky "failed" rows used to keep an error plaque forever.
    fetch('/api/me/media?status=processing', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : { media: [] }))
      .then((data) => setMediaProcessing((data.media ?? []) as typeof mediaProcessing))
      .catch(() => undefined)
  }, [me?.feed_id, applyTimelinePayload])

  const pollTimelineForFeed = useCallback(
    async (feedId: string) => {
      for (let attempt = 0; attempt < 60; attempt++) {
        const res = await fetch(`/api/feeds/${feedId}/timeline?limit=20`, { credentials: 'include' })
        if (res.ok) {
          const t = (await res.json()) as { items?: unknown }
          if (Array.isArray(t.items) && t.items.length > 0) {
            applyTimelinePayload(t)
            setUploadMsg((msg) =>
              msg.includes('в ленте') ? msg : msg.replace('обработка на сервере…', 'готово в ленте'),
            )
            return
          }
        }
        await new Promise((resolve) => setTimeout(resolve, 2000))
      }
    },
    [applyTimelinePayload],
  )

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
            setMe(data)
          }
        }
      })
      .catch(() => setGuest(true))
  }, [])

  useEffect(() => {
    loadPending()
  }, [loadPending])

  useEffect(() => {
    refreshTimeline()
    if (!me?.email_verified) return
    const id = window.setInterval(refreshTimeline, 3000)
    return () => window.clearInterval(id)
  }, [me?.email_verified, refreshTimeline])

  async function applyVisibility(next: 'public' | 'private') {
    if (!me) return
    const res = await fetch('/api/me/feed', {
      method: 'PATCH',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ visibility: next }),
    })
    if (res.ok) {
      setMe({ ...me, feed_visibility: next })
    }
  }

  function onVisibilityClick() {
    if (!me?.feed_visibility) return
    if (me.feed_visibility === 'public') {
      void applyVisibility('private')
      return
    }
    setPublicConfirm(true)
  }

  async function runUpload(file: File, resume?: PendingUpload) {
    setUploadMsg('')
    setUploadDupUrl(null)
    setUploadDupTrash(null)
    setQuotaExceeded(null)
    setUploadPct(0)
    const kind = (file.type.startsWith('video/') ? 'video' : 'photo') as 'photo' | 'video'
    const kindFromServer = resume?.kind === 'video' ? 'video' : 'photo'
    const useKind = resume ? kindFromServer : kind

    if (activeRef.current) {
      activeRef.current.abort(false)
      activeRef.current = null
    }

    try {
      const handlers = {
        onProgress: (done: number, total: number) =>
          setUploadPct(total ? Math.round((100 * done) / total) : 0),
      }
      const id = resume
        ? await tusResumeFromUrl(file, useKind, `/api/uploads/${resume.id}`, handlers)
        : await (() => {
            const control = createTusUpload(file, useKind, handlers)
            activeRef.current = control
            return control.promise
          })()
      setUploadPct(100)
      setUploadMsg(`Загрузка ${id} принята — обработка на сервере…`)
      activeRef.current = null
      loadPending()
      refreshTimeline()
      void (async () => {
        let feedId = me?.feed_id
        if (!feedId) {
          const meNow = await fetch('/api/me', { credentials: 'include' }).then((r) => (r.ok ? r.json() : null))
          feedId = meNow?.feed_id as string | undefined
        }
        if (feedId) await pollTimelineForFeed(feedId)
      })()
    } catch (e) {
      if (e instanceof QuotaExceededError) {
        setUploadPct(null)
        setQuotaExceeded({ need: e.body.need_bytes, free: e.body.quota_free })
        setUploadMsg('Не хватает места для загрузки.')
        return
      }
      if (e instanceof DuplicateUploadError) {
        setUploadPct(null)
        setUploadDupUrl(e.body.preview_url)
        if (e.body.error === 'duplicate_in_trash') {
          setUploadDupTrash(e.body.media_id)
          setUploadMsg('Этот файл в корзине. Второй экземпляр не создаём — восстановите в ленту.')
        } else {
          setUploadMsg('Этот файл уже есть в вашей ленте.')
        }
        refreshTimeline()
        loadPending()
        return
      }
      setUploadMsg('Не удалось загрузить файл')
      setUploadPct(null)
      loadPending()
    }
  }

  async function onFilePicked(file: File, resume?: PendingUpload) {
    await runUpload(file, resume)
  }

  async function trashMedia(id: string) {
    const res = await fetch(`/api/media/${id}/trash`, { method: 'POST', credentials: 'include' })
    if (res.ok) {
      refreshTimeline()
    }
  }

  async function restoreDupFromTrash() {
    if (!uploadDupTrash) return
    const res = await fetch(`/api/media/${uploadDupTrash}/restore`, { method: 'POST', credentials: 'include' })
    if (res.ok) {
      setUploadMsg('Файл восстановлен в ленту.')
      setUploadDupTrash(null)
      refreshTimeline()
    } else if (res.status === 403) {
      setUploadMsg('Недостаточно места в квоте для восстановления.')
    }
  }

  const quotaProps =
    me?.quota_limit != null
      ? {
          limit: me.quota_limit,
          used: me.quota_used ?? 0,
          trash: me.quota_trash ?? 0,
          reserved: me.quota_reserved ?? 0,
          status: me.quota_status,
        }
      : null

  function onScrubJump(_anchorId: string, month: string) {
    setActiveMonth(month)
    document.getElementById(`month-${month}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  return (
    <AppShell quota={quotaProps} onUpload={() => inputRef.current?.click()}>
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
          <h1>Моя лента</h1>
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
          {mediaProcessing.length > 0 ? (
            <div className="banner" data-testid="processing-banner">
              <div>
                <div className="t1">{mediaProcessing.length} файл(ов) обрабатывается</div>
                <p className="note" style={{ margin: 0 }}>
                  Появятся в ленте через пару минут.
                </p>
              </div>
              <button type="button" className="btn sm" data-testid="open-uploads" onClick={() => navigate('/uploads')}>
                Открыть загрузки
              </button>
            </div>
          ) : null}
          {me.quota_status === 'warning' ? (
            <QuotaWarningBanner
              free={me.quota_free ?? 0}
              limit={me.quota_limit ?? 0}
              trash={me.quota_trash ?? 0}
              onOpenTrash={() => navigate('/trash')}
            />
          ) : null}
          {quotaExceeded ? <QuotaExceededBanner need={quotaExceeded.need} free={quotaExceeded.free} /> : null}
          <input
            ref={inputRef}
            type="file"
            accept="image/*,video/*"
            style={{ display: 'none' }}
            data-testid="upload-input"
            onChange={(e) => {
              const f = e.target.files?.[0]
              if (f) {
                void onFilePicked(f, resumeRef.current ?? undefined)
                resumeRef.current = null
              }
              e.target.value = ''
            }}
          />
          {timelineItems.length === 0 ? (
            <div className="drop" data-testid="upload-zone">
              <b>Загрузить фото или видео</b>
              <p className="note">Оригинал не хранится — только сжатая копия после обработки.</p>
              <p className="note">tus с докачкой; до 50 МБ фото / 200 МБ видео; резерв квоты сразу.</p>
              <button type="button" className="btn pri" data-testid="upload-picker" onClick={() => inputRef.current?.click()}>
                Выбрать файл
              </button>
            </div>
          ) : null}
          {uploadPct !== null ? (
            <div className="upload-progress-wrap" data-testid="upload-progress" aria-live="polite">
              <div className="upload-progress-bar" style={{ width: `${uploadPct}%` }} />
              <span className="note">{uploadPct}%</span>
            </div>
          ) : null}
          {uploadMsg ? (
            <p
              className="note"
              data-testid={
                uploadDupTrash
                  ? 'upload-duplicate-trash'
                  : uploadDupUrl
                    ? 'upload-duplicate'
                    : 'upload-message'
              }
            >
              {uploadMsg}
              {uploadDupUrl ? (
                <>
                  {' '}
                  <a href={uploadDupUrl} data-testid="upload-duplicate-link">
                    Открыть превью
                  </a>
                </>
              ) : null}
              {uploadDupTrash ? (
                <>
                  {' '}
                  <button
                    type="button"
                    className="btn sm pri"
                    data-testid="upload-restore-trash"
                    onClick={() => void restoreDupFromTrash()}
                  >
                    Восстановить
                  </button>
                </>
              ) : null}
            </p>
          ) : null}
          {pendingUploads.length > 0 ? (
            <div className="note" data-testid="pending-uploads">
              <b>Незавершённые загрузки</b>
              <ul className="pending-list">
                {pendingUploads.map((u) => (
                  <li key={u.id}>
                    {u.kind} — {u.offset} / {u.length} байт{' '}
                    <button
                      type="button"
                      className="btn sm"
                      data-testid={`resume-upload-${u.id}`}
                      onClick={() => {
                        resumeRef.current = u
                        inputRef.current?.click()
                      }}
                    >
                      Продолжить
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
          <>
            <p className="note">
              Лента{' '}
              <button type="button" className="btn sm" data-testid="visibility-toggle" onClick={onVisibilityClick}>
                {me.feed_visibility === 'public' ? 'публичная' : 'личная'}
              </button>
            </p>
            <Timeline
              items={timelineItems}
              processing={mediaProcessing}
              buckets={buckets}
              activeMonth={activeMonth}
              onScrubJump={onScrubJump}
              onTrash={(id) => void trashMedia(id)}
            />
          </>
        </>
      ) : (
        <div className="drop">
          <b>Лента и загрузки</b>
          <p className="note">Войдите, чтобы загружать фото и видео в ленту.</p>
        </div>
      )}
      <ConfirmModal
        open={publicConfirm}
        title="Публичная лента"
        testId="public-feed-confirm"
        confirmLabel="Сделать публичной"
        cancelLabel="Отмена"
        onCancel={() => setPublicConfirm(false)}
        onConfirm={() => {
          setPublicConfirm(false)
          void applyVisibility('public')
        }}
      >
        <p>Ваша лента станет видна всем по ссылке. Продолжить?</p>
      </ConfirmModal>
    </AppShell>
  )
}

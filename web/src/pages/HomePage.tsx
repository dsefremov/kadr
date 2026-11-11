import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ConfirmModal } from '../components/ConfirmModal'
import { QuotaBar, QuotaExceededBanner, QuotaWarningBanner } from '../components/QuotaBar'
import { Shell } from '../components/Shell'
import { Timeline } from '../components/Timeline'
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
    { id: string; kind: string; width?: number; height?: number; urls?: Record<string, string> }[]
  >([])
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

  const refreshTimeline = useCallback(() => {
    if (!me?.feed_id) return
    fetch(`/api/feeds/${me.feed_id}/timeline`, { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : { items: [] }))
      .then((t) => setTimelineItems((t.items ?? []) as typeof timelineItems))
      .catch(() => undefined)
    fetch('/api/me/media?status=processing', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : { media: [] }))
      .then((data) => {
        const proc = (data.media ?? []) as typeof mediaProcessing
        setMediaProcessing(proc)
      })
      .catch(() => undefined)
    fetch('/api/me/media?status=failed', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : { media: [] }))
      .then((data) => {
        const failed = (data.media ?? []) as typeof mediaProcessing
        if (failed.length) {
          setMediaProcessing((prev) => [...prev.filter((p) => p.state !== 'failed'), ...failed])
        }
      })
      .catch(() => undefined)
  }, [me?.feed_id])

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
          {me.quota_limit ? (
            <QuotaBar
              used={me.quota_used ?? 0}
              trash={me.quota_trash ?? 0}
              reserved={me.quota_reserved ?? 0}
              limit={me.quota_limit}
              status={me.quota_status}
            />
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
          <div className="drop" data-testid="upload-zone">
            <b>Загрузить фото или видео</b>
            <p className="note">Оригинал не хранится — только сжатая копия после обработки.</p>
            <p className="note">tus с докачкой; до 50 МБ фото / 2 ГБ видео; резерв квоты сразу.</p>
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
            <button type="button" className="btn pri" data-testid="upload-picker" onClick={() => inputRef.current?.click()}>
              Выбрать файл
            </button>
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
          </div>
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
          {me.handle ? (
            <>
              <p className="note">
                Лента{' '}
                <button type="button" className="btn sm" data-testid="visibility-toggle" onClick={onVisibilityClick}>
                  {me.feed_visibility === 'public' ? 'публичная' : 'личная'}
                </button>
                {' · '}
                <button type="button" className="btn sm" data-testid="open-trash" onClick={() => navigate('/trash')}>
                  Корзина
                </button>
                {' · '}
                <button type="button" className="btn sm" data-testid="open-account" onClick={() => navigate('/account')}>
                  Аккаунт
                </button>
              </p>
              <Timeline items={timelineItems} processing={mediaProcessing} onTrash={(id) => void trashMedia(id)} />
            </>
          ) : null}
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
    </Shell>
  )
}

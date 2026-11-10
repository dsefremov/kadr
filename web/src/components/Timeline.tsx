type TimelineItem = {
  id: string
  kind: string
  width?: number
  height?: number
  urls?: {
    preview_320?: string
    preview_640?: string
    preview_1280?: string
    poster?: string
    master?: string
  }
}

type ProcessingItem = {
  id: string
  kind: string
  state: string
  error?: string
}

type Props = {
  items: TimelineItem[]
  processing: ProcessingItem[]
  onTrash?: (id: string) => void
}

export function Timeline({ items, processing, onTrash }: Props) {
  return (
    <section className="timeline" data-testid="timeline">
      {processing.length > 0 ? (
        <ul className="processing-list" data-testid="media-processing-list">
          {processing.map((p) => (
            <li key={p.id} data-testid={p.state === 'failed' ? 'media-error' : 'media-processing'}>
              {p.kind === 'video' ? 'Видео' : 'Фото'} —{' '}
              {p.state === 'failed' ? `ошибка: ${p.error ?? 'обработка'}` : 'обрабатывается…'}
            </li>
          ))}
        </ul>
      ) : null}
      {items.length === 0 && processing.length === 0 ? (
        <p className="note" data-testid="timeline-empty">
          Лента пуста.
        </p>
      ) : null}
      <p data-testid="timeline-count" hidden={items.length === 0}>{`Готово: ${items.length}`}</p>
      <div className="timeline-grid">
        {items.map((it) => {
          const thumb = it.urls?.preview_640 ?? it.urls?.preview_1280 ?? it.urls?.poster ?? it.urls?.master
          const srcSet =
            it.urls?.preview_320 && it.urls?.preview_640 && it.urls?.preview_1280
              ? `${it.urls.preview_320} 320w, ${it.urls.preview_640} 640w, ${it.urls.preview_1280} 1280w`
              : undefined
          return (
            <article key={it.id} className="timeline-card" data-testid="timeline-item">
              {thumb ? (
                <img
                  src={thumb}
                  srcSet={srcSet}
                  sizes="(max-width: 390px) 90vw, 280px"
                  alt=""
                  className="timeline-thumb"
                  loading="lazy"
                  data-testid="timeline-thumb"
                />
              ) : (
                <div className="timeline-thumb placeholder" />
              )}
              <div className="timeline-meta">
                <span className="note">
                  {it.kind}
                  {it.width && it.height ? ` ${it.width}×${it.height}` : ''}
                </span>
                {onTrash ? (
                  <button
                    type="button"
                    className="btn sm danger"
                    data-testid={`trash-item-${it.id}`}
                    onClick={() => onTrash(it.id)}
                  >
                    В корзину
                  </button>
                ) : null}
              </div>
            </article>
          )
        })}
      </div>
    </section>
  )
}

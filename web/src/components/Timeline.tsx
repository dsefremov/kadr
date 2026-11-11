import { Link } from 'react-router-dom'
import {
  dayHeading,
  dayStats,
  formatDuration,
  groupTimeline,
  monthTitle,
  tileFlexGrow,
  type TimelineLike,
} from '../lib/timelineFormat'
import { TimelineScrub, type Bucket } from './TimelineScrub'

type ProcessingItem = {
  id: string
  kind: string
  state: string
  error?: string
}

type Props = {
  items: TimelineLike[]
  processing: ProcessingItem[]
  buckets?: Bucket[]
  activeMonth?: string
  onTrash?: (id: string) => void
  onScrubJump?: (anchorId: string, month: string) => void
}

export function Timeline({ items, processing, buckets = [], activeMonth, onTrash, onScrubJump }: Props) {
  const grouped = groupTimeline(items)

  return (
    <div className="layout" data-testid="timeline">
      <div className="tl">
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
        {grouped.map((block) => (
          <section key={block.month} data-month={block.month} id={`month-${block.month}`}>
            <h2 className="month">{monthTitle(block.month)}</h2>
            {block.days.map((day) => (
              <DaySection key={day.date} date={day.date} items={day.items} onTrash={onTrash} />
            ))}
          </section>
        ))}
      </div>
      {buckets.length > 0 && onScrubJump ? (
        <TimelineScrub buckets={buckets} activeMonth={activeMonth} onJump={onScrubJump} />
      ) : buckets.length > 0 ? (
        <TimelineScrub buckets={buckets} activeMonth={activeMonth} onJump={() => undefined} />
      ) : null}
    </div>
  )
}

function DaySection({
  date,
  items,
  onTrash,
}: {
  date: string
  items: TimelineLike[]
  onTrash?: (id: string) => void
}) {
  return (
    <div id={`day-${date}`}>
      <h2>
        {dayHeading(date)} <small>{dayStats(items)}</small>
      </h2>
      <div className="jg">
        {items.map((it) => (
          <TimelineTile key={it.id} item={it} onTrash={onTrash} />
        ))}
      </div>
    </div>
  )
}

function TimelineTile({ item, onTrash }: { item: TimelineLike; onTrash?: (id: string) => void }) {
  const thumb = item.urls?.preview_640 ?? item.urls?.preview_1280 ?? item.urls?.poster ?? item.urls?.master
  const grow = tileFlexGrow(item.width, item.height)
  const aspect = item.width && item.height ? item.width / item.height : 4 / 3
  const dur = formatDuration(item.duration_ms)
  return (
    <article
      className="t timeline-card"
      style={{ width: `calc(var(--rh) * ${aspect})`, flexGrow: grow }}
      data-testid="timeline-item"
    >
      <Link to={`/view/${item.id}`} data-testid={`open-viewer-${item.id}`} aria-label="Открыть">
        {thumb ? (
          <img
            src={thumb}
            alt=""
            className="timeline-thumb"
            loading="lazy"
            data-testid="timeline-thumb"
          />
        ) : (
          <div className="timeline-thumb placeholder" />
        )}
        {dur ? <span className="dur">{dur}</span> : null}
      </Link>
      {onTrash ? (
        <button
          type="button"
          className="trash-fab"
          data-testid={`trash-item-${item.id}`}
          aria-label="В корзину"
          title="В корзину"
          onClick={(e) => {
            e.preventDefault()
            e.stopPropagation()
            onTrash(item.id)
          }}
        >
          <TrashIcon />
        </button>
      ) : null}
    </article>
  )
}

function TrashIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true" focusable="false">
      <path
        fill="currentColor"
        d="M6 2h4l.5 1H14v1.5H2V3h3.5L6 2zm1 3.5h1.5V13H7V5.5zm3 0H11.5V13H10V5.5zM4.5 5.5H6V13H4.5V5.5zM3 14.5h10V4H3v10.5z"
      />
    </svg>
  )
}

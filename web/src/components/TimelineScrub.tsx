import { monthShort, monthTitle } from '../lib/timelineFormat'

export type Bucket = { month: string; label: string; count: number; anchor_id: string }

type Props = {
  buckets: Bucket[]
  activeMonth?: string
  onJump: (anchorId: string, month: string) => void
}

export function TimelineScrub({ buckets, activeMonth, onJump }: Props) {
  const active = activeMonth ?? buckets[0]?.month
  const activeLabel = buckets.find((b) => b.month === active)?.label ?? (active ? monthTitle(active) : '')
  return (
    <aside className="scrub" aria-label="Быстрая прокрутка по времени" data-testid="timeline-scrub">
      {buckets.map((b) => {
        const [, m] = b.month.split('-').map(Number)
        const yearTick = b.month.endsWith('-01')
        return (
          <button
            key={b.month}
            type="button"
            className="tick scrub-btn"
            data-testid={`scrub-${b.month}`}
            onClick={() => onJump(b.anchor_id, b.month)}
          >
            {yearTick ? <b>{b.month.slice(0, 4)}</b> : monthShort(m)}
          </button>
        )
      })}
      {activeLabel ? (
        <div className="now">
          <span>{activeLabel}</span>
          <i aria-hidden="true" />
        </div>
      ) : null}
    </aside>
  )
}

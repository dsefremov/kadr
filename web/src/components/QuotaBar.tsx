type Props = {
  used: number
  trash: number
  reserved: number
  limit: number
  status?: string
  compact?: boolean
}

function formatBytes(b: number): string {
  if (b >= 1_000_000_000) {
    const g = b / 1_000_000_000
    return `${g >= 10 ? g.toFixed(0) : g.toFixed(1)} ГБ`
  }
  const mb = b / 1_000_000
  return `${mb >= 100 ? mb.toFixed(0) : mb.toFixed(1)} МБ`
}

export function QuotaBar({ used, trash, reserved, limit, status, compact }: Props) {
  if (!limit) return null
  const accounted = used + trash + reserved
  const pctUsed = Math.min(100, Math.round((100 * used) / limit))
  const pctTrash = Math.min(100 - pctUsed, Math.round((100 * trash) / limit))
  const barClass =
    status === 'exceeded' ? 'bar full' : status === 'warning' ? 'bar hot' : 'bar'
  return (
    <div className={`quota${compact ? ' compact' : ''}`} data-testid="quota-bar">
      <span data-testid="quota-summary">
        {formatBytes(accounted)} из {formatBytes(limit)}
      </span>
      <div className={barClass} style={{ display: 'flex' }} aria-hidden>
        <b style={{ width: `${pctUsed}%`, borderRadius: 0 }} />
        {pctTrash > 0 ? <b style={{ width: `${pctTrash}%`, background: 'var(--amber)', borderRadius: 0 }} /> : null}
      </div>
      {trash > 0 ? (
        <span className="qt" data-testid="quota-trash-segment">
          из них в корзине {formatBytes(trash)}
        </span>
      ) : null}
    </div>
  )
}

export function QuotaWarningBanner({
  free,
  limit,
  trash,
  onOpenTrash,
}: {
  free: number
  limit: number
  trash: number
  onOpenTrash?: () => void
}) {
  return (
    <div className="banner warn" role="status" data-testid="quota-warning">
      <div>
        <div className="t1">Осталось {formatBytes(free)} из {formatBytes(limit)}</div>
        <p>
          Корзина ({formatBytes(trash)}) тоже занимает квоту. Освободите место в корзине или удалите ненужное.
        </p>
      </div>
      <span className="sp" />
      {onOpenTrash ? (
        <button type="button" className="btn sm" data-testid="quota-warning-trash-link" onClick={onOpenTrash}>
          Корзина
        </button>
      ) : null}
    </div>
  )
}

export function QuotaExceededBanner({
  need,
  free,
}: {
  need?: number
  free: number
}) {
  return (
    <div className="banner err" role="alert" data-testid="quota-exceeded">
      <div>
        <div className="t1">
          {need
            ? `Не хватает места: нужно ${formatBytes(need)}, свободно ${formatBytes(free)}`
            : `Квота исчерпана, свободно ${formatBytes(free)}`}
        </div>
        <p>Загрузка не началась. Очистите корзину или удалите ненужное.</p>
      </div>
    </div>
  )
}

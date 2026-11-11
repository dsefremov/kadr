const ruMonths = [
  'января',
  'февраля',
  'марта',
  'апреля',
  'мая',
  'июня',
  'июля',
  'августа',
  'сентября',
  'октября',
  'ноября',
  'декабря',
] as const

const ruMonthsTitle = [
  'Январь',
  'Февраль',
  'Март',
  'Апрель',
  'Май',
  'Июнь',
  'Июль',
  'Август',
  'Сентябрь',
  'Октябрь',
  'Ноябрь',
  'Декабрь',
] as const

const ruWeekdays = ['воскресенье', 'понедельник', 'вторник', 'среда', 'четверг', 'пятница', 'суббота'] as const

export type TimelineLike = {
  id: string
  kind: string
  taken_date?: string
  width?: number
  height?: number
  duration_ms?: number
  urls?: Record<string, string>
}

export function monthTitle(ym: string): string {
  const [y, m] = ym.split('-').map(Number)
  if (!y || !m) return ym
  return `${ruMonthsTitle[m - 1]} ${y}`
}

export function monthShort(m: number): string {
  return ['янв', 'фев', 'мар', 'апр', 'май', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек'][m - 1] ?? ''
}

export function dayHeading(takenDate: string): string {
  const d = new Date(`${takenDate}T12:00:00Z`)
  const now = new Date()
  const utcDay = (x: Date) => Date.UTC(x.getUTCFullYear(), x.getUTCMonth(), x.getUTCDate())
  const diff = (utcDay(now) - utcDay(d)) / 86400000
  const day = d.getUTCDate()
  const month = ruMonths[d.getUTCMonth()]
  const year = d.getUTCFullYear()
  const wd = ruWeekdays[d.getUTCDay()]
  const wdCap = wd.charAt(0).toUpperCase() + wd.slice(1)
  if (diff === 0) return `Сегодня, ${day} ${month}`
  if (diff === 1) return `Вчера, ${day} ${month}`
  return `${wdCap}, ${day} ${month} ${year}`
}

export function formatDuration(ms?: number): string {
  if (!ms || ms < 0) return ''
  const s = Math.floor(ms / 1000)
  const m = Math.floor(s / 60)
  const r = s % 60
  if (m >= 60) {
    const h = Math.floor(m / 60)
    return `${h}:${String(m % 60).padStart(2, '0')}:${String(r).padStart(2, '0')}`
  }
  return `${m}:${String(r).padStart(2, '0')}`
}

export type DayGroup = { date: string; items: TimelineLike[] }
export type MonthBlock = { month: string; days: DayGroup[] }

export function groupTimeline(items: TimelineLike[]): MonthBlock[] {
  const byMonth = new Map<string, Map<string, TimelineLike[]>>()
  for (const it of items) {
    const date = it.taken_date ?? '1970-01-01'
    const month = date.slice(0, 7)
    if (!byMonth.has(month)) byMonth.set(month, new Map())
    const days = byMonth.get(month)!
    if (!days.has(date)) days.set(date, [])
    days.get(date)!.push(it)
  }
  const months = [...byMonth.keys()].sort((a, b) => b.localeCompare(a))
  return months.map((month) => {
    const daysMap = byMonth.get(month)!
    const dates = [...daysMap.keys()].sort((a, b) => b.localeCompare(a))
    return {
      month,
      days: dates.map((date) => ({ date, items: daysMap.get(date)! })),
    }
  })
}

export function dayStats(items: TimelineLike[]): string {
  const photos = items.filter((i) => i.kind !== 'video').length
  const videos = items.filter((i) => i.kind === 'video').length
  const bits: string[] = []
  if (photos) bits.push(`${photos} фото`)
  if (videos) bits.push(`${videos} видео`)
  return bits.join(' · ')
}

export function tileFlexGrow(width?: number, height?: number): number {
  const w = width && width > 0 ? width : 4
  const h = height && height > 0 ? height : 3
  return Math.max(50, Math.round((w / h) * 100))
}

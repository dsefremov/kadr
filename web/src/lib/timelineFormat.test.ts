import { describe, expect, it } from 'vitest'
import { dayHeading, groupTimeline, monthTitle, takenDateKey } from './timelineFormat'

describe('timelineFormat', () => {
  it('groups by month and day', () => {
    const g = groupTimeline([
      { id: 'a', kind: 'photo', taken_date: '2026-11-07' },
      { id: 'b', kind: 'photo', taken_date: '2026-11-05' },
      { id: 'c', kind: 'photo', taken_date: '2026-10-12' },
    ])
    expect(g).toHaveLength(2)
    expect(g[0].month).toBe('2026-11')
    expect(g[0].days).toHaveLength(2)
  })

  it('groups RFC3339 taken_date as calendar day', () => {
    const g = groupTimeline([{ id: 'a', kind: 'photo', taken_date: '2026-11-11T00:00:00Z' }])
    expect(g[0].days[0].date).toBe('2026-11-11')
  })

  it('formats month title', () => {
    expect(monthTitle('2026-11')).toBe('Ноябрь 2026')
  })

  it('formats day heading', () => {
    expect(dayHeading('2026-11-07')).toMatch(/ноября/)
    expect(dayHeading('2026-11-07T00:00:00Z')).toMatch(/ноября/)
  })

  it('normalizes taken_date keys', () => {
    expect(takenDateKey('2026-11-11T00:00:00Z')).toBe('2026-11-11')
    expect(takenDateKey('2026-11-11')).toBe('2026-11-11')
  })
})

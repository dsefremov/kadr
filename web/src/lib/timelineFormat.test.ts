import { describe, expect, it } from 'vitest'
import { dayHeading, groupTimeline, monthTitle } from './timelineFormat'

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

  it('formats month title', () => {
    expect(monthTitle('2026-11')).toBe('Ноябрь 2026')
  })

  it('formats day heading', () => {
    expect(dayHeading('2026-11-07')).toMatch(/ноября/)
  })
})

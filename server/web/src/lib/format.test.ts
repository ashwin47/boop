import { describe, it, expect } from 'vitest'
import { relative, duration, compact, dayGroup, retentionLabel, clock, seenRange } from './format'

const now = new Date('2026-08-28T12:00:00Z')

describe('relative', () => {
  it('uses abbreviated units', () => {
    expect(relative('2026-08-28T11:58:00Z', now)).toBe('2m ago')
    expect(relative('2026-08-28T09:00:00Z', now)).toBe('3h ago')
    expect(relative('2026-08-26T12:00:00Z', now)).toBe('2d ago')
    expect(relative('2026-08-28T11:59:50Z', now)).toBe('just now')
  })
  it('falls back to a short date after a week', () => {
    expect(relative('2026-08-12T12:00:00Z', now)).toMatch(/Aug 12/)
  })
  it('handles bad input', () => {
    expect(relative(null, now)).toBe('—')
    expect(relative('nope', now)).toBe('—')
  })
})

describe('duration', () => {
  it('formats', () => {
    expect(duration(12)).toBe('12s')
    expect(duration(41 * 60)).toBe('41m')
    expect(duration(2 * 3600 + 14 * 60)).toBe('2h 14m')
    expect(duration(3 * 86400 + 3600)).toBe('3d 1h')
  })
})

describe('compact', () => {
  it('abbreviates', () => {
    expect(compact(999)).toBe('999')
    expect(compact(12400)).toBe('12k')
    expect(compact(1234)).toBe('1.2k')
    expect(compact(2_500_000)).toBe('2.5m')
  })
})

describe('dayGroup', () => {
  it('groups by day', () => {
    expect(dayGroup('2026-08-28T01:00:00', now)).toBe('Today')
    expect(dayGroup('2026-08-27T23:00:00', now)).toBe('Yesterday')
    expect(dayGroup('2026-08-01T12:00:00', now)).toMatch(/Aug 1/)
  })
})

describe('retentionLabel', () => {
  it('reads 0 as unlimited', () => {
    expect(retentionLabel(0)).toBe('Unlimited')
    expect(retentionLabel(30)).toBe('30 days')
  })
})

describe('clock / seenRange', () => {
  // These run in whatever TZ vitest has; compare against the same formatter.
  const hm = (iso: string) => new Date(iso).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', hour12: false })
  it('shows only the time on the same day', () => {
    expect(clock('2026-08-28T09:31:00Z', now)).toBe(hm('2026-08-28T09:31:00Z'))
  })
  it('adds the date on other days', () => {
    expect(clock('2026-08-12T09:31:00Z', now)).toMatch(/^Aug 12 /)
  })
  it('builds the seen range', () => {
    expect(seenRange('2026-08-28T09:31:00Z', '2026-08-28T10:42:00Z', now)).toBe(`First seen ${hm('2026-08-28T09:31:00Z')} · Last seen ${hm('2026-08-28T10:42:00Z')}`)
    expect(clock(null)).toBe('—')
  })
})

import { describe, expect, it } from 'vitest'
import { formatRupiah, timeAgo } from './format'

const NOW = new Date('2026-10-28T10:00:00Z')
const ago = (ms: number) => new Date(NOW.getTime() - ms).toISOString()

describe('formatRupiah', () => {
  it('uses Indonesian thousands separators', () => {
    expect(formatRupiah(1234567)).toBe('Rp 1.234.567')
    expect(formatRupiah(0)).toBe('Rp 0')
  })
})

describe('timeAgo', () => {
  it('handles null, fresh and future timestamps', () => {
    expect(timeAgo(null, NOW)).toBe('No movement')
    expect(timeAgo(ago(30_000), NOW)).toBe('just now')
    expect(timeAgo(ago(-60_000), NOW)).toBe('just now')
  })

  it('picks the largest fitting unit', () => {
    expect(timeAgo(ago(5 * 60_000), NOW)).toBe('5m ago')
    expect(timeAgo(ago(3 * 3_600_000), NOW)).toBe('3h ago')
    expect(timeAgo(ago(5 * 86_400_000), NOW)).toBe('5d ago')
  })
})
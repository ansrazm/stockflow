export const LOW_STOCK_THRESHOLD = 10

const rupiah = new Intl.NumberFormat('id-ID')

export function formatRupiah(n: number): string {
  return `Rp ${rupiah.format(n)}`
}

export function timeAgo(iso: string | null, now: Date = new Date()): string {
  if (!iso) return 'No movement'
  const seconds = Math.max(0, Math.floor((now.getTime() - new Date(iso).getTime()) / 1000))
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}
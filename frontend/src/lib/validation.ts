export function validateQty(raw: string): string | null {
  const s = raw.trim()
  if (s === '') return 'Quantity is required'
  if (!/^\d+$/.test(s)) return 'Quantity must be a whole number greater than 0'
  if (Number(s) <= 0) return 'Quantity must be greater than 0'
  return null
}
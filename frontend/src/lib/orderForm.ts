import type { OrderInput } from '../api/types'
import { validateQty } from './validation'

export interface DraftLine {
  key: number
  sku: string
  qty: string
}

export interface LineErrors {
  sku?: string
  qty?: string
}

export interface OrderErrors {
  customer?: string
  items?: string
  lines: LineErrors[]
}

export function validateOrder(customer: string, lines: DraftLine[]): OrderErrors {
  const errors: OrderErrors = { lines: lines.map(() => ({})) }
  if (customer.trim() === '') errors.customer = 'Customer name is required'
  if (lines.length === 0) errors.items = 'Add at least one item'

  const seen = new Set<string>()
  lines.forEach((l, i) => {
    const sku = l.sku.trim()
    if (sku === '') errors.lines[i].sku = 'SKU is required'
    else if (seen.has(sku)) errors.lines[i].sku = `Duplicate SKU: ${sku}`
    else seen.add(sku)

    const qtyError = validateQty(l.qty)
    if (qtyError) errors.lines[i].qty = qtyError
  })
  return errors
}

export function hasErrors(e: OrderErrors): boolean {
  return Boolean(e.customer || e.items || e.lines.some((l) => l.sku || l.qty))
}

export function toOrderInput(customer: string, lines: DraftLine[]): OrderInput {
  return {
    customer: customer.trim(),
    items: lines.map((l) => ({ sku: l.sku.trim(), qty: Number(l.qty) })),
  }
}

// backend keys: "customer", "items", "items[2].sku", "items[2].qty"
export function mapServerFields(fields: Record<string, string>, lineCount: number): OrderErrors {
  const out: OrderErrors = { lines: Array.from({ length: lineCount }, () => ({})) }
  for (const [key, msg] of Object.entries(fields)) {
    if (key === 'customer') {
      out.customer = msg
      continue
    }
    const m = /^items\[(\d+)\]\.(sku|qty)$/.exec(key)
    const line = m ? out.lines[Number(m[1])] : undefined
    if (m && line) line[m[2] as 'sku' | 'qty'] = msg
    else out.items = msg
  }
  return out
}
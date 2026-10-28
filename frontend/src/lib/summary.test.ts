import { describe, expect, it } from 'vitest'
import type { Product } from '../api/types'
import { LOW_STOCK_THRESHOLD } from './format'
import { computeSummary } from './summary'

const make = (over: Partial<Product>): Product => ({
  id: 1,
  sku: 'TST-001',
  name: 'Test product',
  price: 1000,
  stock: 50,
  is_active: true,
  created_at: '2026-10-28T00:00:00Z',
  last_movement_at: null,
  ...over,
})

describe('computeSummary', () => {
  it('returns zeros for an empty list', () => {
    expect(computeSummary([])).toEqual({
      total: 0,
      active: 0,
      stockValue: 0,
      lowStock: 0,
    })
  })

  it('counts products, active ones and total stock value', () => {
    const products = [
      make({ id: 1, price: 650_000, stock: 2 }),
      make({ id: 2, price: 100_000, stock: 10, is_active: false }),
      make({ id: 3, price: 5_000, stock: 40 }),
    ]
    const s = computeSummary(products)
    expect(s.total).toBe(3)
    expect(s.active).toBe(2)
    expect(s.stockValue).toBe(650_000 * 2 + 100_000 * 10 + 5_000 * 40)
  })

  it('treats stock at or below the threshold as low stock, including zero', () => {
    const products = [
      make({ id: 1, stock: 0 }),
      make({ id: 2, stock: LOW_STOCK_THRESHOLD }),
      make({ id: 3, stock: LOW_STOCK_THRESHOLD + 1 }),
    ]
    expect(computeSummary(products).lowStock).toBe(2)
  })
})
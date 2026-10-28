import type { Product } from '../api/types'
import { LOW_STOCK_THRESHOLD } from './format'

export interface Summary {
  total: number
  active: number
  stockValue: number
  lowStock: number
}

export function computeSummary(products: Product[]): Summary {
  return {
    total: products.length,
    active: products.filter((p) => p.is_active).length,
    stockValue: products.reduce((sum, p) => sum + p.price * p.stock, 0),
    lowStock: products.filter((p) => p.stock <= LOW_STOCK_THRESHOLD).length,
  }
}
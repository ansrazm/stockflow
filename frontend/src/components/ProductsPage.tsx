import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { Product } from '../api/types'
import { useDebounce } from '../hooks/useDebounce'
import { LOW_STOCK_THRESHOLD, formatRupiah, timeAgo } from '../lib/format'
import AdjustStockModal from './AdjustStockModal'
import SummaryCards from './SummaryCards'

type Filter = 'all' | 'active' | 'inactive' | 'low'

type State =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'ready'; products: Product[] }

const FILTERS: { value: Filter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
  { value: 'low', label: `Low stock (≤${LOW_STOCK_THRESHOLD})` },
]

function matches(p: Product, filter: Filter): boolean {
  switch (filter) {
    case 'active':
      return p.is_active
    case 'inactive':
      return !p.is_active
    case 'low':
      return p.stock <= LOW_STOCK_THRESHOLD
    default:
      return true
  }
}

export default function ProductsPage() {
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<Filter>('all')
  const [state, setState] = useState<State>({ status: 'loading' })
  const [reloadKey, setReloadKey] = useState(0)
  const [adjusting, setAdjusting] = useState<Product | null>(null)
  const debouncedSearch = useDebounce(search, 300)

  useEffect(() => {
    let cancelled = false
    setState({ status: 'loading' })
    api
      .listProducts({ search: debouncedSearch, limit: 100 })
      .then((res) => {
        if (!cancelled) setState({ status: 'ready', products: res.data })
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setState({ status: 'error', message: e instanceof Error ? e.message : 'Something went wrong' })
        }
      })
    return () => {
      cancelled = true
    }
  }, [debouncedSearch, reloadKey])

  const rows = state.status === 'ready' ? state.products.filter((p) => matches(p, filter)) : []

  return (
    <section>
      <SummaryCards reloadKey={reloadKey} />

      <h2>Products</h2>

      <div className="toolbar">
        <input
          type="search"
          placeholder="Search by SKU or name"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select value={filter} onChange={(e) => setFilter(e.target.value as Filter)}>
          {FILTERS.map((f) => (
            <option key={f.value} value={f.value}>
              {f.label}
            </option>
          ))}
        </select>
      </div>

      {state.status === 'loading' && <p className="state">Loading products…</p>}

      {state.status === 'error' && (
        <div className="state error" role="alert">
          <p>Could not load products: {state.message}</p>
          <button onClick={() => setReloadKey((k) => k + 1)}>Retry</button>
        </div>
      )}

      {state.status === 'ready' && rows.length === 0 && (
        <p className="state">No products match your search or filter.</p>
      )}

      {state.status === 'ready' && rows.length > 0 && (
        <table>
          <thead>
            <tr>
              <th>SKU</th>
              <th>Name</th>
              <th className="num">Price</th>
              <th className="num">Stock</th>
              <th>Status</th>
              <th>Last movement</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((p) => (
              <tr key={p.id}>
                <td>{p.sku}</td>
                <td>{p.name}</td>
                <td className="num">{formatRupiah(p.price)}</td>
                <td className="num">
                  {p.stock}
                  {p.stock <= LOW_STOCK_THRESHOLD && <span className="badge low">low</span>}
                </td>
                <td>
                  <span className={`badge ${p.is_active ? 'active' : 'inactive'}`}>
                    {p.is_active ? 'Active' : 'Inactive'}
                  </span>
                </td>
                <td>{timeAgo(p.last_movement_at)}</td>
                <td>
                  <button onClick={() => setAdjusting(p)}>Adjust</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {adjusting && (
        <AdjustStockModal
          product={adjusting}
          onClose={() => setAdjusting(null)}
          onSaved={() => {
            setAdjusting(null)
            setReloadKey((k) => k + 1)
          }}
        />
      )}
    </section>
  )
}
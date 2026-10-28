import { useEffect, useState } from 'react'
import { api } from '../api/client'
import { formatRupiah } from '../lib/format'
import { computeSummary, type Summary } from '../lib/summary'

export default function SummaryCards({ reloadKey }: { reloadKey: number }) {
  const [summary, setSummary] = useState<Summary | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .listProducts({ limit: 100 })
      .then((res) => {
        if (!cancelled) setSummary(computeSummary(res.data))
      })
      .catch(() => {
        if (!cancelled) setSummary(null)
      })
    return () => {
      cancelled = true
    }
  }, [reloadKey])

  const cards = [
    { label: 'Total products', value: summary ? String(summary.total) : '–' },
    { label: 'Active products', value: summary ? String(summary.active) : '–' },
    { label: 'Total stock value', value: summary ? formatRupiah(summary.stockValue) : '–' },
    { label: 'Low stock', value: summary ? String(summary.lowStock) : '–' },
  ]

  return (
    <div className="cards">
      {cards.map((c) => (
        <div className="card" key={c.label}>
          <div className="card-label">{c.label}</div>
          <div className="card-value">{c.value}</div>
        </div>
      ))}
    </div>
  )
}
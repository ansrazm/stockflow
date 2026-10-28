import { useEffect, useRef, useState, type FormEvent } from 'react'
import { ApiError, api } from '../api/client'
import type { Product } from '../api/types'
import {
  hasErrors,
  mapServerFields,
  toOrderInput,
  validateOrder,
  type DraftLine,
  type OrderErrors,
} from '../lib/orderForm'

interface Props {
  onCancel: () => void
  onCreated: (orderId: number) => void
}

interface Conflict {
  sku: string
  available: number
}

export default function CreateOrderPage({ onCancel, onCreated }: Props) {
  const nextKey = useRef(1)
  const [customer, setCustomer] = useState('')
  const [lines, setLines] = useState<DraftLine[]>([{ key: 0, sku: '', qty: '1' }])
  const [products, setProducts] = useState<Product[]>([])
  const [showErrors, setShowErrors] = useState(false)
  const [serverErrors, setServerErrors] = useState<OrderErrors | null>(null)
  const [conflict, setConflict] = useState<Conflict | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  // only feeds the SKU suggestions; the form still works if this fails
  useEffect(() => {
    let cancelled = false
    api
      .listProducts({ limit: 100 })
      .then((res) => {
        if (!cancelled) setProducts(res.data)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [])

  function clearServerState() {
    setServerErrors(null)
    setConflict(null)
    setFormError(null)
  }

  function updateLine(key: number, patch: Partial<DraftLine>) {
    clearServerState()
    setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...patch } : l)))
  }

  function addLine() {
    clearServerState()
    setLines((ls) => [...ls, { key: nextKey.current++, sku: '', qty: '1' }])
  }

  function removeLine(key: number) {
    clearServerState()
    setLines((ls) => ls.filter((l) => l.key !== key))
  }

  const clientErrors = validateOrder(customer, lines)
  const shown: OrderErrors | null = serverErrors ?? (showErrors ? clientErrors : null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (saving) return
    clearServerState()
    if (hasErrors(clientErrors)) {
      setShowErrors(true)
      return
    }
    setSaving(true)
    try {
      const order = await api.createOrder(toOrderInput(customer, lines))
      onCreated(order.id)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'INSUFFICIENT_STOCK') {
        setConflict({ sku: String(err.details.sku), available: Number(err.details.available) })
      } else if (err instanceof ApiError && err.code === 'VALIDATION_ERROR' && err.details.fields) {
        setServerErrors(mapServerFields(err.details.fields as Record<string, string>, lines.length))
      } else if (err instanceof ApiError && err.code === 'UNKNOWN_SKU') {
        const bad = String(err.details.sku)
        const errs: OrderErrors = { lines: lines.map(() => ({})) }
        lines.forEach((l, i) => {
          if (l.sku.trim() === bad) errs.lines[i].sku = `Unknown SKU: ${bad}`
        })
        setServerErrors(errs)
      } else {
        setFormError(err instanceof Error ? err.message : 'Something went wrong')
      }
      setSaving(false)
    }
  }

  return (
    <section>
      <h2>New order</h2>
      <form onSubmit={submit} className="order-form">
        <label>
          Customer
          <input value={customer} onChange={(e) => { clearServerState(); setCustomer(e.target.value) }} disabled={saving} />
        </label>
        {shown?.customer && <p className="field-error">{shown.customer}</p>}

        <h3>Items</h3>
        {shown?.items && <p className="field-error">{shown.items}</p>}

        <datalist id="sku-options">
          {products.map((p) => (
            <option key={p.id} value={p.sku}>
              {p.name} (stock {p.stock})
            </option>
          ))}
        </datalist>

        {lines.map((l, i) => {
          const failed = conflict !== null && conflict.sku === l.sku.trim()
          return (
            <div key={l.key} className={`line${failed ? ' conflict' : ''}`}>
              <div className="line-row">
                <input
                  list="sku-options"
                  placeholder="SKU"
                  value={l.sku}
                  onChange={(e) => updateLine(l.key, { sku: e.target.value })}
                  disabled={saving}
                />
                <input
                  inputMode="numeric"
                  placeholder="Qty"
                  className="qty"
                  value={l.qty}
                  onChange={(e) => updateLine(l.key, { qty: e.target.value })}
                  disabled={saving}
                />
                <button type="button" onClick={() => removeLine(l.key)} disabled={saving || lines.length === 1}>
                  Remove
                </button>
              </div>
              {shown?.lines[i]?.sku && <p className="field-error">{shown.lines[i].sku}</p>}
              {shown?.lines[i]?.qty && <p className="field-error">{shown.lines[i].qty}</p>}
              {failed && conflict && (
                <p className="field-error" role="alert">
                  Insufficient stock for {conflict.sku}: only {conflict.available} available.
                </p>
              )}
            </div>
          )
        })}

        <button type="button" onClick={addLine} disabled={saving}>
          + Add item
        </button>

        {formError && (
          <p className="field-error" role="alert">
            {formError}
          </p>
        )}

        <div className="actions">
          <button type="button" onClick={onCancel} disabled={saving}>
            Cancel
          </button>
          <button type="submit" disabled={saving}>
            {saving ? 'Creating…' : 'Create order'}
          </button>
        </div>
      </form>
    </section>
  )
}
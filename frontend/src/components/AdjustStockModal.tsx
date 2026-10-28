import { useState, type FormEvent } from 'react'
import { ApiError, api } from '../api/client'
import type { Product } from '../api/types'
import { validateQty } from '../lib/validation'

interface Props {
  product: Product
  onClose: () => void
  onSaved: () => void
}

export default function AdjustStockModal({ product, onClose, onSaved }: Props) {
  const [type, setType] = useState<'in' | 'out'>('in')
  const [qty, setQty] = useState('')
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const qtyError = validateQty(qty)
  const canSubmit = qtyError === null && !saving

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!canSubmit) return
    setSaving(true)
    setError(null)
    try {
      await api.adjustStock(product.id, { type, qty: Number(qty), note: note.trim() })
      onSaved()
    } catch (err) {
      if (err instanceof ApiError && err.code === 'INSUFFICIENT_STOCK') {
        setError(`Insufficient stock: only ${String(err.details.available)} available.`)
      } else if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Something went wrong')
      }
      setSaving(false)
    }
  }

  return (
    <div className="overlay" onClick={() => !saving && onClose()}>
      <form
        className="modal"
        role="dialog"
        aria-modal="true"
        aria-label={`Adjust stock for ${product.sku}`}
        onClick={(e) => e.stopPropagation()}
        onSubmit={submit}
      >
        <h3>Adjust stock</h3>
        <p className="muted">
          {product.sku} · {product.name} · current stock: {product.stock}
        </p>

        <label>
          Direction
          <select value={type} onChange={(e) => setType(e.target.value as 'in' | 'out')} disabled={saving}>
            <option value="in">Stock in</option>
            <option value="out">Stock out</option>
          </select>
        </label>

        <label>
          Quantity
          <input
            inputMode="numeric"
            value={qty}
            onChange={(e) => setQty(e.target.value)}
            disabled={saving}
            autoFocus
          />
        </label>
        {qty !== '' && qtyError && <p className="field-error">{qtyError}</p>}

        <label>
          Note
          <input value={note} onChange={(e) => setNote(e.target.value)} disabled={saving} />
        </label>

        {error && (
          <p className="field-error" role="alert">
            {error}
          </p>
        )}

        <div className="actions">
          <button type="button" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button type="submit" disabled={!canSubmit}>
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </form>
    </div>
  )
}
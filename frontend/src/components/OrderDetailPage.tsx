import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { Order } from '../api/types'
import { formatRupiah } from '../lib/format'

interface Props {
  id: number
  created: boolean
  onBack: () => void
}

type State =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'ready'; order: Order }

export default function OrderDetailPage({ id, created, onBack }: Props) {
  const [state, setState] = useState<State>({ status: 'loading' })
  const [reloadKey, setReloadKey] = useState(0)
  const [confirming, setConfirming] = useState(false)
  const [cancelling, setCancelling] = useState(false)
  const [cancelError, setCancelError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(created ? 'Order created.' : null)

  useEffect(() => {
    let cancelled = false
    setState({ status: 'loading' })
    api
      .getOrder(id)
      .then((order) => {
        if (!cancelled) setState({ status: 'ready', order })
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setState({ status: 'error', message: e instanceof Error ? e.message : 'Something went wrong' })
        }
      })
    return () => {
      cancelled = true
    }
  }, [id, reloadKey])

  async function doCancel() {
    setCancelling(true)
    setCancelError(null)
    try {
      const order = await api.cancelOrder(id)
      setState({ status: 'ready', order })
      setNotice('Order cancelled. Stock has been restored.')
      setConfirming(false)
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        // someone else cancelled it first: show the real state
        setConfirming(false)
        setNotice(null)
        setReloadKey((k) => k + 1)
      } else {
        setCancelError(err instanceof Error ? err.message : 'Something went wrong')
      }
    } finally {
      setCancelling(false)
    }
  }

  return (
    <section>
      <h2>Order #{id}</h2>

      {state.status === 'loading' && <p className="state">Loading order…</p>}

      {state.status === 'error' && (
        <div className="state error" role="alert">
          <p>Could not load order: {state.message}</p>
          <button onClick={() => setReloadKey((k) => k + 1)}>Retry</button>
        </div>
      )}

      {state.status === 'ready' && (
        <>
          {notice && <p className="notice">{notice}</p>}
          <p>
            Customer: <strong>{state.order.customer}</strong> ·{' '}
            <span className={`badge ${state.order.status === 'CREATED' ? 'active' : 'inactive'}`}>
              {state.order.status}
            </span>
          </p>

          <table>
            <thead>
              <tr>
                <th>SKU</th>
                <th className="num">Qty</th>
                <th className="num">Unit price</th>
                <th className="num">Subtotal</th>
              </tr>
            </thead>
            <tbody>
              {state.order.items.map((it) => (
                <tr key={it.sku}>
                  <td>{it.sku}</td>
                  <td className="num">{it.qty}</td>
                  <td className="num">{formatRupiah(it.unit_price)}</td>
                  <td className="num">{formatRupiah(it.unit_price * it.qty)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <th colSpan={3} className="num">Total</th>
                <th className="num">{formatRupiah(state.order.total)}</th>
              </tr>
            </tfoot>
          </table>

          <div className="actions">
            <button onClick={onBack}>Back to products</button>
            {state.order.status === 'CREATED' && (
              <button onClick={() => setConfirming(true)}>Cancel order</button>
            )}
          </div>

          {confirming && (
            <div className="overlay" onClick={() => !cancelling && setConfirming(false)}>
              <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
                <h3>Cancel order #{id}?</h3>
                <p className="muted">All items go back into stock. This cannot be undone.</p>
                {cancelError && (
                  <p className="field-error" role="alert">
                    {cancelError}
                  </p>
                )}
                <div className="actions">
                  <button onClick={() => setConfirming(false)} disabled={cancelling}>
                    Keep order
                  </button>
                  <button onClick={doCancel} disabled={cancelling}>
                    {cancelling ? 'Cancelling…' : 'Yes, cancel'}
                  </button>
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </section>
  )
}
import { useState, type FormEvent } from 'react'
import CreateOrderPage from './components/CreateOrderPage'
import OrderDetailPage from './components/OrderDetailPage'
import ProductsPage from './components/ProductsPage'
import './index.css'

type View =
  | { name: 'products' }
  | { name: 'new-order' }
  | { name: 'order'; id: number; created: boolean }

export default function App() {
  const [view, setView] = useState<View>({ name: 'products' })
  const [openId, setOpenId] = useState('')

  function openOrder(e: FormEvent) {
    e.preventDefault()
    const id = Number(openId)
    if (Number.isInteger(id) && id > 0) {
      setView({ name: 'order', id, created: false })
      setOpenId('')
    }
  }

  return (
    <main>
      <h1>StockFlow</h1>
      <nav className="nav">
        <button onClick={() => setView({ name: 'products' })}>Products</button>
        <button onClick={() => setView({ name: 'new-order' })}>New order</button>
        <form onSubmit={openOrder} className="open-order">
          <input
            inputMode="numeric"
            placeholder="Order #"
            value={openId}
            onChange={(e) => setOpenId(e.target.value)}
          />
          <button type="submit">Open</button>
        </form>
      </nav>

      {view.name === 'products' && <ProductsPage />}
      {view.name === 'new-order' && (
        <CreateOrderPage
          onCancel={() => setView({ name: 'products' })}
          onCreated={(id) => setView({ name: 'order', id, created: true })}
        />
      )}
      {view.name === 'order' && (
        <OrderDetailPage
          key={view.id}
          id={view.id}
          created={view.created}
          onBack={() => setView({ name: 'products' })}
        />
      )}
    </main>
  )
}
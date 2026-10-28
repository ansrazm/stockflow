export interface Product {
  id: number
  sku: string
  name: string
  price: number
  stock: number
  is_active: boolean
  created_at: string
  last_movement_at: string | null
}

export interface ProductList {
  data: Product[]
  page: number
  limit: number
  total: number
}

export interface Movement {
  id: number
  product_id: number
  type: 'in' | 'out'
  qty: number
  note: string | null
  created_at: string
}

export interface OrderItem {
  sku: string
  qty: number
  unit_price: number
}

export interface Order {
  id: number
  customer: string
  status: 'CREATED' | 'CANCELLED'
  total: number
  created_at: string
  cancelled_at: string | null
  items: OrderItem[]
}

export interface OrderInput {
  customer: string
  items: { sku: string; qty: number }[]
}
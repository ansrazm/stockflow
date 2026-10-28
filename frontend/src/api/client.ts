import type { Movement, Order, OrderInput, Product, ProductList } from './types'

export class ApiError extends Error {
  status: number
  code: string
  // everything else the backend put in the error object (sku, available, fields...)
  details: Record<string, unknown>

  constructor(status: number, code: string, message: string, details: Record<string, unknown>) {
    super(message)
    this.status = status
    this.code = code
    this.details = details
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response
  try {
    res = await fetch(`/api${path}`, {
      ...init,
      headers: { 'Content-Type': 'application/json', ...init?.headers },
    })
  } catch {
    throw new ApiError(0, 'NETWORK_ERROR', 'Cannot reach the server', {})
  }

  if (res.status === 204) return undefined as T

  const body = await res.json().catch(() => null)
  if (!res.ok) {
    const e = body?.error ?? {}
    const { code, message, ...details } = e
    throw new ApiError(res.status, code ?? 'UNKNOWN', message ?? res.statusText, details)
  }
  return body as T
}

export const api = {
  listProducts: (params: { search?: string; page?: number; limit?: number }) => {
    const q = new URLSearchParams()
    if (params.search) q.set('search', params.search)
    if (params.page) q.set('page', String(params.page))
    if (params.limit) q.set('limit', String(params.limit))
    return request<ProductList>(`/products?${q}`)
  },
  adjustStock: (id: number, body: { type: 'in' | 'out'; qty: number; note: string }) =>
    request<Product>(`/products/${id}/stock`, { method: 'POST', body: JSON.stringify(body) }),
  listMovements: (id: number) => request<{ data: Movement[] }>(`/products/${id}/movements`),
  createOrder: (body: OrderInput) =>
    request<Order>('/orders', { method: 'POST', body: JSON.stringify(body) }),
  getOrder: (id: number) => request<Order>(`/orders/${id}`),
  cancelOrder: (id: number) => request<Order>(`/orders/${id}/cancel`, { method: 'POST' }),
}
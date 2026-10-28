import { describe, expect, it } from 'vitest'
import { hasErrors, mapServerFields, toOrderInput, validateOrder, type DraftLine } from './orderForm'

const line = (key: number, sku: string, qty: string): DraftLine => ({ key, sku, qty })

describe('validateOrder', () => {
  it('passes a valid order', () => {
    const e = validateOrder('Budi', [line(0, 'KBD-001', '2'), line(1, 'MSE-001', '1')])
    expect(hasErrors(e)).toBe(false)
  })

  it('requires customer and at least one item', () => {
    const e = validateOrder('  ', [])
    expect(e.customer).toBe('Customer name is required')
    expect(e.items).toBe('Add at least one item')
  })

  it('flags empty sku and bad qty per line', () => {
    const e = validateOrder('Budi', [line(0, '', '0'), line(1, 'MSE-001', 'abc')])
    expect(e.lines[0]).toEqual({ sku: 'SKU is required', qty: 'Quantity must be greater than 0' })
    expect(e.lines[1].qty).toBe('Quantity must be a whole number greater than 0')
  })

  it('flags the second occurrence of a duplicate sku', () => {
    const e = validateOrder('Budi', [line(0, 'KBD-001', '1'), line(1, ' KBD-001 ', '2')])
    expect(e.lines[0].sku).toBeUndefined()
    expect(e.lines[1].sku).toBe('Duplicate SKU: KBD-001')
  })
})

describe('toOrderInput', () => {
  it('trims strings and converts qty to number', () => {
    expect(toOrderInput(' Budi ', [line(0, ' KBD-001 ', '3')])).toEqual({
      customer: 'Budi',
      items: [{ sku: 'KBD-001', qty: 3 }],
    })
  })
})

describe('mapServerFields', () => {
  it('maps backend field keys onto customer, items and lines', () => {
    const e = mapServerFields(
      {
        customer: 'customer is required',
        items: 'at least one item is required',
        'items[1].qty': 'qty is required',
        'items[0].sku': 'sku is required',
      },
      2,
    )
    expect(e.customer).toBe('customer is required')
    expect(e.items).toBe('at least one item is required')
    expect(e.lines[0].sku).toBe('sku is required')
    expect(e.lines[1].qty).toBe('qty is required')
  })

  it('falls back to a general message for unknown keys or out-of-range lines', () => {
    const e = mapServerFields({ 'items[9].qty': 'bad', other: 'oops' }, 1)
    expect(e.items).toBeDefined()
    expect(e.lines[0]).toEqual({})
  })
})
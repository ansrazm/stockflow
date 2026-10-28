import { describe, expect, it } from 'vitest'
import { validateQty } from './validation'

describe('validateQty', () => {
  it('accepts positive whole numbers, ignoring surrounding spaces', () => {
    expect(validateQty('1')).toBeNull()
    expect(validateQty(' 20 ')).toBeNull()
  })

  it('requires a value', () => {
    expect(validateQty('')).toBe('Quantity is required')
    expect(validateQty('   ')).toBe('Quantity is required')
  })

  it('rejects zero', () => {
    expect(validateQty('0')).toBe('Quantity must be greater than 0')
    expect(validateQty('00')).toBe('Quantity must be greater than 0')
  })

  it('rejects anything that is not a whole number', () => {
    for (const raw of ['abc', '1.5', '-3', '1e3']) {
      expect(validateQty(raw)).toBe('Quantity must be a whole number greater than 0')
    }
  })
})
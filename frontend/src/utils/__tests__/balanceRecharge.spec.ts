import { describe, expect, it } from 'vitest'
import { calculateBalanceCredit } from '../balanceRecharge'

describe('balanceRecharge', () => {
  it('calculates credited balance from cny amount and fixed ratio', () => {
    expect(calculateBalanceCredit(10, 100)).toBe(1000)
  })

  it('returns 0 when the ratio is invalid', () => {
    expect(calculateBalanceCredit(10, 0)).toBe(0)
    expect(calculateBalanceCredit(10, -5)).toBe(0)
    expect(calculateBalanceCredit(10, Number.NaN)).toBe(0)
  })

  it('returns 0 when the amount is invalid', () => {
    expect(calculateBalanceCredit(0, 100)).toBe(0)
    expect(calculateBalanceCredit(-1, 100)).toBe(0)
    expect(calculateBalanceCredit(Number.NaN, 100)).toBe(0)
  })
})

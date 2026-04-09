import { describe, expect, it, vi, afterEach } from 'vitest'
import { createIdempotencyKey } from '../idempotency'

describe('idempotency', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('uses crypto.randomUUID when available', () => {
    vi.stubGlobal('crypto', {
      randomUUID: vi.fn(() => 'uuid-123')
    })

    expect(createIdempotencyKey('recharge')).toBe('recharge-uuid-123')
  })

  it('falls back to a generated suffix when crypto.randomUUID is unavailable', () => {
    vi.stubGlobal('crypto', undefined)

    const key = createIdempotencyKey('subscription')
    expect(key).toMatch(/^subscription-/)
    expect(key.length).toBeGreaterThan('subscription-'.length)
  })
})

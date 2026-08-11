import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  emitCouponManagerTelemetry,
  onCouponManagerTelemetry,
  validateCouponManagerTelemetryEvent,
} from '../coupon-manager-telemetry'

describe('coupon manager telemetry', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('accepts known events and strips unknown props', () => {
    const validated = validateCouponManagerTelemetryEvent('coupon_created', {
      discountType: 'percent',
      unknown: 'x',
    })
    expect(validated).toEqual({
      event: 'coupon_created',
      props: { discountType: 'percent' },
    })
  })

  it('rejects forbidden keys (no code / PII)', () => {
    expect(
      validateCouponManagerTelemetryEvent('coupon_created', { code: 'LAUNCH25' }),
    ).toBeNull()
  })

  it('emits to listeners', () => {
    const spy = vi.fn()
    const off = onCouponManagerTelemetry(spy)
    emitCouponManagerTelemetry('coupon_share_link_copied', { target: 'app' })
    expect(spy).toHaveBeenCalledWith({
      event: 'coupon_share_link_copied',
      props: { target: 'app' },
    })
    off()
  })
})

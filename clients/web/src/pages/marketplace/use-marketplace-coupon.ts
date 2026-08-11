/**
 * Coupon apply/remove state for marketplace course detail (MKTC.5).
 */

import { useCallback, useState } from 'react'
import {
  MarketplaceApiError,
  previewMarketplaceCoupon,
  type MarketplaceCouponPreview,
} from '../../lib/marketplace-api'
import {
  clearPendingCoupon,
  normalizeCouponInput,
  rememberPendingCoupon,
  replaceCouponOutOfUrl,
  type CouponReason,
} from '../../lib/marketplace-coupon'
import { emitLearnerCouponTelemetry } from '../../lib/marketplace-learner-coupon-telemetry'
import type { CouponFieldStatus } from './marketplace-coupon-field'

export function useMarketplaceCoupon(slug: string | undefined, couponsEnabled: boolean) {
  const [status, setStatus] = useState<CouponFieldStatus>('idle')
  const [preview, setPreview] = useState<MarketplaceCouponPreview | null>(null)
  const [errorReason, setErrorReason] = useState<CouponReason | null>(null)
  const [seed, setSeed] = useState('')
  const [announcement, setAnnouncement] = useState('')

  const applyPreview = useCallback(
    (
      next: MarketplaceCouponPreview,
      code: string,
      fromUrl: boolean,
      formatApplied: (p: MarketplaceCouponPreview) => string,
    ) => {
      const normalized = normalizeCouponInput(code || next.code)
      setSeed(normalized)
      if (next.applied) {
        setPreview(next)
        setStatus('applied')
        setErrorReason(null)
        if (slug) rememberPendingCoupon(slug, next.code || normalized)
        replaceCouponOutOfUrl()
        setAnnouncement(formatApplied(next))
        emitLearnerCouponTelemetry(fromUrl ? 'coupon_from_url' : 'coupon_applied', {
          result: 'ok',
          fromUrl,
        })
      } else {
        setPreview(null)
        setStatus('rejected')
        setErrorReason((next.reason as CouponReason) || 'not_found')
        emitLearnerCouponTelemetry(fromUrl ? 'coupon_from_url' : 'coupon_applied', {
          result: String(next.reason || 'not_found'),
          fromUrl,
        })
      }
    },
    [slug],
  )

  const runPreview = useCallback(
    async (
      code: string,
      fromUrl: boolean,
      formatApplied: (p: MarketplaceCouponPreview) => string,
    ) => {
      if (!slug || !couponsEnabled) return
      const normalized = normalizeCouponInput(code)
      if (!normalized) return
      setStatus('checking')
      setErrorReason(null)
      try {
        const next = await previewMarketplaceCoupon(slug, normalized)
        applyPreview(next, normalized, fromUrl, formatApplied)
      } catch (e: unknown) {
        if (e instanceof MarketplaceApiError && e.status === 429) {
          setStatus('rate_limited')
          setPreview(null)
          setErrorReason(null)
          emitLearnerCouponTelemetry('coupon_applied', { result: 'rate_limited', fromUrl })
          return
        }
        setStatus('rejected')
        setPreview(null)
        setErrorReason('not_found')
        emitLearnerCouponTelemetry('coupon_applied', { result: 'error', fromUrl })
      }
    },
    [slug, couponsEnabled, applyPreview],
  )

  const remove = useCallback(() => {
    setPreview(null)
    setStatus('idle')
    setErrorReason(null)
    setSeed('')
    setAnnouncement('')
    if (slug) clearPendingCoupon(slug)
    replaceCouponOutOfUrl()
    emitLearnerCouponTelemetry('coupon_removed')
  }, [slug])

  const clearOnOwned = useCallback(() => {
    setPreview(null)
    setStatus('idle')
    setErrorReason(null)
    setSeed('')
    setAnnouncement('')
    if (slug) clearPendingCoupon(slug)
  }, [slug])

  const rejectAtCheckout = useCallback(
    (reason: string) => {
      setPreview(null)
      setStatus('rejected')
      setErrorReason(reason as CouponReason)
      if (slug) clearPendingCoupon(slug)
    },
    [slug],
  )

  const setSeedCode = useCallback((code: string) => {
    setSeed(normalizeCouponInput(code))
  }, [])

  const applied = status === 'applied' && Boolean(preview?.applied)
  const activeCode = applied ? preview!.code : undefined

  return {
    status,
    preview,
    errorReason,
    seed,
    announcement,
    applied,
    activeCode,
    setSeedCode,
    applyPreview,
    runPreview,
    remove,
    clearOnOwned,
    rejectAtCheckout,
  }
}

/**
 * Product-analytics events for the creator coupon manager (MKTC.4 observability).
 * Fire-and-forget listener bus; no PII, no coupon notes or learner identity.
 */

export type CouponManagerTelemetryEventName =
  | 'coupon_manager_opened'
  | 'coupon_created'
  | 'coupon_share_link_copied'
  | 'coupon_paused'
  | 'coupon_archived'
  | 'coupon_redemptions_viewed'

export type CouponManagerTelemetryProps = {
  discountType?: 'percent' | 'fixed'
  target?: 'app' | 'public'
  status?: string
}

export type CouponManagerTelemetryEvent = {
  event: CouponManagerTelemetryEventName
  props: CouponManagerTelemetryProps
}

const EVENT_NAMES = new Set<CouponManagerTelemetryEventName>([
  'coupon_manager_opened',
  'coupon_created',
  'coupon_share_link_copied',
  'coupon_paused',
  'coupon_archived',
  'coupon_redemptions_viewed',
])

const ALLOWED_PROP_KEYS = new Set(['discountType', 'target', 'status'])

const FORBIDDEN_PROP_KEYS = new Set([
  'code',
  'couponId',
  'courseCode',
  'courseId',
  'userId',
  'userEmail',
  'note',
  'shareUrl',
  'publicShareUrl',
])

type Listener = (event: CouponManagerTelemetryEvent) => void
const listeners = new Set<Listener>()

export function isCouponManagerTelemetryOptedOut(): boolean {
  try {
    if (typeof navigator !== 'undefined') {
      const dnt = navigator.doNotTrack
      if (dnt === '1' || dnt === 'yes') return true
    }
  } catch {
    // ignore
  }
  try {
    if (typeof localStorage !== 'undefined' && localStorage) {
      if (localStorage.getItem('lextures.analytics.opt-out') === '1') return true
    }
  } catch {
    // ignore
  }
  return false
}

export function validateCouponManagerTelemetryEvent(
  event: string,
  props: Record<string, unknown>,
): CouponManagerTelemetryEvent | null {
  if (!EVENT_NAMES.has(event as CouponManagerTelemetryEventName)) return null
  for (const k of Object.keys(props)) {
    if (FORBIDDEN_PROP_KEYS.has(k)) return null
  }
  const cleaned: CouponManagerTelemetryProps = {}
  for (const [k, v] of Object.entries(props)) {
    if (!ALLOWED_PROP_KEYS.has(k) || v === undefined) continue
    ;(cleaned as Record<string, unknown>)[k] = v
  }
  return { event: event as CouponManagerTelemetryEventName, props: cleaned }
}

export function onCouponManagerTelemetry(listener: Listener): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function emitCouponManagerTelemetry(
  event: CouponManagerTelemetryEventName,
  props: CouponManagerTelemetryProps = {},
): void {
  if (isCouponManagerTelemetryOptedOut()) return
  const validated = validateCouponManagerTelemetryEvent(
    event,
    props as unknown as Record<string, unknown>,
  )
  if (!validated) return
  for (const listener of listeners) {
    try {
      listener(validated)
    } catch {
      // never block UI
    }
  }
}

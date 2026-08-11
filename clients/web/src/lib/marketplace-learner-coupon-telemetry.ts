/**
 * Product-analytics events for learner coupon entry (MKTC.5 observability).
 * Fire-and-forget; no PII, no raw coupon codes.
 */

export type LearnerCouponTelemetryEventName =
  | 'coupon_field_opened'
  | 'coupon_applied'
  | 'coupon_from_url'
  | 'coupon_removed'
  | 'coupon_checkout_started'
  | 'coupon_free_grant'

export type LearnerCouponTelemetryProps = {
  result?: string
  discounted?: boolean
  fromUrl?: boolean
}

export type LearnerCouponTelemetryEvent = {
  event: LearnerCouponTelemetryEventName
  props: LearnerCouponTelemetryProps
}

const EVENT_NAMES = new Set<LearnerCouponTelemetryEventName>([
  'coupon_field_opened',
  'coupon_applied',
  'coupon_from_url',
  'coupon_removed',
  'coupon_checkout_started',
  'coupon_free_grant',
])

const ALLOWED_PROP_KEYS = new Set(['result', 'discounted', 'fromUrl'])

const FORBIDDEN_PROP_KEYS = new Set([
  'code',
  'couponId',
  'couponCode',
  'courseCode',
  'courseId',
  'userId',
  'userEmail',
  'slug',
])

type Listener = (event: LearnerCouponTelemetryEvent) => void
const listeners = new Set<Listener>()

export function isLearnerCouponTelemetryOptedOut(): boolean {
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

export function validateLearnerCouponTelemetryEvent(
  event: string,
  props: Record<string, unknown>,
): LearnerCouponTelemetryEvent | null {
  if (!EVENT_NAMES.has(event as LearnerCouponTelemetryEventName)) return null
  for (const k of Object.keys(props)) {
    if (FORBIDDEN_PROP_KEYS.has(k)) return null
  }
  const cleaned: LearnerCouponTelemetryProps = {}
  for (const [k, v] of Object.entries(props)) {
    if (!ALLOWED_PROP_KEYS.has(k) || v === undefined) continue
    ;(cleaned as Record<string, unknown>)[k] = v
  }
  return { event: event as LearnerCouponTelemetryEventName, props: cleaned }
}

export function onLearnerCouponTelemetry(listener: Listener): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function emitLearnerCouponTelemetry(
  event: LearnerCouponTelemetryEventName,
  props: LearnerCouponTelemetryProps = {},
): void {
  if (isLearnerCouponTelemetryOptedOut()) return
  const validated = validateLearnerCouponTelemetryEvent(
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

/**
 * Course coupon creator API client + pure helpers (plan MKTC.4).
 * Components MUST call these helpers — never authorizedFetch directly.
 */

import { authorizedFetch } from './api'
import { stripeMinimumMinorUnits } from './currency-exponent'
import { readApiErrorMessage, readApiFieldErrors } from './errors'
import { formatMarketplacePrice } from './marketplace-price'
import type { components } from './generated/openapi-types'

export type CourseCoupon = components['schemas']['CourseCoupon']
export type CourseCouponSeats = components['schemas']['CourseCouponSeats']
export type CreateCourseCouponBody = components['schemas']['CreateCourseCouponBody']
export type UpdateCourseCouponBody = components['schemas']['UpdateCourseCouponBody']
export type CouponRedemptionRow = components['schemas']['CouponRedemptionRow']

export type CouponDiscountType = 'percent' | 'fixed'
export type CouponStatus = CourseCoupon['status']

export type CouponDraft = {
  discountType: CouponDiscountType
  percentOff?: number | null
  amountOffCents?: number | null
}

export type CouponPreview = {
  chargedCents: number
  discountCents: number
  free: boolean
  clampedToFree: boolean
}

/** Client error with HTTP status + optional field map for form binding. */
export class CourseCouponApiError extends Error {
  readonly status: number
  readonly fields: Record<string, string>
  readonly code?: string

  constructor(message: string, status: number, fields: Record<string, string> = {}, code?: string) {
    super(message)
    this.name = 'CourseCouponApiError'
    this.status = status
    this.fields = fields
    this.code = code
  }
}

const CODE_SHAPE = /^[A-Z0-9][A-Z0-9_-]{3,31}$/

/** Unambiguous alphabet — no O/0/I/1 (FR-8). */
const CODE_ALPHABET = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'

export function normalizeCouponCode(raw: string): string {
  return raw.replace(/\s+/g, '').toUpperCase()
}

export function isValidCouponCode(code: string): boolean {
  return CODE_SHAPE.test(code)
}

export function generateCouponCode(length = 8): string {
  const len = Math.min(32, Math.max(4, Math.floor(length) || 8))
  const bytes = new Uint8Array(len)
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    crypto.getRandomValues(bytes)
  } else {
    for (let i = 0; i < len; i++) bytes[i] = Math.floor(Math.random() * 256)
  }
  let out = ''
  for (let i = 0; i < len; i++) {
    out += CODE_ALPHABET[bytes[i]! % CODE_ALPHABET.length]
  }
  return out
}

/**
 * Mirror of server ApplyDiscount (half-up percent, fixed clamp, provider floor → free).
 */
export function previewCouponPrice(
  priceCents: number,
  currency: string,
  draft: CouponDraft,
): CouponPreview {
  const curr = (currency || 'usd').toLowerCase().trim() || 'usd'
  if (priceCents <= 0) {
    return { chargedCents: 0, discountCents: 0, free: true, clampedToFree: false }
  }

  let discount = 0
  if (draft.discountType === 'percent') {
    const pct = Number(draft.percentOff ?? 0)
    if (pct > 0) {
      discount = Math.floor((priceCents * pct) / 100 + 0.5)
    }
  } else {
    discount = Math.max(0, Math.floor(Number(draft.amountOffCents ?? 0)))
    if (discount > priceCents) discount = priceCents
  }
  if (discount < 0) discount = 0
  if (discount > priceCents) discount = priceCents

  let charged = priceCents - discount
  if (charged < 0) charged = 0

  let clampedToFree = false
  if (charged > 0) {
    const min = stripeMinimumMinorUnits(curr)
    if (charged < min) {
      clampedToFree = true
      discount = priceCents
      charged = 0
    }
  }

  return {
    chargedCents: charged,
    discountCents: discount,
    free: charged === 0,
    clampedToFree,
  }
}

export function describeDiscount(
  c: Pick<CourseCoupon, 'discountType' | 'percentOff' | 'amountOffCents' | 'currency'>,
  locale: string,
  labels: { percentOff: (p: number) => string; amountOff: (amount: string) => string },
): string {
  if (c.discountType === 'percent') {
    const p = c.percentOff ?? 0
    return labels.percentOff(p)
  }
  const cents = c.amountOffCents ?? 0
  const curr = c.currency || 'usd'
  const amount = formatMarketplacePrice(cents, curr, locale, '')
  return labels.amountOff(amount)
}

export type CouponWindowKind = 'always' | 'until' | 'from' | 'range'

export function describeCouponWindow(
  startsAt: string | null | undefined,
  endsAt: string | null | undefined,
  formatDate: (iso: string) => string,
  labels: {
    always: string
    until: (date: string) => string
    from: (date: string) => string
    range: (start: string, end: string) => string
  },
): { kind: CouponWindowKind; label: string } {
  const start = startsAt?.trim() || null
  const end = endsAt?.trim() || null
  if (!start && !end) return { kind: 'always', label: labels.always }
  if (!start && end) return { kind: 'until', label: labels.until(formatDate(end)) }
  if (start && !end) return { kind: 'from', label: labels.from(formatDate(start)) }
  return { kind: 'range', label: labels.range(formatDate(start!), formatDate(end!)) }
}

export type ClientCouponValidation = {
  code?: string
  percentOff?: string
  amountOff?: string
  endsAt?: string
  maxRedemptions?: string
  maxRedemptionsPerUser?: string
}

export function validateCouponDraft(input: {
  code: string
  discountType: CouponDiscountType
  percentOff: string
  amountOffMajor: string
  amountOffCents: number | null
  coursePriceCents: number
  startsAtLocal: string
  endsAtLocal: string
  maxRedemptions: string
  maxRedemptionsPerUser: string
  messages: {
    codeShape: string
    percentRange: string
    amountPositive: string
    amountExceedsPrice: string
    endBeforeStart: string
    maxRedemptionsPositive: string
    perUserRange: string
  }
}): ClientCouponValidation {
  const errors: ClientCouponValidation = {}
  const code = normalizeCouponCode(input.code)
  if (!isValidCouponCode(code)) {
    errors.code = input.messages.codeShape
  }

  if (input.discountType === 'percent') {
    const pct = Number.parseFloat(input.percentOff)
    if (!Number.isFinite(pct) || pct <= 0 || pct > 100) {
      errors.percentOff = input.messages.percentRange
    }
  } else {
    const cents = input.amountOffCents
    if (cents == null || cents <= 0) {
      errors.amountOff = input.messages.amountPositive
    } else if (cents > input.coursePriceCents) {
      errors.amountOff = input.messages.amountExceedsPrice
    }
  }

  if (input.startsAtLocal && input.endsAtLocal) {
    const s = new Date(input.startsAtLocal).getTime()
    const e = new Date(input.endsAtLocal).getTime()
    if (Number.isFinite(s) && Number.isFinite(e) && e <= s) {
      errors.endsAt = input.messages.endBeforeStart
    }
  }

  if (input.maxRedemptions.trim()) {
    const n = Number.parseInt(input.maxRedemptions, 10)
    if (!Number.isFinite(n) || n <= 0) {
      errors.maxRedemptions = input.messages.maxRedemptionsPositive
    }
  }

  if (input.maxRedemptionsPerUser.trim()) {
    const n = Number.parseInt(input.maxRedemptionsPerUser, 10)
    if (!Number.isFinite(n) || n < 1 || n > 100) {
      errors.maxRedemptionsPerUser = input.messages.perUserRange
    }
  }

  return errors
}

async function throwCouponError(res: Response, raw: unknown): Promise<never> {
  const message = readApiErrorMessage(raw)
  const fields: Record<string, string> = {}
  for (const f of readApiFieldErrors(raw)) {
    if (f.path && f.message) fields[f.path] = f.message
  }
  // Map common bare messages onto the code field for 409/shape errors.
  if (res.status === 409 && !fields.code) {
    fields.code = message
  }
  const code =
    raw && typeof raw === 'object' && 'error' in raw && typeof (raw as { error?: unknown }).error === 'string'
      ? (raw as { error: string }).error
      : undefined
  throw new CourseCouponApiError(message, res.status, fields, code)
}

export async function fetchCourseCoupons(
  courseCode: string,
  opts?: { includeArchived?: boolean },
): Promise<CourseCoupon[]> {
  const qs = opts?.includeArchived ? '?includeArchived=true' : ''
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/coupons${qs}`,
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) await throwCouponError(res, raw)
  const data = raw as { coupons?: CourseCoupon[] }
  return Array.isArray(data.coupons) ? data.coupons : []
}

export async function createCourseCoupon(
  courseCode: string,
  body: CreateCourseCouponBody,
): Promise<CourseCoupon> {
  const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/coupons`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) await throwCouponError(res, raw)
  const data = raw as { coupon?: CourseCoupon }
  if (!data.coupon) throw new CourseCouponApiError('Create response was missing coupon.', res.status)
  return data.coupon
}

export async function updateCourseCoupon(
  courseCode: string,
  couponId: string,
  body: UpdateCourseCouponBody,
): Promise<CourseCoupon> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/coupons/${encodeURIComponent(couponId)}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) await throwCouponError(res, raw)
  const data = raw as { coupon?: CourseCoupon }
  if (!data.coupon) throw new CourseCouponApiError('Update response was missing coupon.', res.status)
  return data.coupon
}

export async function archiveCourseCoupon(
  courseCode: string,
  couponId: string,
): Promise<CourseCoupon> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/coupons/${encodeURIComponent(couponId)}`,
    { method: 'DELETE' },
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) await throwCouponError(res, raw)
  const data = raw as { coupon?: CourseCoupon }
  if (!data.coupon) throw new CourseCouponApiError('Archive response was missing coupon.', res.status)
  return data.coupon
}

export async function fetchCouponRedemptions(
  courseCode: string,
  couponId: string,
  opts?: { cursor?: string; limit?: number },
): Promise<{ rows: CouponRedemptionRow[]; nextCursor: string }> {
  const params = new URLSearchParams()
  if (opts?.cursor) params.set('cursor', opts.cursor)
  if (opts?.limit != null) params.set('limit', String(opts.limit))
  const qs = params.toString() ? `?${params.toString()}` : ''
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/coupons/${encodeURIComponent(couponId)}/redemptions${qs}`,
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) await throwCouponError(res, raw)
  const data = raw as { redemptions?: CouponRedemptionRow[]; nextCursor?: string }
  return {
    rows: Array.isArray(data.redemptions) ? data.redemptions : [],
    nextCursor: typeof data.nextCursor === 'string' ? data.nextCursor : '',
  }
}

/** Per-coupon performance figures (plan MKTC.7). */
export type CouponSummaryRow = {
  couponId: string
  code: string
  redeemedCount: number
  refundedCount: number
  grossListCents: number
  discountCents: number
  netChargedCents: number
  currency: string
  firstRedeemedAt: string | null
  lastRedeemedAt: string | null
}

export type CouponSummaryResponse = {
  rows: CouponSummaryRow[]
  currency: string
}

export async function fetchCouponSummary(courseCode: string): Promise<CouponSummaryResponse> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/coupons/summary`,
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) await throwCouponError(res, raw)
  const data = raw as { rows?: CouponSummaryRow[]; currency?: string }
  return {
    rows: Array.isArray(data.rows) ? data.rows : [],
    currency: typeof data.currency === 'string' ? data.currency : 'usd',
  }
}

/**
 * Download redemptions CSV for a coupon (MKTC.7 FR-9).
 * Triggers a browser download from the streamed response body.
 */
export async function exportCouponRedemptionsCsv(
  courseCode: string,
  couponId: string,
  code: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/coupons/${encodeURIComponent(couponId)}/redemptions.csv`,
  )
  if (!res.ok) {
    const raw: unknown = await res.json().catch(() => ({}))
    await throwCouponError(res, raw)
  }
  const blob = await res.blob()
  const safeCode = normalizeCouponCode(code).replace(/[^A-Z0-9_-]/g, '') || couponId
  const filename = `coupon-${safeCode}-redemptions.csv`
  const url = URL.createObjectURL(blob)
  try {
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.rel = 'noopener'
    document.body.appendChild(a)
    a.click()
    a.remove()
  } finally {
    URL.revokeObjectURL(url)
  }
}

/** Compact performance line for the table (MKTC.7 FR-7). */
export function formatCouponPerformanceSummary(
  row: CouponSummaryRow | undefined,
  locale: string,
  labels: {
    empty: string
    claimed: (n: number) => string
    off: (amount: string) => string
    net: (amount: string) => string
  },
): { visible: string; sr: string } {
  if (!row || row.redeemedCount === 0) {
    return { visible: labels.empty, sr: labels.empty }
  }
  const currency = row.currency || 'usd'
  const off = formatMarketplacePrice(row.discountCents, currency, locale, '')
  const net = formatMarketplacePrice(row.netChargedCents, currency, locale, '')
  const claimed = labels.claimed(row.redeemedCount)
  const offPart = labels.off(off)
  const netPart = labels.net(net)
  const visible = `${claimed} · ${offPart} · ${netPart}`
  return { visible, sr: visible }
}

/** Async clipboard write; returns false when the browser blocks it. */
export async function copyTextToClipboard(text: string): Promise<boolean> {
  try {
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // fall through
  }
  return false
}

import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import { formatMoney } from './billing-api'

export type EarningsSummary = {
  pendingCents: number
  paidCents: number
  currency: string
  connectConfigured: boolean
}

export type LedgerEntry = {
  id: string
  entryType: string
  amountCents: number
  currency: string
  status: string
  courseId?: string
  affiliateCode?: string
  createdAt: string
}

export type AffiliateCode = {
  id: string
  code: string
  courseId?: string | null
  url: string
  clickCount: number
  conversions: number
  createdAt: string
}

export async function fetchCreatorEarnings(): Promise<EarningsSummary> {
  const res = await authorizedFetch('/api/v1/creator/earnings')
  if (res.status === 404) {
    return { pendingCents: 0, paidCents: 0, currency: 'usd', connectConfigured: false }
  }
  if (!res.ok) {
    throw new Error('Could not load earnings.')
  }
  return (await res.json()) as EarningsSummary
}

export async function fetchCreatorLedger(limit = 20): Promise<LedgerEntry[]> {
  const res = await authorizedFetch(`/api/v1/creator/earnings/ledger?limit=${limit}`)
  if (res.status === 404) {
    return []
  }
  if (!res.ok) {
    throw new Error('Could not load earnings ledger.')
  }
  const data = (await res.json()) as { entries?: LedgerEntry[] }
  return data.entries ?? []
}

export async function createAffiliateCode(courseId?: string): Promise<AffiliateCode> {
  const res = await authorizedFetch('/api/v1/creator/affiliate-codes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(courseId ? { courseId } : {}),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not create affiliate code.')
  }
  return (await res.json()) as AffiliateCode
}

export async function fetchAffiliateCodes(): Promise<AffiliateCode[]> {
  const res = await authorizedFetch('/api/v1/creator/affiliate-codes')
  if (res.status === 404) {
    return []
  }
  if (!res.ok) {
    throw new Error('Could not load affiliate codes.')
  }
  const data = (await res.json()) as { codes?: AffiliateCode[] }
  return data.codes ?? []
}

export async function startConnectOnboarding(): Promise<string> {
  const res = await authorizedFetch('/api/v1/creator/connect/onboarding', { method: 'POST' })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not start Connect onboarding.')
  }
  const data = (await res.json()) as { onboardingUrl?: string }
  if (!data.onboardingUrl) {
    throw new Error('Unexpected response from server.')
  }
  return data.onboardingUrl
}

export { formatMoney }

export const AFFILIATE_REF_COOKIE = 'lextures_ref'
export const AFFILIATE_REF_MAX_AGE_DAYS = 30

export function setAffiliateRefCookie(code: string): void {
  const maxAge = AFFILIATE_REF_MAX_AGE_DAYS * 24 * 60 * 60
  document.cookie = `${AFFILIATE_REF_COOKIE}=${encodeURIComponent(code)}; path=/; max-age=${maxAge}; SameSite=Lax`
}

export async function trackAffiliateClick(code: string): Promise<void> {
  try {
    await fetch(`/api/v1/affiliate/track-click?code=${encodeURIComponent(code)}`, { method: 'POST' })
  } catch {
    // best-effort
  }
}

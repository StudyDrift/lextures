import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type Entitlement = {
  id: string
  entitlementType: string
  courseId?: string
  amountPaidCents: number
  currency: string
  validFrom: string
  validUntil?: string
  status: string
}

export type CheckoutPayload = {
  courseId?: string
  plan?: 'monthly' | 'annual'
  promoCode?: string
  affiliateCode?: string
  successUrl: string
  cancelUrl: string
}

export async function fetchMyEntitlements(): Promise<Entitlement[]> {
  const res = await authorizedFetch('/api/v1/me/entitlements')
  if (res.status === 404) {
    return []
  }
  if (!res.ok) {
    throw new Error('Could not load entitlements.')
  }
  const data = (await res.json()) as { entitlements?: Entitlement[] }
  return data.entitlements ?? []
}

export async function startCheckout(payload: CheckoutPayload): Promise<string> {
  const res = await authorizedFetch('/api/v1/billing/checkout', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not start checkout.')
  }
  const data = (await res.json()) as { checkoutUrl?: string }
  if (!data.checkoutUrl) {
    throw new Error('Unexpected response from server.')
  }
  return data.checkoutUrl
}

export async function openBillingPortal(returnUrl?: string): Promise<string> {
  const q = returnUrl ? `?return_url=${encodeURIComponent(returnUrl)}` : ''
  const res = await authorizedFetch(`/api/v1/billing/portal${q}`)
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not open billing portal.')
  }
  const data = (await res.json()) as { portalUrl?: string }
  if (!data.portalUrl) {
    throw new Error('Unexpected response from server.')
  }
  return data.portalUrl
}

export async function checkEntitlement(userId: string, courseId: string): Promise<boolean> {
  const res = await authorizedFetch(
    `/api/v1/internal/entitlements/check?user_id=${encodeURIComponent(userId)}&course_id=${encodeURIComponent(courseId)}`,
  )
  if (!res.ok) {
    return false
  }
  const data = (await res.json()) as { entitled?: boolean }
  return data.entitled === true
}

export function formatMoney(cents: number, currency: string, locale?: string): string {
  return new Intl.NumberFormat(locale ?? undefined, {
    style: 'currency',
    currency: currency.toUpperCase(),
  }).format(cents / 100)
}

import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type TaxAddress = {
  country: string
  region?: string
  line1?: string
  city?: string
  postalCode?: string
}

export type TaxQuoteLine = {
  label: string
  amountCents: number
}

export type TaxQuote = {
  subtotalCents: number
  taxAmountCents: number
  totalCents: number
  currency: string
  taxRate?: number
  taxJurisdiction?: string
  taxType: string
  taxInclusive: boolean
  reverseCharge: boolean
  lines: TaxQuoteLine[]
  calculationId?: string
}

export type TaxIDValidation = {
  valid: boolean
  reverseCharge: boolean
  taxIdType?: string
  message?: string
}

export type OrgTaxSettings = {
  orgId: string
  enabled: boolean
  registeredJurisdictions: string[]
  defaultTaxCategory: string
  priceDisplay: 'inclusive' | 'exclusive'
  filingMode: string
  recordRetentionYears: number
  sellerName: string
  sellerAddress: string
  sellerTaxId: string
}

export type TaxReportRow = {
  jurisdiction: string
  taxType: string
  transactionCount: number
  taxCollectedCents: number
  subtotalCents: number
}

export async function fetchTaxQuote(payload: {
  courseId: string
  address: TaxAddress
  taxId?: string
  taxIdType?: string
}): Promise<TaxQuote> {
  const res = await authorizedFetch('/api/v1/checkout/quote', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      courseId: payload.courseId,
      address: payload.address,
      taxId: payload.taxId,
      taxIdType: payload.taxIdType,
    }),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not compute tax.')
  }
  return (await res.json()) as TaxQuote
}

export async function validateTaxID(payload: {
  courseId: string
  address: TaxAddress
  taxId: string
  taxIdType?: string
}): Promise<TaxIDValidation> {
  const res = await authorizedFetch('/api/v1/checkout/tax-id', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not validate tax ID.')
  }
  return (await res.json()) as TaxIDValidation
}

export async function fetchOrgTaxSettings(orgId: string): Promise<OrgTaxSettings> {
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/tax/settings`)
  if (!res.ok) {
    throw new Error('Could not load tax settings.')
  }
  return (await res.json()) as OrgTaxSettings
}

export async function saveOrgTaxSettings(orgId: string, settings: Partial<OrgTaxSettings>): Promise<OrgTaxSettings> {
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/tax/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not save tax settings.')
  }
  return (await res.json()) as OrgTaxSettings
}

export async function fetchTaxReport(orgId: string, period?: string, jurisdiction?: string): Promise<{
  period: string
  from: string
  to: string
  rows: TaxReportRow[]
}> {
  const params = new URLSearchParams()
  if (period) params.set('period', period)
  if (jurisdiction) params.set('jurisdiction', jurisdiction)
  const q = params.toString() ? `?${params}` : ''
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/tax/report${q}`)
  if (!res.ok) {
    throw new Error('Could not load tax report.')
  }
  return (await res.json()) as { period: string; from: string; to: string; rows: TaxReportRow[] }
}

export function invoiceDownloadUrl(invoiceId: string): string {
  const base = import.meta.env.VITE_API_URL ?? ''
  return `${base}/api/v1/invoices/${encodeURIComponent(invoiceId)}`
}
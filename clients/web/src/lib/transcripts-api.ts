import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type TranscriptDeliveryType = 'email' | 'mail' | 'pickup'
export type TranscriptUrgencyUnit = 'days' | 'business_days'
export type MailUrgency = 'standard' | 'rush'

export type TranscriptRequest = {
  id: string
  status: 'queued' | 'submitted' | 'failed'
  deliveryType: TranscriptDeliveryType
  deliveryEmail?: string
  deliveryAddress?: string
  urgencyDays?: number
  urgencyDaysMin?: number
  urgencyUnit?: TranscriptUrgencyUnit
  requestedAt: string
  submittedAt?: string
  errorMessage?: string
  webhookResponseCode?: number
}

export type TranscriptsConfig = {
  webhookUrl: string
  webhookSecret: string
  hasWebhookSecret: boolean
  pickupInstructions?: string
  officialEnabled: boolean
  ordersUiEnabled: boolean
  autoApprovalEnabled: boolean
  registrarConsoleEnabled: boolean
  consentRequired: boolean
  feesEnabled: boolean
  deliveryV2?: boolean
}

export type TranscriptOrderStatus =
  | 'draft'
  | 'pending_consent'
  | 'pending_payment'
  | 'in_review'
  | 'on_hold'
  | 'processing'
  | 'completed'
  | 'canceled'
  | 'rejected'
  | 'failed'

export type TranscriptHoldType = 'financial' | 'disciplinary' | 'registrar' | 'library' | 'other'

export type TranscriptHold = {
  id: string
  userId: string
  orgId?: string
  type: TranscriptHoldType
  reason?: string
  studentMessage: string
  externalId?: string
  placedBy?: string
  placedAt: string
  releasedBy?: string
  releasedAt?: string
  active: boolean
}

export type TranscriptOrderEvent = {
  id: string
  itemId?: string
  fromState?: string
  toState: string
  actorId?: string
  reason?: string
  createdAt: string
}

export type TranscriptOrderTransitionAction =
  | 'approve'
  | 'reject'
  | 'cancel'
  | 'complete'
  | 'hold'
  | 'release'

export type TranscriptsStudentConfig = {
  pickupInstructions?: string
  pickupAvailable: boolean
  officialEnabled: boolean
  ordersUiEnabled: boolean
  consentRequired: boolean
  feesEnabled: boolean
}

export type TranscriptPaymentStatus =
  | 'unpaid'
  | 'pending'
  | 'paid'
  | 'waived'
  | 'refunded'
  | 'partially_refunded'
  | 'free'

export type TranscriptQuoteLine = {
  code: string
  description: string
  amount: number
  quantity?: number
}

export type TranscriptQuote = {
  currency: string
  lines: TranscriptQuoteLine[]
  subtotal: number
  waiverAmount: number
  freeAllotmentApplied: boolean
  total: number
  requiresPayment: boolean
  paymentStatusIfZero?: TranscriptPaymentStatus
}

export type TranscriptFeeSchedule = {
  orgId: string
  currency: string
  baseFee: number
  rushFee: number
  perRecipientFee: number
  methodSurcharges: Record<string, number>
  freeAllotment: number
  allotmentPeriod: 'lifetime' | 'year' | 'term'
  updatedAt?: string
}

export type TranscriptWaiverCode = {
  id: string
  orgId: string
  code: string
  kind: 'full' | 'percent' | 'amount'
  value?: number
  maxUses?: number
  usedCount: number
  expiresAt?: string
  createdAt: string
}

export type TranscriptReceipt = {
  orderId: string
  issuedAt: string
  studentEmail?: string
  currency: string
  paymentStatus: string
  paymentRef?: string
  amountPaid: number
  amountPaidFormatted: string
  amountRefunded: number
  lines: TranscriptQuoteLine[]
  isRefund: boolean
}

export type TranscriptConsentSummary = {
  id: string
  signerId: string
  signerRole: 'student' | 'guardian'
  guardianRelationship?: string
  textVersion: string
  locale: string
  signatureMethod: 'typed' | 'drawn'
  payloadHash: string
  signedAt: string
  revokedAt?: string
  expiresAt?: string
}

export type TranscriptConsentPreview = {
  orderId: string
  status: string
  textVersion: string
  locale: string
  authorizationText: string
  scope: string
  purpose: string
  recipients: Array<{ id: string; type: string; name: string }>
  requiresConsent: boolean
  selfDisclosureOnly: boolean
  requiresGuardian: boolean
  isMinor: boolean
  consentRequired: boolean
  activeConsent?: TranscriptConsentSummary
}

export type TranscriptRecipientType =
  | 'institution'
  | 'application_service'
  | 'employer'
  | 'self'
  | 'other'

export type TranscriptDeliveryMethod =
  | 'electronic_pesc'
  | 'edi_speede'
  | 'electronic_pdf'
  | 'secure_link_email'
  | 'postal_mail'
  | 'api_peer'

export type TranscriptDeliveryReceipt = {
  id: string
  orderItemId: string
  adapter: string
  attemptNo: number
  status: 'queued' | 'sent' | 'delivered' | 'opened' | 'failed'
  responseCode?: number
  detail?: string
  idempotencyKey: string
  createdAt: string
  updatedAt: string
}

export type TranscriptShareLinkMeta = {
  token: string
  expiresAt: string
  maxDownloads: number
  downloadsRemaining: number
  openedAt?: string
  documentId: string
  orderItemId: string
  expired: boolean
  exhausted: boolean
  verifyToken?: string
  verificationUrl?: string
}

export type TranscriptOrderUrgency = 'standard' | 'rush'

export type TranscriptRecipient = {
  id: string
  orgId?: string
  type: TranscriptRecipientType
  name: string
  canonicalKey?: string
  capabilities: TranscriptDeliveryMethod[]
  email?: string
  address?: Record<string, unknown>
  verified: boolean
  active: boolean
  createdAt: string
}

export type TranscriptOrderItem = {
  id: string
  recipientId?: string
  documentId?: string
  deliveryMethod: TranscriptDeliveryMethod
  urgency: TranscriptOrderUrgency
  status: string
  createdAt: string
  recipient?: TranscriptRecipient
}

export type TranscriptOrder = {
  id: string
  status: string
  legacyRequestId?: string
  consentId?: string
  consent?: TranscriptConsentSummary
  requiresGuardian?: boolean
  paymentStatus?: TranscriptPaymentStatus | string
  paymentRef?: string
  totalAmount?: number
  currency?: string
  amountRefunded?: number
  createdAt: string
  submittedAt?: string
  items: TranscriptOrderItem[]
  onHold?: boolean
  holds?: Array<{ type: string; studentMessage: string; active: boolean }>
  studentMessage?: string
  rejectionReason?: string
  events?: TranscriptOrderEvent[]
  userId?: string
  userEmail?: string
  activeHoldCount?: number
}

export type AdHocRecipientPayload = {
  type?: TranscriptRecipientType
  name: string
  canonicalKey?: string
  capabilities?: TranscriptDeliveryMethod[]
  email?: string
  address?: Record<string, unknown>
}

export type CreateOrderItemPayload = {
  recipientId?: string
  adHocRecipient?: AdHocRecipientPayload
  documentId?: string
  deliveryMethod: TranscriptDeliveryMethod
  urgency?: TranscriptOrderUrgency
}

export type TranscriptDocumentVariant = 'official' | 'unofficial' | 'partial' | 'in_progress'

export type TranscriptDocument = {
  id: string
  variant: TranscriptDocumentVariant
  version: number
  schemaVersion: string
  templateVersion: string
  contentHash: string
  gpaCumulative?: number
  creditsEarned?: number
  generatedAt: string
  hasPdf: boolean
  hasXml: boolean
  verifyToken?: string
  verificationUrl?: string
  revokedAt?: string
  revokeReason?: string
  disclosePublicly?: boolean
  signed?: boolean
}

export type AcademicRecordCourse = {
  code: string
  title: string
  creditsAttempted: number
  creditsEarned: number
  grade: string
  qualityPoints?: number
  inProgress?: boolean
}

export type AcademicRecordTerm = {
  label: string
  startedOn?: string
  courses: AcademicRecordCourse[]
  termGpa?: number
  termCredits: number
}

export type AcademicRecord = {
  schemaVersion: string
  templateVersion: string
  variant: TranscriptDocumentVariant
  generatedAt: string
  student: { name: string; studentId?: string }
  institution: { name: string }
  terms: AcademicRecordTerm[]
  cumulative: {
    gpa?: number
    creditsAttempted: number
    creditsEarned: number
  }
  legend: Record<string, string>
  hasInProgress?: boolean
}

export type SubmitTranscriptRequestPayload = {
  deliveryType: TranscriptDeliveryType
  deliveryEmail?: string
  deliveryAddress?: string
  mailUrgency?: MailUrgency
  urgencyDays?: number
}

export async function fetchTranscriptRequests(): Promise<TranscriptRequest[]> {
  const res = await authorizedFetch('/api/v1/transcripts/requests')
  if (!res.ok) {
    throw new Error('Could not load transcript requests.')
  }
  const data = (await res.json()) as { requests?: TranscriptRequest[] }
  return data.requests ?? []
}

export async function fetchTranscriptsConfig(): Promise<TranscriptsStudentConfig> {
  const res = await authorizedFetch('/api/v1/transcripts/config')
  if (!res.ok) {
    throw new Error('Could not load transcript options.')
  }
  return (await res.json()) as TranscriptsStudentConfig
}

export async function submitTranscriptRequest(
  payload: SubmitTranscriptRequestPayload,
): Promise<TranscriptRequest> {
  const res = await authorizedFetch('/api/v1/transcripts/requests', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    const msg =
      res.status === 503
        ? 'Transcript requests are not configured yet. Contact your institution.'
        : readApiErrorMessage(raw) || 'Could not submit transcript request.'
    throw new Error(msg)
  }
  const data = (await res.json()) as { request?: TranscriptRequest }
  if (!data.request) {
    throw new Error('Unexpected response from server.')
  }
  return data.request
}

export async function fetchAdminTranscriptRequests(): Promise<TranscriptRequest[]> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/requests')
  if (!res.ok) {
    throw new Error('Could not load transcript delivery failures.')
  }
  const data = (await res.json()) as { requests?: TranscriptRequest[] }
  return data.requests ?? []
}

export async function fetchAdminTranscriptsConfig(): Promise<TranscriptsConfig> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/config')
  if (!res.ok) {
    throw new Error('Could not load transcripts configuration.')
  }
  return (await res.json()) as TranscriptsConfig
}

export async function searchTranscriptRecipients(params: {
  q?: string
  type?: TranscriptRecipientType | ''
}): Promise<TranscriptRecipient[]> {
  const qs = new URLSearchParams()
  if (params.q?.trim()) qs.set('q', params.q.trim())
  if (params.type) qs.set('type', params.type)
  const res = await authorizedFetch(`/api/v1/transcripts/recipients?${qs.toString()}`)
  if (!res.ok) {
    throw new Error('Could not search recipients.')
  }
  const data = (await res.json()) as { recipients?: TranscriptRecipient[] }
  return data.recipients ?? []
}

export async function fetchTranscriptOrders(): Promise<TranscriptOrder[]> {
  const res = await authorizedFetch('/api/v1/transcripts/orders')
  if (!res.ok) {
    throw new Error('Could not load transcript orders.')
  }
  const data = (await res.json()) as { orders?: TranscriptOrder[] }
  return data.orders ?? []
}

export async function createTranscriptOrder(items: CreateOrderItemPayload[]): Promise<TranscriptOrder> {
  const res = await authorizedFetch('/api/v1/transcripts/orders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ items }),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not create transcript order.')
  }
  const data = (await res.json()) as { order?: TranscriptOrder }
  if (!data.order) throw new Error('Unexpected response from server.')
  return data.order
}

export async function submitTranscriptOrder(orderId: string): Promise<TranscriptOrder> {
  const res = await authorizedFetch(`/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/submit`, {
    method: 'POST',
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not submit transcript order.')
  }
  const data = (await res.json()) as { order?: TranscriptOrder }
  if (!data.order) throw new Error('Unexpected response from server.')
  return data.order
}

export async function fetchTranscriptConsentPreview(
  orderId: string,
  locale?: string,
): Promise<TranscriptConsentPreview> {
  const qs = locale ? `?locale=${encodeURIComponent(locale)}` : ''
  const res = await authorizedFetch(
    `/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/consent/preview${qs}`,
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load consent preview.')
  }
  const data = (await res.json()) as { preview?: TranscriptConsentPreview }
  if (!data.preview) throw new Error('Unexpected response from server.')
  return data.preview
}

export async function signTranscriptConsent(
  orderId: string,
  payload: {
    method: 'typed' | 'drawn'
    signatureData: string
    agree: boolean
    locale?: string
    purpose?: string
  },
): Promise<{ consent: TranscriptConsentSummary; order: TranscriptOrder }> {
  const res = await authorizedFetch(`/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/consent`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not sign authorization.')
  }
  const data = (await res.json()) as { consent?: TranscriptConsentSummary; order?: TranscriptOrder }
  if (!data.consent || !data.order) throw new Error('Unexpected response from server.')
  return { consent: data.consent, order: data.order }
}

export async function revokeTranscriptConsent(
  orderId: string,
): Promise<{ consent: TranscriptConsentSummary; order: TranscriptOrder }> {
  const res = await authorizedFetch(
    `/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/consent/revoke`,
    { method: 'POST' },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not revoke authorization.')
  }
  const data = (await res.json()) as { consent?: TranscriptConsentSummary; order?: TranscriptOrder }
  if (!data.consent || !data.order) throw new Error('Unexpected response from server.')
  return { consent: data.consent, order: data.order }
}

export async function exportTranscriptConsent(
  orderId: string,
  format: 'json' | 'pdf' = 'json',
): Promise<Record<string, unknown> | Blob> {
  const res = await authorizedFetch(
    `/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/consent/export?format=${format}`,
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not export consent record.')
  }
  if (format === 'pdf') {
    return res.blob()
  }
  const data = (await res.json()) as { export?: Record<string, unknown> }
  if (!data.export) throw new Error('Unexpected response from server.')
  return data.export
}

export async function fetchAdminTranscriptRecipients(): Promise<TranscriptRecipient[]> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/recipients?includeInactive=true')
  if (!res.ok) {
    throw new Error('Could not load recipient directory.')
  }
  const data = (await res.json()) as { recipients?: TranscriptRecipient[] }
  return data.recipients ?? []
}

export async function createAdminTranscriptRecipient(payload: {
  type: TranscriptRecipientType
  name: string
  canonicalKey?: string
  capabilities: TranscriptDeliveryMethod[]
  email?: string
  verified?: boolean
  active?: boolean
}): Promise<TranscriptRecipient> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/recipients', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not create recipient.')
  }
  const data = (await res.json()) as { recipient?: TranscriptRecipient }
  if (!data.recipient) throw new Error('Unexpected response from server.')
  return data.recipient
}

export async function updateAdminTranscriptRecipient(
  id: string,
  payload: {
    type?: TranscriptRecipientType
    name?: string
    canonicalKey?: string
    capabilities?: TranscriptDeliveryMethod[]
    email?: string
    verified?: boolean
    active?: boolean
  },
): Promise<TranscriptRecipient> {
  const res = await authorizedFetch(`/api/v1/admin/transcripts/recipients/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not update recipient.')
  }
  const data = (await res.json()) as { recipient?: TranscriptRecipient }
  if (!data.recipient) throw new Error('Unexpected response from server.')
  return data.recipient
}

export async function fetchTranscriptItemReceipts(
  orderId: string,
  itemId: string,
): Promise<TranscriptDeliveryReceipt[]> {
  const res = await authorizedFetch(
    `/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/items/${encodeURIComponent(itemId)}/receipts`,
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load delivery receipts.')
  }
  const data = (await res.json()) as { receipts: TranscriptDeliveryReceipt[] }
  return data.receipts ?? []
}

export async function resendTranscriptOrderItem(orderId: string, itemId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/items/${encodeURIComponent(itemId)}/resend`,
    { method: 'POST' },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not resend transcript item.')
  }
}

export type TranscriptTimelineKind = 'order' | 'delivery'

export type TranscriptTimelineEntry = {
  id: string
  kind: TranscriptTimelineKind
  at: string
  status: string
  label: string
  itemId?: string
  adapter?: string
  attemptNo?: number
  detail?: string
  reason?: string
}

export type TranscriptOrderTimeline = {
  orderId: string
  status: TranscriptOrderStatus
  canCancel: boolean
  canResendItems: string[]
  entries: TranscriptTimelineEntry[]
  items: TranscriptOrderItem[]
}

export async function fetchTranscriptOrderTimeline(orderId: string): Promise<TranscriptOrderTimeline> {
  const res = await authorizedFetch(
    `/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/timeline`,
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load order timeline.')
  }
  const data = (await res.json()) as { timeline?: TranscriptOrderTimeline }
  if (!data.timeline) throw new Error('Unexpected response from server.')
  return data.timeline
}

export async function cancelTranscriptOrder(orderId: string): Promise<TranscriptOrder> {
  const res = await authorizedFetch(`/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/cancel`, {
    method: 'POST',
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not cancel transcript order.')
  }
  const data = (await res.json()) as { order?: TranscriptOrder }
  if (!data.order) throw new Error('Unexpected response from server.')
  return data.order
}

const apiBase = import.meta.env.VITE_API_URL ?? ''

export async function fetchTranscriptShareLink(token: string): Promise<TranscriptShareLinkMeta> {
  const res = await fetch(`${apiBase}/api/v1/r/t/${encodeURIComponent(token)}`)
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'This download link is invalid or unavailable.')
  }
  return (await res.json()) as TranscriptShareLinkMeta
}

export async function downloadTranscriptShareLink(token: string): Promise<void> {
  const res = await fetch(`${apiBase}/api/v1/r/t/${encodeURIComponent(token)}/download`)
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not download transcript.')
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'transcript.pdf'
  a.click()
  URL.revokeObjectURL(url)
}

export async function saveAdminTranscriptsConfig(payload: {
  webhookUrl: string
  webhookSecret?: string
  pickupInstructions?: string
  officialEnabled?: boolean
  ordersUiEnabled?: boolean
  autoApprovalEnabled?: boolean
  registrarConsoleEnabled?: boolean
  consentRequired?: boolean
  feesEnabled?: boolean
  deliveryV2?: boolean
}): Promise<TranscriptsConfig> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    throw new Error('Could not save transcripts configuration.')
  }
  return (await res.json()) as TranscriptsConfig
}

export async function fetchAdminTranscriptOrders(params?: {
  status?: string
  hold?: boolean
  q?: string
}): Promise<TranscriptOrder[]> {
  const qs = new URLSearchParams()
  if (params?.status) qs.set('status', params.status)
  if (params?.hold === true) qs.set('hold', 'true')
  if (params?.hold === false) qs.set('hold', 'false')
  if (params?.q?.trim()) qs.set('q', params.q.trim())
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  const res = await authorizedFetch(`/api/v1/admin/transcripts/orders${suffix}`)
  if (!res.ok) {
    throw new Error('Could not load fulfillment queue.')
  }
  const data = (await res.json()) as { orders?: TranscriptOrder[] }
  return data.orders ?? []
}

export async function transitionAdminTranscriptOrder(
  orderId: string,
  action: TranscriptOrderTransitionAction,
  reason?: string,
): Promise<TranscriptOrder> {
  const res = await authorizedFetch(
    `/api/v1/admin/transcripts/orders/${encodeURIComponent(orderId)}/transition`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, reason }),
    },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not update order.')
  }
  const data = (await res.json()) as { order?: TranscriptOrder }
  if (!data.order) throw new Error('Unexpected response from server.')
  return data.order
}

export async function fetchAdminTranscriptHolds(params?: {
  userId?: string
  active?: boolean
}): Promise<TranscriptHold[]> {
  const qs = new URLSearchParams()
  if (params?.userId) qs.set('userId', params.userId)
  if (params?.active === false) qs.set('active', 'false')
  else qs.set('active', 'true')
  const res = await authorizedFetch(`/api/v1/admin/transcripts/holds?${qs.toString()}`)
  if (!res.ok) {
    throw new Error('Could not load holds.')
  }
  const data = (await res.json()) as { holds?: TranscriptHold[] }
  return data.holds ?? []
}

export async function placeAdminTranscriptHold(payload: {
  userId: string
  type: TranscriptHoldType
  reason?: string
  studentMessage?: string
  externalId?: string
}): Promise<TranscriptHold> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/holds', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not place hold.')
  }
  const data = (await res.json()) as { hold?: TranscriptHold }
  if (!data.hold) throw new Error('Unexpected response from server.')
  return data.hold
}

export async function releaseAdminTranscriptHold(holdId: string): Promise<TranscriptHold> {
  const res = await authorizedFetch(
    `/api/v1/admin/transcripts/holds/${encodeURIComponent(holdId)}/release`,
    { method: 'POST' },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not release hold.')
  }
  const data = (await res.json()) as { hold?: TranscriptHold }
  if (!data.hold) throw new Error('Unexpected response from server.')
  return data.hold
}

export async function fetchTranscriptPreview(): Promise<{
  record: AcademicRecord
  contentHash: string
  variant: 'unofficial'
  persisted: false
}> {
  const res = await authorizedFetch('/api/v1/transcripts/preview')
  if (!res.ok) {
    throw new Error('Could not load transcript preview.')
  }
  return (await res.json()) as {
    record: AcademicRecord
    contentHash: string
    variant: 'unofficial'
    persisted: false
  }
}

export async function fetchTranscriptDocuments(): Promise<TranscriptDocument[]> {
  const res = await authorizedFetch('/api/v1/transcripts/documents')
  if (!res.ok) {
    throw new Error('Could not load issued transcripts.')
  }
  const data = (await res.json()) as { documents?: TranscriptDocument[] }
  return data.documents ?? []
}

export async function generateTranscriptDocument(payload: {
  variant: Exclude<TranscriptDocumentVariant, 'unofficial'>
  terms?: string[]
  format?: Array<'pdf' | 'xml'>
}): Promise<{ document: TranscriptDocument; record: AcademicRecord }> {
  const res = await authorizedFetch('/api/v1/transcripts/documents', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not generate transcript.')
  }
  return (await res.json()) as { document: TranscriptDocument; record: AcademicRecord }
}

export async function downloadTranscriptDocument(id: string, format: 'pdf' | 'xml'): Promise<Blob> {
  const res = await authorizedFetch(
    `/api/v1/transcripts/documents/${encodeURIComponent(id)}/download?format=${format}`,
  )
  if (!res.ok) {
    throw new Error('Could not download transcript.')
  }
  return res.blob()
}

export async function downloadTranscriptPreviewPDF(): Promise<Blob> {
  const res = await authorizedFetch('/api/v1/transcripts/preview?format=pdf')
  if (!res.ok) {
    throw new Error('Could not download unofficial PDF.')
  }
  return res.blob()
}

function triggerBlobDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export async function saveTranscriptDocumentDownload(id: string, format: 'pdf' | 'xml'): Promise<void> {
  const blob = await downloadTranscriptDocument(id, format)
  triggerBlobDownload(blob, format === 'pdf' ? 'transcript.pdf' : 'transcript.xml')
}

export async function setTranscriptDocumentDisclosure(
  id: string,
  disclosePublicly: boolean,
): Promise<TranscriptDocument> {
  const res = await authorizedFetch(`/api/v1/transcripts/documents/${encodeURIComponent(id)}/disclosure`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ disclosePublicly }),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not update disclosure.')
  }
  const data = (await res.json()) as { document?: TranscriptDocument }
  if (!data.document) throw new Error('Unexpected response from server.')
  return data.document
}

export async function revokeAdminTranscriptDocument(id: string, reason?: string): Promise<TranscriptDocument> {
  const res = await authorizedFetch(`/api/v1/admin/transcripts/documents/${encodeURIComponent(id)}/revoke`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ reason: reason ?? '' }),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not revoke document.')
  }
  const data = (await res.json()) as { document?: TranscriptDocument }
  if (!data.document) throw new Error('Unexpected response from server.')
  return data.document
}

export async function unrevokeAdminTranscriptDocument(id: string): Promise<TranscriptDocument> {
  const res = await authorizedFetch(`/api/v1/admin/transcripts/documents/${encodeURIComponent(id)}/unrevoke`, {
    method: 'POST',
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not restore document.')
  }
  const data = (await res.json()) as { document?: TranscriptDocument }
  if (!data.document) throw new Error('Unexpected response from server.')
  return data.document
}

export async function saveTranscriptPreviewPDF(): Promise<void> {
  const blob = await downloadTranscriptPreviewPDF()
  triggerBlobDownload(blob, 'transcript-unofficial.pdf')
}

export async function fetchTranscriptOrderQuote(
  orderId: string,
  waiverCode?: string,
): Promise<{ orderId: string; feesEnabled: boolean; paymentStatus: string; quote: TranscriptQuote }> {
  const qs = waiverCode?.trim() ? `?waiverCode=${encodeURIComponent(waiverCode.trim())}` : ''
  const res = await authorizedFetch(
    `/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/quote${qs}`,
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load quote.')
  }
  return (await res.json()) as {
    orderId: string
    feesEnabled: boolean
    paymentStatus: string
    quote: TranscriptQuote
  }
}

export async function checkoutTranscriptOrder(
  orderId: string,
  payload?: { waiverCode?: string; successUrl?: string; cancelUrl?: string },
): Promise<
  | { checkoutUrl: string; sessionId: string }
  | { waived: true; order: TranscriptOrder }
> {
  const res = await authorizedFetch(
    `/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/checkout`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload ?? {}),
    },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not start checkout.')
  }
  const data = (await res.json()) as {
    checkoutUrl?: string
    sessionId?: string
    waived?: boolean
    order?: TranscriptOrder
  }
  if (data.waived && data.order) {
    return { waived: true, order: data.order }
  }
  if (!data.checkoutUrl || !data.sessionId) {
    throw new Error('Unexpected response from server.')
  }
  return { checkoutUrl: data.checkoutUrl, sessionId: data.sessionId }
}

export async function fetchTranscriptOrderReceipt(
  orderId: string,
  format: 'json' | 'pdf' = 'json',
): Promise<TranscriptReceipt | Blob> {
  const res = await authorizedFetch(
    `/api/v1/transcripts/orders/${encodeURIComponent(orderId)}/receipt?format=${format}`,
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load receipt.')
  }
  if (format === 'pdf') return res.blob()
  return (await res.json()) as TranscriptReceipt
}

export async function fetchAdminTranscriptFees(): Promise<TranscriptFeeSchedule> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/fees')
  if (!res.ok) throw new Error('Could not load fee schedule.')
  return (await res.json()) as TranscriptFeeSchedule
}

export async function saveAdminTranscriptFees(payload: {
  currency: string
  baseFee: number
  rushFee: number
  perRecipientFee: number
  methodSurcharges?: Record<string, number>
  freeAllotment: number
  allotmentPeriod: 'lifetime' | 'year' | 'term'
}): Promise<TranscriptFeeSchedule> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/fees', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not save fee schedule.')
  }
  return (await res.json()) as TranscriptFeeSchedule
}

export async function fetchAdminTranscriptWaiverCodes(): Promise<TranscriptWaiverCode[]> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/waiver-codes')
  if (!res.ok) throw new Error('Could not load waiver codes.')
  const data = (await res.json()) as { waiverCodes?: TranscriptWaiverCode[] }
  return data.waiverCodes ?? []
}

export async function createAdminTranscriptWaiverCode(payload: {
  code: string
  kind: 'full' | 'percent' | 'amount'
  value?: number
  maxUses?: number
  expiresAt?: string
}): Promise<TranscriptWaiverCode> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/waiver-codes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not create waiver code.')
  }
  return (await res.json()) as TranscriptWaiverCode
}

export async function waiveAdminTranscriptOrder(
  orderId: string,
  reason?: string,
): Promise<TranscriptOrder> {
  const res = await authorizedFetch(
    `/api/v1/admin/transcripts/orders/${encodeURIComponent(orderId)}/waive`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason }),
    },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not waive order.')
  }
  const data = (await res.json()) as { order?: TranscriptOrder }
  if (!data.order) throw new Error('Unexpected response from server.')
  return data.order
}

export async function refundAdminTranscriptOrder(
  orderId: string,
  amountCents?: number,
): Promise<{ order: TranscriptOrder; refund: { refundId: string; amountCents: number } }> {
  const res = await authorizedFetch(
    `/api/v1/admin/transcripts/orders/${encodeURIComponent(orderId)}/refund`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(amountCents != null ? { amountCents } : {}),
    },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not refund order.')
  }
  const data = (await res.json()) as {
    order?: TranscriptOrder
    refund?: { refundId: string; amountCents: number }
  }
  if (!data.order || !data.refund) throw new Error('Unexpected response from server.')
  return { order: data.order, refund: data.refund }
}

export type TranscriptInboundStatus =
  | 'received'
  | 'quarantined'
  | 'parsed'
  | 'matched'
  | 'accepted'
  | 'rejected'
  | 'unmatched'

export type TranscriptInboundDocument = {
  id: string
  orgId: string
  channel: 'api_peer' | 'sftp' | 'email' | string
  sourceName?: string
  externalRef?: string
  format: 'pesc_xml' | 'pdf' | 'edi' | 'other' | string
  contentHash: string
  contentType?: string
  byteSize: number
  parsed?: unknown
  studentName?: string
  studentDob?: string
  studentRef?: string
  matchedUserId?: string
  matchConfidence?: number
  matchDetail?: unknown
  status: TranscriptInboundStatus
  needsManualReview: boolean
  reviewerId?: string
  rejectReason?: string
  quarantineReason?: string
  receivedAt: string
  processedAt?: string
}

export type TranscriptInboundEvent = {
  id: string
  eventType: string
  actorId?: string
  detail?: unknown
  createdAt: string
}

export type TranscriptInboundCourse = {
  code: string
  title: string
  creditsAttempted: number
  creditsEarned: number
  grade: string
  qualityPoints?: number
  inProgress?: boolean
  transfer?: boolean
}

export async function fetchAdminTranscriptInbound(params?: {
  status?: string
  q?: string
  orgId?: string
  limit?: number
}): Promise<TranscriptInboundDocument[]> {
  const qs = new URLSearchParams()
  if (params?.status) qs.set('status', params.status)
  if (params?.q) qs.set('q', params.q)
  if (params?.orgId) qs.set('orgId', params.orgId)
  if (params?.limit != null) qs.set('limit', String(params.limit))
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  const res = await authorizedFetch(`/api/v1/admin/transcripts/inbound${suffix}`)
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load inbound queue.')
  }
  const data = (await res.json()) as { documents?: TranscriptInboundDocument[] }
  return data.documents ?? []
}

export async function fetchAdminTranscriptInboundDetail(
  id: string,
): Promise<{ document: TranscriptInboundDocument; events: TranscriptInboundEvent[] }> {
  const res = await authorizedFetch(`/api/v1/admin/transcripts/inbound/${encodeURIComponent(id)}`)
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load inbound document.')
  }
  const data = (await res.json()) as {
    document?: TranscriptInboundDocument
    events?: TranscriptInboundEvent[]
  }
  if (!data.document) throw new Error('Unexpected response from server.')
  return { document: data.document, events: data.events ?? [] }
}

export async function fetchAdminTranscriptInboundCourses(
  id: string,
): Promise<{ courses: TranscriptInboundCourse[]; parsed?: unknown }> {
  const res = await authorizedFetch(
    `/api/v1/admin/transcripts/inbound/${encodeURIComponent(id)}/courses`,
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load courses.')
  }
  const data = (await res.json()) as { courses?: TranscriptInboundCourse[]; parsed?: unknown }
  return { courses: data.courses ?? [], parsed: data.parsed }
}

export async function matchAdminTranscriptInbound(
  id: string,
  userId: string,
): Promise<TranscriptInboundDocument> {
  const res = await authorizedFetch(
    `/api/v1/admin/transcripts/inbound/${encodeURIComponent(id)}/match`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId }),
    },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not match document.')
  }
  const data = (await res.json()) as { document?: TranscriptInboundDocument }
  if (!data.document) throw new Error('Unexpected response from server.')
  return data.document
}

export async function acceptAdminTranscriptInbound(id: string): Promise<TranscriptInboundDocument> {
  const res = await authorizedFetch(
    `/api/v1/admin/transcripts/inbound/${encodeURIComponent(id)}/accept`,
    { method: 'POST' },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not accept document.')
  }
  const data = (await res.json()) as { document?: TranscriptInboundDocument }
  if (!data.document) throw new Error('Unexpected response from server.')
  return data.document
}

export async function rejectAdminTranscriptInbound(
  id: string,
  reason: string,
): Promise<TranscriptInboundDocument> {
  const res = await authorizedFetch(
    `/api/v1/admin/transcripts/inbound/${encodeURIComponent(id)}/reject`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason }),
    },
  )
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not reject document.')
  }
  const data = (await res.json()) as { document?: TranscriptInboundDocument }
  if (!data.document) throw new Error('Unexpected response from server.')
  return data.document
}

export async function fetchMyTranscriptInbound(): Promise<TranscriptInboundDocument[]> {
  const res = await authorizedFetch('/api/v1/me/transcripts/inbound')
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load received transcripts.')
  }
  const data = (await res.json()) as { documents?: TranscriptInboundDocument[] }
  return data.documents ?? []
}

/** T12: panel visibility returned by analytics/health endpoints. */
export type TranscriptConsolePanels = {
  queue: boolean
  holds: boolean
  fees: boolean
  delivery: boolean
  recipients: boolean
  settings: boolean
  analytics: boolean
  finance: boolean
  export: boolean
}

export type TranscriptDashboardDaily = {
  day: string
  orders: number
  items: number
  delivered: number
  onHold: number
  rejected: number
  refunded: number
  netRevenueMinor: number
}

export type TranscriptDashboardSummary = {
  orgId: string
  from: string
  to: string
  orders: number
  items: number
  delivered: number
  onHold: number
  rejected: number
  refunded: number
  netRevenueMinor: number
  holdRate: number
  rejectionRate: number
  refundRate: number
  turnaround: {
    sampleSize: number
    avgHours: number
    p50Hours: number
    p90Hours: number
    p95Hours: number
  }
  methodMix: Array<{ method: string; count: number }>
  topDestinations: Array<{ recipientId?: string; recipientName: string; count: number }>
  daily: TranscriptDashboardDaily[]
  lastRefreshedAt?: string
  stale: boolean
  panels: TranscriptConsolePanels
  currency: string
}

export type TranscriptHealthSummary = {
  orgId: string
  backlogCount: number
  oldestPendingAgeHours: number
  oldestPendingOrderId?: string
  deliveryFailureRate: number
  deadLetterCount: number
  backlogAlert: boolean
  ageAlert: boolean
  failureAlert: boolean
  anyAlert: boolean
  thresholds: {
    backlogCount: number
    oldestPendingHours: number
    failureRateBps: number
  }
  panels: TranscriptConsolePanels
}

export type TranscriptDrillDownOrder = {
  id: string
  status: string
  userEmail?: string
  submittedAt?: string
  createdAt: string
  totalAmount?: number
  currency?: string
}

export type TranscriptDeliveryConfig = {
  deliveryV2: boolean
  webhookUrl?: string
  hasWebhookSecret?: boolean
  webhookSecret?: string
  adapters: string[]
}

function dashboardQuery(params?: { from?: string; to?: string; orgId?: string }): string {
  const qs = new URLSearchParams()
  if (params?.from) qs.set('from', params.from)
  if (params?.to) qs.set('to', params.to)
  if (params?.orgId) qs.set('orgId', params.orgId)
  const s = qs.toString()
  return s ? `?${s}` : ''
}

export async function fetchAdminTranscriptDashboard(params?: {
  from?: string
  to?: string
  orgId?: string
}): Promise<TranscriptDashboardSummary> {
  const res = await authorizedFetch(`/api/v1/admin/transcripts/dashboard${dashboardQuery(params)}`)
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load transcript analytics.')
  }
  return (await res.json()) as TranscriptDashboardSummary
}

export async function fetchAdminTranscriptHealth(params?: {
  orgId?: string
}): Promise<TranscriptHealthSummary> {
  const qs = params?.orgId ? `?orgId=${encodeURIComponent(params.orgId)}` : ''
  const res = await authorizedFetch(`/api/v1/admin/transcripts/health${qs}`)
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load transcript health.')
  }
  return (await res.json()) as TranscriptHealthSummary
}

export async function fetchAdminTranscriptDashboardDrilldown(params: {
  metric: string
  from?: string
  to?: string
}): Promise<TranscriptDrillDownOrder[]> {
  const qs = new URLSearchParams()
  qs.set('metric', params.metric)
  if (params.from) qs.set('from', params.from)
  if (params.to) qs.set('to', params.to)
  const res = await authorizedFetch(`/api/v1/admin/transcripts/dashboard/drilldown?${qs.toString()}`)
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load drill-down.')
  }
  const data = (await res.json()) as { orders?: TranscriptDrillDownOrder[] }
  return data.orders ?? []
}

export async function downloadAdminTranscriptAnalyticsExport(params?: {
  type?: 'dashboard' | 'summary'
  from?: string
  to?: string
}): Promise<void> {
  const qs = new URLSearchParams()
  qs.set('type', params?.type ?? 'dashboard')
  if (params?.from) qs.set('from', params.from)
  if (params?.to) qs.set('to', params.to)
  const res = await authorizedFetch(`/api/v1/admin/transcripts/reports/export?${qs.toString()}`)
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not export report.')
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `transcript-analytics-${params?.from ?? 'from'}_${params?.to ?? 'to'}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

export async function fetchAdminTranscriptDeliveryConfig(): Promise<TranscriptDeliveryConfig> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/delivery-config')
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not load delivery config.')
  }
  return (await res.json()) as TranscriptDeliveryConfig
}

export async function saveAdminTranscriptDeliveryConfig(payload: {
  deliveryV2?: boolean
  webhookUrl: string
  webhookSecret?: string
}): Promise<TranscriptDeliveryConfig> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/delivery-config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not save delivery config.')
  }
  return (await res.json()) as TranscriptDeliveryConfig
}
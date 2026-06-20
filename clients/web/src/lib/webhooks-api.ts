import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type WebhookSubscription = {
  id: string
  orgId: string
  label: string
  endpointUrl: string
  eventTypes: string[]
  active: boolean
  pausedAt: string | null
  tlsSkipVerify: boolean
  createdAt: string
  updatedAt: string
  status: 'active' | 'paused' | 'failing' | string
}

export type WebhookDelivery = {
  id: number
  eventType: string
  eventId: string
  attemptCount: number
  status: string
  lastHttpStatus?: number | null
  lastResponse?: string | null
  latencyMs?: number | null
  nextRetryAt?: string | null
  deliveredAt?: string | null
  createdAt: string
  test?: boolean
}

export type WebhookEventGroup = {
  domain: string
  types: string[]
}

export const WEBHOOK_EVENT_LABELS: Record<string, string> = {
  'grade.posted': 'Grade posted',
  'enrollment.created': 'Enrollment created',
  'assignment.submitted': 'Assignment submitted',
}

export function eventTypeLabel(eventType: string): string {
  return WEBHOOK_EVENT_LABELS[eventType] ?? eventType
}

export async function fetchWebhookEventTypes(orgId: string): Promise<{
  eventTypes: string[]
  groups: WebhookEventGroup[]
}> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/webhooks/event-types`,
  )
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as { eventTypes: string[]; groups: WebhookEventGroup[] }
}

export async function fetchWebhookSubscriptions(orgId: string): Promise<WebhookSubscription[]> {
  const res = await authorizedFetch(`/api/v1/admin/orgs/${encodeURIComponent(orgId)}/webhooks`)
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { subscriptions?: WebhookSubscription[] }
  return data.subscriptions ?? []
}

export async function createWebhookSubscription(
  orgId: string,
  body: { label: string; endpointUrl: string; eventTypes: string[]; tlsSkipVerify?: boolean },
): Promise<{ subscription: WebhookSubscription; signingKey: string }> {
  const res = await authorizedFetch(`/api/v1/admin/orgs/${encodeURIComponent(orgId)}/webhooks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as { subscription: WebhookSubscription; signingKey: string }
}

export async function updateWebhookSubscription(
  orgId: string,
  id: string,
  body: {
    label?: string
    endpointUrl?: string
    eventTypes?: string[]
    active?: boolean
    reactivate?: boolean
    tlsSkipVerify?: boolean
  },
): Promise<WebhookSubscription> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/webhooks/${encodeURIComponent(id)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { subscription: WebhookSubscription }
  return data.subscription
}

export async function deleteWebhookSubscription(orgId: string, id: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/webhooks/${encodeURIComponent(id)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export async function testWebhookSubscription(
  orgId: string,
  id: string,
  eventType: string,
): Promise<WebhookDelivery> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/webhooks/${encodeURIComponent(id)}/test`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ eventType }),
    },
  )
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { delivery: WebhookDelivery }
  return data.delivery
}

export async function fetchWebhookDeliveries(orgId: string, id: string): Promise<WebhookDelivery[]> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/webhooks/${encodeURIComponent(id)}/deliveries`,
  )
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { deliveries?: WebhookDelivery[] }
  return data.deliveries ?? []
}

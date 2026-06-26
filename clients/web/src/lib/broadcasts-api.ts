import { authorizedFetch } from './api'

export type BroadcastType = 'announcement' | 'emergency'
export type BroadcastStatus = 'draft' | 'queued' | 'sent'

export type Broadcast = {
  id: string
  orgId: string
  schoolId: string | null
  senderId: string
  type: BroadcastType
  subject: string
  body: string
  status: BroadcastStatus
  audience: unknown
  scheduledAt: string | null
  sentAt: string | null
  createdAt: string
}

export type CreateBroadcastPayload = {
  type?: BroadcastType
  schoolId?: string
  subject: string
  body: string
  scheduledAt?: string
  audience?: Record<string, unknown>
}

export type DeliveryReport = {
  broadcastId: string
  totalRecipients: number
  acknowledged: number
  unacknowledged: Array<{ userId: string; email: string; displayName: string | null }>
}

export async function listOrgBroadcasts(orgId: string): Promise<Broadcast[]> {
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/broadcasts`)
  if (!res.ok) return []
  const data = (await res.json()) as { broadcasts: Broadcast[] }
  return data.broadcasts ?? []
}

export async function createBroadcast(
  orgId: string,
  payload: CreateBroadcastPayload,
): Promise<Broadcast> {
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/broadcasts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? 'Failed to send broadcast')
  }
  const data = (await res.json()) as { broadcast: Broadcast }
  return data.broadcast
}

export async function getBroadcastDeliveryReport(
  orgId: string,
  broadcastId: string,
): Promise<DeliveryReport | null> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/broadcasts/${encodeURIComponent(
      broadcastId,
    )}/delivery-report`,
  )
  if (!res.ok) return null
  return (await res.json()) as DeliveryReport
}

export async function acknowledgeBroadcast(broadcastId: string): Promise<void> {
  await authorizedFetch(
    `/api/v1/broadcasts/${encodeURIComponent(broadcastId)}/acknowledge`,
    { method: 'POST' },
  )
}

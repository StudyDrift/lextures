import { authorizedFetch } from './api'

export type BehaviorCategory = {
  id: string
  orgId: string
  name: string
  type: 'positive' | 'negative'
  color?: string | null
  active: boolean
}

export type PBISAward = {
  id: string
  studentId: string
  awardedBy: string
  categoryId: string
  categoryName: string
  orgId: string
  points: number
  note?: string | null
  awardedAt: string
}

export type BehaviorReferral = {
  id: string
  studentId: string
  filedBy: string
  orgId: string
  schoolId?: string | null
  categoryId: string
  categoryName: string
  incidentAt: string
  location?: string | null
  description?: string
  response?: string | null
  createdAt: string
}

export type StudentBehaviorResponse = {
  studentId: string
  totalPoints: number
  awards: PBISAward[]
  referrals: BehaviorReferral[]
}

export type BehaviorDashboardResponse = {
  weekStart: string
  totalPoints: number
  totalReferrals: number
  pointsByCategory: { categoryId: string; categoryName: string; points: number }[]
  referralsByCategory: { categoryId: string; categoryName: string; count: number }[]
}

export type AwardInput = {
  studentId: string
  categoryId: string
  points?: number
  note?: string | null
}

export async function listBehaviorCategories(orgId: string): Promise<{ categories: BehaviorCategory[] }> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/behavior/categories`,
  )
  if (!res.ok) throw new Error(`Failed to load behavior categories (${res.status})`)
  return (await res.json()) as { categories: BehaviorCategory[] }
}

export async function seedDefaultBehaviorCategories(
  orgId: string,
): Promise<{ categories: BehaviorCategory[] }> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/behavior/categories`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ seedDefaults: true }),
    },
  )
  if (!res.ok) throw new Error(`Failed to seed categories (${res.status})`)
  return (await res.json()) as { categories: BehaviorCategory[] }
}

export async function awardPBISPoints(
  awards: AwardInput[],
): Promise<{ saved: number; awards: PBISAward[]; message: string }> {
  const res = await authorizedFetch('/api/v1/pbis/awards', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ awards }),
  })
  if (!res.ok) {
    const body = (await res.json().catch(() => ({ error: {} }))) as {
      error?: { message?: string }
    }
    throw new Error(body.error?.message ?? `Failed to award points (${res.status})`)
  }
  return (await res.json()) as { saved: number; awards: PBISAward[]; message: string }
}

export async function fileBehaviorReferral(payload: {
  studentId: string
  categoryId: string
  schoolId?: string | null
  incidentAt?: string
  location?: string | null
  description: string
  response?: string | null
}): Promise<BehaviorReferral> {
  const res = await authorizedFetch('/api/v1/behavior/referrals', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const body = (await res.json().catch(() => ({ error: {} }))) as {
      error?: { message?: string }
    }
    throw new Error(body.error?.message ?? `Failed to file referral (${res.status})`)
  }
  return (await res.json()) as BehaviorReferral
}

export async function fetchBehaviorDashboard(
  orgId: string,
  weekStart?: string,
): Promise<BehaviorDashboardResponse> {
  const params = weekStart ? `?weekStart=${encodeURIComponent(weekStart)}` : ''
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/behavior/dashboard${params}`,
  )
  if (!res.ok) throw new Error(`Failed to load behavior dashboard (${res.status})`)
  return (await res.json()) as BehaviorDashboardResponse
}

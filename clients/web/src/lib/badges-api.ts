import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import type { LinkedInCertParams } from './linkedin-share'

export type BadgeDefinition = {
  id: string
  courseId: string
  outcomeId?: string
  subOutcomeId?: string
  slug: string
  name: string
  description: string
  criteriaNarrative: string
  tags: string[]
  autoAward: boolean
  imageKey?: string
  createdAt: string
  updatedAt: string
}

export type AwardedBadge = {
  id: string
  definitionId: string
  recipientId: string
  name?: string
  slug?: string
  description?: string
  criteriaNarrative?: string
  courseId?: string
  awardSource: string
  shareSlug: string
  isPublic: boolean
  revoked: boolean
  issuedAt: string
  imageKey?: string
}

export type BadgeProfile = {
  handle: string
  pagePublic: boolean
  searchIndexable: boolean
  hideRealName: boolean
  displayNameOverride?: string
  publicUrl: string
  handleChangeCount30d: number
}

export type PublicBadge = {
  id: string
  slug: string
  name: string
  description: string
  tags: string[]
  issuedAt: string
  shareSlug: string
  verifyUrl: string
  courseTitle?: string
  imageKey?: string
  recipientDisplayName?: string
  issuerName?: string
  criteriaNarrative?: string
  searchIndexable?: boolean
}

export type BadgeVerifyResponse = {
  verified: boolean
  revoked: boolean
  issuerDid: string
  credential: Record<string, unknown>
  checkedAt: string
  title?: string
  status: string
}

export type HandleAvailability = {
  handle: string
  available: boolean
  valid: boolean
  reason?: string
}

async function parseJson<T>(res: Response): Promise<T> {
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as T
}

export async function fetchMyBadges(): Promise<{ badges: AwardedBadge[] }> {
  const res = await authorizedFetch('/api/v1/me/competency-badges')
  return parseJson(res)
}

export async function patchMyBadge(awardedId: string, isPublic: boolean): Promise<AwardedBadge> {
  const res = await authorizedFetch(`/api/v1/me/competency-badges/${awardedId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ isPublic }),
  })
  return parseJson(res)
}

export async function fetchBadgeProfile(): Promise<BadgeProfile> {
  const res = await authorizedFetch('/api/v1/me/badge-profile')
  return parseJson(res)
}

export async function patchBadgeProfile(body: {
  handle?: string
  pagePublic?: boolean
  searchIndexable?: boolean
  displayNameOverride?: string
  hideRealName?: boolean
}): Promise<BadgeProfile> {
  const res = await authorizedFetch('/api/v1/me/badge-profile', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJson(res)
}

export async function checkBadgeHandleAvailable(handle: string): Promise<HandleAvailability> {
  const res = await authorizedFetch(`/api/v1/badge-handle-available?handle=${encodeURIComponent(handle)}`)
  return parseJson(res)
}

export async function listCourseBadgeDefinitions(courseId: string): Promise<{ definitions: BadgeDefinition[] }> {
  const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseId)}/badge-definitions`)
  return parseJson(res)
}

export async function createBadgeDefinition(
  courseId: string,
  body: {
    name: string
    slug?: string
    description?: string
    criteriaNarrative?: string
    outcomeId?: string
    autoAward?: boolean
    tags?: string[]
  },
): Promise<BadgeDefinition> {
  const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseId)}/badge-definitions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJson(res)
}

export async function awardBadge(
  definitionId: string,
  recipientIds: string[],
): Promise<{ awarded: AwardedBadge[]; skipped: unknown[] }> {
  const res = await authorizedFetch(`/api/v1/badge-definitions/${definitionId}/award`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ recipientIds }),
  })
  return parseJson(res)
}

export async function revokeBadge(awardedId: string, reason: string): Promise<AwardedBadge> {
  const res = await authorizedFetch(`/api/v1/badges/${awardedId}/revoke`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ reason }),
  })
  return parseJson(res)
}

export async function fetchBadgeCandidates(
  definitionId: string,
): Promise<{ candidates: { userId: string; displayName: string; alreadyAwarded: boolean; masteryReached: boolean }[] }> {
  const res = await authorizedFetch(`/api/v1/badge-definitions/${definitionId}/candidates`)
  return parseJson(res)
}

export async function fetchBadgeLinkedInParams(
  awardedId: string,
): Promise<LinkedInCertParams & { url: string }> {
  const res = await authorizedFetch(`/api/v1/badges/${awardedId}/linkedin-params`)
  return parseJson(res)
}

export async function fetchBadgeExportUrl(awardedId: string): Promise<{ downloadUrl: string; expiresAt: string }> {
  const res = await authorizedFetch(`/api/v1/badges/${awardedId}/badge-export`)
  return parseJson(res)
}

/** Unauthenticated public backpack list. */
export async function fetchPublicBadges(handle: string): Promise<{
  handle: string
  displayName: string
  pagePublic: boolean
  searchIndexable?: boolean
  badges: PublicBadge[]
  status: string
  redirectTo?: string
}> {
  const res = await fetch(`/api/v1/public/badges/${encodeURIComponent(handle)}`)
  return parseJson(res)
}

/** Unauthenticated single public badge. */
export async function fetchPublicBadge(handle: string, badgeSlug: string): Promise<PublicBadge & { redirectTo?: string }> {
  const res = await fetch(`/api/v1/public/badges/${encodeURIComponent(handle)}/${encodeURIComponent(badgeSlug)}`)
  return parseJson(res)
}

/** Unauthenticated verify. */
export async function verifyBadge(shareSlug: string): Promise<BadgeVerifyResponse> {
  const res = await fetch(`/api/v1/badges/verify/${encodeURIComponent(shareSlug)}`)
  return parseJson(res)
}

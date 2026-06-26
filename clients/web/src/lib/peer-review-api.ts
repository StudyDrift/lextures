import { authorizedFetch } from './api'

function enc(s: string): string {
  return encodeURIComponent(s)
}

export type PeerReviewAnonymity = 'double_blind' | 'reviewer_anon' | 'named'
export type PeerReviewGradeMode = 'none' | 'score_only' | 'weighted_blend'
export type PeerReviewAggregation = 'mean' | 'median' | 'trimmed'

export type PeerReviewConfig = {
  id: string
  assignmentId: string
  reviewsPerReviewer: number
  anonymity: PeerReviewAnonymity
  opensAt?: string
  closesAt?: string
  gradeMode: PeerReviewGradeMode
  blendWeight: number
  aggregation: PeerReviewAggregation
  excludeSameGroup: boolean
}

export type PeerReviewAllocation = {
  id: string
  configId: string
  assignmentId: string
  courseId: string
  courseCode: string
  targetSubmissionId: string
  status: 'assigned' | 'in_progress' | 'submitted' | 'expired'
  assignedAt: string
  anonymity: PeerReviewAnonymity
  targetLabel?: string
  targetUserId?: string
}

export type PeerReviewReceived = {
  id: string
  score?: number
  comments?: string
  submittedAt: string
  reviewerLabel?: string
}

export type PeerReviewSummary = {
  config: PeerReviewConfig
  totalAllocations: number
  completedReviews: number
  incompleteReviewers: string[]
  outlierReviewers: string[]
  submissions: {
    submissionId: string
    studentUserId: string
    peerAggregate?: number
    reviewCount: number
  }[]
}

export async function putPeerReviewConfig(
  courseCode: string,
  itemId: string,
  body: Partial<PeerReviewConfig>,
): Promise<PeerReviewConfig> {
  const res = await authorizedFetch(
    `/api/v1/courses/${enc(courseCode)}/assignments/${enc(itemId)}/peer-review`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  if (!res.ok) throw new Error(await res.text())
  return (await res.json()) as PeerReviewConfig
}

export async function postPeerReviewAllocate(courseCode: string, itemId: string): Promise<{ allocationsCreated: number }> {
  const res = await authorizedFetch(
    `/api/v1/courses/${enc(courseCode)}/assignments/${enc(itemId)}/peer-review/allocate`,
    { method: 'POST' },
  )
  if (!res.ok) throw new Error(await res.text())
  return (await res.json()) as { allocationsCreated: number }
}

export async function fetchPeerReviewAssigned(): Promise<PeerReviewAllocation[]> {
  const res = await authorizedFetch('/api/v1/peer-review/assigned')
  if (!res.ok) throw new Error(await res.text())
  const raw = (await res.json()) as { allocations?: PeerReviewAllocation[] }
  return raw.allocations ?? []
}

export async function postPeerReviewSubmit(
  allocationId: string,
  body: { score?: number; rubricScores?: Record<string, number>; comments?: string },
): Promise<void> {
  const res = await authorizedFetch(`/api/v1/peer-review/allocations/${enc(allocationId)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await res.text())
}

export async function fetchPeerReviewSummary(courseCode: string, itemId: string): Promise<PeerReviewSummary> {
  const res = await authorizedFetch(
    `/api/v1/courses/${enc(courseCode)}/assignments/${enc(itemId)}/peer-review/summary`,
  )
  if (!res.ok) throw new Error(await res.text())
  return (await res.json()) as PeerReviewSummary
}

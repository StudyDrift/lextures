import { getAccessToken } from './auth'

const apiBase = import.meta.env.VITE_API_URL ?? ''

export type BoardLayout =
  | 'wall'
  | 'stream'
  | 'grid'
  | 'columns'
  | 'canvas'
  | 'timeline'
  | 'map'

export const BOARD_LAYOUTS: BoardLayout[] = [
  'wall',
  'stream',
  'grid',
  'columns',
  'canvas',
  'timeline',
  'map',
]

export type BoardSettings = {
  mapCenter?: { lat: number; lng: number }
  mapZoom?: number
  [key: string]: unknown
}

export type BoardReactionMode = 'none' | 'like' | 'vote' | 'star' | 'grade'

export type BoardVisibility = 'course' | 'section' | 'group' | 'invite' | 'link' | 'public'
export type BoardAttribution = 'named' | 'anon_to_peers' | 'anonymous'
export type BoardMemberRole = 'owner' | 'editor' | 'contributor' | 'viewer'
export type BoardShareCapability = 'view' | 'contribute'
export type BoardModerationMode = 'open' | 'approval'
export type BoardFilterAction = 'block' | 'flag'
export type BoardPostStatus = 'approved' | 'pending' | 'rejected'
export type BoardReportKind = 'user' | 'filter' | 'av_blocked'
export type BoardReportStatus = 'open' | 'resolved' | 'dismissed'

export type BoardCapabilities = {
  canView: boolean
  canPost: boolean
  canInteract: boolean
  canArrange: boolean
  canManage: boolean
}

export type BoardMyReaction = {
  kind: string
  value?: number | null
}

export type Board = {
  id: string
  courseId: string
  title: string
  description: string
  slug: string
  archived: boolean
  layout: BoardLayout
  layoutLocked: boolean
  settings: BoardSettings
  reactionMode: BoardReactionMode
  assignmentId: string | null
  visibility: BoardVisibility
  visibilityTarget: string | null
  attribution: BoardAttribution
  canPost: boolean
  canInteract: boolean
  canArrange: boolean
  moderationMode: BoardModerationMode
  filterAction: BoardFilterAction
  locked: boolean
  frozenUntil: string | null
  capabilities?: BoardCapabilities
  externalSharingAllowed?: boolean
  minorModerationFloor?: boolean
  createdBy: string | null
  createdAt: string
  updatedAt: string
}

export type BoardOrgPolicies = {
  orgId: string
  externalSharing: boolean
  minorModerationFloor: boolean
  defaultAttribution: BoardAttribution
  boardCapPerCourse: number | null
  updatedAt?: string
}

export type BoardAdminOverview = {
  boardCount: number
  activeBoardCount: number
  coursesWithBoards: number
  coursesFeatureEnabled: number
  storageBytes: number
  topContentTypes: { contentType: string; count: number }[]
  activeWindowDays: number
}

export type BoardContributorStat = {
  userId: string
  postCount: number
  commentCount: number
  reactionCount: number
  contributionTotal: number
}

export type BoardDailyAnalytics = {
  boardId: string
  day: string
  cardCount: number
  contributorCount: number
  reactionCount: number
  commentCount: number
}

export type BoardAnalyticsSummary = {
  boardId: string
  cardCount: number
  uniqueContributors: number
  reactionCount: number
  commentCount: number
  lastActivityAt?: string
  contributors: BoardContributorStat[]
  daily: BoardDailyAnalytics[]
}

export type BoardReport = {
  id: string
  boardId: string
  postId?: string
  commentId?: string
  reporterId?: string
  reason: string
  kind: BoardReportKind
  status: BoardReportStatus
  createdAt: string
  resolvedAt?: string
  resolvedBy?: string
}

export type BoardModerationQueue = {
  pending: BoardPost[]
  reports: BoardReport[]
  flagged: BoardReport[]
  minorsFloor: boolean
}

export type BoardModerationLogEntry = {
  id: number
  boardId: string
  actorId?: string
  action: string
  targetType: string
  targetId?: string
  reason: string
  createdAt: string
}

export type BoardMember = {
  boardId: string
  userId: string
  role: BoardMemberRole
  createdAt: string
}

export type BoardShare = {
  id: string
  boardId: string
  capability: BoardShareCapability
  hasPassword: boolean
  expiresAt: string | null
  revokedAt: string | null
  createdBy?: string
  createdAt: string
  token?: string
  url?: string
}

export type BoardSection = {
  id: string
  boardId: string
  title: string
  sortIndex: number
  createdAt: string
}

export type BoardSortMode = 'newest' | 'oldest' | 'author' | 'mostReacted'

export type BoardContentType = 'text' | 'image' | 'file' | 'link' | 'video' | 'audio' | 'drawing'

export type BoardAttachment = {
  id: string
  url: string | null
  fileName: string
  mimeType: string
  sizeBytes: number
  altText: string
  scanStatus: 'pending' | 'clean' | 'blocked'
}

export type BoardLinkPreview = {
  title?: string
  description?: string
  image?: string
  siteName?: string
  fetchedAt?: string
}

export type BoardPostBody = {
  html?: string
  text?: string
}

export type BoardPostPosition = { x: number; y: number; w: number; h: number }

export type BoardPost = {
  id: string
  boardId: string
  authorId: string | null
  guestDisplayName?: string
  contentType: BoardContentType
  title: string
  body?: BoardPostBody
  linkUrl?: string
  linkPreview?: BoardLinkPreview
  drawingData?: unknown
  attachment?: BoardAttachment
  sectionId?: string
  sortIndex: number
  position?: BoardPostPosition
  eventDate?: string
  lat?: number
  lng?: number
  status?: BoardPostStatus
  hidden?: boolean
  removed?: boolean
  reactionCount?: number
  myReaction?: BoardMyReaction | null
  avgStars?: number
  commentCount?: number
  grade?: number
  createdAt: string
  updatedAt: string
}

export type BoardComment = {
  id: string
  postId: string
  parentId?: string | null
  authorId: string | null
  body: BoardPostBody
  hidden: boolean
  createdAt: string
  updatedAt: string
}

export type BoardReactionResult = {
  active: boolean
  removed?: boolean
  reactionCount?: number
  myReaction?: BoardMyReaction | null
  avgStars?: number
  commentCount?: number
  grade?: number
}

export type ArrangeBoardPostInput = {
  sectionId?: string
  sortIndex?: number
  position?: BoardPostPosition
  eventDate?: string | null
  lat?: number
  lng?: number
  clearGeo?: boolean
}

export type CreateBoardPostInput = {
  contentType: BoardContentType
  title?: string
  body?: BoardPostBody | string
  linkUrl?: string
  drawingData?: unknown
  attachmentId?: string
}

async function authHeaders(json = true): Promise<Record<string, string>> {
  const tok = getAccessToken()
  const headers: Record<string, string> = {}
  if (json) headers['Content-Type'] = 'application/json'
  if (tok) headers.Authorization = `Bearer ${tok}`
  return headers
}

function absoluteUrl(pathOrUrl: string | null | undefined): string | null {
  if (!pathOrUrl) return null
  if (/^https?:\/\//i.test(pathOrUrl)) return pathOrUrl
  return `${apiBase}${pathOrUrl}`
}

export async function listBoards(
  courseCode: string,
  opts?: { includeArchived?: boolean },
): Promise<Board[]> {
  const qs = opts?.includeArchived ? '?includeArchived=true' : ''
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards${qs}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`listBoards failed (${res.status})`)
  const body = (await res.json()) as { boards: Board[] }
  return (body.boards ?? []).map(normalizeBoard)
}

export type BoardTemplateScope = 'builtin' | 'course' | 'org'

export type BoardTemplate = {
  id: string
  scope: BoardTemplateScope
  courseId: string | null
  orgId: string | null
  title: string
  description: string
  tags: string[]
  definition: Record<string, unknown>
  createdBy: string | null
  createdAt: string
}

export type BoardCopyMode = 'structure' | 'full'

export type BoardCopyJob = {
  id: string
  sourceBoardId: string
  mode: BoardCopyMode
  title: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  progress: number
  resultBoardId: string | null
  error: string
  createdAt: string
  updatedAt: string
}

export type CreateBoardResult =
  | { kind: 'board'; board: Board }
  | { kind: 'job'; job: BoardCopyJob }

export async function createBoard(
  courseCode: string,
  title: string,
  description?: string,
): Promise<Board> {
  const res = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ title, description: description ?? '' }),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createBoard failed (${res.status}): ${txt}`)
  }
  return normalizeBoard((await res.json()) as Board)
}

export async function listBoardTemplates(opts?: {
  scope?: BoardTemplateScope | ''
  courseCode?: string
  q?: string
  locale?: string
}): Promise<BoardTemplate[]> {
  const qs = new URLSearchParams()
  if (opts?.scope) qs.set('scope', opts.scope)
  if (opts?.courseCode) qs.set('courseCode', opts.courseCode)
  if (opts?.q) qs.set('q', opts.q)
  if (opts?.locale) qs.set('locale', opts.locale)
  const suffix = qs.size > 0 ? `?${qs.toString()}` : ''
  const res = await fetch(`${apiBase}/api/v1/board-templates${suffix}`, {
    headers: await authHeaders(),
  })
  if (!res.ok) throw new Error(`listBoardTemplates failed (${res.status})`)
  const body = (await res.json()) as { templates: BoardTemplate[] }
  return (body.templates ?? []).map((t) => ({
    ...t,
    tags: t.tags ?? [],
    courseId: t.courseId ?? null,
    orgId: t.orgId ?? null,
    definition: (t.definition ?? {}) as Record<string, unknown>,
  }))
}

export async function createBoardFromTemplate(
  courseCode: string,
  templateId: string,
  title?: string,
  description?: string,
): Promise<Board> {
  const qs = new URLSearchParams({ from: `template:${templateId}` })
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards?${qs}`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ title: title ?? '', description: description ?? '' }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createBoardFromTemplate failed (${res.status}): ${txt}`)
  }
  return normalizeBoard((await res.json()) as Board)
}

export async function duplicateBoard(
  targetCourseCode: string,
  sourceBoardId: string,
  mode: BoardCopyMode,
  title?: string,
  description?: string,
): Promise<CreateBoardResult> {
  const qs = new URLSearchParams({ from: `board:${sourceBoardId}`, mode })
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(targetCourseCode)}/boards?${qs}`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ title: title ?? '', description: description ?? '' }),
    },
  )
  if (res.status === 202) {
    const body = (await res.json()) as { job: BoardCopyJob }
    return { kind: 'job', job: body.job }
  }
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`duplicateBoard failed (${res.status}): ${txt}`)
  }
  return { kind: 'board', board: normalizeBoard((await res.json()) as Board) }
}

export async function fetchBoardCopyJob(courseCode: string, jobId: string): Promise<BoardCopyJob> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/board-copy-jobs/${encodeURIComponent(jobId)}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`fetchBoardCopyJob failed (${res.status})`)
  return (await res.json()) as BoardCopyJob
}

export async function saveBoardAsTemplate(
  courseCode: string,
  boardId: string,
  input: {
    scope: 'course' | 'org'
    title?: string
    description?: string
    tags?: string[]
    includePosts?: boolean
  },
): Promise<BoardTemplate> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/save-as-template`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({
        scope: input.scope,
        title: input.title ?? '',
        description: input.description ?? '',
        tags: input.tags ?? [],
        includePosts: !!input.includePosts,
      }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`saveBoardAsTemplate failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as BoardTemplate
}

function normalizeBoard(raw: Board): Board {
  return {
    ...raw,
    layout: (raw.layout || 'wall') as BoardLayout,
    layoutLocked: !!raw.layoutLocked,
    settings: (raw.settings ?? {}) as BoardSettings,
    reactionMode: (raw.reactionMode || 'none') as BoardReactionMode,
    assignmentId: raw.assignmentId ?? null,
    visibility: (raw.visibility || 'course') as BoardVisibility,
    visibilityTarget: raw.visibilityTarget ?? null,
    attribution: (raw.attribution || 'named') as BoardAttribution,
    canPost: raw.canPost !== false,
    canInteract: raw.canInteract !== false,
    canArrange: raw.canArrange === true,
    moderationMode: (raw.moderationMode || 'open') as BoardModerationMode,
    filterAction: (raw.filterAction || 'flag') as BoardFilterAction,
    locked: !!raw.locked,
    frozenUntil: raw.frozenUntil ?? null,
  }
}

export async function fetchBoard(courseCode: string, boardId: string): Promise<Board> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`fetchBoard failed (${res.status})`)
  return normalizeBoard((await res.json()) as Board)
}

export async function patchBoard(
  courseCode: string,
  boardId: string,
  patch: {
    title?: string
    description?: string
    archived?: boolean
    layout?: BoardLayout
    layoutLocked?: boolean
    settings?: BoardSettings
    reactionMode?: BoardReactionMode
    assignmentId?: string | null
    visibility?: BoardVisibility
    visibilityTarget?: string | null
    attribution?: BoardAttribution
    canPost?: boolean
    canInteract?: boolean
    canArrange?: boolean
    moderationMode?: BoardModerationMode
    filterAction?: BoardFilterAction
    locked?: boolean
    frozenUntil?: string | null
    freezeMinutes?: number
  },
): Promise<Board> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}`,
    {
      method: 'PATCH',
      headers: await authHeaders(),
      body: JSON.stringify(patch),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`patchBoard failed (${res.status}): ${txt}`)
  }
  return normalizeBoard((await res.json()) as Board)
}

export async function deleteBoard(
  courseCode: string,
  boardId: string,
  opts?: { hard?: boolean },
): Promise<void> {
  const qs = opts?.hard ? '?hard=true' : ''
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}${qs}`,
    { method: 'DELETE', headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`deleteBoard failed (${res.status})`)
}

function normalizePost(raw: BoardPost): BoardPost {
  if (!raw.attachment) return raw
  return {
    ...raw,
    attachment: {
      ...raw.attachment,
      url: absoluteUrl(raw.attachment.url),
    },
  }
}

export async function listBoardPosts(courseCode: string, boardId: string): Promise<BoardPost[]> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`listBoardPosts failed (${res.status})`)
  const body = (await res.json()) as { posts: BoardPost[] }
  return (body.posts ?? []).map(normalizePost)
}

export async function createBoardPost(
  courseCode: string,
  boardId: string,
  input: CreateBoardPostInput,
): Promise<BoardPost> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(input),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createBoardPost failed (${res.status}): ${txt}`)
  }
  return normalizePost((await res.json()) as BoardPost)
}

export async function patchBoardPost(
  courseCode: string,
  boardId: string,
  postId: string,
  patch: { title?: string; body?: BoardPostBody | string; linkUrl?: string; drawingData?: unknown },
): Promise<BoardPost> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}`,
    {
      method: 'PATCH',
      headers: await authHeaders(),
      body: JSON.stringify(patch),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`patchBoardPost failed (${res.status}): ${txt}`)
  }
  return normalizePost((await res.json()) as BoardPost)
}

export async function deleteBoardPost(
  courseCode: string,
  boardId: string,
  postId: string,
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}`,
    { method: 'DELETE', headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`deleteBoardPost failed (${res.status})`)
}

export async function uploadBoardAttachment(
  courseCode: string,
  boardId: string,
  file: File,
  opts?: { altText?: string; contentType?: BoardContentType },
): Promise<BoardAttachment> {
  const form = new FormData()
  form.append('file', file)
  if (opts?.altText) form.append('altText', opts.altText)
  if (opts?.contentType) form.append('contentType', opts.contentType)
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/attachments`,
    {
      method: 'POST',
      headers: await authHeaders(false),
      body: form,
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    let parsed: unknown
    try {
      parsed = JSON.parse(txt) as unknown
    } catch {
      parsed = null
    }
    const code =
      parsed && typeof parsed === 'object' && 'error' in parsed
        ? (parsed as { error?: { code?: string } }).error?.code
        : undefined
    if (code === 'QUOTA_EXCEEDED' || (res.status === 403 && txt.includes('Storage limit'))) {
      throw new Error('QUOTA_EXCEEDED')
    }
    throw new Error(`uploadBoardAttachment failed (${res.status}): ${txt}`)
  }
  const att = (await res.json()) as BoardAttachment
  return { ...att, url: absoluteUrl(att.url) }
}

export async function fetchBoardAnalytics(
  courseCode: string,
  boardId: string,
  days = 14,
): Promise<BoardAnalyticsSummary> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/analytics?days=${days}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`fetchBoardAnalytics failed (${res.status})`)
  return (await res.json()) as BoardAnalyticsSummary
}

export async function fetchAdminBoardPolicies(orgId?: string): Promise<BoardOrgPolicies> {
  const q = orgId ? `?orgId=${encodeURIComponent(orgId)}` : ''
  const res = await fetch(`${apiBase}/api/v1/admin/boards/policies${q}`, {
    headers: await authHeaders(),
  })
  if (!res.ok) throw new Error(`fetchAdminBoardPolicies failed (${res.status})`)
  return (await res.json()) as BoardOrgPolicies
}

export async function patchAdminBoardPolicies(
  body: Partial<{
    externalSharing: boolean
    minorModerationFloor: boolean
    defaultAttribution: BoardAttribution
    boardCapPerCourse: number | null
    clearBoardCap: boolean
  }>,
  orgId?: string,
): Promise<BoardOrgPolicies> {
  const q = orgId ? `?orgId=${encodeURIComponent(orgId)}` : ''
  const res = await fetch(`${apiBase}/api/v1/admin/boards/policies${q}`, {
    method: 'PATCH',
    headers: await authHeaders(),
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`patchAdminBoardPolicies failed (${res.status})`)
  return (await res.json()) as BoardOrgPolicies
}

export async function fetchAdminBoardsOverview(
  orgId?: string,
  activeDays = 30,
): Promise<BoardAdminOverview> {
  const params = new URLSearchParams({ activeDays: String(activeDays) })
  if (orgId) params.set('orgId', orgId)
  const res = await fetch(`${apiBase}/api/v1/admin/boards/overview?${params}`, {
    headers: await authHeaders(),
  })
  if (!res.ok) throw new Error(`fetchAdminBoardsOverview failed (${res.status})`)
  return (await res.json()) as BoardAdminOverview
}

export async function fetchBoardLinkPreview(
  courseCode: string,
  boardId: string,
  url: string,
): Promise<BoardLinkPreview & { url: string; provider?: string; embedId?: string }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/link-preview`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ url }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`fetchBoardLinkPreview failed (${res.status}): ${txt}`)
  }
  return res.json() as Promise<BoardLinkPreview & { url: string; provider?: string; embedId?: string }>
}

export async function listBoardSections(
  courseCode: string,
  boardId: string,
): Promise<BoardSection[]> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/sections`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`listBoardSections failed (${res.status})`)
  const body = (await res.json()) as { sections: BoardSection[] }
  return body.sections ?? []
}

export async function createBoardSection(
  courseCode: string,
  boardId: string,
  title: string,
  sortIndex?: number,
): Promise<BoardSection> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/sections`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ title, sortIndex }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createBoardSection failed (${res.status}): ${txt}`)
  }
  return res.json() as Promise<BoardSection>
}

export async function patchBoardSection(
  courseCode: string,
  boardId: string,
  sectionId: string,
  patch: { title?: string; sortIndex?: number },
): Promise<BoardSection> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/sections/${encodeURIComponent(sectionId)}`,
    {
      method: 'PATCH',
      headers: await authHeaders(),
      body: JSON.stringify(patch),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`patchBoardSection failed (${res.status}): ${txt}`)
  }
  return res.json() as Promise<BoardSection>
}

export async function deleteBoardSection(
  courseCode: string,
  boardId: string,
  sectionId: string,
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/sections/${encodeURIComponent(sectionId)}`,
    { method: 'DELETE', headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`deleteBoardSection failed (${res.status})`)
}

export async function arrangeBoardPost(
  courseCode: string,
  boardId: string,
  postId: string,
  input: ArrangeBoardPostInput,
): Promise<BoardPost> {
  const body: Record<string, unknown> = {}
  if (input.sectionId !== undefined) body.sectionId = input.sectionId
  if (input.sortIndex !== undefined) body.sortIndex = input.sortIndex
  if (input.position !== undefined) body.position = input.position
  if (input.eventDate !== undefined) body.eventDate = input.eventDate ?? ''
  if (input.lat !== undefined) body.lat = input.lat
  if (input.lng !== undefined) body.lng = input.lng
  if (input.clearGeo) body.clearGeo = true

  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}/arrange`,
    {
      method: 'PATCH',
      headers: await authHeaders(),
      body: JSON.stringify(body),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`arrangeBoardPost failed (${res.status}): ${txt}`)
  }
  return normalizePost((await res.json()) as BoardPost)
}

export async function putBoardPostReaction(
  courseCode: string,
  boardId: string,
  postId: string,
  input: { kind?: string; value?: number },
): Promise<BoardReactionResult> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}/reaction`,
    {
      method: 'PUT',
      headers: await authHeaders(),
      body: JSON.stringify(input),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`putBoardPostReaction failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as BoardReactionResult
}

export async function deleteBoardPostReaction(
  courseCode: string,
  boardId: string,
  postId: string,
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}/reaction`,
    { method: 'DELETE', headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`deleteBoardPostReaction failed (${res.status})`)
}

export async function listBoardPostComments(
  courseCode: string,
  boardId: string,
  postId: string,
): Promise<BoardComment[]> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}/comments`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`listBoardPostComments failed (${res.status})`)
  const body = (await res.json()) as { comments: BoardComment[] }
  return body.comments ?? []
}

export async function createBoardPostComment(
  courseCode: string,
  boardId: string,
  postId: string,
  input: { body: BoardPostBody | string; parentId?: string },
): Promise<BoardComment> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}/comments`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(input),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createBoardPostComment failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as BoardComment
}

export async function patchBoardPostComment(
  courseCode: string,
  boardId: string,
  postId: string,
  commentId: string,
  patch: { body?: BoardPostBody | string; hidden?: boolean },
): Promise<BoardComment> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}/comments/${encodeURIComponent(commentId)}`,
    {
      method: 'PATCH',
      headers: await authHeaders(),
      body: JSON.stringify(patch),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`patchBoardPostComment failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as BoardComment
}

export async function deleteBoardPostComment(
  courseCode: string,
  boardId: string,
  postId: string,
  commentId: string,
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}/comments/${encodeURIComponent(commentId)}`,
    { method: 'DELETE', headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`deleteBoardPostComment failed (${res.status})`)
}

export async function syncBoardPostGrade(
  courseCode: string,
  boardId: string,
  postId: string,
): Promise<{ synced: boolean; pointsEarned: number }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}/grade-sync`,
    { method: 'POST', headers: await authHeaders() },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`syncBoardPostGrade failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as { synced: boolean; pointsEarned: number }
}

/** Merge reaction/comment aggregates from a PUT reaction response onto a post. */
export function applyReactionResult(post: BoardPost, result: BoardReactionResult): BoardPost {
  return {
    ...post,
    reactionCount: result.reactionCount ?? post.reactionCount,
    myReaction: result.active ? (result.myReaction ?? post.myReaction) : null,
    avgStars: result.avgStars ?? (result.active ? post.avgStars : undefined),
    commentCount: result.commentCount ?? post.commentCount,
    grade: result.grade ?? (result.active ? post.grade : undefined),
  }
}

/** Sort score for most-reacted (higher first). */
export function boardPostReactionScore(post: BoardPost, mode: BoardReactionMode): number {
  switch (mode) {
    case 'star':
      return (post.avgStars ?? 0) * 1000 + (post.reactionCount ?? 0)
    case 'grade':
      return post.grade ?? post.reactionCount ?? 0
    case 'like':
    case 'vote':
      return post.reactionCount ?? 0
    case 'none':
      return 0
    default: {
      const _exhaustive: never = mode
      return _exhaustive
    }
  }
}

/** Fractional index between neighbors (client-side helper for drag reorder). */
export function midpointSortIndex(before?: number, after?: number): number {
  if (before === undefined && after === undefined) return 0
  if (before === undefined) return after! - 1
  if (after === undefined) return before + 1
  if (after <= before) return before + 1
  return (before + after) / 2
}

export async function listBoardMembers(courseCode: string, boardId: string): Promise<BoardMember[]> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/members`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`listBoardMembers failed (${res.status})`)
  const body = (await res.json()) as { members: BoardMember[] }
  return body.members ?? []
}

export async function upsertBoardMember(
  courseCode: string,
  boardId: string,
  userId: string,
  role: BoardMemberRole,
): Promise<BoardMember> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/members`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ userId, role }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`upsertBoardMember failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as BoardMember
}

export async function removeBoardMember(
  courseCode: string,
  boardId: string,
  userId: string,
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/members/${encodeURIComponent(userId)}`,
    { method: 'DELETE', headers: await authHeaders(false) },
  )
  if (!res.ok) throw new Error(`removeBoardMember failed (${res.status})`)
}

export async function listBoardShares(courseCode: string, boardId: string): Promise<BoardShare[]> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/shares`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`listBoardShares failed (${res.status})`)
  const body = (await res.json()) as { shares: BoardShare[] }
  return body.shares ?? []
}

export async function createBoardShare(
  courseCode: string,
  boardId: string,
  input: { capability: BoardShareCapability; password?: string; expiresAt?: string | null },
): Promise<BoardShare> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/shares`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(input),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createBoardShare failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as BoardShare
}

export async function revokeBoardShare(
  courseCode: string,
  boardId: string,
  shareId: string,
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/shares/${encodeURIComponent(shareId)}`,
    { method: 'DELETE', headers: await authHeaders(false) },
  )
  if (!res.ok) throw new Error(`revokeBoardShare failed (${res.status})`)
}

export type BoardLinkResolve = {
  board: Board
  capability: BoardShareCapability
  requiresPassword: boolean
  posts: BoardPost[]
}

export async function resolveBoardLink(token: string, password?: string): Promise<BoardLinkResolve> {
  const headers: Record<string, string> = {}
  if (password) headers['X-Board-Share-Password'] = password
  const res = await fetch(`${apiBase}/api/v1/board-links/${encodeURIComponent(token)}`, { headers })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`resolveBoardLink failed (${res.status}): ${txt}`)
  }
  const body = (await res.json()) as BoardLinkResolve
  return {
    ...body,
    board: normalizeBoard(body.board),
  }
}

export async function createBoardLinkPost(
  token: string,
  input: { displayName: string; contentType: BoardContentType; title?: string; body?: BoardPostBody | string; linkUrl?: string },
  password?: string,
): Promise<BoardPost> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (password) headers['X-Board-Share-Password'] = password
  const res = await fetch(`${apiBase}/api/v1/board-links/${encodeURIComponent(token)}/posts`, {
    method: 'POST',
    headers,
    body: JSON.stringify(input),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createBoardLinkPost failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as BoardPost
}

export async function fetchBoardModerationQueue(
  courseCode: string,
  boardId: string,
): Promise<BoardModerationQueue> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/moderation/queue`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`fetchBoardModerationQueue failed (${res.status})`)
  const body = (await res.json()) as BoardModerationQueue
  return {
    pending: (body.pending ?? []).map(normalizePost),
    reports: body.reports ?? [],
    flagged: body.flagged ?? [],
    minorsFloor: !!body.minorsFloor,
  }
}

export async function approveBoardPost(
  courseCode: string,
  boardId: string,
  postId: string,
  reason?: string,
): Promise<BoardPost> {
  return postModerationAction(courseCode, boardId, postId, 'approve', reason)
}

export async function rejectBoardPost(
  courseCode: string,
  boardId: string,
  postId: string,
  reason?: string,
): Promise<BoardPost> {
  return postModerationAction(courseCode, boardId, postId, 'reject', reason)
}

export async function hideBoardPost(
  courseCode: string,
  boardId: string,
  postId: string,
  reason?: string,
): Promise<BoardPost> {
  return postModerationAction(courseCode, boardId, postId, 'hide', reason)
}

export async function removeBoardPost(
  courseCode: string,
  boardId: string,
  postId: string,
  reason?: string,
): Promise<BoardPost> {
  return postModerationAction(courseCode, boardId, postId, 'remove', reason)
}

async function postModerationAction(
  courseCode: string,
  boardId: string,
  postId: string,
  action: 'approve' | 'reject' | 'hide' | 'remove',
  reason?: string,
): Promise<BoardPost> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/posts/${encodeURIComponent(postId)}/${action}`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ reason: reason ?? '' }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`${action}BoardPost failed (${res.status}): ${txt}`)
  }
  return normalizePost((await res.json()) as BoardPost)
}

export async function reportBoardContent(
  courseCode: string,
  boardId: string,
  input: { postId?: string; commentId?: string; reason?: string },
): Promise<BoardReport> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/reports`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(input),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`reportBoardContent failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as BoardReport
}

export async function resolveBoardReport(
  courseCode: string,
  boardId: string,
  reportId: string,
  action: 'dismiss' | 'hide' | 'remove' | 'resolve',
  reason?: string,
): Promise<BoardReport> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/reports/${encodeURIComponent(reportId)}/resolve`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ action, reason: reason ?? '' }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`resolveBoardReport failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as BoardReport
}

export type BoardExportFormat = 'pdf' | 'csv' | 'image'

export type BoardExportJob = {
  id: string
  boardId: string
  format: BoardExportFormat
  status: 'pending' | 'running' | 'done' | 'failed'
  storageKey?: string | null
  error: string
  includeModeration: boolean
  requestedBy?: string | null
  createdAt: string
  completedAt?: string | null
  downloadUrl?: string | null
}

export type BoardEmbedMode = 'interactive' | 'readonly' | 'denied'

export type BoardEmbedContext = {
  mode: BoardEmbedMode
  board: Board | null
  posts: BoardPost[]
  sections: BoardSection[]
  capabilities: BoardCapabilities
}

export async function createBoardExport(
  courseCode: string,
  boardId: string,
  input: { format: BoardExportFormat; includeModeration?: boolean },
): Promise<BoardExportJob> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/export`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(input),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createBoardExport failed (${res.status}): ${txt}`)
  }
  const body = (await res.json()) as { job: BoardExportJob }
  return body.job
}

export async function fetchBoardExportJob(
  courseCode: string,
  boardId: string,
  jobId: string,
): Promise<BoardExportJob> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/export/${encodeURIComponent(jobId)}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`fetchBoardExportJob failed (${res.status})`)
  return (await res.json()) as BoardExportJob
}

export async function downloadBoardExport(
  courseCode: string,
  boardId: string,
  jobId: string,
): Promise<Blob> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/export/${encodeURIComponent(jobId)}/content`,
    { headers: await authHeaders(false) },
  )
  if (!res.ok) throw new Error(`downloadBoardExport failed (${res.status})`)
  return res.blob()
}

/** Fetch QR image; returns blob + the URL encoded in the QR (from response header). */
export async function fetchBoardQR(
  courseCode: string,
  boardId: string,
  opts?: { format?: 'png' | 'svg'; size?: number; url?: string },
): Promise<{ blob: Blob; accessUrl: string }> {
  const params = new URLSearchParams()
  if (opts?.format) params.set('format', opts.format)
  if (opts?.size) params.set('size', String(opts.size))
  if (opts?.url) params.set('url', opts.url)
  const qs = params.toString()
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/qr${qs ? `?${qs}` : ''}`,
    { headers: await authHeaders(false) },
  )
  if (!res.ok) throw new Error(`fetchBoardQR failed (${res.status})`)
  const accessUrl = res.headers.get('X-Board-Access-Url') ?? ''
  return { blob: await res.blob(), accessUrl }
}

export async function fetchBoardEmbed(
  courseCode: string,
  boardId: string,
): Promise<BoardEmbedContext> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}/embed`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`fetchBoardEmbed failed (${res.status})`)
  const body = (await res.json()) as BoardEmbedContext
  return {
    mode: body.mode,
    board: body.board ? normalizeBoard(body.board) : null,
    posts: (body.posts ?? []).map(normalizePost),
    sections: body.sections ?? [],
    capabilities: body.capabilities ?? {
      canView: false,
      canPost: false,
      canInteract: false,
      canArrange: false,
      canManage: false,
    },
  }
}

/** Client-side PNG snapshot of card titles for image export (VC.9 FR-6 / AC-7). */
export function renderBoardSurfacePng(
  title: string,
  cards: Array<{ sectionTitle?: string; title: string; body?: string }>,
): Promise<Blob> {
  const width = 800
  const lineH = 18
  const pad = 24
  const lines: string[] = [title, '']
  let prevSec = ''
  for (const c of cards) {
    if (c.sectionTitle && c.sectionTitle !== prevSec) {
      lines.push(`§ ${c.sectionTitle}`)
      prevSec = c.sectionTitle
    }
    lines.push(`• ${c.title || 'Card'}`)
    if (c.body) lines.push(`  ${c.body}`)
    lines.push('')
  }
  if (cards.length === 0) lines.push('(empty board)')
  const height = Math.max(200, pad * 2 + lines.length * lineH)
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) return Promise.reject(new Error('canvas unsupported'))
  ctx.fillStyle = '#fafafc'
  ctx.fillRect(0, 0, width, height)
  ctx.fillStyle = '#0f172a'
  ctx.font = '600 16px system-ui, sans-serif'
  let y = pad + 16
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]!
    if (i === 0) {
      ctx.font = '700 20px system-ui, sans-serif'
      ctx.fillStyle = '#0f172a'
    } else if (line.startsWith('§ ')) {
      ctx.font = '600 14px system-ui, sans-serif'
      ctx.fillStyle = '#4338ca'
    } else {
      ctx.font = '400 14px system-ui, sans-serif'
      ctx.fillStyle = '#1e293b'
    }
    ctx.fillText(line.slice(0, 100), pad, y)
    y += lineH
  }
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) reject(new Error('png encode failed'))
      else resolve(blob)
    }, 'image/png')
  })
}

/** Extract YouTube / Vimeo embed ids for inline players. */
export function videoEmbedFromUrl(url: string): { provider: 'youtube' | 'vimeo'; id: string } | null {
  try {
    const u = new URL(url.trim())
    const host = u.hostname.toLowerCase()
    if (host === 'youtu.be') {
      const id = u.pathname.split('/').filter(Boolean)[0]
      return id ? { provider: 'youtube', id } : null
    }
    if (host.includes('youtube.com')) {
      const v = u.searchParams.get('v')
      if (v) return { provider: 'youtube', id: v }
      const parts = u.pathname.split('/').filter(Boolean)
      const idx = parts.findIndex((p) => p === 'embed' || p === 'shorts' || p === 'v')
      if (idx >= 0 && parts[idx + 1]) return { provider: 'youtube', id: parts[idx + 1] }
    }
    if (host.includes('vimeo.com')) {
      const id = u.pathname.split('/').filter(Boolean).pop()
      if (id && /^\d+$/.test(id)) return { provider: 'vimeo', id }
    }
  } catch {
    return null
  }
  return null
}

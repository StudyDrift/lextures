import { getAccessToken } from './auth'

const apiBase = import.meta.env.VITE_API_URL ?? ''

export type KitStatus = 'draft' | 'ready' | 'archived'
export type KitVisibility = 'private' | 'course' | 'org' | 'public'

export type CatalogStatus = 'unlisted' | 'pending' | 'listed' | 'rejected'
export type TemplateScope = 'system' | 'org' | 'course'
export type KitSharePermission = 'view' | 'copy' | 'edit'
export type KitShareGranteeType = 'user' | 'course' | 'org_unit' | 'org'

export type QuizKit = {
  id: string
  courseId: string | null
  title: string
  description: string
  slug: string
  coverImageRef: string | null
  status: KitStatus
  visibility: KitVisibility
  tags: string[]
  questionCount: number
  archived: boolean
  createdBy: string | null
  createdAt: string
  updatedAt: string
  isTemplate?: boolean
  templateScope?: TemplateScope | null
  derivedFromKitId?: string | null
  attribution?: string
  subject?: string | null
  gradeBand?: string | null
  language?: string | null
  catalogStatus?: CatalogStatus
  validation?: { isReady: boolean; issues: { code: string; message: string }[] }
}

export type QuizKitShare = {
  id: string
  kitId: string
  granteeType: KitShareGranteeType
  granteeId: string | null
  permission: KitSharePermission
  createdBy: string | null
  createdAt: string
}

export type ListKitsResult = {
  kits: QuizKit[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

async function authHeaders(json = true): Promise<Record<string, string>> {
  const tok = getAccessToken()
  const headers: Record<string, string> = {}
  if (json) headers['Content-Type'] = 'application/json'
  if (tok) headers.Authorization = `Bearer ${tok}`
  return headers
}

function normalizeKit(raw: QuizKit): QuizKit {
  return {
    ...raw,
    courseId: raw.courseId ?? null,
    coverImageRef: raw.coverImageRef ?? null,
    tags: raw.tags ?? [],
    questionCount: raw.questionCount ?? 0,
    status: (raw.status || 'draft') as KitStatus,
    visibility: (raw.visibility || 'course') as KitVisibility,
    isTemplate: raw.isTemplate === true,
    templateScope: (raw.templateScope ?? null) as TemplateScope | null,
    derivedFromKitId: raw.derivedFromKitId ?? null,
    attribution: raw.attribution ?? '',
    subject: raw.subject ?? null,
    gradeBand: raw.gradeBand ?? null,
    language: raw.language ?? null,
    catalogStatus: (raw.catalogStatus || 'unlisted') as CatalogStatus,
  }
}

export async function listQuizKits(
  courseCode: string,
  opts?: { q?: string; tag?: string; page?: number; pageSize?: number; includeArchived?: boolean },
): Promise<ListKitsResult> {
  const params = new URLSearchParams()
  if (opts?.q) params.set('q', opts.q)
  if (opts?.tag) params.set('tag', opts.tag)
  if (opts?.page) params.set('page', String(opts.page))
  if (opts?.pageSize) params.set('pageSize', String(opts.pageSize))
  if (opts?.includeArchived) params.set('includeArchived', 'true')
  const qs = params.toString() ? `?${params.toString()}` : ''
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits${qs}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`listQuizKits failed (${res.status})`)
  const body = (await res.json()) as ListKitsResult
  return {
    kits: (body.kits ?? []).map(normalizeKit),
    total: body.total ?? 0,
    page: body.page ?? 1,
    pageSize: body.pageSize ?? 50,
    totalPages: body.totalPages ?? 0,
  }
}

export async function createQuizKit(
  courseCode: string,
  title: string,
  description?: string,
  tags?: string[],
): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ title, description: description ?? '', tags: tags ?? [] }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createQuizKit failed (${res.status}): ${txt}`)
  }
  return normalizeKit((await res.json()) as QuizKit)
}

export async function fetchQuizKit(courseCode: string, kitId: string): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`fetchQuizKit failed (${res.status})`)
  return normalizeKit((await res.json()) as QuizKit)
}

export async function patchQuizKit(
  courseCode: string,
  kitId: string,
  patch: {
    title?: string
    description?: string
    coverImageRef?: string | null
    status?: KitStatus
    visibility?: KitVisibility
    tags?: string[]
    archived?: boolean
  },
): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}`,
    {
      method: 'PATCH',
      headers: await authHeaders(),
      body: JSON.stringify(patch),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`patchQuizKit failed (${res.status}): ${txt}`)
  }
  return normalizeKit((await res.json()) as QuizKit)
}

export async function duplicateQuizKit(
  courseCode: string,
  kitId: string,
  targetCourseCode?: string,
): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/duplicate`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(targetCourseCode ? { targetCourseCode } : {}),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`duplicateQuizKit failed (${res.status}): ${txt}`)
  }
  return normalizeKit((await res.json()) as QuizKit)
}

export async function saveQuizKitAsTemplate(
  courseCode: string,
  kitId: string,
  input: { scope: 'course' | 'org'; title?: string; description?: string; tags?: string[] },
): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/save-as-template`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(input),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`saveQuizKitAsTemplate failed (${res.status}): ${txt}`)
  }
  return normalizeKit((await res.json()) as QuizKit)
}

export async function listQuizTemplates(opts?: {
  courseCode?: string
  scope?: string
  q?: string
}): Promise<QuizKit[]> {
  const params = new URLSearchParams()
  if (opts?.courseCode) params.set('courseCode', opts.courseCode)
  if (opts?.scope) params.set('scope', opts.scope)
  if (opts?.q) params.set('q', opts.q)
  const qs = params.toString() ? `?${params.toString()}` : ''
  const res = await fetch(`${apiBase}/api/v1/live-quizzes/templates${qs}`, {
    headers: await authHeaders(),
  })
  if (!res.ok) throw new Error(`listQuizTemplates failed (${res.status})`)
  const body = (await res.json()) as { templates?: QuizKit[] }
  return (body.templates ?? []).map(normalizeKit)
}

export async function createKitFromTemplate(
  templateId: string,
  targetCourseCode: string,
): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/live-quizzes/templates/${encodeURIComponent(templateId)}/create-kit`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ targetCourseCode }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createKitFromTemplate failed (${res.status}): ${txt}`)
  }
  return normalizeKit((await res.json()) as QuizKit)
}

export async function listQuizKitShares(courseCode: string, kitId: string): Promise<QuizKitShare[]> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/shares`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`listQuizKitShares failed (${res.status})`)
  const body = (await res.json()) as { shares?: QuizKitShare[] }
  return body.shares ?? []
}

export async function createQuizKitShare(
  courseCode: string,
  kitId: string,
  input: { granteeType: KitShareGranteeType; granteeId?: string | null; permission?: KitSharePermission },
): Promise<QuizKitShare> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/shares`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(input),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createQuizKitShare failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as QuizKitShare
}

export async function deleteQuizKitShare(
  courseCode: string,
  kitId: string,
  shareId: string,
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/shares/${encodeURIComponent(shareId)}`,
    { method: 'DELETE', headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`deleteQuizKitShare failed (${res.status})`)
}

export async function searchQuizLibrary(opts?: {
  q?: string
  subject?: string
  grade?: string
  lang?: string
  tag?: string
  page?: number
  pageSize?: number
}): Promise<ListKitsResult> {
  const params = new URLSearchParams()
  if (opts?.q) params.set('q', opts.q)
  if (opts?.subject) params.set('subject', opts.subject)
  if (opts?.grade) params.set('grade', opts.grade)
  if (opts?.lang) params.set('lang', opts.lang)
  if (opts?.tag) params.set('tag', opts.tag)
  if (opts?.page) params.set('page', String(opts.page))
  if (opts?.pageSize) params.set('pageSize', String(opts.pageSize))
  const qs = params.toString() ? `?${params.toString()}` : ''
  const res = await fetch(`${apiBase}/api/v1/live-quizzes/library${qs}`, {
    headers: await authHeaders(),
  })
  if (!res.ok) throw new Error(`searchQuizLibrary failed (${res.status})`)
  const body = (await res.json()) as ListKitsResult
  return {
    kits: (body.kits ?? []).map(normalizeKit),
    total: body.total ?? 0,
    page: body.page ?? 1,
    pageSize: body.pageSize ?? 50,
    totalPages: body.totalPages ?? 0,
  }
}

export async function previewQuizLibraryKit(
  kitId: string,
): Promise<{ kit: QuizKit; questions: LiveQuizQuestion[] }> {
  const res = await fetch(
    `${apiBase}/api/v1/live-quizzes/library/${encodeURIComponent(kitId)}/preview`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`previewQuizLibraryKit failed (${res.status})`)
  const body = (await res.json()) as { kit: QuizKit; questions: LiveQuizQuestion[] }
  return { kit: normalizeKit(body.kit), questions: body.questions ?? [] }
}

export async function importQuizLibraryKit(
  kitId: string,
  targetCourseCode: string,
): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/live-quizzes/library/${encodeURIComponent(kitId)}/import`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ targetCourseCode }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`importQuizLibraryKit failed (${res.status}): ${txt}`)
  }
  return normalizeKit((await res.json()) as QuizKit)
}

export async function submitQuizKitToCatalog(courseCode: string, kitId: string): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/submit-to-catalog`,
    { method: 'POST', headers: await authHeaders() },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`submitQuizKitToCatalog failed (${res.status}): ${txt}`)
  }
  return normalizeKit((await res.json()) as QuizKit)
}

export async function archiveQuizKit(courseCode: string, kitId: string): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/archive`,
    { method: 'POST', headers: await authHeaders() },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`archiveQuizKit failed (${res.status}): ${txt}`)
  }
  return normalizeKit((await res.json()) as QuizKit)
}

export async function restoreQuizKit(courseCode: string, kitId: string): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/restore`,
    { method: 'POST', headers: await authHeaders() },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`restoreQuizKit failed (${res.status}): ${txt}`)
  }
  return normalizeKit((await res.json()) as QuizKit)
}

export type LiveQuizQuestionType =
  | 'mc_single'
  | 'mc_multiple'
  | 'true_false'
  | 'type_answer'
  | 'numeric'
  | 'poll'
  | 'ordering'
  | 'word_cloud'

export type LiveQuizPointsStyle = 'standard' | 'double' | 'no_points'

export type LiveQuizOption = {
  id: string
  text: string
  mediaRef?: string | null
  mediaAlt?: string | null
  isCorrect: boolean
}

export type LiveQuizQuestionSource = 'authored' | 'ai_generated' | 'bank_import'

export type LiveQuizQuestion = {
  id: string
  kitId: string
  position: number
  questionType: LiveQuizQuestionType
  prompt: string
  promptMediaRef: string | null
  promptMediaAlt: string | null
  options: LiveQuizOption[]
  correctAnswer: unknown
  timeLimitSeconds: number
  pointsStyle: LiveQuizPointsStyle
  answerShuffle: boolean
  explanation: string | null
  sourceQuestionId: string | null
  source?: LiveQuizQuestionSource
  needsReview?: boolean
  generationJobId?: string | null
  generationConfidence?: number | null
  version: number
  createdAt: string
  updatedAt: string
}

export type LiveQuizGenSourceType = 'topic' | 'passage' | 'course_content_ref'

export type LiveQuizGenerationParams = {
  count?: number
  types?: LiveQuizQuestionType[]
  difficulty?: 'easy' | 'medium' | 'hard'
  gradeBand?: string
  language?: string
  includeExplanations?: boolean
  likeQuestionId?: string
  replaceQuestionId?: string
}

export type LiveQuizGenerationJob = {
  id: string
  kitId: string
  courseId: string
  requestedBy: string | null
  sourceType: LiveQuizGenSourceType
  sourceRef: Record<string, unknown>
  params: LiveQuizGenerationParams
  status: 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled'
  provider: string | null
  model: string | null
  usageId: string | null
  error: string | null
  resultSummary: {
    inserted?: number
    repaired?: number
    dropped?: number
    questionIds?: string[]
  } | null
  progress: number
  createdAt: string
  startedAt: string | null
  completedAt: string | null
}

export type KitValidationIssue = {
  questionId: string
  code: string
  message: string
}

export type KitValidationResult = {
  isReady: boolean
  issues: KitValidationIssue[]
}

export type BankCandidate = {
  id: string
  questionType: string
  stem: string
  status: string
}

export type CreateQuestionBody = {
  questionType?: LiveQuizQuestionType
  prompt?: string
  promptMediaRef?: string | null
  promptMediaAlt?: string | null
  options?: LiveQuizOption[]
  correctAnswer?: unknown
  timeLimitSeconds?: number
  pointsStyle?: LiveQuizPointsStyle
  answerShuffle?: boolean
  explanation?: string | null
}

export type PatchQuestionBody = CreateQuestionBody & {
  clearPromptMedia?: boolean
}

function normalizeQuestion(raw: LiveQuizQuestion): LiveQuizQuestion {
  return {
    ...raw,
    promptMediaRef: raw.promptMediaRef ?? null,
    promptMediaAlt: raw.promptMediaAlt ?? null,
    options: Array.isArray(raw.options) ? raw.options : [],
    correctAnswer: raw.correctAnswer ?? null,
    explanation: raw.explanation ?? null,
    sourceQuestionId: raw.sourceQuestionId ?? null,
    source: (raw.source || 'authored') as LiveQuizQuestionSource,
    needsReview: raw.needsReview === true,
    generationJobId: raw.generationJobId ?? null,
    generationConfidence:
      typeof raw.generationConfidence === 'number' ? raw.generationConfidence : null,
    answerShuffle: raw.answerShuffle ?? true,
    timeLimitSeconds: raw.timeLimitSeconds ?? 20,
    pointsStyle: (raw.pointsStyle || 'standard') as LiveQuizPointsStyle,
    questionType: (raw.questionType || 'mc_single') as LiveQuizQuestionType,
  }
}

function questionsBase(courseCode: string, kitId: string) {
  return `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}`
}

export async function listQuizQuestions(courseCode: string, kitId: string): Promise<LiveQuizQuestion[]> {
  const res = await fetch(`${questionsBase(courseCode, kitId)}/questions`, {
    headers: await authHeaders(),
  })
  if (!res.ok) throw new Error(`listQuizQuestions failed (${res.status})`)
  const body = (await res.json()) as { questions?: LiveQuizQuestion[] }
  return (body.questions ?? []).map(normalizeQuestion)
}

export async function createQuizQuestion(
  courseCode: string,
  kitId: string,
  body: CreateQuestionBody,
): Promise<LiveQuizQuestion> {
  const res = await fetch(`${questionsBase(courseCode, kitId)}/questions`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createQuizQuestion failed (${res.status}): ${txt}`)
  }
  return normalizeQuestion((await res.json()) as LiveQuizQuestion)
}

export class VersionConflictError extends Error {
  constructor(message = 'Question was modified elsewhere') {
    super(message)
    this.name = 'VersionConflictError'
  }
}

export async function patchQuizQuestion(
  courseCode: string,
  kitId: string,
  questionId: string,
  version: number,
  body: PatchQuestionBody,
): Promise<LiveQuizQuestion> {
  const headers = await authHeaders()
  headers['If-Match'] = String(version)
  const res = await fetch(
    `${questionsBase(courseCode, kitId)}/questions/${encodeURIComponent(questionId)}`,
    { method: 'PATCH', headers, body: JSON.stringify(body) },
  )
  if (res.status === 409) throw new VersionConflictError()
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`patchQuizQuestion failed (${res.status}): ${txt}`)
  }
  return normalizeQuestion((await res.json()) as LiveQuizQuestion)
}

export async function deleteQuizQuestion(
  courseCode: string,
  kitId: string,
  questionId: string,
): Promise<void> {
  const res = await fetch(
    `${questionsBase(courseCode, kitId)}/questions/${encodeURIComponent(questionId)}`,
    { method: 'DELETE', headers: await authHeaders(false) },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`deleteQuizQuestion failed (${res.status}): ${txt}`)
  }
}

export async function duplicateQuizQuestion(
  courseCode: string,
  kitId: string,
  questionId: string,
): Promise<LiveQuizQuestion> {
  const res = await fetch(
    `${questionsBase(courseCode, kitId)}/questions/${encodeURIComponent(questionId)}/duplicate`,
    { method: 'POST', headers: await authHeaders(false) },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`duplicateQuizQuestion failed (${res.status}): ${txt}`)
  }
  return normalizeQuestion((await res.json()) as LiveQuizQuestion)
}

export async function reorderQuizQuestions(
  courseCode: string,
  kitId: string,
  items: { id: string; position: number }[],
): Promise<LiveQuizQuestion[]> {
  const res = await fetch(`${questionsBase(courseCode, kitId)}/questions/reorder`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ items }),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`reorderQuizQuestions failed (${res.status}): ${txt}`)
  }
  const body = (await res.json()) as { questions?: LiveQuizQuestion[] }
  return (body.questions ?? []).map(normalizeQuestion)
}

export async function importBankQuestions(
  courseCode: string,
  kitId: string,
  questionIds: string[],
): Promise<LiveQuizQuestion[]> {
  const res = await fetch(`${questionsBase(courseCode, kitId)}/questions/import-bank`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ questionIds }),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`importBankQuestions failed (${res.status}): ${txt}`)
  }
  const body = (await res.json()) as { questions?: LiveQuizQuestion[] }
  return (body.questions ?? []).map(normalizeQuestion)
}

export async function startQuizKitGeneration(
  courseCode: string,
  kitId: string,
  input: {
    sourceType: LiveQuizGenSourceType
    sourceRef: Record<string, unknown>
    params?: LiveQuizGenerationParams
  },
): Promise<LiveQuizGenerationJob> {
  const res = await fetch(`${questionsBase(courseCode, kitId)}/generate`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify(input),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`startQuizKitGeneration failed (${res.status}): ${txt}`)
  }
  const body = (await res.json()) as { job: LiveQuizGenerationJob }
  return body.job
}

export async function fetchQuizKitGenerationJob(
  courseCode: string,
  kitId: string,
  jobId: string,
): Promise<LiveQuizGenerationJob> {
  const res = await fetch(
    `${questionsBase(courseCode, kitId)}/generate/${encodeURIComponent(jobId)}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`fetchQuizKitGenerationJob failed (${res.status}): ${txt}`)
  }
  const body = (await res.json()) as { job: LiveQuizGenerationJob }
  return body.job
}

export async function cancelQuizKitGenerationJob(
  courseCode: string,
  kitId: string,
  jobId: string,
): Promise<LiveQuizGenerationJob> {
  const res = await fetch(
    `${questionsBase(courseCode, kitId)}/generate/${encodeURIComponent(jobId)}/cancel`,
    { method: 'POST', headers: await authHeaders() },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`cancelQuizKitGenerationJob failed (${res.status}): ${txt}`)
  }
  const body = (await res.json()) as { job: LiveQuizGenerationJob }
  return body.job
}

export async function regenerateQuizQuestion(
  courseCode: string,
  kitId: string,
  questionId: string,
  input?: {
    sourceType?: LiveQuizGenSourceType
    sourceRef?: Record<string, unknown>
    params?: LiveQuizGenerationParams
  },
): Promise<LiveQuizGenerationJob> {
  const res = await fetch(
    `${questionsBase(courseCode, kitId)}/questions/${encodeURIComponent(questionId)}/regenerate`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(input ?? {}),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`regenerateQuizQuestion failed (${res.status}): ${txt}`)
  }
  const body = (await res.json()) as { job: LiveQuizGenerationJob }
  return body.job
}

export async function pushQuestionToBank(
  courseCode: string,
  kitId: string,
  questionId: string,
): Promise<string> {
  const res = await fetch(
    `${questionsBase(courseCode, kitId)}/questions/${encodeURIComponent(questionId)}/push-to-bank`,
    { method: 'POST', headers: await authHeaders(false) },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`pushQuestionToBank failed (${res.status}): ${txt}`)
  }
  const body = (await res.json()) as { bankQuestionId?: string }
  return body.bankQuestionId ?? ''
}

export async function validateQuizKit(
  courseCode: string,
  kitId: string,
): Promise<KitValidationResult> {
  const res = await fetch(`${questionsBase(courseCode, kitId)}/validate`, {
    headers: await authHeaders(),
  })
  if (!res.ok) throw new Error(`validateQuizKit failed (${res.status})`)
  const body = (await res.json()) as KitValidationResult
  return { isReady: !!body.isReady, issues: body.issues ?? [] }
}

export async function listBankCandidates(
  courseCode: string,
  kitId: string,
  opts?: { q?: string; limit?: number },
): Promise<BankCandidate[]> {
  const params = new URLSearchParams()
  if (opts?.q) params.set('q', opts.q)
  if (opts?.limit) params.set('limit', String(opts.limit))
  const qs = params.toString() ? `?${params.toString()}` : ''
  const res = await fetch(`${questionsBase(courseCode, kitId)}/questions/bank-candidates${qs}`, {
    headers: await authHeaders(),
  })
  if (!res.ok) throw new Error(`listBankCandidates failed (${res.status})`)
  const body = (await res.json()) as { questions?: BankCandidate[] }
  return body.questions ?? []
}

export type LiveGamePacing = 'manual' | 'auto'

export type LiveGamePlayer = {
  id: string
  nickname: string
  totalScore: number
  streak: number
  connected: boolean
}

export type LiveGame = {
  id: string
  kitId: string
  joinCode: string
  mode?: LiveGameMode
  status: string
  phase: string
  pacing: LiveGamePacing
  questionIndex: number
  kitTitle: string
  questionCount: number
  players?: LiveGamePlayer[]
  openedAt?: string
  deadline?: string
  allowGuests?: boolean
  lobbyLocked?: boolean
  namesMuted?: boolean
  oneSessionRule?: 'takeover' | 'refuse' | 'off'
  maxJoinsPerIp?: number
}

export type LiveGameScoringProfile = 'competitive' | 'formative' | 'custom'
export type LeaderboardPrivacy = 'names' | 'nicknames' | 'hidden'
export type LiveGameMode = 'live_classic' | 'team' | 'student_paced' | 'homework'

export type LiveGameScoringConfig = {
  base?: number
  speedWeight?: number
  streakStep?: number
  streakCap?: number
  powerUpsEnabled?: boolean
  participationPoints?: number
}

export type TeamConfig = {
  teamCount?: number
  aggregate?: 'average' | 'sum'
  answerRule?: 'each_member_answers' | 'one_device_per_team'
  autoBalance?: boolean
}

export type PacedConfig = {
  shuffle?: boolean
  timeBudgetSeconds?: number
  perQuestionTimers?: boolean
  liveLeaderboard?: boolean
}

export type StartLiveGameOpts = {
  pacing?: LiveGamePacing
  mode?: LiveGameMode
  teamConfig?: TeamConfig
  pacedConfig?: PacedConfig
  scoringProfile?: LiveGameScoringProfile
  scoringConfig?: LiveGameScoringConfig
  leaderboardPrivacy?: LeaderboardPrivacy
  powerUpsEnabled?: boolean
}

export async function startLiveGame(
  courseCode: string,
  kitId: string,
  opts?: StartLiveGameOpts,
): Promise<{ gameId: string; joinCode: string; game: LiveGame }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/games`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({
        pacing: opts?.pacing ?? 'manual',
        mode: opts?.mode ?? 'live_classic',
        teamConfig: opts?.teamConfig,
        pacedConfig: opts?.pacedConfig,
        scoringProfile: opts?.scoringProfile ?? 'competitive',
        scoringConfig: opts?.scoringConfig ?? {},
        leaderboardPrivacy: opts?.leaderboardPrivacy ?? 'names',
        powerUpsEnabled: opts?.powerUpsEnabled ?? false,
      }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`startLiveGame failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as { gameId: string; joinCode: string; game: LiveGame }
}

export type LiveQuizAssignment = {
  id: string
  kitId: string
  courseId: string
  title: string
  opensAt?: string | null
  dueAt?: string | null
  closesAt?: string | null
  attemptsAllowed: number
  gradePolicy: 'best' | 'last' | 'average'
  shuffle: boolean
  state?: string
  gradebookScore?: number
  attemptsUsed?: number
  effectiveAttemptsAllowed?: number
}

export type CreateAssignmentOpts = {
  title?: string
  opensAt?: string | null
  dueAt?: string | null
  closesAt?: string | null
  attemptsAllowed?: number
  gradePolicy?: 'best' | 'last' | 'average'
  shuffle?: boolean
  pointsPossible?: number
  scoringProfile?: LiveGameScoringProfile
}

export async function createLiveQuizAssignment(
  courseCode: string,
  kitId: string,
  opts: CreateAssignmentOpts,
): Promise<LiveQuizAssignment> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/assignments`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(opts),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`createLiveQuizAssignment failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as LiveQuizAssignment
}

export async function listLiveQuizAssignments(courseCode: string): Promise<LiveQuizAssignment[]> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/assignments`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`listLiveQuizAssignments failed (${res.status})`)
  const body = (await res.json()) as { assignments: LiveQuizAssignment[] }
  return body.assignments ?? []
}

export async function fetchLiveQuizAssignment(
  courseCode: string,
  assignmentId: string,
): Promise<LiveQuizAssignment> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/assignments/${encodeURIComponent(assignmentId)}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`fetchLiveQuizAssignment failed (${res.status})`)
  return (await res.json()) as LiveQuizAssignment
}

export async function startLiveQuizAssignment(
  courseCode: string,
  assignmentId: string,
  nickname?: string,
): Promise<{ attemptId: string; sessionId: string; playerId: string; playerToken: string }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/assignments/${encodeURIComponent(assignmentId)}/start`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ nickname: nickname ?? '' }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`startLiveQuizAssignment failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as {
    attemptId: string
    sessionId: string
    playerId: string
    playerToken: string
  }
}

export async function assignLiveGameTeams(
  courseCode: string,
  gameId: string,
  opts?: { autoBalance?: boolean; assignments?: Record<string, string> },
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/teams/assign`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({
        autoBalance: opts?.autoBalance ?? true,
        assignments: opts?.assignments ?? {},
      }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`assignLiveGameTeams failed (${res.status}): ${txt}`)
  }
}

export async function startPacedLiveGame(courseCode: string, gameId: string): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/paced/start`,
    { method: 'POST', headers: await authHeaders(false) },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`startPacedLiveGame failed (${res.status}): ${txt}`)
  }
}

export async function fetchLiveGame(courseCode: string, gameId: string): Promise<LiveGame> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}`,
    { headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`fetchLiveGame failed (${res.status})`)
  return (await res.json()) as LiveGame
}

export async function endLiveGame(courseCode: string, gameId: string): Promise<LiveGame> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/end`,
    { method: 'POST', headers: await authHeaders(false) },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`endLiveGame failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as LiveGame
}

export type GradebookMapping = 'raw_points' | 'percent_correct' | 'participation'

export type QuestionAggregate = {
  index: number
  prompt: string
  correctPct: number
  avgMs: number
  answerCount: number
  distribution: Record<string, number>
  sourceQuestionId?: string
  hardestRank?: number
}

export type GameReport = {
  sessionId: string
  playerCount: number
  answeredCount: number
  scoreAvg: number | null
  scoreMedian: number | null
  scoreMax: number | null
  perQuestion: QuestionAggregate[]
  generatedAt: string
}

export type PlayerResultRow = {
  playerId: string
  nickname: string
  userId?: string
  isGuest: boolean
  totalScore: number
  rank: number
  answered: number
  correct: number
}

export type GradebookLink = {
  id: string
  sessionId?: string
  assignmentId?: string
  courseId: string
  gradebookItemId: string
  mapping: GradebookMapping
  pointsPossible?: number
  participationPct: number
}

export type GradePreview = {
  userId?: string
  nickname?: string
  pointsEarned: number
  pointsPossible: number
  skippedGuest?: boolean
}

export type GameReportResponse = {
  report: GameReport
  players: PlayerResultRow[]
  leaderboard: Array<{ rank: number; playerId: string; nickname: string; totalScore: number }>
  title: string
  status: string
  mode: string
  guestCount: number
  gradebookLink?: GradebookLink
}

export type ReviewItem = {
  index: number
  prompt: string
  explanation?: string
  isCorrect: boolean
  points: number
  responseMs: number
  answer: unknown
  correctOptionIds?: string[]
  correctAnswer?: Record<string, unknown>
  reason: 'incorrect' | 'slow' | string
}

export type MyResults = {
  sessionId: string
  nickname: string
  totalScore: number
  rank: number
  playerCount: number
  answered: number
  correct: number
  reviewThese: ReviewItem[]
}

function gameReportBase(courseCode: string, gameId: string) {
  return `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}`
}

export async function fetchGameReport(courseCode: string, gameId: string): Promise<GameReportResponse> {
  const res = await fetch(`${gameReportBase(courseCode, gameId)}/report`, {
    headers: await authHeaders(),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`fetchGameReport failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as GameReportResponse
}

export async function fetchMyGameResults(courseCode: string, gameId: string): Promise<MyResults> {
  const res = await fetch(`${gameReportBase(courseCode, gameId)}/my-results`, {
    headers: await authHeaders(),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`fetchMyGameResults failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as MyResults
}

export async function rebuildGameReport(courseCode: string, gameId: string): Promise<GameReport> {
  const res = await fetch(`${gameReportBase(courseCode, gameId)}/report/rebuild`, {
    method: 'POST',
    headers: await authHeaders(false),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`rebuildGameReport failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as GameReport
}

export function gameReportExportUrl(
  courseCode: string,
  gameId: string,
  format: 'csv' | 'pdf' | 'html' = 'csv',
): string {
  return `${gameReportBase(courseCode, gameId)}/report/export?format=${encodeURIComponent(format)}`
}

export async function pushGameGradebookLink(
  courseCode: string,
  gameId: string,
  body: {
    mapping: GradebookMapping
    pointsPossible: number
    participationPct?: number
    title?: string
    previewOnly?: boolean
  },
): Promise<{ link?: GradebookLink; preview: GradePreview[] }> {
  const res = await fetch(`${gameReportBase(courseCode, gameId)}/gradebook-link`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`pushGameGradebookLink failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as { link?: GradebookLink; preview: GradePreview[] }
}

export async function unlinkGameGradebook(
  courseCode: string,
  gameId: string,
  linkId: string,
): Promise<void> {
  const res = await fetch(
    `${gameReportBase(courseCode, gameId)}/gradebook-link/${encodeURIComponent(linkId)}`,
    { method: 'DELETE', headers: await authHeaders(false) },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`unlinkGameGradebook failed (${res.status}): ${txt}`)
  }
}

export type JoinLookup = {
  gameId: string
  courseCode: string
  kitTitle: string
  requiresAuth: boolean
  allowsGuests: boolean
  phase: string
  status: string
}

export type JoinPlayerResult = {
  playerId: string
  nickname: string
  playerToken: string
  totalScore: number
  streak?: number
  rejoined?: boolean
}

export class LiveQuizJoinError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'LiveQuizJoinError'
    this.status = status
  }
}

export async function lookupJoinCode(code: string): Promise<JoinLookup> {
  const res = await fetch(`${apiBase}/api/v1/live-quizzes/join/${encodeURIComponent(code)}`)
  if (res.status === 404) {
    throw new LiveQuizJoinError(404, 'not_found')
  }
  if (res.status === 429) {
    throw new LiveQuizJoinError(429, 'rate_limited')
  }
  if (!res.ok) throw new LiveQuizJoinError(res.status, `lookupJoinCode failed (${res.status})`)
  const body = (await res.json()) as JoinLookup
  return {
    gameId: body.gameId,
    courseCode: body.courseCode ?? '',
    kitTitle: body.kitTitle ?? '',
    requiresAuth: body.requiresAuth !== false,
    allowsGuests: !!body.allowsGuests,
    phase: body.phase ?? '',
    status: body.status ?? '',
  }
}

export async function joinLiveGame(
  courseCode: string,
  gameId: string,
  nickname: string,
): Promise<JoinPlayerResult> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/players`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ nickname }),
    },
  )
  return parseJoinResponse(res)
}

/** Guest join via public join code (IQ.9). */
export async function joinLiveGameAsGuest(code: string, nickname: string): Promise<JoinPlayerResult> {
  const res = await fetch(`${apiBase}/api/v1/live-quizzes/join/${encodeURIComponent(code)}/players`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ nickname }),
  })
  return parseJoinResponse(res)
}

async function parseJoinResponse(res: Response): Promise<JoinPlayerResult> {
  if (res.status === 409) {
    const txt = await res.text()
    if (txt.toLowerCase().includes('already connected')) {
      throw new LiveQuizJoinError(409, 'one_session')
    }
    throw new LiveQuizJoinError(409, 'nickname_taken')
  }
  if (res.status === 401 || res.status === 403) {
    const txt = await res.text()
    if (txt.toLowerCase().includes('lobby')) {
      throw new LiveQuizJoinError(403, 'lobby_locked')
    }
    if (txt.toLowerCase().includes('rejoin') || txt.toLowerCase().includes('cannot')) {
      throw new LiveQuizJoinError(403, 'banned')
    }
    throw new LiveQuizJoinError(res.status, 'auth_required')
  }
  if (res.status === 400) {
    const txt = await res.text()
    if (txt.toLowerCase().includes('ended')) {
      throw new LiveQuizJoinError(400, 'game_ended')
    }
    if (txt.toLowerCase().includes('isn’t allowed') || txt.toLowerCase().includes("isn't allowed") || txt.toLowerCase().includes('not allowed')) {
      throw new LiveQuizJoinError(400, 'nickname_denied')
    }
    if (txt.toLowerCase().includes('nickname')) {
      throw new LiveQuizJoinError(400, 'nickname_invalid')
    }
    throw new LiveQuizJoinError(400, 'join_failed')
  }
  if (res.status === 429) {
    throw new LiveQuizJoinError(429, 'rate_limited')
  }
  if (!res.ok) {
    throw new LiveQuizJoinError(res.status, `join failed (${res.status})`)
  }
  return (await res.json()) as JoinPlayerResult
}

export type GameSafetySettings = {
  allowGuests: boolean
  lobbyLocked: boolean
  namesMuted: boolean
  oneSessionRule: string
  maxJoinsPerIp: number
}

export async function patchGameSafety(
  courseCode: string,
  gameId: string,
  patch: Partial<{
    allowGuests: boolean
    lobbyLocked: boolean
    namesMuted: boolean
    oneSessionRule: string
    maxJoinsPerIp: number
  }>,
): Promise<GameSafetySettings> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/safety`,
    {
      method: 'PATCH',
      headers: await authHeaders(),
      body: JSON.stringify(patch),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`patchGameSafety failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as GameSafetySettings
}

export async function renameLiveGamePlayer(
  courseCode: string,
  gameId: string,
  playerId: string,
  nickname?: string,
): Promise<{ playerId: string; nickname: string }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/players/${encodeURIComponent(playerId)}/rename`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(nickname ? { nickname } : {}),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`renameLiveGamePlayer failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as { playerId: string; nickname: string }
}

export async function flagLiveGameContent(
  courseCode: string,
  gameId: string,
  body: { playerId?: string; questionIndex?: number; reason: string },
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/flag`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify(body),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`flagLiveGameContent failed (${res.status}): ${txt}`)
  }
}

export type IntegrityFlag = { kind: string; playerId?: string; detail: string }

export async function fetchGameSafetyEvents(
  courseCode: string,
  gameId: string,
): Promise<{ events: unknown[]; integrityFlags: IntegrityFlag[] }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/safety-events`,
    { headers: await authHeaders(false) },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`fetchGameSafetyEvents failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as { events: unknown[]; integrityFlags: IntegrityFlag[] }
}

// --- Admin governance (IQ.11) ---

export type IQGuestJoinPolicy = 'disabled' | 'teacher_mediated' | 'open'
export type IQDefaultMode = 'live_classic' | 'team' | 'student_paced' | 'homework'
export type IQLeaderboardPrivacy = 'names' | 'anon_to_peers' | 'anonymous'

export type InteractiveQuizPlatformSettings = {
  maxConcurrentGames: number | null
  maxPlayersPerGame: number
  maxKitsPerCourse: number | null
  retentionDays: number
  guestJoinPolicy: IQGuestJoinPolicy
  defaultMode: IQDefaultMode
  defaultLeaderboardPrivacy: IQLeaderboardPrivacy
  aiGenerationEnabled: boolean
  aiGenerationsPerDay: number | null
}

export type InteractiveQuizAnalytics = {
  games: number
  gamesByMode: Record<string, number>
  uniqueHosts: number
  uniquePlayers: number
  answersSubmitted: number
  avgParticipation: number
  guestPlayers: number
  enrolledPlayers: number
  aiCostCents: number
  coursesUsing: number
  pendingReviewCount: number
  liveGamesNow: number
  daily: Array<{
    day: string
    orgId: string
    courseId: string
    games: number
    players: number
    answers: number
    guestPlayers: number
    enrolledPlayers: number
    aiCostCents: number
  }>
  from: string
  to: string
  orgId?: string
}

export type InteractiveQuizReviewItem = {
  id: string
  kind: 'catalog_submission' | 'reported_content'
  kitId?: string | null
  sessionId?: string | null
  detail: Record<string, unknown>
  status: 'pending' | 'approved' | 'rejected' | 'actioned'
  reviewerId?: string | null
  reason?: string | null
  createdAt: string
  reviewedAt?: string | null
  kitTitle?: string
  submitterId?: string | null
}

export type InteractiveQuizLiveGame = {
  id: string
  courseCode: string
  joinCode?: string
  status: string
  mode: string
  hostId?: string | null
  players: number
  createdAt: string
  startedAt?: string | null
}

export async function fetchAdminIQSettings(): Promise<InteractiveQuizPlatformSettings> {
  const res = await fetch(`${apiBase}/api/v1/admin/settings/interactive-quizzes`, {
    headers: await authHeaders(false),
  })
  if (!res.ok) throw new Error(`fetchAdminIQSettings failed (${res.status})`)
  return (await res.json()) as InteractiveQuizPlatformSettings
}

export async function patchAdminIQSettings(
  body: Partial<InteractiveQuizPlatformSettings> & {
    clearMaxConcurrentGames?: boolean
    clearMaxKitsPerCourse?: boolean
    clearAiGenerationsPerDay?: boolean
  },
): Promise<InteractiveQuizPlatformSettings> {
  const res = await fetch(`${apiBase}/api/v1/admin/settings/interactive-quizzes`, {
    method: 'PATCH',
    headers: await authHeaders(),
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`patchAdminIQSettings failed (${res.status})`)
  return (await res.json()) as InteractiveQuizPlatformSettings
}

export async function fetchAdminIQAnalytics(
  orgId?: string,
  from?: string,
  to?: string,
): Promise<InteractiveQuizAnalytics> {
  const params = new URLSearchParams()
  if (orgId) params.set('orgId', orgId)
  if (from) params.set('from', from)
  if (to) params.set('to', to)
  const q = params.toString() ? `?${params}` : ''
  const res = await fetch(`${apiBase}/api/v1/admin/interactive-quizzes/analytics${q}`, {
    headers: await authHeaders(false),
  })
  if (!res.ok) throw new Error(`fetchAdminIQAnalytics failed (${res.status})`)
  return (await res.json()) as InteractiveQuizAnalytics
}

export async function fetchAdminIQReviewQueue(
  status = 'pending',
): Promise<{ items: InteractiveQuizReviewItem[]; pendingCount: number }> {
  const params = new URLSearchParams({ status })
  const res = await fetch(
    `${apiBase}/api/v1/admin/interactive-quizzes/review-queue?${params}`,
    { headers: await authHeaders(false) },
  )
  if (!res.ok) throw new Error(`fetchAdminIQReviewQueue failed (${res.status})`)
  return (await res.json()) as { items: InteractiveQuizReviewItem[]; pendingCount: number }
}

export async function postAdminIQReviewAction(
  id: string,
  action: 'approve' | 'reject' | 'action' | 'takedown',
  reason?: string,
): Promise<InteractiveQuizReviewItem> {
  const res = await fetch(
    `${apiBase}/api/v1/admin/interactive-quizzes/review-queue/${encodeURIComponent(id)}/${action}`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ reason: reason ?? '' }),
    },
  )
  if (!res.ok) throw new Error(`postAdminIQReviewAction failed (${res.status})`)
  return (await res.json()) as InteractiveQuizReviewItem
}

export async function fetchAdminIQLiveGames(
  orgId?: string,
): Promise<{ games: InteractiveQuizLiveGame[] }> {
  const q = orgId ? `?orgId=${encodeURIComponent(orgId)}` : ''
  const res = await fetch(`${apiBase}/api/v1/admin/interactive-quizzes/games${q}`, {
    headers: await authHeaders(false),
  })
  if (!res.ok) throw new Error(`fetchAdminIQLiveGames failed (${res.status})`)
  return (await res.json()) as { games: InteractiveQuizLiveGame[] }
}

export async function postAdminIQForceEnd(gameId: string): Promise<unknown> {
  const res = await fetch(
    `${apiBase}/api/v1/admin/interactive-quizzes/games/${encodeURIComponent(gameId)}/force-end`,
    { method: 'POST', headers: await authHeaders() },
  )
  if (!res.ok) throw new Error(`postAdminIQForceEnd failed (${res.status})`)
  return res.json()
}

export async function postAdminIQBulkArchiveKits(
  olderThanDays = 365,
  orgId?: string,
): Promise<{ archived: number }> {
  const q = orgId ? `?orgId=${encodeURIComponent(orgId)}` : ''
  const res = await fetch(`${apiBase}/api/v1/admin/interactive-quizzes/kits/bulk-archive${q}`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ olderThanDays }),
  })
  if (!res.ok) throw new Error(`postAdminIQBulkArchiveKits failed (${res.status})`)
  return (await res.json()) as { archived: number }
}

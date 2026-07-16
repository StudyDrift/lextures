import { getAccessToken } from './auth'

const apiBase = import.meta.env.VITE_API_URL ?? ''

export type KitStatus = 'draft' | 'ready' | 'archived'
export type KitVisibility = 'private' | 'course' | 'org' | 'public'

export type QuizKit = {
  id: string
  courseId: string
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
    coverImageRef: raw.coverImageRef ?? null,
    tags: raw.tags ?? [],
    questionCount: raw.questionCount ?? 0,
    status: (raw.status || 'draft') as KitStatus,
    visibility: (raw.visibility || 'course') as KitVisibility,
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

export async function duplicateQuizKit(courseCode: string, kitId: string): Promise<QuizKit> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/duplicate`,
    { method: 'POST', headers: await authHeaders() },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`duplicateQuizKit failed (${res.status}): ${txt}`)
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
  version: number
  createdAt: string
  updatedAt: string
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
  status: string
  phase: string
  pacing: LiveGamePacing
  questionIndex: number
  kitTitle: string
  questionCount: number
  players?: LiveGamePlayer[]
  openedAt?: string
  deadline?: string
}

export async function startLiveGame(
  courseCode: string,
  kitId: string,
  opts?: { pacing?: LiveGamePacing },
): Promise<{ gameId: string; joinCode: string; game: LiveGame }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits/${encodeURIComponent(kitId)}/games`,
    {
      method: 'POST',
      headers: await authHeaders(),
      body: JSON.stringify({ pacing: opts?.pacing ?? 'manual' }),
    },
  )
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`startLiveGame failed (${res.status}): ${txt}`)
  }
  return (await res.json()) as { gameId: string; joinCode: string; game: LiveGame }
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
  if (res.status === 409) {
    throw new LiveQuizJoinError(409, 'nickname_taken')
  }
  if (res.status === 401 || res.status === 403) {
    throw new LiveQuizJoinError(res.status, 'auth_required')
  }
  if (res.status === 400) {
    const txt = await res.text()
    if (txt.toLowerCase().includes('ended')) {
      throw new LiveQuizJoinError(400, 'game_ended')
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
    throw new LiveQuizJoinError(res.status, `joinLiveGame failed (${res.status})`)
  }
  return (await res.json()) as JoinPlayerResult
}

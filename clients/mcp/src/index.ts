#!/usr/bin/env node
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import { z } from 'zod'

const apiUrl = (process.env.LEXTURES_API_URL ?? 'http://localhost:8080').replace(/\/$/, '')
const apiToken = process.env.LEXTURES_API_TOKEN?.trim() ?? ''

const MAX_FILE_BYTES = 1_048_576

const courseCodeParam = z.string().min(1).describe('Course code (e.g. CS101)')
const itemIdParam = z.string().uuid().describe('Structure or file item UUID')
const moduleIdParam = z.string().uuid().describe('Module structure item UUID')
const instanceIdParam = z.string().uuid().describe('Content tool instance UUID')

const contentToolHostKind = z
  .enum(['content_page', 'assignment', 'quiz', 'syllabus', 'portfolio_artifact'])
  .describe('Host surface for the content tool instance')

function requireToken(): void {
  if (!apiToken) {
    throw new Error('LEXTURES_API_TOKEN is required')
  }
}

async function apiRequest(
  method: string,
  path: string,
  body?: unknown,
): Promise<unknown> {
  requireToken()
  const headers: Record<string, string> = {
    Authorization: `Bearer ${apiToken}`,
    Accept: 'application/json',
  }
  let payload: string | undefined
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }
  const res = await fetch(`${apiUrl}${path}`, {
    method,
    headers,
    body: payload,
  })
  const text = await res.text()
  if (!res.ok) {
    throw new Error(`API ${res.status}: ${text.slice(0, 400)}`)
  }
  return text ? JSON.parse(text) : {}
}

async function apiGet(path: string): Promise<unknown> {
  return apiRequest('GET', path)
}

async function apiPost(path: string, body?: unknown): Promise<unknown> {
  return apiRequest('POST', path, body ?? {})
}

async function apiPatch(path: string, body: unknown): Promise<unknown> {
  return apiRequest('PATCH', path, body)
}

async function apiGetRaw(path: string): Promise<{ body: Uint8Array; contentType: string }> {
  requireToken()
  const res = await fetch(`${apiUrl}${path}`, {
    headers: {
      Authorization: `Bearer ${apiToken}`,
    },
    redirect: 'follow',
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`API ${res.status}: ${text.slice(0, 400)}`)
  }
  const buf = new Uint8Array(await res.arrayBuffer())
  return {
    body: buf,
    contentType: res.headers.get('content-type') ?? 'application/octet-stream',
  }
}

function jsonToolResult(data: unknown) {
  return {
    content: [{ type: 'text' as const, text: JSON.stringify(data, null, 2) }],
  }
}

function encCourse(courseCode: string): string {
  return encodeURIComponent(courseCode)
}

function encId(id: string): string {
  return encodeURIComponent(id)
}

/** Stable-key ```lex-tool fence used to embed a content tool instance in markdown. */
function serializeLexToolFence(instanceId: string, toolId: string): string {
  const body = `{"instanceId":${JSON.stringify(instanceId)},"toolId":${JSON.stringify(toolId)},"v":1}`
  return `\`\`\`lex-tool\n${body}\n\`\`\``
}

async function fetchStructureItems(courseCode: string): Promise<StructureItem[]> {
  const data = (await apiGet(`/api/v1/courses/${encCourse(courseCode)}/structure`)) as {
    items?: StructureItem[]
  }
  return data.items ?? []
}

function mapStructureSummary(item: StructureItem) {
  return {
    itemId: item.id,
    kind: item.kind,
    title: item.title,
    parentId: item.parentId ?? null,
    published: item.published ?? false,
    dueAt: item.dueAt ?? null,
    pointsWorth: item.pointsWorth ?? null,
    assignmentGroupId: item.assignmentGroupId ?? null,
  }
}

/**
 * Assignment PATCH is not a true partial update (non-pointer bools reset).
 * Merge existing fields so we do not wipe grading settings (same approach as the CLI).
 */
function buildAssignmentPatch(
  current: Record<string, unknown>,
  overrides: Record<string, unknown>,
): Record<string, unknown> {
  const str = (key: string, fallback: string) => {
    const v = current[key]
    return typeof v === 'string' && v !== '' ? v : fallback
  }
  const bool = (key: string, fallback: boolean) => {
    const v = current[key]
    return typeof v === 'boolean' ? v : fallback
  }
  const patch: Record<string, unknown> = {
    markdown: typeof current.markdown === 'string' ? current.markdown : '',
    lateSubmissionPolicy: str('lateSubmissionPolicy', 'allow'),
    postingPolicy: str('postingPolicy', 'automatic'),
    blindGrading: bool('blindGrading', false),
    moderatedGrading: bool('moderatedGrading', false),
    neverDrop: bool('neverDrop', false),
    replaceWithFinal: bool('replaceWithFinal', false),
  }
  for (const key of [
    'dueAt',
    'pointsWorth',
    'availableFrom',
    'availableUntil',
    'assignmentGroupId',
    'latePenaltyPercent',
    'originalityDetection',
    'originalityStudentVisibility',
    'gradingType',
    'releaseAt',
    'submissionAllowText',
    'submissionAllowFileUpload',
    'submissionAllowUrl',
  ]) {
    if (current[key] !== undefined && current[key] !== null) {
      patch[key] = current[key]
    }
  }
  for (const [key, value] of Object.entries(overrides)) {
    if (value !== undefined) {
      patch[key] = value
    }
  }
  return patch
}

async function getAssignmentRecord(
  courseCode: string,
  itemId: string,
): Promise<Record<string, unknown>> {
  const data = await apiGet(
    `/api/v1/courses/${encCourse(courseCode)}/assignments/${encId(itemId)}`,
  )
  return (data && typeof data === 'object' ? data : {}) as Record<string, unknown>
}

type StructureItem = {
  id: string
  kind: string
  title: string
  parentId?: string | null
  published?: boolean
  dueAt?: string | null
  pointsWorth?: number | null
  assignmentGroupId?: string | null
  archived?: boolean
}

type FeedChannel = {
  id: string
  name: string
}

type FeedMessage = {
  id: string
  channelId: string
  authorUserId: string
  authorEmail: string
  authorDisplayName?: string | null
  body: string
  createdAt: string
  editedAt?: string | null
  pinnedAt?: string | null
  likeCount?: number
  replies?: FeedMessage[]
}

function filterFeedMessages(messages: FeedMessage[], sinceMs: number): FeedMessage[] {
  const out: FeedMessage[] = []
  for (const msg of messages) {
    const createdMs = Date.parse(msg.createdAt)
    const replies = msg.replies ? filterFeedMessages(msg.replies, sinceMs) : []
    if (createdMs >= sinceMs || replies.length > 0) {
      out.push({ ...msg, replies })
    }
  }
  return out
}

function flattenFeedMessages(
  messages: FeedMessage[],
  channel: FeedChannel,
  out: Array<FeedMessage & { channelName: string }> = [],
): Array<FeedMessage & { channelName: string }> {
  for (const msg of messages) {
    out.push({ ...msg, channelName: channel.name })
    if (msg.replies?.length) {
      flattenFeedMessages(msg.replies, channel, out)
    }
  }
  return out
}

function isTextContentType(contentType: string): boolean {
  const ct = contentType.toLowerCase().split(';')[0]?.trim() ?? ''
  return (
    ct.startsWith('text/') ||
    ct === 'application/json' ||
    ct === 'application/xml' ||
    ct === 'application/javascript' ||
    ct === 'application/markdown' ||
    ct === 'application/xhtml+xml'
  )
}

function decodeFileContent(body: Uint8Array, contentType: string): Record<string, unknown> {
  if (body.byteLength > MAX_FILE_BYTES) {
    return {
      contentType,
      byteSize: body.byteLength,
      truncated: true,
      note: `File exceeds ${MAX_FILE_BYTES} bytes; content omitted.`,
    }
  }
  if (isTextContentType(contentType)) {
    return {
      contentType,
      byteSize: body.byteLength,
      encoding: 'utf-8',
      content: new TextDecoder('utf-8', { fatal: false }).decode(body),
    }
  }
  return {
    contentType,
    byteSize: body.byteLength,
    encoding: 'base64',
    content: Buffer.from(body).toString('base64'),
    note: 'Binary file returned as base64.',
  }
}

const server = new McpServer({
  name: 'lextures',
  version: '0.3.0',
})

server.tool(
  'list_courses',
  'List courses visible to the authenticated user',
  {
    termId: z.string().uuid().optional().describe('Optional academic term UUID filter'),
  },
  async ({ termId }) => {
    const q = termId ? `?term_id=${encodeURIComponent(termId)}` : ''
    const data = await apiGet(`/api/v1/courses${q}`)
    return jsonToolResult(data)
  },
)

server.tool(
  'whoami',
  'Return the authenticated Lextures user profile',
  {},
  async () => {
    const data = await apiGet('/api/v1/me')
    return jsonToolResult(data)
  },
)

server.tool(
  'list_structure',
  'List course structure items (modules, content pages, assignments, quizzes, headings, etc.)',
  {
    courseCode: courseCodeParam,
    kind: z
      .string()
      .optional()
      .describe(
        'Optional kind filter: module, content_page, assignment, quiz, heading, external_link, etc.',
      ),
  },
  async ({ courseCode, kind }) => {
    const items = await fetchStructureItems(courseCode)
    const filtered = items
      .filter((item) => !item.archived)
      .filter((item) => (kind ? item.kind === kind : true))
      .map(mapStructureSummary)
    return jsonToolResult({ courseCode, kind: kind ?? null, items: filtered })
  },
)

server.tool(
  'list_assignments',
  'List assignments in a course (metadata from course structure; use read_assignment for full content)',
  {
    courseCode: courseCodeParam,
  },
  async ({ courseCode }) => {
    const assignments = (await fetchStructureItems(courseCode))
      .filter((item) => item.kind === 'assignment' && !item.archived)
      .map(mapStructureSummary)
    return jsonToolResult({ courseCode, assignments })
  },
)

server.tool(
  'list_quizzes',
  'List quizzes in a course (metadata from course structure; use read_quiz for full content)',
  {
    courseCode: courseCodeParam,
  },
  async ({ courseCode }) => {
    const quizzes = (await fetchStructureItems(courseCode))
      .filter((item) => item.kind === 'quiz' && !item.archived)
      .map(mapStructureSummary)
    return jsonToolResult({ courseCode, quizzes })
  },
)

server.tool(
  'list_content_pages',
  'List content pages (learning content activities) in a course',
  {
    courseCode: courseCodeParam,
  },
  async ({ courseCode }) => {
    const contentPages = (await fetchStructureItems(courseCode))
      .filter((item) => item.kind === 'content_page' && !item.archived)
      .map(mapStructureSummary)
    return jsonToolResult({ courseCode, contentPages })
  },
)

server.tool(
  'list_enrollments',
  'List enrollments (roster) for a course',
  {
    courseCode: courseCodeParam,
  },
  async ({ courseCode }) => {
    const data = await apiGet(`/api/v1/courses/${encCourse(courseCode)}/enrollments`)
    return jsonToolResult(data)
  },
)

server.tool(
  'list_activity_feed',
  'List course activity feed messages from the last N days across all channels',
  {
    courseCode: courseCodeParam,
    days: z
      .number()
      .int()
      .positive()
      .max(365)
      .describe('Include messages created within this many days (UTC)'),
  },
  async ({ courseCode, days }) => {
    const since = new Date()
    since.setUTCDate(since.getUTCDate() - days)
    const sinceMs = since.getTime()

    const channelsData = (await apiGet(`/api/v1/courses/${encCourse(courseCode)}/feed/channels`)) as {
      channels?: FeedChannel[]
    }
    const channels = channelsData.channels ?? []

    const messages: Array<FeedMessage & { channelName: string }> = []
    for (const channel of channels) {
      const channelMessagesData = (await apiGet(
        `/api/v1/courses/${encCourse(courseCode)}/feed/channels/${encodeURIComponent(channel.id)}/messages`,
      )) as { messages?: FeedMessage[] }
      const filtered = filterFeedMessages(channelMessagesData.messages ?? [], sinceMs)
      flattenFeedMessages(filtered, channel, messages)
    }

    messages.sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt))

    return jsonToolResult({
      courseCode,
      days,
      since: since.toISOString(),
      messageCount: messages.length,
      messages,
    })
  },
)

server.tool(
  'list_files',
  'List files and folders in a course file space (root or a folder)',
  {
    courseCode: courseCodeParam,
    folderId: z.string().uuid().optional().describe('Optional folder UUID; omit for course root'),
  },
  async ({ courseCode, folderId }) => {
    const path = folderId
      ? `/api/v1/courses/${encCourse(courseCode)}/files/folders/${encodeURIComponent(folderId)}`
      : `/api/v1/courses/${encCourse(courseCode)}/files`
    const data = await apiGet(path)
    return jsonToolResult(data)
  },
)

server.tool(
  'read_file',
  'Download a course file item by id (text as UTF-8, binary as base64; large files are truncated)',
  {
    courseCode: courseCodeParam,
    itemId: itemIdParam,
  },
  async ({ courseCode, itemId }) => {
    const { body, contentType } = await apiGetRaw(
      `/api/v1/courses/${encCourse(courseCode)}/files/items/${encodeURIComponent(itemId)}/content`,
    )
    return jsonToolResult({
      courseCode,
      itemId,
      ...decodeFileContent(body, contentType),
    })
  },
)

server.tool(
  'read_assignment',
  'Read an assignment including markdown content and metadata (due date, points, availability, etc.)',
  {
    courseCode: courseCodeParam,
    itemId: itemIdParam,
  },
  async ({ courseCode, itemId }) => {
    const data = await apiGet(
      `/api/v1/courses/${encCourse(courseCode)}/assignments/${encId(itemId)}`,
    )
    return jsonToolResult(data)
  },
)

server.tool(
  'read_quiz',
  'Read a quiz including questions, settings, and markdown intro',
  {
    courseCode: courseCodeParam,
    itemId: itemIdParam,
  },
  async ({ courseCode, itemId }) => {
    const data = await apiGet(`/api/v1/courses/${encCourse(courseCode)}/quizzes/${encId(itemId)}`)
    return jsonToolResult(data)
  },
)

server.tool(
  'read_content_page',
  'Read a content page (markdown body and metadata)',
  {
    courseCode: courseCodeParam,
    itemId: itemIdParam,
  },
  async ({ courseCode, itemId }) => {
    const data = await apiGet(
      `/api/v1/courses/${encCourse(courseCode)}/content-pages/${encId(itemId)}`,
    )
    return jsonToolResult(data)
  },
)

// --- Write: modules & structure items ---

server.tool(
  'create_module',
  'Create a module in a course',
  {
    courseCode: courseCodeParam,
    title: z.string().min(1).describe('Module title'),
  },
  async ({ courseCode, title }) => {
    const data = await apiPost(`/api/v1/courses/${encCourse(courseCode)}/structure/modules`, {
      title,
    })
    return jsonToolResult(data)
  },
)

server.tool(
  'create_content_page',
  'Create a content page under a module; optionally set markdown body and publish',
  {
    courseCode: courseCodeParam,
    moduleId: moduleIdParam,
    title: z.string().min(1).describe('Content page title'),
    markdown: z
      .string()
      .optional()
      .describe('Optional markdown body (can include ```lex-tool fences for content tools)'),
    published: z.boolean().optional().describe('If true, publish the page after creation'),
  },
  async ({ courseCode, moduleId, title, markdown, published }) => {
    const created = (await apiPost(
      `/api/v1/courses/${encCourse(courseCode)}/structure/modules/${encId(moduleId)}/content-pages`,
      { title },
    )) as { id?: string; title?: string }

    const itemId = created.id
    if (!itemId) {
      return jsonToolResult(created)
    }

    let page: unknown = created
    if (markdown !== undefined) {
      page = await apiPatch(
        `/api/v1/courses/${encCourse(courseCode)}/content-pages/${encId(itemId)}`,
        { markdown },
      )
    }

    if (published) {
      await apiPatch(`/api/v1/courses/${encCourse(courseCode)}/structure/items/${encId(itemId)}`, {
        published: true,
      })
    }

    return jsonToolResult({
      itemId,
      title: created.title ?? title,
      published: published ?? false,
      page,
    })
  },
)

server.tool(
  'update_content_page',
  'Update a content page markdown body (and optional due date)',
  {
    courseCode: courseCodeParam,
    itemId: itemIdParam,
    markdown: z.string().describe('Full markdown body to save'),
    dueAt: z
      .string()
      .nullable()
      .optional()
      .describe('Optional due date (RFC3339) or null to clear'),
  },
  async ({ courseCode, itemId, markdown, dueAt }) => {
    const body: Record<string, unknown> = { markdown }
    if (dueAt !== undefined) {
      body.dueAt = dueAt
    }
    const data = await apiPatch(
      `/api/v1/courses/${encCourse(courseCode)}/content-pages/${encId(itemId)}`,
      body,
    )
    return jsonToolResult(data)
  },
)

server.tool(
  'create_assignment',
  'Create an assignment under a module; optionally set markdown, points, due date, and publish',
  {
    courseCode: courseCodeParam,
    moduleId: moduleIdParam,
    title: z.string().min(1).describe('Assignment title'),
    markdown: z.string().optional().describe('Assignment instructions (markdown)'),
    pointsWorth: z.number().int().min(0).optional().describe('Point value'),
    dueAt: z.string().optional().describe('Due date (RFC3339 or YYYY-MM-DD)'),
    published: z.boolean().optional().describe('If true, publish after creation'),
    submissionAllowText: z.boolean().optional(),
    submissionAllowFileUpload: z.boolean().optional(),
    submissionAllowUrl: z.boolean().optional(),
  },
  async ({
    courseCode,
    moduleId,
    title,
    markdown,
    pointsWorth,
    dueAt,
    published,
    submissionAllowText,
    submissionAllowFileUpload,
    submissionAllowUrl,
  }) => {
    const created = (await apiPost(
      `/api/v1/courses/${encCourse(courseCode)}/structure/modules/${encId(moduleId)}/assignments`,
      { title },
    )) as { id?: string; title?: string }

    const itemId = created.id
    if (!itemId) {
      return jsonToolResult(created)
    }

    const needsPatch =
      markdown !== undefined ||
      pointsWorth !== undefined ||
      dueAt !== undefined ||
      submissionAllowText !== undefined ||
      submissionAllowFileUpload !== undefined ||
      submissionAllowUrl !== undefined

    let assignment: unknown = created
    if (needsPatch) {
      const overrides: Record<string, unknown> = {
        markdown: markdown ?? '',
      }
      if (pointsWorth !== undefined) overrides.pointsWorth = pointsWorth
      if (dueAt !== undefined) overrides.dueAt = dueAt
      if (submissionAllowText !== undefined) overrides.submissionAllowText = submissionAllowText
      if (submissionAllowFileUpload !== undefined) {
        overrides.submissionAllowFileUpload = submissionAllowFileUpload
      }
      if (submissionAllowUrl !== undefined) overrides.submissionAllowUrl = submissionAllowUrl
      const patch = buildAssignmentPatch({}, overrides)
      assignment = await apiPatch(
        `/api/v1/courses/${encCourse(courseCode)}/assignments/${encId(itemId)}`,
        patch,
      )
    }

    if (published) {
      await apiPatch(`/api/v1/courses/${encCourse(courseCode)}/structure/items/${encId(itemId)}`, {
        published: true,
      })
    }

    return jsonToolResult({
      itemId,
      title: created.title ?? title,
      published: published ?? false,
      assignment,
    })
  },
)

server.tool(
  'update_assignment',
  'Update an assignment (markdown, points, due date, submission types, policies)',
  {
    courseCode: courseCodeParam,
    itemId: itemIdParam,
    markdown: z.string().optional().describe('Assignment instructions (markdown)'),
    pointsWorth: z.number().int().min(0).nullable().optional(),
    dueAt: z.string().nullable().optional().describe('RFC3339 due date or null to clear'),
    availableFrom: z.string().nullable().optional(),
    availableUntil: z.string().nullable().optional(),
    submissionAllowText: z.boolean().optional(),
    submissionAllowFileUpload: z.boolean().optional(),
    submissionAllowUrl: z.boolean().optional(),
    lateSubmissionPolicy: z.enum(['allow', 'penalty', 'block']).optional(),
    latePenaltyPercent: z.number().int().min(0).max(100).nullable().optional(),
    postingPolicy: z.enum(['automatic', 'manual']).optional(),
  },
  async ({ courseCode, itemId, ...fields }) => {
    const overrides: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(fields)) {
      if (value !== undefined) overrides[key] = value
    }
    if (Object.keys(overrides).length === 0) {
      throw new Error('Provide at least one field to update')
    }
    const current = await getAssignmentRecord(courseCode, itemId)
    const patch = buildAssignmentPatch(current, overrides)
    const data = await apiPatch(
      `/api/v1/courses/${encCourse(courseCode)}/assignments/${encId(itemId)}`,
      patch,
    )
    return jsonToolResult(data)
  },
)

server.tool(
  'create_quiz',
  'Create a quiz under a module; optionally set intro markdown, points, questions, and publish',
  {
    courseCode: courseCodeParam,
    moduleId: moduleIdParam,
    title: z.string().min(1).describe('Quiz title'),
    markdown: z.string().optional().describe('Quiz intro / instructions (markdown)'),
    pointsWorth: z.number().int().min(0).optional(),
    questions: z
      .array(z.record(z.unknown()))
      .optional()
      .describe(
        'Optional question objects: { id, prompt, questionType, choices?, correctChoiceIndex?, points, ... }. Types: multiple_choice, true_false, short_answer, essay, fill_in_blank, matching, ordering, numeric, code, file_upload, etc.',
      ),
    published: z.boolean().optional(),
  },
  async ({ courseCode, moduleId, title, markdown, pointsWorth, questions, published }) => {
    const created = (await apiPost(
      `/api/v1/courses/${encCourse(courseCode)}/structure/modules/${encId(moduleId)}/quizzes`,
      { title },
    )) as { id?: string; title?: string }

    const itemId = created.id
    if (!itemId) {
      return jsonToolResult(created)
    }

    const needsPatch =
      markdown !== undefined || pointsWorth !== undefined || questions !== undefined

    let quiz: unknown = created
    if (needsPatch) {
      const patch: Record<string, unknown> = {}
      if (markdown !== undefined) patch.markdown = markdown
      if (pointsWorth !== undefined) patch.pointsWorth = pointsWorth
      if (questions !== undefined) patch.questions = questions
      quiz = await apiPatch(
        `/api/v1/courses/${encCourse(courseCode)}/quizzes/${encId(itemId)}`,
        patch,
      )
    }

    if (published) {
      await apiPatch(`/api/v1/courses/${encCourse(courseCode)}/structure/items/${encId(itemId)}`, {
        published: true,
      })
    }

    return jsonToolResult({
      itemId,
      title: created.title ?? title,
      published: published ?? false,
      quiz,
    })
  },
)

server.tool(
  'update_quiz',
  'Modify a quiz: title, intro markdown, questions, attempts, shuffle, time limit, availability, points, etc.',
  {
    courseCode: courseCodeParam,
    itemId: itemIdParam,
    title: z.string().optional(),
    markdown: z.string().optional(),
    questions: z
      .array(z.record(z.unknown()))
      .optional()
      .describe(
        'Full replacement question list. Each: { id, prompt, questionType, choices?, correctChoiceIndex?, points, multipleAnswer?, required? }',
      ),
    pointsWorth: z.number().int().min(0).nullable().optional(),
    dueAt: z.string().nullable().optional(),
    availableFrom: z.string().nullable().optional(),
    availableUntil: z.string().nullable().optional(),
    unlimitedAttempts: z.boolean().optional(),
    maxAttempts: z.number().int().min(1).optional(),
    gradeAttemptPolicy: z.enum(['highest', 'latest', 'first', 'average']).optional(),
    timeLimitMinutes: z.number().int().min(0).nullable().optional(),
    shuffleQuestions: z.boolean().optional(),
    shuffleChoices: z.boolean().optional(),
    oneQuestionAtATime: z.boolean().optional(),
    allowBackNavigation: z.boolean().optional(),
    showScoreTiming: z.enum(['immediate', 'after_due', 'manual']).optional(),
    reviewVisibility: z
      .enum(['none', 'score_only', 'responses', 'correct_answers', 'full'])
      .optional(),
    reviewWhen: z.enum(['after_submit', 'after_due', 'always', 'never']).optional(),
  },
  async ({ courseCode, itemId, ...fields }) => {
    const patch: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(fields)) {
      if (value !== undefined) patch[key] = value
    }
    if (Object.keys(patch).length === 0) {
      throw new Error('Provide at least one field to update')
    }
    const data = await apiPatch(
      `/api/v1/courses/${encCourse(courseCode)}/quizzes/${encId(itemId)}`,
      patch,
    )
    return jsonToolResult(data)
  },
)

server.tool(
  'publish_structure_item',
  'Publish or unpublish a structure item (content page, assignment, quiz, module, etc.)',
  {
    courseCode: courseCodeParam,
    itemId: itemIdParam,
    published: z.boolean().describe('true to publish, false to unpublish'),
    title: z.string().optional().describe('Optional title update'),
  },
  async ({ courseCode, itemId, published, title }) => {
    const body: Record<string, unknown> = { published }
    if (title !== undefined) body.title = title
    const data = await apiPatch(
      `/api/v1/courses/${encCourse(courseCode)}/structure/items/${encId(itemId)}`,
      body,
    )
    return jsonToolResult(data)
  },
)

// --- Content tools (inline questions, flashcards, etc.) ---

server.tool(
  'list_content_tool_catalog',
  'List content tools available in a course (inline_questions, flashcards, predict_reveal, etc.)',
  {
    courseCode: courseCodeParam,
  },
  async ({ courseCode }) => {
    const data = await apiGet(`/api/v1/courses/${encCourse(courseCode)}/content-tools/catalog`)
    return jsonToolResult(data)
  },
)

server.tool(
  'get_content_tool_manifest',
  'Get a content tool manifest including configSchema (settings shape, e.g. for inline_questions)',
  {
    courseCode: courseCodeParam,
    toolId: z
      .string()
      .min(1)
      .describe(
        'Tool id (e.g. inline_questions, flashcards, predict_reveal, ask_questions, code_sandbox)',
      ),
  },
  async ({ courseCode, toolId }) => {
    const data = await apiGet(
      `/api/v1/courses/${encCourse(courseCode)}/content-tools/manifests/${encId(toolId)}`,
    )
    return jsonToolResult(data)
  },
)

server.tool(
  'list_content_tool_instances',
  'List content tool instances in a course, optionally filtered by structure item',
  {
    courseCode: courseCodeParam,
    itemId: z
      .string()
      .uuid()
      .optional()
      .describe('Optional structure item UUID (content page / assignment / quiz)'),
    hostKind: contentToolHostKind.optional(),
    withState: z
      .boolean()
      .optional()
      .describe('If true, include the authenticated learner state for each instance'),
  },
  async ({ courseCode, itemId, hostKind, withState }) => {
    const q = new URLSearchParams()
    if (itemId) q.set('itemId', itemId)
    if (hostKind) q.set('hostKind', hostKind)
    if (withState) q.set('withState', '1')
    const qs = q.toString()
    const path = `/api/v1/courses/${encCourse(courseCode)}/content-tools/instances${qs ? `?${qs}` : ''}`
    const data = await apiGet(path)
    return jsonToolResult(data)
  },
)

server.tool(
  'create_content_tool_instance',
  'Create a content tool instance with settings (config). For content_page/assignment/quiz hosts, pass structureItemId. Does not insert a markdown fence — use attach_content_tool for that.',
  {
    courseCode: courseCodeParam,
    toolId: z
      .string()
      .min(1)
      .describe('Tool id (e.g. inline_questions)'),
    hostKind: contentToolHostKind,
    structureItemId: z
      .string()
      .uuid()
      .optional()
      .describe('Required except for hostKind=syllabus'),
    title: z.string().optional().describe('Optional instance title'),
    sectionKey: z.string().optional().describe('Optional section key within the host body'),
    config: z
      .record(z.unknown())
      .describe(
        'Tool settings object validated against the tool configSchema. For inline_questions: { questions: [{ id, type, prompt, options?, ... }], attempts?, revealCorrectAfter?, questionsAtATime?, ... }',
      ),
  },
  async ({ courseCode, toolId, hostKind, structureItemId, title, sectionKey, config }) => {
    const body: Record<string, unknown> = {
      toolId,
      hostKind,
      config,
    }
    if (structureItemId) body.structureItemId = structureItemId
    if (title !== undefined) body.title = title
    if (sectionKey !== undefined) body.sectionKey = sectionKey

    const data = await apiPost(
      `/api/v1/courses/${encCourse(courseCode)}/content-tools/instances`,
      body,
    )
    return jsonToolResult(data)
  },
)

server.tool(
  'update_content_tool_instance',
  'Update a content tool instance title, section key, config (settings), or status',
  {
    courseCode: courseCodeParam,
    instanceId: instanceIdParam,
    title: z.string().nullable().optional(),
    sectionKey: z.string().nullable().optional(),
    config: z
      .record(z.unknown())
      .optional()
      .describe('Full replacement config object (must satisfy the tool configSchema)'),
    status: z.enum(['active', 'archived']).optional(),
  },
  async ({ courseCode, instanceId, title, sectionKey, config, status }) => {
    const body: Record<string, unknown> = {}
    if (title !== undefined) body.title = title
    if (sectionKey !== undefined) body.sectionKey = sectionKey
    if (config !== undefined) body.config = config
    if (status !== undefined) body.status = status
    if (Object.keys(body).length === 0) {
      throw new Error('Provide at least one field to update')
    }
    const data = await apiPatch(
      `/api/v1/courses/${encCourse(courseCode)}/content-tools/instances/${encId(instanceId)}`,
      body,
    )
    return jsonToolResult(data)
  },
)

server.tool(
  'attach_content_tool',
  'Create a content tool instance on a content activity and append a ```lex-tool fence to the host markdown so learners see it. Supports content_page, assignment, and quiz hosts.',
  {
    courseCode: courseCodeParam,
    structureItemId: z.string().uuid().describe('Host content page, assignment, or quiz item UUID'),
    hostKind: z
      .enum(['content_page', 'assignment', 'quiz'])
      .describe('Host kind of the structure item'),
    toolId: z.string().min(1).describe('Tool id (e.g. inline_questions)'),
    title: z.string().optional(),
    config: z
      .record(z.unknown())
      .describe('Tool settings (configSchema). Required for tools like inline_questions.'),
    placement: z
      .enum(['append', 'prepend'])
      .optional()
      .describe('Where to insert the fence in the markdown body (default append)'),
  },
  async ({ courseCode, structureItemId, hostKind, toolId, title, config, placement }) => {
    const instance = (await apiPost(
      `/api/v1/courses/${encCourse(courseCode)}/content-tools/instances`,
      {
        toolId,
        hostKind,
        structureItemId,
        title,
        config,
      },
    )) as { id?: string; toolId?: string; config?: unknown }

    const instanceId = instance.id
    if (!instanceId) {
      return jsonToolResult({ instance, note: 'Instance created but id missing; fence not inserted.' })
    }

    const fence = serializeLexToolFence(instanceId, instance.toolId ?? toolId)
    const pathSegment =
      hostKind === 'content_page'
        ? 'content-pages'
        : hostKind === 'assignment'
          ? 'assignments'
          : 'quizzes'

    const current = (await apiGet(
      `/api/v1/courses/${encCourse(courseCode)}/${pathSegment}/${encId(structureItemId)}`,
    )) as { markdown?: string }

    const existing = current.markdown ?? ''
    const joined =
      placement === 'prepend'
        ? `${fence}\n\n${existing}`.trim()
        : `${existing.trimEnd()}\n\n${fence}\n`.trimStart()

    let updatedHost: unknown
    if (hostKind === 'content_page') {
      updatedHost = await apiPatch(
        `/api/v1/courses/${encCourse(courseCode)}/content-pages/${encId(structureItemId)}`,
        { markdown: joined },
      )
    } else if (hostKind === 'assignment') {
      const currentAssign = await getAssignmentRecord(courseCode, structureItemId)
      updatedHost = await apiPatch(
        `/api/v1/courses/${encCourse(courseCode)}/assignments/${encId(structureItemId)}`,
        buildAssignmentPatch(currentAssign, { markdown: joined }),
      )
    } else {
      updatedHost = await apiPatch(
        `/api/v1/courses/${encCourse(courseCode)}/quizzes/${encId(structureItemId)}`,
        { markdown: joined },
      )
    }

    return jsonToolResult({
      instance,
      fence,
      host: updatedHost,
    })
  },
)

async function main() {
  const transport = new StdioServerTransport()
  await server.connect(transport)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
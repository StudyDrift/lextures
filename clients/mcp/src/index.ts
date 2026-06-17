#!/usr/bin/env node
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import { z } from 'zod'

const apiUrl = (process.env.LEXTURES_API_URL ?? 'http://localhost:8080').replace(/\/$/, '')
const apiToken = process.env.LEXTURES_API_TOKEN?.trim() ?? ''

const MAX_FILE_BYTES = 1_048_576

const courseCodeParam = z.string().min(1).describe('Course code (e.g. CS101)')
const itemIdParam = z.string().uuid().describe('Structure or file item UUID')

function requireToken(): void {
  if (!apiToken) {
    throw new Error('LEXTURES_API_TOKEN is required')
  }
}

async function apiGet(path: string): Promise<unknown> {
  requireToken()
  const res = await fetch(`${apiUrl}${path}`, {
    headers: {
      Authorization: `Bearer ${apiToken}`,
      Accept: 'application/json',
    },
  })
  const text = await res.text()
  if (!res.ok) {
    throw new Error(`API ${res.status}: ${text.slice(0, 400)}`)
  }
  return text ? JSON.parse(text) : {}
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
  version: '0.2.0',
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
  'list_assignments',
  'List assignments in a course (metadata from course structure; use read_assignment for full content)',
  {
    courseCode: courseCodeParam,
  },
  async ({ courseCode }) => {
    const data = (await apiGet(`/api/v1/courses/${encCourse(courseCode)}/structure`)) as {
      items?: StructureItem[]
    }
    const assignments = (data.items ?? [])
      .filter((item) => item.kind === 'assignment' && !item.archived)
      .map((item) => ({
        itemId: item.id,
        title: item.title,
        parentId: item.parentId ?? null,
        published: item.published ?? false,
        dueAt: item.dueAt ?? null,
        pointsWorth: item.pointsWorth ?? null,
        assignmentGroupId: item.assignmentGroupId ?? null,
      }))
    return jsonToolResult({ courseCode, assignments })
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
      `/api/v1/courses/${encCourse(courseCode)}/assignments/${encodeURIComponent(itemId)}`,
    )
    return jsonToolResult(data)
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
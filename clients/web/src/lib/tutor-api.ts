import { authorizedFetch } from './api'

const API_BASE = '/api/v1'

export type TutorCitation = {
  sourceId: string
  chunkId: string
  excerpt: string
  title?: string
}

export type TutorSessionMessage = {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  citations?: TutorCitation[]
  conceptTags?: string[]
  createdAt?: string
}

export type TutorSessionSummary = {
  id: string
  title?: string
  createdAt: string
  lastActive: string
}

export type TutorSessionDetail = TutorSessionSummary & {
  messages: TutorSessionMessage[]
}

export type TutorStreamEvent =
  | { type: 'content'; text: string }
  | { type: 'error'; message: string }
  | { type: 'done'; messageId: string; citations: TutorCitation[] }

export async function fetchTutorSessions(courseCode: string): Promise<TutorSessionSummary[]> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/tutor/sessions`,
  )
  if (res.status === 404) return []
  if (!res.ok) throw new Error(`Failed to load tutor sessions: ${res.status}`)
  return res.json() as Promise<TutorSessionSummary[]>
}

export async function createTutorSession(courseCode: string, title?: string): Promise<TutorSessionSummary> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/tutor/sessions`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(title ? { title } : {}),
    },
  )
  if (!res.ok) throw new Error(`Failed to create tutor session: ${res.status}`)
  return res.json() as Promise<TutorSessionSummary>
}

export async function fetchTutorSession(
  courseCode: string,
  sessionId: string,
): Promise<TutorSessionDetail> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/tutor/sessions/${encodeURIComponent(sessionId)}`,
  )
  if (!res.ok) throw new Error(`Failed to load tutor session: ${res.status}`)
  return res.json() as Promise<TutorSessionDetail>
}

export async function deleteTutorSession(courseCode: string, sessionId: string): Promise<void> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/tutor/sessions/${encodeURIComponent(sessionId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) throw new Error(`Failed to delete tutor session: ${res.status}`)
}

export async function fetchAiTutorOptOut(): Promise<boolean> {
  const res = await authorizedFetch('/api/v1/settings/ai-tutor-opt-out')
  if (res.status === 404) return false
  if (!res.ok) return false
  const data = (await res.json()) as { aiTutorOptOut?: boolean }
  return Boolean(data.aiTutorOptOut)
}

export async function sendTutorSessionMessage(
  courseCode: string,
  sessionId: string,
  content: string,
  onEvent: (ev: TutorStreamEvent) => void,
): Promise<void> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/tutor/sessions/${encodeURIComponent(sessionId)}/messages`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
    },
  )
  if (!res.ok || !res.body) {
    const body = await res.text()
    onEvent({ type: 'error', message: body || `Error ${res.status}` })
    return
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() ?? ''
    for (const line of lines) {
      if (!line.startsWith('data: ')) continue
      try {
        onEvent(JSON.parse(line.slice('data: '.length)) as TutorStreamEvent)
      } catch {
        // skip malformed lines
      }
    }
  }
}

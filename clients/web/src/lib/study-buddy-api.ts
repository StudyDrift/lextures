import { authorizedFetch } from './api'

const API_BASE = '/api/v1'

export type StudyBuddyPrompt = {
  id: string
  kind: string
  message: string
  itemId?: string
}

export type StudyBuddyCitation = {
  itemId: string
  title: string
  excerpt: string
}

export type StudyBuddyMemory = {
  goalsSummary?: string
  struggleConcepts: string[]
  lastSessionSummary?: string
  lastActiveAt?: string
}

export async function fetchStudyBuddyPrompts(courseCode: string): Promise<StudyBuddyPrompt[]> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/study-buddy/prompts`,
  )
  if (res.status === 404) return []
  if (!res.ok) throw new Error(`Failed to load study buddy prompts: ${res.status}`)
  const data = (await res.json()) as { prompts?: StudyBuddyPrompt[] }
  return data.prompts ?? []
}

export async function fetchStudyBuddyMemory(courseCode: string): Promise<StudyBuddyMemory | null> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/study-buddy/memory`,
  )
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`Failed to load study buddy memory: ${res.status}`)
  return res.json() as Promise<StudyBuddyMemory>
}

export async function fetchAiProcessingOptOut(): Promise<boolean> {
  const res = await authorizedFetch('/api/v1/settings/ai-opt-out')
  if (res.status === 404) return false
  if (!res.ok) return false
  const data = (await res.json()) as { aiProcessingOptOut?: boolean }
  return Boolean(data.aiProcessingOptOut)
}

export type StudyBuddyStreamEvent =
  | { type: 'content'; text: string }
  | { type: 'error'; message: string }
  | { type: 'done'; sessionId: string; citations: StudyBuddyCitation[] }

export async function sendStudyBuddyMessage(
  courseCode: string,
  message: string,
  sessionId: string | null,
  onEvent: (ev: StudyBuddyStreamEvent) => void,
): Promise<void> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/study-buddy/message`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message, sessionId: sessionId ?? '' }),
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
      const payload = line.slice('data: '.length)
      try {
        const ev = JSON.parse(payload) as StudyBuddyStreamEvent
        onEvent(ev)
      } catch {
        /* skip malformed SSE lines */
      }
    }
  }
}

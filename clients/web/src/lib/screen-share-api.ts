import { authorizedFetch } from './api'

export type ScreenShareSession = {
  id: string
  courseId: string
  hostId?: string | null
  title?: string | null
  status: string
  policy: string
  presentAudio: boolean
  viewerCap: number
  activePresenterId?: string | null
  startedAt?: string | null
  endedAt?: string | null
  createdAt: string
}

export type IceServersPayload = {
  iceServers: RTCIceServer[]
  ttlSeconds: number
}

export type CreateSessionResult = {
  sessionId: string
  joinToken: string
  turn: IceServersPayload
  session: ScreenShareSession
}

export async function createScreenShareSession(
  courseCode: string,
  body: { title?: string; policy?: string; presentAudio?: boolean; viewerCap?: number } = {},
): Promise<CreateSessionResult> {
  const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/screen-share/sessions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `create session failed (${res.status})`)
  }
  return (await res.json()) as CreateSessionResult
}

export async function getScreenShareSession(courseCode: string, sessionId: string): Promise<ScreenShareSession> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/screen-share/sessions/${encodeURIComponent(sessionId)}`,
  )
  if (!res.ok) throw new Error(`get session failed (${res.status})`)
  return (await res.json()) as ScreenShareSession
}

export async function endScreenShareSession(courseCode: string, sessionId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/screen-share/sessions/${encodeURIComponent(sessionId)}/end`,
    { method: 'POST' },
  )
  if (!res.ok) throw new Error(`end session failed (${res.status})`)
}

export async function setScreenSharePresenter(
  courseCode: string,
  sessionId: string,
  action: 'grant' | 'revoke',
  userId: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/screen-share/sessions/${encodeURIComponent(sessionId)}/presenter`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, userId }),
    },
  )
  if (!res.ok) throw new Error(`presenter ${action} failed (${res.status})`)
}

export async function refreshScreenShareTurn(courseCode: string, sessionId: string): Promise<IceServersPayload> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/screen-share/sessions/${encodeURIComponent(sessionId)}/turn`,
    { method: 'POST' },
  )
  if (!res.ok) throw new Error(`turn refresh failed (${res.status})`)
  return (await res.json()) as IceServersPayload
}

/** Returns true when the browser can capture an entire screen (desktop Chromium/Firefox/Safari). */
export function canCaptureEntireScreen(): boolean {
  if (typeof navigator === 'undefined' || !navigator.mediaDevices?.getDisplayMedia) return false
  // Mobile browsers generally lack full-monitor capture; SS.M1 covers mobile origin.
  const ua = navigator.userAgent || ''
  if (/Android|iPhone|iPad|iPod/i.test(ua)) return false
  return true
}

export function displaySurfaceOf(track: MediaStreamTrack): string | undefined {
  const settings = track.getSettings?.() as { displaySurface?: string } | undefined
  return settings?.displaySurface
}

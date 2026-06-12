import { authorizedFetch } from './api'
import { getAccessToken } from './auth'
import {
  loadCourseNotebook,
  localNotebookCourseCodes,
  saveCourseNotebookFromServer,
  setNotebookSavedListener,
} from './student-notebook-storage'

/**
 * Server sync for student notebooks (last-write-wins by `updatedAt`).
 *
 * Local storage stays the source of truth for the editor; this module pushes saves to the
 * server (debounced per course) and pulls server copies on notebook page load so notebooks
 * written on other devices (e.g. mobile) appear here.
 */

const PUSH_DEBOUNCE_MS = 1500

type ServerNotebookEntry = {
  courseCode: string
  updatedAt: string
  data: unknown
}

const pushTimers = new Map<string, ReturnType<typeof setTimeout>>()

function signedIn(): boolean {
  return Boolean(getAccessToken())
}

async function pushNow(courseCode: string): Promise<void> {
  if (!signedIn()) return
  const data = loadCourseNotebook(courseCode)
  try {
    await authorizedFetch(`/api/v1/me/notebooks?courseCode=${encodeURIComponent(courseCode)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
  } catch {
    /* offline / transient — next save retries */
  }
}

/** Debounced fire-and-forget push of one notebook to the server. */
export function schedulePushStudentNotebook(courseCode: string): void {
  const prev = pushTimers.get(courseCode)
  if (prev) clearTimeout(prev)
  pushTimers.set(
    courseCode,
    setTimeout(() => {
      pushTimers.delete(courseCode)
      void pushNow(courseCode)
    }, PUSH_DEBOUNCE_MS),
  )
}

function parseTime(iso: string | undefined): number {
  const t = iso ? Date.parse(iso) : NaN
  return Number.isFinite(t) ? t : 0
}

/**
 * Pull server notebooks and merge into local storage. Server copy wins when newer;
 * local notebooks that are newer or missing on the server are pushed back.
 */
export async function pullStudentNotebooks(): Promise<void> {
  if (!signedIn()) return
  let entries: ServerNotebookEntry[]
  try {
    const res = await authorizedFetch('/api/v1/me/notebooks')
    if (!res.ok) return
    const body = (await res.json()) as { notebooks?: ServerNotebookEntry[] }
    entries = body.notebooks ?? []
  } catch {
    return
  }
  const localCodes = new Set(localNotebookCourseCodes())
  const serverCodes = new Set<string>()
  for (const entry of entries) {
    if (!entry?.courseCode) continue
    serverCodes.add(entry.courseCode)
    if (!localCodes.has(entry.courseCode)) {
      saveCourseNotebookFromServer(entry.courseCode, entry.data)
      continue
    }
    const local = loadCourseNotebook(entry.courseCode)
    const serverTime = parseTime(entry.updatedAt)
    const localTime = parseTime(local.updatedAt)
    if (serverTime > localTime) {
      saveCourseNotebookFromServer(entry.courseCode, entry.data)
    } else if (localTime > serverTime) {
      void pushNow(entry.courseCode)
    }
  }
  for (const code of localCodes) {
    if (!serverCodes.has(code)) void pushNow(code)
  }
}

// Every local save (all consumers go through saveCourseNotebookStore) schedules a push.
setNotebookSavedListener(schedulePushStudentNotebook)

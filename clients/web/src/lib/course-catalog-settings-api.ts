import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import {
  DEFAULT_KANBAN_COLUMN_LABELS,
  type CourseCatalogView,
  type KanbanColumnId,
  type KanbanColumnLabels,
} from './course-catalog-types'

export type { KanbanColumnLabels }
export { DEFAULT_KANBAN_COLUMN_LABELS }

export type CourseCatalogSettings = {
  view: CourseCatalogView
  kanbanColumnLabels: KanbanColumnLabels
  hiddenColumnExpanded: boolean
  nicknames: Record<string, string>
}

export type PinnedCourseSummary = {
  id: string
  courseCode: string
  title: string
  heroImageUrl: string | null
  heroImageObjectPosition: string | null
  catalogNickname?: string | null
}

const LEGACY_VIEW_KEY = 'lextures.courseCatalogView'
const LEGACY_HIDDEN_KEY = 'lextures.courseKanbanHiddenExpanded'

const VALID_VIEWS: CourseCatalogView[] = ['cards', 'list', 'gallery', 'table', 'status']

function isCourseCatalogView(value: string): value is CourseCatalogView {
  return (VALID_VIEWS as string[]).includes(value)
}

function normalizeKanbanColumnLabels(raw: Partial<KanbanColumnLabels> | undefined): KanbanColumnLabels {
  return {
    todo: raw?.todo?.trim() || DEFAULT_KANBAN_COLUMN_LABELS.todo,
    'in-progress': raw?.['in-progress']?.trim() || DEFAULT_KANBAN_COLUMN_LABELS['in-progress'],
    done: raw?.done?.trim() || DEFAULT_KANBAN_COLUMN_LABELS.done,
    hidden: raw?.hidden?.trim() || DEFAULT_KANBAN_COLUMN_LABELS.hidden,
  }
}

export async function fetchCourseCatalogSettings(): Promise<CourseCatalogSettings> {
  const res = await authorizedFetch('/api/v1/courses/catalog-settings')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as Partial<CourseCatalogSettings> & { kanbanColumnLabels?: Partial<KanbanColumnLabels> }
  const view = data.view && isCourseCatalogView(data.view) ? data.view : 'cards'
  return {
    view,
    kanbanColumnLabels: normalizeKanbanColumnLabels(data.kanbanColumnLabels),
    hiddenColumnExpanded: Boolean(data.hiddenColumnExpanded),
    nicknames: data.nicknames ?? {},
  }
}

export async function putCourseCatalogSettings(
  patch: Partial<Pick<CourseCatalogSettings, 'view' | 'kanbanColumnLabels' | 'hiddenColumnExpanded'>>,
): Promise<CourseCatalogSettings> {
  const res = await authorizedFetch('/api/v1/courses/catalog-settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as Partial<CourseCatalogSettings> & { kanbanColumnLabels?: Partial<KanbanColumnLabels> }
  const view = data.view && isCourseCatalogView(data.view) ? data.view : 'cards'
  return {
    view,
    kanbanColumnLabels: normalizeKanbanColumnLabels(data.kanbanColumnLabels),
    hiddenColumnExpanded: Boolean(data.hiddenColumnExpanded),
    nicknames: {},
  }
}

export async function putCourseCatalogNickname(courseId: string, nickname: string | null): Promise<void> {
  const res = await authorizedFetch('/api/v1/courses/catalog-nickname', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ courseId, nickname }),
  })
  if (res.ok) return
  const raw: unknown = await res.json().catch(() => ({}))
  throw new Error(readApiErrorMessage(raw))
}

export async function fetchPinnedCourses(): Promise<PinnedCourseSummary[]> {
  const res = await authorizedFetch('/api/v1/courses/catalog-pins')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { courses?: PinnedCourseSummary[] }
  return data.courses ?? []
}

export async function putCourseCatalogPin(courseId: string, pinned: boolean): Promise<void> {
  const res = await authorizedFetch('/api/v1/courses/catalog-pin', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ courseId, pinned }),
  })
  if (res.ok) return
  const raw: unknown = await res.json().catch(() => ({}))
  throw new Error(readApiErrorMessage(raw))
}

export async function putCourseKanbanBoard(columns: Record<KanbanColumnId, string[]>): Promise<void> {
  const res = await authorizedFetch('/api/v1/courses/kanban-board', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ columns }),
  })
  if (res.ok) return
  const raw: unknown = await res.json().catch(() => ({}))
  throw new Error(readApiErrorMessage(raw))
}

/** One-time migration of legacy localStorage prefs into the database. */
export async function migrateLegacyCourseCatalogLocalStorage(): Promise<void> {
  if (typeof window === 'undefined') return
  let view: CourseCatalogView | undefined
  let hiddenColumnExpanded: boolean | undefined
  try {
    const rawView = window.localStorage.getItem(LEGACY_VIEW_KEY)?.trim().toLowerCase()
    if (rawView && isCourseCatalogView(rawView)) view = rawView
    const rawHidden = window.localStorage.getItem(LEGACY_HIDDEN_KEY)
    if (rawHidden !== null) hiddenColumnExpanded = rawHidden === '1'
  } catch {
    return
  }
  if (!view && hiddenColumnExpanded === undefined) return
  const body: { view?: CourseCatalogView; hiddenColumnExpanded?: boolean } = {}
  if (view) body.view = view
  if (hiddenColumnExpanded !== undefined) body.hiddenColumnExpanded = hiddenColumnExpanded
  const res = await authorizedFetch('/api/v1/courses/catalog-settings/migrate-local', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) return
  try {
    window.localStorage.removeItem(LEGACY_VIEW_KEY)
    window.localStorage.removeItem(LEGACY_HIDDEN_KEY)
  } catch {
    /* ignore */
  }
}

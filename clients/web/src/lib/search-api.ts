import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import type { SearchGroup, SearchListItem } from './build-search-items'
import { normalizeSearchCourseItem } from './search-course-features'

export type SearchCourseItem = {
  courseCode: string
  title: string
  /** When false, hidden from search and nav (defaults on). */
  notebookEnabled?: boolean
  feedEnabled?: boolean
  calendarEnabled?: boolean
  discussionsEnabled?: boolean
  collabDocsEnabled?: boolean
  sbgEnabled?: boolean
  liveSessionsEnabled?: boolean
  groupSpacesEnabled?: boolean
  officeHoursEnabled?: boolean
  filesEnabled?: boolean
  attendanceEnabled?: boolean
  whiteboardEnabled?: boolean
  reportCardsEnabled?: boolean
  visualBoardsEnabled?: boolean
  interactiveQuizzesEnabled?: boolean
  questionBankEnabled?: boolean
  standardsAlignmentEnabled?: boolean
}

export type SearchPersonItem = {
  userId: string
  email: string
  displayName: string | null
  role: string
  courseCode: string
  courseTitle: string
}

export type SearchIndexResponse = {
  courses: SearchCourseItem[]
  people: SearchPersonItem[]
}

export type SearchQueryResultItem = {
  id: string
  type: string
  title: string
  subtitle: string
  path: string
  score?: number
}

export type SearchQueryGroup = {
  type: string
  label: string
  total: number
  items: SearchQueryResultItem[]
}

export type SearchQueryResponse = {
  groups: SearchQueryGroup[]
  tookMs: number
}

async function parseJson(res: Response): Promise<unknown> {
  return res.json().catch(() => ({}))
}

/** Courses and people visible to the signed-in user (same access rules as the LMS). */
export async function fetchSearchIndex(): Promise<SearchIndexResponse> {
  const res = await authorizedFetch('/api/v1/search')
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const body = raw as Partial<SearchIndexResponse>
  const rawCourses = Array.isArray(body.courses) ? body.courses : []
  return {
    courses: rawCourses.map((c) => normalizeSearchCourseItem(c as SearchCourseItem)),
    people: Array.isArray(body.people) ? body.people : [],
  }
}

export type FetchSearchQueryParams = {
  q: string
  scope?: string | null
  types?: string
}

/** Server-side FTS for courses, people, and module content. */
export async function fetchSearchQuery(params: FetchSearchQueryParams): Promise<SearchQueryResponse> {
  const sp = new URLSearchParams()
  sp.set('q', params.q)
  if (params.scope) sp.set('scope', params.scope)
  if (params.types) sp.set('types', params.types)
  const res = await authorizedFetch(`/api/v1/search/query?${sp.toString()}`)
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const body = raw as Partial<SearchQueryResponse>
  return {
    groups: Array.isArray(body.groups) ? body.groups : [],
    tookMs: typeof body.tookMs === 'number' ? body.tookMs : 0,
  }
}

const SERVER_GROUP_MAP: Record<string, SearchGroup> = {
  course: 'course',
  person: 'person',
  content: 'content',
}

export function queryResultsToSearchItems(groups: SearchQueryGroup[]): SearchListItem[] {
  const out: SearchListItem[] = []
  for (const group of groups) {
    const mappedGroup = SERVER_GROUP_MAP[group.type] ?? 'page'
    for (const item of group.items) {
      out.push({
        id: item.id,
        group: mappedGroup,
        title: item.title,
        subtitle: item.subtitle,
        path: item.path,
        haystack: `${item.title} ${item.subtitle} ${item.type}`.toLowerCase(),
        score: item.score,
      })
    }
  }
  return out
}

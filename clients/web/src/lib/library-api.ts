import { authorizedFetch } from './api'

export type LibraryBook = {
  id: string
  orgId: string
  title: string
  author: string | null
  isbn: string | null
  coverUrl: string | null
  lexileLevel: number | null
  fpBand: string | null
  gradeBand: string | null
  summary: string | null
  createdAt: string
  updatedAt: string
}

export type CreateBookPayload = {
  title: string
  author?: string
  isbn?: string
  coverUrl?: string
  lexileLevel?: number
  fpBand?: string
  gradeBand?: string
  summary?: string
}

export type LibraryBooksFilter = {
  lexile_min?: number
  lexile_max?: number
  grade_band?: string
}

export type ReadingLogEntry = {
  id: string
  studentId: string
  bookId: string | null
  bookTitle: string | null
  logDate: string
  pagesRead: number | null
  reflection: string | null
  loggedAt: string
}

export type CreateReadingLogPayload = {
  bookId?: string
  bookTitle?: string
  logDate: string
  pagesRead?: number
  reflection?: string
}

export type ReadingDashboardStudent = {
  studentId: string
  email: string
  displayName: string | null
  weeklyPages: number
  totalEntries: number
  totalPages: number
}

export async function listLibraryBooks(orgId: string, filter?: LibraryBooksFilter): Promise<LibraryBook[]> {
  const params = new URLSearchParams()
  if (filter?.lexile_min != null) params.set('lexile_min', String(filter.lexile_min))
  if (filter?.lexile_max != null) params.set('lexile_max', String(filter.lexile_max))
  if (filter?.grade_band) params.set('grade_band', filter.grade_band)
  const qs = params.toString() ? `?${params.toString()}` : ''
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/library${qs}`)
  const data = (await res.json()) as { books: LibraryBook[] }
  return data.books ?? []
}

export async function createLibraryBook(orgId: string, payload: CreateBookPayload): Promise<LibraryBook> {
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/library`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? 'Failed to add book')
  }
  const data = (await res.json()) as { book: LibraryBook }
  return data.book
}

export async function deleteLibraryBook(orgId: string, bookId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/library/${encodeURIComponent(bookId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) {
    throw new Error('Failed to delete book')
  }
}

export async function listReadingLogEntries(limit = 100): Promise<ReadingLogEntry[]> {
  const res = await authorizedFetch(`/api/v1/me/reading-log?limit=${limit}`)
  if (!res.ok) return []
  const data = (await res.json()) as { entries: ReadingLogEntry[] }
  return data.entries ?? []
}

export async function createReadingLogEntry(payload: CreateReadingLogPayload): Promise<ReadingLogEntry> {
  const res = await authorizedFetch('/api/v1/me/reading-log', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? 'Failed to save reading log entry')
  }
  const data = (await res.json()) as { entry: ReadingLogEntry }
  return data.entry
}

export async function getReadingDashboard(courseCode: string): Promise<ReadingDashboardStudent[]> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/reading-dashboard`,
  )
  if (!res.ok) return []
  const data = (await res.json()) as { students: ReadingDashboardStudent[] }
  return data.students ?? []
}

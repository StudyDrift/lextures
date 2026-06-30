import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { BookOpen, FileText, Search, Users } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  adminSearchResultsPath,
  fetchAdminOmnisearch,
  type AdminSearchResult,
} from '../../lib/admin-search-api'
import { LiveRegion } from '../a11y/live-region'

const DEBOUNCE_MS = 200
const MIN_QUERY = 2

type GroupKey = 'users' | 'courses' | 'content'

const GROUP_META: Record<GroupKey, { label: string; icon: typeof Users }> = {
  users: { label: 'Users', icon: Users },
  courses: { label: 'Courses', icon: BookOpen },
  content: { label: 'Content', icon: FileText },
}

function flatResults(data: {
  users: AdminSearchResult[]
  courses: AdminSearchResult[]
  content: AdminSearchResult[]
}): AdminSearchResult[] {
  return [...data.users, ...data.courses, ...data.content]
}

export function AdminSearchBar() {
  const { adminSearchEnabled } = usePlatformFeatures()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const listboxId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const activeRef = useRef<HTMLButtonElement | null>(null)

  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [users, setUsers] = useState<AdminSearchResult[]>([])
  const [courses, setCourses] = useState<AdminSearchResult[]>([])
  const [content, setContent] = useState<AdminSearchResult[]>([])
  const [cursor, setCursor] = useState(0)
  const [announcement, setAnnouncement] = useState('')

  const allFlat = flatResults({ users, courses, content })
  const totalCount = users.length + courses.length + content.length

  const runSearch = useCallback(
    async (q: string) => {
      if (q.length < MIN_QUERY) {
        setUsers([])
        setCourses([])
        setContent([])
        setAnnouncement('')
        return
      }
      setLoading(true)
      setError(null)
      try {
        const resp = await fetchAdminOmnisearch({ q, orgId })
        setUsers(resp.users)
        setCourses(resp.courses)
        setContent(resp.content)
        const n = resp.users.length + resp.courses.length + resp.content.length
        setAnnouncement(n === 0 ? `No results for ${q}` : `${n} results for ${q}`)
        setCursor(0)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Search failed.')
        setUsers([])
        setCourses([])
        setContent([])
      } finally {
        setLoading(false)
      }
    },
    [orgId],
  )

  useEffect(() => {
    if (!open) return
    const t = window.setTimeout(() => void runSearch(query.trim()), DEBOUNCE_MS)
    return () => window.clearTimeout(t)
  }, [query, open, runSearch])

  useEffect(() => {
    if (!adminSearchEnabled) return
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setOpen(true)
        window.setTimeout(() => inputRef.current?.focus(), 0)
      }
      if (e.key === 'Escape') setOpen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [adminSearchEnabled])

  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: 'nearest' })
  }, [cursor])

  if (!adminSearchEnabled) return null

  function navigateTo(item: AdminSearchResult) {
    const path = item.path + (orgId ? `${item.path.includes('?') ? '&' : '?'}orgId=${encodeURIComponent(orgId)}` : '')
    setOpen(false)
    navigate(path)
  }

  function onInputKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      if (allFlat.length > 0) setCursor((c) => (c + 1) % allFlat.length)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      if (allFlat.length > 0) setCursor((c) => (c - 1 + allFlat.length) % allFlat.length)
    } else if (e.key === 'Enter' && allFlat[cursor]) {
      e.preventDefault()
      navigateTo(allFlat[cursor])
    } else if (e.key === 'Escape') {
      setOpen(false)
    }
  }

  let flatIdx = 0

  function renderGroup(key: GroupKey, items: AdminSearchResult[]) {
    if (items.length === 0) return null
    const meta = GROUP_META[key]
    const Icon = meta.icon
    return (
      <div key={key} role="group" aria-label={meta.label}>
        <div className="flex items-center justify-between px-3 py-1.5 text-xs font-semibold uppercase tracking-wide text-slate-500">
          <span className="flex items-center gap-1.5">
            <Icon className="h-3.5 w-3.5" aria-hidden />
            {meta.label}
          </span>
          <button
            type="button"
            className="font-medium normal-case text-indigo-600 hover:underline dark:text-indigo-400"
            onClick={() => {
              setOpen(false)
              navigate(adminSearchResultsPath(query.trim(), key, orgId))
            }}
          >
            See all
          </button>
        </div>
        <ul>
          {items.map((item) => {
            const idx = flatIdx++
            const active = cursor === idx
            return (
              <li key={item.id}>
                <button
                  ref={active ? activeRef : undefined}
                  type="button"
                  role="option"
                  aria-selected={active}
                  className={`flex w-full flex-col gap-0.5 px-3 py-2 text-left text-sm ${
                    active ? 'bg-indigo-50 dark:bg-indigo-950/50' : 'hover:bg-slate-50 dark:hover:bg-neutral-900'
                  }`}
                  onMouseEnter={() => setCursor(idx)}
                  onClick={() => navigateTo(item)}
                >
                  <span className="font-medium text-slate-900 dark:text-slate-100">{item.title}</span>
                  <span className="text-xs text-slate-500 dark:text-slate-400">{item.subtitle}</span>
                  {item.snippet ? (
                    <span
                      className="text-xs text-slate-600 dark:text-slate-300"
                      dangerouslySetInnerHTML={{ __html: item.snippet }}
                    />
                  ) : null}
                </button>
              </li>
            )
          })}
        </ul>
      </div>
    )
  }

  return (
    <div className="relative mb-4 max-w-xl">
      <LiveRegion>{announcement}</LiveRegion>
      <label className="sr-only" htmlFor={`${listboxId}-input`}>
        Search organization
      </label>
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden />
        <input
          ref={inputRef}
          id={`${listboxId}-input`}
          type="search"
          role="combobox"
          aria-expanded={open}
          aria-controls={`${listboxId}-listbox`}
          aria-autocomplete="list"
          aria-activedescendant={allFlat[cursor] ? `${listboxId}-opt-${cursor}` : undefined}
          placeholder="Search users, courses, content… (⌘K)"
          value={query}
          onFocus={() => setOpen(true)}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={onInputKeyDown}
          className="w-full rounded-lg border border-slate-300 bg-white py-2 pl-9 pr-3 text-sm dark:border-neutral-700 dark:bg-neutral-900"
        />
      </div>
      {open && (
        <div
          id={`${listboxId}-listbox`}
          role="listbox"
          className="absolute z-50 mt-1 max-h-96 w-full overflow-auto rounded-lg border border-slate-200 bg-white shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
        >
          {loading ? (
            <p className="px-3 py-4 text-sm text-slate-500">Searching…</p>
          ) : error ? (
            <p className="px-3 py-4 text-sm text-red-600">{error}</p>
          ) : query.trim().length < MIN_QUERY ? (
            <p className="px-3 py-4 text-sm text-slate-500">Type at least {MIN_QUERY} characters.</p>
          ) : totalCount === 0 ? (
            <p className="px-3 py-4 text-sm text-slate-500">No results for &ldquo;{query.trim()}&rdquo;</p>
          ) : (
            <>
              {renderGroup('users', users)}
              {renderGroup('courses', courses)}
              {renderGroup('content', content)}
            </>
          )}
        </div>
      )}
      {open ? (
        <button
          type="button"
          aria-label="Close search"
          className="fixed inset-0 z-40 cursor-default bg-transparent"
          onClick={() => setOpen(false)}
        />
      ) : null}
    </div>
  )
}

import { useCallback, useEffect, useId, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import {
  fetchAdminSearchContent,
  fetchAdminSearchCourses,
  fetchAdminSearchUsers,
  type AdminSearchPaginated,
  type AdminSearchResult,
} from '../../lib/admin-search-api'

const TABS = [
  { key: 'users', label: 'Users' },
  { key: 'courses', label: 'Courses' },
  { key: 'content', label: 'Content' },
] as const

type TabKey = (typeof TABS)[number]['key']

function ResultRow({ item }: { item: AdminSearchResult }) {
  return (
    <li className="border-b border-border-subtle px-1 py-3 last:border-0 dark:border-border-subtle">
      <Link to={item.path} className="block hover:text-accent-fg dark:hover:text-indigo-300">
        <p className="font-medium text-fg-default dark:text-slate-100">{item.title}</p>
        <p className="text-sm text-fg-muted dark:text-fg-subtle">{item.subtitle}</p>
        {item.snippet ? (
          <p
            className="mt-1 text-sm text-fg-muted dark:text-slate-300"
            dangerouslySetInnerHTML={{ __html: item.snippet }}
          />
        ) : null}
      </Link>
    </li>
  )
}

export default function AdminSearchResults() {
  const titleId = useId()
  const [searchParams, setSearchParams] = useSearchParams()
  const q = searchParams.get('q') ?? ''
  const type = (searchParams.get('type') as TabKey | null) ?? 'users'
  const orgId = searchParams.get('orgId')
  const page = Math.max(1, Number(searchParams.get('page') ?? '1') || 1)

  const [data, setData] = useState<AdminSearchPaginated | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (q.trim().length < 2) {
      setData(null)
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const params = { q: q.trim(), page, perPage: 25, orgId }
      if (type === 'courses') {
        setData(await fetchAdminSearchCourses(params))
      } else if (type === 'content') {
        setData(await fetchAdminSearchContent(params))
      } else {
        setData(await fetchAdminSearchUsers(params))
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed.')
    } finally {
      setLoading(false)
    }
  }, [q, type, page, orgId])

  useEffect(() => {
    void load()
  }, [load])

  function setTab(next: TabKey) {
    const sp = new URLSearchParams(searchParams)
    sp.set('type', next)
    sp.set('page', '1')
    setSearchParams(sp)
  }

  function setPage(next: number) {
    const sp = new URLSearchParams(searchParams)
    sp.set('page', String(next))
    setSearchParams(sp)
  }

  return (
    <div>
      <h1 id={titleId} className="text-xl font-semibold text-fg-default dark:text-slate-100">
        Search results
      </h1>
      {q ? (
        <p className="mt-1 text-sm text-fg-muted dark:text-fg-subtle">
          Showing matches for &ldquo;{q}&rdquo;
          {data ? ` (${data.total} total, ${data.tookMs} ms)` : ''}
        </p>
      ) : (
        <p className="mt-1 text-sm text-fg-muted dark:text-fg-subtle">Enter a query from the search bar.</p>
      )}

      <div className="mt-4 flex gap-2 border-b border-border-default dark:border-border-subtle">
        {TABS.map((tab) => (
          <button
            key={tab.key}
            type="button"
            onClick={() => setTab(tab.key)}
            className={`border-b-2 px-3 py-2 text-sm font-medium ${ type === tab.key ? 'border-indigo-600 text-accent-fg dark:border-indigo-400 dark:text-indigo-300' : 'border-transparent text-fg-muted hover:text-fg-default dark:text-fg-subtle' }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="mt-6 space-y-3" aria-busy="true">
          {[1, 2, 3].map((n) => (
            <div key={n} className="h-14 animate-pulse rounded-lg bg-surface-sunken" />
          ))}
        </div>
      ) : error ? (
        <p className="mt-6 text-sm text-danger-fg">{error}</p>
      ) : !data || data.items.length === 0 ? (
        <p className="mt-6 text-sm text-fg-muted">No {type} found.</p>
      ) : (
        <>
          <ul className="mt-4 divide-y divide-slate-100 dark:divide-neutral-800">
            {data.items.map((item) => (
              <ResultRow key={item.id} item={item} />
            ))}
          </ul>
          {data.totalPages > 1 ? (
            <div className="mt-4 flex items-center gap-3 text-sm">
              <button
                type="button"
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
                className="rounded border border-border-strong px-3 py-1 disabled:opacity-40 dark:border-border-default"
              >
                Previous
              </button>
              <span className="text-fg-muted dark:text-fg-subtle">
                Page {page} of {data.totalPages}
              </span>
              <button
                type="button"
                disabled={page >= data.totalPages}
                onClick={() => setPage(page + 1)}
                className="rounded border border-border-strong px-3 py-1 disabled:opacity-40 dark:border-border-default"
              >
                Next
              </button>
            </div>
          ) : null}
        </>
      )}
    </div>
  )
}

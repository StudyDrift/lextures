import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { formatDateTime } from '../../lib/format'
import { BookOpen, ExternalLink, Plus, Search, X } from 'lucide-react'
import {
  fetchOERProviders,
  searchOER,
  importOERToModule,
  type OERProviderId,
  type OERSearchResult,
} from '../../lib/oer-api'

const PROVIDER_TABS: { id: OERProviderId; label: string }[] = [
  { id: 'oer_commons', label: 'OER Commons' },
  { id: 'merlot', label: 'MERLOT' },
  { id: 'openstax', label: 'OpenStax' },
]

const LICENSE_OPTIONS = [
  { value: '', label: 'Any license' },
  { value: 'CC-BY', label: 'CC BY only' },
]

function licenseAriaLabel(spdx: string, label: string): string {
  const l = label || spdx
  if (l.includes('BY-NC')) return 'Creative Commons BY-NC'
  if (l.includes('BY-ND')) return 'Creative Commons BY-ND'
  if (l.includes('BY-SA')) return 'Creative Commons BY-SA'
  if (l.includes('BY')) return 'Creative Commons BY'
  return `License: ${l}`
}

function LicenseBadge({ spdx, label }: { spdx: string; label: string }) {
  return (
    <span
      className="inline-flex items-center rounded-full border border-emerald-200/90 bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:border-emerald-500/35 dark:bg-emerald-950 dark:text-emerald-200"
      aria-label={licenseAriaLabel(spdx, label)}
      title={spdx}
    >
      {label || spdx}
    </span>
  )
}

type OERSearchPanelProps = {
  open: boolean
  courseCode: string
  moduleId: string
  onClose: () => void
  onImported: () => void | Promise<void>
}

export function OERSearchPanel({ open, courseCode, moduleId, onClose, onImported }: OERSearchPanelProps) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDialogElement>(null)
  const [enabledProviders, setEnabledProviders] = useState<OERProviderId[]>([])
  const [activeTab, setActiveTab] = useState<OERProviderId>('oer_commons')
  const [query, setQuery] = useState('')
  const [license, setLicense] = useState('')
  const [results, setResults] = useState<OERSearchResult[]>([])
  const [cacheAsOf, setCacheAsOf] = useState<string | null>(null)
  const [staleCache, setStaleCache] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [importingId, setImportingId] = useState<string | null>(null)

  const visibleTabs = PROVIDER_TABS.filter((t) => enabledProviders.includes(t.id))

  useEffect(() => {
    if (!open) return
    void fetchOERProviders()
      .then((ids) => {
        setEnabledProviders(ids)
        if (ids.length > 0 && !ids.includes(activeTab)) {
          setActiveTab(ids[0])
        }
      })
      .catch(() => setEnabledProviders(['oer_commons', 'merlot', 'openstax']))
  }, [open, activeTab])

  useEffect(() => {
    const el = dialogRef.current
    if (!el) return
    if (open && !el.open) {
      el.showModal()
    } else if (!open && el.open) {
      el.close()
    }
  }, [open])

  const runSearch = useCallback(async () => {
    if (!activeTab) return
    setLoading(true)
    setError(null)
    try {
      const resp = await searchOER(activeTab, { q: query.trim(), license: license || undefined })
      setResults(resp.results)
      setStaleCache(resp.staleCache ?? false)
      setCacheAsOf(resp.cacheAsOf ?? (resp.fromCache ? resp.cacheAsOf ?? null : null))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed.')
      setResults([])
    } finally {
      setLoading(false)
    }
  }, [activeTab, query, license])

  useEffect(() => {
    if (!open || !activeTab) return
    const t = window.setTimeout(() => void runSearch(), 300)
    return () => window.clearTimeout(t)
  }, [open, activeTab, runSearch])

  async function handleAdd(result: OERSearchResult) {
    setImportingId(result.id)
    setError(null)
    try {
      const title =
        result.provider === 'openstax' && !result.title.startsWith('OpenStax:')
          ? `OpenStax: ${result.title}`
          : result.title
      await importOERToModule(courseCode, moduleId, {
        title,
        url: result.url,
        provider: result.provider,
        externalId: result.id,
        licenseSpdx: result.licenseSpdx,
        attributionText: result.attribution,
      })
      await onImported()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not add to module.')
    } finally {
      setImportingId(null)
    }
  }

  if (!open) return null

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleId}
      className="fixed inset-0 z-50 m-0 flex h-full max-h-none w-full max-w-none items-stretch justify-end bg-transparent p-0 backdrop:bg-slate-900/40"
      onClose={() => onClose()}
    >
      <div className="ml-auto flex h-full w-full max-w-lg flex-col border-s border-slate-200 bg-white shadow-xl dark:border-neutral-600 dark:bg-neutral-900">
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-600">
          <h2 id={titleId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            Find open resources
          </h2>
          <button
            type="button"
            onClick={() => onClose()}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-700"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="border-b border-slate-100 px-4 py-3 dark:border-neutral-700">
          <div role="tablist" aria-label="OER providers" className="flex flex-wrap gap-1">
            {visibleTabs.map((tab) => (
              <button
                key={tab.id}
                type="button"
                role="tab"
                aria-selected={activeTab === tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition ${
                  activeTab === tab.id
                    ? 'bg-indigo-600 text-white'
                    : 'bg-slate-100 text-slate-700 hover:bg-slate-200 dark:bg-neutral-800 dark:text-neutral-200'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
          <form
            className="mt-3 flex flex-col gap-2"
            onSubmit={(e) => {
              e.preventDefault()
              void runSearch()
            }}
          >
            <label className="sr-only" htmlFor="oer-search-q">
              Search keywords
            </label>
            <div className="relative">
              <Search className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden />
              <input
                id="oer-search-q"
                type="search"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search by keyword…"
                className="w-full rounded-xl border border-slate-200 bg-white py-2.5 ps-9 pe-3 text-sm dark:border-neutral-600 dark:bg-neutral-800"
              />
            </div>
            <label className="text-xs font-medium text-slate-600 dark:text-neutral-400" htmlFor="oer-license-filter">
              License
            </label>
            <select
              id="oer-license-filter"
              value={license}
              onChange={(e) => setLicense(e.target.value)}
              className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            >
              {LICENSE_OPTIONS.map((o) => (
                <option key={o.value || 'any'} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </form>
          {staleCache && cacheAsOf && (
            <p className="mt-2 text-xs text-amber-800 dark:text-amber-200" role="status">
              Results may be outdated. As of {formatDateTime(cacheAsOf)}.
            </p>
          )}
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-3">
          {loading && <p className="text-sm text-slate-500">Searching…</p>}
          {error && (
            <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
              {error}
            </p>
          )}
          {!loading && !error && results.length === 0 && (
            <p className="text-sm text-slate-500">No results. Try another keyword or provider.</p>
          )}
          <ul role="list" className="space-y-3">
            {results.map((r) => (
              <li
                key={r.id}
                className="rounded-xl border border-slate-200 p-3 dark:border-neutral-600"
              >
                <div className="flex items-start gap-2">
                  <BookOpen className="mt-0.5 h-4 w-4 shrink-0 text-indigo-500" aria-hidden />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">{r.title}</p>
                    <div className="mt-1 flex flex-wrap items-center gap-2">
                      <LicenseBadge spdx={r.licenseSpdx} label={r.licenseLabel} />
                      {r.gradeLevel && (
                        <span className="text-xs text-slate-500 dark:text-neutral-400">{r.gradeLevel}</span>
                      )}
                    </div>
                    {r.description && (
                      <p className="mt-1 line-clamp-2 text-xs text-slate-600 dark:text-neutral-400">
                        {r.description}
                      </p>
                    )}
                    <div className="mt-2 flex flex-wrap gap-2">
                      <a
                        href={r.previewUrl || r.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-2.5 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200"
                      >
                        <ExternalLink className="h-3.5 w-3.5" aria-hidden />
                        Preview
                      </a>
                      <button
                        type="button"
                        disabled={importingId === r.id}
                        onClick={() => void handleAdd(r)}
                        className="inline-flex items-center gap-1 rounded-lg bg-indigo-600 px-2.5 py-1 text-xs font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
                      >
                        <Plus className="h-3.5 w-3.5" aria-hidden />
                        {importingId === r.id ? 'Adding…' : 'Add to module'}
                      </button>
                    </div>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </dialog>
  )
}

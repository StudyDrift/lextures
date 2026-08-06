import { useCallback, useEffect, useState } from 'react'
import { BookOpen, CalendarDays, Clock, Search, X } from 'lucide-react'
import {
  getCatalogSection,
  listCatalogSections,
  type CatalogSection,
  type CatalogListFilter,
} from '../../lib/catalog-api'
import {
  enrollConsortiumCourse,
  listConsortiumCourses,
  type ConsortiumSharedCourse,
} from '../../lib/consortium-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'
import { CourseCatalogStatusPill } from '../../components/ui/status-vocabulary'

type FilterState = {
  department: string
  days: string
  minCredits: string
  maxCredits: string
  q: string
}

const DAY_OPTIONS = ['', 'MWF', 'TR', 'MW', 'TTH']

function formatMeeting(mp?: CatalogSection['meetingPattern']): string {
  if (!mp) return '—'
  const parts: string[] = []
  if (mp.days) parts.push(mp.days)
  if (mp.startTime && mp.endTime) parts.push(`${mp.startTime}–${mp.endTime}`)
  else if (mp.startTime) parts.push(mp.startTime)
  return parts.join(' · ') || '—'
}

function prereqLabel(status: string): string {
  switch (status) {
    case 'met':
      return 'Met'
    case 'not_met':
      return 'Not met'
    case 'waived':
      return 'Waived'
    default: {
      const _exhaustive: never = status as never
      return _exhaustive
    }
  }
}

export default function CourseCatalogPage() {
  const { ffCatalogIntegration, ffConsortiumSharing } = usePlatformFeatures()
  const [tab, setTab] = useState<'catalog' | 'partner'>('catalog')
  const [partnerCourses, setPartnerCourses] = useState<ConsortiumSharedCourse[]>([])
  const [partnerLoading, setPartnerLoading] = useState(false)
  const [partnerError, setPartnerError] = useState<string | null>(null)
  const [sections, setSections] = useState<CatalogSection[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastSyncedAt, setLastSyncedAt] = useState<string | null>(null)
  const [resultCount, setResultCount] = useState<number | null>(null)
  const [filter, setFilter] = useState<FilterState>({
    department: '',
    days: '',
    minCredits: '',
    maxCredits: '',
    q: '',
  })
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [detail, setDetail] = useState<CatalogSection | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const f: CatalogListFilter = {
        department: filter.department || undefined,
        days: filter.days || undefined,
        minCredits: filter.minCredits ? Number(filter.minCredits) : undefined,
        maxCredits: filter.maxCredits ? Number(filter.maxCredits) : undefined,
        q: filter.q || undefined,
        limit: 50,
      }
      const result = await listCatalogSections(f)
      setSections(result.sections)
      setResultCount(result.sections.length)
      if (result.lastSyncedAt) setLastSyncedAt(result.lastSyncedAt)
    } catch {
      setError('Failed to load course catalog.')
    } finally {
      setLoading(false)
    }
  }, [filter])

  useEffect(() => {
    if (!ffCatalogIntegration) return
    void load()
  }, [ffCatalogIntegration, load])

  const loadPartner = useCallback(async () => {
    setPartnerLoading(true)
    setPartnerError(null)
    try {
      setPartnerCourses(await listConsortiumCourses())
    } catch {
      setPartnerError('Failed to load partner institution courses.')
    } finally {
      setPartnerLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!ffConsortiumSharing || tab !== 'partner') return
    void loadPartner()
  }, [ffConsortiumSharing, tab, loadPartner])

  useEffect(() => {
    if (!selectedId) {
      setDetail(null)
      return
    }
    setDetailLoading(true)
    void getCatalogSection(selectedId)
      .then(setDetail)
      .catch(() => setDetail(null))
      .finally(() => setDetailLoading(false))
  }, [selectedId])

  const handleFilterSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    void load()
  }

  if (!ffCatalogIntegration && !ffConsortiumSharing) {
    return (
      <main className="mx-auto max-w-4xl p-6">
        <p className="text-sm text-fg-muted">
          Course catalog integration is not enabled on this platform. Enable{' '}
          <strong>Course catalog integration</strong> or <strong>Consortium sharing</strong> in Settings → Global platform.
        </p>
      </main>
    )
  }

  if (tab === 'partner' && ffConsortiumSharing) {
    return (
      <LmsPage title="Partner institution courses" description="Courses offered by partner campuses through consortium sharing.">
        {ffCatalogIntegration ? (
          <div className="mb-4 flex gap-2">
            <button type="button" onClick={() => setTab('catalog')} className="rounded-lg border px-3 py-1.5 text-sm">
              Institution catalog
            </button>
            <button type="button" className="rounded-lg bg-accent-solid px-3 py-1.5 text-sm font-semibold text-white">
              Partner courses
            </button>
          </div>
        ) : null}
        {partnerLoading ? <p className="text-sm text-fg-muted">Loading…</p> : null}
        {partnerError ? <p className="text-sm text-rose-700">{partnerError}</p> : null}
        {!partnerLoading && partnerCourses.length === 0 ? (
          <p className="text-sm text-fg-muted" role="status">No partner courses available.</p>
        ) : null}
        <ul className="mt-4 space-y-3">
          {partnerCourses.map((c) => (
            <li key={c.id} className="rounded-xl border border-border-default p-4 dark:border-border-default">
              <p className="font-semibold">{c.title}</p>
              <p className="text-sm text-fg-muted">{c.hostOrgName} · {c.courseCode}</p>
              <button
                type="button"
                className="mt-2 rounded-lg bg-accent-solid px-3 py-1.5 text-xs font-semibold text-white"
                onClick={() => void enrollConsortiumCourse(c.id).then(() => loadPartner())}
              >
                Enroll
              </button>
            </li>
          ))}
        </ul>
      </LmsPage>
    )
  }

  if (!ffCatalogIntegration) {
    return (
      <main className="mx-auto max-w-4xl p-6">
        <p className="text-sm text-fg-muted">
          Browse partner courses using the Partner institution courses tab.
        </p>
        <button type="button" onClick={() => setTab('partner')} className="mt-3 rounded-lg bg-accent-solid px-4 py-2 text-sm font-semibold text-white">
          Partner institution courses
        </button>
      </main>
    )
  }

  return (
    <LmsPage
      title="Course catalog"
      description="Browse official sections synced from your institution's SIS."
    >
      {ffConsortiumSharing ? (
        <div className="mb-4 flex gap-2">
          <button type="button" className="rounded-lg bg-accent-solid px-3 py-1.5 text-sm font-semibold text-white">
            Institution catalog
          </button>
          <button type="button" onClick={() => setTab('partner')} className="rounded-lg border px-3 py-1.5 text-sm">
            Partner institution courses
          </button>
        </div>
      ) : null}
      {lastSyncedAt && (
        <div
          className="mb-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100"
          role="status"
        >
          Catalog data last synced {new Date(lastSyncedAt).toLocaleString()}.
        </div>
      )}

      <form
        role="search"
        aria-label="Filter catalog sections"
        onSubmit={handleFilterSubmit}
        className="mb-6 rounded-2xl border border-border-default bg-surface-raised p-4 shadow-sm dark:border-border-default dark:bg-surface-raised"
      >
        <div className="flex flex-wrap items-end gap-3">
          <div className="min-w-[200px] flex-1">
            <label htmlFor="catalog-search" className="text-sm font-medium text-fg-default">
              Search
            </label>
            <div className="relative mt-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" aria-hidden />
              <input
                id="catalog-search"
                type="search"
                value={filter.q}
                onChange={(e) => setFilter((f) => ({ ...f, q: e.target.value }))}
                placeholder="Title, subject, CRN…"
                className="w-full rounded-xl border border-border-default py-2 pl-9 pr-3 text-sm dark:border-border-default dark:bg-surface-overlay"
              />
            </div>
          </div>
          <div>
            <label htmlFor="catalog-dept" className="text-sm font-medium text-fg-default">
              Department
            </label>
            <input
              id="catalog-dept"
              type="text"
              value={filter.department}
              onChange={(e) => setFilter((f) => ({ ...f, department: e.target.value }))}
              placeholder="CS"
              className="mt-1 w-24 rounded-xl border border-border-default px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
            />
          </div>
          <div>
            <label htmlFor="catalog-days" className="text-sm font-medium text-fg-default">
              Days
            </label>
            <select
              id="catalog-days"
              value={filter.days}
              onChange={(e) => setFilter((f) => ({ ...f, days: e.target.value }))}
              className="mt-1 rounded-xl border border-border-default px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
            >
              {DAY_OPTIONS.map((d) => (
                <option key={d || 'any'} value={d}>
                  {d || 'Any'}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="catalog-min-credits" className="text-sm font-medium text-fg-default">
              Min credits
            </label>
            <input
              id="catalog-min-credits"
              type="number"
              min={0}
              step={0.5}
              value={filter.minCredits}
              onChange={(e) => setFilter((f) => ({ ...f, minCredits: e.target.value }))}
              className="mt-1 w-20 rounded-xl border border-border-default px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
            />
          </div>
          <button
            type="submit"
            className="rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500"
          >
            Apply filters
          </button>
        </div>
      </form>

      <p className="sr-only" aria-live="polite" aria-atomic="true">
        {resultCount != null ? `${resultCount} sections found` : ''}
      </p>

      {error && (
        <p className="mb-4 text-sm text-rose-600 dark:text-rose-400" role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3" aria-busy="true" aria-label="Loading sections">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-40 animate-pulse rounded-2xl bg-surface-sunken" />
          ))}
        </div>
      ) : sections.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border-default px-6 py-12 text-center dark:border-border-default">
          <BookOpen className="mx-auto h-10 w-10 text-slate-300 dark:text-neutral-600" aria-hidden />
          <p className="mt-3 text-sm text-fg-muted">
            No sections found for selected filters.
          </p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {sections.map((sec) => (
            <button
              key={sec.id}
              type="button"
              onClick={() => setSelectedId(sec.id)}
              className="rounded-2xl border border-border-default bg-surface-raised p-4 text-left shadow-sm transition-[box-shadow,background-color,color,border-color] hover:border-indigo-200 hover:shadow-md dark:border-border-default dark:bg-surface-raised dark:hover:border-indigo-800"
            >
              <div className="flex items-start justify-between gap-2">
                <p className="text-xs font-medium uppercase tracking-wide text-fg-muted">
                  {sec.subject} {sec.courseNumber}
                  {sec.sectionNumber ? ` · ${sec.sectionNumber}` : ''}
                </p>
                <CourseCatalogStatusPill label={sec.status} />
              </div>
              <p className="mt-1 font-semibold text-fg-default">{sec.title}</p>
              <p className="mt-2 flex items-center gap-1 text-xs text-fg-muted">
                <CalendarDays className="h-3.5 w-3.5" aria-hidden />
                {formatMeeting(sec.meetingPattern)}
              </p>
              {sec.credits != null && (
                <p className="mt-1 text-xs text-fg-muted">{sec.credits} credits</p>
              )}
              {sec.instructorName && (
                <p className="mt-1 text-xs text-fg-muted">{sec.instructorName}</p>
              )}
              {sec.fulfillsRequirements && sec.fulfillsRequirements.length > 0 ? (
                <p className="mt-2 inline-flex rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200">
                  Fulfills: {sec.fulfillsRequirements.join(', ')}
                </p>
              ) : null}
            </button>
          ))}
        </div>
      )}

      {selectedId && (
        <div
          className="fixed inset-0 z-50 flex justify-end bg-black/30"
          role="dialog"
          aria-modal="true"
          aria-label="Section detail"
          onClick={() => setSelectedId(null)}
        >
          <div
            className="h-full w-full max-w-md overflow-y-auto bg-surface-raised p-6 shadow-xl dark:bg-surface-raised"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-fg-default">Section detail</h2>
              <button
                type="button"
                onClick={() => setSelectedId(null)}
                className="rounded-lg p-2 text-fg-subtle hover:bg-surface-sunken dark:hover:bg-surface-overlay"
                aria-label="Close detail panel"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            {detailLoading || !detail ? (
              <p className="mt-4 text-sm text-fg-muted">Loading…</p>
            ) : (
              <div className="mt-4 space-y-4 text-sm">
                <div>
                  <p className="font-semibold text-fg-default">{detail.title}</p>
                  <p className="text-fg-muted">
                    {detail.subject} {detail.courseNumber}
                    {detail.sectionNumber ? ` · Section ${detail.sectionNumber}` : ''}
                  </p>
                </div>
                {detail.crn && (
                  <p>
                    <span className="font-medium">CRN:</span> {detail.crn}
                  </p>
                )}
                {detail.credits != null && (
                  <p>
                    <span className="font-medium">Credits:</span> {detail.credits}
                  </p>
                )}
                <p className="flex items-center gap-2">
                  <Clock className="h-4 w-4 text-fg-subtle" aria-hidden />
                  {formatMeeting(detail.meetingPattern)}
                  {detail.room ? ` · ${detail.room}` : ''}
                </p>
                {detail.instructorName && (
                  <p>
                    <span className="font-medium">Instructor:</span> {detail.instructorName}
                  </p>
                )}
                {detail.fulfillsRequirements && detail.fulfillsRequirements.length > 0 ? (
                  <p className="inline-flex rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200">
                    Fulfills: {detail.fulfillsRequirements.join(', ')}
                  </p>
                ) : null}
                {detail.prerequisites && detail.prerequisites.length > 0 && (
                  <div>
                    <p className="font-medium">Prerequisites</p>
                    <ul className="mt-1 space-y-1">
                      {detail.prerequisites.map((p) => {
                        const st = detail.prerequisiteStatus?.find((s) => s.code === p.code)
                        return (
                          <li key={p.code} className="flex items-center justify-between gap-2">
                            <span>
                              {p.code}
                              {p.title ? ` — ${p.title}` : ''}
                            </span>
                            {st && (
                              <span
                                className={
                                  st.status === 'met'
                                    ? 'text-emerald-600 dark:text-emerald-400'
                                    : st.status === 'waived'
                                      ? 'text-amber-600 dark:text-amber-400'
                                      : 'text-rose-600 dark:text-rose-400'
                                }
                              >
                                {prereqLabel(st.status)}
                              </span>
                            )}
                          </li>
                        )
                      })}
                    </ul>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </LmsPage>
  )
}

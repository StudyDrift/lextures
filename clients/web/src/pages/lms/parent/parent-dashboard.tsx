import { useEffect, useMemo, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { formatDateTime } from '../../../lib/format'
import { Link, useSearchParams } from 'react-router-dom'
import { CalendarHeart, Users } from 'lucide-react'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import {
  fetchParentChildren,
  fetchParentStudentAssignments,
  fetchParentStudentGrades,
  type ParentAssignmentRow,
  type ParentChildSummary,
  type ParentCourseGradesRow,
} from '../../../lib/parent-api'

function childLabel(c: ParentChildSummary): string {
  const n = c.displayName?.trim()
  if (n) return n
  return c.email
}

export default function ParentDashboard() {
  const { t } = useTranslation('parent')
  const { ffConferenceScheduling } = usePlatformFeatures()
  const [params, setParams] = useSearchParams()
  const [children, setChildren] = useState<ParentChildSummary[] | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [grades, setGrades] = useState<ParentCourseGradesRow[] | null>(null)
  const [assignments, setAssignments] = useState<ParentAssignmentRow[] | null>(null)
  const [detailError, setDetailError] = useState<string | null>(null)

  const selectedId = params.get('student') ?? ''

  const gradesForView = selectedId ? grades : null
  const assignmentsForView = selectedId ? assignments : null

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const data = await fetchParentChildren()
        if (cancelled) return
        setChildren(data.children)
        setLoadError(null)
      } catch (e) {
        if (!cancelled) {
          setLoadError(e instanceof Error ? e.message : t('parent.dashboard.errors.loadChildren'))
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [t])

  useEffect(() => {
    if (!children || children.length === 0) return
    if (!selectedId || !children.some((c) => c.studentUserId === selectedId)) {
      setParams({ student: children[0].studentUserId }, { replace: true })
    }
  }, [children, selectedId, setParams])

  useEffect(() => {
    if (!selectedId) {
      return
    }
    let cancelled = false
    void (async () => {
      try {
        const [g, a] = await Promise.all([
          fetchParentStudentGrades(selectedId),
          fetchParentStudentAssignments(selectedId),
        ])
        if (cancelled) return
        setDetailError(null)
        setGrades(g.courses)
        setAssignments(a.assignments)
      } catch (e) {
        if (!cancelled) {
          setDetailError(e instanceof Error ? e.message : t('parent.dashboard.errors.loadStudentData'))
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [selectedId, t])

  const selectedChild = useMemo(
    () => children?.find((c) => c.studentUserId === selectedId),
    [children, selectedId],
  )

  const displayName = selectedChild ? childLabel(selectedChild) : ''

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 md:px-8">
      <header className="flex flex-col gap-2 border-b border-slate-200 pb-6 dark:border-neutral-800">
        <div className="flex items-center gap-2 text-sm font-medium text-indigo-700 dark:text-indigo-300">
          <Users className="h-4 w-4" aria-hidden />
          {t('parent.dashboard.badge')}
        </div>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
          {t('parent.dashboard.title')}
        </h1>
        <p className="max-w-prose text-sm leading-relaxed text-slate-600 dark:text-neutral-400">
          {t('parent.dashboard.description')}
        </p>
        {ffConferenceScheduling && (
          <Link
            to={`/parent/conferences${selectedId ? `?student=${selectedId}` : ''}`}
            className="mt-2 inline-flex w-fit items-center gap-2 rounded-lg border border-indigo-200 bg-indigo-50 px-3 py-2 text-sm font-medium text-indigo-800 hover:bg-indigo-100 dark:border-indigo-800 dark:bg-indigo-950/40 dark:text-indigo-200"
          >
            <CalendarHeart className="h-4 w-4" aria-hidden />
            {t('parent.dashboard.conferencesLink')}
          </Link>
        )}
      </header>

      {loadError && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-100">
          {loadError}
        </div>
      )}

      {children && children.length === 0 && !loadError && (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100">
          {t('parent.dashboard.noChildren')}
        </p>
      )}

      {children && children.length > 0 && (
        <>
          <div role="listbox" aria-label={t('parent.dashboard.selectStudent')} className="flex flex-wrap gap-2">
            {children.map((c) => {
              const active = c.studentUserId === selectedId
              return (
                <button
                  key={c.studentUserId}
                  type="button"
                  role="option"
                  aria-selected={active}
                  className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-medium transition-colors ${
                    active
                      ? 'border-indigo-600 bg-indigo-600 text-white dark:border-indigo-500 dark:bg-indigo-600'
                      : 'border-slate-200 bg-white text-slate-800 hover:border-indigo-300 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-indigo-500/60'
                  }`}
                  onClick={() => setParams({ student: c.studentUserId })}
                >
                  <span className="max-w-[12rem] truncate">{childLabel(c)}</span>
                </button>
              )
            })}
          </div>

          {selectedChild && (
            <div
              role="status"
              aria-live="polite"
              className="sticky top-0 z-10 rounded-md border border-amber-300/80 bg-amber-50 px-4 py-2 text-sm text-amber-950 shadow-sm dark:border-amber-700/60 dark:bg-amber-950/50 dark:text-amber-50"
            >
              <Trans
                i18nKey="parent.dashboard.viewingAs"
                ns="parent"
                values={{ name: displayName }}
                components={{ strong: <strong className="font-semibold" /> }}
              />
            </div>
          )}

          {detailError && (
            <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-100">
              {detailError}
            </div>
          )}

          <section className="space-y-3">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              {t('parent.dashboard.grades.title')}
            </h2>
            {!gradesForView && !detailError && (
              <p className="text-sm text-slate-500 dark:text-neutral-400">{t('parent.dashboard.grades.loading')}</p>
            )}
            {gradesForView && gradesForView.length === 0 && (
              <p className="text-sm text-slate-600 dark:text-neutral-400">{t('parent.dashboard.grades.empty')}</p>
            )}
            {gradesForView && gradesForView.length > 0 && (
              <ul className="space-y-4">
                {gradesForView.map((row) => (
                  <li
                    key={row.courseCode}
                    className="rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950"
                  >
                    <h3 className="font-medium text-slate-900 dark:text-neutral-50">{row.title}</h3>
                    <p className="text-xs text-slate-500 dark:text-neutral-500">{row.courseCode}</p>
                    {Object.keys(row.grades).length === 0 ? (
                      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
                        {t('parent.dashboard.grades.noScores')}
                      </p>
                    ) : (
                      <ul className="mt-3 grid gap-2 sm:grid-cols-2">
                        {Object.entries(row.grades).map(([itemId, pts]) => (
                          <li
                            key={itemId}
                            className="flex justify-between gap-2 rounded-md bg-slate-50 px-3 py-2 text-sm dark:bg-neutral-900"
                          >
                            <span className="truncate font-mono text-xs text-slate-500 dark:text-neutral-500">
                              {itemId.slice(0, 8)}…
                            </span>
                            <span className="shrink-0 font-medium tabular-nums text-slate-900 dark:text-neutral-100">
                              {pts}
                            </span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              {t('parent.dashboard.assignments.title')}
            </h2>
            {!assignmentsForView && !detailError && (
              <p className="text-sm text-slate-500 dark:text-neutral-400">{t('parent.dashboard.grades.loading')}</p>
            )}
            {assignmentsForView && assignmentsForView.length === 0 && (
              <p className="text-sm text-slate-600 dark:text-neutral-400">{t('parent.dashboard.assignments.empty')}</p>
            )}
            {assignmentsForView && assignmentsForView.length > 0 && (
              <ul className="divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
                {assignmentsForView.map((a) => (
                  <li key={`${a.courseCode}-${a.itemId}`} className="bg-white px-4 py-3 dark:bg-neutral-950">
                    <div className="flex flex-wrap items-baseline justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate font-medium text-slate-900 dark:text-neutral-50">{a.title}</p>
                        <p className="text-xs text-slate-500 dark:text-neutral-500">
                          {a.courseTitle} · {a.kind}
                        </p>
                      </div>
                      {a.dueAt && (
                        <time className="shrink-0 text-xs text-slate-600 dark:text-neutral-400" dateTime={a.dueAt}>
                          {t('parent.dashboard.assignments.due', { date: formatDateTime(a.dueAt) })}
                        </time>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <p className="text-sm text-slate-600 dark:text-neutral-400">
            <Trans
              i18nKey="parent.dashboard.inboxHint"
              ns="parent"
              components={{
                inboxLink: (
                  <Link
                    to="/inbox"
                    className="text-indigo-700 underline underline-offset-2 dark:text-indigo-300"
                  />
                ),
              }}
            />
          </p>
        </>
      )}
    </div>
  )
}

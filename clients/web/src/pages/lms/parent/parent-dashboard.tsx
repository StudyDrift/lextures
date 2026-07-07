import { useEffect, useMemo, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { formatDateTime } from '../../../lib/format'
import { Link, useSearchParams } from 'react-router-dom'
import { CalendarHeart, Mail, Users } from 'lucide-react'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import {
  fetchParentChildren,
  fetchParentStudentAssignments,
  fetchParentStudentAttendanceSummary,
  fetchParentStudentBehavior,
  fetchParentStudentGrades,
  fetchParentStudentReportCards,
  type ParentAssignmentRow,
  type ParentAttendanceSummary,
  type ParentBehaviorResponse,
  type ParentChildSummary,
  type ParentCourseGradesRow,
  type ParentGradeItem,
  type ParentReportCard,
} from '../../../lib/parent-api'
import {
  parentChildLabel,
  parentGradeItemsForCourse,
  parentGradeScoreLabel,
  parentMessageTeacherHref,
} from '../../../lib/parent-portal'

type SectionState<T> = {
  data: T | null
  loading: boolean
  error: string | null
}

function emptySection<T>(): SectionState<T> {
  return { data: null, loading: false, error: null }
}

function gradeStatusKey(status: string): string {
  switch (status) {
    case 'posted':
      return 'parent.grades.status.posted'
    case 'excused':
      return 'parent.grades.status.excused'
    case 'graded':
      return 'parent.grades.status.graded'
    default:
      return 'parent.grades.status.graded'
  }
}

function SectionError({ message }: { message: string }) {
  return (
    <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-100">
      {message}
    </div>
  )
}

function GradeItemRow({ item, t }: { item: ParentGradeItem; t: (key: string, opts?: Record<string, unknown>) => string }) {
  return (
    <li className="rounded-md bg-slate-50 px-3 py-2 text-sm dark:bg-neutral-900">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="font-medium text-slate-900 dark:text-neutral-100">{item.title}</p>
          {item.category && (
            <p className="text-xs text-slate-500 dark:text-neutral-500">{item.category}</p>
          )}
          <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
            {t(gradeStatusKey(item.status))}
            {item.dueAt && (
              <>
                {' · '}
                {t('parent.grades.due', { date: formatDateTime(item.dueAt) })}
              </>
            )}
            {item.postedAt && (
              <>
                {' · '}
                {t('parent.grades.posted', { date: formatDateTime(item.postedAt) })}
              </>
            )}
          </p>
        </div>
        <span className="shrink-0 font-medium tabular-nums text-slate-900 dark:text-neutral-100">
          {parentGradeScoreLabel(item)}
        </span>
      </div>
    </li>
  )
}

export default function ParentDashboard() {
  const { t } = useTranslation('parent')
  const { ffConferenceScheduling, ffParentPortalV2, ffReportCards } = usePlatformFeatures()
  const [params, setParams] = useSearchParams()
  const [children, setChildren] = useState<ParentChildSummary[] | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [grades, setGrades] = useState<SectionState<ParentCourseGradesRow[]>>(emptySection)
  const [assignments, setAssignments] = useState<SectionState<ParentAssignmentRow[]>>(emptySection)
  const [attendance, setAttendance] = useState<SectionState<ParentAttendanceSummary>>(emptySection)
  const [behavior, setBehavior] = useState<SectionState<ParentBehaviorResponse>>(emptySection)
  const [reportCards, setReportCards] = useState<SectionState<ParentReportCard[]>>(emptySection)

  const selectedId = params.get('student') ?? ''

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
          setLoadError(e instanceof Error ? e.message : t('parent.loadChildrenError'))
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
    if (!selectedId) return
    let cancelled = false

    setGrades({ data: null, loading: true, error: null })
    setAssignments({ data: null, loading: true, error: null })
    void (async () => {
      try {
        const [g, a] = await Promise.all([
          fetchParentStudentGrades(selectedId),
          fetchParentStudentAssignments(selectedId),
        ])
        if (cancelled) return
        setGrades({ data: g.courses, loading: false, error: null })
        setAssignments({ data: a.assignments, loading: false, error: null })
      } catch (e) {
        if (!cancelled) {
          const msg = e instanceof Error ? e.message : t('parent.loadDetailError')
          setGrades({ data: null, loading: false, error: msg })
          setAssignments({ data: null, loading: false, error: msg })
        }
      }
    })()

    if (ffParentPortalV2) {
      setAttendance({ data: null, loading: true, error: null })
      setBehavior({ data: null, loading: true, error: null })
      setReportCards({ data: null, loading: true, error: null })

      void fetchParentStudentAttendanceSummary(selectedId)
        .then((data) => {
          if (!cancelled) setAttendance({ data, loading: false, error: null })
        })
        .catch((e) => {
          if (!cancelled) {
            setAttendance({
              data: null,
              loading: false,
              error: e instanceof Error ? e.message : t('parent.attendance.error'),
            })
          }
        })

      void fetchParentStudentBehavior(selectedId)
        .then((data) => {
          if (!cancelled) setBehavior({ data, loading: false, error: null })
        })
        .catch((e) => {
          if (!cancelled) {
            setBehavior({
              data: null,
              loading: false,
              error: e instanceof Error ? e.message : t('parent.behavior.error'),
            })
          }
        })

      if (ffReportCards) {
        void fetchParentStudentReportCards(selectedId)
          .then((data) => {
            if (!cancelled) setReportCards({ data: data.reportCards, loading: false, error: null })
          })
          .catch((e) => {
            if (!cancelled) {
              setReportCards({
                data: null,
                loading: false,
                error: e instanceof Error ? e.message : t('parent.reportCards.error'),
              })
            }
          })
      } else {
        setReportCards(emptySection())
      }
    } else {
      setAttendance(emptySection())
      setBehavior(emptySection())
      setReportCards(emptySection())
    }

    return () => {
      cancelled = true
    }
  }, [selectedId, ffParentPortalV2, ffReportCards, t])

  const selectedChild = useMemo(
    () => children?.find((c) => c.studentUserId === selectedId),
    [children, selectedId],
  )

  const displayName = selectedChild ? parentChildLabel(selectedChild.displayName, selectedChild.email) : ''

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 md:px-8">
      <header className="flex flex-col gap-2 border-b border-slate-200 pb-6 dark:border-neutral-800">
        <div className="flex items-center gap-2 text-sm font-medium text-indigo-700 dark:text-indigo-300">
          <Users className="h-4 w-4" aria-hidden />
          {t('parent.badge')}
        </div>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
          {t('parent.title')}
        </h1>
        <p className="max-w-prose text-sm leading-relaxed text-slate-600 dark:text-neutral-400">
          {t('parent.subtitle')}
        </p>
        {ffConferenceScheduling && (
          <Link
            to={`/parent/conferences${selectedId ? `?student=${selectedId}` : ''}`}
            className="mt-2 inline-flex w-fit items-center gap-2 rounded-lg border border-indigo-200 bg-indigo-50 px-3 py-2 text-sm font-medium text-indigo-800 hover:bg-indigo-100 dark:border-indigo-800 dark:bg-indigo-950/40 dark:text-indigo-200"
          >
            <CalendarHeart className="h-4 w-4" aria-hidden />
            {t('parent.bookConferences')}
          </Link>
        )}
      </header>

      {loadError && <SectionError message={loadError} />}

      {children && children.length === 0 && !loadError && (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100">
          {t('parent.noChildren')}
        </p>
      )}

      {children && children.length > 0 && (
        <>
          <div role="listbox" aria-label={t('parent.childSwitcher')} className="flex flex-wrap gap-2">
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
                  <span className="max-w-[12rem] truncate">
                    {parentChildLabel(c.displayName, c.email)}
                  </span>
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
              {t('parent.readOnly', { name: displayName })}
            </div>
          )}

          <section className="space-y-3" aria-labelledby="parent-grades-heading">
            <h2 id="parent-grades-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              {t('parent.section.grades')}
            </h2>
            {grades.loading && (
              <p className="text-sm text-slate-500 dark:text-neutral-400">{t('parent.grades.loading')}</p>
            )}
            {grades.error && <SectionError message={grades.error} />}
            {!grades.loading && !grades.error && grades.data && grades.data.length === 0 && (
              <p className="text-sm text-slate-600 dark:text-neutral-400">{t('parent.grades.empty')}</p>
            )}
            {!grades.loading && !grades.error && grades.data && grades.data.length > 0 && (
              <ul className="space-y-4">
                {grades.data.map((row) => {
                  const items = parentGradeItemsForCourse(row)
                  return (
                    <li
                      key={row.courseCode}
                      className="rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950"
                    >
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div>
                          <h3 className="font-medium text-slate-900 dark:text-neutral-50">{row.title}</h3>
                          <p className="text-xs text-slate-500 dark:text-neutral-500">{row.courseCode}</p>
                        </div>
                        {ffParentPortalV2 && row.teacherEmail && (
                          <Link
                            to={parentMessageTeacherHref({
                              teacherEmail: row.teacherEmail,
                              subject: t('parent.messageTeacherSubject', {
                                childName: displayName,
                                courseTitle: row.title,
                              }),
                            })}
                            className="inline-flex items-center gap-1.5 rounded-md border border-indigo-200 px-2.5 py-1.5 text-xs font-medium text-indigo-800 hover:bg-indigo-50 dark:border-indigo-800 dark:text-indigo-200 dark:hover:bg-indigo-950/40"
                          >
                            <Mail className="h-3.5 w-3.5" aria-hidden />
                            {t('parent.messageTeacher')}
                          </Link>
                        )}
                      </div>
                      {items.length === 0 ? (
                        <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
                          {t('parent.grades.noScores')}
                        </p>
                      ) : (
                        <ul className="mt-3 grid gap-2 sm:grid-cols-2">
                          {items.map((item) => (
                            <GradeItemRow key={item.itemId} item={item} t={t} />
                          ))}
                        </ul>
                      )}
                    </li>
                  )
                })}
              </ul>
            )}
          </section>

          {ffParentPortalV2 && (
            <section className="space-y-3" aria-labelledby="parent-attendance-heading">
              <h2 id="parent-attendance-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
                {t('parent.section.attendance')}
              </h2>
              {attendance.loading && (
                <p className="text-sm text-slate-500 dark:text-neutral-400">{t('parent.attendance.loading')}</p>
              )}
              {attendance.error && <SectionError message={attendance.error} />}
              {!attendance.loading && !attendance.error && attendance.data && (
                <div className="rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950">
                  <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">
                    {t('parent.attendance.summary', {
                      present: attendance.data.present,
                      absent: attendance.data.absent,
                      tardy: attendance.data.tardy,
                    })}
                  </p>
                  {attendance.data.recentDays.length === 0 ? (
                    <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
                      {t('parent.attendance.empty')}
                    </p>
                  ) : (
                    <>
                      <p className="mt-3 text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-500">
                        {t('parent.attendance.recent')}
                      </p>
                      <ul className="mt-2 divide-y divide-slate-100 dark:divide-neutral-800">
                        {attendance.data.recentDays.map((day) => (
                          <li key={`${day.date}-${day.period ?? day.code}`} className="flex justify-between py-2 text-sm">
                            <time dateTime={day.date}>{day.date}</time>
                            <span className="text-slate-600 dark:text-neutral-400">{day.codeLabel || day.category}</span>
                          </li>
                        ))}
                      </ul>
                    </>
                  )}
                </div>
              )}
            </section>
          )}

          {ffParentPortalV2 && (
            <section className="space-y-3" aria-labelledby="parent-behavior-heading">
              <h2 id="parent-behavior-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
                {t('parent.section.behavior')}
              </h2>
              {behavior.loading && (
                <p className="text-sm text-slate-500 dark:text-neutral-400">{t('parent.behavior.loading')}</p>
              )}
              {behavior.error && <SectionError message={behavior.error} />}
              {!behavior.loading && !behavior.error && behavior.data && (
                <div className="rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950">
                  {(behavior.data.totalPoints ?? 0) === 0 &&
                  (behavior.data.referrals?.length ?? 0) === 0 ? (
                    <p className="text-sm text-slate-600 dark:text-neutral-400">{t('parent.behavior.empty')}</p>
                  ) : (
                    <p className="text-sm text-slate-900 dark:text-neutral-100">
                      {t('parent.behavior.summary', {
                        points: behavior.data.totalPoints ?? 0,
                        referrals: behavior.data.referrals?.length ?? 0,
                      })}
                    </p>
                  )}
                </div>
              )}
            </section>
          )}

          {ffParentPortalV2 && ffReportCards && (
            <section className="space-y-3" aria-labelledby="parent-report-cards-heading">
              <h2 id="parent-report-cards-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
                {t('parent.section.reportCards')}
              </h2>
              {reportCards.loading && (
                <p className="text-sm text-slate-500 dark:text-neutral-400">{t('parent.reportCards.loading')}</p>
              )}
              {reportCards.error && <SectionError message={reportCards.error} />}
              {!reportCards.loading && !reportCards.error && reportCards.data && reportCards.data.length === 0 && (
                <p className="text-sm text-slate-600 dark:text-neutral-400">{t('parent.reportCards.empty')}</p>
              )}
              {!reportCards.loading && !reportCards.error && reportCards.data && reportCards.data.length > 0 && (
                <ul className="space-y-2">
                  {reportCards.data
                    .filter((card) => card.pdfUrl)
                    .map((card) => (
                      <li key={card.id}>
                        <a
                          href={card.pdfUrl!}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex text-sm font-medium text-indigo-700 underline underline-offset-2 dark:text-indigo-300"
                        >
                          {t('parent.reportCards.viewPdf', { period: card.gradingPeriod })}
                        </a>
                      </li>
                    ))}
                </ul>
              )}
            </section>
          )}

          <section className="space-y-3" aria-labelledby="parent-assignments-heading">
            <h2 id="parent-assignments-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              {t('parent.section.assignments')}
            </h2>
            {assignments.loading && (
              <p className="text-sm text-slate-500 dark:text-neutral-400">{t('parent.assignments.loading')}</p>
            )}
            {assignments.error && <SectionError message={assignments.error} />}
            {!assignments.loading && !assignments.error && assignments.data && assignments.data.length === 0 && (
              <p className="text-sm text-slate-600 dark:text-neutral-400">{t('parent.assignments.empty')}</p>
            )}
            {!assignments.loading && !assignments.error && assignments.data && assignments.data.length > 0 && (
              <ul className="divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
                {assignments.data.map((a) => (
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
                          {t('parent.assignments.due', { date: formatDateTime(a.dueAt) })}
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
              i18nKey="parent.messageInboxHint"
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

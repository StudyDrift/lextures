import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, Navigate, useOutletContext, useParams, useSearchParams } from 'react-router-dom'
import { BarChart3, ChevronRight } from 'lucide-react'
import { LearnerProgressCards } from '../../components/lms/learner-progress-cards'
import { LearnerSwitcher } from '../../components/lms/learner-switcher'
import { usePermissions } from '../../context/use-permissions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { useFamilyProgressAudience } from '../../hooks/use-family-audience'
import {
  courseGradebookViewPermission,
  fetchCourseEnrollmentsList,
  type CourseEnrollmentRosterRow,
  type CoursePublic,
} from '../../lib/courses-api'
import { progressSurfaceCopy } from '../../lib/family-progress-copy'
import { LmsPage } from './lms-page'

function normEnrollmentRole(role: string): string {
  return role.trim().toLowerCase()
}

function isStudentEnrollment(row: CourseEnrollmentRosterRow): boolean {
  const role = normEnrollmentRole(row.role)
  return role === 'student' || role === 'learner'
}

function studentDisplayName(row: CourseEnrollmentRosterRow): string {
  return row.displayName?.trim() || '—'
}

function sectionLabel(row: CourseEnrollmentRosterRow): string | null {
  const code = row.sectionCode?.trim()
  if (!code) return null
  const name = row.sectionName?.trim()
  return name ? `${code} (${name})` : code
}

export default function CourseStudentReportsPage() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const outlet = useOutletContext<{ course?: CoursePublic | null } | null>()
  const [searchParams, setSearchParams] = useSearchParams()
  const { studentProgressEnabled, loading: featuresLoading } = usePlatformFeatures()
  const { allows, loading: permLoading } = usePermissions()
  const canViewGradebook =
    !!courseCode && !permLoading && allows(courseGradebookViewPermission(courseCode))
  const [students, setStudents] = useState<CourseEnrollmentRosterRow[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  const loadStudents = useCallback(async () => {
    if (!courseCode) return
    setError(null)
    try {
      const rows = await fetchCourseEnrollmentsList(courseCode)
      const filtered = rows
        .filter(isStudentEnrollment)
        .sort((a, b) =>
          studentDisplayName(a).localeCompare(studentDisplayName(b), undefined, { sensitivity: 'base' }),
        )
      setStudents(filtered)
    } catch (e: unknown) {
      setStudents([])
      setError(e instanceof Error ? e.message : 'Could not load students.')
    }
  }, [courseCode])

  useEffect(() => {
    if (!courseCode || !canViewGradebook) return
    if (featuresLoading || !studentProgressEnabled) return
    void loadStudents()
  }, [canViewGradebook, courseCode, featuresLoading, loadStudents, studentProgressEnabled])

  const sectionsEnabled = useMemo(
    () => students?.some((s) => s.sectionCode?.trim()) ?? false,
    [students],
  )
  const rosterUserIds = useMemo(() => (students ?? []).map((s) => s.userId), [students])
  const familyProgress = useFamilyProgressAudience(outlet?.course ?? null, rosterUserIds)
  const copy = progressSurfaceCopy(familyProgress)
  const selectedLearner = searchParams.get('learner')?.trim() ?? ''
  const visibleStudents = useMemo(() => {
    if (!students) return students
    if (!familyProgress || !selectedLearner) return students
    return students.filter((student) => student.id === selectedLearner)
  }, [familyProgress, selectedLearner, students])

  if (!courseCode) {
    return <Navigate to="/courses" replace />
  }

  if (featuresLoading || permLoading) {
    return (
      <LmsPage title="Reports">
        <p className="mt-6 text-sm text-fg-muted">Loading…</p>
      </LmsPage>
    )
  }

  if (!studentProgressEnabled) {
    return <Navigate to={`/courses/${encodeURIComponent(courseCode)}`} replace />
  }

  if (!canViewGradebook) {
    return <Navigate to={`/courses/${encodeURIComponent(courseCode)}`} replace />
  }

  return (
    <LmsPage
      title="Reports"
      titleContent={
        <div className="min-w-0 flex-1">
          <h1 className="text-2xl font-semibold tracking-tight text-fg-default">
            Reports
          </h1>
          <p className="mt-2 max-w-2xl text-xs text-fg-muted">
            {copy.reportsDescription(courseCode)}
          </p>
        </div>
      }
    >
      {error ? (
        <p className="mt-6 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-200">
          {error}
        </p>
      ) : null}

      {students === null && !error ? (
        <p className="mt-8 text-sm text-fg-muted">{copy.reportsLoading}</p>
      ) : null}

      {students && students.length === 0 && !error ? (
        <p className="mt-8 text-sm text-fg-muted">{copy.reportsEmpty}</p>
      ) : null}

      {familyProgress && students && students.length > 0 ? (
        <>
          <LearnerSwitcher
            label={copy.learnerSwitcherLabel}
            allLabel={copy.allLearners}
            value={selectedLearner}
            learners={students.map((student) => ({
              id: student.id,
              name: studentDisplayName(student),
            }))}
            onChange={(id) => {
              const next = new URLSearchParams(searchParams)
              if (id) next.set('learner', id)
              else next.delete('learner')
              setSearchParams(next, { replace: true })
            }}
          />
          {visibleStudents && visibleStudents.length > 0 ? (
            <LearnerProgressCards
              courseCode={courseCode}
              loadSummaries={visibleStudents.length <= 8}
              learners={visibleStudents.map((student) => ({
                enrollmentId: student.id,
                name: studentDisplayName(student),
                sectionLabel: sectionLabel(student),
              }))}
            />
          ) : (
            <p className="mt-6 text-sm text-fg-muted">That learner is not enrolled in this course.</p>
          )}
        </>
      ) : null}

      {!familyProgress && students && students.length > 0 ? (
        <div className="mt-8 overflow-x-auto rounded-xl border border-border-default bg-surface-raised shadow-sm dark:border-border-default dark:bg-surface-raised">
          <table className="w-full min-w-[16rem] text-start text-sm">
            <thead>
              <tr className="border-b border-border-default bg-surface-base text-xs font-semibold uppercase tracking-wide text-fg-muted dark:border-border-default/60 dark:text-fg-muted">
                <th className="px-4 py-3">{copy.reportsColumn}</th>
                {sectionsEnabled ? <th className="px-4 py-3">Section</th> : null}
                <th className="px-2 py-3 text-end font-normal" aria-label="Actions" />
              </tr>
            </thead>
            <tbody>
              {students.map((student) => {
                const reportPath = `/courses/${encodeURIComponent(courseCode)}/students/${encodeURIComponent(student.id)}/progress`
                const name = studentDisplayName(student)
                return (
                  <tr
                    key={student.id}
                    className="group border-b border-border-subtle last:border-0 dark:border-border-subtle"
                  >
                    <td className="px-4 py-3 font-medium text-fg-default">
                      <Link
                        to={reportPath}
                        className="text-accent-fg hover:underline dark:text-indigo-300"
                      >
                        {name}
                      </Link>
                    </td>
                    {sectionsEnabled ? (
                      <td className="px-4 py-3 text-fg-muted">
                        {student.sectionCode?.trim()
                          ? student.sectionName?.trim()
                            ? `${student.sectionCode} (${student.sectionName})`
                            : student.sectionCode
                          : '—'}
                      </td>
                    ) : null}
                    <td className="px-2 py-3 text-end align-middle">
                      <Link
                        to={reportPath}
                        className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm font-medium text-accent-fg opacity-0 transition-[opacity,background-color,color,border-color] hover:bg-indigo-50 group-hover:opacity-100 focus-visible:opacity-100 dark:text-indigo-300 dark:hover:bg-indigo-950/40"
                        aria-label={`View report for ${name}`}
                      >
                        <BarChart3 className="h-4 w-4" aria-hidden />
                        View report
                        <ChevronRight className="h-4 w-4" aria-hidden />
                      </Link>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      ) : null}
    </LmsPage>
  )
}
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { BarChart3, ChevronRight } from 'lucide-react'
import { usePermissions } from '../../context/use-permissions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  courseGradebookViewPermission,
  fetchCourseEnrollmentsList,
  type CourseEnrollmentRosterRow,
} from '../../lib/courses-api'
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

export default function CourseStudentReportsPage() {
  const { courseCode } = useParams<{ courseCode: string }>()
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

  if (!courseCode) {
    return <Navigate to="/courses" replace />
  }

  if (featuresLoading || permLoading) {
    return (
      <LmsPage title="Reports">
        <p className="mt-6 text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
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
          <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-neutral-100">
            Reports
          </h1>
          <p className="mt-2 max-w-2xl text-xs text-slate-500 dark:text-neutral-400">
            Student progress reports for course {courseCode}. Select a student to view their report.
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
        <p className="mt-8 text-sm text-slate-500 dark:text-neutral-400">Loading students…</p>
      ) : null}

      {students && students.length === 0 && !error ? (
        <p className="mt-8 text-sm text-slate-500 dark:text-neutral-400">No students enrolled yet.</p>
      ) : null}

      {students && students.length > 0 ? (
        <div className="mt-8 overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
          <table className="w-full min-w-[16rem] text-start text-sm">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-neutral-400">
                <th className="px-4 py-3">Student</th>
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
                    className="group border-b border-slate-100 last:border-0 dark:border-neutral-800"
                  >
                    <td className="px-4 py-3 font-medium text-slate-900 dark:text-neutral-100">
                      <Link
                        to={reportPath}
                        className="text-indigo-700 hover:underline dark:text-indigo-300"
                      >
                        {name}
                      </Link>
                    </td>
                    {sectionsEnabled ? (
                      <td className="px-4 py-3 text-slate-600 dark:text-neutral-400">
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
                        className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm font-medium text-indigo-700 opacity-0 transition-[opacity,background-color,color,border-color] hover:bg-indigo-50 group-hover:opacity-100 focus-visible:opacity-100 dark:text-indigo-300 dark:hover:bg-indigo-950/40"
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
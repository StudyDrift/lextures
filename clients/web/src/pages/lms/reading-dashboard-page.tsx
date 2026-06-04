import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { BarChart2, RefreshCw } from 'lucide-react'
import { getReadingDashboard, type ReadingDashboardStudent } from '../../lib/library-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

export default function ReadingDashboardPage() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const { ffLibrary } = usePlatformFeatures()
  const [students, setStudents] = useState<ReadingDashboardStudent[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    setError(null)
    try {
      const result = await getReadingDashboard(courseCode)
      setStudents(result)
    } catch {
      setError('Failed to load reading dashboard.')
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    if (!ffLibrary) return
    void load()
  }, [courseCode, ffLibrary, load])

  if (!ffLibrary) {
    return (
      <LmsPage title="Reading Dashboard">
        <p className="text-muted-foreground">The library feature is not enabled.</p>
      </LmsPage>
    )
  }

  const totalWeeklyPages = students.reduce((sum, s) => sum + s.weeklyPages, 0)
  const onTrack = students.filter((s) => s.weeklyPages > 0).length

  return (
    <LmsPage title="Reading Dashboard">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <BarChart2 className="h-5 w-5" aria-hidden />
            <h1 className="text-xl font-semibold">Class Reading Dashboard</h1>
          </div>
          <button
            onClick={() => void load()}
            disabled={loading}
            className="inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm hover:bg-muted disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} aria-hidden />
            Refresh
          </button>
        </div>

        <div className="flex gap-4">
          <div className="rounded-lg border bg-card p-4 flex flex-col gap-0.5">
            <p className="text-xs text-muted-foreground">Total pages this week</p>
            <p className="text-2xl font-bold tabular-nums">{totalWeeklyPages}</p>
          </div>
          <div className="rounded-lg border bg-card p-4 flex flex-col gap-0.5">
            <p className="text-xs text-muted-foreground">Students reading this week</p>
            <p className="text-2xl font-bold tabular-nums">
              {onTrack} / {students.length}
            </p>
          </div>
        </div>

        {error && <p className="text-sm text-destructive">{error}</p>}

        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : students.length === 0 ? (
          <p className="text-sm text-muted-foreground">No enrolled students found.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-start text-xs text-muted-foreground">
                <th className="pb-2 font-medium">Student</th>
                <th className="pb-2 font-medium text-end">Pages (week)</th>
                <th className="pb-2 font-medium text-end">Total pages</th>
                <th className="pb-2 font-medium text-end">Total entries</th>
              </tr>
            </thead>
            <tbody>
              {students.map((s) => (
                <tr key={s.studentId} className="border-b last:border-0">
                  <td className="py-2 pe-3">
                    <span className="font-medium">{s.displayName ?? s.email}</span>
                    {s.displayName && (
                      <span className="ms-1 text-xs text-muted-foreground">{s.email}</span>
                    )}
                  </td>
                  <td className="py-2 pe-3 text-end tabular-nums">
                    <span
                      className={
                        s.weeklyPages > 0
                          ? 'font-medium text-green-700 dark:text-green-400'
                          : 'text-muted-foreground'
                      }
                    >
                      {s.weeklyPages}
                    </span>
                  </td>
                  <td className="py-2 pe-3 text-end tabular-nums">{s.totalPages}</td>
                  <td className="py-2 text-end tabular-nums">{s.totalEntries}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </LmsPage>
  )
}

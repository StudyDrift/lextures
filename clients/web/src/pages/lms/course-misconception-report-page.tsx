import { useEffect, useState } from 'react'
import { useOutletContext, useParams } from 'react-router-dom'
import { ConfirmDialog } from '../../components/confirm-dialog'
import { usePermissions } from '../../context/use-permissions'
import {
  courseItemsCreatePermission,
  fetchMisconceptionReport,
  postImportMisconceptionSeedLibrary,
  type CoursePublic,
  type MisconceptionReportRow,
} from '../../lib/courses-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

type CourseLayoutContext = {
  course?: CoursePublic | null
}

function isMissingRoute(message: string): boolean {
  return /no http route is registered/i.test(message)
}

function isFeatureOff(message: string): boolean {
  return /question bank/i.test(message) && /enable|disabled/i.test(message)
}

export default function CourseMisconceptionReportPage() {
  const { courseCode: raw } = useParams()
  const outlet = useOutletContext<CourseLayoutContext | null>()
  // CourseLayout renders the outlet before the course fetch resolves, so course is null
  // on the first paint. Gradebook treats that context as optional; this page must too.
  const course = outlet?.course ?? null
  const courseCode = course?.courseCode || (raw ? decodeURIComponent(raw) : '')
  const bankOn = course?.questionBankEnabled === true
  const { allows, loading: permLoading } = usePermissions()
  const canManage = Boolean(courseCode) && !permLoading && allows(courseItemsCreatePermission(courseCode))
  const [rows, setRows] = useState<MisconceptionReportRow[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [unavailable, setUnavailable] = useState(false)
  const [importBusy, setImportBusy] = useState(false)
  const [replaceSeedsOpen, setReplaceSeedsOpen] = useState(false)

  useEffect(() => {
    if (!course) return
    if (!bankOn) {
      setRows(null)
      setError(null)
      setUnavailable(false)
      return
    }
    if (!courseCode) return
    let cancelled = false
    ;(async () => {
      setError(null)
      setUnavailable(false)
      try {
        const res = await fetchMisconceptionReport(courseCode)
        if (!cancelled) setRows(res.misconceptions)
      } catch (e) {
        if (cancelled) return
        const msg = e instanceof Error ? e.message : 'Could not load report.'
        if (isMissingRoute(msg) || isFeatureOff(msg)) {
          setRows(null)
          setUnavailable(true)
          return
        }
        setError(msg)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [bankOn, course, courseCode])

  async function runImport(replaceExistingSeeds: boolean) {
    if (!courseCode) return
    setImportBusy(true)
    setError(null)
    try {
      const res = await postImportMisconceptionSeedLibrary(courseCode, { replaceExistingSeeds })
      toastSaveOk(`Imported ${res.imported} seed misconception${res.imported === 1 ? '' : 's'} (${res.skipped} skipped).`)
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Import failed.'
      setError(msg)
      toastMutationError(msg)
    } finally {
      setImportBusy(false)
      setReplaceSeedsOpen(false)
    }
  }

  const featureOff = course != null && !bankOn

  return (
    <div className="mx-auto max-w-5xl space-y-4 p-4">
      <ConfirmDialog
        open={replaceSeedsOpen}
        title="Replace seed misconceptions?"
        description="This removes existing items marked as seed in this course, then re-imports the built-in K–12 library."
        confirmLabel="Replace and import"
        variant="danger"
        busy={importBusy}
        onClose={() => !importBusy && setReplaceSeedsOpen(false)}
        onConfirm={() => void runImport(true)}
      />
      <div>
        <h1 className="text-lg font-semibold text-fg-default">Misconception report</h1>
        <p className="mt-1 text-sm text-fg-muted">
          Trigger counts for tagged distractors across submitted quiz attempts in this course.
        </p>
        {canManage && bankOn ? (
          <div className="mt-3 flex flex-wrap gap-2">
            <button
              type="button"
              disabled={importBusy}
              onClick={() => void runImport(false)}
              className="rounded-md border border-border-strong bg-surface-raised px-3 py-1.5 text-sm font-medium text-fg-default hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
            >
              {importBusy ? 'Working…' : 'Import seed library'}
            </button>
            <button
              type="button"
              disabled={importBusy}
              onClick={() => setReplaceSeedsOpen(true)}
              className="rounded-md border border-rose-300 bg-rose-50 px-3 py-1.5 text-sm font-medium text-rose-900 hover:bg-rose-100 disabled:opacity-50 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-100 dark:hover:bg-rose-950/70"
            >
              Replace seeds & re-import
            </button>
          </div>
        ) : null}
      </div>
      {course == null && (
        <p className="text-sm text-fg-muted">Loading course…</p>
      )}
      {featureOff && (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-100">
          The question bank feature is disabled for this course. Turn on &quot;Question bank&quot; in course
          features to use the misconception report.
        </p>
      )}
      {unavailable && !featureOff && (
        <p className="text-sm text-fg-muted">Misconception report is not available for this course.</p>
      )}
      {error && (
        <p className="text-sm text-rose-700 dark:text-rose-400" role="alert">
          {error}
        </p>
      )}
      {rows && rows.length === 0 && !error && (
        <p className="text-sm text-fg-muted">No misconception events recorded yet.</p>
      )}
      {rows && rows.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-border-default dark:border-border-subtle">
          <table className="min-w-full divide-y divide-slate-200 text-sm dark:divide-neutral-800">
            <thead className="bg-surface-base">
              <tr>
                <th className="px-3 py-2 text-start font-medium text-fg-default">Misconception</th>
                <th className="px-3 py-2 text-start font-medium text-fg-default">Question</th>
                <th className="px-3 py-2 text-end font-medium text-fg-default">Triggers</th>
                <th className="px-3 py-2 text-end font-medium text-fg-default">Students</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
              {rows.map((r) => (
                <tr key={`${r.misconceptionId}-${r.questionId}`}>
                  <td className="px-3 py-2 text-fg-default">{r.misconceptionName}</td>
                  <td className="max-w-md px-3 py-2 text-fg-muted">{r.questionStem}</td>
                  <td className="px-3 py-2 text-end tabular-nums text-fg-default">
                    {r.triggerCount}
                  </td>
                  <td className="px-3 py-2 text-end tabular-nums text-fg-default">
                    {r.affectedStudents}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

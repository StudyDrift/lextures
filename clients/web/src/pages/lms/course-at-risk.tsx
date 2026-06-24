import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  MoreHorizontal,
  RefreshCw,
  Users,
} from 'lucide-react'
import {
  AtRiskRunConfigDialog,
  type AtRiskConfigForm,
} from '../../components/at-risk/at-risk-run-config-dialog'
import { AtRiskReportRunningOverlay } from '../../components/at-risk/at-risk-report-running-overlay'
import { atRiskI18n } from '../../lib/at-risk-i18n'
import {
  courseGradebookViewPermission,
  fetchCourseAtRisk,
  patchCourseAtRiskAlert,
  runCourseAtRiskScoring,
  saveCourseAtRiskConfig,
  type AtRiskAlert,
} from '../../lib/courses-api'
import { usePermissions } from '../../context/use-permissions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

function scoreSeverity(score: number): 'moderate' | 'high' {
  return score >= 80 ? 'high' : 'moderate'
}

function ScoreBadge({ score }: { score: number }) {
  const sev = scoreSeverity(score)
  const label = sev === 'high' ? atRiskI18n.severityHigh : atRiskI18n.severityModerate
  const icon =
    sev === 'high' ? (
      <AlertTriangle className="h-4 w-4 text-red-600" aria-hidden />
    ) : (
      <AlertTriangle className="h-4 w-4 text-orange-600" aria-hidden />
    )
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-sm font-medium ${
        sev === 'high'
          ? 'bg-red-50 text-red-800 dark:bg-red-950/50 dark:text-red-200'
          : 'bg-orange-50 text-orange-800 dark:bg-orange-950/50 dark:text-orange-200'
      }`}
      title={`${atRiskI18n.score}: ${score}`}
    >
      {icon}
      <span>
        {label} ({score})
      </span>
    </span>
  )
}

function AlertRow({
  alert,
  courseCode,
  onUpdated,
}: {
  alert: AtRiskAlert
  courseCode: string
  onUpdated: () => void
}) {
  const [menuOpen, setMenuOpen] = useState(false)
  const [noteOpen, setNoteOpen] = useState(false)
  const [note, setNote] = useState(alert.notes ?? '')
  const [busy, setBusy] = useState(false)

  const patch = async (body: { status?: string; snoozeDays?: number; notes?: string }) => {
    setBusy(true)
    try {
      await patchCourseAtRiskAlert(courseCode, alert.id, body)
      onUpdated()
      setMenuOpen(false)
      setNoteOpen(false)
    } finally {
      setBusy(false)
    }
  }

  return (
    <li className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-700 dark:bg-neutral-900">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="font-medium text-slate-900 dark:text-neutral-100">{alert.displayName}</p>
          <div className="mt-1 flex flex-wrap items-center gap-2">
            <ScoreBadge score={alert.score} />
            <span className="rounded bg-slate-100 px-2 py-0.5 text-xs text-slate-700 dark:bg-neutral-800 dark:text-neutral-300">
              {alert.topFactorLabel}
            </span>
          </div>
        </div>
        <div className="relative">
          <button
            type="button"
            className="rounded p-0.5 hover:bg-slate-100 dark:hover:bg-neutral-800"
            aria-label="Actions"
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((o) => !o)}
          >
            <MoreHorizontal className="h-5 w-5" />
          </button>
          {menuOpen && (
            <ul
              role="menu"
              className="absolute end-0 z-10 mt-1 min-w-[10rem] rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-600 dark:bg-neutral-900"
            >
              <li>
                <button
                  type="button"
                  role="menuitem"
                  className="w-full px-2.5 py-1.5 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                  disabled={busy}
                  onClick={() => patch({ status: 'dismissed' })}
                >
                  {atRiskI18n.dismiss}
                </button>
              </li>
              <li>
                <button
                  type="button"
                  role="menuitem"
                  className="w-full px-2.5 py-1.5 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                  disabled={busy}
                  onClick={() => patch({ status: 'snoozed', snoozeDays: 7 })}
                >
                  {atRiskI18n.snooze7}
                </button>
              </li>
              <li>
                <button
                  type="button"
                  role="menuitem"
                  className="w-full px-2.5 py-1.5 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                  disabled={busy}
                  onClick={() => patch({ status: 'snoozed', snoozeDays: 14 })}
                >
                  {atRiskI18n.snooze14}
                </button>
              </li>
              <li>
                <button
                  type="button"
                  role="menuitem"
                  className="w-full px-2.5 py-1.5 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                  disabled={busy}
                  onClick={() => patch({ status: 'supported', snoozeDays: 14 })}
                >
                  {atRiskI18n.supported}
                </button>
              </li>
              <li>
                <button
                  type="button"
                  role="menuitem"
                  className="w-full px-2.5 py-1.5 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                  onClick={() => {
                    setMenuOpen(false)
                    setNoteOpen(true)
                  }}
                >
                  {atRiskI18n.addNote}
                </button>
              </li>
              <li>
                <Link
                  role="menuitem"
                  to={`/courses/${encodeURIComponent(courseCode)}/enrollments`}
                  className="block px-3 py-2 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                  onClick={() => setMenuOpen(false)}
                >
                  {atRiskI18n.viewProgress}
                </Link>
              </li>
            </ul>
          )}
        </div>
      </div>
      {noteOpen && (
        <div className="flex flex-col gap-2">
          <textarea
            className="w-full rounded border border-slate-300 p-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            rows={2}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder={atRiskI18n.notePlaceholder}
          />
          <button
            type="button"
            className="self-start rounded bg-indigo-600 px-3 py-1 text-sm text-white disabled:opacity-50"
            disabled={busy}
            onClick={() => patch({ notes: note })}
          >
            Save note
          </button>
        </div>
      )}
      {alert.notes && !noteOpen && (
        <p className="text-sm text-slate-600 dark:text-neutral-400">{alert.notes}</p>
      )}
    </li>
  )
}

const REPORT_LOADER_MIN_MS = 1000

function AtRiskEmptyState({
  afterReport,
  onRunReport,
}: {
  afterReport: boolean
  onRunReport: () => void
}) {
  if (afterReport) {
    return (
      <div
        className="mt-8 rounded-2xl border border-emerald-200 bg-emerald-50/80 px-6 py-12 text-center dark:border-emerald-900/40 dark:bg-emerald-950/30"
        role="status"
      >
        <CheckCircle2 className="mx-auto h-12 w-12 text-emerald-600 dark:text-emerald-400" aria-hidden />
        <p className="mt-4 text-lg font-semibold text-slate-900 dark:text-neutral-100">
          {atRiskI18n.emptyAfterReportTitle}
        </p>
        <p className="mx-auto mt-2 max-w-md text-sm leading-relaxed text-slate-600 dark:text-neutral-400">
          {atRiskI18n.emptyAfterReportDescription}
        </p>
      </div>
    )
  }

  return (
    <div
      className="mt-8 rounded-2xl border border-slate-200 bg-slate-50 px-6 py-12 text-center dark:border-neutral-700 dark:bg-neutral-900"
      role="status"
    >
      <Users className="mx-auto h-12 w-12 text-indigo-500 dark:text-indigo-400" aria-hidden />
      <p className="mt-4 text-lg font-semibold text-slate-900 dark:text-neutral-100">
        {atRiskI18n.emptyInitialTitle}
      </p>
      <p className="mx-auto mt-2 max-w-md text-sm leading-relaxed text-slate-600 dark:text-neutral-400">
        {atRiskI18n.emptyInitialDescription}
      </p>
      <button
        type="button"
        onClick={onRunReport}
        className="mt-6 inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500"
      >
        <RefreshCw className="h-4 w-4" aria-hidden />
        {atRiskI18n.emptyRunReportCta}
      </button>
    </div>
  )
}

export default function CourseAtRiskPage() {
  const { courseCode = '' } = useParams()
  const { atRiskAlertsEnabled, loading: featuresLoading } = usePlatformFeatures()
  const { allows, loading: permLoading } = usePermissions()
  const canView = allows(courseGradebookViewPermission(courseCode))
  const [alerts, setAlerts] = useState<AtRiskAlert[]>([])
  const [resolved, setResolved] = useState<AtRiskAlert[]>([])
  const [resolvedOpen, setResolvedOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [configOpen, setConfigOpen] = useState(false)
  const [reportRunning, setReportRunning] = useState(false)
  const [hasRunReport, setHasRunReport] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!courseCode || !atRiskAlertsEnabled) return
    setLoading(true)
    setError(null)
    try {
      const data = await fetchCourseAtRisk(courseCode, true)
      setAlerts(data.alerts)
      setResolved(data.resolved)
      if (data.alerts.length > 0 || data.resolved.length > 0) {
        setHasRunReport(true)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load at-risk alerts')
    } finally {
      setLoading(false)
    }
  }, [courseCode, atRiskAlertsEnabled])

  useEffect(() => {
    if (featuresLoading || !atRiskAlertsEnabled || permLoading || !canView) return
    void load()
  }, [load, featuresLoading, atRiskAlertsEnabled, permLoading, canView])

  const handleRunReport = useCallback(
    async (form: AtRiskConfigForm) => {
      if (!courseCode) return
      setConfigOpen(false)
      setReportRunning(true)
      setError(null)
      const started = Date.now()
      try {
        await saveCourseAtRiskConfig(courseCode, form)
        await runCourseAtRiskScoring(courseCode)
        const data = await fetchCourseAtRisk(courseCode, true)
        setAlerts(data.alerts)
        setResolved(data.resolved)
        setHasRunReport(true)
      } catch (e) {
        setError(e instanceof Error ? e.message : atRiskI18n.runScoringFailed)
        throw e
      } finally {
        const remaining = REPORT_LOADER_MIN_MS - (Date.now() - started)
        if (remaining > 0) {
          await new Promise<void>((resolve) => {
            window.setTimeout(resolve, remaining)
          })
        }
        setReportRunning(false)
      }
    },
    [courseCode],
  )

  if (featuresLoading || permLoading) {
    return (
      <LmsPage title={atRiskI18n.title}>
        <p className="text-slate-600 dark:text-neutral-400">Loading…</p>
      </LmsPage>
    )
  }

  if (!atRiskAlertsEnabled) {
    return (
      <LmsPage title={atRiskI18n.title}>
        <p className="text-slate-600 dark:text-neutral-400">At-risk alerts are not enabled.</p>
      </LmsPage>
    )
  }

  if (!canView) {
    return (
      <LmsPage title={atRiskI18n.title}>
        <p className="text-slate-600 dark:text-neutral-400">You do not have permission to view this page.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage
      title={atRiskI18n.title}
      actions={
        <button
          type="button"
          onClick={() => setConfigOpen(true)}
          disabled={loading || reportRunning}
          aria-label={atRiskI18n.runScoring}
          className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition-[background-color,color,border-color] hover:border-indigo-200 hover:bg-indigo-50/60 disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200"
        >
          <RefreshCw className="h-4 w-4" aria-hidden />
          {atRiskI18n.runScoring}
        </button>
      }
    >
      <AtRiskReportRunningOverlay open={reportRunning} />
      <AtRiskRunConfigDialog
        open={configOpen}
        courseCode={courseCode}
        onClose={() => setConfigOpen(false)}
        onRunReport={handleRunReport}
      />
      {loading && !reportRunning && (
        <p className="text-slate-600 dark:text-neutral-400">Loading…</p>
      )}
      {error && (
        <p
          className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-100"
          role="alert"
        >
          {error}
        </p>
      )}
      {!loading && !reportRunning && !error && alerts.length === 0 && (
        <AtRiskEmptyState afterReport={hasRunReport} onRunReport={() => setConfigOpen(true)} />
      )}
      {!loading && !reportRunning && alerts.length > 0 && (
        <ul role="list" className="flex flex-col gap-3">
          {alerts.map((a) => (
            <AlertRow key={a.id} alert={a} courseCode={courseCode} onUpdated={() => void load()} />
          ))}
        </ul>
      )}
      {resolved.length > 0 && (
        <div className="mt-8">
          <button
            type="button"
            className="flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-neutral-300"
            onClick={() => setResolvedOpen((o) => !o)}
          >
            {resolvedOpen ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
            {atRiskI18n.resolved} ({resolved.length})
          </button>
          {resolvedOpen && (
            <ul role="list" className="mt-3 flex flex-col gap-2 opacity-80">
              {resolved.map((a) => (
                <li
                  key={a.id}
                  className="rounded border border-slate-100 p-3 text-sm dark:border-neutral-800"
                >
                  {a.displayName} — {a.status}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </LmsPage>
  )
}

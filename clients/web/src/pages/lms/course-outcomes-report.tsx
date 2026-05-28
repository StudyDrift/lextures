import { useCallback, useEffect, useState } from 'react'
import { formatDateTime } from '../../lib/format'
import { Link, useParams } from 'react-router-dom'
import { RefreshCw, Target } from 'lucide-react'
import { LmsPage } from './lms-page'
import {
  fetchOutcomesReport,
  refreshOutcomesReport,
  saveOutcomeImprovementNote,
  updateOutcomesReportThreshold,
  type OutcomesReport,
  type OutcomesReportOutcome,
} from '../../lib/outcomes-report-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

function OutcomeAchievementBar({ outcome }: { outcome: OutcomesReportOutcome }) {
  if (outcome.noAlignments) {
    return (
      <p className="text-sm text-slate-500 dark:text-neutral-400">
        No assessments aligned — align assignments to see data.
      </p>
    )
  }
  const metPct = outcome.nAssessed > 0 ? outcome.pctMet : 0
  const notMetPct = outcome.nAssessed > 0 ? outcome.pctNotMet : 0
  const label = `Outcome ${outcome.title}: ${metPct}% met mastery threshold, ${notMetPct}% not met`
  return (
    <div className="space-y-1">
      <div
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(metPct)}
        aria-label={label}
        className="flex h-3 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-neutral-800"
      >
        <div
          className="h-full bg-emerald-500"
          style={{ width: `${metPct}%` }}
          title={`${metPct}% met`}
        />
        <div
          className="h-full bg-rose-400"
          style={{ width: `${notMetPct}%` }}
          title={`${notMetPct}% not met`}
        />
      </div>
      <p className="text-xs text-slate-600 dark:text-neutral-400 tabular-nums">
        {metPct}% met · {notMetPct}% not met
        {outcome.meanScore != null ? ` · class avg ${outcome.meanScore.toFixed(1)}%` : ''}
      </p>
    </div>
  )
}

function OutcomeNoteField({
  courseCode,
  outcome,
  onSaved,
}: {
  courseCode: string
  outcome: OutcomesReportOutcome
  onSaved: (text: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [text, setText] = useState(outcome.improvementNote)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setText(outcome.improvementNote)
  }, [outcome.improvementNote])

  async function save() {
    setSaving(true)
    try {
      await saveOutcomeImprovementNote(courseCode, outcome.outcomeId, text)
      onSaved(text)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="mt-3">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="text-sm font-medium text-indigo-600 hover:underline dark:text-indigo-400"
      >
        {open ? 'Hide note' : 'Add improvement note'}
      </button>
      {open && (
        <div className="mt-2">
          <label
            htmlFor={`outcome-note-${outcome.outcomeId}`}
            className="sr-only"
          >
            Improvement note for {outcome.title}
          </label>
          <textarea
            id={`outcome-note-${outcome.outcomeId}`}
            rows={3}
            value={text}
            onChange={(e) => setText(e.target.value)}
            onBlur={() => void save()}
            disabled={saving}
            className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
            placeholder="Qualitative notes for accreditation portfolio…"
          />
        </div>
      )}
    </div>
  )
}

export default function CourseOutcomesReport() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const { outcomesReportEnabled, loading: featuresLoading } = usePlatformFeatures()
  const [report, setReport] = useState<OutcomesReport | null>(null)
  const [threshold, setThreshold] = useState(70)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshing, setRefreshing] = useState(false)

  const load = useCallback(async () => {
    if (!courseCode || !outcomesReportEnabled) return
    setLoading(true)
    setError(null)
    try {
      const data = await fetchOutcomesReport(courseCode)
      setReport(data)
      setThreshold(data.masteryThreshold)
    } catch (e) {
      setReport(null)
      setError(e instanceof Error ? e.message : 'Could not load outcomes report.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, outcomesReportEnabled])

  useEffect(() => {
    if (featuresLoading || !outcomesReportEnabled) return
    void load()
  }, [load, featuresLoading, outcomesReportEnabled])

  async function handleRefresh() {
    if (!courseCode) return
    setRefreshing(true)
    try {
      await refreshOutcomesReport(courseCode)
      await load()
    } finally {
      setRefreshing(false)
    }
  }

  async function handleThresholdBlur() {
    if (!courseCode || !report) return
    if (threshold === report.masteryThreshold) return
    try {
      await updateOutcomesReportThreshold(courseCode, threshold)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save threshold.')
    }
  }

  function updateOutcomeNote(outcomeId: string, noteText: string) {
    setReport((prev) => {
      if (!prev) return prev
      return {
        ...prev,
        outcomes: prev.outcomes.map((o) =>
          o.outcomeId === outcomeId ? { ...o, improvementNote: noteText } : o,
        ),
      }
    })
  }

  if (featuresLoading) {
    return (
      <LmsPage title="Outcomes report" description="Course learning outcomes achievement.">
        <p className="mt-8 text-sm text-slate-500 dark:text-neutral-400" aria-live="polite">
          Loading…
        </p>
      </LmsPage>
    )
  }

  if (!outcomesReportEnabled) {
    return (
      <LmsPage title="Outcomes report" description="Course learning outcomes achievement.">
        <p className="mt-8 text-sm text-slate-600 dark:text-neutral-400">
          Outcomes reporting is not enabled on this platform. Ask a global administrator to turn on
          &quot;Outcomes report&quot; in Settings → Global platform.
        </p>
      </LmsPage>
    )
  }

  const hasOutcomes = Boolean(report?.outcomes.length)
  const allEmpty =
    hasOutcomes && report!.outcomes.every((o) => o.noAlignments)

  return (
    <LmsPage
      title="Outcomes report"
      description="Cohort achievement on course learning outcomes for accreditation and standards reporting."
      actions={
        <div className="flex flex-wrap items-center gap-3">
          <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
            <span>Mastery threshold</span>
            <input
              type="number"
              min={1}
              max={100}
              value={threshold}
              onChange={(e) => setThreshold(Number(e.target.value))}
              onBlur={() => void handleThresholdBlur()}
              aria-label="Mastery threshold percent"
              className="w-16 rounded-lg border border-slate-200 px-2 py-1 tabular-nums dark:border-neutral-700 dark:bg-neutral-900"
            />
            <span>%</span>
          </label>
          <button
            type="button"
            onClick={() => void handleRefresh()}
            disabled={refreshing}
            aria-label="Refresh outcomes report"
            className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition hover:border-indigo-200 hover:bg-indigo-50/60 disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200"
          >
            <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} aria-hidden />
            {refreshing ? 'Refreshing…' : 'Refresh'}
          </button>
        </div>
      }
    >
      {loading && (
        <p className="mt-8 text-sm text-slate-500 dark:text-neutral-400" aria-live="polite">
          Loading outcomes report…
        </p>
      )}
      {error && (
        <p
          className="mt-8 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-100"
          role="alert"
        >
          {error}
        </p>
      )}
      {!loading && !error && allEmpty && (
        <div className="mt-8 rounded-2xl border border-slate-200 bg-slate-50 px-6 py-10 text-center dark:border-neutral-700 dark:bg-neutral-900">
          <Target className="mx-auto h-10 w-10 text-indigo-500" aria-hidden />
          <p className="mt-3 text-base font-semibold text-slate-800 dark:text-neutral-100">
            Align your assignments to outcomes to see this report
          </p>
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
            <Link
              to={`/courses/${encodeURIComponent(courseCode ?? '')}/settings/outcomes`}
              className="text-indigo-600 underline-offset-2 hover:underline dark:text-indigo-400"
            >
              Open learning outcomes settings
            </Link>{' '}
            to map graded work, then return here after grades are posted.
          </p>
        </div>
      )}
      {!loading && !error && report && hasOutcomes && !allEmpty && (
        <div className="mt-8 space-y-6">
          <p className="text-xs text-slate-500 dark:text-neutral-400">
            Data as of{' '}
            {formatDateTime(report.dataAsOf, {
              dateStyle: 'medium',
              timeStyle: 'short',
            })}
            {report.staleMinutes >= 120 && (
              <span className="ms-2 text-amber-700 dark:text-amber-300">
                (stale — refresh recommended)
              </span>
            )}
          </p>
          <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
            <table className="min-w-full text-start text-sm">
              <caption className="sr-only">
                Course learning outcomes achievement summary
              </caption>
              <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:border-neutral-700 dark:bg-neutral-800/80 dark:text-neutral-300">
                <tr>
                  <th scope="col" className="px-4 py-3">
                    Outcome
                  </th>
                  <th scope="col" className="px-4 py-3 text-end">
                    Assessed
                  </th>
                  <th scope="col" className="px-4 py-3 min-w-[200px]">
                    Achievement
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                {report.outcomes.map((o) => (
                  <tr key={o.outcomeId} className="align-top">
                    <td className="px-4 py-4">
                      <p className="font-medium text-slate-900 dark:text-neutral-100">{o.title}</p>
                      {o.noAlignments ? (
                        <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                          No assessments aligned —{' '}
                          <Link
                            to={`/courses/${encodeURIComponent(courseCode ?? '')}/settings/outcomes`}
                            className="text-indigo-600 hover:underline dark:text-indigo-400"
                          >
                            align assessments
                          </Link>
                        </p>
                      ) : (
                        <OutcomeNoteField
                          courseCode={courseCode!}
                          outcome={o}
                          onSaved={(text) => updateOutcomeNote(o.outcomeId, text)}
                        />
                      )}
                    </td>
                    <td className="px-4 py-4 text-end tabular-nums text-slate-700 dark:text-neutral-300">
                      {o.nAssessed} / {o.nStudents}
                    </td>
                    <td className="px-4 py-4">
                      <OutcomeAchievementBar outcome={o} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div
            role="img"
            aria-label="Chart legend: green indicates students who met the mastery threshold; red indicates students who did not."
            className="flex flex-wrap gap-4 text-xs text-slate-600 dark:text-neutral-400"
          >
            <span className="inline-flex items-center gap-1.5">
              <span className="h-2 w-4 rounded bg-emerald-500" /> Met threshold
            </span>
            <span className="inline-flex items-center gap-1.5">
              <span className="h-2 w-4 rounded bg-rose-400" /> Not met
            </span>
          </div>
        </div>
      )}
    </LmsPage>
  )
}

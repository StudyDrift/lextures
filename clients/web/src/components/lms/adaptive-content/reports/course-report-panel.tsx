import { useCallback, useEffect, useId, useState } from 'react'
import { Download, Loader2, RefreshCw } from 'lucide-react'
import { Link } from 'react-router-dom'
import {
  exportAdaptiveContentCourseReportCsv,
  fetchAdaptiveContentCourseReport,
  refreshAdaptiveContentEffectiveness,
  type AdaptiveContentCourseReport,
  type AdaptiveContentUnitToReview,
} from '../../../../lib/courses-api'
import { EffectivenessChip } from '../effectiveness-chip'

type Props = {
  courseCode: string
}

function formatLift(v: number | null | undefined): string {
  if (v == null || Number.isNaN(v)) return '—'
  const sign = v > 0 ? '+' : ''
  return `${sign}${v.toFixed(1)} pts`
}

function formatPct(v: number): string {
  if (!Number.isFinite(v)) return '—'
  return `${v.toFixed(0)}%`
}

function reasonLabel(reason: string): string {
  switch (reason) {
    case 'regressing':
      return 'Regressing'
    case 'low_fidelity':
      return 'Low fidelity'
    case 'insufficient_data':
      return 'Insufficient data'
    default:
      return reason
  }
}

function Kpi({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="rounded-xl border border-border-default px-3 py-2 dark:border-border-default">
      <p className="text-xs font-medium uppercase tracking-wide text-fg-muted">
        {label}
      </p>
      <p className="mt-0.5 text-lg font-semibold tabular-nums text-fg-default">
        {value}
      </p>
      {hint ? <p className="mt-0.5 text-xs text-fg-muted">{hint}</p> : null}
    </div>
  )
}

function ModeBarChart({
  modes,
}: {
  modes: AdaptiveContentCourseReport['byMode']
}) {
  const tableId = useId()
  const [showTable, setShowTable] = useState(false)
  const maxN = Math.max(1, ...modes.map((m) => m.n))
  if (modes.length === 0) {
    return <p className="text-sm text-fg-muted">No mode breakdown yet.</p>
  }
  return (
    <div>
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm text-fg-muted">
          Mean lift by emphasis mode (suppressed when n &lt; small-cell minimum).
        </p>
        <button
          type="button"
          className="text-xs font-medium text-accent-fg underline dark:text-indigo-300"
          aria-controls={tableId}
          aria-expanded={showTable}
          onClick={() => setShowTable((v) => !v)}
        >
          {showTable ? 'Hide data table' : 'Show data table'}
        </button>
      </div>
      <ul className="mt-3 space-y-2" aria-hidden={showTable}>
        {modes.map((m) => (
          <li key={m.emphasisMode} className="flex items-center gap-3 text-sm">
            <span className="w-24 shrink-0 font-medium text-fg-default">
              {m.emphasisMode}
            </span>
            <div
              className="h-3 flex-1 rounded bg-surface-sunken"
              role="presentation"
            >
              <div
                className="h-3 rounded bg-teal-600 dark:bg-teal-500"
                style={{ width: `${(m.n / maxN) * 100}%` }}
              />
            </div>
            <span className="w-28 shrink-0 tabular-nums text-fg-muted">
              n={m.n} · {formatLift(m.meanLift)}
            </span>
          </li>
        ))}
      </ul>
      {showTable ? (
        <table id={tableId} className="mt-3 w-full text-left text-sm">
          <caption className="sr-only">Effectiveness by emphasis mode</caption>
          <thead>
            <tr className="border-b border-border-default">
              <th scope="col" className="py-1 pe-2 font-medium">
                Mode
              </th>
              <th scope="col" className="py-1 pe-2 font-medium">
                N
              </th>
              <th scope="col" className="py-1 font-medium">
                Mean lift
              </th>
            </tr>
          </thead>
          <tbody>
            {modes.map((m) => (
              <tr key={m.emphasisMode} className="border-b border-border-subtle">
                <td className="py-1 pe-2">{m.emphasisMode}</td>
                <td className="py-1 pe-2 tabular-nums">{m.n}</td>
                <td className="py-1 tabular-nums">{formatLift(m.meanLift)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </div>
  )
}

function UnitsToReviewList({
  courseCode,
  items,
}: {
  courseCode: string
  items: AdaptiveContentUnitToReview[]
}) {
  if (items.length === 0) {
    return (
      <p className="text-sm text-fg-muted">
        No units currently need review.
      </p>
    )
  }
  return (
    <ol className="space-y-2">
      {items.map((u, idx) => (
        <li
          key={`${u.unitId}-${u.reason}`}
          className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border-default px-3 py-2 dark:border-border-default"
          data-testid={idx === 0 ? 'ace-report-top-review-unit' : undefined}
        >
          <div className="min-w-0">
            <p className="text-sm font-medium text-fg-default">
              {reasonLabel(u.reason)}
              <span className="ms-2 font-mono text-xs text-fg-muted">
                {u.unitId.slice(0, 8)}…
              </span>
            </p>
            <p className="mt-0.5 text-xs text-fg-muted">
              Verdict: {u.verdict.replaceAll('_', ' ')}
              {u.meanLift != null ? ` · lift ${formatLift(u.meanLift)}` : ''}
              {u.meanFidelity != null
                ? ` · fidelity ${Math.round(u.meanFidelity * 100)}%`
                : ''}
            </p>
          </div>
          <Link
            to={`/courses/${encodeURIComponent(courseCode)}/settings/adaptive-content?unit=${encodeURIComponent(u.unitId)}`}
            className="text-sm font-medium text-accent-fg hover:underline dark:text-indigo-300"
          >
            Open in workspace
          </Link>
        </li>
      ))}
    </ol>
  )
}

/**
 * AC.9 — Instructor Adaptive Content course report (coverage, lift, units to review, cost).
 */
export function CourseReportPanel({ courseCode }: Props) {
  const [report, setReport] = useState<AdaptiveContentCourseReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [showUnitTable, setShowUnitTable] = useState(false)
  const unitTableId = useId()

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const r = await fetchAdaptiveContentCourseReport(courseCode)
      setReport(r)
    } catch (e) {
      setReport(null)
      setError(e instanceof Error ? e.message : 'Failed to load report.')
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    void load()
  }, [load])

  if (loading) {
    return (
      <div className="space-y-2" aria-busy="true" data-testid="ace-report-loading">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-14 animate-pulse rounded-xl bg-surface-sunken" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <p className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800" role="alert">
        {error}
      </p>
    )
  }

  if (!report) return null

  if (report.empty) {
    return (
      <section
        className="rounded-2xl border border-dashed border-border-strong px-6 py-10 text-center dark:border-border-default"
        data-testid="ace-report-empty"
      >
        <h3 className="text-base font-semibold text-fg-default">
          No adaptive data yet
        </h3>
        <p className="mx-auto mt-2 max-w-md text-sm text-fg-muted">
          Set up an adaptive content unit and collect servings and post-assessment outcomes to see
          coverage, lift vs. control, and units to review here.
        </p>
      </section>
    )
  }

  const budgetHint = report.cost.unlimited
    ? 'Unlimited budget'
    : `Remaining ${report.cost.budgetRemaining ?? 0} tokens`

  return (
    <div className="space-y-6" data-testid="ace-course-report">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold text-fg-default">
            Adaptive Content report
          </h3>
          {report.dataAsOf ? (
            <p className="mt-0.5 text-xs text-fg-muted">
              Data as of {new Date(report.dataAsOf).toLocaleString()}
            </p>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            disabled={busy}
            data-testid="ace-report-refresh"
            className="inline-flex items-center gap-1.5 rounded-xl border border-border-default px-3 py-1.5 text-sm font-medium text-fg-muted dark:border-border-default dark:text-fg-default"
            onClick={() => {
              void (async () => {
                setBusy(true)
                try {
                  await refreshAdaptiveContentEffectiveness(courseCode)
                  await load()
                } catch (e) {
                  setError(e instanceof Error ? e.message : 'Refresh failed.')
                } finally {
                  setBusy(false)
                }
              })()
            }}
          >
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
            Refresh
          </button>
          <button
            type="button"
            disabled={busy}
            data-testid="ace-report-export"
            className="inline-flex items-center gap-1.5 rounded-xl bg-slate-900 px-3 py-1.5 text-sm font-semibold text-white dark:bg-neutral-100 dark:text-neutral-900"
            onClick={() => {
              void (async () => {
                setBusy(true)
                try {
                  await exportAdaptiveContentCourseReportCsv(courseCode)
                } catch (e) {
                  setError(e instanceof Error ? e.message : 'Export failed.')
                } finally {
                  setBusy(false)
                }
              })()
            }}
          >
            <Download className="h-4 w-4" />
            Export CSV
          </button>
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4" data-testid="ace-report-kpis">
        <Kpi
          label="Coverage"
          value={formatPct(report.coverage.coveragePct)}
          hint={`${report.coverage.adaptedUnits} / ${report.coverage.eligibleContentItems} units adapted`}
        />
        <Kpi
          label="Students impacted"
          value={String(report.coverage.studentsServedVariant)}
          hint={`${report.coverage.studentsProfiled} profiled · ${report.coverage.studentsHoldout} holdout`}
        />
        <Kpi
          label="Mean lift vs control"
          value={formatLift(report.meanLiftVsControl)}
          hint={`${report.nHelping} helping · ${report.nRegressing} regressing`}
        />
        <Kpi
          label="Spend this period"
          value={`${report.cost.tokensUsedPeriod.toLocaleString()} tokens`}
          hint={budgetHint}
        />
      </div>

      <section aria-labelledby="ace-units-to-review-heading">
        <h4
          id="ace-units-to-review-heading"
          className="text-sm font-semibold text-fg-default"
        >
          Units to review
        </h4>
        <p className="mt-1 text-xs text-fg-muted">
          Ranked regressing → low fidelity → insufficient data. Open a unit in the authoring
          workspace to act.
        </p>
        <div className="mt-3">
          <UnitsToReviewList courseCode={courseCode} items={report.unitsToReview} />
        </div>
      </section>

      <section aria-labelledby="ace-mode-heading">
        <h4
          id="ace-mode-heading"
          className="text-sm font-semibold text-fg-default"
        >
          Effectiveness by emphasis mode
        </h4>
        <div className="mt-3">
          <ModeBarChart modes={report.byMode} />
        </div>
      </section>

      <section aria-labelledby="ace-unit-eff-heading">
        <div className="flex items-center justify-between gap-2">
          <h4
            id="ace-unit-eff-heading"
            className="text-sm font-semibold text-fg-default"
          >
            Effectiveness by unit
          </h4>
          <button
            type="button"
            className="text-xs font-medium text-accent-fg underline dark:text-indigo-300"
            aria-controls={unitTableId}
            aria-expanded={showUnitTable}
            onClick={() => setShowUnitTable((v) => !v)}
          >
            {showUnitTable ? 'Hide data table' : 'Show data table'}
          </button>
        </div>
        <ul className="mt-3 space-y-2">
          {report.units.map((u) => (
            <li
              key={u.unitId}
              className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border-default px-3 py-2 dark:border-border-default"
            >
              <span className="font-mono text-xs text-fg-muted">{u.unitId.slice(0, 8)}…</span>
              <EffectivenessChip effectiveness={u} />
              <span className="text-sm tabular-nums text-fg-muted">
                {formatLift(u.treatmentMinusHoldout)}
              </span>
            </li>
          ))}
          {report.units.length === 0 ? (
            <li className="text-sm text-fg-muted">No effectiveness rows yet.</li>
          ) : null}
        </ul>
        {showUnitTable ? (
          <table id={unitTableId} className="mt-3 w-full text-left text-sm">
            <caption className="sr-only">Per-unit effectiveness</caption>
            <thead>
              <tr className="border-b border-border-default">
                <th scope="col" className="py-1 pe-2">
                  Unit
                </th>
                <th scope="col" className="py-1 pe-2">
                  Verdict
                </th>
                <th scope="col" className="py-1 pe-2">
                  N treatment
                </th>
                <th scope="col" className="py-1 pe-2">
                  N holdout
                </th>
                <th scope="col" className="py-1">
                  Lift vs control
                </th>
              </tr>
            </thead>
            <tbody>
              {report.units.map((u) => (
                <tr key={u.unitId} className="border-b border-border-subtle">
                  <td className="py-1 pe-2 font-mono text-xs">{u.unitId}</td>
                  <td className="py-1 pe-2">{u.verdict}</td>
                  <td className="py-1 pe-2 tabular-nums">{u.nTreatment}</td>
                  <td className="py-1 pe-2 tabular-nums">{u.nHoldout}</td>
                  <td className="py-1 tabular-nums">{formatLift(u.treatmentMinusHoldout)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : null}
      </section>

      <section aria-labelledby="ace-cost-heading">
        <h4
          id="ace-cost-heading"
          className="text-sm font-semibold text-fg-default"
        >
          Cost &amp; budget
        </h4>
        <div className="mt-3">
          {report.cost.unlimited ? (
            <p className="text-sm text-fg-muted">
              {report.cost.tokensUsedPeriod.toLocaleString()} tokens used this period (unlimited
              budget). Period start {report.cost.periodStart}.
            </p>
          ) : (
            <>
              <div
                className="h-3 overflow-hidden rounded-full bg-surface-sunken"
                role="progressbar"
                aria-valuemin={0}
                aria-valuemax={report.cost.monthlyTokenBudget}
                aria-valuenow={report.cost.tokensUsedPeriod}
                aria-label="Token budget used this period"
              >
                <div
                  className="h-3 rounded-full bg-amber-600 dark:bg-amber-500"
                  style={{
                    width: `${Math.min(
                      100,
                      (report.cost.tokensUsedPeriod / Math.max(1, report.cost.monthlyTokenBudget)) *
                        100,
                    )}%`,
                  }}
                />
              </div>
              <p className="mt-2 text-sm text-fg-muted">
                {report.cost.tokensUsedPeriod.toLocaleString()} /{' '}
                {report.cost.monthlyTokenBudget.toLocaleString()} tokens ·{' '}
                {(report.cost.budgetRemaining ?? 0).toLocaleString()} remaining · period start{' '}
                {report.cost.periodStart}
              </p>
            </>
          )}
          <p className="mt-2 text-xs text-fg-muted">
            Platform AI reports include the same spend under feature{' '}
            <code className="rounded bg-surface-sunken px-1 dark:bg-surface-overlay">adaptive_content</code>.
          </p>
        </div>
      </section>
    </div>
  )
}

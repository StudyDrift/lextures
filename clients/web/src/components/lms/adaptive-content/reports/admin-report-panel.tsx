import { useCallback, useEffect, useState } from 'react'
import { Download, Loader2, RefreshCw } from 'lucide-react'
import { Link } from 'react-router-dom'
import {
  exportAdaptiveContentAdminReportCsv,
  fetchAdaptiveContentAdminReport,
  type AdaptiveContentAdminReport,
} from '../../../../lib/courses-api'

function formatLift(v: number | null | undefined): string {
  if (v == null || Number.isNaN(v)) return '—'
  const sign = v > 0 ? '+' : ''
  return `${sign}${v.toFixed(1)} pts`
}

/**
 * AC.9 — Admin org rollup for Adaptive Content (adoption, spend, disparities, incidents).
 */
export function AdminAdaptiveContentReportPanel() {
  const [report, setReport] = useState<AdaptiveContentAdminReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setReport(await fetchAdaptiveContentAdminReport())
    } catch (e) {
      setReport(null)
      setError(e instanceof Error ? e.message : 'Failed to load org report.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  if (loading) {
    return (
      <div className="mt-8 space-y-2" aria-busy="true">
        <div className="h-10 animate-pulse rounded-xl bg-surface-sunken" />
        <div className="h-24 animate-pulse rounded-xl bg-surface-sunken" />
      </div>
    )
  }

  if (error) {
    return (
      <p className="mt-8 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800" role="alert">
        {error}
      </p>
    )
  }

  if (!report) return null

  return (
    <section
      aria-labelledby="ace-admin-report-heading"
      className="mt-8"
      data-testid="ace-admin-report-panel"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2
            id="ace-admin-report-heading"
            className="text-base font-semibold text-fg-default"
          >
            Adaptive content org report
          </h2>
          <p className="mt-1 text-sm text-fg-muted">
            Adoption, spend, aggregate lift, disparity flags, and incident state across courses.
          </p>
          {report.dataAsOf ? (
            <p className="mt-0.5 text-xs text-fg-muted">
              Data as of {new Date(report.dataAsOf).toLocaleString()}
            </p>
          ) : null}
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            disabled={busy}
            className="inline-flex items-center gap-1.5 rounded-xl border border-border-default px-3 py-1.5 text-sm font-medium dark:border-border-default"
            onClick={() => {
              void (async () => {
                setBusy(true)
                try {
                  await load()
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
            data-testid="ace-admin-report-export"
            className="inline-flex items-center gap-1.5 rounded-xl bg-slate-900 px-3 py-1.5 text-sm font-semibold text-white dark:bg-neutral-100 dark:text-neutral-900"
            onClick={() => {
              void (async () => {
                setBusy(true)
                try {
                  await exportAdaptiveContentAdminReportCsv()
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

      <dl className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4" data-testid="ace-admin-report-kpis">
        {[
          ['Courses using ACE', String(report.coursesUsingAce)],
          ['Students impacted', String(report.studentsImpacted)],
          ['Spend (30d)', `$${report.costUsd30d.toFixed(2)}`],
          ['Budget headroom', `${report.budgetHeadroomTokens.toLocaleString()} tokens`],
          ['Aggregate lift', formatLift(report.aggregateLift)],
          ['Disparity flags', String(report.disparityFlags)],
          ['Open contests', String(report.openContests)],
          ['Regressing units', String(report.regressingUnits)],
          ['Kill-switch', report.killSwitch ? 'Engaged' : 'Disengaged'],
          ['Queue depth', String(report.queueDepth)],
        ].map(([label, value]) => (
          <div
            key={label}
            className="rounded-lg border border-border-default px-3 py-2 dark:border-border-default"
          >
            <dt className="text-xs font-medium uppercase tracking-wide text-fg-muted">{label}</dt>
            <dd className="mt-0.5 text-sm font-semibold text-fg-default">
              {value}
            </dd>
          </div>
        ))}
      </dl>

      <h3 className="mt-6 text-sm font-semibold text-fg-default">
        Courses
      </h3>
      {report.courses.length === 0 ? (
        <p className="mt-2 text-sm text-fg-muted" data-testid="ace-admin-report-empty">
          No courses have Adaptive Content enabled yet.
        </p>
      ) : (
        <div className="mt-2 overflow-x-auto">
          <table className="w-full min-w-[40rem] text-left text-sm">
            <caption className="sr-only">Adaptive content courses</caption>
            <thead>
              <tr className="border-b border-border-default">
                <th scope="col" className="py-2 pe-2 font-medium">
                  Course
                </th>
                <th scope="col" className="py-2 pe-2 font-medium">
                  Coverage
                </th>
                <th scope="col" className="py-2 pe-2 font-medium">
                  Lift
                </th>
                <th scope="col" className="py-2 pe-2 font-medium">
                  Regressing
                </th>
                <th scope="col" className="py-2 pe-2 font-medium">
                  Disparity
                </th>
                <th scope="col" className="py-2 font-medium">
                  Drill-down
                </th>
              </tr>
            </thead>
            <tbody>
              {report.courses.map((c) => (
                <tr
                  key={c.courseId}
                  className="border-b border-border-subtle"
                  data-testid="ace-admin-report-course-row"
                >
                  <td className="py-2 pe-2">
                    <span className="font-medium">{c.courseCode}</span>
                    <span className="ms-2 text-fg-muted">{c.title}</span>
                  </td>
                  <td className="py-2 pe-2 tabular-nums">{c.coveragePct.toFixed(0)}%</td>
                  <td className="py-2 pe-2 tabular-nums">{formatLift(c.meanLiftVsControl)}</td>
                  <td className="py-2 pe-2 tabular-nums">{c.nRegressing}</td>
                  <td className="py-2 pe-2 tabular-nums">{c.disparityFlags}</td>
                  <td className="py-2">
                    <Link
                      to={`/courses/${encodeURIComponent(c.courseCode)}/settings/adaptive-content?tab=report`}
                      className="font-medium text-accent-fg hover:underline dark:text-indigo-300"
                    >
                      Open report
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

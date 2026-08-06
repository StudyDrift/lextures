import { useCallback, useEffect, useMemo, useState } from 'react'
import { formatDateTime, formatNumber } from '../../lib/format'
import { providerLabel } from '../../lib/ai-providers'
import {
  aiFeatureLabel,
  aiReportsUtcRange,
  fetchAIReports,
  type AIReportsPayload,
  type AIReportsPreset,
} from '../../lib/ai-reports-api'

const PRESETS: { id: AIReportsPreset; label: string }[] = [
  { id: '24h', label: '24 hours' },
  { id: '7d', label: '7 days' },
  { id: '30d', label: '30 days' },
  { id: '90d', label: '90 days' },
]

function formatUsd(value: number): string {
  if (!Number.isFinite(value) || value === 0) return '$0.00'
  if (value < 0.01) return `$${value.toFixed(4)}`
  return `$${value.toFixed(2)}`
}

function formatRange(from: string, to: string): string {
  const opts: Intl.DateTimeFormatOptions = { dateStyle: 'medium', timeStyle: 'short' }
  return `${formatDateTime(new Date(from), opts)} → ${formatDateTime(new Date(to), opts)}`
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border-default bg-surface-raised px-4 py-3 dark:border-border-default dark:bg-surface-raised">
      <p className="text-xs font-medium uppercase tracking-wide text-fg-muted">{label}</p>
      <p className="mt-1 text-xl font-semibold tabular-nums text-fg-default">{value}</p>
    </div>
  )
}

export function AiReportsPanel() {
  const [preset, setPreset] = useState<AIReportsPreset>('24h')
  const [feature, setFeature] = useState('')
  const [provider, setProvider] = useState('')
  const [userQuery, setUserQuery] = useState('')
  const [courseCode, setCourseCode] = useState('')
  const [report, setReport] = useState<AIReportsPayload | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    const range = aiReportsUtcRange(preset)
    try {
      const data = await fetchAIReports({
        ...range,
        feature: feature.trim() || undefined,
        provider: provider.trim() || undefined,
        userQuery: userQuery.trim() || undefined,
        courseCode: courseCode.trim() || undefined,
      })
      setReport(data)
    } catch (e) {
      setReport(null)
      setError(e instanceof Error ? e.message : 'Could not load AI reports.')
    } finally {
      setLoading(false)
    }
  }, [preset, feature, provider, userQuery, courseCode])

  useEffect(() => {
    void load()
  }, [load])

  const featureOptions = useMemo(() => {
    const keys = new Set<string>(['adaptive_content'])
    for (const row of report?.cost.byFeature ?? []) keys.add(row.feature)
    return Array.from(keys).sort()
  }, [report])

  const aceCost = useMemo(
    () => report?.cost.byFeature.find((r) => r.feature === 'adaptive_content') ?? null,
    [report],
  )

  const providerOptions = useMemo(() => {
    const keys = new Set<string>(report?.providers ?? [])
    for (const row of report?.cost.byProvider ?? []) keys.add(row.provider)
    return Array.from(keys).sort()
  }, [report])

  return (
    <div>
      <h2 className="text-base font-semibold text-fg-default">Reports</h2>
      <p className="mt-1 text-sm text-fg-muted">
        Platform-wide AI spend and usage across configured providers. Costs may include estimates when
        a provider omits billing metadata.
      </p>

      <div className="mt-6 flex flex-wrap items-center gap-2">
        {PRESETS.map((p) => (
          <button
            key={p.id}
            type="button"
            onClick={() => setPreset(p.id)}
            className={`rounded-xl border px-3 py-2 text-sm font-semibold shadow-sm transition-[background-color,color,border-color] ${ preset === p.id ? 'border-indigo-300 bg-indigo-50 text-indigo-900 dark:border-indigo-500/50 dark:bg-indigo-950/60 dark:text-indigo-100' : 'border-border-default bg-surface-raised text-fg-muted hover:border-indigo-200 dark:border-border-default dark:bg-surface-raised dark:text-fg-default' }`}
          >
            {p.label}
          </button>
        ))}
      </div>

      {report && (
        <p className="mt-3 text-xs text-fg-muted">
          Window: {formatRange(report.range.from, report.range.to)}
        </p>
      )}

      {loading && <p className="mt-6 text-sm text-fg-muted">Loading…</p>}
      {error && (
        <p className="mt-6 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-100">
          {error}
        </p>
      )}

      {!loading && report && (
        <div className="mt-8 space-y-10">
          <section aria-labelledby="ai-cost-heading">
            <div className="flex flex-wrap items-end justify-between gap-3">
              <h3 id="ai-cost-heading" className="text-sm font-semibold text-fg-default">
                AI cost
              </h3>
              <div className="flex flex-wrap gap-2">
                <label className="text-sm text-fg-muted">
                  <span className="sr-only">Filter by provider</span>
                  <select
                    value={provider}
                    onChange={(e) => setProvider(e.target.value)}
                    className="rounded-lg border border-border-default bg-surface-raised px-2.5 py-1.5 text-sm dark:border-border-default dark:bg-surface-overlay"
                  >
                    <option value="">All providers</option>
                    {providerOptions.map((p) => (
                      <option key={p} value={p}>
                        {providerLabel(p)}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="text-sm text-fg-muted">
                  <span className="sr-only">Filter by feature</span>
                  <select
                    value={feature}
                    onChange={(e) => setFeature(e.target.value)}
                    className="rounded-lg border border-border-default bg-surface-raised px-2.5 py-1.5 text-sm dark:border-border-default dark:bg-surface-overlay"
                  >
                    <option value="">All features</option>
                    {featureOptions.map((f) => (
                      <option key={f} value={f}>
                        {aiFeatureLabel(f)}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
            </div>

            <div className="mt-4 grid gap-3 sm:grid-cols-3">
              <SummaryCard label="Total cost" value={formatUsd(report.cost.summary.totalCostUsd)} />
              <SummaryCard label="API calls" value={formatNumber(report.cost.summary.totalCalls)} />
              <SummaryCard label="Tokens" value={formatNumber(report.cost.summary.totalTokens)} />
            </div>

            {(report.cost.byProvider?.length ?? 0) > 0 && (
              <div className="mt-5 overflow-x-auto rounded-xl border border-border-default">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-surface-base text-xs uppercase tracking-wide text-fg-muted/80 dark:text-fg-muted">
                    <tr>
                      <th className="px-4 py-2.5 font-semibold">Provider</th>
                      <th className="px-4 py-2.5 font-semibold">Cost</th>
                      <th className="px-4 py-2.5 font-semibold">Calls</th>
                      <th className="px-4 py-2.5 font-semibold">Tokens</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                    {report.cost.byProvider.map((row) => (
                      <tr key={row.provider}>
                        <td className="px-4 py-2.5 text-fg-default">
                          {providerLabel(row.provider)}
                        </td>
                        <td className="px-4 py-2.5 tabular-nums">{formatUsd(row.costUsd)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.calls)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.tokens)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {report.cost.byDay.length > 0 && (
              <div className="mt-5 overflow-x-auto rounded-xl border border-border-default">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-surface-base text-xs uppercase tracking-wide text-fg-muted/80 dark:text-fg-muted">
                    <tr>
                      <th className="px-4 py-2.5 font-semibold">Day (UTC)</th>
                      <th className="px-4 py-2.5 font-semibold">Cost</th>
                      <th className="px-4 py-2.5 font-semibold">Calls</th>
                      <th className="px-4 py-2.5 font-semibold">Tokens</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                    {report.cost.byDay.map((row) => (
                      <tr key={row.day}>
                        <td className="px-4 py-2.5 text-fg-default">{row.day}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatUsd(row.costUsd)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.calls)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.tokens)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {report.cost.byFeature.length > 0 && (
              <div className="mt-5 overflow-x-auto rounded-xl border border-border-default">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-surface-base text-xs uppercase tracking-wide text-fg-muted/80 dark:text-fg-muted">
                    <tr>
                      <th className="px-4 py-2.5 font-semibold">Feature</th>
                      <th className="px-4 py-2.5 font-semibold">Cost</th>
                      <th className="px-4 py-2.5 font-semibold">Calls</th>
                      <th className="px-4 py-2.5 font-semibold">Tokens</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                    {report.cost.byFeature.map((row) => (
                      <tr key={row.feature}>
                        <td className="px-4 py-2.5 text-fg-default">
                          {aiFeatureLabel(row.feature)}
                        </td>
                        <td className="px-4 py-2.5 tabular-nums">{formatUsd(row.costUsd)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.calls)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.tokens)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            <section
              aria-labelledby="ace-ai-cost-heading"
              className="mt-6 rounded-xl border border-border-default px-4 py-3 dark:border-border-default"
              data-testid="ace-ai-reports-slice"
            >
              <h4
                id="ace-ai-cost-heading"
                className="text-sm font-semibold text-fg-default"
              >
                Adaptive Content Engine
              </h4>
              <p className="mt-1 text-xs text-fg-muted">
                Cost slice for feature <code>adaptive_content</code> (AC.9). Filter the reports above
                by this feature for course/user drill-down.
              </p>
              {aceCost ? (
                <dl className="mt-3 grid gap-2 sm:grid-cols-3">
                  <div>
                    <dt className="text-xs uppercase text-fg-muted">Cost</dt>
                    <dd className="text-sm font-semibold tabular-nums">{formatUsd(aceCost.costUsd)}</dd>
                  </div>
                  <div>
                    <dt className="text-xs uppercase text-fg-muted">Calls</dt>
                    <dd className="text-sm font-semibold tabular-nums">
                      {formatNumber(aceCost.calls)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-xs uppercase text-fg-muted">Tokens</dt>
                    <dd className="text-sm font-semibold tabular-nums">
                      {formatNumber(aceCost.tokens)}
                    </dd>
                  </div>
                </dl>
              ) : (
                <p className="mt-2 text-sm text-fg-muted">
                  No Adaptive Content usage in this window.
                </p>
              )}
              <button
                type="button"
                className="mt-3 text-sm font-medium text-accent-fg underline dark:text-indigo-300"
                onClick={() => setFeature('adaptive_content')}
              >
                Filter reports to Adaptive content
              </button>
            </section>

            {report.cost.summary.totalCalls === 0 && (
              <p className="mt-4 text-sm text-fg-muted">
                No AI usage recorded in this window.
              </p>
            )}
          </section>

          <section aria-labelledby="ai-by-user-heading">
            <div className="flex flex-wrap items-end justify-between gap-3">
              <h3 id="ai-by-user-heading" className="text-sm font-semibold text-fg-default">
                AI usage by user
              </h3>
              <label className="flex min-w-[12rem] flex-1 flex-col text-sm text-fg-muted sm:max-w-xs">
                <span className="mb-1 text-xs font-medium">Search user</span>
                <input
                  type="search"
                  value={userQuery}
                  onChange={(e) => setUserQuery(e.target.value)}
                  placeholder="Email or name"
                  className="rounded-lg border border-border-default bg-surface-raised px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
                />
              </label>
            </div>

            {report.byUser.length > 0 ? (
              <div className="mt-4 overflow-x-auto rounded-xl border border-border-default">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-surface-base text-xs uppercase tracking-wide text-fg-muted/80 dark:text-fg-muted">
                    <tr>
                      <th className="px-4 py-2.5 font-semibold">User</th>
                      <th className="px-4 py-2.5 font-semibold">Calls</th>
                      <th className="px-4 py-2.5 font-semibold">Tokens</th>
                      <th className="px-4 py-2.5 font-semibold">Cost</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                    {report.byUser.map((row) => (
                      <tr key={row.userId}>
                        <td className="px-4 py-2.5">
                          <div className="font-medium text-fg-default">{row.displayName}</div>
                          <div className="text-xs text-fg-muted">{row.email}</div>
                        </td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.calls)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.totalTokens)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatUsd(row.costUsd)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="mt-4 text-sm text-fg-muted">No user usage in this window.</p>
            )}
          </section>

          <section aria-labelledby="ai-by-course-heading">
            <div className="flex flex-wrap items-end justify-between gap-3">
              <h3 id="ai-by-course-heading" className="text-sm font-semibold text-fg-default">
                AI usage by course
              </h3>
              <label className="flex min-w-[12rem] flex-1 flex-col text-sm text-fg-muted sm:max-w-xs">
                <span className="mb-1 text-xs font-medium">Search course</span>
                <input
                  type="search"
                  value={courseCode}
                  onChange={(e) => setCourseCode(e.target.value)}
                  placeholder="Course code or title"
                  className="rounded-lg border border-border-default bg-surface-raised px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
                />
              </label>
            </div>

            {report.byCourse.length > 0 ? (
              <div className="mt-4 overflow-x-auto rounded-xl border border-border-default">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-surface-base text-xs uppercase tracking-wide text-fg-muted/80 dark:text-fg-muted">
                    <tr>
                      <th className="px-4 py-2.5 font-semibold">Course</th>
                      <th className="px-4 py-2.5 font-semibold">Calls</th>
                      <th className="px-4 py-2.5 font-semibold">Tokens</th>
                      <th className="px-4 py-2.5 font-semibold">Cost</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                    {report.byCourse.map((row) => (
                      <tr key={row.courseId}>
                        <td className="px-4 py-2.5">
                          <div className="font-medium text-fg-default">{row.title}</div>
                          <div className="text-xs text-fg-muted">{row.courseCode}</div>
                        </td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.calls)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatNumber(row.totalTokens)}</td>
                        <td className="px-4 py-2.5 tabular-nums">{formatUsd(row.costUsd)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="mt-4 text-sm text-fg-muted">No course usage in this window.</p>
            )}
          </section>
        </div>
      )}
    </div>
  )
}

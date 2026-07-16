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
    <div className="rounded-xl border border-slate-200 bg-white px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">{label}</p>
      <p className="mt-1 text-xl font-semibold tabular-nums text-slate-900 dark:text-neutral-100">{value}</p>
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
    const keys = new Set<string>()
    for (const row of report?.cost.byFeature ?? []) keys.add(row.feature)
    return Array.from(keys).sort()
  }, [report])

  const providerOptions = useMemo(() => {
    const keys = new Set<string>(report?.providers ?? [])
    for (const row of report?.cost.byProvider ?? []) keys.add(row.provider)
    return Array.from(keys).sort()
  }, [report])

  return (
    <div>
      <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Reports</h2>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        Platform-wide AI spend and usage across configured providers. Costs may include estimates when
        a provider omits billing metadata.
      </p>

      <div className="mt-6 flex flex-wrap items-center gap-2">
        {PRESETS.map((p) => (
          <button
            key={p.id}
            type="button"
            onClick={() => setPreset(p.id)}
            className={`rounded-xl border px-3 py-2 text-sm font-semibold shadow-sm transition-[background-color,color,border-color] ${
              preset === p.id
                ? 'border-indigo-300 bg-indigo-50 text-indigo-900 dark:border-indigo-500/50 dark:bg-indigo-950/60 dark:text-indigo-100'
                : 'border-slate-200 bg-white text-slate-700 hover:border-indigo-200 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200'
            }`}
          >
            {p.label}
          </button>
        ))}
      </div>

      {report && (
        <p className="mt-3 text-xs text-slate-500 dark:text-neutral-400">
          Window: {formatRange(report.range.from, report.range.to)}
        </p>
      )}

      {loading && <p className="mt-6 text-sm text-slate-500">Loading…</p>}
      {error && (
        <p className="mt-6 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-100">
          {error}
        </p>
      )}

      {!loading && report && (
        <div className="mt-8 space-y-10">
          <section aria-labelledby="ai-cost-heading">
            <div className="flex flex-wrap items-end justify-between gap-3">
              <h3 id="ai-cost-heading" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                AI cost
              </h3>
              <div className="flex flex-wrap gap-2">
                <label className="text-sm text-slate-600 dark:text-neutral-300">
                  <span className="sr-only">Filter by provider</span>
                  <select
                    value={provider}
                    onChange={(e) => setProvider(e.target.value)}
                    className="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                  >
                    <option value="">All providers</option>
                    {providerOptions.map((p) => (
                      <option key={p} value={p}>
                        {providerLabel(p)}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="text-sm text-slate-600 dark:text-neutral-300">
                  <span className="sr-only">Filter by feature</span>
                  <select
                    value={feature}
                    onChange={(e) => setFeature(e.target.value)}
                    className="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-800"
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
              <div className="mt-5 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-700">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-neutral-800/80 dark:text-neutral-400">
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
                        <td className="px-4 py-2.5 text-slate-800 dark:text-neutral-200">
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
              <div className="mt-5 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-700">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-neutral-800/80 dark:text-neutral-400">
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
                        <td className="px-4 py-2.5 text-slate-800 dark:text-neutral-200">{row.day}</td>
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
              <div className="mt-5 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-700">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-neutral-800/80 dark:text-neutral-400">
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
                        <td className="px-4 py-2.5 text-slate-800 dark:text-neutral-200">
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

            {report.cost.summary.totalCalls === 0 && (
              <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">
                No AI usage recorded in this window.
              </p>
            )}
          </section>

          <section aria-labelledby="ai-by-user-heading">
            <div className="flex flex-wrap items-end justify-between gap-3">
              <h3 id="ai-by-user-heading" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                AI usage by user
              </h3>
              <label className="flex min-w-[12rem] flex-1 flex-col text-sm text-slate-600 dark:text-neutral-300 sm:max-w-xs">
                <span className="mb-1 text-xs font-medium">Search user</span>
                <input
                  type="search"
                  value={userQuery}
                  onChange={(e) => setUserQuery(e.target.value)}
                  placeholder="Email or name"
                  className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                />
              </label>
            </div>

            {report.byUser.length > 0 ? (
              <div className="mt-4 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-700">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-neutral-800/80 dark:text-neutral-400">
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
                          <div className="font-medium text-slate-900 dark:text-neutral-100">{row.displayName}</div>
                          <div className="text-xs text-slate-500 dark:text-neutral-400">{row.email}</div>
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
              <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">No user usage in this window.</p>
            )}
          </section>

          <section aria-labelledby="ai-by-course-heading">
            <div className="flex flex-wrap items-end justify-between gap-3">
              <h3 id="ai-by-course-heading" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                AI usage by course
              </h3>
              <label className="flex min-w-[12rem] flex-1 flex-col text-sm text-slate-600 dark:text-neutral-300 sm:max-w-xs">
                <span className="mb-1 text-xs font-medium">Search course</span>
                <input
                  type="search"
                  value={courseCode}
                  onChange={(e) => setCourseCode(e.target.value)}
                  placeholder="Course code or title"
                  className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                />
              </label>
            </div>

            {report.byCourse.length > 0 ? (
              <div className="mt-4 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-700">
                <table className="min-w-full text-start text-sm">
                  <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-neutral-800/80 dark:text-neutral-400">
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
                          <div className="font-medium text-slate-900 dark:text-neutral-100">{row.title}</div>
                          <div className="text-xs text-slate-500 dark:text-neutral-400">{row.courseCode}</div>
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
              <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">No course usage in this window.</p>
            )}
          </section>
        </div>
      )}
    </div>
  )
}

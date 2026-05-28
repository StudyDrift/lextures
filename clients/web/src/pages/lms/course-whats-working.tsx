import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { RefreshCw, TrendingUp, AlertTriangle, X } from 'lucide-react'
import { LmsPage } from './lms-page'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchInsights,
  refreshInsights,
  dismissSignal,
  fetchCrossSection,
  type Insights,
  type CrossSectionRow,
} from '../../lib/instructor-insights-api'

function SignalCard({
  item,
  onDismiss,
}: {
  item: Insights['workingWell'][number]
  onDismiss: () => void
}) {
  const [dismissing, setDismissing] = useState(false)

  async function handleDismiss() {
    setDismissing(true)
    try {
      onDismiss()
    } finally {
      setDismissing(false)
    }
  }

  return (
    <div className="flex items-start justify-between gap-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
      <div className="min-w-0 flex-1">
        <p className="truncate font-medium text-slate-900 dark:text-neutral-100">{item.title}</p>
        <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400 capitalize">{item.kind}</p>
        <p className="mt-2 text-sm text-slate-700 dark:text-neutral-300">{item.narrative}</p>
        <div className="mt-2 flex flex-wrap gap-3 text-xs tabular-nums text-slate-600 dark:text-neutral-400">
          <span>Completion {(item.completionRate * 100).toFixed(0)}%</span>
          {item.avgScore != null && <span>Avg score {item.avgScore.toFixed(1)}%</span>}
          <span>Engagement {item.engagement}s</span>
          {item.difficulty != null && <span>Difficulty {item.difficulty.toFixed(1)}%</span>}
        </div>
      </div>
      <button
        type="button"
        aria-label={`Dismiss signal for ${item.title}`}
        onClick={() => void handleDismiss()}
        disabled={dismissing}
        className="shrink-0 rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-700 disabled:opacity-40 dark:text-neutral-500 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
      >
        <X className="h-4 w-4" aria-hidden />
      </button>
    </div>
  )
}

function ScatterTable({ points }: { points: Insights['scatter'] | null | undefined }) {
  if (!points || points.length === 0) return null
  const flagged = points.filter((p) => p.flag === 'needs_redesign')
  if (flagged.length === 0) return null

  return (
    <div className="mt-8">
      <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
        Content that may need redesign
      </h2>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        High difficulty + low engagement — students struggle and disengage.
      </p>
      <div className="mt-3 overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
        <table className="min-w-full text-start text-sm">
          <caption className="sr-only">Content flagged for possible redesign</caption>
          <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:border-neutral-700 dark:bg-neutral-800/80 dark:text-neutral-300">
            <tr>
              <th scope="col" className="px-4 py-3">Item</th>
              <th scope="col" className="px-4 py-3 text-end">Difficulty %</th>
              <th scope="col" className="px-4 py-3 text-end">Engagement (s)</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
            {flagged.map((p) => (
              <tr key={p.itemId}>
                <td className="px-4 py-3 font-medium text-slate-900 dark:text-neutral-100">{p.title}</td>
                <td className="px-4 py-3 text-end tabular-nums text-slate-700 dark:text-neutral-300">{p.difficulty.toFixed(1)}</td>
                <td className="px-4 py-3 text-end tabular-nums text-slate-700 dark:text-neutral-300">{p.engagement}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function CrossSectionTable({ rows }: { rows: CrossSectionRow[] }) {
  if (rows.length === 0) return null

  return (
    <div className="mt-8">
      <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
        Cross-section comparison
      </h2>
      <div className="mt-3 overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
        <table className="min-w-full text-start text-sm">
          <caption className="sr-only">Cross-section performance comparison</caption>
          <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:border-neutral-700 dark:bg-neutral-800/80 dark:text-neutral-300">
            <tr>
              <th scope="col" className="px-4 py-3">Section</th>
              <th scope="col" className="px-4 py-3 text-end">Students</th>
              <th scope="col" className="px-4 py-3 text-end">Avg quiz score</th>
              <th scope="col" className="px-4 py-3 text-end">Completion</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
            {rows.map((row) => (
              <tr key={row.sectionId}>
                <td className="px-4 py-3 font-medium text-slate-900 dark:text-neutral-100">{row.sectionName}</td>
                <td className="px-4 py-3 text-end tabular-nums text-slate-700 dark:text-neutral-300">{row.nStudents}</td>
                <td className="px-4 py-3 text-end tabular-nums text-slate-700 dark:text-neutral-300">
                  {row.avgQuizScore != null ? `${row.avgQuizScore.toFixed(1)}%` : '—'}
                </td>
                <td className="px-4 py-3 text-end tabular-nums text-slate-700 dark:text-neutral-300">
                  {(row.completionRate * 100).toFixed(0)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export default function CourseWhatsWorking() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const { instructorInsightsEnabled, loading: featuresLoading } = usePlatformFeatures()
  const [insights, setInsights] = useState<Insights | null>(null)
  const [crossSection, setCrossSection] = useState<CrossSectionRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshing, setRefreshing] = useState(false)

  const load = useCallback(async () => {
    if (!courseCode || !instructorInsightsEnabled) return
    setLoading(true)
    setError(null)
    try {
      const [ins, cs] = await Promise.all([
        fetchInsights(courseCode),
        fetchCrossSection(courseCode),
      ])
      setInsights(ins)
      setCrossSection(Array.isArray(cs) ? cs : [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load insights.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, instructorInsightsEnabled])

  useEffect(() => {
    if (featuresLoading || !instructorInsightsEnabled) return
    void load()
  }, [load, featuresLoading, instructorInsightsEnabled])

  async function handleRefresh() {
    if (!courseCode) return
    setRefreshing(true)
    try {
      const ins = await refreshInsights(courseCode)
      setInsights(ins)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Refresh failed.')
    } finally {
      setRefreshing(false)
    }
  }

  async function handleDismiss(itemId: string) {
    if (!courseCode) return
    try {
      await dismissSignal(courseCode, itemId, '')
      setInsights((prev) => {
        if (!prev) return prev
        return {
          ...prev,
          workingWell: prev.workingWell.filter((s) => s.itemId !== itemId),
          needsAttention: prev.needsAttention.filter((s) => s.itemId !== itemId),
        }
      })
    } catch {
      /* ignore */
    }
  }

  if (featuresLoading) {
    return (
      <LmsPage title="What's working" description="Instructor signals for course content.">
        <p className="mt-8 text-sm text-slate-500 dark:text-neutral-400" aria-live="polite">Loading…</p>
      </LmsPage>
    )
  }

  if (!instructorInsightsEnabled) {
    return (
      <LmsPage title="What's working" description="Instructor signals for course content.">
        <p className="mt-8 text-sm text-slate-600 dark:text-neutral-400">
          Instructor insights are not enabled on this platform. Ask a global administrator to turn on
          &quot;Instructor insights&quot; in Settings → Global platform.
        </p>
      </LmsPage>
    )
  }

  const generatedAt = insights?.generatedAt
    ? new Date(insights.generatedAt).toLocaleString(undefined, {
        dateStyle: 'medium',
        timeStyle: 'short',
      })
    : null

  return (
    <LmsPage
      title="What's working"
      description="Signals about which content is landing well and which needs attention."
      actions={
        <button
          type="button"
          onClick={() => void handleRefresh()}
          disabled={refreshing}
          aria-label="Refresh insights"
          className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition hover:border-indigo-200 hover:bg-indigo-50/60 disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200"
        >
          <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} aria-hidden />
          {refreshing ? 'Refreshing…' : 'Refresh'}
        </button>
      }
    >
      {loading && (
        <p className="mt-8 text-sm text-slate-500 dark:text-neutral-400" aria-live="polite">
          Loading insights…
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
      {!loading && !error && insights && (
        <>
          {generatedAt && (
            <p className="mt-4 text-xs text-slate-500 dark:text-neutral-400">
              Last computed {generatedAt}
            </p>
          )}

          <section aria-labelledby="working-well-heading" className="mt-6">
            <h2
              id="working-well-heading"
              className="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-neutral-100"
            >
              <TrendingUp className="h-4 w-4 text-emerald-500" aria-hidden />
              Working well
            </h2>
            {insights.workingWell.length === 0 ? (
              <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">
                No top-performing items yet — add more student activity to see signals.
              </p>
            ) : (
              <div className="mt-3 space-y-3">
                {insights.workingWell.map((item) => (
                  <SignalCard
                    key={item.itemId}
                    item={item}
                    onDismiss={() => void handleDismiss(item.itemId)}
                  />
                ))}
              </div>
            )}
          </section>

          <section aria-labelledby="needs-attention-heading" className="mt-8">
            <h2
              id="needs-attention-heading"
              className="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-neutral-100"
            >
              <AlertTriangle className="h-4 w-4 text-amber-500" aria-hidden />
              Needs attention
            </h2>
            {insights.needsAttention.length === 0 ? (
              <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">
                No items flagged — everything looks good.
              </p>
            ) : (
              <div className="mt-3 space-y-3">
                {insights.needsAttention.map((item) => (
                  <SignalCard
                    key={item.itemId}
                    item={item}
                    onDismiss={() => void handleDismiss(item.itemId)}
                  />
                ))}
              </div>
            )}
          </section>

          <ScatterTable points={insights.scatter} />
          <CrossSectionTable rows={crossSection} />
        </>
      )}
    </LmsPage>
  )
}

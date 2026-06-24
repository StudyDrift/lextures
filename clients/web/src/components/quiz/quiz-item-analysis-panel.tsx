import { useCallback, useEffect, useState } from 'react'
import { AlertTriangle, Download, Flame, RefreshCw, Sun } from 'lucide-react'
import {
  computeItemAnalysis,
  fetchItemAnalysis,
  itemAnalysisExportUrl,
  type ItemAnalysisResult,
  type ItemStat,
  type TestStats,
} from '../../lib/item-analysis-api'
import { authorizedFetch } from '../../lib/api'

interface Props {
  courseCode: string
  itemId: string
}

/** Displays classical test theory statistics for a quiz (instructor-only). */
export function QuizItemAnalysisPanel({ courseCode, itemId }: Props) {
  const [result, setResult] = useState<ItemAnalysisResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [computing, setComputing] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchItemAnalysis(courseCode, itemId)
      setResult(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load item analysis.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    void load()
  }, [load])

  const handleCompute = useCallback(async () => {
    setComputing(true)
    setError(null)
    try {
      const data = await computeItemAnalysis(courseCode, itemId)
      setResult(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Computation failed.')
    } finally {
      setComputing(false)
    }
  }, [courseCode, itemId])

  const handleExportCSV = useCallback(() => {
    const url = itemAnalysisExportUrl(courseCode, itemId)
    // Use authorizedFetch to download with auth token then trigger browser download
    void (async () => {
      try {
        const res = await authorizedFetch(url)
        if (!res.ok) throw new Error(`Export failed (${res.status})`)
        const blob = await res.blob()
        const a = document.createElement('a')
        a.href = URL.createObjectURL(blob)
        a.download = `item-analysis-${itemId.slice(0, 8)}.csv`
        a.click()
        URL.revokeObjectURL(a.href)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Export failed.')
      }
    })()
  }, [courseCode, itemId])

  return (
    <section
      aria-labelledby="item-analysis-heading"
      className="mt-6 rounded-2xl border border-slate-200/90 bg-white p-5 dark:border-neutral-600 dark:bg-neutral-950/80"
    >
      <div className="flex items-center justify-between gap-4">
        <h2
          id="item-analysis-heading"
          className="text-base font-semibold text-slate-900 dark:text-neutral-100"
        >
          Item Analysis
        </h2>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => void handleCompute()}
            disabled={computing || loading}
            aria-label="Recompute item statistics"
            className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-600 transition-[background-color,color,border-color] hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-300"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${computing ? 'animate-spin' : ''}`} aria-hidden />
            {computing ? 'Computing…' : 'Recompute'}
          </button>
          {result?.status === 'ok' && (
            <button
              type="button"
              onClick={handleExportCSV}
              aria-label="Export item analysis as CSV"
              className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-600 transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-300"
            >
              <Download className="h-3.5 w-3.5" aria-hidden />
              Export CSV
            </button>
          )}
        </div>
      </div>

      {error && (
        <p role="alert" className="mt-3 text-sm text-rose-600 dark:text-rose-400">
          {error}
        </p>
      )}

      {loading && (
        <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
      )}

      {!loading && result?.status === 'insufficient' && (
        <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">
          Not enough responses yet — at least {result.minimumRequired} are required per question
          (current: {result.nResponses}).
        </p>
      )}

      {!loading && result?.status === 'pending' && (
        <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">
          Statistics are pending — click Recompute to generate them.
        </p>
      )}

      {!loading && result?.status === 'ok' && (
        <>
          <TestStatsSummary stats={result.testStats} />
          <ItemStatsTable items={result.itemStats} />
        </>
      )}
    </section>
  )
}

function TestStatsSummary({ stats }: { stats: TestStats }) {
  const reliability = stats.kr20 ?? stats.cronbachAlpha
  const reliabilityLabel = stats.kr20 != null ? 'KR-20' : 'Cronbach α'

  return (
    <dl className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
      <SummaryCard label="Responses" value={String(stats.nResponses)} />
      {stats.meanScore != null && (
        <SummaryCard label="Mean score" value={`${stats.meanScore.toFixed(1)}%`} />
      )}
      {stats.stdDev != null && (
        <SummaryCard label="Std dev" value={`${stats.stdDev.toFixed(1)}%`} />
      )}
      {reliability != null && (
        <SummaryCard
          label={reliabilityLabel}
          value={reliability.toFixed(3)}
          warn={reliability < 0.7}
        />
      )}
    </dl>
  )
}

function SummaryCard({
  label,
  value,
  warn = false,
}: {
  label: string
  value: string
  warn?: boolean
}) {
  return (
    <div className="rounded-xl border border-slate-100 bg-slate-50 px-3 py-2.5 dark:border-neutral-700 dark:bg-neutral-900">
      <dt className="text-[11px] uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        {label}
      </dt>
      <dd
        className={`mt-0.5 text-lg font-semibold tabular-nums ${
          warn ? 'text-amber-600 dark:text-amber-400' : 'text-slate-900 dark:text-neutral-100'
        }`}
      >
        {value}
      </dd>
    </div>
  )
}

function ItemStatsTable({ items }: { items: ItemStat[] }) {
  if (items.length === 0) {
    return (
      <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">No item data available.</p>
    )
  }

  return (
    <div className="mt-4 overflow-x-auto">
      <table className="w-full min-w-[640px] text-sm" aria-label="Item analysis statistics">
        <caption className="sr-only">
          Per-question difficulty, discrimination, distractor breakdown, and quality flags
        </caption>
        <thead>
          <tr className="border-b border-slate-100 dark:border-neutral-700">
            <th scope="col" className="pb-2 pe-4 text-start text-xs font-semibold text-slate-500 dark:text-neutral-400">
              #
            </th>
            <th scope="col" className="pb-2 pe-4 text-start text-xs font-semibold text-slate-500 dark:text-neutral-400">
              Question
            </th>
            <th scope="col" className="pb-2 pe-4 text-end text-xs font-semibold text-slate-500 dark:text-neutral-400">
              N
            </th>
            <th scope="col" className="pb-2 pe-4 text-end text-xs font-semibold text-slate-500 dark:text-neutral-400">
              Difficulty (p)
            </th>
            <th scope="col" className="pb-2 pe-4 text-end text-xs font-semibold text-slate-500 dark:text-neutral-400">
              Discrimination (r<sub>pb</sub>)
            </th>
            <th scope="col" className="pb-2 pe-4 text-start text-xs font-semibold text-slate-500 dark:text-neutral-400">
              Distractors
            </th>
            <th scope="col" className="pb-2 text-start text-xs font-semibold text-slate-500 dark:text-neutral-400">
              Flag
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-50 dark:divide-neutral-800">
          {items.map((item) => (
            <ItemRow key={item.questionIndex} item={item} />
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ItemRow({ item }: { item: ItemStat }) {
  const truncatedText =
    item.questionText.length > 80
      ? item.questionText.slice(0, 80) + '…'
      : item.questionText

  return (
    <tr className="py-1.5 align-top">
      <td className="py-1.5 pe-4 tabular-nums text-slate-500 dark:text-neutral-400">
        {item.questionIndex + 1}
      </td>
      <td className="py-1.5 pe-4 text-slate-700 dark:text-neutral-300" title={item.questionText}>
        {truncatedText || <span className="italic text-slate-400">No text</span>}
      </td>
      <td className="py-1.5 pe-4 text-end tabular-nums text-slate-700 dark:text-neutral-300">
        {item.nResponses}
      </td>
      <td className="py-1.5 pe-4 text-end">
        <DifficultyBar pValue={item.pValue} />
      </td>
      <td className="py-1.5 pe-4 text-end tabular-nums text-slate-700 dark:text-neutral-300">
        {item.rPb != null ? item.rPb.toFixed(3) : '—'}
      </td>
      <td className="py-1.5 pe-4">
        <DistractorBar freqs={item.distractorFreqs} />
      </td>
      <td className="py-1.5">
        <FlagBadge flag={item.flag} />
      </td>
    </tr>
  )
}

function DifficultyBar({ pValue }: { pValue: number | null }) {
  if (pValue == null) return <span className="text-slate-400">—</span>

  const pct = Math.round(pValue * 100)
  const color =
    pValue < 0.2
      ? 'bg-rose-500'
      : pValue > 0.9
        ? 'bg-amber-400'
        : 'bg-emerald-500'

  return (
    <span className="inline-flex items-center gap-1.5">
      <span
        role="img"
        aria-label={`Difficulty ${pct}%`}
        className="inline-block h-2 w-16 overflow-hidden rounded-full bg-slate-100 dark:bg-neutral-700"
      >
        <span
          className={`block h-full rounded-full ${color}`}
          style={{ width: `${pct}%` }}
        />
      </span>
      <span className="tabular-nums text-slate-700 dark:text-neutral-300">{pct}%</span>
    </span>
  )
}

const DISTRACTOR_LABELS = ['A', 'B', 'C', 'D', 'E', 'F']
const DISTRACTOR_COLORS = [
  'bg-indigo-400',
  'bg-sky-400',
  'bg-teal-400',
  'bg-amber-400',
  'bg-rose-400',
  'bg-purple-400',
]

function DistractorBar({ freqs }: { freqs: Record<string, number> | null }) {
  if (!freqs) return <span className="text-slate-400">—</span>

  const entries = DISTRACTOR_LABELS.filter((l) => l in freqs).map((l) => ({
    label: l,
    pct: Math.round((freqs[l] ?? 0) * 100),
    color: DISTRACTOR_COLORS[DISTRACTOR_LABELS.indexOf(l)] ?? 'bg-slate-400',
  }))

  if (entries.length === 0) return <span className="text-slate-400">—</span>

  return (
    <span className="inline-flex items-center gap-0.5" aria-label={entries.map((e) => `${e.label}: ${e.pct}%`).join(', ')}>
      {entries.map((e) => (
        <span
          key={e.label}
          title={`${e.label}: ${e.pct}%`}
          className={`inline-block h-3 rounded-sm ${e.color}`}
          style={{ width: `${Math.max(e.pct, 2)}px` }}
        />
      ))}
    </span>
  )
}

function FlagBadge({ flag }: { flag: ItemStat['flag'] }) {
  if (!flag) return null

  if (flag === 'hard') {
    return (
      <span
        role="img"
        aria-label="Very hard — p < 0.20, consider revising"
        title="Very hard — p < 0.20"
        className="inline-flex items-center gap-1 rounded-md bg-rose-50 px-1.5 py-0.5 text-[11px] font-medium text-rose-700 dark:bg-rose-950/40 dark:text-rose-400"
      >
        <Flame className="h-3 w-3" aria-hidden />
        Very hard
      </span>
    )
  }
  if (flag === 'easy') {
    return (
      <span
        role="img"
        aria-label="Very easy — p > 0.90, consider revising"
        title="Very easy — p > 0.90"
        className="inline-flex items-center gap-1 rounded-md bg-amber-50 px-1.5 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-950/40 dark:text-amber-400"
      >
        <Sun className="h-3 w-3" aria-hidden />
        Very easy
      </span>
    )
  }
  if (flag === 'poor_discriminator') {
    return (
      <span
        role="img"
        aria-label="Poor discriminator — r_pb < 0.15, consider revising"
        title="Poor discriminator — r_pb < 0.15"
        className="inline-flex items-center gap-1 rounded-md bg-orange-50 px-1.5 py-0.5 text-[11px] font-medium text-orange-700 dark:bg-orange-950/40 dark:text-orange-400"
      >
        <AlertTriangle className="h-3 w-3" aria-hidden />
        Poor disc.
      </span>
    )
  }
  return null
}

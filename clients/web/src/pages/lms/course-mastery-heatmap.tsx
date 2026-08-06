import { useCallback, useEffect, useState } from 'react'
import { formatDateTime } from '../../lib/format'
import { useParams } from 'react-router-dom'
import { RefreshCw } from 'lucide-react'
import { LmsPage } from './lms-page'
import {
  fetchMasteryHeatmap,
  fetchConceptDrillDown,
  refreshMasteryHeatmap,
  masteryColorClass,
  masteryLabel,
  type MasteryHeatmapResult,
  type DrillDownStudent,
  type ConceptMeta,
} from '../../lib/mastery-heatmap-api'

function pct(n: number): string {
  return `${Math.round(n * 100)}%`
}

function studentName(displayName: string | null, fallback = 'Unknown student'): string {
  return displayName && displayName.trim() !== '' ? displayName : fallback
}

type DrillDownPanelProps = {
  concept: ConceptMeta
  students: DrillDownStudent[]
  onClose: () => void
}

function DrillDownPanel({ concept, students, onClose }: DrillDownPanelProps) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={`Students for concept: ${concept.name}`}
      className="fixed inset-y-0 end-0 z-50 flex w-full max-w-md flex-col border-s border-border-default bg-surface-raised shadow-xl dark:border-border-default dark:bg-surface-base"
    >
      <div className="flex items-center justify-between border-b border-border-default px-4 py-3 dark:border-border-default">
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-fg-muted">
            Concept drill-down
          </p>
          <h2 className="text-base font-semibold text-fg-default">
            {concept.name}
          </h2>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close drill-down panel"
          className="rounded-lg p-1 text-fg-muted hover:bg-surface-sunken hover:text-fg-muted dark:text-fg-muted dark:hover:bg-surface-overlay dark:hover:text-fg-default"
        >
          ✕
        </button>
      </div>
      <div className="flex-1 overflow-y-auto px-4 py-3">
        {students.length === 0 ? (
          <p className="text-sm text-fg-muted">No students enrolled.</p>
        ) : (
          <ul className="space-y-2">
            {students.map((s) => (
              <li
                key={s.enrollmentId}
                className="flex items-center justify-between gap-4 rounded-xl border border-border-subtle bg-surface-base px-3 py-2 dark:border-border-subtle dark:bg-surface-raised"
              >
                <span className="text-sm text-fg-default">
                  {studentName(s.displayName)}
                </span>
                <span
                  className={`shrink-0 rounded-full px-2 py-0.5 text-xs font-semibold text-white ${masteryColorClass(s.assessed, s.masteryScore)}`}
                  aria-label={masteryLabel(s.assessed, s.masteryScore)}
                >
                  {s.assessed && s.masteryScore !== null ? pct(s.masteryScore) : '—'}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

export default function CourseMasteryHeatmap() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const [result, setResult] = useState<MasteryHeatmapResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const [drillDown, setDrillDown] = useState<{
    concept: ConceptMeta
    students: DrillDownStudent[]
  } | null>(null)
  const [drillLoading, setDrillLoading] = useState(false)

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    setError(null)
    try {
      const data = await fetchMasteryHeatmap(courseCode)
      setResult(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load heatmap.')
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    void load()
  }, [load])

  async function handleRefresh() {
    if (!courseCode) return
    setRefreshing(true)
    try {
      await refreshMasteryHeatmap(courseCode)
      await load()
    } catch {
      // Refresh is best-effort; surface in next load if broken.
    } finally {
      setRefreshing(false)
    }
  }

  async function handleCellClick(concept: ConceptMeta) {
    if (!courseCode) return
    setDrillLoading(true)
    try {
      const { students } = await fetchConceptDrillDown(courseCode, concept.id)
      setDrillDown({ concept, students })
    } catch {
      // Non-critical; ignore drill-down errors.
    } finally {
      setDrillLoading(false)
    }
  }

  const hasData = Boolean(result && result.concepts?.length && result.rows?.length)

  return (
    <LmsPage
      title="Mastery Heatmap"
      description="Skill and concept mastery per student — sourced from adaptive quiz data."
      actions={
        <button
          type="button"
          onClick={() => void handleRefresh()}
          disabled={refreshing}
          aria-label="Refresh heatmap data"
          className="inline-flex items-center gap-2 rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm font-semibold text-fg-muted shadow-sm transition-[background-color,color,border-color] hover:border-indigo-200 hover:bg-indigo-50/60 disabled:opacity-50 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:border-indigo-500/40 dark:hover:bg-indigo-950/40"
        >
          <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} aria-hidden />
          {refreshing ? 'Refreshing…' : 'Refresh'}
        </button>
      }
    >
      {loading && (
        <p className="mt-8 text-sm text-fg-muted" aria-live="polite">
          Loading heatmap…
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

      {!loading && !error && result && !hasData && (
        <div className="mt-8 rounded-2xl border border-border-default bg-surface-base px-6 py-10 text-center dark:border-border-default dark:bg-surface-raised">
          <p className="text-base font-semibold text-fg-default">
            No skill data yet
          </p>
          <p className="mt-1 text-sm text-fg-muted">
            This view requires adaptive quiz data. Once students attempt quizzes mapped to
            concepts, their mastery will appear here.
          </p>
        </div>
      )}

      {!loading && !error && hasData && result && (
        <div className="mt-8 space-y-8">
          {result.refreshedAt && (
            <p className="text-xs text-fg-muted">
              Last updated:{' '}
              {formatDateTime(result.refreshedAt, {
                dateStyle: 'medium',
                timeStyle: 'short',
              })}
            </p>
          )}

          {/* Legend */}
          <div className="flex flex-wrap gap-4 text-xs text-fg-muted">
            {[
              { label: 'Mastered (≥80%)', cls: 'bg-emerald-500' },
              { label: 'Developing (60–79%)', cls: 'bg-lime-400' },
              { label: 'Beginning (40–59%)', cls: 'bg-amber-400' },
              { label: 'At risk (<40%)', cls: 'bg-rose-500' },
              { label: 'Not assessed', cls: 'bg-slate-200 dark:bg-neutral-700' },
            ].map(({ label, cls }) => (
              <span key={label} className="inline-flex items-center gap-1.5">
                <span className={`h-3 w-3 rounded ${cls}`} aria-hidden />
                {label}
              </span>
            ))}
          </div>

          {/* Heatmap table */}
          <div className="overflow-x-auto rounded-2xl border border-border-default shadow-sm dark:border-border-subtle">
            <table className="min-w-full border-collapse text-sm" aria-label="Mastery heatmap">
              <thead>
                <tr className="bg-surface-sunken/80">
                  <th
                    scope="col"
                    className="sticky start-0 z-10 min-w-[160px] border-b border-e border-border-default bg-surface-base px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-fg-muted dark:border-border-default/80 dark:text-fg-muted"
                  >
                    Student
                  </th>
                  {result.concepts.map((c) => (
                    <th
                      key={c.id}
                      scope="col"
                      className="border-b border-border-default px-2 py-3 text-center text-xs font-semibold uppercase tracking-wide text-fg-muted dark:border-border-default dark:text-fg-muted"
                    >
                      <span title={c.name} className="block max-w-[80px] truncate">
                        {c.name}
                      </span>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                {result.rows.map((row) => (
                  <tr
                    key={row.enrollmentId}
                    className="hover:bg-slate-50/60 dark:hover:bg-neutral-800/40"
                  >
                    <th
                      scope="row"
                      className="sticky start-0 z-10 border-e border-border-default bg-surface-raised px-4 py-2 font-medium text-fg-default dark:border-border-default dark:bg-surface-base dark:text-fg-default"
                    >
                      {studentName(row.displayName)}
                    </th>
                    {row.cells.map((cell, ci) => {
                      const concept = result.concepts[ci]!
                      const label = masteryLabel(cell.assessed, cell.masteryScore)
                      const ariaLabel = `${concept.name} — ${studentName(row.displayName)} — ${label}${
                        cell.assessed && cell.masteryScore !== null
                          ? ` (${pct(cell.masteryScore)})`
                          : ''
                      }`
                      return (
                        <td
                          key={cell.conceptId}
                          className="px-1 py-2 text-center"
                        >
                          <button
                            type="button"
                            onClick={() => void handleCellClick(concept)}
                            disabled={drillLoading}
                            aria-label={ariaLabel}
                            title={ariaLabel}
                            className={`inline-flex h-8 w-14 items-center justify-center rounded-md text-xs font-semibold text-white transition-[background-color,color,border-color] hover:ring-2 hover:ring-offset-1 hover:ring-indigo-400 ${masteryColorClass(cell.assessed, cell.masteryScore)}`}
                          >
                            {cell.assessed && cell.masteryScore !== null
                              ? pct(cell.masteryScore)
                              : '—'}
                          </button>
                        </td>
                      )
                    })}
                  </tr>
                ))}

                {/* Summary row */}
                <tr className="bg-surface-sunken/50">
                  <td className="sticky start-0 z-10 border-e border-t border-border-default bg-surface-base px-4 py-2 text-xs font-semibold text-fg-muted dark:border-border-default/50 dark:text-fg-muted">
                    Class avg
                  </td>
                  {result.summary.map((s) => (
                    <td
                      key={s.conceptId}
                      className="border-t border-border-default px-1 py-2 text-center dark:border-border-default"
                    >
                      <span
                        className={`inline-flex h-8 w-14 items-center justify-center rounded-md text-xs font-semibold text-white ${masteryColorClass(true, s.meanMastery)}`}
                        title={`Mean: ${pct(s.meanMastery)} — Mastered: ${pct(s.pctMastered)} — At risk: ${pct(s.pctAtRisk)}`}
                        aria-label={`${s.conceptName} class mean: ${pct(s.meanMastery)}`}
                      >
                        {pct(s.meanMastery)}
                      </span>
                    </td>
                  ))}
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Drill-down slide-over */}
      {drillDown && (
        <>
          <div
            className="fixed inset-0 z-40 bg-black/20 dark:bg-black/40"
            onClick={() => setDrillDown(null)}
            aria-hidden
          />
          <DrillDownPanel
            concept={drillDown.concept}
            students={drillDown.students}
            onClose={() => setDrillDown(null)}
          />
        </>
      )}
    </LmsPage>
  )
}

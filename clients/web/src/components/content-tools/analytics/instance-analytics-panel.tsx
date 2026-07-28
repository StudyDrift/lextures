import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolInstanceAnalytics,
  type ContentToolInstanceAnalytics,
} from '../../../lib/courses-api'
import { FacetDistributionChart } from './facet-distribution-chart'
import { GradeLinkDialog } from './grade-link-dialog'
import { NeedsAttentionList } from './needs-attention-list'
import { AnnotationHeatmap } from './annotation-heatmap'
import { CalibrationMatrix } from './calibration-matrix'
import { DiagramHeatmap } from './diagram-heatmap'
import { SortConfusionView } from './sort-confusion-view'
import { buildConfusionRows } from './build-confusion-rows'

export type InstanceAnalyticsPanelProps = {
  open: boolean
  courseCode: string
  instanceId: string
  onClose: () => void
  onOpenRoster?: () => void
}

export function InstanceAnalyticsPanel({
  open,
  courseCode,
  instanceId,
  onClose,
  onOpenRoster,
}: InstanceAnalyticsPanelProps) {
  const { t } = useTranslation('contentTools')
  const [data, setData] = useState<ContentToolInstanceAnalytics | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [gradeOpen, setGradeOpen] = useState(false)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    setLoading(true)
    setError(null)
    void (async () => {
      try {
        const a = await fetchContentToolInstanceAnalytics(courseCode, instanceId)
        if (!cancelled) setData(a)
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load analytics.')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, courseCode, instanceId])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const engRate =
    data && data.learners > 0 ? Math.round((data.engaged / data.learners) * 100) : 0
  const compRate =
    data && data.learners > 0 ? Math.round((data.completed / data.learners) * 100) : 0

  return (
    <div
      className="fixed inset-0 z-[70] flex items-end justify-center bg-slate-900/50 p-0 sm:items-center sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="instance-analytics-title"
      data-testid="instance-analytics-panel"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="flex max-h-[min(90vh,56rem)] w-full max-w-2xl flex-col overflow-hidden rounded-t-2xl border border-slate-200 bg-white shadow-xl sm:rounded-2xl dark:border-neutral-600 dark:bg-neutral-950">
        <div className="flex shrink-0 items-start justify-between gap-3 border-b border-slate-200 px-4 py-3 sm:px-5 dark:border-neutral-700">
          <div className="min-w-0">
            <h2
              id="instance-analytics-title"
              className="text-base font-semibold text-slate-900 dark:text-neutral-100"
            >
              {t('contentTools.analytics.panelTitle')}
            </h2>
            <p className="mt-0.5 truncate text-sm text-slate-500 dark:text-neutral-400">
              {data?.title || data?.toolId || '—'}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={t('contentTools.instructor.close')}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
          >
            <X className="h-5 w-5" aria-hidden />
          </button>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4 sm:px-5">
          {loading ? (
            <p className="text-sm text-slate-500 dark:text-neutral-400">
              {t('contentTools.runtime.loading')}
            </p>
          ) : null}
          {error ? (
            <p
              className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-300"
              role="alert"
            >
              {error}
            </p>
          ) : null}

          {data && !loading ? (
            <div className="space-y-5">
              {data.learners === 0 ? (
                <p
                  className="rounded-xl border border-dashed border-slate-200 bg-slate-50 px-3 py-3 text-sm text-slate-600 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300"
                  data-testid="analytics-empty"
                >
                  {t('contentTools.analytics.empty')}
                </p>
              ) : null}

              <div className="grid grid-cols-2 gap-3 sm:grid-cols-4" data-testid="analytics-rates">
                <Stat label={t('contentTools.analytics.learners')} value={String(data.learners)} />
                <Stat label={t('contentTools.analytics.engaged')} value={`${engRate}%`} />
                <Stat label={t('contentTools.analytics.completed')} value={`${compRate}%`} />
                <Stat
                  label={t('contentTools.analytics.meanScore')}
                  value={data.score ? `${Math.round(data.score.mean)}%` : '—'}
                />
              </div>

              {data.suppressed ? (
                <p
                  className="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/40 dark:text-amber-100"
                  data-testid="analytics-suppressed"
                >
                  {t('contentTools.analytics.suppressed')}
                </p>
              ) : null}

              {!data.suppressed && data.facets.length > 0 ? (
                <div className="space-y-4">
                  {data.facets.map((f) => (
                    <FacetDistributionChart
                      key={f.key}
                      facetKey={f.key}
                      label={f.label}
                      values={f.values}
                    />
                  ))}
                </div>
              ) : null}

              {!data.suppressed && data.askQuestionsThemes && data.askQuestionsThemes.length > 0 ? (
                <div data-testid="ask-questions-themes">
                  <h3 className="mb-2 text-sm font-semibold text-slate-900 dark:text-neutral-100">
                    {t('contentTools.tools.ask_questions.themesTitle')}
                    {typeof data.totalQuestions === 'number'
                      ? ` (${data.totalQuestions})`
                      : null}
                  </h3>
                  <ul className="space-y-2">
                    {data.askQuestionsThemes.map((theme) => (
                      <li
                        key={theme.theme}
                        className="rounded-xl border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700"
                      >
                        <div className="font-medium text-slate-900 dark:text-neutral-100">
                          {theme.theme}{' '}
                          <span className="text-xs font-normal text-slate-500">×{theme.count}</span>
                        </div>
                        <ul className="mt-1 list-disc ps-4 text-xs text-slate-600 dark:text-neutral-300">
                          {theme.representativeExamples.map((ex) => (
                            <li key={ex}>{ex}</li>
                          ))}
                        </ul>
                      </li>
                    ))}
                  </ul>
                </div>
              ) : null}

              {!data.suppressed &&
              data.calibrationMatrix &&
              data.calibrationMatrix.length > 0 ? (
                <CalibrationMatrix cells={data.calibrationMatrix} />
              ) : null}

              {!data.suppressed && data.toolId === 'highlight_annotate'
                ? (() => {
                    const unitFacet = data.facets.find((f) => f.key === 'unitIndex')
                    if (!unitFacet?.values?.length) return null
                    const units = unitFacet.values
                      .map((v) => ({
                        unitIndex: Number(v.value),
                        label: t('contentTools.tools.highlight_annotate.heatmap.unitFallback', {
                          n: Number(v.value) + 1,
                        }),
                        count: v.count,
                      }))
                      .filter((u) => Number.isFinite(u.unitIndex))
                      .sort((a, b) => a.unitIndex - b.unitIndex)
                    return <AnnotationHeatmap units={units} />
                  })()
                : null}

              {!data.suppressed && data.toolId === 'sort_sequence'
                ? (() => {
                    const placedFacet = data.facets.find((f) => f.key === 'placedIn')
                    if (!placedFacet?.values?.length) return null
                    const itemFacet = data.facets.find((f) => f.key === 'itemId')
                    const labels: Record<string, string> = {}
                    for (const v of itemFacet?.values ?? []) {
                      labels[v.value] = v.value
                    }
                    const rows = buildConfusionRows(placedFacet.values, labels)
                    return <SortConfusionView rows={rows} />
                  })()
                : null}

              {!data.suppressed && data.toolId === 'diagram_hotspot'
                ? (() => {
                    const regionFacet = data.facets.find((f) => f.key === 'regionId')
                    const gridFacet = data.facets.find((f) => f.key === 'gridCell')
                    const assignedFacet = data.facets.find((f) => f.key === 'assignedTo')
                    if (!regionFacet?.values?.length && !gridFacet?.values?.length) return null
                    const cells = (gridFacet?.values ?? []).map((v) => ({
                      cell: v.value,
                      count: v.count,
                    }))
                    const regions = (regionFacet?.values ?? []).map((v) => ({
                      regionId: v.value,
                      label: v.value,
                      count: v.count,
                    }))
                    const swaps = (assignedFacet?.values ?? [])
                      .map((v) => ({ pair: v.value, count: v.count }))
                      .sort((a, b) => b.count - a.count)
                    return <DiagramHeatmap cells={cells} regions={regions} swaps={swaps} />
                  })()
                : null}

              <div>
                <h3 className="mb-2 text-sm font-semibold text-slate-900 dark:text-neutral-100">
                  {t('contentTools.analytics.needsAttention')}
                </h3>
                <NeedsAttentionList items={data.needsAttention} />
              </div>
            </div>
          ) : null}
        </div>

        {data && !loading ? (
          <div className="flex shrink-0 flex-wrap items-center gap-2 border-t border-slate-200 bg-slate-50/80 px-4 py-3 sm:px-5 dark:border-neutral-700 dark:bg-neutral-900/80">
            {onOpenRoster ? (
              <button
                type="button"
                className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
                onClick={onOpenRoster}
              >
                {t('contentTools.instructor.openResponses')}
              </button>
            ) : null}
            <button
              type="button"
              className="rounded-lg bg-indigo-600 px-3.5 py-1.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 dark:bg-indigo-500 dark:hover:bg-indigo-400"
              onClick={() => setGradeOpen(true)}
              data-testid="open-grade-link"
            >
              {data.countsForGrade
                ? t('contentTools.grading.manage')
                : t('contentTools.grading.countForGrade')}
            </button>
            {data.countsForGrade ? (
              <span className="rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-900 dark:bg-amber-950/40 dark:text-amber-200">
                {t('contentTools.grading.countsBadge')}
              </span>
            ) : null}
          </div>
        ) : null}
      </div>

      <GradeLinkDialog
        open={gradeOpen}
        courseCode={courseCode}
        instanceId={instanceId}
        onClose={() => setGradeOpen(false)}
        onSaved={(link) =>
          setData((prev) => (prev ? { ...prev, countsForGrade: link.countsForGrade } : prev))
        }
      />
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-slate-100 bg-slate-50 px-3 py-2.5 dark:border-neutral-700 dark:bg-neutral-900">
      <div className="text-[11px] uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        {label}
      </div>
      <div className="mt-0.5 text-lg font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
        {value}
      </div>
    </div>
  )
}

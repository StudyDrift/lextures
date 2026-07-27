import { useEffect, useState } from 'react'
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

  if (!open) return null

  const engRate =
    data && data.learners > 0 ? Math.round((data.engaged / data.learners) * 100) : 0
  const compRate =
    data && data.learners > 0 ? Math.round((data.completed / data.learners) * 100) : 0

  return (
    <div
      className="fixed inset-0 z-[70] flex items-end justify-center bg-slate-900/40 p-0 sm:items-center sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="instance-analytics-title"
      data-testid="instance-analytics-panel"
    >
      <div className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-t-lg bg-white p-4 shadow-lg sm:rounded-lg dark:bg-neutral-900">
        <div className="mb-3 flex items-start justify-between gap-2">
          <div>
            <h2
              id="instance-analytics-title"
              className="text-lg font-semibold text-slate-900 dark:text-neutral-100"
            >
              {t('contentTools.analytics.panelTitle')}
            </h2>
            <p className="text-sm text-slate-600 dark:text-neutral-300">
              {data?.title || data?.toolId || '—'}
            </p>
          </div>
          <button
            type="button"
            className="text-sm text-slate-600 underline dark:text-neutral-300"
            onClick={onClose}
          >
            {t('contentTools.instructor.close')}
          </button>
        </div>

        {loading ? <p className="text-sm text-slate-500">{t('contentTools.runtime.loading')}</p> : null}
        {error ? (
          <p className="text-sm text-rose-600" role="alert">
            {error}
          </p>
        ) : null}

        {data && !loading ? (
          <div className="space-y-4">
            {data.learners === 0 ? (
              <p className="text-sm text-slate-600" data-testid="analytics-empty">
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
              <p className="rounded bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:bg-amber-950/40 dark:text-amber-100" data-testid="analytics-suppressed">
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
                <h3 className="mb-2 text-sm font-medium text-slate-800 dark:text-neutral-100">
                  {t('contentTools.tools.ask_questions.themesTitle')}
                  {typeof data.totalQuestions === 'number'
                    ? ` (${data.totalQuestions})`
                    : null}
                </h3>
                <ul className="space-y-2">
                  {data.askQuestionsThemes.map((theme) => (
                    <li
                      key={theme.theme}
                      className="rounded border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700"
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

            <div>
              <h3 className="mb-2 text-sm font-medium text-slate-800 dark:text-neutral-100">
                {t('contentTools.analytics.needsAttention')}
              </h3>
              <NeedsAttentionList items={data.needsAttention} />
            </div>

            <div className="flex flex-wrap gap-2 border-t border-slate-200 pt-3 dark:border-neutral-700">
              {onOpenRoster ? (
                <button
                  type="button"
                  className="rounded bg-slate-100 px-3 py-1.5 text-sm dark:bg-neutral-800"
                  onClick={onOpenRoster}
                >
                  {t('contentTools.instructor.openResponses')}
                </button>
              ) : null}
              <button
                type="button"
                className="rounded bg-sky-700 px-3 py-1.5 text-sm text-white"
                onClick={() => setGradeOpen(true)}
                data-testid="open-grade-link"
              >
                {data.countsForGrade
                  ? t('contentTools.grading.manage')
                  : t('contentTools.grading.countForGrade')}
              </button>
              {data.countsForGrade ? (
                <span className="self-center text-xs font-medium text-amber-800 dark:text-amber-200">
                  {t('contentTools.grading.countsBadge')}
                </span>
              ) : null}
            </div>
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
    <div className="rounded border border-slate-200 px-3 py-2 dark:border-neutral-700">
      <div className="text-xs text-slate-500">{label}</div>
      <div className="text-lg font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
        {value}
      </div>
    </div>
  )
}

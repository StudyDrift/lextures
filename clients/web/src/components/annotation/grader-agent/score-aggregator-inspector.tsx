import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { GraderWorkflowGraph, ScoreAggregatorNodeData } from './types'
import { HANDLE_GRADE } from './types'
import { workflowNodeDisplayLabel } from './workflow-node-label'

type ScoreAggregatorInspectorProps = {
  nodeId: string
  graph: GraderWorkflowGraph
  data: Record<string, unknown>
  onChange: (patch: Partial<ScoreAggregatorNodeData>) => void
  onDelete: () => void
  fieldClass: string
}

const MODES = ['sum', 'weightedSum', 'average', 'min', 'max', 'rubricMerge'] as const
const CONFIDENCE_MODES = ['min', 'mean', 'weighted'] as const
const ON_MISSING = ['treatAsZero', 'skipAndRenormalize', 'failItem'] as const

export function ScoreAggregatorInspector({
  nodeId,
  graph,
  data,
  onChange,
  onDelete,
  fieldClass,
}: ScoreAggregatorInspectorProps) {
  const { t } = useTranslation('common')
  const mode = typeof data.mode === 'string' ? data.mode : 'sum'
  const confidence = typeof data.confidence === 'string' ? data.confidence : 'min'
  const onMissing = typeof data.onMissing === 'string' ? data.onMissing : 'treatAsZero'
  const mergeComments = data.mergeComments !== false
  const weights = (data.weights && typeof data.weights === 'object' ? data.weights : {}) as Record<string, number>

  const wiredSources = useMemo(() => {
    return graph.edges
      .filter((edge) => edge.target === nodeId && (edge.targetHandle ?? '') === HANDLE_GRADE)
      .map((edge) => {
        const source = graph.nodes.find((n) => n.id === edge.source)
        if (!source) return null
        const defaultLabel = t(`gradingAgent.canvas.nodes.${source.type}.title`, {
          defaultValue: source.type,
        })
        return {
          id: source.id,
          label: workflowNodeDisplayLabel(source.data, defaultLabel),
        }
      })
      .filter((entry): entry is { id: string; label: string } => Boolean(entry))
  }, [graph.edges, graph.nodes, nodeId, t])

  const weightSum = wiredSources.reduce((sum, source) => sum + (weights[source.id] ?? 1), 0)
  const showWeightWarning = mode === 'weightedSum' && weightSum > 0 && Math.abs(weightSum - 1) > 0.01

  const normalizeWeights = () => {
    if (weightSum <= 0) return
    const next: Record<string, number> = {}
    for (const source of wiredSources) {
      next[source.id] = Number(((weights[source.id] ?? 1) / weightSum).toFixed(4))
    }
    onChange({ weights: next })
  }

  return (
    <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
      <p>{t('gradingAgent.canvas.inspector.aggregatorHelp')}</p>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.aggregatorMode')}
        </span>
        <select
          value={mode}
          onChange={(e) => onChange({ mode: e.target.value as ScoreAggregatorNodeData['mode'] })}
          className={fieldClass}
        >
          {MODES.map((value) => (
            <option key={value} value={value}>
              {t(`gradingAgent.canvas.inspector.aggregatorMode.${value}`)}
            </option>
          ))}
        </select>
      </label>
      {mode === 'weightedSum' ? (
        <div className="space-y-2">
          <p className="font-medium text-slate-800 dark:text-neutral-100">
            {t('gradingAgent.canvas.inspector.aggregatorWeights')}
          </p>
          {wiredSources.length === 0 ? (
            <p className="text-xs text-slate-500 dark:text-neutral-400">
              {t('gradingAgent.canvas.inspector.aggregatorNoInputs')}
            </p>
          ) : (
            <div className="-mx-1 overflow-x-auto px-1">
              <table className="w-full min-w-[240px] text-left text-sm">
                <thead>
                  <tr className="text-xs uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                    <th className="pb-1 pe-2 font-semibold">{t('gradingAgent.canvas.inspector.aggregatorSource')}</th>
                    <th className="pb-1 font-semibold">{t('gradingAgent.canvas.inspector.aggregatorWeight')}</th>
                  </tr>
                </thead>
                <tbody>
                  {wiredSources.map((source) => (
                    <tr key={source.id}>
                      <td className="py-1 pe-2 align-middle">{source.label}</td>
                      <td className="py-1 align-middle">
                        <input
                          type="number"
                          min={0}
                          step={0.05}
                          value={weights[source.id] ?? 1}
                          onChange={(e) =>
                            onChange({
                              weights: { ...weights, [source.id]: Number(e.target.value) },
                            })
                          }
                          className={fieldClass}
                          aria-label={t('gradingAgent.canvas.inspector.aggregatorWeightFor', { source: source.label })}
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {showWeightWarning ? (
            <div className="flex flex-wrap items-center gap-2 text-xs text-amber-700 dark:text-amber-300">
              <span>{t('gradingAgent.canvas.inspector.aggregatorWeightWarning', { sum: weightSum.toFixed(2) })}</span>
              <button
                type="button"
                onClick={normalizeWeights}
                className="font-medium underline hover:no-underline"
              >
                {t('gradingAgent.canvas.inspector.aggregatorNormalizeWeights')}
              </button>
            </div>
          ) : null}
        </div>
      ) : null}
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.aggregatorConfidence')}
        </span>
        <select
          value={confidence}
          onChange={(e) => onChange({ confidence: e.target.value as ScoreAggregatorNodeData['confidence'] })}
          className={fieldClass}
        >
          {CONFIDENCE_MODES.map((value) => (
            <option key={value} value={value}>
              {t(`gradingAgent.canvas.inspector.aggregatorConfidence.${value}`)}
            </option>
          ))}
        </select>
      </label>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.aggregatorOnMissing')}
        </span>
        <select
          value={onMissing}
          onChange={(e) => onChange({ onMissing: e.target.value as ScoreAggregatorNodeData['onMissing'] })}
          className={fieldClass}
        >
          {ON_MISSING.map((value) => (
            <option key={value} value={value}>
              {t(`gradingAgent.canvas.inspector.aggregatorOnMissing.${value}`)}
            </option>
          ))}
        </select>
      </label>
      <label className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={mergeComments}
          onChange={(e) => onChange({ mergeComments: e.target.checked })}
          className="rounded border-slate-300 dark:border-neutral-600"
        />
        <span>{t('gradingAgent.canvas.inspector.aggregatorMergeComments')}</span>
      </label>
      <button
        type="button"
        onClick={onDelete}
        className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
      >
        {t('gradingAgent.canvas.inspector.deleteNode')}
      </button>
    </div>
  )
}
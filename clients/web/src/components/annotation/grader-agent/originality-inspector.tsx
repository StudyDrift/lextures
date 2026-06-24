import { useTranslation } from 'react-i18next'
import type { OriginalityNodeData } from './types'

type OriginalityInspectorProps = {
  data: Record<string, unknown>
  aiLikelihoodAllowed: boolean
  onChange: (patch: Partial<OriginalityNodeData>) => void
  onDelete: () => void
  fieldClass: string
}

const METRICS = ['similarity', 'aiLikelihood'] as const

export function OriginalityInspector({
  data,
  aiLikelihoodAllowed,
  onChange,
  onDelete,
  fieldClass,
}: OriginalityInspectorProps) {
  const { t } = useTranslation('common')
  const metric = typeof data.metric === 'string' ? data.metric : 'similarity'
  const flagThreshold =
    typeof data.flagThreshold === 'number' && Number.isFinite(data.flagThreshold)
      ? data.flagThreshold
      : 0.4

  return (
    <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
      <p>{t('gradingAgent.canvas.inspector.originalityHelp')}</p>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.originalityMetric')}
        </span>
        <select
          value={metric}
          onChange={(e) => onChange({ metric: e.target.value as OriginalityNodeData['metric'] })}
          className={fieldClass}
        >
          {METRICS.map((value) => (
            <option key={value} value={value} disabled={value === 'aiLikelihood' && !aiLikelihoodAllowed}>
              {t(`gradingAgent.canvas.inspector.originalityMetric.${value}`)}
            </option>
          ))}
        </select>
        {!aiLikelihoodAllowed ? (
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.canvas.inspector.originalityAiLikelihoodDisabled')}
          </p>
        ) : (
          <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
            {t('gradingAgent.canvas.inspector.originalityAiLikelihoodWarning')}
          </p>
        )}
      </label>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.originalityFlagThreshold')}
        </span>
        <input
          type="range"
          min={0}
          max={1}
          step={0.05}
          value={flagThreshold}
          onChange={(e) => onChange({ flagThreshold: Number(e.target.value) })}
          className="w-full"
        />
        <span className="text-xs text-slate-500 dark:text-neutral-400">
          {Math.round(flagThreshold * 100)}%
        </span>
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
import { useTranslation } from 'react-i18next'
import type { HumanReviewGateNodeData } from './types'

type HumanReviewGateInspectorProps = {
  data: Record<string, unknown>
  onChange: (patch: Partial<HumanReviewGateNodeData>) => void
  onDelete: () => void
  fieldClass: string
}

const MODES = ['always', 'belowConfidence', 'onFlag'] as const
const QUEUES = ['default', 'integrity', 'general'] as const

export function HumanReviewGateInspector({
  data,
  onChange,
  onDelete,
  fieldClass,
}: HumanReviewGateInspectorProps) {
  const { t } = useTranslation('common')
  const mode = typeof data.mode === 'string' ? data.mode : 'belowConfidence'
  const queue = typeof data.queue === 'string' ? data.queue : 'default'
  const confidenceFloor =
    typeof data.confidenceFloor === 'number' && Number.isFinite(data.confidenceFloor)
      ? data.confidenceFloor
      : 0.7

  return (
    <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
      <p>{t('gradingAgent.canvas.inspector.reviewGateHelp')}</p>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.reviewGateMode')}
        </span>
        <select
          value={mode}
          onChange={(e) => onChange({ mode: e.target.value as HumanReviewGateNodeData['mode'] })}
          className={fieldClass}
        >
          {MODES.map((value) => (
            <option key={value} value={value}>
              {t(`gradingAgent.canvas.inspector.reviewGateMode.${value}`)}
            </option>
          ))}
        </select>
      </label>
      {mode === 'belowConfidence' ? (
        <label className="block">
          <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
            {t('gradingAgent.canvas.inspector.reviewGateFloor')}
          </span>
          <input
            type="number"
            min={0}
            max={1}
            step={0.05}
            value={confidenceFloor}
            onChange={(e) => onChange({ confidenceFloor: Number(e.target.value) })}
            className={fieldClass}
          />
        </label>
      ) : null}
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.reviewGateQueue')}
        </span>
        <select value={queue} onChange={(e) => onChange({ queue: e.target.value })} className={fieldClass}>
          {QUEUES.map((value) => (
            <option key={value} value={value}>
              {t(`gradingAgent.canvas.inspector.flagForReviewQueue.${value}`)}
            </option>
          ))}
        </select>
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
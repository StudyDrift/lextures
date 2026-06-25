import { useTranslation } from 'react-i18next'
import type { SetScoreNodeData } from './types'
import { workflowNodeDisplayLabel } from './workflow-node-label'

type SetScoreInspectorProps = {
  data: Record<string, unknown>
  onChange: (patch: Partial<SetScoreNodeData>) => void
  onDelete: () => void
  fieldClass: string
}

export function SetScoreInspector({ data, onChange, onDelete, fieldClass }: SetScoreInspectorProps) {
  const { t } = useTranslation('common')
  const score = typeof data.score === 'number' ? data.score : 0
  const comment = typeof data.comment === 'string' ? data.comment : ''

  return (
    <div className="space-y-3">
      <p className="text-sm text-slate-700 dark:text-neutral-200">
        {t('gradingAgent.canvas.inspector.setScoreHelp')}
      </p>
      <label className="block text-sm text-slate-700 dark:text-neutral-200">
        <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.setScoreValue')}</span>
        <input
          type="number"
          min={0}
          step="any"
          value={score}
          onChange={(e) => {
            const parsed = parseFloat(e.target.value)
            onChange({ score: isNaN(parsed) ? 0 : parsed })
          }}
          className={fieldClass}
        />
        <p className="mt-1.5 text-xs text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.canvas.inspector.setScoreValueHelp')}
        </p>
      </label>
      <label className="block text-sm text-slate-700 dark:text-neutral-200">
        <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.setScoreComment')}</span>
        <input
          type="text"
          value={comment}
          onChange={(e) => onChange({ comment: e.target.value })}
          placeholder={t('gradingAgent.canvas.inspector.setScoreCommentPlaceholder')}
          className={fieldClass}
        />
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

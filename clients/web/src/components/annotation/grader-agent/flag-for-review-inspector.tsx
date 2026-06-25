import { useTranslation } from 'react-i18next'
import type { FlagForReviewNodeData, GraderWorkflowGraph } from './types'
import { WorkflowPromptEditor } from './workflow-prompt-editor'
import type { WorkflowNodeDefaultLabels } from './workflow-prompt-variable'

type FlagForReviewInspectorProps = {
  nodeId: string
  data: Record<string, unknown>
  graph: GraderWorkflowGraph
  defaults: WorkflowNodeDefaultLabels
  onChange: (patch: Partial<FlagForReviewNodeData>) => void
  onDelete: () => void
  fieldClass: string
}

const QUEUES = ['default', 'integrity', 'general'] as const
const PRIORITIES = ['low', 'normal', 'high'] as const

export function FlagForReviewInspector({
  nodeId,
  data,
  graph,
  defaults,
  onChange,
  onDelete,
  fieldClass,
}: FlagForReviewInspectorProps) {
  const { t } = useTranslation('common')
  const queue = typeof data.queue === 'string' ? data.queue : 'default'
  const priority = typeof data.priority === 'string' ? data.priority : 'normal'
  const reasonTemplate = typeof data.reasonTemplate === 'string' ? data.reasonTemplate : ''

  return (
    <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
      <p>{t('gradingAgent.canvas.inspector.flagForReviewHelp')}</p>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.flagForReviewQueue')}
        </span>
        <select value={queue} onChange={(e) => onChange({ queue: e.target.value })} className={fieldClass}>
          {QUEUES.map((value) => (
            <option key={value} value={value}>
              {t(`gradingAgent.canvas.inspector.flagForReviewQueue.${value}`)}
            </option>
          ))}
        </select>
      </label>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.flagForReviewPriority')}
        </span>
        <select
          value={priority}
          onChange={(e) => onChange({ priority: e.target.value as FlagForReviewNodeData['priority'] })}
          className={fieldClass}
        >
          {PRIORITIES.map((value) => (
            <option key={value} value={value}>
              {t(`gradingAgent.canvas.inspector.flagForReviewPriority.${value}`)}
            </option>
          ))}
        </select>
      </label>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.flagForReviewReasonTemplate')}
        </span>
        <WorkflowPromptEditor
          value={reasonTemplate}
          onChange={(value) => onChange({ reasonTemplate: value })}
          graph={graph}
          promptNodeId={nodeId}
          defaults={defaults}
          className={fieldClass}
          placeholder={t('gradingAgent.canvas.inspector.flagForReviewReasonPlaceholder')}
          expandTitle={t('gradingAgent.canvas.inspector.flagForReviewReasonTemplate')}
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
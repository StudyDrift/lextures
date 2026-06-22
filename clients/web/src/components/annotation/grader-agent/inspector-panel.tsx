import { useTranslation } from 'react-i18next'
import type { GraderAgentWorkflowState } from './use-grader-agent-workflow'

type InspectorPanelProps = {
  workflow: GraderAgentWorkflowState
  accepted: boolean
}

export function InspectorPanel({ workflow, accepted }: InspectorPanelProps) {
  const { t } = useTranslation('common')
  const { graph, selectedNodeId, updateGraderNode, updateContextNode, removeNode } = workflow
  if (!graph || !selectedNodeId) {
    return (
      <p className="text-sm text-slate-500 dark:text-neutral-400">{t('gradingAgent.canvas.inspector.empty')}</p>
    )
  }
  const node = graph.nodes.find((n) => n.id === selectedNodeId)
  if (!node) return null

  if (node.type === 'output') {
    return (
      <div className="space-y-2 text-sm text-slate-700 dark:text-neutral-200">
        <p className="font-medium">{t('gradingAgent.canvas.nodes.output.title')}</p>
        <p>{t('gradingAgent.canvas.inspector.outputHelp')}</p>
      </div>
    )
  }

  if (node.type === 'grader') {
    return (
      <div className="space-y-3">
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.prompt.label')}</span>
          <textarea
            value={typeof node.data.prompt === 'string' ? node.data.prompt : ''}
            onChange={(e) => updateGraderNode(node.id, { prompt: e.target.value })}
            rows={6}
            disabled={accepted}
            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            placeholder={t('gradingAgent.prompt.placeholder')}
          />
        </label>
        {!accepted ? (
          <button
            type="button"
            onClick={() => removeNode(node.id)}
            className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
          >
            {t('gradingAgent.canvas.inspector.deleteNode')}
          </button>
        ) : null}
      </div>
    )
  }

  if (node.type === 'assignmentContext') {
    return (
      <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={Boolean(node.data.includeContent)}
            onChange={(e) =>
              updateContextNode(node.id, {
                includeContent: e.target.checked,
                includeRubric: e.target.checked,
              })
            }
            disabled={accepted}
          />
          {t('gradingAgent.includeContentRubric')}
        </label>
        {!accepted ? (
          <button
            type="button"
            onClick={() => removeNode(node.id)}
            className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
          >
            {t('gradingAgent.canvas.inspector.deleteNode')}
          </button>
        ) : null}
      </div>
    )
  }

  return (
    <p className="text-sm text-slate-500 dark:text-neutral-400">
      {t('gradingAgent.canvas.nodes.submission.title')}
    </p>
  )
}

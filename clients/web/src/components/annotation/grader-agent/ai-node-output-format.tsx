import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { RubricDefinition } from '../../../lib/courses-api'
import type { GraderWorkflowGraph } from './types'
import { aiOutputFormatForNode, buildAiSystemPrompt } from './ai-output-system-prompt'

type AiNodeOutputFormatProps = {
  graph: GraderWorkflowGraph
  nodeId: string
  rubric: RubricDefinition | null | undefined
  maxPoints: number | null | undefined
}

export function AiNodeOutputFormat({ graph, nodeId, rubric, maxPoints }: AiNodeOutputFormatProps) {
  const { t } = useTranslation('common')
  const format = aiOutputFormatForNode(graph, nodeId)
  const systemPrompt = useMemo(
    () => buildAiSystemPrompt(format, rubric, maxPoints ?? null),
    [format, rubric, maxPoints],
  )

  return (
    <div className="space-y-2">
      <div>
        <p className="text-sm font-medium text-fg-default">
          {t('gradingAgent.canvas.inspector.aiOutputFormat.title')}
        </p>
        <p className="mt-1 text-xs text-fg-muted">
          {format === 'rubric'
            ? t('gradingAgent.canvas.inspector.aiOutputFormat.rubricHelp')
            : t('gradingAgent.canvas.inspector.aiOutputFormat.scoreHelp')}
        </p>
      </div>
      <pre
        aria-readonly="true"
        className="max-h-56 overflow-auto whitespace-pre-wrap rounded-lg border border-border-default bg-surface-base px-3 py-2 font-mono text-xs leading-relaxed text-fg-muted dark:border-border-default dark:bg-surface-base dark:text-fg-muted"
      >
        {systemPrompt}
      </pre>
      <p className="text-xs text-fg-muted">
        {t('gradingAgent.canvas.inspector.aiOutputFormat.locked')}
      </p>
    </div>
  )
}
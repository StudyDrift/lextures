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
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.aiOutputFormat.title')}
        </p>
        <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
          {format === 'rubric'
            ? t('gradingAgent.canvas.inspector.aiOutputFormat.rubricHelp')
            : t('gradingAgent.canvas.inspector.aiOutputFormat.scoreHelp')}
        </p>
      </div>
      <pre
        aria-readonly="true"
        className="max-h-56 overflow-auto whitespace-pre-wrap rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-xs leading-relaxed text-slate-700 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-300"
      >
        {systemPrompt}
      </pre>
      <p className="text-xs text-slate-500 dark:text-neutral-400">
        {t('gradingAgent.canvas.inspector.aiOutputFormat.locked')}
      </p>
    </div>
  )
}
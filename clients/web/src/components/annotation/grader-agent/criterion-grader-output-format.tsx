import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { RubricDefinition } from '../../../lib/courses-api'
import { buildCriterionSystemPrompt } from './ai-output-system-prompt'
import { criterionGraderRubric } from './criterion-grader-rubric'
import type { GraderWorkflowGraph } from './types'

type CriterionGraderOutputFormatProps = {
  graph: GraderWorkflowGraph
  nodeId: string
  criterionId?: string
  rubric: RubricDefinition | null | undefined
  assignmentItemId: string
}

export function CriterionGraderOutputFormat({
  graph,
  nodeId,
  criterionId,
  rubric,
  assignmentItemId,
}: CriterionGraderOutputFormatProps) {
  const { t } = useTranslation('common')
  const systemPrompt = useMemo(() => {
    const resolvedRubric = criterionGraderRubric(graph, nodeId, rubric, assignmentItemId)
    const criterion =
      resolvedRubric?.criteria?.find((entry) => entry.id === criterionId) ?? null
    return buildCriterionSystemPrompt(criterion)
  }, [assignmentItemId, criterionId, graph, nodeId, rubric])

  return (
    <div className="space-y-2 rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm dark:border-neutral-700 dark:bg-neutral-900/60">
      <p className="font-medium text-slate-800 dark:text-neutral-100">
        {t('gradingAgent.canvas.inspector.criterionOutputFormat.title')}
      </p>
      <p className="text-xs text-slate-600 dark:text-neutral-400">
        {t('gradingAgent.canvas.inspector.criterionOutputFormat.help')}
      </p>
      <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-md border border-slate-200 bg-white p-2 text-xs text-slate-700 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-200">
        {systemPrompt}
      </pre>
      <p className="text-xs text-slate-500 dark:text-neutral-500">
        {t('gradingAgent.canvas.inspector.aiOutputFormat.locked')}
      </p>
    </div>
  )
}
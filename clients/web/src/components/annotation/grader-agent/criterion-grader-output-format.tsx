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
    <div className="space-y-2 rounded-lg border border-border-default bg-surface-base p-3 text-sm dark:border-border-default/60">
      <p className="font-medium text-fg-default">
        {t('gradingAgent.canvas.inspector.criterionOutputFormat.title')}
      </p>
      <p className="text-xs text-fg-muted">
        {t('gradingAgent.canvas.inspector.criterionOutputFormat.help')}
      </p>
      <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-md border border-border-default bg-surface-raised p-2 text-xs text-fg-muted dark:border-border-default dark:bg-surface-base dark:text-fg-default">
        {systemPrompt}
      </pre>
      <p className="text-xs text-fg-subtle">
        {t('gradingAgent.canvas.inspector.aiOutputFormat.locked')}
      </p>
    </div>
  )
}
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useCourseAssignments } from '../../../hooks/use-course-assignments'
import { useTextModels } from '../../../hooks/use-text-models'
import { activityAssignmentItemId } from './activity-node-data'
import { AssignmentPicker } from './assignment-picker'
import type { GraderAgentWorkflowState } from './use-grader-agent-workflow'
import { isActivityNodeType, isAiNodeType, isStudentSubmissionNodeType } from './types'
import { AiNodeCompiledPrompt } from './ai-node-compiled-prompt'
import { AiNodeOutputFormat } from './ai-node-output-format'
import type { RubricDefinition } from '../../../lib/courses-api'
import { WorkflowPromptEditor } from './workflow-prompt-editor'
import { workflowNodeDisplayLabel } from './workflow-node-label'
import { workflowHasAttachedRubric } from './workflow-grade-slot'
import type { WorkflowNodeDefaultLabels } from './workflow-prompt-variable'

type InspectorPanelProps = {
  workflow: GraderAgentWorkflowState
  accepted: boolean
  courseCode: string
  itemId: string
  assignmentTitle?: string
  rubric?: RubricDefinition | null
  maxPoints?: number | null
}

const fieldClass =
  'w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100'

export function InspectorPanel({
  workflow,
  accepted,
  courseCode,
  itemId,
  assignmentTitle,
  rubric,
  maxPoints,
}: InspectorPanelProps) {
  const { t } = useTranslation('common')
  const {
    graph,
    selectedNodeId,
    updateGraderNode,
    updateAiNode,
    updateActivityNode,
    removeNode,
    nodeDryRunDetails,
    nodeExecutionStates,
  } = workflow
  const selectedNode = graph?.nodes.find((n) => n.id === selectedNodeId) ?? null
  const showActivityInspector = Boolean(selectedNode && isActivityNodeType(selectedNode.type))
  const { models } = useTextModels(Boolean(graph && selectedNodeId && selectedNode?.type === 'grader'))
  const { assignments, loading: assignmentsLoading, error: assignmentsError } = useCourseAssignments(
    courseCode,
    showActivityInspector,
  )
  const selectedAssignmentId =
    selectedNode && isActivityNodeType(selectedNode.type)
      ? activityAssignmentItemId(selectedNode.data, itemId)
      : itemId
  const variableDefaults = useMemo(
    (): WorkflowNodeDefaultLabels => ({
      studentSubmission: t('gradingAgent.canvas.nodes.studentSubmission.title'),
      activity: t('gradingAgent.canvas.nodes.activity.title'),
      ai: t('gradingAgent.canvas.nodes.ai.title'),
      grader: t('gradingAgent.canvas.nodes.grader.title'),
      output: t('gradingAgent.canvas.nodes.output.title'),
    }),
    [t],
  )
  const pickerAssignments = useMemo(() => {
    if (!showActivityInspector) return assignments
    if (assignments.some((assignment) => assignment.id === selectedAssignmentId)) {
      return assignments
    }
    const fallbackTitle =
      selectedAssignmentId === itemId && assignmentTitle?.trim()
        ? assignmentTitle.trim()
        : t('gradingAgent.canvas.inspector.activityAssignmentCurrent')
    return [...assignments, { id: selectedAssignmentId, title: fallbackTitle }]
  }, [assignmentTitle, assignments, itemId, selectedAssignmentId, showActivityInspector, t])

  if (!graph || !selectedNodeId) {
    return (
      <p className="text-sm text-slate-500 dark:text-neutral-400">{t('gradingAgent.canvas.inspector.empty')}</p>
    )
  }
  const node = graph.nodes.find((n) => n.id === selectedNodeId)
  if (!node) return null

  const modelId = typeof node.data.modelId === 'string' ? node.data.modelId : ''
  const nodeTitle = (key: string) => workflowNodeDisplayLabel(node.data, t(key))

  if (node.type === 'output') {
    const gradeSlotLabel = workflowHasAttachedRubric(graph)
      ? t('gradingAgent.canvas.slots.gradeRubric')
      : t('gradingAgent.canvas.slots.gradeScore')
    return (
      <div className="space-y-2 text-sm text-slate-700 dark:text-neutral-200">
        <p className="font-medium">{nodeTitle('gradingAgent.canvas.nodes.output.title')}</p>
        <p>{t('gradingAgent.canvas.inspector.outputHelp')}</p>
        <dl className="space-y-2 pt-1">
          <div>
            <dt className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              {gradeSlotLabel}
            </dt>
            <dd>{t('gradingAgent.canvas.inspector.outputGradeSlot')}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              {t('gradingAgent.canvas.slots.comments')}
            </dt>
            <dd>{t('gradingAgent.canvas.inspector.outputCommentsSlot')}</dd>
          </div>
        </dl>
      </div>
    )
  }

  if (node.type === 'grader') {
    return (
      <div className="space-y-3">
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.grader.title')}
        </p>
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.prompt.label')}</span>
          <WorkflowPromptEditor
            value={typeof node.data.prompt === 'string' ? node.data.prompt : ''}
            onChange={(prompt) => updateGraderNode(node.id, { prompt })}
            graph={graph}
            promptNodeId={node.id}
            defaults={variableDefaults}
            disabled={accepted}
            className={fieldClass}
            placeholder={t('gradingAgent.prompt.placeholder')}
          />
        </label>
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.model.label')}</span>
          <select
            value={modelId}
            onChange={(e) => updateGraderNode(node.id, { modelId: e.target.value || null })}
            disabled={accepted}
            className={fieldClass}
          >
            <option value="">{t('gradingAgent.model.default')}</option>
            {models.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
              </option>
            ))}
          </select>
          <p className="mt-1.5 text-xs text-slate-500 dark:text-neutral-400">{t('gradingAgent.model.help')}</p>
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

  if (isActivityNodeType(node.type)) {
    return (
      <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
        <p className="font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.activity.title')}
        </p>
        <p>{t('gradingAgent.canvas.inspector.activityHelp')}</p>
        <label className="block">
          <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
            {t('gradingAgent.canvas.inspector.activityAssignment')}
          </span>
          <AssignmentPicker
            assignments={pickerAssignments}
            value={selectedAssignmentId}
            disabled={accepted}
            loading={assignmentsLoading}
            filterPlaceholder={t('gradingAgent.canvas.inspector.activityAssignmentFilter')}
            emptyLabel={t('gradingAgent.canvas.inspector.activityAssignmentEmpty')}
            noMatchLabel={t('gradingAgent.canvas.inspector.activityAssignmentNoMatch')}
            onChange={(assignmentId) => updateActivityNode(node.id, { assignmentItemId: assignmentId })}
          />
          <p className="mt-1.5 text-xs text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.canvas.inspector.activityAssignmentHelp')}
          </p>
          {assignmentsError ? (
            <p className="mt-1.5 text-xs text-rose-700 dark:text-rose-300">{assignmentsError}</p>
          ) : null}
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

  if (isAiNodeType(node.type)) {
    const dryRunDetail = nodeDryRunDetails[node.id]
    const showCompiledPrompt = nodeExecutionStates[node.id] === 'success' && dryRunDetail
    return (
      <div className="space-y-3">
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.ai.title')}
        </p>
        <p className="text-sm text-slate-700 dark:text-neutral-200">{t('gradingAgent.canvas.inspector.aiHelp')}</p>
        <AiNodeOutputFormat graph={graph} nodeId={node.id} rubric={rubric} maxPoints={maxPoints} />
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.prompt.label')}</span>
          <WorkflowPromptEditor
            value={typeof node.data.prompt === 'string' ? node.data.prompt : ''}
            onChange={(prompt) => updateAiNode(node.id, { prompt })}
            graph={graph}
            promptNodeId={node.id}
            defaults={variableDefaults}
            disabled={accepted}
            className={fieldClass}
            placeholder={t('gradingAgent.prompt.placeholder')}
          />
        </label>
        {showCompiledPrompt ? <AiNodeCompiledPrompt detail={dryRunDetail} /> : null}
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

  if (isStudentSubmissionNodeType(node.type)) {
    return (
      <div className="space-y-2 text-sm text-slate-700 dark:text-neutral-200">
        <p className="font-medium">{nodeTitle('gradingAgent.canvas.nodes.studentSubmission.title')}</p>
        <p>{t('gradingAgent.canvas.inspector.submissionHelp')}</p>
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

  return null
}

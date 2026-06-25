import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useCourseAssignments } from '../../../hooks/use-course-assignments'
import { useTextModels } from '../../../hooks/use-text-models'
import { activityAssignmentItemId } from './activity-node-data'
import { AssignmentPicker } from './assignment-picker'
import type { GraderAgentWorkflowState } from './use-grader-agent-workflow'
import { criterionGraderRubric } from './criterion-grader-rubric'
import { CriterionGraderOutputFormat } from './criterion-grader-output-format'
import {
  isActivityNodeType,
  isAiNodeType,
  isCodeTestRunnerNodeType,
  isConditionalRouterNodeType,
  isCriterionGraderNodeType,
  isFlagForReviewNodeType,
  isHumanReviewGateNodeType,
  isOriginalityNodeType,
  isReferenceNodeType,
  isRubricNodeType,
  isScoreAggregatorNodeType,
  isStudentSubmissionNodeType,
} from './types'
import { CodeTestRunnerInspector } from './code-test-runner-inspector'
import { ConditionalRouterInspector } from './conditional-router-inspector'
import { FlagForReviewInspector } from './flag-for-review-inspector'
import { HumanReviewGateInspector } from './human-review-gate-inspector'
import { OriginalityInspector } from './originality-inspector'
import { ReferenceInspector } from './reference-inspector'
import { RubricInspector } from './rubric-inspector'
import { ScoreAggregatorInspector } from './score-aggregator-inspector'
import { AiNodeCompiledPrompt } from './ai-node-compiled-prompt'
import { AiNodeOutputFormat } from './ai-node-output-format'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import type { GraderAgentConfigApi, ModuleAssignmentSubmissionApi, RubricDefinition } from '../../../lib/courses-api'
import { AgentConfidenceFloorSettings } from './agent-confidence-floor-settings'
import { SubmissionInspectorSection } from './submission-inspector-section'
import { WorkflowPromptEditor } from './workflow-prompt-editor'
import { workflowNodeDisplayLabel } from './workflow-node-label'
import { workflowHasAttachedRubric } from './workflow-grade-slot'
import type { WorkflowNodeDefaultLabels } from './workflow-prompt-variable'

type InspectorPanelProps = {
  workflow: GraderAgentWorkflowState
  config?: GraderAgentConfigApi | null
  onSetConfidenceFloor?: (floor: number | null) => void | Promise<void>
  courseCode: string
  itemId: string
  assignmentTitle?: string
  rubric?: RubricDefinition | null
  maxPoints?: number | null
  selectedSubmission?: ModuleAssignmentSubmissionApi | null
}

const fieldClass =
  'w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100'

export function InspectorPanel({
  workflow,
  config = null,
  onSetConfidenceFloor,
  courseCode,
  itemId,
  assignmentTitle,
  rubric,
  maxPoints,
  selectedSubmission = null,
}: InspectorPanelProps) {
  const { t } = useTranslation('common')
  const { ffPlagiarismChecks } = usePlatformFeatures()
  const {
    graph,
    selectedNodeId,
    updateGraderNode,
    updateAiNode,
    updateCriterionGraderNode,
    updateActivityNode,
    updateCodeTestRunnerNode,
    updateConditionalRouterNode,
    updateFlagForReviewNode,
    updateHumanReviewGateNode,
    updateScoreAggregatorNode,
    updateOriginalityNode,
    updateReferenceNode,
    updateRubricNode,
    setLibraryRubricAvailability,
    removeNode,
    nodeDryRunDetails,
    nodeExecutionStates,
  } = workflow
  const selectedNode = graph?.nodes.find((n) => n.id === selectedNodeId) ?? null
  const showActivityInspector = Boolean(selectedNode && isActivityNodeType(selectedNode.type))
  const usesTextModels = Boolean(
    graph &&
      selectedNodeId &&
      (selectedNode?.type === 'grader' || isCriterionGraderNodeType(selectedNode?.type ?? '')),
  )
  const { models } = useTextModels(usesTextModels)
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
      codeTestRunner: t('gradingAgent.canvas.nodes.codeTests.title'),
      conditionalRouter: t('gradingAgent.canvas.nodes.router.title'),
      grader: t('gradingAgent.canvas.nodes.grader.title'),
      criterionGrader: t('gradingAgent.canvas.nodes.criterionGrader.title'),
      output: t('gradingAgent.canvas.nodes.output.title'),
      flagForReview: t('gradingAgent.canvas.nodes.flagForReview.title'),
      humanReviewGate: t('gradingAgent.canvas.nodes.reviewGate.title'),
      originality: t('gradingAgent.canvas.nodes.originality.title'),
      reference: t('gradingAgent.canvas.nodes.reference.title'),
      rubric: t('gradingAgent.canvas.nodes.rubric.title'),
      scoreAggregator: t('gradingAgent.canvas.nodes.aggregator.title'),
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
      <div className="space-y-4 text-sm text-slate-700 dark:text-neutral-200">
        <p className="text-sm text-slate-500 dark:text-neutral-400">{t('gradingAgent.canvas.inspector.empty')}</p>
        {onSetConfidenceFloor ? (
          <section className="space-y-2 border-t border-slate-200 pt-3 dark:border-neutral-700">
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              {t('gradingAgent.settings.confidenceFloor.title')}
            </p>
            <AgentConfidenceFloorSettings
              confidenceFloor={config?.confidenceFloor}
              disabled={workflow.saving}
              onChange={(floor) => void onSetConfidenceFloor(floor)}
            />
          </section>
        ) : null}
      </div>
    )
  }
  const node = graph.nodes.find((n) => n.id === selectedNodeId)
  if (!node) {
    return (
      <p className="text-sm text-slate-500 dark:text-neutral-400">{t('gradingAgent.canvas.inspector.empty')}</p>
    )
  }

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
            className={fieldClass}
            placeholder={t('gradingAgent.prompt.placeholder')}
          />
        </label>
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.model.label')}</span>
          <select
            value={modelId}
            onChange={(e) => updateGraderNode(node.id, { modelId: e.target.value || null })}
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
        <button
          type="button"
          onClick={() => removeNode(node.id)}
          className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
        >
          {t('gradingAgent.canvas.inspector.deleteNode')}
        </button>
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
        <button
          type="button"
          onClick={() => removeNode(node.id)}
          className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
        >
          {t('gradingAgent.canvas.inspector.deleteNode')}
        </button>
      </div>
    )
  }

  if (isCriterionGraderNodeType(node.type)) {
    const dryRunDetail = nodeDryRunDetails[node.id]
    const showCompiledPrompt = nodeExecutionStates[node.id] === 'success' && dryRunDetail
    const resolvedRubric = criterionGraderRubric(graph, node.id, rubric, itemId)
    const criteria = resolvedRubric?.criteria ?? []
    const criterionId = typeof node.data.criterionId === 'string' ? node.data.criterionId : ''
    const rubricWired = graph.edges.some(
      (edge) => edge.target === node.id && (edge.targetHandle ?? '') === 'rubric',
    )
    return (
      <div className="space-y-3">
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.criterionGrader.title')}
        </p>
        <p className="text-sm text-slate-700 dark:text-neutral-200">
          {t('gradingAgent.canvas.inspector.criterionGraderHelp')}
        </p>
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.criterion')}</span>
          <select
            value={criterionId}
            onChange={(e) => updateCriterionGraderNode(node.id, { criterionId: e.target.value || undefined })}
            className={fieldClass}
            disabled={criteria.length === 0}
            aria-label={t('gradingAgent.canvas.inspector.criterion')}
          >
            <option value="">
              {criteria.length === 0
                ? t('gradingAgent.canvas.inspector.criterionNoRubric')
                : t('gradingAgent.canvas.inspector.criterionPlaceholder')}
            </option>
            {criteria.map((criterion) => (
              <option key={criterion.id} value={criterion.id}>
                {criterion.title}
              </option>
            ))}
          </select>
          {!rubricWired && criteria.length > 0 ? (
            <p className="mt-1.5 text-xs text-slate-500 dark:text-neutral-400">
              {t('gradingAgent.canvas.inspector.criterionUsingAssignmentRubric')}
            </p>
          ) : null}
          {criteria.length === 0 ? (
            <p className="mt-1.5 text-xs text-amber-700 dark:text-amber-300">
              {t('gradingAgent.canvas.inspector.criterionWireRubricHint')}
            </p>
          ) : null}
        </label>
        <CriterionGraderOutputFormat
          graph={graph}
          nodeId={node.id}
          criterionId={criterionId}
          rubric={rubric}
          assignmentItemId={itemId}
        />
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.prompt.label')}</span>
          <WorkflowPromptEditor
            value={typeof node.data.prompt === 'string' ? node.data.prompt : ''}
            onChange={(prompt) => updateCriterionGraderNode(node.id, { prompt })}
            graph={graph}
            promptNodeId={node.id}
            defaults={variableDefaults}
            className={fieldClass}
            placeholder={t('gradingAgent.canvas.nodes.criterionGrader.emptyPrompt')}
          />
        </label>
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.model.label')}</span>
          <select
            value={modelId}
            onChange={(e) => updateCriterionGraderNode(node.id, { modelId: e.target.value || null })}
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
        {showCompiledPrompt ? <AiNodeCompiledPrompt detail={dryRunDetail} /> : null}
        <button
          type="button"
          onClick={() => removeNode(node.id)}
          className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
        >
          {t('gradingAgent.canvas.inspector.deleteNode')}
        </button>
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
            className={fieldClass}
            placeholder={t('gradingAgent.prompt.placeholder')}
          />
        </label>
        {showCompiledPrompt ? <AiNodeCompiledPrompt detail={dryRunDetail} /> : null}
        <button
          type="button"
          onClick={() => removeNode(node.id)}
          className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
        >
          {t('gradingAgent.canvas.inspector.deleteNode')}
        </button>
      </div>
    )
  }

  if (isStudentSubmissionNodeType(node.type)) {
    return (
      <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
        <p className="font-medium">{nodeTitle('gradingAgent.canvas.nodes.studentSubmission.title')}</p>
        <p>{t('gradingAgent.canvas.inspector.submissionHelp')}</p>
        <SubmissionInspectorSection submission={selectedSubmission} />
        <button
          type="button"
          onClick={() => removeNode(node.id)}
          className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
        >
          {t('gradingAgent.canvas.inspector.deleteNode')}
        </button>
      </div>
    )
  }

  if (isCodeTestRunnerNodeType(node.type)) {
    return (
      <CodeTestRunnerInspector
        data={node.data}
        maxPoints={maxPoints}
        title={nodeTitle('gradingAgent.canvas.nodes.codeTests.title')}
        onChange={(patch) => updateCodeTestRunnerNode(node.id, patch)}
        onDelete={() => removeNode(node.id)}
      />
    )
  }

  if (isConditionalRouterNodeType(node.type)) {
    return (
      <ConditionalRouterInspector
        nodeId={node.id}
        graph={graph}
        data={node.data}
        title={nodeTitle('gradingAgent.canvas.nodes.router.title')}
        onChange={(patch) => updateConditionalRouterNode(node.id, patch)}
        onDelete={() => removeNode(node.id)}
      />
    )
  }

  if (isReferenceNodeType(node.type)) {
    return (
      <div className="space-y-2">
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.reference.title')}
        </p>
        <ReferenceInspector
          courseCode={courseCode}
          data={node.data}
          onChange={(patch) => updateReferenceNode(node.id, patch)}
          onDelete={() => removeNode(node.id)}
          fieldClass={fieldClass}
        />
      </div>
    )
  }

  if (isRubricNodeType(node.type)) {
    return (
      <div className="space-y-2">
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.rubric.title')}
        </p>
        <RubricInspector
          courseCode={courseCode}
          assignmentTitle={assignmentTitle}
          assignmentHasRubric={Boolean(rubric?.criteria?.length)}
          maxPoints={maxPoints}
          data={node.data}
          onChange={(patch) => updateRubricNode(node.id, patch)}
          onDelete={() => removeNode(node.id)}
          onLibraryRubricResolved={setLibraryRubricAvailability}
          fieldClass={fieldClass}
        />
      </div>
    )
  }

  if (isOriginalityNodeType(node.type)) {
    return (
      <div className="space-y-2">
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.originality.title')}
        </p>
        <OriginalityInspector
          data={node.data}
          aiLikelihoodAllowed={ffPlagiarismChecks}
          onChange={(patch) => updateOriginalityNode(node.id, patch)}
          onDelete={() => removeNode(node.id)}
          fieldClass={fieldClass}
        />
      </div>
    )
  }

  if (isScoreAggregatorNodeType(node.type)) {
    return (
      <div className="space-y-2">
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.aggregator.title')}
        </p>
        <ScoreAggregatorInspector
          nodeId={node.id}
          graph={graph}
          data={node.data}
          onChange={(patch) => updateScoreAggregatorNode(node.id, patch)}
          onDelete={() => removeNode(node.id)}
          fieldClass={fieldClass}
        />
      </div>
    )
  }

  if (isHumanReviewGateNodeType(node.type)) {
    return (
      <div className="space-y-2">
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.reviewGate.title')}
        </p>
        <HumanReviewGateInspector
          data={node.data}
          onChange={(patch) => updateHumanReviewGateNode(node.id, patch)}
          onDelete={() => removeNode(node.id)}
          fieldClass={fieldClass}
        />
      </div>
    )
  }

  if (isFlagForReviewNodeType(node.type)) {
    return (
      <div className="space-y-2">
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {nodeTitle('gradingAgent.canvas.nodes.flagForReview.title')}
        </p>
        <FlagForReviewInspector
          nodeId={node.id}
          data={node.data}
          graph={graph}
          defaults={variableDefaults}
          onChange={(patch) => updateFlagForReviewNode(node.id, patch)}
          onDelete={() => removeNode(node.id)}
          fieldClass={fieldClass}
        />
      </div>
    )
  }

  return null
}

import { lazy, Suspense, useEffect, useId, useRef } from 'react'
import { GripVertical, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { GraderAgentReviewQueueItem, QuizQuestion, RubricDefinition } from '../../../lib/courses-api'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import { SubmissionStudentPicker } from '../submission-navigator'
import { InspectorPanel } from './inspector-panel'
import { NodePalette } from './node-palette'
import { ActionErrorTooltip } from '../../ui/action-error-tooltip'
import { DryRunDock } from './dry-run-dock'
import { RunAgentPopover } from './run-agent-popover'
import { ReviewInboxPanel } from './review-inbox-panel'
import { RunHistoryPanel } from './run-history-panel'
import { useGraderAgentReviewQueue } from './use-grader-agent-review-queue'
import { SaveWorkflowMenu } from './save-workflow-menu'
import { useGraderAgentSubmissions } from './use-grader-agent-submissions'
import {
  primaryValidationMessage,
  useGraderAgentWorkflow,
  type GraderAgentTemplateMode,
  type GraderAgentWorkflowSeed,
} from './use-grader-agent-workflow'
import type { GradingAgentItemKind } from './types'
import type { QuizQuestionSlot } from './quiz-question-slots'
import { useHorizontalPanelResize } from './use-horizontal-panel-resize'

const INSPECTOR_DEFAULT_WIDTH = 288
const INSPECTOR_MIN_WIDTH = 240
const INSPECTOR_MAX_WIDTH = 480

const CanvasView = lazy(() =>
  import('./canvas-view').then((m) => ({ default: m.CanvasView })),
)

type GraderAgentWorkflowModalProps = {
  open: boolean
  onClose: () => void
  courseCode: string
  itemId: string
  itemKind?: GradingAgentItemKind
  assignmentTitle?: string
  submissionId: string | null
  rubric: RubricDefinition | null
  maxPoints: number | null
  quizQuestionSlots?: QuizQuestionSlot[]
  quizQuestions?: QuizQuestion[]
  seedWorkflow?: GraderAgentWorkflowSeed | null
  templateMode?: GraderAgentTemplateMode | null
  onApplied?: () => void
}

export function GraderAgentWorkflowModal({
  open,
  onClose,
  courseCode,
  itemId,
  itemKind = 'assignment',
  assignmentTitle,
  submissionId,
  rubric,
  maxPoints,
  quizQuestionSlots = [],
  quizQuestions = [],
  seedWorkflow = null,
  templateMode = null,
  onApplied,
}: GraderAgentWorkflowModalProps) {
  const { t } = useTranslation('common')
  const { width: inspectorWidth, resizeHandleProps } = useHorizontalPanelResize({
    defaultWidth: INSPECTOR_DEFAULT_WIDTH,
    minWidth: INSPECTOR_MIN_WIDTH,
    maxWidth: INSPECTOR_MAX_WIDTH,
  })
  const isTemplateMode = templateMode != null
  const { codeExecutionEnabled, graderAgentReviewInboxEnabled, graderAgentSuggestModeEnabled, graderAgentRunFiltersEnabled, graderAgentCostEstimateEnabled } =
    usePlatformFeatures()
  const titleId = useId()
  const statusId = useId()
  const modalRef = useRef<HTMLDivElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  const {
    submissions,
    index: submissionIndex,
    setIndex: setSubmissionIndex,
    selectedSubmission,
    selectedSubmissionId,
    loading: submissionsLoading,
    loadError: submissionsLoadError,
    markSubmissionGraded,
  } = useGraderAgentSubmissions({
    open,
    courseCode,
    itemId,
    initialSubmissionId: submissionId,
    enabled: !isTemplateMode && itemKind === 'assignment',
  })
  const workflow = useGraderAgentWorkflow({
    open,
    courseCode,
    itemId,
    itemKind,
    quizQuestionSlots,
    submissionId: selectedSubmissionId,
    rubric,
    seedWorkflow,
    templateMode,
    onApplied,
    onSubmissionGraded: markSubmissionGraded,
  })

  const {
    config,
    dryRunning,
    dryRunError,
    dryRunLogs,
    dryRunConsoleOpen,
    batchRunning,
    cancellingRun,
    cancelRunEnabled,
    handleCancelRun,
    hadDryRun,
    saving,
    syncingSubmissionIds,
    runnable,
    validationIssues,
    runScope,
    setRunScope,
    runMode,
    setRunMode,
    confirmOverwrite,
    setConfirmOverwrite,
    runFilterState,
    setRunFilterState,
    runCostEstimate,
    runCostEstimateLoading,
    budgetUsd,
    setBudgetUsd,
    runProgress,
    statusMessage,
    handleDryRun,
    handleSave,
    handleSaveAsTemplate,
    handleAccept,
    handleRun,
    handleToggleAutoGrade,
    handleTogglePostPolicy,
    handleSetConfidenceFloor,
    addPaletteNode,
    runResults,
    refreshRunResults,
  } = workflow

  const reviewInbox = useGraderAgentReviewQueue({
    enabled: open && !isTemplateMode && graderAgentReviewInboxEnabled === true,
    courseCode,
    itemId,
  })

  const submissionLabelById = Object.fromEntries(
    submissions
      .filter((submission) => submission.id)
      .map((submission) => [
        submission.id as string,
        submission.submittedByDisplayName ??
          submission.blindLabel ??
          submission.id ??
          'submission',
      ]),
  )

  const liveReviewItems: GraderAgentReviewQueueItem[] = runResults
    .filter((result) => result.id && (result.status === 'suggested' || result.status === 'flagged'))
    .map((result) => ({
      id: result.id as string,
      submissionId: result.submissionId,
      submissionLabel: submissionLabelById[result.submissionId],
      status: result.status,
      suggestedPoints: result.suggestedPoints,
      comment: result.comment,
      confidence: result.confidence,
      flagReason: result.flagReason,
      flagPriority: result.flagPriority,
      heldReason: result.heldReason,
      heldAt: result.heldAt,
      heldQueue: result.heldQueue,
    }))

  const reviewHeld = graderAgentReviewInboxEnabled ? reviewInbox.held : liveReviewItems.filter((i) => i.status === 'suggested')
  const reviewFlagged = graderAgentReviewInboxEnabled ? reviewInbox.flagged : liveReviewItems.filter((i) => i.status === 'flagged')
  const showReviewInbox =
    graderAgentReviewInboxEnabled ||
    reviewHeld.length > 0 ||
    reviewFlagged.length > 0

  const handleReviewUpdated = () => {
    if (graderAgentReviewInboxEnabled) {
      void reviewInbox.refresh()
    } else {
      void refreshRunResults()
    }
    onApplied?.()
  }

  const handleOpenSubmission = (targetSubmissionId: string) => {
    const idx = submissions.findIndex((submission) => submission.id === targetSubmissionId)
    if (idx >= 0) setSubmissionIndex(idx)
  }

  useEffect(() => {
    if (!batchRunning && graderAgentReviewInboxEnabled) {
      void reviewInbox.refresh()
    }
  }, [batchRunning, graderAgentReviewInboxEnabled, reviewInbox.refresh])

  const accepted = config?.status === 'accepted'
  const validationMsg = primaryValidationMessage(validationIssues)
  const actionAlert = dryRunError ?? (!runnable ? validationMsg : null)

  const dryRunDisabled = dryRunning || !runnable || !selectedSubmissionId
  const dryRunTooltip = dryRunDisabled
    ? dryRunning
      ? null
      : !selectedSubmissionId
        ? t('gradingAgent.dryRun.needsSubmission')
        : actionAlert
    : null

  const acceptDisabled = !hadDryRun || saving || !runnable
  const acceptTooltip = acceptDisabled
    ? saving
      ? null
      : !hadDryRun
        ? t('gradingAgent.accept.needsDryRun')
        : actionAlert
    : null

  const runDisabled = saving || !runnable
  const runTooltip = runDisabled && !saving ? actionAlert : null

  useEffect(() => {
    if (!open) return
    closeRef.current?.focus()
  }, [open])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[520] flex flex-col bg-white dark:bg-neutral-950" role="presentation">
      <div
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="flex h-full flex-col"
      >
        <header className="flex shrink-0 flex-wrap items-center gap-3 border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-3">
              <div className="min-w-0">
                <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
                  {isTemplateMode
                    ? t('gradingAgent.settings.create.templateEditorTitle')
                    : t('gradingAgent.canvas.modal.title')}
                </h2>
                {isTemplateMode ? (
                  <p className="truncate text-sm text-slate-500 dark:text-neutral-400">{templateMode.name}</p>
                ) : assignmentTitle ? (
                  <p className="truncate text-sm text-slate-500 dark:text-neutral-400">{assignmentTitle}</p>
                ) : null}
              </div>
              {!isTemplateMode && config ? (
                <span
                  className={`inline-flex shrink-0 rounded-full px-2 py-0.5 text-xs font-semibold ${
                    accepted
                      ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-200'
                      : 'bg-amber-100 text-amber-900 dark:bg-amber-950/40 dark:text-amber-200'
                  }`}
                >
                  {accepted
                    ? t('gradingAgent.settings.status.accepted')
                    : t('gradingAgent.settings.status.draft')}
                </span>
              ) : null}
            </div>
          </div>
          {!isTemplateMode ? (
            <div className="flex min-w-[14rem] max-w-xs items-center gap-2">
              <SubmissionStudentPicker
                submissions={submissions}
                index={submissionIndex}
                disabled={submissionsLoading}
                syncingSubmissionIds={syncingSubmissionIds}
                onIndexChange={setSubmissionIndex}
              />
              {submissions.length > 0 ? (
                <span className="w-10 shrink-0 text-end text-xs tabular-nums text-slate-500 dark:text-neutral-400">
                  {submissionIndex + 1}/{submissions.length}
                </span>
              ) : null}
            </div>
          ) : null}
          {isTemplateMode ? (
            <ActionErrorTooltip message={!runnable ? validationMsg : null}>
              <button
                type="button"
                disabled={saving || !runnable}
                onClick={() => void handleSave()}
                className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {saving ? (
                  <>
                    <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                    <span>{t('gradingAgent.save.saving')}</span>
                  </>
                ) : (
                  t('gradingAgent.save.asTemplate')
                )}
              </button>
            </ActionErrorTooltip>
          ) : !accepted ? (
            <ActionErrorTooltip message={dryRunTooltip}>
              <button
                type="button"
                disabled={dryRunDisabled}
                onClick={() => void handleDryRun()}
                className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {dryRunning ? (
                  <>
                    <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                    <span>{t('gradingAgent.dryRun.running')}</span>
                  </>
                ) : (
                  t('gradingAgent.dryRun')
                )}
              </button>
            </ActionErrorTooltip>
          ) : (
            <RunAgentPopover
              disabled={runDisabled}
              tooltip={runTooltip}
              dryRunDisabled={dryRunDisabled}
              dryRunTooltip={dryRunTooltip}
              dryRunning={dryRunning}
              batchRunning={batchRunning}
              cancelRunEnabled={cancelRunEnabled}
              cancellingRun={cancellingRun}
              onCancelRun={handleCancelRun}
              runScope={runScope}
              setRunScope={setRunScope}
              confirmOverwrite={confirmOverwrite}
              setConfirmOverwrite={setConfirmOverwrite}
              runProgress={runProgress}
              autoGradeNew={Boolean(config?.autoGradeNew)}
              postPolicy={config?.postPolicy ?? 'draft'}
              confidenceFloor={config?.confidenceFloor}
              suggestModeEnabled={graderAgentSuggestModeEnabled === true}
              runMode={runMode}
              setRunMode={setRunMode}
              saving={saving}
              onDryRun={handleDryRun}
              onToggleAutoGrade={handleToggleAutoGrade}
              onTogglePostPolicy={handleTogglePostPolicy}
              onSetConfidenceFloor={handleSetConfidenceFloor}
              onRun={handleRun}
              runFiltersEnabled={graderAgentRunFiltersEnabled === true}
              courseCode={courseCode}
              itemId={itemId}
              currentSubmissionId={selectedSubmissionId}
              submissions={submissions}
              filterState={runFilterState}
              setFilterState={setRunFilterState}
              costEstimateEnabled={graderAgentCostEstimateEnabled === true}
              runCostEstimate={runCostEstimate}
              runCostEstimateLoading={runCostEstimateLoading}
              budgetUsd={budgetUsd}
              setBudgetUsd={setBudgetUsd}
              onRequestDryRunEstimate={handleDryRun}
            />
          )}
          {!isTemplateMode && !accepted ? (
            <ActionErrorTooltip message={acceptTooltip}>
              <button
                type="button"
                disabled={acceptDisabled}
                onClick={() => void handleAccept()}
                className="rounded-lg border border-indigo-300 px-3 py-2 text-sm font-semibold text-indigo-700 dark:border-indigo-800 dark:text-indigo-300 disabled:opacity-50"
              >
                {t('gradingAgent.accept')}
              </button>
            </ActionErrorTooltip>
          ) : null}
          {!isTemplateMode ? (
            <SaveWorkflowMenu
              saving={saving}
              defaultTemplateName={assignmentTitle}
              acceptVisible={!accepted}
              acceptDisabled={acceptDisabled}
              acceptTooltip={acceptTooltip}
              onSave={handleSave}
              onSaveAsTemplate={handleSaveAsTemplate}
              onAccept={() => void handleAccept()}
            />
          ) : null}
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            className="rounded-lg px-2 py-1 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
          >
            {t('gradingAgent.close')}
          </button>
        </header>

        <div id={statusId} role="status" aria-live="polite" className="sr-only">
          {statusMessage}
        </div>

        {!isTemplateMode && dryRunError ? (
          <p className="shrink-0 border-b border-rose-200 bg-rose-50 px-4 py-2 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200">
            {dryRunError}
          </p>
        ) : null}

        {!isTemplateMode && submissionsLoadError ? (
          <p className="shrink-0 border-b border-rose-200 bg-rose-50 px-4 py-2 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200">
            {submissionsLoadError}
          </p>
        ) : null}

        <div className="relative z-0 flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
          <aside className="flex min-h-0 w-full shrink-0 flex-col border-b border-slate-200 px-3 py-3 lg:w-60 lg:border-b-0 lg:border-e lg:px-3.5 dark:border-neutral-700">
            <NodePalette
              codeExecutionEnabled={codeExecutionEnabled}
              itemKind={itemKind}
              onAddNode={addPaletteNode}
            />
          </aside>
          <main className="relative z-0 min-h-0 flex-1 overflow-hidden p-3">
            <div className="h-full min-h-0">
              <Suspense fallback={<div className="h-full motion-safe:animate-pulse rounded-xl bg-slate-100 dark:bg-neutral-800" />}>
                <CanvasView workflow={workflow} />
              </Suspense>
            </div>
          </main>
          <aside
            className="relative flex min-h-0 w-full shrink-0 flex-col border-t border-slate-200 p-3 pl-4 lg:w-[var(--inspector-width)] lg:shrink-0 lg:border-t-0 lg:border-s dark:border-neutral-700"
            style={{ ['--inspector-width' as string]: `${inspectorWidth}px` }}
          >
            <div
              role="separator"
              aria-orientation="vertical"
              aria-label={t('gradingAgent.canvas.inspector.resize')}
              tabIndex={0}
              className="group absolute inset-y-0 left-0 z-10 hidden w-3 -translate-x-1/2 cursor-ew-resize touch-none items-center justify-center rounded-sm border border-slate-200 bg-slate-100 shadow-sm transition-colors hover:border-indigo-300 hover:bg-indigo-50 active:bg-indigo-100 dark:border-neutral-600 dark:bg-neutral-900 dark:hover:border-indigo-700 dark:hover:bg-indigo-950/60 dark:active:bg-indigo-950 lg:flex"
              {...resizeHandleProps}
            >
              <GripVertical
                className="h-4 w-4 text-slate-400 transition-colors group-hover:text-indigo-500 dark:text-neutral-500 dark:group-hover:text-indigo-400"
                aria-hidden
              />
            </div>
            <p className="mb-2 shrink-0 text-xs font-semibold uppercase tracking-wide text-slate-500">
              {t('gradingAgent.canvas.inspector.title')}
            </p>
            <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
              <InspectorPanel
                workflow={workflow}
                config={config}
                onSetConfidenceFloor={handleSetConfidenceFloor}
                courseCode={courseCode}
                itemId={itemId}
                assignmentTitle={assignmentTitle}
                rubric={rubric}
                maxPoints={maxPoints}
                selectedSubmission={selectedSubmission}
                quizQuestionSlots={quizQuestionSlots}
                quizQuestions={quizQuestions}
              />
              {!isTemplateMode && showReviewInbox ? (
                <>
                  <ReviewInboxPanel
                    courseCode={courseCode}
                    itemId={itemId}
                    held={reviewHeld}
                    flagged={reviewFlagged}
                    suggestModeEnabled={graderAgentSuggestModeEnabled === true}
                    loading={graderAgentReviewInboxEnabled ? reviewInbox.loading : false}
                    error={graderAgentReviewInboxEnabled ? reviewInbox.error : null}
                    onUpdated={handleReviewUpdated}
                    onOpenSubmission={handleOpenSubmission}
                  />
                  {graderAgentReviewInboxEnabled ? (
                    <RunHistoryPanel runs={reviewInbox.runs} loading={reviewInbox.loading} />
                  ) : null}
                </>
              ) : null}
            </div>
          </aside>
        </div>

        {!isTemplateMode ? (
          <footer className="relative z-10 shrink-0 border-t border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-950">
            <DryRunDock
              workflow={workflow}
              rubric={rubric}
              maxPoints={maxPoints}
              submissionId={selectedSubmissionId}
              consoleOpen={dryRunConsoleOpen}
              logs={dryRunLogs}
              running={dryRunning || batchRunning}
              batchRunning={batchRunning}
              cancelRunEnabled={cancelRunEnabled}
              cancellingRun={cancellingRun}
              onCancelRun={handleCancelRun}
              runProgress={runProgress}
            />
          </footer>
        ) : null}
      </div>
    </div>
  )
}

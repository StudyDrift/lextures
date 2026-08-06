import { useState } from 'react'
import { ChevronUp } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '../../ui/button'
import type { RubricDefinition } from '../../../lib/courses-api'
import { DryRunConsole, dryRunConsoleSummary } from './dry-run-console'
import { PreviewDock, previewDockSummary } from './preview-dock'
import { useVerticalPanelResize } from './use-vertical-panel-resize'
import type { DryRunLogEntry, GraderAgentWorkflowState } from './use-grader-agent-workflow'

const DOCK_MIN_HEIGHT = 96
const DOCK_DEFAULT_HEIGHT = 160
const DOCK_MAX_HEIGHT = 480

type DryRunDockProps = {
  workflow: GraderAgentWorkflowState
  rubric: RubricDefinition | null
  maxPoints: number | null
  submissionId: string | null
  consoleOpen: boolean
  logs: DryRunLogEntry[]
  running: boolean
  batchRunning: boolean
  cancelRunEnabled?: boolean
  cancellingRun?: boolean
  onCancelRun?: () => void | Promise<void>
  runProgress: { completed: number; failed: number; total: number } | null
}

type StatusBarSegmentProps = {
  title: string
  summary: string
  expanded: boolean
  onToggle: () => void
  toggleLabel: string
  tone?: 'console' | 'preview'
}

function StatusBarSegment({
  title,
  summary,
  expanded,
  onToggle,
  toggleLabel,
  tone = 'console',
}: StatusBarSegmentProps) {
  const isConsole = tone === 'console'
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-expanded={expanded}
      aria-label={toggleLabel}
      className={`flex min-w-0 flex-1 items-center gap-2 border-e px-4 py-2.5 text-start last:border-e-0 ${ isConsole ? 'border-slate-800 bg-slate-950 text-slate-100 hover:bg-slate-900 dark:border-border-subtle dark:bg-surface-base dark:hover:bg-surface-raised' : 'border-border-default bg-surface-raised text-fg-default hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay' }`}
    >
      <ChevronUp
        className={`h-4 w-4 shrink-0 transition-transform ${expanded ? '' : 'rotate-180'} ${ isConsole ? 'text-fg-subtle' : 'text-fg-muted' }`}
        aria-hidden
      />
      <span
        className={`shrink-0 text-xs font-semibold uppercase tracking-wide ${ isConsole ? 'text-fg-subtle' : 'text-fg-muted' }`}
      >
        {title}
      </span>
      <span
        className={`min-w-0 truncate text-xs ${ isConsole ? 'text-slate-300' : 'font-semibold tabular-nums text-fg-default' }`}
      >
        {summary}
      </span>
    </button>
  )
}

export function DryRunDock({
  workflow,
  rubric,
  maxPoints,
  submissionId,
  consoleOpen,
  logs,
  running,
  batchRunning,
  cancelRunEnabled = false,
  cancellingRun = false,
  onCancelRun,
  runProgress,
}: DryRunDockProps) {
  const { t } = useTranslation('common')
  const { dryRunResult } = workflow
  const [consoleExpanded, setConsoleExpanded] = useState(true)
  const [previewExpanded, setPreviewExpanded] = useState(true)
  const { height: panelHeight, resizeHandleProps } = useVerticalPanelResize({
    defaultHeight: DOCK_DEFAULT_HEIGHT,
    minHeight: DOCK_MIN_HEIGHT,
    maxHeight: DOCK_MAX_HEIGHT,
  })

  const dockVisible = consoleOpen || Boolean(dryRunResult) || batchRunning
  if (!dockVisible) return null

  const showPreview = Boolean(dryRunResult) && !batchRunning
  const showConsole = consoleOpen || batchRunning
  const showConsolePanel = showConsole && consoleExpanded
  const showPreviewPanel = showPreview && previewExpanded
  const showExpandedPanels = showConsolePanel || showPreviewPanel
  const bothPanelsVisible = showConsolePanel && showPreviewPanel

  const consoleSummary = batchRunning && runProgress
    ? t('gradingAgent.run.progress', {
        completed: runProgress.completed,
        failed: runProgress.failed,
        total: runProgress.total,
      })
    : dryRunConsoleSummary(
        logs,
        running,
        t('gradingAgent.dryRun.running'),
        t('gradingAgent.dryRun.console.empty'),
      )
  const previewSummary = previewDockSummary(dryRunResult, maxPoints)

  return (
    <div className="relative z-10 flex flex-col bg-surface-raised">
      {showExpandedPanels ? (
        <>
          <div
            role="separator"
            aria-orientation="horizontal"
            aria-label={t('gradingAgent.dryRun.resize')}
            tabIndex={0}
            className="flex h-2 shrink-0 cursor-ns-resize touch-none items-center justify-center border-b border-border-default bg-surface-sunken hover:bg-slate-200 dark:border-border-default dark:bg-surface-raised dark:hover:bg-surface-overlay"
            {...resizeHandleProps}
          >
            <span className="h-1 w-10 rounded-full bg-slate-300 dark:bg-neutral-600" aria-hidden />
          </div>
          <div className="px-4 pb-3" style={{ height: panelHeight }}>
            <div
              className={`grid h-full min-h-0 gap-3 ${bothPanelsVisible ? 'lg:grid-cols-2' : 'grid-cols-1'}`}
            >
              {showConsolePanel ? <DryRunConsole logs={logs} /> : null}
              {showPreviewPanel ? (
                <PreviewDock
                  workflow={workflow}
                  rubric={rubric}
                  maxPoints={maxPoints}
                  submissionId={submissionId}
                />
              ) : null}
            </div>
          </div>
        </>
      ) : null}

      <div
        role="toolbar"
        aria-label={t('gradingAgent.dryRun.statusBar.label')}
        className="flex border-t border-border-default"
      >
        {batchRunning && cancelRunEnabled ? (
          <div className="flex shrink-0 items-center border-e border-border-default px-3 py-2 dark:border-border-default">
            <Button
              variant="secondary"
              disabled={cancellingRun}
              onClick={() => void onCancelRun?.()}
            >
              {cancellingRun
                ? t('gradingAgent.run.cancel.cancelling')
                : t('gradingAgent.run.cancel.button')}
            </Button>
          </div>
        ) : null}
        {showConsole ? (
          <StatusBarSegment
            tone="console"
            title={t('gradingAgent.dryRun.console.title')}
            summary={consoleSummary}
            expanded={consoleExpanded}
            onToggle={() => setConsoleExpanded((v) => !v)}
            toggleLabel={
              consoleExpanded
                ? t('gradingAgent.dryRun.console.collapse')
                : t('gradingAgent.dryRun.console.expand')
            }
          />
        ) : null}
        {showPreview ? (
          <StatusBarSegment
            tone="preview"
            title={t('gradingAgent.result.title')}
            summary={previewSummary}
            expanded={previewExpanded}
            onToggle={() => setPreviewExpanded((v) => !v)}
            toggleLabel={
              previewExpanded
                ? t('gradingAgent.result.collapse')
                : t('gradingAgent.result.expand')
            }
          />
        ) : null}
      </div>
    </div>
  )
}
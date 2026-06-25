import { useEffect, useId, useRef, useState } from 'react'
import { Check, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ActionErrorTooltip } from '../../ui/action-error-tooltip'
import { Button } from '../../ui/button'
import type { GraderAgentRunMode, ModuleAssignmentSubmissionApi } from '../../../lib/courses-api'
import { AgentConfidenceFloorSettings } from './agent-confidence-floor-settings'
import {
  RunAgentFilterPicker,
  type RunAgentFilterState,
} from './run-agent-filter-picker'
import type { RunScope } from './use-grader-agent-workflow'

const RUN_SCOPES = ['current', 'ungraded', 'all'] as const satisfies readonly RunScope[]
const RUN_MODES = ['suggest', 'apply'] as const satisfies readonly GraderAgentRunMode[]

const pressScale =
  'motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96]'

type RunProgress = {
  completed: number
  failed: number
  total: number
}

type RunAgentPopoverProps = {
  disabled: boolean
  tooltip: string | null
  dryRunDisabled: boolean
  dryRunTooltip: string | null
  dryRunning: boolean
  batchRunning: boolean
  runScope: RunScope
  setRunScope: (scope: RunScope) => void
  confirmOverwrite: boolean
  setConfirmOverwrite: (value: boolean) => void
  runProgress: RunProgress | null
  autoGradeNew: boolean
  postPolicy: 'draft' | 'auto_post'
  confidenceFloor?: number | null
  suggestModeEnabled: boolean
  runMode: GraderAgentRunMode
  setRunMode: (mode: GraderAgentRunMode) => void
  saving: boolean
  onDryRun: () => void | Promise<void>
  onToggleAutoGrade: (enabled: boolean) => void
  onTogglePostPolicy: (autoPost: boolean) => void
  onSetConfidenceFloor: (floor: number | null) => void | Promise<void>
  onRun: () => void | Promise<void>
  runFiltersEnabled?: boolean
  courseCode?: string
  itemId?: string
  currentSubmissionId?: string | null
  submissions?: ModuleAssignmentSubmissionApi[]
  filterState?: RunAgentFilterState
  setFilterState?: (next: RunAgentFilterState | ((prev: RunAgentFilterState) => RunAgentFilterState)) => void
}

export function RunAgentPopover({
  disabled,
  tooltip,
  dryRunDisabled,
  dryRunTooltip,
  dryRunning,
  batchRunning,
  runScope,
  setRunScope,
  confirmOverwrite,
  setConfirmOverwrite,
  runProgress,
  autoGradeNew,
  postPolicy,
  confidenceFloor,
  suggestModeEnabled,
  runMode,
  setRunMode,
  saving,
  onDryRun,
  onToggleAutoGrade,
  onTogglePostPolicy,
  onSetConfidenceFloor,
  onRun,
  runFiltersEnabled = false,
  courseCode = '',
  itemId = '',
  currentSubmissionId = null,
  submissions = [],
  filterState,
  setFilterState,
}: RunAgentPopoverProps) {
  const { t } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const buttonId = useId()
  const panelId = useId()

  useEffect(() => {
    if (!open) {
      setConfirmOverwrite(false)
      return
    }
    function onPointerDown(e: PointerEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open, setConfirmOverwrite])

  const handleInnerRun = async () => {
    const needsConfirmFirst = runScope === 'all' && !confirmOverwrite
    await onRun()
    if (!needsConfirmFirst) {
      setOpen(false)
    }
  }

  return (
    <div ref={rootRef} className="relative shrink-0">
      <ActionErrorTooltip message={tooltip}>
        <button
          id={buttonId}
          type="button"
          disabled={disabled}
          aria-haspopup="dialog"
          aria-expanded={open}
          aria-controls={open ? panelId : undefined}
          onClick={() => setOpen((prev) => !prev)}
          className={`rounded-xl bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-700 disabled:opacity-50 ${pressScale}`}
        >
          {t('gradingAgent.run.start')}
        </button>
      </ActionErrorTooltip>

      <div
        id={panelId}
        role="dialog"
        aria-labelledby={buttonId}
        hidden={!open}
        className="absolute end-0 top-full z-50 mt-2 w-80 rounded-3xl bg-white p-4 shadow-[0_8px_30px_-4px_rgba(15,23,42,0.14),0_4px_12px_-6px_rgba(15,23,42,0.08)] ring-1 ring-black/5 dark:bg-neutral-900 dark:shadow-[0_8px_30px_-4px_rgba(0,0,0,0.55),0_4px_12px_-6px_rgba(0,0,0,0.35)] dark:ring-white/10"
      >
        <p className="text-sm font-medium text-balance text-slate-900 dark:text-neutral-50">
          {t('gradingAgent.run.title')}
        </p>
        <fieldset className="mt-3">
          <legend className="mb-2 text-xs font-medium text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.run.scopeLabel')}
          </legend>
          <div
            role="radiogroup"
            aria-label={t('gradingAgent.run.scopeLabel')}
            className="overflow-hidden rounded-xl bg-slate-50 ring-1 ring-black/5 dark:bg-neutral-800/60 dark:ring-white/10"
          >
            {RUN_SCOPES.map((scope, index) => {
              const selected = runScope === scope
              return (
                <label
                  key={scope}
                  className={`flex min-h-10 cursor-pointer items-center gap-3 px-3 py-2.5 text-sm transition-colors ${
                    index > 0 ? 'border-t border-slate-200/80 dark:border-neutral-700/80' : ''
                  } ${
                    selected
                      ? 'bg-white text-indigo-900 dark:bg-neutral-900 dark:text-indigo-100'
                      : 'text-slate-700 hover:bg-white/70 dark:text-neutral-300 dark:hover:bg-neutral-900/50'
                  }`}
                >
                  <input
                    type="radio"
                    name="grader-agent-run-scope"
                    value={scope}
                    checked={selected}
                    onChange={() => {
                      setRunScope(scope)
                      setConfirmOverwrite(false)
                    }}
                    className="sr-only"
                  />
                  <span className="min-w-0 flex-1 leading-snug">{t(`gradingAgent.run.scope.${scope}`)}</span>
                  <Check
                    className={`h-4 w-4 shrink-0 text-indigo-600 motion-safe:transition-[opacity,transform,filter] motion-safe:duration-150 dark:text-indigo-400 ${
                      selected ? 'scale-100 opacity-100 blur-0' : 'scale-[0.25] opacity-0 blur-[4px]'
                    }`}
                    aria-hidden
                  />
                </label>
              )
            })}
          </div>
        </fieldset>
        {runFiltersEnabled && filterState && setFilterState ? (
          <RunAgentFilterPicker
            enabled={runFiltersEnabled}
            courseCode={courseCode}
            itemId={itemId}
            runScope={runScope}
            confirmOverwrite={confirmOverwrite}
            filterState={filterState}
            setFilterState={setFilterState}
            submissions={submissions}
            currentSubmissionId={currentSubmissionId}
          />
        ) : null}
        {suggestModeEnabled ? (
          <fieldset className="mt-3">
            <legend className="mb-2 text-xs font-medium text-slate-500 dark:text-neutral-400">
              {t('gradingAgent.run.modeLabel')}
            </legend>
            <div
              role="radiogroup"
              aria-label={t('gradingAgent.run.modeLabel')}
              className="overflow-hidden rounded-xl bg-slate-50 ring-1 ring-black/5 dark:bg-neutral-800/60 dark:ring-white/10"
            >
              {RUN_MODES.map((mode, index) => {
                const selected = runMode === mode
                return (
                  <label
                    key={mode}
                    className={`flex min-h-10 cursor-pointer items-center gap-3 px-3 py-2.5 text-sm transition-colors ${
                      index > 0 ? 'border-t border-slate-200/80 dark:border-neutral-700/80' : ''
                    } ${
                      selected
                        ? 'bg-white text-indigo-900 dark:bg-neutral-900 dark:text-indigo-100'
                        : 'text-slate-700 hover:bg-white/70 dark:text-neutral-300 dark:hover:bg-neutral-900/50'
                    }`}
                  >
                    <input
                      type="radio"
                      name="grader-agent-run-mode"
                      value={mode}
                      checked={selected}
                      onChange={() => setRunMode(mode)}
                      className="sr-only"
                    />
                    <span className="min-w-0 flex-1 leading-snug">{t(`gradingAgent.run.mode.${mode}`)}</span>
                    <Check
                      className={`h-4 w-4 shrink-0 text-indigo-600 motion-safe:transition-[opacity,transform,filter] motion-safe:duration-150 dark:text-indigo-400 ${
                        selected ? 'scale-100 opacity-100 blur-0' : 'scale-[0.25] opacity-0 blur-[4px]'
                      }`}
                      aria-hidden
                    />
                  </label>
                )
              })}
            </div>
            <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
              {runMode === 'suggest'
                ? t('gradingAgent.run.mode.suggestNote')
                : t('gradingAgent.run.mode.applyNote')}
            </p>
          </fieldset>
        ) : null}
        {confirmOverwrite ? (
          <p className="mt-3 text-sm text-amber-800 dark:text-amber-200">
            {t('gradingAgent.run.overwriteWarning')}
          </p>
        ) : null}
        {runProgress ? (
          <p className="mt-3 text-sm tabular-nums text-slate-600 dark:text-neutral-400">
            {t('gradingAgent.run.progress', {
              completed: runProgress.completed,
              failed: runProgress.failed,
              total: runProgress.total,
            })}
          </p>
        ) : null}
        <label className="mt-3 flex min-h-10 cursor-pointer items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
          <input
            type="checkbox"
            className="size-4"
            checked={autoGradeNew}
            onChange={(e) => void onToggleAutoGrade(e.target.checked)}
          />
          {t('gradingAgent.autoGradeNew')}
        </label>
        <label className="mt-1 flex min-h-10 cursor-pointer items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
          <input
            type="checkbox"
            className="size-4"
            checked={postPolicy === 'auto_post'}
            onChange={(e) => void onTogglePostPolicy(e.target.checked)}
          />
          {t('gradingAgent.posting.autoPost')}
        </label>
        <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
          {postPolicy === 'auto_post'
            ? t('gradingAgent.posting.autoPostNote')
            : t('gradingAgent.posting.draftNote')}
        </p>
        <div className="mt-3 border-t border-slate-200/80 pt-3 dark:border-neutral-700/80">
          <p className="mb-2 text-xs font-medium text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.settings.confidenceFloor.title')}
          </p>
          <AgentConfidenceFloorSettings
            compact
            disabled={saving}
            confidenceFloor={confidenceFloor}
            onChange={(floor) => void onSetConfidenceFloor(floor)}
          />
        </div>
        <div className="mt-4 flex justify-end gap-2">
          <ActionErrorTooltip message={dryRunTooltip}>
            <Button variant="secondary" disabled={dryRunDisabled} onClick={() => void onDryRun()}>
              {dryRunning ? (
                <>
                  <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                  <span>{t('gradingAgent.dryRun.running')}</span>
                </>
              ) : (
                t('gradingAgent.dryRun')
              )}
            </Button>
          </ActionErrorTooltip>
          <Button disabled={saving || disabled || batchRunning} onClick={() => void handleInnerRun()}>
            {batchRunning ? (
              <>
                <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                <span>{t('gradingAgent.run.running')}</span>
              </>
            ) : confirmOverwrite ? (
              t('gradingAgent.run.confirm')
            ) : (
              t('gradingAgent.run.execute')
            )}
          </Button>
        </div>
      </div>
    </div>
  )
}

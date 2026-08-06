import { useEffect, useId, useRef, useState } from 'react'
import { Check, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ActionErrorTooltip } from '../../ui/action-error-tooltip'
import { Button } from '../../ui/button'
import type { GraderAgentRunMode, GraderAgentRunCostEstimate, ModuleAssignmentSubmissionApi } from '../../../lib/courses-api'
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
  cancelRunEnabled?: boolean
  cancellingRun?: boolean
  onCancelRun?: () => void | Promise<void>
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
  costEstimateEnabled?: boolean
  runCostEstimate?: GraderAgentRunCostEstimate | null
  runCostEstimateLoading?: boolean
  budgetUsd?: number | null
  setBudgetUsd?: (value: number | null) => void
  onRequestDryRunEstimate?: () => void | Promise<void>
}

function formatUsd(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { style: 'currency', currency: 'USD', maximumFractionDigits: 4 }).format(value)
}

export function RunAgentPopover({
  disabled,
  tooltip,
  dryRunDisabled,
  dryRunTooltip,
  dryRunning,
  batchRunning,
  cancelRunEnabled = false,
  cancellingRun = false,
  onCancelRun,
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
  costEstimateEnabled = false,
  runCostEstimate = null,
  runCostEstimateLoading = false,
  budgetUsd = null,
  setBudgetUsd,
  onRequestDryRunEstimate,
}: RunAgentPopoverProps) {
  const { t, i18n } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const buttonId = useId()
  const panelId = useId()

  const costEstimateId = useId()

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
          aria-describedby={costEstimateEnabled && open ? costEstimateId : undefined}
          onClick={() => setOpen((prev) => !prev)}
          className={`rounded-xl bg-accent-solid px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-accent disabled:opacity-50 ${pressScale}`}
        >
          {t('gradingAgent.run.start')}
        </button>
      </ActionErrorTooltip>

      <div
        id={panelId}
        role="dialog"
        aria-labelledby={buttonId}
        hidden={!open}
        className="absolute end-0 top-full z-50 mt-2 w-80 rounded-3xl bg-surface-raised p-4 shadow-[0_8px_30px_-4px_rgba(15,23,42,0.14),0_4px_12px_-6px_rgba(15,23,42,0.08)] ring-1 ring-black/5 dark:bg-surface-raised dark:shadow-[0_8px_30px_-4px_rgba(0,0,0,0.55),0_4px_12px_-6px_rgba(0,0,0,0.35)] dark:ring-white/10"
      >
        <p className="text-sm font-medium text-balance text-fg-default">
          {t('gradingAgent.run.title')}
        </p>
        <fieldset className="mt-3">
          <legend className="mb-2 text-xs font-medium text-fg-muted">
            {t('gradingAgent.run.scopeLabel')}
          </legend>
          <div
            role="radiogroup"
            aria-label={t('gradingAgent.run.scopeLabel')}
            className="overflow-hidden rounded-xl bg-surface-base ring-1 ring-black/5/60 dark:ring-white/10"
          >
            {RUN_SCOPES.map((scope, index) => {
              const selected = runScope === scope
              return (
                <label
                  key={scope}
                  className={`flex min-h-10 cursor-pointer items-center gap-3 px-3 py-2.5 text-sm transition-colors ${ index > 0 ? 'border-t border-slate-200/80/80' : '' } ${ selected ? 'bg-surface-raised text-indigo-900 dark:bg-surface-raised dark:text-indigo-100' : 'text-fg-muted hover:bg-white/70 dark:text-fg-muted dark:hover:bg-neutral-900/50' }`}
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
                    className={`h-4 w-4 shrink-0 text-accent-fg motion-safe:transition-[opacity,transform,filter] motion-safe:duration-150 dark:text-indigo-400 ${ selected ? 'scale-100 opacity-100 blur-0' : 'scale-[0.25] opacity-0 blur-[4px]' }`}
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
            <legend className="mb-2 text-xs font-medium text-fg-muted">
              {t('gradingAgent.run.modeLabel')}
            </legend>
            <div
              role="radiogroup"
              aria-label={t('gradingAgent.run.modeLabel')}
              className="overflow-hidden rounded-xl bg-surface-base ring-1 ring-black/5/60 dark:ring-white/10"
            >
              {RUN_MODES.map((mode, index) => {
                const selected = runMode === mode
                return (
                  <label
                    key={mode}
                    className={`flex min-h-10 cursor-pointer items-center gap-3 px-3 py-2.5 text-sm transition-colors ${ index > 0 ? 'border-t border-slate-200/80/80' : '' } ${ selected ? 'bg-surface-raised text-indigo-900 dark:bg-surface-raised dark:text-indigo-100' : 'text-fg-muted hover:bg-white/70 dark:text-fg-muted dark:hover:bg-neutral-900/50' }`}
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
                      className={`h-4 w-4 shrink-0 text-accent-fg motion-safe:transition-[opacity,transform,filter] motion-safe:duration-150 dark:text-indigo-400 ${ selected ? 'scale-100 opacity-100 blur-0' : 'scale-[0.25] opacity-0 blur-[4px]' }`}
                      aria-hidden
                    />
                  </label>
                )
              })}
            </div>
            <p className="mt-2 text-xs text-fg-muted">
              {runMode === 'suggest'
                ? t('gradingAgent.run.mode.suggestNote')
                : t('gradingAgent.run.mode.applyNote')}
            </p>
          </fieldset>
        ) : null}
        {costEstimateEnabled ? (
          <div id={costEstimateId} className="mt-3 rounded-xl bg-surface-base px-3 py-2 text-xs text-fg-muted ring-1 ring-black/5/60 dark:text-fg-muted dark:ring-white/10">
            {runCostEstimateLoading ? (
              <p>{t('gradingAgent.run.cost.loading')}</p>
            ) : runCostEstimate && runCostEstimate.submissionCount > 0 ? (
              <>
                <p>
                  {t('gradingAgent.run.cost.submissions', { count: runCostEstimate.submissionCount })}
                </p>
                {runCostEstimate.hasSample ? (
                  runCostEstimate.estimatedCostMinUsd != null && runCostEstimate.estimatedCostMaxUsd != null ? (
                    <p className="mt-1">
                      {t('gradingAgent.run.cost.approximateRange', {
                        min: formatUsd(runCostEstimate.estimatedCostMinUsd, i18n.language),
                        max: formatUsd(runCostEstimate.estimatedCostMaxUsd, i18n.language),
                      })}
                    </p>
                  ) : runCostEstimate.tokensOnly ? (
                    <p className="mt-1">
                      {t('gradingAgent.run.cost.approximateTokens', {
                        prompt: runCostEstimate.estimatedPromptTokens ?? 0,
                        completion: runCostEstimate.estimatedCompletionTokens ?? 0,
                      })}
                    </p>
                  ) : null
                ) : (
                  <div className="mt-2 flex flex-wrap items-center gap-2">
                    <p>{t('gradingAgent.run.cost.noSample')}</p>
                    {onRequestDryRunEstimate ? (
                      <button
                        type="button"
                        className="font-medium text-accent-fg underline-offset-2 hover:underline dark:text-indigo-300"
                        onClick={() => void onRequestDryRunEstimate()}
                      >
                        {t('gradingAgent.run.cost.dryRunCta')}
                      </button>
                    ) : null}
                  </div>
                )}
                <p className="mt-1 text-fg-muted">{t('gradingAgent.run.cost.disclaimer')}</p>
              </>
            ) : (
              <p>{t('gradingAgent.run.cost.noTarget')}</p>
            )}
          </div>
        ) : null}
        {costEstimateEnabled && setBudgetUsd ? (
          <div className="mt-3">
            <label className="block text-xs font-medium text-fg-muted" htmlFor={`${panelId}-budget`}>
              {t('gradingAgent.run.cost.budgetLabel')}
            </label>
            <input
              id={`${panelId}-budget`}
              type="number"
              min={0}
              step="0.01"
              inputMode="decimal"
              value={budgetUsd ?? ''}
              onChange={(e) => {
                const raw = e.target.value.trim()
                if (raw === '') {
                  setBudgetUsd(null)
                  return
                }
                const parsed = Number(raw)
                setBudgetUsd(Number.isFinite(parsed) && parsed > 0 ? parsed : null)
              }}
              placeholder={t('gradingAgent.run.cost.budgetPlaceholder')}
              className="mt-1 w-full rounded-lg border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default dark:border-border-default dark:bg-surface-raised"
            />
            <p className="mt-1 text-xs text-fg-muted">{t('gradingAgent.run.cost.budgetHelp')}</p>
          </div>
        ) : null}
        {confirmOverwrite ? (
          <p className="mt-3 text-sm text-amber-800 dark:text-amber-200">
            {t('gradingAgent.run.overwriteWarning')}
          </p>
        ) : null}
        {runProgress ? (
          <p className="mt-3 text-sm tabular-nums text-fg-muted">
            {t('gradingAgent.run.progress', {
              completed: runProgress.completed,
              failed: runProgress.failed,
              total: runProgress.total,
            })}
          </p>
        ) : null}
        <label className="mt-3 flex min-h-10 cursor-pointer items-center gap-2 text-sm text-fg-default">
          <input
            type="checkbox"
            className="size-4"
            checked={autoGradeNew}
            onChange={(e) => void onToggleAutoGrade(e.target.checked)}
          />
          {t('gradingAgent.autoGradeNew')}
        </label>
        <label className="mt-1 flex min-h-10 cursor-pointer items-center gap-2 text-sm text-fg-default">
          <input
            type="checkbox"
            className="size-4"
            checked={postPolicy === 'auto_post'}
            onChange={(e) => void onTogglePostPolicy(e.target.checked)}
          />
          {t('gradingAgent.posting.autoPost')}
        </label>
        <p className="mt-1 text-xs text-fg-muted">
          {postPolicy === 'auto_post'
            ? t('gradingAgent.posting.autoPostNote')
            : t('gradingAgent.posting.draftNote')}
        </p>
        <div className="mt-3 border-t border-slate-200/80 pt-3/80">
          <p className="mb-2 text-xs font-medium text-fg-muted">
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
          {batchRunning && cancelRunEnabled ? (
            <Button
              variant="secondary"
              disabled={cancellingRun}
              onClick={() => void onCancelRun?.()}
            >
              {cancellingRun ? (
                <>
                  <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                  <span>{t('gradingAgent.run.cancel.cancelling')}</span>
                </>
              ) : (
                t('gradingAgent.run.cancel.button')
              )}
            </Button>
          ) : null}
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

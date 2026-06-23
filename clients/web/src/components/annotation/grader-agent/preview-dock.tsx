/* eslint-disable react-refresh/only-export-components -- component file exports dock summary helper */
import { useTranslation } from 'react-i18next'
import { RubricGradePicker } from '../../grading/rubric-grade-picker'
import type { GraderAgentDryRunResult, RubricDefinition } from '../../../lib/courses-api'
import type { GraderAgentWorkflowState } from './use-grader-agent-workflow'

type PreviewDockProps = {
  workflow: GraderAgentWorkflowState
  rubric: RubricDefinition | null
  maxPoints: number | null
  submissionId: string | null
}

export function PreviewDock({ workflow, rubric, maxPoints, submissionId }: PreviewDockProps) {
  const { t } = useTranslation('common')
  const { dryRunResult, setDryRunResult, saving, dryRunning, handleApply, handleDryRun } = workflow
  if (!dryRunResult) return null

  return (
    <div className="h-full min-h-0 space-y-3 overflow-y-auto rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-600 dark:bg-neutral-900">
      <p className="text-2xl font-semibold tabular-nums text-slate-900 dark:text-neutral-50">
        {dryRunResult.suggestedPoints}
        {maxPoints != null ? (
          <span className="text-base font-normal text-slate-500"> / {maxPoints}</span>
        ) : null}
      </p>
      {rubric && dryRunResult.rubricScores ? (
        <RubricGradePicker
          rubric={rubric}
          scores={dryRunResult.rubricScores}
          onScoresChange={(scores) =>
            setDryRunResult((prev: GraderAgentDryRunResult | null) =>
              prev ? { ...prev, rubricScores: scores } : prev,
            )
          }
          compact
        />
      ) : null}
      <textarea
        value={dryRunResult.comment}
        onChange={(e) =>
          setDryRunResult((prev: GraderAgentDryRunResult | null) =>
            prev ? { ...prev, comment: e.target.value } : prev,
          )
        }
        rows={4}
        className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
      />
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          disabled={saving || !submissionId}
          onClick={() => void handleApply()}
          className="rounded-lg bg-emerald-600 px-3 py-2 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
        >
          {t('gradingAgent.apply')}
        </button>
        <button
          type="button"
          disabled={dryRunning}
          onClick={() => void handleDryRun()}
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-semibold dark:border-neutral-600"
        >
          {t('gradingAgent.rerun')}
        </button>
      </div>
    </div>
  )
}

export function previewDockSummary(
  dryRunResult: GraderAgentDryRunResult | null,
  maxPoints: number | null,
): string {
  if (!dryRunResult) return ''
  return maxPoints != null
    ? `${dryRunResult.suggestedPoints} / ${maxPoints}`
    : String(dryRunResult.suggestedPoints)
}
import type { CheckResult } from './types'

export type DiagramActionBarProps = {
  canCheck: boolean
  busy: boolean
  exhausted: boolean
  readOnly: boolean
  attemptCount: number
  attemptsLeft: number | null
  checkResult: CheckResult | null
  t: (key: string, opts?: Record<string, unknown>) => string
  onCheck: () => void
  onTryAgain: () => void
}

export function DiagramActionBar({
  canCheck,
  busy,
  exhausted,
  readOnly,
  attemptCount,
  attemptsLeft,
  checkResult,
  t,
  onCheck,
  onTryAgain,
}: DiagramActionBarProps) {
  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          data-testid="diagram-check"
          className="rounded bg-sky-700 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
          disabled={!canCheck || busy || exhausted}
          onClick={onCheck}
        >
          {busy
            ? t('contentTools.tools.diagram_hotspot.checking')
            : t('contentTools.tools.diagram_hotspot.check')}
        </button>
        {attemptCount > 0 && !exhausted ? (
          <button
            type="button"
            data-testid="diagram-try-again"
            className="rounded border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
            disabled={busy || readOnly}
            onClick={onTryAgain}
          >
            {t('contentTools.tools.diagram_hotspot.tryAgain')}
          </button>
        ) : null}
        {typeof checkResult?.scorePct === 'number' ? (
          <span className="text-sm text-slate-700 dark:text-neutral-200" data-testid="diagram-score">
            {t('contentTools.tools.diagram_hotspot.score', {
              score: Math.round(checkResult.scorePct),
            })}
          </span>
        ) : null}
        {attemptsLeft != null && attemptsLeft >= 0 ? (
          <span className="text-xs text-slate-500">
            {t('contentTools.tools.diagram_hotspot.attemptsLeft', { count: attemptsLeft })}
          </span>
        ) : null}
        {exhausted ? (
          <span className="text-sm text-amber-800 dark:text-amber-200">
            {t('contentTools.tools.diagram_hotspot.exhausted')}
          </span>
        ) : null}
      </div>
      {checkResult?.error ? (
        <p className="text-sm text-rose-600" role="alert">
          {checkResult.message || checkResult.error}
        </p>
      ) : null}
    </>
  )
}

import type { CheckTestRow, RunStatus } from './types'

type Props = {
  tab: 'output' | 'tests'
  onTab: (tab: 'output' | 'tests') => void
  status?: RunStatus
  stdout?: string
  stderr?: string
  hint?: string
  tests?: CheckTestRow[]
  passed?: number
  total?: number
  t: (key: string, opts?: Record<string, unknown>) => string
}

export function OutputPanel({
  tab,
  onTab,
  status,
  stdout,
  stderr,
  hint,
  tests,
  passed,
  total,
  t,
}: Props) {
  const hasTests = Array.isArray(tests) && tests.length > 0
  const showTestsTab = hasTests || Boolean(total)

  return (
    <div
      className="overflow-hidden rounded-xl border border-border-default bg-surface-raised dark:border-border-default/60"
      data-testid="code-sandbox-output"
    >
      <div
        role="tablist"
        className="flex gap-1 border-b border-border-default bg-slate-50/80 p-1.5 dark:border-border-default/80"
        aria-label={t('contentTools.tools.code_sandbox.outputTabs')}
      >
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'output'}
          className={
            tab === 'output'
              ? 'rounded-lg bg-surface-raised px-3 py-1.5 text-xs font-semibold text-fg-default shadow-sm dark:bg-surface-overlay'
              : 'rounded-lg px-3 py-1.5 text-xs font-medium text-fg-muted hover:bg-white/70 hover:text-fg-default dark:text-fg-muted dark:hover:bg-neutral-800/80 dark:hover:text-fg-default'
          }
          onClick={() => onTab('output')}
        >
          {t('contentTools.tools.code_sandbox.outputTab')}
        </button>
        {showTestsTab ? (
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'tests'}
            className={
              tab === 'tests'
                ? 'rounded-lg bg-surface-raised px-3 py-1.5 text-xs font-semibold text-fg-default shadow-sm dark:bg-surface-overlay'
                : 'rounded-lg px-3 py-1.5 text-xs font-medium text-fg-muted hover:bg-white/70 hover:text-fg-default dark:text-fg-muted dark:hover:bg-neutral-800/80 dark:hover:text-fg-default'
            }
            onClick={() => onTab('tests')}
          >
            {t('contentTools.tools.code_sandbox.testsTab')}
          </button>
        ) : null}
      </div>

      {tab === 'output' ? (
        <div className="space-y-2.5 p-3" role="tabpanel">
          {status ? (
            <p
              className="text-xs font-medium text-fg-muted"
              data-testid="code-sandbox-status"
            >
              {t(`contentTools.tools.code_sandbox.status.${status}`)}
            </p>
          ) : null}
          {stdout ? (
            <pre
              className="max-h-48 overflow-auto whitespace-pre-wrap rounded-lg border border-border-subtle bg-surface-base p-3 font-mono text-xs text-fg-default dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
              data-testid="code-sandbox-stdout"
            >
              {stdout}
            </pre>
          ) : null}
          {stderr ? (
            <pre
              className="max-h-48 overflow-auto whitespace-pre-wrap rounded-lg border border-rose-200 bg-rose-50 p-3 font-mono text-xs text-rose-900 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-100"
              data-testid="code-sandbox-stderr"
            >
              {stderr}
            </pre>
          ) : null}
          {hint ? (
            <p
              className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100"
              data-testid="code-sandbox-hint"
            >
              {hint}
            </p>
          ) : null}
          {!stdout && !stderr && !status ? (
            <p className="text-xs text-fg-muted">
              {t('contentTools.tools.code_sandbox.outputEmpty')}
            </p>
          ) : null}
        </div>
      ) : (
        <div className="space-y-2.5 p-3" role="tabpanel" data-testid="code-sandbox-tests">
          {typeof passed === 'number' && typeof total === 'number' ? (
            <p className="text-sm font-semibold text-fg-default">
              {t('contentTools.tools.code_sandbox.testsSummary', { passed, total })}
            </p>
          ) : null}
          <ul className="space-y-1.5">
            {(tests ?? []).map((tr) => (
              <li
                key={tr.id}
                className={`flex flex-wrap items-baseline gap-2 rounded-lg border px-3 py-2 text-sm ${ tr.passed ? 'border-emerald-200 bg-emerald-50/70 dark:border-emerald-900/40 dark:bg-emerald-950/20' : 'border-rose-200 bg-rose-50/70 dark:border-rose-900/40 dark:bg-rose-950/20' }`}
                data-testid={`code-sandbox-test-${tr.id}`}
                data-passed={tr.passed ? 'true' : 'false'}
              >
                <span
                  className={
                    tr.passed
                      ? 'font-semibold text-emerald-700 dark:text-emerald-300'
                      : 'font-semibold text-rose-700 dark:text-rose-300'
                  }
                  aria-hidden="true"
                >
                  {tr.passed ? '✓' : '✗'}
                </span>
                <span className="sr-only">
                  {tr.passed
                    ? t('contentTools.tools.code_sandbox.testPassed')
                    : t('contentTools.tools.code_sandbox.testFailed')}
                </span>
                <span className="font-medium text-fg-default">{tr.name}</span>
                {tr.hidden ? (
                  <span className="rounded-full bg-surface-sunken px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-fg-muted dark:bg-surface-overlay dark:text-fg-muted">
                    {t('contentTools.tools.code_sandbox.hiddenTest')}
                  </span>
                ) : null}
                {tr.feedback ? (
                  <span className="basis-full text-xs text-fg-muted">
                    {tr.feedback}
                  </span>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}

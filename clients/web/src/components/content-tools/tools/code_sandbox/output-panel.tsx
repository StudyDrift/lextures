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
  return (
    <div className="mt-3 space-y-2" data-testid="code-sandbox-output">
      <div role="tablist" className="flex gap-2 text-xs" aria-label={t('contentTools.tools.code_sandbox.outputTabs')}>
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'output'}
          className={`rounded px-2 py-1 ${tab === 'output' ? 'bg-slate-800 text-white dark:bg-neutral-200 dark:text-neutral-900' : 'bg-slate-100 dark:bg-neutral-800'}`}
          onClick={() => onTab('output')}
        >
          {t('contentTools.tools.code_sandbox.outputTab')}
        </button>
        {hasTests || total ? (
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'tests'}
            className={`rounded px-2 py-1 ${tab === 'tests' ? 'bg-slate-800 text-white dark:bg-neutral-200 dark:text-neutral-900' : 'bg-slate-100 dark:bg-neutral-800'}`}
            onClick={() => onTab('tests')}
          >
            {t('contentTools.tools.code_sandbox.testsTab')}
          </button>
        ) : null}
      </div>

      {tab === 'output' ? (
        <div className="space-y-2" role="tabpanel">
          {status ? (
            <p className="text-xs text-slate-600 dark:text-neutral-400" data-testid="code-sandbox-status">
              {t(`contentTools.tools.code_sandbox.status.${status}`)}
            </p>
          ) : null}
          {stdout ? (
            <pre
              className="max-h-48 overflow-auto whitespace-pre-wrap rounded bg-slate-50 p-2 font-mono text-xs dark:bg-neutral-900"
              data-testid="code-sandbox-stdout"
            >
              {stdout}
            </pre>
          ) : null}
          {stderr ? (
            <pre
              className="max-h-48 overflow-auto whitespace-pre-wrap rounded border border-rose-200 bg-rose-50 p-2 font-mono text-xs text-rose-900 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-100"
              data-testid="code-sandbox-stderr"
            >
              {stderr}
            </pre>
          ) : null}
          {hint ? (
            <p className="text-sm text-amber-800 dark:text-amber-200" data-testid="code-sandbox-hint">
              {hint}
            </p>
          ) : null}
          {!stdout && !stderr && !status ? (
            <p className="text-xs text-slate-500">{t('contentTools.tools.code_sandbox.outputEmpty')}</p>
          ) : null}
        </div>
      ) : (
        <div className="space-y-2" role="tabpanel" data-testid="code-sandbox-tests">
          {typeof passed === 'number' && typeof total === 'number' ? (
            <p className="text-sm font-medium">
              {t('contentTools.tools.code_sandbox.testsSummary', { passed, total })}
            </p>
          ) : null}
          <ul className="space-y-1">
            {(tests ?? []).map((tr) => (
              <li
                key={tr.id}
                className="flex flex-wrap items-baseline gap-2 rounded border border-slate-200 px-2 py-1.5 text-sm dark:border-neutral-700"
                data-testid={`code-sandbox-test-${tr.id}`}
                data-passed={tr.passed ? 'true' : 'false'}
              >
                <span aria-hidden="true">{tr.passed ? '✓' : '✗'}</span>
                <span className="sr-only">
                  {tr.passed
                    ? t('contentTools.tools.code_sandbox.testPassed')
                    : t('contentTools.tools.code_sandbox.testFailed')}
                </span>
                <span className="font-medium">{tr.name}</span>
                {tr.hidden ? (
                  <span className="text-xs text-slate-500">
                    {t('contentTools.tools.code_sandbox.hiddenTest')}
                  </span>
                ) : null}
                {tr.feedback ? (
                  <span className="basis-full text-xs text-slate-600 dark:text-neutral-400">
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

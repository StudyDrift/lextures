/* eslint-disable react-refresh/only-export-components -- component file exports console summary helper */
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import type { DryRunLogEntry } from './use-grader-agent-workflow'

type DryRunConsoleProps = {
  logs: DryRunLogEntry[]
}

export function DryRunConsole({ logs }: DryRunConsoleProps) {
  const { t } = useTranslation('common')
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    el.scrollTop = el.scrollHeight
  }, [logs])

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden rounded-xl border border-slate-800 bg-slate-950 text-slate-100 dark:border-neutral-700">
      <div
        ref={scrollRef}
        role="log"
        aria-live="polite"
        aria-relevant="additions"
        className="min-h-0 flex-1 overflow-y-auto px-3 py-2 font-mono text-xs leading-relaxed"
      >
        {logs.length === 0 ? (
          <p className="text-slate-500">{t('gradingAgent.dryRun.console.empty')}</p>
        ) : (
          logs.map((entry, index) => (
            <p
              key={`${index}-${entry.message}`}
              className={
                entry.level === 'error'
                  ? 'text-rose-300'
                  : entry.level === 'warn'
                    ? 'text-amber-300'
                    : 'text-slate-200'
              }
            >
              {entry.message}
            </p>
          ))
        )}
      </div>
    </div>
  )
}

export function dryRunConsoleSummary(
  logs: DryRunLogEntry[],
  running: boolean,
  runningLabel: string,
  emptyLabel: string,
): string {
  if (running) return runningLabel
  const last = logs[logs.length - 1]
  if (!last?.message) return emptyLabel
  const trimmed = last.message.trim()
  return trimmed.length > 72 ? `${trimmed.slice(0, 72)}…` : trimmed
}
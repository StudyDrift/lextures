import type { ReactNode } from 'react'

export function ToolShell({
  children,
  label,
}: {
  children: ReactNode
  label?: string
}) {
  return (
    <section className="lex-tool-shell space-y-3" aria-label={label} data-tool-shell>
      {children}
    </section>
  )
}

export function ToolPrompt({ children }: { children: ReactNode }) {
  return <div className="lex-tool-prompt text-sm" data-tool-prompt>{children}</div>
}

export function ToolActions({ children }: { children: ReactNode }) {
  return (
    <div className="lex-tool-actions flex flex-wrap items-center gap-2" data-tool-actions>
      {children}
    </div>
  )
}

export function ToolFeedback({
  children,
  tone = 'neutral',
}: {
  children: ReactNode
  tone?: 'neutral' | 'success' | 'error'
}) {
  const cls =
    tone === 'success'
      ? 'text-emerald-700 dark:text-emerald-300'
      : tone === 'error'
        ? 'text-rose-700 dark:text-rose-300'
        : 'text-slate-600 dark:text-neutral-300'
  return (
    <p className={`lex-tool-feedback text-xs ${cls}`} role="status" data-tool-feedback={tone}>
      {children}
    </p>
  )
}

export function ToolScore({ raw, max }: { raw: number; max: number }) {
  return (
    <p className="lex-tool-score text-xs text-slate-600 dark:text-neutral-300" data-tool-score>
      {raw}/{max}
    </p>
  )
}

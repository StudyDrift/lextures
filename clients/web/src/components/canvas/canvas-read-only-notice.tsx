import { ShieldCheck } from 'lucide-react'

type Props = {
  className?: string
  variant?: 'default' | 'compact'
}

/** Explains that Canvas import only uses read API access and does not mutate Canvas. */
export function CanvasReadOnlyNotice({ className = '', variant = 'default' }: Props) {
  if (variant === 'compact') {
    return (
      <div
        role="note"
        className={[
          'flex h-full gap-2.5 rounded-lg bg-sky-50/80 px-3 py-2.5 shadow-[inset_0_0_0_1px_rgba(14,165,233,0.18)] dark:bg-sky-950/30 dark:shadow-[inset_0_0_0_1px_rgba(56,189,248,0.16)]',
          className,
        ]
          .filter(Boolean)
          .join(' ')}
      >
        <ShieldCheck
          className="mt-0.5 h-4 w-4 shrink-0 text-sky-600 dark:text-sky-400"
          aria-hidden
        />
        <div className="min-w-0">
          <p className="text-sm font-medium text-sky-950 dark:text-sky-100">Read-only on Canvas</p>
          <p className="mt-0.5 text-pretty text-xs leading-relaxed text-sky-900/85 dark:text-sky-100/85">
            Lextures only reads from Canvas. Nothing in your account or courses is created, changed,
            or deleted.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div
      role="note"
      className={[
        'rounded-xl border border-sky-200 bg-sky-50 px-3 py-2.5 text-sm text-sky-950 dark:border-sky-900/50 dark:bg-sky-950/40 dark:text-sky-100',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <p className="text-balance font-medium">Read-only on Canvas</p>
      <p className="mt-0.5 text-pretty text-sky-900/90 dark:text-sky-100/90">
        Lextures only reads from Canvas (view courses, modules, assignments, and related data). Nothing
        in your Canvas account or courses is created, changed, or deleted. Import creates and updates
        content in Lextures only.
      </p>
    </div>
  )
}

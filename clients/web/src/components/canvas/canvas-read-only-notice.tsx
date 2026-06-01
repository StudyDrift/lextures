type Props = {
  className?: string
}

/** Explains that Canvas import only uses read API access and does not mutate Canvas. */
export function CanvasReadOnlyNotice({ className = '' }: Props) {
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
      <p className="font-medium">Read-only on Canvas</p>
      <p className="mt-0.5 text-sky-900/90 dark:text-sky-100/90">
        Lextures only reads from Canvas (view courses, modules, assignments, and related data). Nothing
        in your Canvas account or courses is created, changed, or deleted. Import creates and updates
        content in Lextures only.
      </p>
    </div>
  )
}

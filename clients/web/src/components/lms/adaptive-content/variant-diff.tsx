import { diffLines } from '../../../lib/text-diff'

type Props = {
  baseMarkdown: string
  variantMarkdown: string
  className?: string
}

/**
 * Accessible line diff: not color-only — each change is marked with icons + text labels.
 */
export function VariantDiff({ baseMarkdown, variantMarkdown, className = '' }: Props) {
  const lines = diffLines(baseMarkdown, variantMarkdown)
  return (
    <div
      className={`overflow-auto rounded-lg border border-border-default bg-surface-raised text-sm dark:border-border-default dark:bg-surface-base ${className}`}
      role="region"
      aria-label="Base versus variant diff"
    >
      <pre className="m-0 whitespace-pre-wrap break-words p-3 font-mono text-xs leading-relaxed text-fg-default">
        {lines.map((line, idx) => {
          if (line.type === 'same') {
            return (
              <div key={idx} className="text-fg-muted">
                <span className="select-none text-fg-subtle dark:text-neutral-600" aria-hidden>
                  {'  '}
                </span>
                {line.text || ' '}
              </div>
            )
          }
          if (line.type === 'add') {
            return (
              <div
                key={idx}
                className="bg-emerald-50 text-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-100"
              >
                <span className="mr-1 font-semibold" aria-label="added">
                  +
                </span>
                <span className="sr-only">added: </span>
                {line.text || ' '}
              </div>
            )
          }
          return (
            <div
              key={idx}
              className="bg-rose-50 text-rose-900 dark:bg-rose-950/40 dark:text-rose-100"
            >
              <span className="mr-1 font-semibold" aria-label="removed">
                −
              </span>
              <span className="sr-only">removed: </span>
              {line.text || ' '}
            </div>
          )
        })}
      </pre>
    </div>
  )
}

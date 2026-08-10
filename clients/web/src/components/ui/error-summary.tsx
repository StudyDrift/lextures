import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  type HTMLAttributes,
  type ReactNode,
} from 'react'
import { cx } from './utils'

export type ErrorSummaryItem = {
  /** Control id to focus when the link is activated. */
  id: string
  /** Visible field label. */
  label: ReactNode
  /** Actionable error text. */
  message: ReactNode
}

export type ErrorSummaryProps = Omit<HTMLAttributes<HTMLDivElement>, 'title'> & {
  /** Heading announced with the alert. */
  title?: ReactNode
  errors: ErrorSummaryItem[]
  /**
   * When true (default), moves focus to the summary whenever `errors` becomes non-empty
   * or its content identity changes after a failed submit (UX.6 FR-6 / AC-3).
   */
  autoFocus?: boolean
}

export type ErrorSummaryHandle = {
  focus: () => void
}

/**
 * Form-level error summary: role="alert", focus target on failed submit,
 * each entry is a link that moves focus to the offending field (UX.6 FR-6).
 */
export const ErrorSummary = forwardRef<ErrorSummaryHandle, ErrorSummaryProps>(
  function ErrorSummary(
    {
      title = 'Please fix the following errors',
      errors,
      autoFocus = true,
      className = '',
      ...props
    },
    ref,
  ) {
    const rootRef = useRef<HTMLDivElement>(null)
    const prevKey = useRef<string>('')

    useImperativeHandle(ref, () => ({
      focus: () => rootRef.current?.focus(),
    }))

    const key = errors.map((e) => `${e.id}:${String(e.message)}`).join('|')

    useEffect(() => {
      if (!autoFocus || errors.length === 0) return
      if (key === prevKey.current) return
      prevKey.current = key
      // Defer so the alert is in the DOM and announced before focus moves.
      const t = window.setTimeout(() => rootRef.current?.focus(), 0)
      return () => window.clearTimeout(t)
    }, [autoFocus, errors.length, key])

    if (errors.length === 0) return null

    return (
      <div
        ref={rootRef}
        role="alert"
        tabIndex={-1}
        className={cx(
          'rounded-xl border border-danger-fg/40 bg-danger-surface px-4 py-3 text-sm text-danger-fg outline-none focus-visible:ring-2 focus-visible:ring-danger-fg/40',
          className,
        )}
        {...props}
      >
        <p className="font-semibold">{title}</p>
        <ul className="mt-2 list-inside list-disc space-y-1">
          {errors.map((item) => (
            <li key={item.id}>
              <a
                href={`#${item.id}`}
                className="font-medium underline underline-offset-2 hover:no-underline focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-danger-fg/50"
                onClick={(e) => {
                  e.preventDefault()
                  const el = document.getElementById(item.id)
                  if (!el) return
                  el.focus()
                  el.scrollIntoView({ block: 'center', behavior: 'smooth' })
                }}
              >
                <span className="font-semibold">{item.label}</span>
                {': '}
                {item.message}
              </a>
            </li>
          ))}
        </ul>
      </div>
    )
  },
)

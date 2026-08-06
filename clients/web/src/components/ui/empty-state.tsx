import { useId, type ComponentType, type ReactNode } from 'react'
import { Button } from './button'
import { LinkButton } from './link-button'

export type EmptyStateAction =
  | { label: string; to: string }
  | { label: string; onClick: () => void }

export type EmptyStateProps = {
  icon: ComponentType<{ className?: string; 'aria-hidden'?: boolean | 'true' }>
  title: string
  body?: ReactNode
  primaryAction?: EmptyStateAction
  secondaryAction?: EmptyStateAction
  className?: string
}

function ActionButton({ action, variant }: { action: EmptyStateAction; variant: 'primary' | 'secondary' }) {
  if ('to' in action) {
    return (
      <LinkButton to={action.to} variant={variant === 'primary' ? 'primary' : 'secondary'}>
        {action.label}
      </LinkButton>
    )
  }
  return (
    <Button type="button" variant={variant === 'primary' ? 'primary' : 'secondary'} onClick={action.onClick}>
      {action.label}
    </Button>
  )
}

export function EmptyState({
  icon: Icon,
  title,
  body,
  primaryAction,
  secondaryAction,
  className = '',
}: EmptyStateProps) {
  const titleId = useId()
  return (
    <section
      className={`rounded-2xl bg-surface-sunken/80 px-6 py-12 shadow-card/40 ${className}`}
      role="status"
      aria-labelledby={titleId}
    >
      <div className="mx-auto flex max-w-md flex-col items-center text-center">
        <span className="flex h-12 w-12 items-center justify-center rounded-xl bg-surface-raised text-fg-subtle shadow-sm ring-1 ring-border-default">
          <Icon className="h-6 w-6 shrink-0" aria-hidden />
        </span>
        <h2 id={titleId} className="mt-4 text-base font-semibold tracking-tight text-fg-default">
          {title}
        </h2>
        {body != null && body !== false ? (
          <div className="mt-2 text-sm leading-relaxed text-fg-muted">{body}</div>
        ) : null}
        {(primaryAction || secondaryAction) && (
          <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
            {primaryAction ? <ActionButton action={primaryAction} variant="primary" /> : null}
            {secondaryAction ? <ActionButton action={secondaryAction} variant="secondary" /> : null}
          </div>
        )}
      </div>
    </section>
  )
}

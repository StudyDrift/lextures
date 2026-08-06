import { type ComponentType, type ReactNode } from 'react'
import { EmptyState, type EmptyStateAction } from './empty-state'

export type ErrorStateProps = {
  title: string
  body?: ReactNode
  icon?: ComponentType<{ className?: string; 'aria-hidden'?: boolean | 'true' }>
  primaryAction?: EmptyStateAction
  secondaryAction?: EmptyStateAction
  className?: string
}

function DefaultErrorIcon({ className, ...rest }: { className?: string; 'aria-hidden'?: boolean | 'true' }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      className={className}
      {...rest}
    >
      <circle cx="12" cy="12" r="9" />
      <path d="M12 8v5" strokeLinecap="round" />
      <circle cx="12" cy="16" r="0.75" fill="currentColor" stroke="none" />
    </svg>
  )
}

/** Error empty-state variant; reuses EmptyState layout and Button actions. */
export function ErrorState({
  title,
  body,
  icon = DefaultErrorIcon,
  primaryAction,
  secondaryAction,
  className = '',
}: ErrorStateProps) {
  return (
    <EmptyState
      icon={icon}
      title={title}
      body={body}
      primaryAction={primaryAction}
      secondaryAction={secondaryAction}
      className={className}
    />
  )
}

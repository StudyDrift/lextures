import type { ReactNode } from 'react'

type Props = {
  id?: string
  title: string
  description?: string
  children: ReactNode
  className?: string
}

export function SettingsSection({ id, title, description, children, className = '' }: Props) {
  return (
    <section
      id={id}
      aria-labelledby={id ? `${id}-heading` : undefined}
      className={`rounded-2xl border border-border-default bg-surface-raised p-5 sm:p-6 dark:border-border-default dark:bg-surface-raised ${className}`}
    >
      <header>
        <h3
          id={id ? `${id}-heading` : undefined}
          className="text-base font-semibold text-fg-default"
        >
          {title}
        </h3>
        {description ? (
          <p className="mt-1 text-sm text-fg-muted">{description}</p>
        ) : null}
      </header>
      <div className="mt-5">{children}</div>
    </section>
  )
}
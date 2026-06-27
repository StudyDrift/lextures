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
      className={`rounded-2xl border border-slate-200 bg-white p-5 sm:p-6 dark:border-neutral-700 dark:bg-neutral-900 ${className}`}
    >
      <header>
        <h3
          id={id ? `${id}-heading` : undefined}
          className="text-base font-semibold text-slate-900 dark:text-neutral-100"
        >
          {title}
        </h3>
        {description ? (
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{description}</p>
        ) : null}
      </header>
      <div className="mt-5">{children}</div>
    </section>
  )
}
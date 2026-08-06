import type { ReactNode } from 'react'

export function GalleryBlock({
  id,
  title,
  pattern,
  keyboard,
  children,
}: {
  id: string
  title: string
  pattern: string
  keyboard: string
  children: ReactNode
}) {
  return (
    <section
      id={id}
      data-gallery-component={id}
      className="scroll-mt-20 rounded-2xl border border-border-default bg-surface-raised p-5"
    >
      <header className="mb-4 border-b border-border-default pb-3">
        <h2 className="text-lg font-semibold text-fg-default">{title}</h2>
        <p className="mt-1 text-xs text-fg-muted">
          <span className="font-medium text-fg-default">ARIA:</span> {pattern}
        </p>
        <p className="text-xs text-fg-muted">
          <span className="font-medium text-fg-default">Keyboard:</span> {keyboard}
        </p>
      </header>
      <div className="flex flex-col gap-4">{children}</div>
    </section>
  )
}

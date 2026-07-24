import type { ReactNode } from 'react'

export type BlockFloatingToolbarProps = {
  /** Describes the toolbar for assistive tech. */
  label?: string
  /** When true, omits outer chrome for use inside a parent card (e.g. generate panel stack). */
  embedded?: boolean
  children: ReactNode
}

/**
 * Pill that holds text-formatting controls next to the caret while writing.
 *
 * Block management (reorder, delete) deliberately lives in the block header
 * instead, so this bar only ever changes the text you are typing.
 */
export function BlockFloatingToolbar({
  label = 'Text formatting',
  embedded = false,
  children,
}: BlockFloatingToolbarProps) {
  return (
    <div
      data-toolbar-anchor
      className={[
        'pointer-events-auto flex h-9 w-max max-w-[calc(100vw-2rem)] items-center gap-0.5 px-1 py-0.5',
        embedded
          ? 'w-full rounded-none border-0 bg-transparent shadow-none'
          : 'rounded-lg border border-slate-200 bg-white shadow-md shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40',
      ].join(' ')}
      onClick={(e) => e.stopPropagation()}
      onKeyDown={(e) => e.stopPropagation()}
      role="toolbar"
      aria-label={label}
    >
      {children}
    </div>
  )
}

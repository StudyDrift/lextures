import { useState, type ReactNode } from 'react'

export type EditorSidebarTab = 'document' | 'block'

export type EditorSidebarProps = {
  documentLabel?: string
  blockLabel?: string
  documentPanel: ReactNode
  blockPanel: ReactNode
  /** When true, Block tab shows a disabled empty state. */
  blockDisabled?: boolean
  blockDisabledMessage?: string
}

/**
 * Right column tabs: global document settings vs. selected block (Gutenberg-style).
 */
export function EditorSidebar({
  documentLabel = 'Document',
  blockLabel = 'Block',
  documentPanel,
  blockPanel,
  blockDisabled,
  blockDisabledMessage = 'Select a block to see its settings.',
}: EditorSidebarProps) {
  const [tab, setTab] = useState<EditorSidebarTab>('block')

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div
        className="flex shrink-0 border-b border-border-default"
        role="tablist"
        aria-label="Editor settings"
      >
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'document'}
          className={`flex-1 px-3 py-2.5 text-center text-xs font-semibold uppercase tracking-wide transition-[background-color,color,border-color] ${ tab === 'document' ? 'border-b-2 border-indigo-600 text-fg-default dark:border-indigo-500 dark:text-fg-default' : 'border-b-2 border-transparent text-fg-muted hover:text-fg-default dark:text-fg-muted dark:hover:text-fg-default' }`}
          onClick={() => setTab('document')}
        >
          {documentLabel}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'block'}
          className={`flex-1 px-3 py-2.5 text-center text-xs font-semibold uppercase tracking-wide transition-[background-color,color,border-color] ${ tab === 'block' ? 'border-b-2 border-indigo-600 text-fg-default dark:border-indigo-500 dark:text-fg-default' : 'border-b-2 border-transparent text-fg-muted hover:text-fg-default dark:text-fg-muted dark:hover:text-fg-default' }`}
          onClick={() => setTab('block')}
        >
          {blockLabel}
        </button>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-3" role="tabpanel">
        {tab === 'document' && documentPanel}
        {tab === 'block' &&
          (blockDisabled ? (
            <p className="rounded-lg border border-dashed border-border-default bg-surface-base px-3 py-6 text-center text-sm text-fg-muted dark:border-border-default/80 dark:text-fg-muted">
              {blockDisabledMessage}
            </p>
          ) : (
            blockPanel
          ))}
      </div>
    </div>
  )
}

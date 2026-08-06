import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  filterToolPalette,
  groupToolsByCategory,
  type ToolPaletteItem,
} from './tool-palette-utils'

export type { ToolPaletteItem }

export type ToolPaletteListProps = {
  tools: ToolPaletteItem[]
  onSelect: (toolId: string) => void
  disabled?: boolean
  disabledReason?: string
  loading?: boolean
  emptyMessage?: string
  searchPlaceholder?: string
}

export function ToolPaletteList({
  tools,
  onSelect,
  disabled,
  disabledReason,
  loading,
  emptyMessage,
  searchPlaceholder,
}: ToolPaletteListProps) {
  const { t } = useTranslation('contentTools')
  const searchId = useId()
  const [query, setQuery] = useState('')
  const [highlightedId, setHighlightedId] = useState<string | null>(null)
  const filtered = filterToolPalette(tools, query)
  const groups = groupToolsByCategory(filtered)

  if (loading) {
    return (
      <div className="space-y-2 p-2" aria-busy="true">
        <div className="h-8 motion-safe:animate-pulse rounded bg-surface-sunken" />
        <div className="h-10 motion-safe:animate-pulse rounded bg-surface-sunken" />
        <div className="h-10 motion-safe:animate-pulse rounded bg-surface-sunken" />
      </div>
    )
  }

  return (
    <div className="flex max-h-72 flex-col">
      <div className="shrink-0 border-b border-border-default p-2 dark:border-border-default">
        <label htmlFor={searchId} className="sr-only">
          {searchPlaceholder ?? t('contentTools.authoring.searchTools')}
        </label>
        <input
          id={searchId}
          type="search"
          value={query}
          autoComplete="off"
          placeholder={searchPlaceholder ?? t('contentTools.authoring.searchTools')}
          onChange={(e) => setQuery(e.target.value)}
          className="w-full rounded-md border border-border-default bg-surface-raised px-2.5 py-1.5 text-sm text-fg-default placeholder:text-fg-subtle focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:placeholder:text-neutral-500"
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-1" role="listbox">
        {groups.length === 0 ? (
          <p className="px-2 py-3 text-xs text-fg-muted">
            {emptyMessage ?? t('contentTools.authoring.noToolsMatch')}
          </p>
        ) : (
          groups.map((group) => (
            <div key={group.category} className="mb-1">
              <p className="px-2 py-1 text-[10px] font-semibold uppercase tracking-wide text-fg-muted">
                {group.category}
              </p>
              <ul className="space-y-0.5">
                {group.tools.map((tool) => {
                  const label = t(`contentTools.tools.${tool.id}.name`, {
                    defaultValue: tool.name,
                  })
                  const description = t(`contentTools.tools.${tool.id}.description`, {
                    defaultValue: tool.description ?? '',
                  })
                  const selected = highlightedId === tool.id
                  return (
                    <li key={tool.id}>
                      <button
                        type="button"
                        role="option"
                        aria-selected={selected}
                        disabled={disabled}
                        title={disabled ? disabledReason : undefined}
                        onMouseEnter={() => setHighlightedId(tool.id)}
                        onFocus={() => setHighlightedId(tool.id)}
                        onClick={() => onSelect(tool.id)}
                        className="flex w-full flex-col rounded-md px-2 py-1.5 text-start hover:bg-surface-sunken disabled:cursor-not-allowed disabled:opacity-40 dark:hover:bg-surface-overlay"
                      >
                        <span className="text-sm font-medium text-fg-default">
                          {label}
                        </span>
                        {description ? (
                          <span className="text-xs text-fg-muted">
                            {description}
                          </span>
                        ) : null}
                      </button>
                    </li>
                  )
                })}
              </ul>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

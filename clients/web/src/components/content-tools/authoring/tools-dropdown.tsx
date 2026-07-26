import { ChevronDown } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ToolPaletteList } from './tool-palette-list'
import type { ToolPaletteItem } from './tool-palette-utils'

export type ToolsDropdownProps = {
  tools: ToolPaletteItem[]
  onSelect: (toolId: string) => void
  disabled?: boolean
  atMaxInstances?: boolean
  loading?: boolean
  emptyCatalog?: boolean
  settingsHref?: string
}

export function ToolsDropdown({
  tools,
  onSelect,
  disabled,
  atMaxInstances,
  loading,
  emptyCatalog,
  settingsHref,
}: ToolsDropdownProps) {
  const { t } = useTranslation('contentTools')
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const buttonId = useId()
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  const triggerDisabled = Boolean(disabled)
  const entriesDisabled = Boolean(atMaxInstances)
  const disabledReason = atMaxInstances
    ? t('contentTools.authoring.maxInstancesReached')
    : undefined

  return (
    <div ref={rootRef} className="relative shrink-0">
      <button
        type="button"
        id={buttonId}
        disabled={triggerDisabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        title={disabledReason}
        onMouseDown={(e) => e.preventDefault()}
        onClick={() => setOpen((v) => !v)}
        className="inline-flex shrink-0 items-center gap-0.5 rounded px-2 py-1 text-xs font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-neutral-200 dark:hover:bg-neutral-700"
      >
        {t('contentTools.authoring.tools')}
        <ChevronDown className="h-3.5 w-3.5 opacity-70" aria-hidden />
      </button>
      {open ? (
        <div
          id={menuId}
          role="menu"
          aria-labelledby={buttonId}
          className="absolute start-0 top-full z-[60] mt-1 w-72 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-900"
        >
          {emptyCatalog && !loading ? (
            <div className="space-y-2 p-3 text-xs text-slate-600 dark:text-neutral-300">
              <p>{t('contentTools.authoring.noToolsEnabled')}</p>
              {settingsHref ? (
                <a
                  href={settingsHref}
                  className="font-medium text-slate-800 underline dark:text-neutral-100"
                >
                  {t('contentTools.authoring.openSettings')}
                </a>
              ) : null}
            </div>
          ) : (
            <ToolPaletteList
              tools={tools}
              loading={loading}
              disabled={entriesDisabled}
              disabledReason={disabledReason}
              onSelect={(toolId) => {
                if (entriesDisabled) return
                setOpen(false)
                onSelect(toolId)
              }}
            />
          )}
        </div>
      ) : null}
    </div>
  )
}

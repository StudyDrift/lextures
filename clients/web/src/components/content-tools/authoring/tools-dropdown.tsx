import { ChevronDown } from 'lucide-react'
import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { ToolPaletteList } from './tool-palette-list'
import type { ToolPaletteItem } from './tool-palette-utils'
import { handleMenuKeyDown, focusFirstMenuitem } from '../../../lib/a11y/menu-keyboard'

export type ToolsDropdownProps = {
  tools: ToolPaletteItem[]
  onSelect: (toolId: string) => void
  disabled?: boolean
  atMaxInstances?: boolean
  loading?: boolean
  emptyCatalog?: boolean
  settingsHref?: string
}

type MenuPos = { left: number; top: number; width: number }

/**
 * Portals the menu to document.body so it paints above sticky block headers,
 * content-tool cards, and other editor stacking contexts. Absolute positioning
 * inside the sticky toolbar (z-20) let page content show through the menu.
 */
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
  const menuTypeaheadRef = useRef({ buffer: '', at: 0 })
  const menuListRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    focusFirstMenuitem(menuListRef.current)
  }, [open])
  const [menuPos, setMenuPos] = useState<MenuPos | null>(null)
  const rootRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const buttonId = useId()
  const menuId = useId()

  const updateMenuPos = () => {
    const btn = buttonRef.current
    if (!btn) return
    const rect = btn.getBoundingClientRect()
    const width = 288 // w-72
    const left = Math.min(rect.left, window.innerWidth - width - 8)
    setMenuPos({
      left: Math.max(8, left),
      top: rect.bottom + 4,
      width,
    })
  }

  useLayoutEffect(() => {
    if (!open) {
      setMenuPos(null)
      return
    }
    updateMenuPos()
  }, [open])

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node
      if (rootRef.current?.contains(target)) return
      if (menuRef.current?.contains(target)) return
      setOpen(false)
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    const onReposition = () => updateMenuPos()
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    window.addEventListener('resize', onReposition)
    // Capture scroll from any scrollable ancestor (block editor canvas, etc.).
    window.addEventListener('scroll', onReposition, true)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
      window.removeEventListener('resize', onReposition)
      window.removeEventListener('scroll', onReposition, true)
    }
  }, [open])

  const triggerDisabled = Boolean(disabled)
  const entriesDisabled = Boolean(atMaxInstances)
  const disabledReason = atMaxInstances
    ? t('contentTools.authoring.maxInstancesReached')
    : undefined

  const menu =
    open && menuPos
      ? createPortal(
          <div
            ref={menuRef}
            id={menuId}
            role="menu"
            aria-labelledby={buttonId}
            style={{
              position: 'fixed',
              left: menuPos.left,
              top: menuPos.top,
              width: menuPos.width,
              zIndex: 80,
            }}
            className="overflow-hidden rounded-lg border border-border-default bg-surface-raised shadow-lg shadow-slate-900/10 dark:border-border-default dark:bg-surface-raised"
           onKeyDown={(e) => handleMenuKeyDown(e, { onClose: () => setOpen(false) }, menuTypeaheadRef.current)} tabIndex={-1}>
            {emptyCatalog && !loading ? (
              <div className="space-y-2 p-3 text-xs text-fg-muted">
                <p>{t('contentTools.authoring.noToolsEnabled')}</p>
                {settingsHref ? (
                  <a
                    href={settingsHref}
                    className="font-medium text-fg-default underline dark:text-fg-default"
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
          </div>,
          document.body,
        )
      : null

  return (
    <div ref={rootRef} className="relative shrink-0">
      <button
        ref={buttonRef}
        type="button"
        id={buttonId}
        disabled={triggerDisabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        title={disabledReason}
        onMouseDown={(e) => e.preventDefault()}
        onClick={() => setOpen((v) => !v)}
        className="inline-flex shrink-0 items-center gap-0.5 rounded px-2 py-1 text-xs font-medium text-fg-muted hover:bg-surface-sunken disabled:cursor-not-allowed disabled:opacity-40 dark:text-fg-default dark:hover:bg-neutral-700"
      >
        {t('contentTools.authoring.tools')}
        <ChevronDown className="h-3.5 w-3.5 opacity-70" aria-hidden />
      </button>
      {menu}
    </div>
  )
}

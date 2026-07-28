import { ChevronDown, ChevronLeft, FileText, Plus, Wrench } from 'lucide-react'
import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { ToolPaletteList } from './tool-palette-list'
import type { ToolPaletteItem } from './tool-palette-utils'

export type AddSectionDropdownProps = {
  /** Visual style matching the insert affordance placement. */
  variant: 'divider' | 'row'
  /** Button label when closed (e.g. "Add a section"). */
  label: string
  onAddContent: () => void
  onAddTool: (toolId: string) => void
  tools: ToolPaletteItem[]
  disabled?: boolean
  atMaxInstances?: boolean
  loading?: boolean
  emptyCatalog?: boolean
  settingsHref?: string
}

type MenuPos = {
  left: number
  width: number
  /** Distance from viewport top when opening downward. */
  top?: number
  /** Distance from viewport bottom when opening upward (bottom insert row). */
  bottom?: number
}
type MenuView = 'root' | 'tools'

/**
 * Add-section control when Content Tools are enabled: Content (empty section)
 * or Tool (pick from catalog → new section seeded with that tool).
 * Portals the menu so sticky block chrome does not clip it.
 */
export function AddSectionDropdown({
  variant,
  label,
  onAddContent,
  onAddTool,
  tools,
  disabled,
  atMaxInstances,
  loading,
  emptyCatalog,
  settingsHref,
}: AddSectionDropdownProps) {
  const { t } = useTranslation('contentTools')
  const [open, setOpen] = useState(false)
  const [view, setView] = useState<MenuView>('root')
  const [menuPos, setMenuPos] = useState<MenuPos | null>(null)
  const rootRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const buttonId = useId()
  const menuId = useId()

  const close = () => {
    setOpen(false)
    setView('root')
  }

  const updateMenuPos = () => {
    const btn = buttonRef.current
    if (!btn) return
    const rect = btn.getBoundingClientRect()
    const width = variant === 'row' ? Math.max(rect.width, 288) : 288
    const left = Math.min(rect.left, window.innerWidth - width - 8)
    // Bottom insert row opens upward so the menu stays on-screen.
    if (variant === 'row') {
      setMenuPos({
        left: Math.max(8, left),
        bottom: window.innerHeight - rect.top + 4,
        width,
      })
      return
    }
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
  }, [open, view, variant])

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node
      if (rootRef.current?.contains(target)) return
      if (menuRef.current?.contains(target)) return
      close()
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (view === 'tools') {
          setView('root')
          return
        }
        close()
      }
    }
    const onReposition = () => updateMenuPos()
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    window.addEventListener('resize', onReposition)
    window.addEventListener('scroll', onReposition, true)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
      window.removeEventListener('resize', onReposition)
      window.removeEventListener('scroll', onReposition, true)
    }
  }, [open, view])

  const toolsDisabled = Boolean(atMaxInstances)
  const toolsDisabledReason = atMaxInstances
    ? t('contentTools.authoring.maxInstancesReached')
    : undefined

  const dividerVisible = open
    ? 'pointer-events-auto opacity-100'
    : 'pointer-events-none opacity-0 group-hover/divider:pointer-events-auto group-hover/divider:opacity-100 group-focus-within/divider:pointer-events-auto group-focus-within/divider:opacity-100'
  const triggerClass =
    variant === 'divider'
      ? `relative z-10 inline-flex h-7 items-center gap-1.5 rounded-full border border-slate-200 bg-white px-3 text-xs font-medium text-slate-600 shadow-sm hover:text-slate-900 disabled:cursor-not-allowed motion-safe:transition-opacity dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:text-neutral-50 ${dividerVisible}`
      : 'flex w-full items-center justify-center gap-2 rounded-xl border border-dashed border-slate-300 px-4 py-4 text-sm font-medium text-slate-600 motion-safe:transition-[background-color,color,border-color] hover:border-indigo-400 hover:bg-white hover:text-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:border-indigo-500 dark:hover:bg-neutral-900 dark:hover:text-indigo-400'

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
              ...(menuPos.bottom != null
                ? { bottom: menuPos.bottom }
                : { top: menuPos.top ?? 0 }),
              width: menuPos.width,
              zIndex: 80,
            }}
            className="overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-900"
          >
            {view === 'root' ? (
              <div className="p-1">
                <button
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    close()
                    onAddContent()
                  }}
                  className="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-start text-sm font-medium text-slate-800 hover:bg-slate-100 dark:text-neutral-100 dark:hover:bg-neutral-800"
                >
                  <FileText className="h-4 w-4 shrink-0 opacity-70" aria-hidden />
                  {t('contentTools.authoring.addSectionContent')}
                </button>
                <button
                  type="button"
                  role="menuitem"
                  aria-haspopup="menu"
                  title={toolsDisabled ? toolsDisabledReason : undefined}
                  onClick={() => setView('tools')}
                  className="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-start text-sm font-medium text-slate-800 hover:bg-slate-100 dark:text-neutral-100 dark:hover:bg-neutral-800"
                >
                  <Wrench className="h-4 w-4 shrink-0 opacity-70" aria-hidden />
                  <span className="flex-1">{t('contentTools.authoring.addSectionTool')}</span>
                  <ChevronDown className="h-3.5 w-3.5 -rotate-90 opacity-60" aria-hidden />
                </button>
              </div>
            ) : emptyCatalog && !loading ? (
              <div className="space-y-2 p-3 text-xs text-slate-600 dark:text-neutral-300">
                <button
                  type="button"
                  onClick={() => setView('root')}
                  className="mb-1 inline-flex items-center gap-1 text-xs font-medium text-slate-700 hover:text-slate-900 dark:text-neutral-200 dark:hover:text-neutral-50"
                >
                  <ChevronLeft className="h-3.5 w-3.5" aria-hidden />
                  {t('contentTools.authoring.addSectionBack')}
                </button>
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
              <div className="flex flex-col">
                <div className="flex items-center gap-1 border-b border-slate-200 px-2 py-1.5 dark:border-neutral-700">
                  <button
                    type="button"
                    onClick={() => setView('root')}
                    className="inline-flex items-center gap-1 rounded px-1.5 py-1 text-xs font-medium text-slate-700 hover:bg-slate-100 dark:text-neutral-200 dark:hover:bg-neutral-800"
                  >
                    <ChevronLeft className="h-3.5 w-3.5" aria-hidden />
                    {t('contentTools.authoring.addSectionBack')}
                  </button>
                  <span className="text-xs font-semibold text-slate-500 dark:text-neutral-400">
                    {t('contentTools.authoring.addSectionPickTool')}
                  </span>
                </div>
                <ToolPaletteList
                  tools={tools}
                  loading={loading}
                  disabled={toolsDisabled}
                  disabledReason={toolsDisabledReason}
                  onSelect={(toolId) => {
                    if (toolsDisabled) return
                    close()
                    onAddTool(toolId)
                  }}
                />
              </div>
            )}
          </div>,
          document.body,
        )
      : null

  return (
    <div ref={rootRef} className={variant === 'divider' ? 'relative z-10' : 'relative w-full'}>
      <button
        ref={buttonRef}
        type="button"
        id={buttonId}
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        onMouseDown={(e) => e.preventDefault()}
        onClick={() => {
          if (open) {
            close()
            return
          }
          setView('root')
          setOpen(true)
        }}
        className={triggerClass}
      >
        <Plus className={variant === 'divider' ? 'h-3.5 w-3.5' : 'h-4 w-4'} aria-hidden />
        {label}
        <ChevronDown
          className={variant === 'divider' ? 'h-3 w-3 opacity-70' : 'h-4 w-4 opacity-70'}
          aria-hidden
        />
      </button>
      {menu}
    </div>
  )
}

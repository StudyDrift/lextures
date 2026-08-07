import { ChevronDown, Pencil, Sparkles } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { handleMenuKeyDown, focusFirstMenuitem } from '../../lib/a11y/menu-keyboard'

type AssignmentPageActionsMenuProps = {
  disabled: boolean
  onEdit: () => void
  showGradingAgent?: boolean
  onGradingAgent?: () => void
  reviewCount?: number
}

export function AssignmentPageActionsMenu({
  disabled,
  onEdit,
  showGradingAgent = false,
  onGradingAgent,
  reviewCount = 0,
}: AssignmentPageActionsMenuProps) {
  const { t } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const menuTypeaheadRef = useRef({ buffer: '', at: 0 })
  const menuListRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    focusFirstMenuitem(menuListRef.current)
  }, [open])
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex h-10 items-center gap-2 rounded-xl border border-border-strong bg-surface-raised px-4 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
      >
        Actions
        <ChevronDown className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`} aria-hidden />
      </button>
      {open && (
        <div
          id={menuId}
          ref={menuListRef} role="menu"
          aria-label="Assignment actions"
          className="absolute end-0 z-50 mt-1 min-w-[12rem] overflow-hidden rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg shadow-slate-900/10 dark:border-border-default dark:bg-surface-raised"
         onKeyDown={(e) => handleMenuKeyDown(e, { onClose: () => setOpen(false) }, menuTypeaheadRef.current)} tabIndex={-1}>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onEdit()
              setOpen(false)
            }}
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-fg-default transition-[background-color,color,border-color] hover:bg-surface-base dark:text-fg-default dark:hover:bg-surface-overlay"
          >
            <Pencil className="h-4 w-4 shrink-0 text-fg-muted" aria-hidden />
            Edit
          </button>
          {showGradingAgent && onGradingAgent ? (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onGradingAgent()
                setOpen(false)
              }}
              className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-fg-default transition-[background-color,color,border-color] hover:bg-surface-base dark:text-fg-default dark:hover:bg-surface-overlay"
            >
              <Sparkles className="h-4 w-4 shrink-0 text-fg-muted" aria-hidden />
              <span className="flex flex-1 items-center justify-between gap-2">
                <span>{t('gradingAgent.button')}</span>
                {reviewCount > 0 ? (
                  <span
                    className="rounded-full bg-amber-500 px-2 py-0.5 text-xs font-semibold text-white"
                    aria-live="polite"
                  >
                    {t('gradingAgent.review.inbox.countShort', { count: reviewCount })}
                  </span>
                ) : null}
              </span>
            </button>
          ) : null}
        </div>
      )}
    </div>
  )
}
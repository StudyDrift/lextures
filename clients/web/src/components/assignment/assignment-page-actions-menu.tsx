import { ChevronDown, Pencil, Sparkles } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

type AssignmentPageActionsMenuProps = {
  disabled: boolean
  onEdit: () => void
  showGradingAgent?: boolean
  onGradingAgent?: () => void
}

export function AssignmentPageActionsMenu({
  disabled,
  onEdit,
  showGradingAgent = false,
  onGradingAgent,
}: AssignmentPageActionsMenuProps) {
  const { t } = useTranslation('common')
  const [open, setOpen] = useState(false)
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
        className="inline-flex h-10 items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-800 shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
      >
        Actions
        <ChevronDown className={`h-4 w-4 shrink-0 transition ${open ? 'rotate-180' : ''}`} aria-hidden />
      </button>
      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Assignment actions"
          className="absolute end-0 z-50 mt-1 min-w-[12rem] overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-900"
        >
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onEdit()
              setOpen(false)
            }}
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-slate-800 transition hover:bg-slate-50 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            <Pencil className="h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
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
              className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-slate-800 transition hover:bg-slate-50 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              <Sparkles className="h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
              {t('gradingAgent.button')}
            </button>
          ) : null}
        </div>
      )}
    </div>
  )
}
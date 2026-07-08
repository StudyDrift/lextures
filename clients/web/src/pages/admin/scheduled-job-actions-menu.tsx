import { useEffect, useId, useRef, useState } from 'react'
import { ChevronDown, Clock, History, Play, Power } from 'lucide-react'

type ScheduledJobActionsMenuProps = {
  disabled?: boolean
  enabled: boolean
  historyOpen: boolean
  onToggleEnabled: () => void
  onTrigger: () => void
  onToggleHistory: () => void
}

export function ScheduledJobActionsMenu({
  disabled,
  enabled,
  historyOpen,
  onToggleEnabled,
  onTrigger,
  onToggleHistory,
}: ScheduledJobActionsMenuProps) {
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
    <div ref={rootRef} className="relative inline-block text-start">
      <button
        type="button"
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-2.5 py-1.5 text-sm font-semibold text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:border-slate-300 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800"
      >
        Actions
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open ? (
        <div
          id={menuId}
          role="menu"
          aria-label="Scheduled job actions"
          className="absolute end-0 z-50 mt-1 min-w-[12rem] overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40"
        >
          <button
            type="button"
            role="menuitem"
            disabled={disabled}
            onClick={() => {
              onToggleEnabled()
              setOpen(false)
            }}
            className={`flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium transition-[background-color,color,border-color] hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:hover:bg-neutral-700/80 ${
              enabled
                ? 'text-rose-700 dark:text-rose-300'
                : 'text-slate-800 dark:text-neutral-100'
            }`}
          >
            <Power className="h-4 w-4 shrink-0" aria-hidden />
            {disabled ? 'Saving…' : enabled ? 'Disable' : 'Enable'}
          </button>
          <button
            type="button"
            role="menuitem"
            disabled={disabled}
            onClick={() => {
              onTrigger()
              setOpen(false)
            }}
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-slate-800 transition-[background-color,color,border-color] hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:text-neutral-100 dark:hover:bg-neutral-700/80"
          >
            <Play className="h-4 w-4 shrink-0" aria-hidden />
            Trigger now
          </button>
          <button
            type="button"
            role="menuitem"
            aria-expanded={historyOpen}
            onClick={() => {
              onToggleHistory()
              setOpen(false)
            }}
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-slate-800 transition-[background-color,color,border-color] hover:bg-slate-50 dark:text-neutral-100 dark:hover:bg-neutral-700/80"
          >
            {historyOpen ? (
              <Clock className="h-4 w-4 shrink-0" aria-hidden />
            ) : (
              <History className="h-4 w-4 shrink-0" aria-hidden />
            )}
            {historyOpen ? 'Hide history' : 'View history'}
          </button>
        </div>
      ) : null}
    </div>
  )
}

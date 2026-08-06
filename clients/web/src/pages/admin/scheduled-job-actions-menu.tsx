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
        className="inline-flex items-center gap-1.5 rounded-xl border border-border-default bg-surface-raised px-2.5 py-1.5 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:border-border-strong hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:border-border-default dark:hover:bg-surface-overlay"
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
          className="absolute end-0 z-50 mt-1 min-w-[12rem] overflow-hidden rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg shadow-slate-900/10 dark:border-border-default dark:bg-surface-overlay dark:shadow-black/40"
        >
          <button
            type="button"
            role="menuitem"
            disabled={disabled}
            onClick={() => {
              onToggleEnabled()
              setOpen(false)
            }}
            className={`flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 dark:hover:bg-neutral-700/80 ${ enabled ? 'text-rose-700 dark:text-rose-300' : 'text-fg-default' }`}
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
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-fg-default transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 dark:text-fg-default dark:hover:bg-neutral-700/80"
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
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-fg-default transition-[background-color,color,border-color] hover:bg-surface-base dark:text-fg-default dark:hover:bg-neutral-700/80"
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

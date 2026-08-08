import { useEffect, useId, useRef, useState } from 'react'
import { Check, ChevronDown, GraduationCap, UserPlus, Users } from 'lucide-react'
import { handleMenuKeyDown, focusFirstMenuitem } from '../../lib/a11y/menu-keyboard'

type EnrollmentsActionsMenuProps = {
  disabled?: boolean
  canEnrollSelfAsStudent: boolean
  onEnrollAsStudent: () => void
  enrollAsStudentBusy: boolean
  onAddEnrollment: () => void
  /** When true, shows a checked "Enable groups" state (still clickable as no-op or refresh). */
  groupsEnabled: boolean
  canToggleGroups: boolean
  onEnableGroups: () => void
  enableGroupsBusy: boolean
}

export function EnrollmentsActionsMenu({
  disabled,
  canEnrollSelfAsStudent,
  onEnrollAsStudent,
  enrollAsStudentBusy,
  onAddEnrollment,
  groupsEnabled,
  canToggleGroups,
  onEnableGroups,
  enableGroupsBusy,
}: EnrollmentsActionsMenuProps) {
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
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  return (
    <div ref={rootRef} className="relative inline-block w-full text-start sm:w-auto">
      <button
        type="button"
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => {
          if (disabled) return
          setOpen((o) => !o)
        }}
        className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-accent-solid px-2 py-1.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 sm:w-auto sm:justify-start sm:px-3 sm:py-2"
      >
        <span>Actions</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          ref={menuListRef} role="menu"
          aria-label="Enrollments actions"
          className="absolute end-0 z-50 mt-1 min-w-[14rem] overflow-hidden rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg shadow-slate-900/10 dark:border-border-default dark:bg-surface-overlay dark:shadow-black/40"
         onKeyDown={(e) => handleMenuKeyDown(e, { onClose: () => setOpen(false) }, menuTypeaheadRef.current)} tabIndex={-1}>
          {canEnrollSelfAsStudent ? (
            <button
              type="button"
              role="menuitem"
              disabled={disabled || enrollAsStudentBusy}
              onClick={() => {
                onEnrollAsStudent()
                setOpen(false)
              }}
              className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-fg-default transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 dark:text-fg-default dark:hover:bg-neutral-700/80"
            >
              <GraduationCap className="h-4 w-4 shrink-0" aria-hidden />
              {enrollAsStudentBusy ? 'Adding…' : 'Add Test Student seat'}
            </button>
          ) : null}
          <button
            type="button"
            role="menuitem"
            disabled={disabled}
            onClick={() => {
              onAddEnrollment()
              setOpen(false)
            }}
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-fg-default transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 dark:text-fg-default dark:hover:bg-neutral-700/80"
          >
            <UserPlus className="h-4 w-4 shrink-0" aria-hidden />
            Add enrollment
          </button>
          {canToggleGroups ? (
            <>
              <div className="my-1 border-t border-border-subtle dark:border-border-default" role="separator" />
              <button
                type="button"
                role="menuitemcheckbox"
                aria-checked={groupsEnabled}
                disabled={disabled || enableGroupsBusy || groupsEnabled}
                onClick={() => {
                  if (groupsEnabled) return
                  onEnableGroups()
                  setOpen(false)
                }}
                className="flex w-full items-start gap-2 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 dark:hover:bg-neutral-700/80"
              >
                <span
                  className={`mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border ${ groupsEnabled ? 'border-indigo-600 bg-accent-solid text-white dark:border-indigo-500 dark:bg-indigo-500' : 'border-border-strong bg-surface-raised dark:border-neutral-500 dark:bg-surface-overlay' }`}
                  aria-hidden
                >
                  {groupsEnabled ? <Check className="h-3 w-3" strokeWidth={3} /> : null}
                </span>
                <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <span className="inline-flex items-center gap-1.5 font-semibold text-slate-950 dark:text-fg-default">
                    <Users className="h-4 w-4 shrink-0" aria-hidden />
                    Enable groups
                  </span>
                  <span className="text-xs text-fg-muted">
                    {groupsEnabled
                      ? 'Group sets and the Groups tab are on.'
                      : 'Sort students into named groups (an empty default set is created for you).'}
                  </span>
                </span>
              </button>
            </>
          ) : null}
        </div>
      )}
    </div>
  )
}

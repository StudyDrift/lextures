import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { CommandPaletteContext } from './command-palette-context'
import { CommandPaletteDialog } from './command-palette-dialog'

export function CommandPaletteProvider({ children }: { children: ReactNode }) {
  const [isOpen, setOpen] = useState(false)
  const triggerRef = useRef<HTMLElement | null>(null)

  const open = useCallback(() => {
    triggerRef.current = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null
    setOpen(true)
  }, [])

  const close = useCallback(() => {
    setOpen(false)
  }, [])

  const toggle = useCallback(() => {
    setOpen((v) => {
      if (!v) {
        triggerRef.current = document.activeElement instanceof HTMLElement
          ? document.activeElement
          : null
      }
      return !v
    })
  }, [])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.key !== 'k') return
      const t = e.target as HTMLElement | null
      if (t?.closest?.('[data-no-command-palette]')) return
      e.preventDefault()
      setOpen((v) => {
        if (!v) {
          triggerRef.current = document.activeElement instanceof HTMLElement
            ? document.activeElement
            : null
        }
        return !v
      })
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  // Return focus to the trigger element when the palette closes.
  useEffect(() => {
    if (!isOpen && triggerRef.current) {
      const el = triggerRef.current
      // Defer so the dialog has fully unmounted before shifting focus.
      const t = window.setTimeout(() => el.focus(), 0)
      return () => window.clearTimeout(t)
    }
  }, [isOpen])

  const value = useMemo(
    () => ({ open, close, toggle, isOpen, triggerRef }),
    [open, close, toggle, isOpen],
  )

  return (
    <CommandPaletteContext.Provider value={value}>
      {children}
      {isOpen && <CommandPaletteDialog />}
    </CommandPaletteContext.Provider>
  )
}

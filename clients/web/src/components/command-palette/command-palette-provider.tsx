import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { useOverlayPresence } from '../../lib/use-overlay-presence'
import { CommandPaletteContext } from './command-palette-context'
import { CommandPaletteDialog } from './command-palette-dialog'

export function CommandPaletteProvider({ children }: { children: ReactNode }) {
  const [isOpen, setOpen] = useState(false)
  const triggerRef = useRef<HTMLElement | null>(null)
  const { ffMotionOverlays } = usePlatformFeatures()
  const presence = useOverlayPresence({
    open: isOpen,
    kind: 'menu',
    enabled: ffMotionOverlays !== false,
  })

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

  // Return focus to the trigger element when the palette closes (exit-start).
  useEffect(() => {
    if (!isOpen && triggerRef.current) {
      const el = triggerRef.current
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
      {presence.mounted ? (
        <CommandPaletteDialog open={isOpen} presence={presence} />
      ) : null}
    </CommandPaletteContext.Provider>
  )
}

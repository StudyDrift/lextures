import { createContext, type RefObject } from 'react'

export type CommandPaletteContextValue = {
  open: () => void
  close: () => void
  toggle: () => void
  isOpen: boolean
  /** The element that had focus immediately before the palette opened; restored on close. */
  triggerRef: RefObject<HTMLElement | null>
}

export const CommandPaletteContext = createContext<CommandPaletteContextValue | null>(null)

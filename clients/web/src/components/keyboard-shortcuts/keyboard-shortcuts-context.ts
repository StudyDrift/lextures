import { createContext } from 'react'

export type KeyboardShortcutsContextValue = {
  openSheet: () => void
  closeSheet: () => void
}

export const KeyboardShortcutsContext = createContext<KeyboardShortcutsContextValue | null>(null)

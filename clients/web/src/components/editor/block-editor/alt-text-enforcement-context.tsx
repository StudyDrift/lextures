/* eslint-disable react-refresh/only-export-components -- context module exports provider + hook */
import { createContext, useContext, type ReactNode } from 'react'

export type AltTextEnforcementContextValue = {
  enabled: boolean
  hardBlock: boolean
  courseCode?: string
  onAiUnavailable?: () => void
}

const AltTextEnforcementContext = createContext<AltTextEnforcementContextValue>({
  enabled: false,
  hardBlock: false,
})

export function AltTextEnforcementProvider({
  value,
  children,
}: {
  value: AltTextEnforcementContextValue
  children: ReactNode
}) {
  return (
    <AltTextEnforcementContext.Provider value={value}>{children}</AltTextEnforcementContext.Provider>
  )
}

export function useAltTextEnforcement(): AltTextEnforcementContextValue {
  return useContext(AltTextEnforcementContext)
}

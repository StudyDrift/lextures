import { createContext, useContext, type ReactNode } from 'react'
import type { SsrData } from './ssr-data'

const SsrContext = createContext<SsrData>({})

export function SsrDataProvider({ data, children }: { data: SsrData; children: ReactNode }) {
  return <SsrContext.Provider value={data}>{children}</SsrContext.Provider>
}

export function useSsrData(): SsrData {
  return useContext(SsrContext)
}

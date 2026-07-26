/* eslint-disable react-refresh/only-export-components -- provider + announce hook */
import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'

type AnnounceOptions = {
  politeness?: 'polite' | 'assertive'
}

type LiveRegionContextValue = {
  announce: (message: string, options?: AnnounceOptions) => void
}

const LiveRegionContext = createContext<LiveRegionContextValue | null>(null)

export function LiveRegionProvider({ children }: { children: ReactNode }) {
  const [polite, setPolite] = useState('')
  const [assertive, setAssertive] = useState('')
  const politeClearRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const assertiveClearRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (politeClearRef.current) clearTimeout(politeClearRef.current)
      if (assertiveClearRef.current) clearTimeout(assertiveClearRef.current)
    }
  }, [])

  const value: LiveRegionContextValue = {
    announce(message, options) {
      const text = message.trim()
      if (!text) return
      const politeness = options?.politeness ?? 'polite'
      if (politeness === 'assertive') {
        if (assertiveClearRef.current) clearTimeout(assertiveClearRef.current)
        setAssertive('')
        // Force a DOM change so repeated identical messages are re-announced.
        requestAnimationFrame(() => {
          setAssertive(text)
          assertiveClearRef.current = setTimeout(() => setAssertive(''), 1500)
        })
        return
      }
      if (politeClearRef.current) clearTimeout(politeClearRef.current)
      setPolite('')
      requestAnimationFrame(() => {
        setPolite(text)
        politeClearRef.current = setTimeout(() => setPolite(''), 1500)
      })
    },
  }

  return (
    <LiveRegionContext.Provider value={value}>
      {children}
      <div aria-live="polite" aria-atomic="true" role="status" className="sr-only">
        {polite}
      </div>
      <div aria-live="assertive" aria-atomic="true" role="alert" className="sr-only">
        {assertive}
      </div>
    </LiveRegionContext.Provider>
  )
}

export function useAnnounce(): (message: string, options?: AnnounceOptions) => void {
  const ctx = useContext(LiveRegionContext)
  if (!ctx) {
    return () => {}
  }
  return ctx.announce
}

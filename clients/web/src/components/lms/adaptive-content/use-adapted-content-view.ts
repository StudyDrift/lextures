import { useCallback, useState } from 'react'

/**
 * Client-side toggle between adapted markdown and original base (AC.6).
 * Prefer shipping original alongside adapted so the toggle needs no extra round-trip.
 */
export function useAdaptedContentView(
  adaptedMarkdown: string | undefined,
  originalMarkdown: string,
  isAdapted: boolean,
) {
  const hasAdapted = Boolean(isAdapted && adaptedMarkdown?.trim())
  const [showingOriginal, setShowingOriginal] = useState(false)

  const displayMarkdown =
    hasAdapted && !showingOriginal ? (adaptedMarkdown as string) : originalMarkdown

  const showOriginal = useCallback(() => setShowingOriginal(true), [])
  const showAdapted = useCallback(() => setShowingOriginal(false), [])
  const toggle = useCallback(() => setShowingOriginal((v) => !v), [])

  return {
    hasAdapted,
    showingOriginal,
    displayMarkdown,
    showOriginal,
    showAdapted,
    toggle,
  }
}

import { useState } from 'react'

export function useSimplifiedContentView(
  simplifiedMarkdown: string | undefined,
  originalMarkdown: string,
) {
  const hasSimplified = Boolean(simplifiedMarkdown?.trim())
  const [showingOriginal, setShowingOriginal] = useState(false)
  const displayMarkdown =
    hasSimplified && !showingOriginal ? (simplifiedMarkdown as string) : originalMarkdown
  return {
    hasSimplified,
    showingOriginal,
    displayMarkdown,
    showOriginal: () => setShowingOriginal(true),
    showSimplified: () => setShowingOriginal(false),
  }
}

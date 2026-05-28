export type SimplifiedContentBannerProps = {
  targetFkgl?: number
  originalMarkdown: string
  simplifiedMarkdown: string
  onShowOriginal: () => void
  onShowSimplified: () => void
  showingOriginal: boolean
}

export function SimplifiedContentBanner({
  targetFkgl,
  onShowOriginal,
  onShowSimplified,
  showingOriginal,
}: SimplifiedContentBannerProps) {
  const grade = targetFkgl ?? 4
  return (
    <div
      className="mb-4 rounded-lg border border-sky-200 bg-sky-50 px-4 py-3 text-sm text-sky-950 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-100"
      role="status"
    >
      <p className="font-medium">Simplified for your reading level (Grade {grade})</p>
      <p className="mt-1 text-sky-800 dark:text-sky-200">
        This version uses shorter sentences and simpler words. The original is always available.
      </p>
      <button
        type="button"
        className="mt-2 text-sm font-medium underline underline-offset-2"
        onClick={showingOriginal ? onShowSimplified : onShowOriginal}
      >
        {showingOriginal ? 'View simplified version' : 'View original'}
      </button>
    </div>
  )
}

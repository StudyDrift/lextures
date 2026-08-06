import { BookOpen } from 'lucide-react'
import { useReadingShellFocus } from './reading-shell-focus-context'

/** Hides the course shell side nav for long-form reading (exit from the slim top bar). */
export function ReadingFocusToggle() {
  const { readingFocus, setReadingFocus } = useReadingShellFocus()
  if (readingFocus) return null
  return (
    <button
      type="button"
      onClick={() => setReadingFocus(true)}
      className="inline-flex items-center gap-2 rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
    >
      <BookOpen className="h-4 w-4 text-accent-fg" aria-hidden />
      Reading focus
    </button>
  )
}

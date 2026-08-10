import { BookOpen, PanelLeft } from 'lucide-react'
import { useReadingShellFocus } from './reading-shell-focus-context'

export function ReadingFocusTopBar() {
  const { setReadingFocus } = useReadingShellFocus()
  return (
    <header
      className="lms-chrome flex h-14 shrink-0 items-center justify-between gap-3 border-b border-border-default bg-surface-raised px-3 shadow-sm print:hidden sm:px-5 dark:border-border-default dark:bg-surface-raised"
      data-reading-focus-bar
      data-lx-sticky-chrome
    >
      <div className="flex min-w-0 items-center gap-2 text-sm font-medium text-fg-default">
        <BookOpen className="h-4 w-4 shrink-0 text-accent-fg" aria-hidden />
        <span className="truncate">Reading focus</span>
      </div>
      <button
        type="button"
        onClick={() => setReadingFocus(false)}
        className="inline-flex shrink-0 items-center gap-2 rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
      >
        <PanelLeft className="h-4 w-4" aria-hidden />
        Show navigation
      </button>
    </header>
  )
}

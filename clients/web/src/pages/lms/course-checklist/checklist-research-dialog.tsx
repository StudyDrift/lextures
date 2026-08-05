import { useEffect, useId, useRef } from 'react'
import { Link } from 'react-router-dom'
import { X } from 'lucide-react'
import { ChecklistResearchBody } from '../../../components/checklist/checklist-research-body'
import { courseDesignResearchHref } from '../../../lib/checklist-research-anchors'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'

type Props = {
  open: boolean
  onClose: () => void
  /** Source chip label (e.g. "OSCQR 7") — scrolls the standards index to that anchor. */
  sourceLabel?: string | null
}

/**
 * In-place research viewer so source chips do not leave the checklist.
 * Full-page deep links use the same anchors at /help/course-checklist/research#src-…
 */
export function ChecklistResearchDialog({ open, onClose, sourceLabel }: Props) {
  const titleId = useId()
  const closeRef = useRef<HTMLButtonElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const title = sourceLabel
    ? `${sourceLabel} — ${courseChecklistI18n.sourcesResearchLink}`
    : courseChecklistI18n.sourcesResearchLink
  const fullPageHref = courseDesignResearchHref(sourceLabel)

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-0 sm:items-center sm:p-4"
      onClick={onClose}
    >
      <div
        className="flex max-h-[92vh] w-full max-w-3xl flex-col overflow-hidden rounded-t-xl border border-slate-200 bg-white shadow-xl sm:rounded-xl dark:border-neutral-700 dark:bg-neutral-900"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex shrink-0 items-start justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
          <div className="min-w-0">
            <h2 id={titleId} className="text-base font-semibold text-slate-900 dark:text-neutral-50">
              {title}
            </h2>
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              Rule-to-standard mapping for Quality Matters, OSCQR, NSQ, UDL, and WCAG.
            </p>
          </div>
          <button
            ref={closeRef}
            type="button"
            className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
            aria-label={courseChecklistI18n.helpClose}
            onClick={onClose}
          >
            <X className="h-4 w-4" aria-hidden />
          </button>
        </div>

        <div ref={scrollRef} className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
          <ChecklistResearchBody focusSource={sourceLabel} scrollRootRef={scrollRef} />
        </div>

        <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-t border-slate-200 px-5 py-3 dark:border-neutral-700">
          <Link
            to={fullPageHref}
            className="text-xs font-semibold text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300"
            onClick={onClose}
          >
            {courseChecklistI18n.sourcesOpenFullPage}
          </Link>
          <button
            type="button"
            className="inline-flex min-h-11 items-center rounded-lg bg-slate-900 px-3 text-sm font-medium text-white hover:bg-slate-800 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-white"
            onClick={onClose}
          >
            {courseChecklistI18n.helpClose}
          </button>
        </div>
      </div>
    </div>
  )
}

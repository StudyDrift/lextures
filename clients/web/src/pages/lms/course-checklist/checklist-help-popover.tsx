import { useEffect, useId, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { X } from 'lucide-react'
import {
  helpSupportUrl,
  resolveChecklistHelp,
} from '../../../lib/checklist-help'
import { courseDesignResearchHref } from '../../../lib/checklist-research-anchors'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import { emitChecklistTelemetry } from '../../../lib/checklist-telemetry'
import { ChecklistResearchDialog } from './checklist-research-dialog'

type Props = {
  helpRef: string | null | undefined
  itemId: string
  open: boolean
  onClose: () => void
  sources?: string[]
}

export function ChecklistHelpPopover({ helpRef, itemId, open, onClose, sources }: Props) {
  const titleId = useId()
  const closeRef = useRef<HTMLButtonElement>(null)
  const entry = resolveChecklistHelp(helpRef)
  const supportHref = helpSupportUrl(helpRef)
  const [researchOpen, setResearchOpen] = useState(false)
  const [researchSource, setResearchSource] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    emitChecklistTelemetry('checklist_help_opened', { itemId })
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !researchOpen) {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, itemId, onClose, researchOpen])

  useEffect(() => {
    if (!open) {
      setResearchOpen(false)
      setResearchSource(null)
    }
  }, [open])

  if (!open || !entry) return null

  return (
    <>
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="fixed inset-0 z-40 flex items-end justify-center bg-black/40 p-4 sm:items-center"
        onClick={onClose}
      >
        <div
          className="max-h-[85vh] w-full max-w-lg overflow-y-auto rounded-xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-start justify-between gap-3">
            <h2 id={titleId} className="text-base font-semibold text-slate-900 dark:text-neutral-50">
              {entry.title}
            </h2>
            <button
              ref={closeRef}
              type="button"
              className="inline-flex h-11 w-11 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
              aria-label={courseChecklistI18n.helpClose}
              onClick={onClose}
            >
              <X className="h-4 w-4" aria-hidden />
            </button>
          </div>

          <section className="mt-4 space-y-3 text-sm text-slate-700 dark:text-neutral-300">
            <div>
              <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                {courseChecklistI18n.helpWhat}
              </h3>
              <p className="mt-1">{entry.what}</p>
            </div>
            <div>
              <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                {courseChecklistI18n.helpWhy}
              </h3>
              <p className="mt-1">{entry.why}</p>
            </div>
            <div>
              <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                {courseChecklistI18n.helpHow}
              </h3>
              <p className="mt-1">{entry.how}</p>
            </div>
            <div>
              <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                {courseChecklistI18n.helpWhenDismiss}
              </h3>
              <p className="mt-1">{entry.whenToDismiss}</p>
            </div>
          </section>

          {(sources?.length || entry.sources?.length) ? (
            <ul className="mt-4 flex flex-wrap gap-1.5">
              {(sources?.length ? sources : entry.sources).map((src) => (
                <li key={src}>
                  <a
                    href={courseDesignResearchHref(src)}
                    className="rounded bg-slate-100 px-1.5 py-0.5 text-[11px] font-medium text-slate-700 underline-offset-2 hover:underline dark:bg-neutral-800 dark:text-neutral-300"
                    onClick={(e) => {
                      e.preventDefault()
                      setResearchSource(src)
                      setResearchOpen(true)
                    }}
                  >
                    {src}
                  </a>
                </li>
              ))}
            </ul>
          ) : null}

          {supportHref ? (
            <p className="mt-4 text-xs">
              <Link
                to={supportHref}
                className="font-semibold text-amber-800 underline-offset-2 hover:underline dark:text-amber-300"
              >
                {courseChecklistI18n.helpSupportLink}
              </Link>
            </p>
          ) : null}
        </div>
      </div>

      <ChecklistResearchDialog
        open={researchOpen}
        sourceLabel={researchSource}
        onClose={() => {
          setResearchOpen(false)
          setResearchSource(null)
        }}
      />
    </>
  )
}

import { useEffect, useRef, useState } from 'react'
import type { SubmissionAnnotationApi } from '../../lib/courses-api'
import { parseTextAnchor } from '../../lib/text-anchor'

export type AnnotationCommentPanelProps = {
  annotations: SubmissionAnnotationApi[]
  selectedId: string | null
  onSelect: (id: string) => void
  readOnly?: boolean
  onDelete?: (id: string) => void
  /** Save (or clear) the comment text on an annotation. Omitted when read-only. */
  onUpdateBody?: (annotation: SubmissionAnnotationApi, body: string) => void
}

const TOOL_LABELS: Record<string, string> = {
  highlight: 'Highlight',
  draw: 'Drawing',
  pin: 'Pin',
  text: 'Text box',
  anchor: 'Highlight',
}

/** Short quoted passage for a text-anchor annotation, or null for geometric tools. */
function anchorQuote(a: SubmissionAnnotationApi): string | null {
  if (a.toolType !== 'anchor') return null
  const anchor = parseTextAnchor(a.coordsJson)
  const quote = anchor?.quote?.trim()
  return quote ? quote : null
}

/** Anchor highlights aren't paginated, so hide the "Page N" prefix for them. */
function annotationLabel(a: SubmissionAnnotationApi): string {
  const tool = TOOL_LABELS[a.toolType] ?? a.toolType
  return a.toolType === 'anchor' ? tool : `Page ${a.page} · ${tool}`
}

function CommentEditor({
  annotation,
  onUpdateBody,
  autoFocus,
}: {
  annotation: SubmissionAnnotationApi
  onUpdateBody: (annotation: SubmissionAnnotationApi, body: string) => void
  autoFocus: boolean
}) {
  const [draft, setDraft] = useState(annotation.body ?? '')
  const ref = useRef<HTMLTextAreaElement | null>(null)

  // Resync when switching between annotations or after a save round-trip.
  useEffect(() => {
    setDraft(annotation.body ?? '')
  }, [annotation.id, annotation.body])

  useEffect(() => {
    if (autoFocus) ref.current?.focus()
  }, [autoFocus])

  const dirty = draft.trim() !== (annotation.body ?? '').trim()

  return (
    <div className="mt-2 space-y-1">
      <textarea
        ref={ref}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        rows={2}
        placeholder="Add a comment…"
        className="w-full rounded-md border border-border-strong bg-surface-raised px-2 py-1.5 text-xs text-fg-default shadow-sm focus:border-indigo-500 focus:outline-none dark:border-border-default dark:bg-surface-base"
      />
      {dirty ? (
        <div className="flex gap-1">
          <button
            type="button"
            className="rounded-md bg-accent-solid px-2 py-1 text-[11px] font-semibold text-white hover:bg-indigo-500"
            onClick={() => onUpdateBody(annotation, draft)}
          >
            Save comment
          </button>
          <button
            type="button"
            className="rounded-md border border-border-strong px-2 py-1 text-[11px] font-medium text-fg-muted hover:bg-surface-base dark:border-border-default dark:text-fg-muted dark:hover:bg-surface-raised"
            onClick={() => setDraft(annotation.body ?? '')}
          >
            Cancel
          </button>
        </div>
      ) : null}
    </div>
  )
}

export function AnnotationCommentPanel({
  annotations,
  selectedId,
  onSelect,
  readOnly,
  onDelete,
  onUpdateBody,
}: AnnotationCommentPanelProps) {
  return (
    <aside
      aria-label="Annotation comments"
      className="flex max-h-[70vh] w-full max-w-sm flex-col rounded-xl border border-border-default bg-surface-raised shadow-sm dark:border-border-default dark:bg-surface-base lg:max-h-none"
    >
      <div className="border-b border-border-default px-3 py-2 text-sm font-semibold text-fg-default dark:border-border-default dark:text-fg-default">
        Comments
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-2">
        {annotations.length === 0 ? (
          <p className="px-2 py-6 text-center text-sm text-fg-muted">
            {readOnly
              ? 'No annotations on this submission.'
              : 'No annotations yet — use the toolbar above to highlight, draw, or pin feedback.'}
          </p>
        ) : (
          <ul className="space-y-2">
            {annotations.map((a) => {
              const selected = selectedId === a.id
              return (
                <li key={a.id}>
                  <button
                    type="button"
                    onClick={() => onSelect(a.id)}
                    className={`w-full rounded-lg border px-2 py-2 text-start text-xs transition-[background-color,color,border-color] ${ selected ? 'border-indigo-500 bg-indigo-50 dark:border-indigo-400 dark:bg-indigo-950/40' : 'border-border-default hover:bg-surface-base dark:border-border-default dark:hover:bg-surface-raised' }`}
                  >
                    <div className="flex items-center gap-1.5 font-semibold text-fg-default">
                      <span
                        aria-hidden
                        className="inline-block h-3 w-3 shrink-0 rounded-sm border border-black/10"
                        style={{ backgroundColor: a.colour }}
                      />
                      {annotationLabel(a)}
                    </div>
                    {anchorQuote(a) ? (
                      <div className="mt-1 line-clamp-2 border-s-2 border-border-strong ps-2 italic text-fg-muted dark:border-border-default dark:text-fg-muted">
                        “{anchorQuote(a)}”
                      </div>
                    ) : null}
                    {readOnly || !selected ? (
                      a.body ? (
                        <div className="mt-1 line-clamp-4 whitespace-pre-wrap text-fg-muted">
                          {a.body}
                        </div>
                      ) : (
                        <div className="mt-1 italic text-fg-subtle">
                          {readOnly ? 'No comment' : 'No comment yet — select to add one'}
                        </div>
                      )
                    ) : null}
                  </button>
                  {!readOnly && selected && onUpdateBody ? (
                    <CommentEditor annotation={a} onUpdateBody={onUpdateBody} autoFocus={!a.body} />
                  ) : null}
                  {!readOnly && onDelete ? (
                    <button
                      type="button"
                      className="mt-1 w-full rounded-md border border-rose-200 px-2 py-1 text-[11px] font-medium text-rose-700 hover:bg-rose-50 dark:border-rose-900 dark:text-rose-300 dark:hover:bg-rose-950/40"
                      onClick={() => onDelete(a.id)}
                    >
                      Delete
                    </button>
                  ) : null}
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </aside>
  )
}

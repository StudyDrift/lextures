import type { RefObject } from 'react'
import { type Annotation, type Tag } from './types'

export type AnnotationsPanelProps = {
  promptId: string
  tags: Tag[]
  active: Annotation[]
  orphaned: Annotation[]
  readOnly: boolean
  noteForId: string | null
  noteDraft: string
  passageRef: RefObject<HTMLDivElement | null>
  t: (key: string, options?: Record<string, unknown>) => string
  onFocusUnit: (index: number) => void
  onEditNote: (id: string, note: string) => void
  onNoteDraftChange: (value: string) => void
  onSaveNote: (id: string) => void
  onRemove: (id: string) => void
}

export function AnnotationsPanel({
  promptId,
  tags,
  active,
  orphaned,
  readOnly,
  noteForId,
  noteDraft,
  passageRef,
  t,
  onFocusUnit,
  onEditNote,
  onNoteDraftChange,
  onSaveNote,
  onRemove,
}: AnnotationsPanelProps) {
  if (active.length === 0 && orphaned.length === 0) {
    return null
  }

  return (
    <>
      {active.length > 0 ? (
        <section
          aria-labelledby={`${promptId}-mine`}
          className="space-y-2 rounded-xl border border-slate-200 bg-white p-3 dark:border-neutral-700 dark:bg-neutral-950"
        >
          <div className="flex items-center justify-between gap-2">
            <h3
              id={`${promptId}-mine`}
              className="text-sm font-semibold text-slate-800 dark:text-neutral-100"
            >
              {t('contentTools.tools.highlight_annotate.myAnnotations')}
            </h3>
            <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
              {t('contentTools.tools.highlight_annotate.annotationCount', {
                count: active.length,
              })}
            </span>
          </div>
          <ul className="space-y-2">
            {active.map((a) => {
              const tag = tags.find((tg) => tg.id === a.tagId)
              return (
                <li
                  key={a.id}
                  className="rounded-lg border border-slate-200 bg-slate-50/80 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900/50"
                  data-testid={`ha-ann-${a.id}`}
                >
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <span
                      className="inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-semibold text-white"
                      style={{ backgroundColor: tag?.color ?? '#64748b' }}
                    >
                      {tag?.label ?? a.tagId}
                    </span>
                    {!readOnly ? (
                      <div className="flex gap-2 text-xs">
                        <button
                          type="button"
                          className="font-medium text-slate-600 underline-offset-2 hover:underline dark:text-neutral-300"
                          onClick={() => onEditNote(a.id, a.note ?? '')}
                        >
                          {t('contentTools.tools.highlight_annotate.editNote')}
                        </button>
                        <button
                          type="button"
                          className="font-medium text-rose-600 underline-offset-2 hover:underline"
                          onClick={() => onRemove(a.id)}
                        >
                          {t('contentTools.tools.highlight_annotate.delete')}
                        </button>
                      </div>
                    ) : null}
                  </div>
                  <button
                    type="button"
                    className="mt-1.5 block w-full text-start text-slate-800 underline-offset-2 hover:underline dark:text-neutral-100"
                    onClick={() => {
                      if (a.anchor.unitIndex != null) {
                        onFocusUnit(a.anchor.unitIndex)
                        passageRef.current
                          ?.querySelector<HTMLElement>(
                            `[data-unit-index="${a.anchor.unitIndex}"]`,
                          )
                          ?.focus()
                      }
                    }}
                  >
                    “{a.quote}”
                  </button>
                  {a.note ? (
                    <p className="mt-1 text-xs text-slate-600 dark:text-neutral-300">{a.note}</p>
                  ) : null}
                  {noteForId === a.id ? (
                    <div className="mt-2 space-y-2">
                      <textarea
                        className="w-full rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                        rows={2}
                        value={noteDraft}
                        onChange={(e) => onNoteDraftChange(e.target.value)}
                        aria-label={t('contentTools.tools.highlight_annotate.noteLabel')}
                      />
                      <button
                        type="button"
                        className="rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-500"
                        onClick={() => onSaveNote(a.id)}
                      >
                        {t('contentTools.tools.highlight_annotate.saveNote')}
                      </button>
                    </div>
                  ) : null}
                </li>
              )
            })}
          </ul>
        </section>
      ) : null}

      {orphaned.length > 0 ? (
        <details
          className="rounded-xl border border-amber-200 bg-amber-50 p-3 text-xs dark:border-amber-900 dark:bg-amber-950/30"
          data-testid="ha-orphaned"
        >
          <summary className="cursor-pointer font-medium text-amber-900 dark:text-amber-100">
            {t('contentTools.tools.highlight_annotate.orphanedTitle', {
              count: orphaned.length,
            })}
          </summary>
          <ul className="mt-2 space-y-1 text-amber-950 dark:text-amber-50">
            {orphaned.map((a) => (
              <li key={a.id}>
                “{a.quote}”
                {!readOnly ? (
                  <button
                    type="button"
                    className="ms-2 underline"
                    onClick={() => onRemove(a.id)}
                  >
                    {t('contentTools.tools.highlight_annotate.delete')}
                  </button>
                ) : null}
              </li>
            ))}
          </ul>
        </details>
      ) : null}
    </>
  )
}

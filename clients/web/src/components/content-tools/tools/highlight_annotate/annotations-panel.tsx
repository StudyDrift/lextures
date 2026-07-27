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
  return (
    <>
      <section aria-labelledby={`${promptId}-mine`} className="space-y-2">
        <h3
          id={`${promptId}-mine`}
          className="text-sm font-medium text-slate-800 dark:text-neutral-100"
        >
          {t('contentTools.tools.highlight_annotate.myAnnotations')}
        </h3>
        {tags.map((tag) => {
          const group = active.filter((a) => a.tagId === tag.id)
          if (!group.length) return null
          return (
            <div key={tag.id} className="space-y-1">
              <h4 className="text-xs font-semibold" style={{ color: tag.color }}>
                {tag.label}
              </h4>
              <ul className="space-y-1">
                {group.map((a) => (
                  <li
                    key={a.id}
                    className="rounded border border-slate-200 px-2 py-1.5 text-xs dark:border-neutral-700"
                    data-testid={`ha-ann-${a.id}`}
                  >
                    <button
                      type="button"
                      className="text-start underline-offset-2 hover:underline"
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
                      <p className="mt-1 text-slate-600 dark:text-neutral-300">{a.note}</p>
                    ) : null}
                    {!readOnly ? (
                      <div className="mt-1 flex gap-2">
                        <button
                          type="button"
                          className="underline"
                          onClick={() => onEditNote(a.id, a.note ?? '')}
                        >
                          {t('contentTools.tools.highlight_annotate.editNote')}
                        </button>
                        <button
                          type="button"
                          className="text-rose-600 underline"
                          onClick={() => onRemove(a.id)}
                        >
                          {t('contentTools.tools.highlight_annotate.delete')}
                        </button>
                      </div>
                    ) : null}
                    {noteForId === a.id ? (
                      <div className="mt-2 space-y-1">
                        <textarea
                          className="w-full rounded border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                          rows={2}
                          value={noteDraft}
                          onChange={(e) => onNoteDraftChange(e.target.value)}
                        />
                        <button
                          type="button"
                          className="rounded bg-slate-800 px-2 py-1 text-white dark:bg-neutral-200 dark:text-neutral-900"
                          onClick={() => onSaveNote(a.id)}
                        >
                          {t('contentTools.tools.highlight_annotate.saveNote')}
                        </button>
                      </div>
                    ) : null}
                  </li>
                ))}
              </ul>
            </div>
          )
        })}
        {active.length === 0 ? (
          <p className="text-xs text-slate-500">
            {t('contentTools.tools.highlight_annotate.noneYet')}
          </p>
        ) : null}
      </section>

      {orphaned.length > 0 ? (
        <details
          className="rounded border border-amber-200 bg-amber-50 p-2 text-xs dark:border-amber-900 dark:bg-amber-950/30"
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

import type { RefObject } from 'react'
import type { Tag } from './types'

export type TagMenuProps = {
  menuRef: RefObject<HTMLDivElement | null>
  menuId: string
  quote: string
  tags: Tag[]
  noteDraft: string
  requireNote: boolean
  applying: boolean
  t: (key: string, options?: Record<string, unknown>) => string
  onNoteDraftChange: (value: string) => void
  onApplyTag: (tagId: string) => void
  onClose: () => void
}

/** Label picker shown after the learner selects a passage span. */
export function TagMenu({
  menuRef,
  menuId,
  quote,
  tags,
  noteDraft,
  requireNote,
  applying,
  t,
  onNoteDraftChange,
  onApplyTag,
  onClose,
}: TagMenuProps) {
  return (
    <div
      ref={menuRef}
      role="dialog"
      aria-modal="false"
      id={menuId}
      aria-label={t('contentTools.tools.highlight_annotate.tagMenu')}
      className="rounded-xl border-2 border-indigo-300 bg-surface-raised p-4 shadow-lg dark:border-indigo-700 dark:bg-surface-base"
      data-testid="ha-tag-menu"
    >
      <p className="text-sm font-semibold text-fg-default">
        {t('contentTools.tools.highlight_annotate.tagMenuTitle')}
      </p>
      <blockquote className="mt-2 rounded-lg border-l-4 border-indigo-400 bg-surface-base px-3 py-2 text-sm text-fg-default dark:bg-surface-raised dark:text-fg-default">
        “{quote}”
      </blockquote>
      <p className="mt-3 mb-2 text-xs font-medium text-fg-muted">
        {t('contentTools.tools.highlight_annotate.pickLabel')}
      </p>
      <div className="flex flex-wrap gap-2" role="menu">
        {tags.map((tag) => (
          <button
            key={tag.id}
            type="button"
            role="menuitem"
            disabled={applying}
            className="inline-flex min-h-10 items-center gap-2 rounded-lg border-2 bg-surface-raised px-3 py-2 text-sm font-semibold shadow-sm transition-transform hover:scale-[1.02] hover:shadow disabled:opacity-60 dark:bg-surface-raised"
            style={{
              borderColor: tag.color,
              color: tag.color,
              backgroundColor: `${tag.color}12`,
            }}
            data-testid={`ha-tag-${tag.id}`}
            onClick={() => onApplyTag(tag.id)}
          >
            <span
              aria-hidden
              className="inline-block h-3 w-3 rounded-full"
              style={{ backgroundColor: tag.color }}
            />
            {tag.label}
          </button>
        ))}
      </div>
      <label className="mt-3 block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {requireNote
            ? t('contentTools.tools.highlight_annotate.noteLabelRequired')
            : t('contentTools.tools.highlight_annotate.noteLabel')}
        </span>
        <textarea
          className="w-full rounded-lg border border-border-strong bg-surface-raised px-3 py-2 text-sm dark:border-border-default dark:bg-surface-base"
          rows={2}
          value={noteDraft}
          data-testid="ha-note-input"
          placeholder={
            requireNote
              ? t('contentTools.tools.highlight_annotate.notePlaceholderRequired')
              : t('contentTools.tools.highlight_annotate.notePlaceholder')
          }
          onChange={(e) => onNoteDraftChange(e.target.value)}
        />
      </label>
      <div className="mt-3 flex flex-wrap items-center gap-3">
        <button
          type="button"
          className="text-sm font-medium text-fg-muted underline-offset-2 hover:underline dark:text-fg-muted"
          onClick={onClose}
        >
          {t('contentTools.tools.highlight_annotate.cancel')}
        </button>
        {requireNote ? (
          <p className="text-xs text-fg-muted">
            {t('contentTools.tools.highlight_annotate.noteThenTag')}
          </p>
        ) : (
          <p className="text-xs text-fg-muted">
            {t('contentTools.tools.highlight_annotate.clickTagToSave')}
          </p>
        )}
      </div>
    </div>
  )
}

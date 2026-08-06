import { useId } from 'react'
import { useTranslation } from 'react-i18next'

export type HighlightAnnotateEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
}

type Tag = { id: string; label: string; color: string; description?: string }
type ExpectedRegion = { tagId: string; quote: string }

const PALETTE = ['#0f766e', '#b45309', '#1d4ed8', '#be123c', '#4338ca', '#15803d']

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asTags(value: Record<string, unknown>): Tag[] {
  if (!Array.isArray(value.tags)) return []
  return value.tags as Tag[]
}

function asExpected(value: Record<string, unknown>): ExpectedRegion[] {
  if (!Array.isArray(value.expectedRegions)) return []
  return value.expectedRegions as ExpectedRegion[]
}

function contrastOk(hex: string): boolean {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim())
  if (!m) return true
  const n = parseInt(m[1], 16)
  const r = (n >> 16) & 255
  const g = (n >> 8) & 255
  const b = n & 255
  const lum = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255
  // Prefer mid/dark accents on light surfaces (WCAG-ish heuristic).
  return lum < 0.75
}

export function HighlightAnnotateEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'ha-editor',
}: HighlightAnnotateEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const tags = asTags(value)
  const expected = asExpected(value)
  const passageSource =
    value.passageSource === 'preceding_block' || value.passageSource === 'section_anchor'
      ? value.passageSource
      : 'inline'

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setTags(next: Tag[]) {
    patch({ tags: next.slice(0, 6) })
  }

  return (
    <div className="space-y-4" data-testid="highlight-annotate-editor">
      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.highlight_annotate.editor.prompt')}
        </span>
        <textarea
          id={`${idPrefix}-${baseId}-prompt`}
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          rows={2}
          disabled={disabled}
          value={typeof value.prompt === 'string' ? value.prompt : ''}
          onChange={(e) => patch({ prompt: e.target.value })}
        />
      </label>

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.highlight_annotate.editor.passageSource')}
        </span>
        <select
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          disabled={disabled}
          value={passageSource}
          onChange={(e) => patch({ passageSource: e.target.value })}
        >
          <option value="inline">
            {t('contentTools.tools.highlight_annotate.editor.passageInline')}
          </option>
          <option value="preceding_block">
            {t('contentTools.tools.highlight_annotate.editor.passagePreceding')}
          </option>
          <option value="section_anchor">
            {t('contentTools.tools.highlight_annotate.editor.passageAnchor')}
          </option>
        </select>
      </label>

      {passageSource === 'inline' ? (
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-fg-muted">
            {t('contentTools.tools.highlight_annotate.editor.passageMarkdown')}
          </span>
          <textarea
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            rows={5}
            disabled={disabled}
            value={typeof value.passageMarkdown === 'string' ? value.passageMarkdown : ''}
            onChange={(e) => patch({ passageMarkdown: e.target.value })}
            data-testid="ha-editor-passage"
          />
        </label>
      ) : null}

      {passageSource === 'section_anchor' ? (
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-fg-muted">
            {t('contentTools.tools.highlight_annotate.editor.sectionAnchor')}
          </span>
          <input
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            disabled={disabled}
            value={typeof value.sectionAnchor === 'string' ? value.sectionAnchor : ''}
            onChange={(e) => patch({ sectionAnchor: e.target.value })}
          />
        </label>
      ) : null}

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.highlight_annotate.editor.granularity')}
        </span>
        <select
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          disabled={disabled}
          value={
            typeof value.unitGranularity === 'string' ? value.unitGranularity : 'sentence'
          }
          onChange={(e) => patch({ unitGranularity: e.target.value })}
        >
          <option value="sentence">sentence</option>
          <option value="paragraph">paragraph</option>
          <option value="line">line</option>
        </select>
      </label>

      <div className="space-y-2">
        <div className="text-xs font-medium text-fg-muted">
          {t('contentTools.tools.highlight_annotate.editor.tags')}
        </div>
        {tags.map((tag, idx) => (
          <div key={tag.id} className="flex flex-wrap items-center gap-2">
            <input
              className="min-w-[8rem] flex-1 rounded border border-border-strong bg-surface-raised px-2 py-1 text-sm dark:border-border-default dark:bg-surface-base"
              disabled={disabled}
              value={tag.label}
              placeholder={t('contentTools.tools.highlight_annotate.editor.tagLabel')}
              onChange={(e) => {
                const copy = [...tags]
                copy[idx] = { ...tag, label: e.target.value }
                setTags(copy)
              }}
            />
            <input
              type="color"
              disabled={disabled}
              value={/^#[0-9a-f]{6}$/i.test(tag.color) ? tag.color : '#0f766e'}
              aria-label={t('contentTools.tools.highlight_annotate.editor.tagColor')}
              onChange={(e) => {
                const copy = [...tags]
                copy[idx] = { ...tag, color: e.target.value }
                setTags(copy)
              }}
            />
            {!contrastOk(tag.color) ? (
              <span className="text-[10px] text-warning-fg">
                {t('contentTools.tools.highlight_annotate.editor.contrastWarn')}
              </span>
            ) : null}
            <button
              type="button"
              className="text-xs text-rose-600 underline"
              disabled={disabled || tags.length <= 1}
              onClick={() => setTags(tags.filter((_, i) => i !== idx))}
            >
              {t('contentTools.tools.highlight_annotate.editor.remove')}
            </button>
          </div>
        ))}
        <button
          type="button"
          className="text-xs text-fg-muted underline dark:text-fg-muted"
          disabled={disabled || tags.length >= 6}
          onClick={() =>
            setTags([
              ...tags,
              {
                id: newId('tag'),
                label: '',
                color: PALETTE[tags.length % PALETTE.length],
              },
            ])
          }
        >
          {t('contentTools.tools.highlight_annotate.editor.addTag')}
        </button>
      </div>

      <div className="grid gap-3 sm:grid-cols-3">
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-fg-muted">
            {t('contentTools.tools.highlight_annotate.editor.minAnnotations')}
          </span>
          <input
            type="number"
            min={1}
            max={50}
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            disabled={disabled}
            value={typeof value.minAnnotations === 'number' ? value.minAnnotations : 1}
            onChange={(e) => patch({ minAnnotations: Number(e.target.value) || 1 })}
          />
        </label>
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-fg-muted">
            {t('contentTools.tools.highlight_annotate.editor.maxAnnotations')}
          </span>
          <input
            type="number"
            min={1}
            max={50}
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            disabled={disabled}
            value={typeof value.maxAnnotations === 'number' ? value.maxAnnotations : 20}
            onChange={(e) => patch({ maxAnnotations: Number(e.target.value) || 20 })}
          />
        </label>
        <label className="flex items-center gap-2 text-xs">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.requireNote === true}
            onChange={(e) => patch({ requireNote: e.target.checked })}
          />
          <span className="font-medium text-fg-muted">
            {t('contentTools.tools.highlight_annotate.editor.requireNote')}
          </span>
        </label>
      </div>

      <div className="space-y-2">
        <div className="text-xs font-medium text-fg-muted">
          {t('contentTools.tools.highlight_annotate.editor.expectedRegions')}
        </div>
        <p className="text-[11px] text-fg-muted">
          {t('contentTools.tools.highlight_annotate.editor.expectedHelp')}
        </p>
        {expected.map((er, idx) => (
          <div key={`${er.tagId}-${idx}`} className="flex flex-wrap items-center gap-2">
            <select
              className="rounded border border-border-strong bg-surface-raised px-2 py-1 text-sm dark:border-border-default dark:bg-surface-base"
              disabled={disabled}
              value={er.tagId}
              onChange={(e) => {
                const copy = [...expected]
                copy[idx] = { ...er, tagId: e.target.value }
                patch({ expectedRegions: copy })
              }}
            >
              {tags.map((tag) => (
                <option key={tag.id} value={tag.id}>
                  {tag.label || tag.id}
                </option>
              ))}
            </select>
            <input
              className="min-w-[12rem] flex-1 rounded border border-border-strong bg-surface-raised px-2 py-1 text-sm dark:border-border-default dark:bg-surface-base"
              disabled={disabled}
              value={er.quote}
              placeholder={t('contentTools.tools.highlight_annotate.editor.expectedQuote')}
              onChange={(e) => {
                const copy = [...expected]
                copy[idx] = { ...er, quote: e.target.value }
                patch({ expectedRegions: copy })
              }}
            />
            <button
              type="button"
              className="text-xs text-rose-600 underline"
              disabled={disabled}
              onClick={() =>
                patch({ expectedRegions: expected.filter((_, i) => i !== idx) })
              }
            >
              {t('contentTools.tools.highlight_annotate.editor.remove')}
            </button>
          </div>
        ))}
        <button
          type="button"
          className="text-xs text-fg-muted underline dark:text-fg-muted"
          disabled={disabled || tags.length === 0}
          onClick={() =>
            patch({
              expectedRegions: [
                ...expected,
                { tagId: tags[0]?.id ?? '', quote: '' },
              ],
            })
          }
        >
          {t('contentTools.tools.highlight_annotate.editor.addExpected')}
        </button>
      </div>
    </div>
  )
}

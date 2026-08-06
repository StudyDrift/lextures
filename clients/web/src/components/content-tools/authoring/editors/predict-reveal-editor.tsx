import { useId } from 'react'
import { useTranslation } from 'react-i18next'

export type PredictRevealEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
}

type Outcome = { id: string; text: string; correct?: boolean }

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asOutcomes(value: Record<string, unknown>): Outcome[] {
  if (!Array.isArray(value.outcomes)) return []
  return value.outcomes as Outcome[]
}

function asReveal(value: Record<string, unknown>): { markdown: string; imageUrl?: string } {
  const r = value.reveal
  if (r && typeof r === 'object') {
    const o = r as { markdown?: string; imageUrl?: string }
    return { markdown: o.markdown ?? '', imageUrl: o.imageUrl }
  }
  return { markdown: '' }
}

export function PredictRevealEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'pr-editor',
}: PredictRevealEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const mode = value.mode === 'open' ? 'open' : 'choice'
  const outcomes = asOutcomes(value)
  const reveal = asReveal(value)

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setOutcomes(next: Outcome[]) {
    patch({ outcomes: next.slice(0, 8) })
  }

  return (
    <div className="space-y-4" data-testid="predict-reveal-editor">
      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.predict_reveal.editor.question')}
        </span>
        <textarea
          id={`${idPrefix}-${baseId}-q`}
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          rows={2}
          disabled={disabled}
          value={typeof value.question === 'string' ? value.question : ''}
          onChange={(e) => patch({ question: e.target.value })}
        />
      </label>

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.predict_reveal.editor.mode')}
        </span>
        <select
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          disabled={disabled}
          value={mode}
          onChange={(e) => {
            const next = e.target.value === 'open' ? 'open' : 'choice'
            patch({
              mode: next,
              outcomes:
                next === 'choice' && outcomes.length < 2
                  ? [
                      { id: newId('out'), text: '', correct: true },
                      { id: newId('out'), text: '', correct: false },
                    ]
                  : outcomes,
            })
          }}
        >
          <option value="choice">{t('contentTools.tools.predict_reveal.editor.modeChoice')}</option>
          <option value="open">{t('contentTools.tools.predict_reveal.editor.modeOpen')}</option>
        </select>
      </label>

      {mode === 'choice' ? (
        <div className="space-y-2">
          <div className="text-xs font-medium text-fg-muted">
            {t('contentTools.tools.predict_reveal.editor.outcomes')}
          </div>
          {outcomes.map((o, idx) => (
            <div key={o.id} className="flex flex-wrap items-center gap-2">
              <input
                className="min-w-[12rem] flex-1 rounded border border-border-strong bg-surface-raised px-2 py-1 text-sm dark:border-border-default dark:bg-surface-base"
                disabled={disabled}
                value={o.text}
                placeholder={t('contentTools.tools.predict_reveal.editor.outcomeText')}
                onChange={(e) => {
                  const copy = [...outcomes]
                  copy[idx] = { ...o, text: e.target.value }
                  setOutcomes(copy)
                }}
              />
              <label className="flex items-center gap-1 text-xs">
                <input
                  type="checkbox"
                  disabled={disabled}
                  checked={Boolean(o.correct)}
                  onChange={(e) => {
                    const copy = [...outcomes]
                    copy[idx] = { ...o, correct: e.target.checked }
                    setOutcomes(copy)
                  }}
                />
                {t('contentTools.tools.predict_reveal.editor.correct')}
              </label>
              <button
                type="button"
                className="text-xs text-rose-600 underline"
                disabled={disabled || outcomes.length <= 2}
                onClick={() => setOutcomes(outcomes.filter((_, i) => i !== idx))}
              >
                {t('contentTools.tools.predict_reveal.editor.remove')}
              </button>
            </div>
          ))}
          <button
            type="button"
            className="text-xs text-fg-muted underline dark:text-fg-muted"
            disabled={disabled || outcomes.length >= 8}
            onClick={() =>
              setOutcomes([...outcomes, { id: newId('out'), text: '', correct: false }])
            }
          >
            {t('contentTools.tools.predict_reveal.editor.addOutcome')}
          </button>
        </div>
      ) : (
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-fg-muted">
            {t('contentTools.tools.predict_reveal.editor.openPlaceholder')}
          </span>
          <input
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            disabled={disabled}
            value={typeof value.openPlaceholder === 'string' ? value.openPlaceholder : ''}
            onChange={(e) => patch({ openPlaceholder: e.target.value })}
          />
        </label>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-fg-muted">
            {t('contentTools.tools.predict_reveal.editor.confidenceScale')}
          </span>
          <select
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            disabled={disabled}
            value={typeof value.confidenceScale === 'string' ? value.confidenceScale : 'three'}
            onChange={(e) => patch({ confidenceScale: e.target.value })}
          >
            <option value="none">none</option>
            <option value="three">three</option>
            <option value="five">five</option>
            <option value="percent">percent</option>
          </select>
        </label>
        <label className="flex items-center gap-2 text-xs">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.confidenceRequired !== false}
            onChange={(e) => patch({ confidenceRequired: e.target.checked })}
          />
          <span className="font-medium text-fg-muted">
            {t('contentTools.tools.predict_reveal.editor.confidenceRequired')}
          </span>
        </label>
      </div>

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.predict_reveal.editor.revealMarkdown')}
        </span>
        <textarea
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          rows={3}
          disabled={disabled}
          value={reveal.markdown}
          onChange={(e) => patch({ reveal: { ...reveal, markdown: e.target.value } })}
        />
      </label>

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.predict_reveal.editor.revealImage')}
        </span>
        <input
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          disabled={disabled}
          value={reveal.imageUrl ?? ''}
          onChange={(e) =>
            patch({ reveal: { ...reveal, imageUrl: e.target.value || undefined } })
          }
        />
      </label>

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.predict_reveal.editor.reflectionPrompt')}
        </span>
        <input
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          disabled={disabled}
          value={typeof value.reflectionPrompt === 'string' ? value.reflectionPrompt : ''}
          onChange={(e) => patch({ reflectionPrompt: e.target.value })}
        />
      </label>

      <label className="flex items-center gap-2 text-xs">
        <input
          type="checkbox"
          disabled={disabled}
          checked={value.showPeerResults === true}
          onChange={(e) => patch({ showPeerResults: e.target.checked })}
        />
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.predict_reveal.editor.showPeerResults')}
        </span>
      </label>
    </div>
  )
}

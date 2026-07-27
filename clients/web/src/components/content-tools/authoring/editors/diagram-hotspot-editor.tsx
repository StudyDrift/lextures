import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import { RegionEditorCard } from './diagram-hotspot-region-card'
import { defaultShape, type EditorRegion } from './diagram-hotspot-region-helpers'

export type DiagramHotspotEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
}

type LabelChip = { id: string; text: string }
type Prompt = { id: string; text: string }
type ImageRef = { url: string; alt: string; naturalWidth: number; naturalHeight: number }

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asRegions(value: Record<string, unknown>): EditorRegion[] {
  return Array.isArray(value.regions) ? (value.regions as EditorRegion[]) : []
}

function asLabels(value: Record<string, unknown>): LabelChip[] {
  return Array.isArray(value.labels) ? (value.labels as LabelChip[]) : []
}

function asPrompts(value: Record<string, unknown>): Prompt[] {
  return Array.isArray(value.prompts) ? (value.prompts as Prompt[]) : []
}

function asImage(value: Record<string, unknown>): ImageRef {
  const img = value.image && typeof value.image === 'object' ? (value.image as ImageRef) : null
  return {
    url: img?.url ?? '',
    alt: img?.alt ?? '',
    naturalWidth: img?.naturalWidth || 800,
    naturalHeight: img?.naturalHeight || 600,
  }
}

function asMap(value: Record<string, unknown>, key: string): Record<string, string> {
  const v = value[key]
  if (v && typeof v === 'object') return v as Record<string, string>
  return {}
}

export function DiagramHotspotEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'dh-editor',
}: DiagramHotspotEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const mode = value.mode === 'hotspot' ? 'hotspot' : 'label'
  const image = asImage(value)
  const regions = asRegions(value)
  const labels = asLabels(value)
  const prompts = asPrompts(value)
  const correctByLabel = asMap(value, 'correctRegionByLabel')
  const correctByPrompt = asMap(value, 'correctRegionByPrompt')
  const feedback = asMap(value, 'feedbackByRegion')

  const missingDescriptions = regions.some((r) => !r.description.trim())
  const missingAlt = !image.alt.trim()

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setRegions(next: EditorRegion[]) {
    const capped = next.slice(0, 40)
    const ids = new Set(capped.map((r) => r.id))
    const nextFeedback: Record<string, string> = {}
    for (const id of ids) {
      if (feedback[id]) nextFeedback[id] = feedback[id]!
    }
    const nextCorrectLabel: Record<string, string> = {}
    for (const [k, v] of Object.entries(correctByLabel)) {
      if (ids.has(v)) nextCorrectLabel[k] = v
    }
    const nextCorrectPrompt: Record<string, string> = {}
    for (const [k, v] of Object.entries(correctByPrompt)) {
      if (ids.has(v)) nextCorrectPrompt[k] = v
    }
    patch({
      regions: capped,
      feedbackByRegion: nextFeedback,
      correctRegionByLabel: nextCorrectLabel,
      correctRegionByPrompt: nextCorrectPrompt,
    })
  }

  return (
    <div className="space-y-4" data-testid="diagram-hotspot-editor">
      {missingDescriptions ? (
        <p
          className="rounded border border-rose-300 bg-rose-50 px-3 py-2 text-xs text-rose-800 dark:border-rose-700 dark:bg-rose-950 dark:text-rose-100"
          role="alert"
        >
          {t('contentTools.tools.diagram_hotspot.editor.descriptionRequired')}
        </p>
      ) : null}
      {missingAlt ? (
        <p
          className="rounded border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-100"
          role="alert"
        >
          {t('contentTools.tools.diagram_hotspot.editor.altRequired')}
        </p>
      ) : null}

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.diagram_hotspot.editor.prompt')}
        </span>
        <textarea
          id={`${idPrefix}-${baseId}-prompt`}
          className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
          rows={2}
          disabled={disabled}
          value={typeof value.prompt === 'string' ? value.prompt : ''}
          onChange={(e) => patch({ prompt: e.target.value })}
        />
      </label>

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.diagram_hotspot.editor.mode')}
        </span>
        <select
          className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
          disabled={disabled}
          value={mode}
          onChange={(e) => patch({ mode: e.target.value })}
        >
          <option value="label">{t('contentTools.tools.diagram_hotspot.editor.modeLabel')}</option>
          <option value="hotspot">{t('contentTools.tools.diagram_hotspot.editor.modeHotspot')}</option>
        </select>
      </label>

      <fieldset className="space-y-2 rounded border border-slate-200 p-3 dark:border-neutral-700">
        <legend className="px-1 text-xs font-medium">
          {t('contentTools.tools.diagram_hotspot.editor.image')}
        </legend>
        <label className="block space-y-1 text-xs">
          <span>{t('contentTools.tools.diagram_hotspot.editor.imageUrl')}</span>
          <input
            className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            disabled={disabled}
            value={image.url}
            onChange={(e) => patch({ image: { ...image, url: e.target.value } })}
          />
        </label>
        <label className="block space-y-1 text-xs">
          <span>{t('contentTools.tools.diagram_hotspot.editor.imageAlt')}</span>
          <input
            data-testid="diagram-editor-alt"
            className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            disabled={disabled}
            value={image.alt}
            onChange={(e) => patch({ image: { ...image, alt: e.target.value } })}
          />
        </label>
        <div className="grid grid-cols-2 gap-2">
          <label className="block space-y-1 text-xs">
            <span>{t('contentTools.tools.diagram_hotspot.editor.width')}</span>
            <input
              type="number"
              min={1}
              className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              disabled={disabled}
              value={image.naturalWidth}
              onChange={(e) =>
                patch({ image: { ...image, naturalWidth: Number(e.target.value) || 1 } })
              }
            />
          </label>
          <label className="block space-y-1 text-xs">
            <span>{t('contentTools.tools.diagram_hotspot.editor.height')}</span>
            <input
              type="number"
              min={1}
              className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              disabled={disabled}
              value={image.naturalHeight}
              onChange={(e) =>
                patch({ image: { ...image, naturalHeight: Number(e.target.value) || 1 } })
              }
            />
          </label>
        </div>
      </fieldset>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.diagram_hotspot.editor.regions')}
          </h3>
          <button
            type="button"
            data-testid="diagram-editor-add-region"
            className="text-xs text-sky-700 underline dark:text-sky-300"
            disabled={disabled || regions.length >= 40}
            onClick={() =>
              setRegions([
                ...regions,
                {
                  id: newId('region'),
                  label: '',
                  description: '',
                  shape: defaultShape('rect'),
                },
              ])
            }
          >
            {t('contentTools.tools.diagram_hotspot.editor.addRegion')}
          </button>
        </div>
        {regions.map((region, idx) => (
          <RegionEditorCard
            key={region.id}
            region={region}
            feedback={feedback[region.id] ?? ''}
            disabled={disabled}
            onChange={(next) => {
              const copy = [...regions]
              copy[idx] = next
              setRegions(copy)
            }}
            onFeedbackChange={(v) =>
              patch({ feedbackByRegion: { ...feedback, [region.id]: v } })
            }
            onRemove={() => setRegions(regions.filter((r) => r.id !== region.id))}
          />
        ))}
      </div>

      {mode === 'label' ? (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-medium">{t('contentTools.tools.diagram_hotspot.editor.labels')}</h3>
            <button
              type="button"
              className="text-xs text-sky-700 underline"
              disabled={disabled}
              onClick={() =>
                patch({ labels: [...labels, { id: newId('label'), text: '' }].slice(0, 40) })
              }
            >
              {t('contentTools.tools.diagram_hotspot.editor.addLabel')}
            </button>
          </div>
          {labels.map((label, idx) => (
            <div key={label.id} className="grid gap-2 sm:grid-cols-3">
              <input
                className="rounded border px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                disabled={disabled}
                placeholder={t('contentTools.tools.diagram_hotspot.editor.labelText')}
                value={label.text}
                onChange={(e) => {
                  const next = [...labels]
                  next[idx] = { ...label, text: e.target.value }
                  patch({ labels: next })
                }}
              />
              <select
                className="rounded border px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                disabled={disabled}
                value={correctByLabel[label.id] ?? ''}
                onChange={(e) =>
                  patch({
                    correctRegionByLabel: { ...correctByLabel, [label.id]: e.target.value },
                  })
                }
              >
                <option value="">{t('contentTools.tools.diagram_hotspot.editor.correctRegion')}</option>
                {regions.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.label || r.id}
                  </option>
                ))}
              </select>
              <button
                type="button"
                className="text-xs text-rose-700 underline"
                disabled={disabled}
                onClick={() => patch({ labels: labels.filter((l) => l.id !== label.id) })}
              >
                {t('contentTools.tools.diagram_hotspot.editor.remove')}
              </button>
            </div>
          ))}
        </div>
      ) : (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-medium">{t('contentTools.tools.diagram_hotspot.editor.prompts')}</h3>
            <button
              type="button"
              className="text-xs text-sky-700 underline"
              disabled={disabled}
              onClick={() =>
                patch({ prompts: [...prompts, { id: newId('prompt'), text: '' }].slice(0, 20) })
              }
            >
              {t('contentTools.tools.diagram_hotspot.editor.addPrompt')}
            </button>
          </div>
          {prompts.map((p, idx) => (
            <div key={p.id} className="grid gap-2 sm:grid-cols-3">
              <input
                className="rounded border px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                disabled={disabled}
                value={p.text}
                onChange={(e) => {
                  const next = [...prompts]
                  next[idx] = { ...p, text: e.target.value }
                  patch({ prompts: next })
                }}
              />
              <select
                className="rounded border px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                disabled={disabled}
                value={correctByPrompt[p.id] ?? ''}
                onChange={(e) =>
                  patch({
                    correctRegionByPrompt: { ...correctByPrompt, [p.id]: e.target.value },
                  })
                }
              >
                <option value="">{t('contentTools.tools.diagram_hotspot.editor.correctRegion')}</option>
                {regions.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.label || r.id}
                  </option>
                ))}
              </select>
              <button
                type="button"
                className="text-xs text-rose-700 underline"
                disabled={disabled}
                onClick={() => patch({ prompts: prompts.filter((x) => x.id !== p.id) })}
              >
                {t('contentTools.tools.diagram_hotspot.editor.remove')}
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="grid gap-2 sm:grid-cols-2">
        <label className="block space-y-1 text-xs">
          <span>{t('contentTools.tools.diagram_hotspot.editor.attempts')}</span>
          <select
            className="w-full rounded border px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            disabled={disabled}
            value={value.attempts === 'unlimited' ? 'unlimited' : String(value.attempts ?? 3)}
            onChange={(e) =>
              patch({
                attempts: e.target.value === 'unlimited' ? 'unlimited' : Number(e.target.value),
              })
            }
          >
            {[1, 2, 3, 4, 5].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
            <option value="unlimited">{t('contentTools.tools.diagram_hotspot.editor.unlimited')}</option>
          </select>
        </label>
        <label className="block space-y-1 text-xs">
          <span>{t('contentTools.tools.diagram_hotspot.editor.outlines')}</span>
          <select
            className="w-full rounded border px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            disabled={disabled}
            value={
              typeof value.showRegionOutlines === 'string' ? value.showRegionOutlines : 'on_focus'
            }
            onChange={(e) => patch({ showRegionOutlines: e.target.value })}
          >
            <option value="always">{t('contentTools.tools.diagram_hotspot.editor.outlinesAlways')}</option>
            <option value="on_focus">
              {t('contentTools.tools.diagram_hotspot.editor.outlinesOnFocus')}
            </option>
            <option value="after_check">
              {t('contentTools.tools.diagram_hotspot.editor.outlinesAfterCheck')}
            </option>
          </select>
        </label>
      </div>

      <label className="flex items-center gap-2 text-xs">
        <input
          type="checkbox"
          disabled={disabled}
          checked={value.lockCorrect !== false}
          onChange={(e) => patch({ lockCorrect: e.target.checked })}
        />
        {t('contentTools.tools.diagram_hotspot.editor.lockCorrect')}
      </label>
      <label className="flex items-center gap-2 text-xs">
        <input
          type="checkbox"
          disabled={disabled}
          checked={value.showPerItemCorrectness !== false}
          onChange={(e) => patch({ showPerItemCorrectness: e.target.checked })}
        />
        {t('contentTools.tools.diagram_hotspot.editor.showPerItem')}
      </label>
    </div>
  )
}

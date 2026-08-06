import { useTranslation } from 'react-i18next'
import {
  defaultShape,
  descriptionWarning,
  type EditorRegion,
} from './diagram-hotspot-region-helpers'

export type RegionEditorCardProps = {
  region: EditorRegion
  feedback: string
  disabled?: boolean
  onChange: (next: EditorRegion) => void
  onFeedbackChange: (value: string) => void
  onRemove: () => void
}

export function RegionEditorCard({
  region,
  feedback,
  disabled,
  onChange,
  onFeedbackChange,
  onRemove,
}: RegionEditorCardProps) {
  const { t } = useTranslation('contentTools')
  const warn = descriptionWarning(region.label, region.description)

  return (
    <div
      className="space-y-2 rounded border border-border-default p-3 dark:border-border-default"
      data-testid={`diagram-editor-region-${region.id}`}
    >
      <div className="grid gap-2 sm:grid-cols-2">
        <label className="block space-y-1 text-xs">
          <span>{t('contentTools.tools.diagram_hotspot.editor.regionLabel')}</span>
          <input
            className="w-full rounded border px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            disabled={disabled}
            value={region.label}
            onChange={(e) => onChange({ ...region, label: e.target.value })}
          />
        </label>
        <label className="block space-y-1 text-xs">
          <span>{t('contentTools.tools.diagram_hotspot.editor.shape')}</span>
          <select
            className="w-full rounded border px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            disabled={disabled}
            value={region.shape.kind}
            onChange={(e) => {
              const kind = e.target.value as EditorRegion['shape']['kind']
              onChange({ ...region, shape: defaultShape(kind) })
            }}
          >
            <option value="rect">rect</option>
            <option value="circle">circle</option>
            <option value="polygon">polygon</option>
          </select>
        </label>
      </div>
      <label className="block space-y-1 text-xs">
        <span>{t('contentTools.tools.diagram_hotspot.editor.regionDescription')}</span>
        <textarea
          data-testid={`diagram-editor-desc-${region.id}`}
          className="w-full rounded border px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          rows={2}
          disabled={disabled}
          value={region.description}
          onChange={(e) => onChange({ ...region, description: e.target.value })}
        />
      </label>
      {warn === 'same_as_label' || warn === 'too_short' ? (
        <p className="text-xs text-warning-fg">
          {t('contentTools.tools.diagram_hotspot.editor.descriptionQualityWarning')}
        </p>
      ) : null}
      {region.shape.kind === 'rect' ? (
        <div className="grid grid-cols-4 gap-1 text-xs">
          {(['x', 'y', 'w', 'h'] as const).map((k) => (
            <label key={k} className="block">
              <span>{k}</span>
              <input
                type="number"
                step="0.01"
                min={0}
                max={1}
                className="w-full rounded border px-1 py-1 dark:border-border-default dark:bg-surface-base"
                disabled={disabled}
                value={region.shape.kind === 'rect' ? region.shape[k] : 0}
                onChange={(e) => {
                  if (region.shape.kind !== 'rect') return
                  onChange({
                    ...region,
                    shape: { ...region.shape, [k]: Number(e.target.value) },
                  })
                }}
              />
            </label>
          ))}
        </div>
      ) : null}
      {region.shape.kind === 'circle' ? (
        <div className="grid grid-cols-3 gap-1 text-xs">
          {(['cx', 'cy', 'r'] as const).map((k) => (
            <label key={k} className="block">
              <span>{k}</span>
              <input
                type="number"
                step="0.01"
                min={0}
                max={1}
                className="w-full rounded border px-1 py-1 dark:border-border-default dark:bg-surface-base"
                disabled={disabled}
                value={region.shape.kind === 'circle' ? region.shape[k] : 0}
                onChange={(e) => {
                  if (region.shape.kind !== 'circle') return
                  onChange({
                    ...region,
                    shape: { ...region.shape, [k]: Number(e.target.value) },
                  })
                }}
              />
            </label>
          ))}
        </div>
      ) : null}
      {region.shape.kind === 'polygon' ? (
        <label className="block space-y-1 text-xs">
          <span>{t('contentTools.tools.diagram_hotspot.editor.polygonPoints')}</span>
          <textarea
            className="w-full rounded border px-2 py-1.5 font-mono text-xs dark:border-border-default dark:bg-surface-base"
            rows={3}
            disabled={disabled}
            value={region.shape.points.map((p) => p.join(',')).join('\n')}
            onChange={(e) => {
              const points = e.target.value
                .split(/\r?\n/)
                .map((line) => line.split(',').map((n) => Number(n.trim())))
                .filter((p) => p.length >= 2 && p.every((n) => Number.isFinite(n)))
                .map((p) => [p[0]!, p[1]!] as [number, number])
              onChange({ ...region, shape: { kind: 'polygon', points } })
            }}
          />
        </label>
      ) : null}
      <label className="block space-y-1 text-xs">
        <span>{t('contentTools.tools.diagram_hotspot.editor.feedback')}</span>
        <input
          className="w-full rounded border px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          disabled={disabled}
          value={feedback}
          onChange={(e) => onFeedbackChange(e.target.value)}
        />
      </label>
      <button
        type="button"
        className="text-xs text-rose-700 underline"
        disabled={disabled}
        onClick={onRemove}
      >
        {t('contentTools.tools.diagram_hotspot.editor.remove')}
      </button>
    </div>
  )
}

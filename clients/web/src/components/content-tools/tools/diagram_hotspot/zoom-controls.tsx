export type ZoomControlsProps = {
  zoom: number
  t: (key: string, opts?: Record<string, unknown>) => string
  onZoomIn: () => void
  onZoomOut: () => void
  onReset: () => void
  onPan: (dx: number, dy: number) => void
}

export function ZoomControls({
  zoom,
  t,
  onZoomIn,
  onZoomOut,
  onReset,
  onPan,
}: ZoomControlsProps) {
  return (
    <div className="flex flex-wrap items-center gap-2 text-xs">
      <button
        type="button"
        data-testid="diagram-zoom-out"
        className="rounded border border-border-strong px-2 py-1 dark:border-border-default"
        onClick={onZoomOut}
      >
        {t('contentTools.tools.diagram_hotspot.zoomOut')}
      </button>
      <span className="tabular-nums">{Math.round(zoom * 100)}%</span>
      <button
        type="button"
        data-testid="diagram-zoom-in"
        className="rounded border border-border-strong px-2 py-1 dark:border-border-default"
        onClick={onZoomIn}
      >
        {t('contentTools.tools.diagram_hotspot.zoomIn')}
      </button>
      <button
        type="button"
        className="rounded border border-border-strong px-2 py-1 dark:border-border-default"
        onClick={onReset}
      >
        {t('contentTools.tools.diagram_hotspot.resetView')}
      </button>
      {zoom > 1 ? (
        <span className="flex gap-1">
          <button type="button" className="rounded border px-2 py-1" onClick={() => onPan(-24, 0)}>
            ←
          </button>
          <button type="button" className="rounded border px-2 py-1" onClick={() => onPan(0, -24)}>
            ↑
          </button>
          <button type="button" className="rounded border px-2 py-1" onClick={() => onPan(0, 24)}>
            ↓
          </button>
          <button type="button" className="rounded border px-2 py-1" onClick={() => onPan(24, 0)}>
            →
          </button>
        </span>
      ) : null}
    </div>
  )
}

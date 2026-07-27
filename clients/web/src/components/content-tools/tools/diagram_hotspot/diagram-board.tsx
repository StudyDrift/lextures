import type { CSSProperties, KeyboardEvent, ReactNode, RefObject } from 'react'
import type { DiagramRegion } from './types'

export type DiagramBoardProps = {
  imageUrl: string
  imageAlt: string
  naturalWidth: number
  naturalHeight: number
  regions: DiagramRegion[]
  showOutlines: boolean
  focusedRegionId: string | null
  selectedRegionId: string | null
  assignments: Record<string, string | null>
  itemByRegion: Record<string, string>
  zoom: number
  pan: { x: number; y: number }
  reducedMotion: boolean
  imageFailed: boolean
  surfaceRef: RefObject<HTMLDivElement | null>
  onImageError: () => void
  onPointerSelect: (clientX: number, clientY: number) => void
  onRegionActivate: (regionId: string) => void
  onRegionKeyDown: (e: KeyboardEvent, regionId: string) => void
  children?: ReactNode
}

function shapeStyle(region: DiagramRegion): CSSProperties {
  const s = region.shape
  if (s.kind === 'rect') {
    return {
      left: `${s.x * 100}%`,
      top: `${s.y * 100}%`,
      width: `${s.w * 100}%`,
      height: `${s.h * 100}%`,
      borderRadius: 2,
    }
  }
  if (s.kind === 'circle') {
    return {
      left: `${(s.cx - s.r) * 100}%`,
      top: `${(s.cy - s.r) * 100}%`,
      width: `${s.r * 2 * 100}%`,
      height: `${s.r * 2 * 100}%`,
      borderRadius: '50%',
    }
  }
  // polygon: approximate with bounding box + clip-path
  const xs = s.points.map((p) => p[0])
  const ys = s.points.map((p) => p[1])
  const minX = Math.min(...xs)
  const minY = Math.min(...ys)
  const maxX = Math.max(...xs)
  const maxY = Math.max(...ys)
  const clip = s.points
    .map(([px, py]) => `${((px - minX) / (maxX - minX || 1)) * 100}% ${((py - minY) / (maxY - minY || 1)) * 100}%`)
    .join(', ')
  return {
    left: `${minX * 100}%`,
    top: `${minY * 100}%`,
    width: `${(maxX - minX) * 100}%`,
    height: `${(maxY - minY) * 100}%`,
    clipPath: `polygon(${clip})`,
  }
}

export function DiagramBoard({
  imageUrl,
  imageAlt,
  naturalWidth,
  naturalHeight,
  regions,
  showOutlines,
  focusedRegionId,
  selectedRegionId,
  itemByRegion,
  zoom,
  pan,
  reducedMotion,
  imageFailed,
  surfaceRef,
  onImageError,
  onPointerSelect,
  onRegionActivate,
  onRegionKeyDown,
  children,
}: DiagramBoardProps) {
  const transform = `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`
  const transition = reducedMotion ? undefined : 'transform 120ms ease-out'

  if (imageFailed) {
    return (
      <div
        className="rounded border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-100"
        data-testid="diagram-image-failed"
        role="alert"
      >
        {children}
      </div>
    )
  }

  return (
    <div
      ref={surfaceRef}
      className="relative overflow-hidden rounded border border-slate-300 bg-slate-100 dark:border-neutral-600 dark:bg-neutral-900"
      data-testid="diagram-board"
      style={{ aspectRatio: `${naturalWidth} / ${naturalHeight}`, maxHeight: '28rem' }}
      onClick={(e) => onPointerSelect(e.clientX, e.clientY)}
    >
      <div
        className="absolute inset-0 origin-center"
        style={{ transform, transition }}
      >
        <img
          src={imageUrl}
          alt={imageAlt}
          className="h-full w-full object-contain select-none"
          draggable={false}
          onError={onImageError}
        />
        {regions.map((region) => {
          const focused = focusedRegionId === region.id
          const selected = selectedRegionId === region.id
          const placedLabel = itemByRegion[region.id]
          const visible = showOutlines || focused || selected || Boolean(placedLabel)
          return (
            <button
              key={region.id}
              type="button"
              data-testid={`diagram-region-${region.id}`}
              aria-label={`${region.label}: ${region.description}`}
              className={`absolute border-2 bg-sky-500/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 ${
                selected
                  ? 'border-sky-700 dark:border-sky-300'
                  : focused
                    ? 'border-amber-500'
                    : 'border-teal-700/80 dark:border-teal-300/80'
              } ${visible ? 'opacity-100' : 'opacity-0 focus-visible:opacity-100'}`}
              style={shapeStyle(region)}
              onClick={(e) => {
                e.stopPropagation()
                onRegionActivate(region.id)
              }}
              onKeyDown={(e) => onRegionKeyDown(e, region.id)}
            >
              {placedLabel ? (
                <span
                  className="absolute rounded bg-white/90 px-1 text-[10px] font-medium text-slate-800 shadow dark:bg-neutral-900/90 dark:text-neutral-100"
                  style={{ left: 4, top: 4 }}
                >
                  {placedLabel}
                </span>
              ) : null}
            </button>
          )
        })}
      </div>
      {children}
    </div>
  )
}

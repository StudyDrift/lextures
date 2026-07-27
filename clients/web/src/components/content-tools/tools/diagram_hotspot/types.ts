import type { RegionShape } from '../../shared/region-geometry'

export type DiagramLabel = { id: string; text: string }
export type DiagramPrompt = { id: string; text: string }
export type DiagramRegion = {
  id: string
  label: string
  description: string
  shape: RegionShape
}
export type DiagramImage = {
  url: string
  alt: string
  naturalWidth: number
  naturalHeight: number
}

export type CheckResult = {
  perItem?: Record<string, { correct?: boolean; feedback?: string }>
  scorePct?: number
  attemptsRemaining?: number
  showPerItem?: boolean
  error?: string
  message?: string
}

export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export function regionPositionLabel(
  region: DiagramRegion,
  index: number,
  total: number,
): string {
  const [cx, cy] =
    region.shape.kind === 'rect'
      ? [region.shape.x + region.shape.w / 2, region.shape.y + region.shape.h / 2]
      : region.shape.kind === 'circle'
        ? [region.shape.cx, region.shape.cy]
        : region.shape.points.length
          ? [
              region.shape.points.reduce((s, p) => s + p[0], 0) / region.shape.points.length,
              region.shape.points.reduce((s, p) => s + p[1], 0) / region.shape.points.length,
            ]
          : [0.5, 0.5]
  const vertical = cy < 0.33 ? 'top' : cy > 0.66 ? 'bottom' : 'middle'
  const horizontal = cx < 0.33 ? 'left' : cx > 0.66 ? 'right' : 'center'
  return `${vertical}-${horizontal} region, ${index + 1} of ${total}`
}

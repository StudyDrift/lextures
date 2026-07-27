/** Shared normalized region geometry for CT.15 Diagram & Hotspot. */

export type RegionShape =
  | { kind: 'rect'; x: number; y: number; w: number; h: number }
  | { kind: 'circle'; cx: number; cy: number; r: number }
  | { kind: 'polygon'; points: Array<[number, number]> }

export type DiagramRegion = {
  id: string
  label: string
  description: string
  shape: RegionShape
}

export function clamp01(v: number): number {
  if (v < 0) return 0
  if (v > 1) return 1
  return v
}

export function pointInShape(x: number, y: number, shape: RegionShape): boolean {
  switch (shape.kind) {
    case 'rect':
      return x >= shape.x && x <= shape.x + shape.w && y >= shape.y && y <= shape.y + shape.h
    case 'circle': {
      const dx = x - shape.cx
      const dy = y - shape.cy
      return dx * dx + dy * dy <= shape.r * shape.r
    }
    case 'polygon':
      return pointInPolygon(x, y, shape.points)
    default: {
      const _exhaustive: never = shape
      return _exhaustive
    }
  }
}

function pointInPolygon(x: number, y: number, points: Array<[number, number]>): boolean {
  if (points.length < 3) return false
  let inside = false
  for (let i = 0, j = points.length - 1; i < points.length; j = i++) {
    const [xi, yi] = points[i]!
    const [xj, yj] = points[j]!
    const intersect = yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi + 1e-15) + xi
    if (intersect) inside = !inside
  }
  return inside
}

export function shapeArea(shape: RegionShape): number {
  switch (shape.kind) {
    case 'rect':
      return Math.abs(shape.w * shape.h)
    case 'circle':
      return Math.PI * shape.r * shape.r
    case 'polygon':
      return Math.abs(polygonArea(shape.points))
    default: {
      const _exhaustive: never = shape
      return _exhaustive
    }
  }
}

function polygonArea(points: Array<[number, number]>): number {
  if (points.length < 3) return 0
  let sum = 0
  for (let i = 0, j = points.length - 1; i < points.length; j = i++) {
    const [xi, yi] = points[i]!
    const [xj, yj] = points[j]!
    sum += (xj + xi) * (yj - yi)
  }
  return sum / 2
}

export function centroid(shape: RegionShape): [number, number] {
  switch (shape.kind) {
    case 'rect':
      return [shape.x + shape.w / 2, shape.y + shape.h / 2]
    case 'circle':
      return [shape.cx, shape.cy]
    case 'polygon': {
      const points = shape.points
      if (!points.length) return [0.5, 0.5]
      let sx = 0
      let sy = 0
      for (const [px, py] of points) {
        sx += px
        sy += py
      }
      return [sx / points.length, sy / points.length]
    }
    default: {
      const _exhaustive: never = shape
      return _exhaustive
    }
  }
}

/** Smallest containing region wins when overlaps exist. */
export function hitTestRegions(
  regions: DiagramRegion[],
  x: number,
  y: number,
): DiagramRegion | null {
  let best: DiagramRegion | null = null
  let bestArea = Number.POSITIVE_INFINITY
  for (const region of regions) {
    if (!pointInShape(x, y, region.shape)) continue
    const area = shapeArea(region.shape)
    if (area < bestArea) {
      bestArea = area
      best = region
    }
  }
  return best
}

/**
 * Convert a pointer event on an element into normalized image coords,
 * accounting for CSS object-fit:contain letterboxing and zoom transform origin.
 */
export function pointerToNormalized(
  clientX: number,
  clientY: number,
  el: HTMLElement,
  naturalWidth: number,
  naturalHeight: number,
  zoom = 1,
  panX = 0,
  panY = 0,
): [number, number] | null {
  const rect = el.getBoundingClientRect()
  if (rect.width <= 0 || rect.height <= 0 || naturalWidth <= 0 || naturalHeight <= 0) return null

  // Undo pan/zoom (transform-origin center).
  const cx = rect.left + rect.width / 2
  const cy = rect.top + rect.height / 2
  const localX = (clientX - cx) / zoom - panX + rect.width / 2
  const localY = (clientY - cy) / zoom - panY + rect.height / 2

  const scale = Math.min(rect.width / naturalWidth, rect.height / naturalHeight)
  const drawW = naturalWidth * scale
  const drawH = naturalHeight * scale
  const offsetX = (rect.width - drawW) / 2
  const offsetY = (rect.height - drawH) / 2
  const imgX = localX - offsetX
  const imgY = localY - offsetY
  if (imgX < 0 || imgY < 0 || imgX > drawW || imgY > drawH) return null
  return [clamp01(imgX / drawW), clamp01(imgY / drawH)]
}

export const GRID_SIZE = 8

export function heatCellForPoint(x: number, y: number): string {
  let col = Math.floor(clamp01(x) * GRID_SIZE)
  let row = Math.floor(clamp01(y) * GRID_SIZE)
  if (col >= GRID_SIZE) col = GRID_SIZE - 1
  if (row >= GRID_SIZE) row = GRID_SIZE - 1
  return `r${row}c${col}`
}

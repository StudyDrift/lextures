import type { RegionShape } from '../../shared/region-geometry'

export type EditorRegion = {
  id: string
  label: string
  description: string
  shape: RegionShape
}

export function defaultShape(kind: RegionShape['kind']): RegionShape {
  switch (kind) {
    case 'rect':
      return { kind: 'rect', x: 0.1, y: 0.1, w: 0.25, h: 0.2 }
    case 'circle':
      return { kind: 'circle', cx: 0.5, cy: 0.5, r: 0.12 }
    case 'polygon':
      return {
        kind: 'polygon',
        points: [
          [0.2, 0.2],
          [0.4, 0.2],
          [0.3, 0.4],
        ],
      }
    default: {
      const _exhaustive: never = kind
      return _exhaustive
    }
  }
}

export function descriptionWarning(label: string, description: string): string | null {
  const d = description.trim()
  if (!d) return 'empty'
  if (d.toLowerCase() === label.trim().toLowerCase()) return 'same_as_label'
  if (d.length < 12) return 'too_short'
  return null
}

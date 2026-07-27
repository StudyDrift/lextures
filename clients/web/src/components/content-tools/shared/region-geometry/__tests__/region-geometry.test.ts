import { describe, expect, it } from 'vitest'
import {
  centroid,
  heatCellForPoint,
  hitTestRegions,
  pointInShape,
  type DiagramRegion,
} from '../index'

describe('region-geometry', () => {
  it('detects points in rect, circle, and polygon', () => {
    expect(pointInShape(0.2, 0.2, { kind: 'rect', x: 0.1, y: 0.1, w: 0.3, h: 0.2 })).toBe(true)
    expect(pointInShape(0.9, 0.9, { kind: 'rect', x: 0.1, y: 0.1, w: 0.3, h: 0.2 })).toBe(false)
    expect(pointInShape(0.5, 0.5, { kind: 'circle', cx: 0.5, cy: 0.5, r: 0.2 })).toBe(true)
    expect(
      pointInShape(0.3, 0.15, {
        kind: 'polygon',
        points: [
          [0.1, 0.1],
          [0.5, 0.1],
          [0.3, 0.4],
        ],
      }),
    ).toBe(true)
  })

  it('picks the smallest overlapping region', () => {
    const regions: DiagramRegion[] = [
      {
        id: 'outer',
        label: 'Outer',
        description: 'big',
        shape: { kind: 'rect', x: 0, y: 0, w: 1, h: 1 },
      },
      {
        id: 'inner',
        label: 'Inner',
        description: 'small',
        shape: { kind: 'rect', x: 0.4, y: 0.4, w: 0.2, h: 0.2 },
      },
    ]
    expect(hitTestRegions(regions, 0.5, 0.5)?.id).toBe('inner')
  })

  it('maps heat cells and centroids', () => {
    expect(heatCellForPoint(0, 0)).toBe('r0c0')
    expect(heatCellForPoint(0.99, 0.99)).toBe('r7c7')
    expect(centroid({ kind: 'rect', x: 0, y: 0, w: 1, h: 1 })).toEqual([0.5, 0.5])
  })
})

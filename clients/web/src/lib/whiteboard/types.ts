export type StrokeEl = {
  type: 'stroke'
  color: string
  width: number
  pts: [number, number][]
}

export type RectEl = {
  type: 'rect'
  color: string
  width: number
  x: number
  y: number
  w: number
  h: number
}

export type CircleEl = {
  type: 'circle'
  color: string
  width: number
  cx: number
  cy: number
  rx: number
  ry: number
}

export type TriangleEl = {
  type: 'triangle'
  color: string
  width: number
  x1: number
  y1: number
  x2: number
  y2: number
  x3: number
  y3: number
}

export type LineEl = {
  type: 'line'
  color: string
  width: number
  x1: number
  y1: number
  x2: number
  y2: number
}

export type DrawEl = StrokeEl | RectEl | CircleEl | TriangleEl | LineEl

export type WhiteboardTool = 'select' | 'pen' | 'line' | 'rect' | 'circle' | 'triangle' | 'eraser'

export const WHITEBOARD_COLORS = [
  '#1e293b',
  '#ef4444',
  '#f97316',
  '#eab308',
  '#22c55e',
  '#3b82f6',
  '#a855f7',
  '#ec4899',
  '#ffffff',
] as const

export const WHITEBOARD_STROKE_WIDTHS = [2, 4, 8] as const
export const WHITEBOARD_ERASER_SIZES = [8, 16, 32] as const

/** Shape icons for MC/poll options — colour is redundant (a11y). */
export type AnswerShape = 'triangle' | 'diamond' | 'circle' | 'square' | 'pentagon' | 'hexagon'

export const ANSWER_SHAPES: AnswerShape[] = [
  'triangle',
  'diamond',
  'circle',
  'square',
  'pentagon',
  'hexagon',
]

export const ANSWER_COLORS = [
  'bg-rose-600 hover:bg-rose-500',
  'bg-sky-600 hover:bg-sky-500',
  'bg-amber-500 hover:bg-amber-400',
  'bg-emerald-600 hover:bg-emerald-500',
  'bg-violet-600 hover:bg-violet-500',
  'bg-cyan-600 hover:bg-cyan-500',
] as const

export function shapeForIndex(i: number): AnswerShape {
  return ANSWER_SHAPES[i % ANSWER_SHAPES.length]!
}

export function colorForIndex(i: number): string {
  return ANSWER_COLORS[i % ANSWER_COLORS.length]!
}

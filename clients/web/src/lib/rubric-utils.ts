import type { RubricDefinition } from './courses-api'

export function formatPointsCell(n: number): string {
  if (!Number.isFinite(n)) return ''
  if (Math.abs(n - Math.round(n)) < 1e-9) return String(Math.round(n))
  let s = n.toFixed(4)
  while (s.includes('.') && (s.endsWith('0') || s.endsWith('.'))) {
    s = s.slice(0, -1)
  }
  return s
}

export function rubricScoresComplete(rubric: RubricDefinition, scores: Record<string, number>): boolean {
  return rubric.criteria.every((c) => scores[c.id] !== undefined && Number.isFinite(scores[c.id]))
}

export function rubricTotal(rubric: RubricDefinition, scores: Record<string, number>): number {
  return rubric.criteria.reduce((sum, c) => sum + (scores[c.id] ?? 0), 0)
}

export function rubricGradedCount(rubric: RubricDefinition, scores: Record<string, number>): number {
  return rubric.criteria.filter((c) => scores[c.id] !== undefined && Number.isFinite(scores[c.id])).length
}

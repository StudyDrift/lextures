import type { CourseOutcome, CourseOutcomeLink } from '../../lib/courses-api'

export const OUTCOME_MEASUREMENT_LABELS: Record<string, string> = {
  diagnostic: 'Diagnostic',
  formative: 'Formative',
  summative: 'Summative',
  performance: 'Performance / transfer',
}

export const OUTCOME_INTENSITY_LABELS: Record<string, string> = {
  low: 'Light emphasis',
  medium: 'Moderate emphasis',
  high: 'Strong emphasis',
}

export function formatOutcomeLinkLevels(link: CourseOutcomeLink): string {
  const m = OUTCOME_MEASUREMENT_LABELS[link.measurementLevel] ?? link.measurementLevel
  const i = OUTCOME_INTENSITY_LABELS[link.intensityLevel] ?? link.intensityLevel
  return `${m} · ${i}`
}

export type OutcomeLinkTargetKind = 'assignment' | 'quiz' | 'quiz_question'

export type OutcomeLinkRow = {
  outcome: CourseOutcome
  link: CourseOutcomeLink
}

export function filterOutcomeLinksForTarget(
  outcomes: CourseOutcome[],
  itemId: string,
  targetKind: OutcomeLinkTargetKind,
  quizQuestionId?: string,
): OutcomeLinkRow[] {
  const rows: OutcomeLinkRow[] = []
  for (const outcome of outcomes) {
    for (const link of outcome.links) {
      if (link.structureItemId !== itemId) continue
      if (link.targetKind !== targetKind) continue
      if (targetKind === 'quiz_question') {
        if (!quizQuestionId || link.quizQuestionId !== quizQuestionId) continue
      }
      rows.push({ outcome, link })
    }
  }
  return rows
}

export function countOutcomeLinksForItem(
  outcomes: CourseOutcome[],
  itemId: string,
  mode: 'assignment' | 'quiz',
): number {
  let n = 0
  for (const outcome of outcomes) {
    for (const link of outcome.links) {
      if (link.structureItemId !== itemId) continue
      if (mode === 'assignment') {
        if (link.targetKind === 'assignment') n += 1
      } else if (link.targetKind === 'quiz' || link.targetKind === 'quiz_question') {
        n += 1
      }
    }
  }
  return n
}

import type { QuizQuestion } from '../../../lib/courses-api'

export function visibleQuizChoices(question: QuizQuestion): string[] {
  const choices = Array.isArray(question.choices) ? question.choices : []
  return choices.map((choice) => String(choice).trim()).filter((choice) => choice.length > 0)
}

export function orderingItemsForPreview(question: QuizQuestion): string[] {
  const configured = question.typeConfig?.items
  if (Array.isArray(configured)) {
    const items = configured.map((item) => String(item).trim()).filter((item) => item.length > 0)
    if (items.length > 0) return items
  }
  return visibleQuizChoices(question)
}

export type MatchingPairPreview = {
  left: string
  right: string
}

export function matchingPairsForPreview(question: QuizQuestion): MatchingPairPreview[] {
  const configured = question.typeConfig?.pairs
  if (!Array.isArray(configured)) return []
  return configured
    .map((pair) => {
      const record = pair as Record<string, unknown>
      const left = typeof record.left === 'string' ? record.left.trim() : ''
      const right = typeof record.right === 'string' ? record.right.trim() : ''
      return { left, right }
    })
    .filter((pair) => pair.left.length > 0 || pair.right.length > 0)
}

export type QuizAnswerPreview =
  | { kind: 'choices'; labels: string[]; multipleAnswer: boolean }
  | { kind: 'matching'; pairs: MatchingPairPreview[] }
  | { kind: 'ordering'; items: string[] }
  | { kind: 'code'; language: string }
  | { kind: 'open' }
  | { kind: 'media'; mediaKind: 'file' | 'audio' | 'video' }
  | { kind: 'hotspot' }
  | { kind: 'numeric' }
  | { kind: 'formula' }

export function buildQuizAnswerPreview(question: QuizQuestion): QuizAnswerPreview {
  switch (question.questionType) {
    case 'multiple_choice':
    case 'true_false':
      return {
        kind: 'choices',
        labels: visibleQuizChoices(question),
        multipleAnswer: Boolean(question.multipleAnswer),
      }
    case 'matching':
      return { kind: 'matching', pairs: matchingPairsForPreview(question) }
    case 'ordering':
      return { kind: 'ordering', items: orderingItemsForPreview(question) }
    case 'code': {
      const language =
        typeof question.typeConfig?.language === 'string' ? question.typeConfig.language.trim() : ''
      return { kind: 'code', language: language || 'text' }
    }
    case 'file_upload':
      return { kind: 'media', mediaKind: 'file' }
    case 'audio_response':
      return { kind: 'media', mediaKind: 'audio' }
    case 'video_response':
      return { kind: 'media', mediaKind: 'video' }
    case 'hotspot':
      return { kind: 'hotspot' }
    case 'numeric':
      return { kind: 'numeric' }
    case 'formula':
      return { kind: 'formula' }
    default:
      return { kind: 'open' }
  }
}

export function quizQuestionForSlot(questions: QuizQuestion[], index: number): QuizQuestion | null {
  return questions[index] ?? null
}
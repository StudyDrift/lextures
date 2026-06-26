import type { ModuleQuizPayload, QuizQuestion } from '../../../lib/courses-api'

export const MANUAL_GRADING_QUESTION_TYPES = new Set([
  'essay',
  'short_answer',
  'fill_in_blank',
  'file_upload',
  'audio_response',
  'video_response',
  'code',
  'hotspot',
  'formula',
])

export function isManualGradingQuestionType(type: string): boolean {
  return MANUAL_GRADING_QUESTION_TYPES.has(type)
}

export type QuizQuestionSlot = {
  index: number
  label: string
  questionType: string
  needsManualGrading: boolean
  isPoolSlot: boolean
  isShuffled: boolean
  maxPoints: number
  promptPreview: string
}

function isPoolQuestion(question: QuizQuestion | undefined): boolean {
  if (!question?.typeConfig || typeof question.typeConfig !== 'object') return false
  const poolId = (question.typeConfig as { poolId?: string }).poolId
  return typeof poolId === 'string' && poolId.trim() !== ''
}

/** Delivery slot count accounts for question-bank pool sampling. */
export function computeQuizDeliverySlotCount(
  quiz: Pick<ModuleQuizPayload, 'questions' | 'randomQuestionPoolCount'>,
): number {
  const registered = quiz.questions?.length ?? 0
  if (registered === 0) return 0
  const poolCount = quiz.randomQuestionPoolCount
  if (typeof poolCount === 'number' && poolCount > 0 && poolCount < registered) {
    return poolCount
  }
  return registered
}

/**
 * Builds one slot per delivery position. Indices follow the order students see
 * questions (after shuffle / bank sampling at attempt time).
 */
export function computeQuizQuestionSlots(
  quiz: Pick<
    ModuleQuizPayload,
    'questions' | 'randomQuestionPoolCount' | 'shuffleQuestions' | 'usesServerQuestionSampling'
  >,
): QuizQuestionSlot[] {
  const count = computeQuizDeliverySlotCount(quiz)
  const shuffled = Boolean(quiz.shuffleQuestions || quiz.usesServerQuestionSampling)
  return Array.from({ length: count }, (_, index) => {
    const question = quiz.questions[index]
    const questionType = question?.questionType ?? 'essay'
    const maxPoints = typeof question?.points === 'number' ? question.points : 0
    const poolSlot = isPoolQuestion(question) || Boolean(quiz.usesServerQuestionSampling)
    const prompt = typeof question?.prompt === 'string' ? question.prompt.trim() : ''
    return {
      index,
      label: `Question ${index + 1}`,
      questionType,
      needsManualGrading: isManualGradingQuestionType(questionType),
      isPoolSlot: poolSlot,
      isShuffled: shuffled,
      maxPoints,
      promptPreview: prompt.length > 80 ? `${prompt.slice(0, 77)}…` : prompt,
    }
  })
}

export const QUIZ_QUESTION_HANDLE_PREFIX = 'question-'
export const QUIZ_GRADE_HANDLE_PREFIX = 'grade-'

export function quizQuestionHandle(index: number): string {
  return `${QUIZ_QUESTION_HANDLE_PREFIX}${index}`
}

export function quizGradeHandle(index: number): string {
  return `${QUIZ_GRADE_HANDLE_PREFIX}${index}`
}

export function parseQuizQuestionHandle(handle: string): number | null {
  if (!handle.startsWith(QUIZ_QUESTION_HANDLE_PREFIX)) return null
  const parsed = Number.parseInt(handle.slice(QUIZ_QUESTION_HANDLE_PREFIX.length), 10)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : null
}

export function parseQuizGradeHandle(handle: string): number | null {
  if (!handle.startsWith(QUIZ_GRADE_HANDLE_PREFIX)) return null
  const parsed = Number.parseInt(handle.slice(QUIZ_GRADE_HANDLE_PREFIX.length), 10)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : null
}

export function isQuizQuestionHandle(handle: string): boolean {
  return parseQuizQuestionHandle(handle) !== null
}

export function isQuizGradeHandle(handle: string): boolean {
  return parseQuizGradeHandle(handle) !== null
}
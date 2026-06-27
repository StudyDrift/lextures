import { describe, expect, it } from 'vitest'
import type { QuizQuestion } from '../../../../lib/courses-api'
import {
  buildQuizAnswerPreview,
  matchingPairsForPreview,
  orderingItemsForPreview,
  visibleQuizChoices,
} from '../quiz-question-preview'

function question(overrides: Partial<QuizQuestion> = {}): QuizQuestion {
  return {
    id: 'q1',
    prompt: 'Sample prompt',
    questionType: 'multiple_choice',
    choices: ['A', 'B'],
    correctChoiceIndex: 0,
    multipleAnswer: false,
    answerWithImage: false,
    required: true,
    points: 5,
    estimatedMinutes: 0,
    ...overrides,
  }
}

describe('quiz-question-preview', () => {
  it('extracts visible choices', () => {
    expect(visibleQuizChoices(question({ choices: [' Yes ', '', 'No'] }))).toEqual(['Yes', 'No'])
  })

  it('builds choice preview for multiple choice', () => {
    expect(buildQuizAnswerPreview(question())).toEqual({
      kind: 'choices',
      labels: ['A', 'B'],
      multipleAnswer: false,
    })
  })

  it('builds matching preview from typeConfig pairs', () => {
    const preview = buildQuizAnswerPreview(
      question({
        questionType: 'matching',
        typeConfig: {
          pairs: [
            { left: 'Term', right: 'Definition' },
            { left: '  ', right: 'Empty left' },
          ],
        },
      }),
    )
    expect(preview).toEqual({
      kind: 'matching',
      pairs: [
        { left: 'Term', right: 'Definition' },
        { left: '', right: 'Empty left' },
      ],
    })
    expect(matchingPairsForPreview(question({ questionType: 'matching', typeConfig: {} }))).toEqual([])
  })

  it('builds ordering preview from typeConfig items', () => {
    expect(
      orderingItemsForPreview(
        question({
          questionType: 'ordering',
          typeConfig: { items: ['First', 'Second'] },
          choices: ['ignored'],
        }),
      ),
    ).toEqual(['First', 'Second'])
  })

  it('maps open-response question types', () => {
    expect(buildQuizAnswerPreview(question({ questionType: 'essay' }))).toEqual({ kind: 'open' })
    expect(buildQuizAnswerPreview(question({ questionType: 'code', typeConfig: { language: 'python' } }))).toEqual({
      kind: 'code',
      language: 'python',
    })
  })
})
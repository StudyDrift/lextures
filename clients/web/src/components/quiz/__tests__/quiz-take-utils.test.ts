import { describe, expect, it } from 'vitest'
import { type QuizQuestion } from '../../../lib/courses-api'
import { prepareStaticQuestions, visibleChoices } from '../quiz-take-utils'
import { defaultQuizAdvancedSettings } from '../../../lib/courses-api'

function baseQuestion(overrides: Partial<QuizQuestion> = {}): QuizQuestion {
  return {
    id: 'q1',
    prompt: 'Pick one',
    questionType: 'multiple_choice',
    choices: ['A', 'B'],
    correctChoiceIndex: 0,
    multipleAnswer: false,
    answerWithImage: false,
    required: true,
    points: 1,
    estimatedMinutes: 1,
    ...overrides,
  }
}

describe('visibleChoices', () => {
  it('returns trimmed non-empty choices', () => {
    expect(visibleChoices(baseQuestion({ choices: [' A ', '', 'B'] }))).toEqual(['A', 'B'])
  })

  it('treats null choices as empty', () => {
    expect(visibleChoices(baseQuestion({ choices: null as unknown as string[] }))).toEqual([])
  })
})

describe('prepareStaticQuestions', () => {
  it('does not throw when choices is null and shuffleChoices is enabled', () => {
    const advanced = { ...defaultQuizAdvancedSettings(), shuffleChoices: true }
    const questions = [baseQuestion({ choices: null as unknown as string[] })]
    expect(() => prepareStaticQuestions(questions, advanced)).not.toThrow()
  })
})

import { describe, expect, it } from 'vitest'
import { type QuizQuestion } from '../../../lib/courses-api'
import { displayChoiceOptions, prepareStaticQuestions, visibleChoices } from '../quiz-take-utils'
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
    allowAnyAnswer: false,
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

  it('keeps the authored choice index after shuffling', () => {
    const advanced = { ...defaultQuizAdvancedSettings(), shuffleChoices: true }
    const original = baseQuestion({
      choices: ['A', ' ', 'B', 'C'],
      choiceIds: ['a', 'blank', 'b', 'c'],
      correctChoiceIndex: 3,
    })
    for (let n = 0; n < 20; n++) {
      const [shuffled] = prepareStaticQuestions([structuredClone(original)], advanced)
      const shown = displayChoiceOptions(shuffled!)
      expect(shown.map((option) => option.label).sort()).toEqual(['A', 'B', 'C'])
      const byAuthored = new Map(shown.map((option) => [option.index, option.label]))
      expect(byAuthored.get(0)).toBe('A')
      expect(byAuthored.get(2)).toBe('B')
      expect(byAuthored.get(3)).toBe('C')
      expect(shown.find((option) => option.label === 'C')?.index).toBe(original.correctChoiceIndex)
      expect(shuffled!.choiceIds).toEqual(shown.map((option) => original.choiceIds![option.index]))
    }
  })
})

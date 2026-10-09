import { describe, expect, it } from 'vitest'
import {
  datetimeLocalValueToIso,
  formatGradePolicyShort,
  formatLockdownModeLabel,
  formatQuizDateTime,
  isoToDatetimeLocalValue,
  makeQuestion,
  quizDateTimeIsSet,
  setQuestionAllowAnyAnswer,
  summarizeLearnerQuizStanding,
} from '../course-module-quiz-utils'

describe('course-module-quiz-utils', () => {
  it('round-trips datetime-local to ISO', () => {
    const iso = '2026-04-20T15:30:00.000Z'
    const local = isoToDatetimeLocalValue(iso)
    expect(local.length).toBeGreaterThan(0)
    const back = datetimeLocalValueToIso(local)
    expect(back).not.toBeNull()
  })

  it('quizDateTimeIsSet rejects invalid dates', () => {
    expect(quizDateTimeIsSet(null)).toBe(false)
    expect(quizDateTimeIsSet('not-a-date')).toBe(false)
  })

  it('formatQuizDateTime handles null and invalid', () => {
    expect(formatQuizDateTime(null)).toBe('Not set')
    expect(formatQuizDateTime('invalid')).toBe('Not set')
  })

  it('formatGradePolicyShort maps known policies', () => {
    expect(formatGradePolicyShort('highest')).toBe('Highest score')
    expect(formatGradePolicyShort('latest')).toBe('Latest attempt')
    expect(formatGradePolicyShort('custom')).toBe('custom')
  })

  it('summarizeLearnerQuizStanding reports the kept score and a spent cap', () => {
    const standing = summarizeLearnerQuizStanding({
      attempts: [
        {
          attemptNumber: 1,
          pointsEarned: 1,
          pointsPossible: 2,
          scorePercent: 50,
        },
      ],
      policy: 'latest',
      unlimited: false,
      maxAttempts: 1,
      attemptsRemaining: 0,
    })
    expect(standing).toEqual({
      scoreLabel: 'Score: 1 / 2 (50%)',
      attemptsLabel: 'Attempts used: 1 of 1',
      exhausted: true,
    })
  })

  it('summarizeLearnerQuizStanding keeps the highest score and leaves attempts open', () => {
    const standing = summarizeLearnerQuizStanding({
      attempts: [
        { attemptNumber: 1, pointsEarned: 1, pointsPossible: 2, scorePercent: 50 },
        { attemptNumber: 2, pointsEarned: 2, pointsPossible: 2, scorePercent: 100 },
      ],
      policy: 'highest',
      unlimited: false,
      maxAttempts: 3,
      attemptsRemaining: 1,
    })
    expect(standing?.scoreLabel).toBe('Score: 2 / 2 (100%)')
    expect(standing?.attemptsLabel).toBe('Attempts used: 2 of 3')
    expect(standing?.exhausted).toBe(false)
  })

  it('summarizeLearnerQuizStanding waits when every attempt still needs grading', () => {
    const standing = summarizeLearnerQuizStanding({
      attempts: [
        {
          attemptNumber: 1,
          pointsEarned: 0,
          pointsPossible: 2,
          scorePercent: 0,
          needsManualGrading: true,
        },
      ],
      policy: 'latest',
      unlimited: false,
      maxAttempts: 1,
      attemptsRemaining: 0,
    })
    expect(standing?.scoreLabel).toBe('Score pending review')
    expect(standing?.exhausted).toBe(true)
  })

  it('formatLockdownModeLabel maps modes', () => {
    expect(formatLockdownModeLabel('standard')).toBe('Standard')
    expect(formatLockdownModeLabel('one_at_a_time')).toBe('One at a time')
    expect(formatLockdownModeLabel('kiosk')).toBe('Kiosk')
    expect(formatLockdownModeLabel('kiosk', true)).toBe('Full screen')
    expect(formatLockdownModeLabel('standard', true)).toBe('Usual')
  })

  it('makeQuestion produces a valid multiple-choice draft', () => {
    const q = makeQuestion()
    expect(q.questionType).toBe('multiple_choice')
    expect(q.choices.length).toBe(4)
    expect(q.allowAnyAnswer).toBe(false)
    expect(q.id.length).toBeGreaterThan(0)
  })

  it('setQuestionAllowAnyAnswer clears the marked choice and keeps other config', () => {
    const q = makeQuestion()
    q.correctChoiceIndex = 1
    q.typeConfig = { correctChoiceIndices: [1], unit: 'pts' }
    const on = setQuestionAllowAnyAnswer(q, true)
    expect(on.allowAnyAnswer).toBe(true)
    expect(on.correctChoiceIndex).toBeNull()
    expect(on.typeConfig).toEqual({ unit: 'pts' })
    expect(q.correctChoiceIndex).toBe(1)
    const off = setQuestionAllowAnyAnswer(on, false)
    expect(off.allowAnyAnswer).toBe(false)
    expect(off.correctChoiceIndex).toBeNull()
  })
})

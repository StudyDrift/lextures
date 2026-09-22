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

  it('formatLockdownModeLabel maps modes', () => {
    expect(formatLockdownModeLabel('standard')).toBe('Standard')
    expect(formatLockdownModeLabel('one_at_a_time')).toBe('One at a time')
    expect(formatLockdownModeLabel('kiosk')).toBe('Kiosk')
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

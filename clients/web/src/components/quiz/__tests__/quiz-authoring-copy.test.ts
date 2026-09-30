import { describe, expect, it } from 'vitest'
import { quizAuthoringCopy } from '../quiz-authoring-copy'

describe('quizAuthoringCopy', () => {
  it('keeps campus names for higher-ed', () => {
    const copy = quizAuthoringCopy(false)
    expect(copy.buildWithAi).toBe('Build with AI')
    expect(copy.suggestQuestions).toBe('Generate questions')
    expect(copy.lockdownSummary).toBe('Course lockdown feature')
    expect(copy.lockdownSettingsLabel).toBe('Lockdown delivery')
    expect(copy.buildWithAiDetail).toBeNull()
    expect(copy.lockdownOffNote('Kiosk')).toBe(
      'This quiz is set to kiosk, but the course lockdown feature is off, so learners currently get standard delivery.',
    )
  })

  it('uses parent-friendly labels and keeps power-user names for disclosure', () => {
    const copy = quizAuthoringCopy(true)
    expect(copy.buildWithAi).toBe('Help write the intro')
    expect(copy.buildWithAiDetail).toBe('Also called Build with AI')
    expect(copy.suggestQuestions).toBe('Suggest questions')
    expect(copy.suggestQuestionsAdvanced).toBe('Advanced name: Generate questions.')
    expect(copy.lockdownSummary).toBe('Keep the quiz locked while taking')
    expect(copy.lockdownSummaryDetail).toBe('Also called Course lockdown feature')
    expect(copy.lockdownOptionKiosk).toBe('Full screen')
    expect(copy.advancedLockdown).toContain('Kiosk')
    expect(copy.lockdownOffNote('Full screen')).toContain('keeping the quiz locked while taking')
  })
})

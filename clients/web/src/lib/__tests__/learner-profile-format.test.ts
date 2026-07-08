import { describe, expect, it } from 'vitest'
import { confidenceLevel, formatInsightValue } from '../learner-profile-format'
import type { Insight } from '../learner-profile-api'

const t = ((key: string, vars?: Record<string, unknown>) => {
  if (vars) return `${key}:${JSON.stringify(vars)}`
  return key
}) as import('i18next').TFunction

describe('learner-profile-format', () => {
  it('maps confidence scores to levels', () => {
    expect(confidenceLevel(0.9)).toBe('high')
    expect(confidenceLevel(0.6)).toBe('medium')
    expect(confidenceLevel(0.2)).toBe('low')
  })

  it('formats study rhythm peak window insight', () => {
    const insight: Insight = {
      insightKey: 'peak_study_window',
      label: 'When you study most',
      value: {
        peakWindows: [{ dow: 'Monday', hourBucket: '19:00', share: 0.42 }],
      },
      confidence: 0.42,
      salience: 100,
    }
    const out = formatInsightValue(t, insight, 'study_rhythm')
    expect(out).toContain('learnerProfile.insight.studyRhythm.peak')
    expect(out).toContain('Monday')
  })

  it('formats dynamic topic insights', () => {
    const insight: Insight = {
      insightKey: 'topic_biology',
      label: 'Topic',
      value: { topic: 'Biology', affinity: 0.81 },
      confidence: 0.81,
      salience: 100,
    }
    const out = formatInsightValue(t, insight, 'interests')
    expect(out).toContain('learnerProfile.insight.interests.top')
    expect(out).toContain('Biology')
  })
})
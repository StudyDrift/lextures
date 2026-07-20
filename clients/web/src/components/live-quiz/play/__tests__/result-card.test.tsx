import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { PlatformFeaturesProvider } from '../../../../context/platform-features-context'
import { ResultCard } from '../result-card'
import type { AnswerAck } from '../../../../lib/live-quiz-realtime'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (key === 'liveQuiz.answer.correct') return 'Correct!'
      if (key === 'liveQuiz.answer.points') return `+${opts?.points}`
      if (key === 'liveQuiz.score.breakdown') {
        return `+${opts?.base} base +${opts?.speed} speed +${opts?.streak} streak = ${opts?.total}`
      }
      if (key === 'liveQuiz.score.breakdownAria') return 'Points breakdown'
      if (key === 'liveQuiz.answer.streak') return `Streak ×${opts?.streak}`
      if (key === 'liveQuiz.answer.rank') return `You're in ${opts?.rank}`
      return key
    },
  }),
}))

function wrap(ui: ReactNode) {
  return render(<PlatformFeaturesProvider>{ui}</PlatformFeaturesProvider>)
}

describe('ResultCard', () => {
  it('renders explainable points breakdown', () => {
    const ack: AnswerAck = {
      type: 'answer_ack',
      ok: true,
      isCorrect: true,
      points: 1740,
      streak: 2,
      rank: 1,
      pointsBreakdown: {
        base: 1000,
        speedBonus: 640,
        streakBonus: 100,
        styleMultiplier: 1,
        powerUpFactor: 1,
        total: 1740,
      },
    }
    wrap(<ResultCard ack={ack} />)
    expect(screen.getByText('Correct!')).toBeTruthy()
    expect(screen.getByText('+1740')).toBeTruthy()
    expect(screen.getByText('+1000 base +640 speed +100 streak = 1740')).toBeTruthy()
    expect(screen.getByTestId('quiz-answer-feedback')).toHaveAttribute('data-feedback', 'correct')
  })

  it('applies incorrect feedback without blocking (FR-3 / AC-2)', () => {
    const ack: AnswerAck = {
      type: 'answer_ack',
      ok: true,
      isCorrect: false,
      points: 0,
      pointsBreakdown: { base: 0, speedBonus: 0, streakBonus: 0, styleMultiplier: 1, powerUpFactor: 1, total: 0 },
    }
    wrap(<ResultCard ack={ack} />)
    expect(screen.getByTestId('quiz-answer-feedback')).toHaveAttribute('data-feedback', 'incorrect')
  })
})

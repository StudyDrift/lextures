import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { FeedbackWidgetMenu } from '../feedback-widget'

const platformFeaturesMock = vi.fn()

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => platformFeaturesMock(),
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => (key === 'feedback.button' ? 'Share Feedback' : key),
  }),
}))

vi.mock('../feedback-dialog', () => ({
  FeedbackDialog: () => null,
}))

describe('FeedbackWidgetMenu', () => {
  it('does not render when ffFeedback is off', () => {
    platformFeaturesMock.mockReturnValue({ ffFeedback: false })
    render(<FeedbackWidgetMenu />)
    expect(screen.queryByTestId('feedback-widget-trigger')).not.toBeInTheDocument()
  })

  it('renders accent trigger when ffFeedback is on', () => {
    platformFeaturesMock.mockReturnValue({ ffFeedback: true })
    render(<FeedbackWidgetMenu />)
    const trigger = screen.getByTestId('feedback-widget-trigger')
    expect(trigger).toBeInTheDocument()
    expect(trigger).toHaveAttribute('aria-haspopup', 'dialog')
    expect(trigger).toHaveAttribute('aria-label', 'Share Feedback')
    expect(trigger).toHaveTextContent('Share Feedback')
  })
})

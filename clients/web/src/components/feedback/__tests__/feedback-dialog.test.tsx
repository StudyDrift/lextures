import { type ComponentProps } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { FeedbackDialog } from '../feedback-dialog'
import * as feedbackApi from '../../../lib/feedback-api'
import * as lmsToast from '../../../lib/lms-toast'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (key === 'feedback.message.counter') {
        return `${opts?.count} / ${opts?.max}`
      }
      const labels: Record<string, string> = {
        'feedback.dialog.title': 'Share feedback',
        'feedback.dialog.privacy': 'Privacy note',
        'feedback.message.label': 'Your message',
        'feedback.message.placeholder': 'Type here',
        'feedback.category.label': 'Category',
        'feedback.category.none': 'No category',
        'feedback.category.bug': 'Bug report',
        'feedback.category.idea': 'Feature idea',
        'feedback.category.question': 'Question',
        'feedback.category.praise': 'Praise',
        'feedback.category.other': 'Other',
        'feedback.send': 'Send',
        'feedback.cancel': 'Cancel',
        'feedback.success': 'Thanks for your feedback',
        'feedback.error': 'Something went wrong.',
        'feedback.rateLimited': 'Slow down.',
        'feedback.offline': 'Offline.',
        'dialogs.close': 'Close dialog',
        'dialogs.working': 'Working…',
      }
      return labels[key] ?? key
    },
    i18n: { language: 'en' },
  }),
}))

function renderDialog(props: Partial<ComponentProps<typeof FeedbackDialog>> = {}) {
  const onClose = props.onClose ?? vi.fn()
  render(
    <MemoryRouter initialEntries={['/courses/demo']}>
      <FeedbackDialog open={props.open ?? true} onClose={onClose} />
    </MemoryRouter>,
  )
  return { onClose }
}

describe('FeedbackDialog', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('disables Send when message is empty', () => {
    renderDialog()
    expect(screen.getByRole('button', { name: 'Send' })).toBeDisabled()
  })

  it('enables Send when message has content', async () => {
    renderDialog()
    await userEvent.type(screen.getByLabelText('Your message'), 'A bug on this page')
    expect(screen.getByRole('button', { name: 'Send' })).toBeEnabled()
  })

  it('submits feedback, shows toast, and closes on success', async () => {
    const submitSpy = vi.spyOn(feedbackApi, 'submitFeedback').mockResolvedValue({ ok: true })
    const toastSpy = vi.spyOn(lmsToast, 'toastSaveOk')
    const onClose = vi.fn()
    renderDialog({ onClose })

    await userEvent.type(screen.getByLabelText('Your message'), 'Love the new dashboard')
    await userEvent.selectOptions(screen.getByLabelText('Category'), 'praise')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() => {
      expect(submitSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          message: 'Love the new dashboard',
          category: 'praise',
          route: '/courses/demo',
          locale: 'en',
        }),
      )
    })
    expect(toastSpy).toHaveBeenCalledWith('Thanks for your feedback')
    expect(onClose).toHaveBeenCalled()
  })

  it('shows rate-limit message on 429', async () => {
    vi.spyOn(feedbackApi, 'submitFeedback').mockResolvedValue({ ok: false, kind: 'rate_limited' })
    renderDialog()
    await userEvent.type(screen.getByLabelText('Your message'), 'Another note')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Slow down.')
  })

  it('closes on Escape and preserves input until closed', async () => {
    const onClose = vi.fn()
    renderDialog({ onClose })
    await userEvent.type(screen.getByLabelText('Your message'), 'Draft text')
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('focuses message field on open', async () => {
    renderDialog()
    const message = screen.getByLabelText('Your message')
    await waitFor(() => expect(message).toHaveFocus())
  })
})

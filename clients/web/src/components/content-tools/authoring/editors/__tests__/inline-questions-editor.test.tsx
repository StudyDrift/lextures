import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { InlineQuestionsEditor } from '../inline-questions-editor'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) =>
      opts?.n != null ? `${key}:${opts.n}` : key,
  }),
}))

describe('InlineQuestionsEditor', () => {
  it('adds a question and marks a correct option', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(
      <InlineQuestionsEditor
        value={{
          attempts: 2,
          questions: [
            {
              id: 'q1',
              type: 'single',
              prompt: 'Capital?',
              options: [
                { id: 'a', text: 'A', correct: true },
                { id: 'b', text: 'B', correct: false },
              ],
            },
          ],
        }}
        onChange={onChange}
      />,
    )

    expect(screen.getByTestId('inline-questions-editor')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /addQuestion/i }))
    expect(onChange).toHaveBeenCalled()
    const last = onChange.mock.calls.at(-1)?.[0] as { questions: unknown[] }
    expect(last.questions).toHaveLength(2)
  })

  it('updates questionsAtATime setting', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(
      <InlineQuestionsEditor
        value={{
          attempts: 2,
          questionsAtATime: 'all',
          questions: [
            {
              id: 'q1',
              type: 'single',
              prompt: 'Capital?',
              options: [
                { id: 'a', text: 'A', correct: true },
                { id: 'b', text: 'B', correct: false },
              ],
            },
          ],
        }}
        onChange={onChange}
      />,
    )

    const select = screen.getByLabelText(/questionsAtATime$/i)
    await user.selectOptions(select, '1')
    expect(onChange).toHaveBeenCalled()
    const last = onChange.mock.calls.at(-1)?.[0] as { questionsAtATime: unknown }
    expect(last.questionsAtATime).toBe(1)
  })
})

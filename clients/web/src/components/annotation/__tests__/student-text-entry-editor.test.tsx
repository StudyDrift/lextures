import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { StudentTextEntryEditor, SubmissionBodyText } from '../student-text-entry-editor'
import { submissionTextHasMarkdown } from '../student-text-entry-markdown'

describe('submissionTextHasMarkdown', () => {
  it('keeps plain paragraphs as plain text', () => {
    expect(submissionTextHasMarkdown('I looked at the letter e.\nThe crossbar is thin.')).toBe(false)
  })

  it('detects bold, lists, and links', () => {
    expect(submissionTextHasMarkdown('The letter **e** has a crossbar.')).toBe(true)
    expect(submissionTextHasMarkdown('- open counter\n- crossbar')).toBe(true)
    expect(submissionTextHasMarkdown('See [the chart](https://example.com).')).toBe(true)
  })
})

describe('SubmissionBodyText', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders bold markdown', () => {
    render(<SubmissionBodyText markdown="The letter **e** has a crossbar." />)
    expect(screen.getByText('e').tagName).toBe('STRONG')
  })

  it('preserves line breaks in plain answers', () => {
    render(<SubmissionBodyText markdown={'Line one\nLine two'} />)
    expect(screen.getByText(/Line one/)).toHaveClass('whitespace-pre-wrap')
  })
})

describe('StudentTextEntryEditor', () => {
  afterEach(() => {
    cleanup()
  })

  it('offers clickable formatting controls', async () => {
    const onChange = vi.fn()
    render(
      <>
        <label id="response-label">Your response</label>
        <StudentTextEntryEditor value="" onChange={onChange} labelledBy="response-label" />
      </>,
    )

    expect(await screen.findByRole('toolbar', { name: 'Response formatting' })).toBeInTheDocument()
    for (const name of ['Bold', 'Italic', 'Bullet list', 'Numbered list', 'Link', 'Insert table']) {
      expect(screen.getByRole('button', { name })).toBeEnabled()
    }

    const editor = await screen.findByLabelText('Your response')
    expect(editor).toHaveAttribute('aria-labelledby', 'response-label')

    fireEvent.click(screen.getByRole('button', { name: 'Bullet list' }))

    const markdown = onChange.mock.calls.map((call) => String(call[0])).join('\n')
    expect(markdown).toMatch(/^-\s/m)
  })
})

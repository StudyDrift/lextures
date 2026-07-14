import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../../../context/i18n-provider'
import { MarkdownEmailEditor } from '../markdown-email-editor'

describe('MarkdownEmailEditor', () => {
  it('inserts merge tokens at cursor via onInsertReady', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    let insert: ((token: string) => void) | null = null
    render(
      <I18nProvider>
        <MarkdownEmailEditor
          value="Hello "
          onChange={onChange}
          onInsertReady={(fn) => {
            insert = fn
          }}
        />
      </I18nProvider>,
    )
    const ta = screen.getByRole('textbox') as HTMLTextAreaElement
    await user.click(ta)
    // Move caret to end
    ta.setSelectionRange(6, 6)
    insert!('{{link}}')
    expect(onChange).toHaveBeenCalledWith('Hello {{link}}')
  })

  it('applies bold markdown from toolbar', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(
      <I18nProvider>
        <MarkdownEmailEditor value="hi" onChange={onChange} />
      </I18nProvider>,
    )
    const ta = screen.getByRole('textbox') as HTMLTextAreaElement
    await user.click(ta)
    ta.setSelectionRange(0, 2)
    await user.click(screen.getByRole('button', { name: 'Bold' }))
    expect(onChange).toHaveBeenCalled()
    const last = onChange.mock.calls.at(-1)?.[0] as string
    expect(last).toContain('**')
  })
})

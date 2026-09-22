import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { resolveMarkdownTheme } from '../../../lib/markdown-theme'
import { server } from '../../../test/mocks/server'
import { ContentPageReader } from '../content-page-reader'

vi.mock('../../../context/course-nav-features-context', () => ({
  useCourseNavFeatures: () => ({ contentToolsEnabled: false }),
}))

const theme = resolveMarkdownTheme('classic', null)

function renderReader() {
  return render(
    <ContentPageReader
      markdown="Etymology helps readers."
      theme={theme}
      markups={[]}
      onMarkupsChange={() => {}}
      courseCode="demo"
      markupTarget={{ variant: 'content_page', itemId: 'page-1' }}
      contentTitle="Reading"
    />,
  )
}

function selectWord(word: string) {
  const root = document.querySelector('[data-content-reader]')
  if (!root) throw new Error('reader missing')
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  let node = walker.nextNode()
  while (node) {
    const text = node.textContent ?? ''
    const at = text.indexOf(word)
    if (at >= 0) {
      const range = document.createRange()
      range.setStart(node, at)
      range.setEnd(node, at + word.length)
      const selection = window.getSelection()
      selection?.removeAllRanges()
      selection?.addRange(range)
      act(() => {
        document.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
      })
      return
    }
    node = walker.nextNode()
  }
  throw new Error(`word not found: ${word}`)
}

describe('ContentPageReader selection menu', () => {
  beforeEach(() => {
    const rect = {
      left: 48,
      top: 90,
      width: 88,
      height: 22,
      right: 136,
      bottom: 112,
      x: 48,
      y: 90,
      toJSON() {
        return {}
      },
    }
    Object.defineProperty(Range.prototype, 'getClientRects', {
      configurable: true,
      value() {
        return {
          length: 1,
          item: (index: number) => (index === 0 ? rect : null),
          0: rect,
        } as unknown as DOMRectList
      },
    })
  })

  it('parks the grabbers outside the word and offers define and copy', async () => {
    const user = userEvent.setup()
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })
    server.use(
      http.get('https://api.dictionaryapi.dev/api/v2/entries/en/:word', () =>
        HttpResponse.json([
          {
            word: 'etymology',
            phonetic: '/ˌɛtɪˈmɒlədʒi/',
            meanings: [
              {
                partOfSpeech: 'noun',
                definitions: [{ definition: 'The study of the origin of words.' }],
              },
            ],
          },
        ]),
      ),
    )

    renderReader()
    selectWord('Etymology')

    expect(await screen.findByRole('dialog', { name: 'Selection actions' })).toBeInTheDocument()
    const start = screen.getByRole('button', { name: 'Adjust selection start' })
    const end = screen.getByRole('button', { name: 'Adjust selection end' })
    expect(start).toHaveStyle({ transform: 'translate(calc(-100% + 8px), -50%)' })
    expect(end).toHaveStyle({ transform: 'translate(-8px, -50%)' })
    expect(start.querySelector('span')).toHaveClass('bg-accent-solid')

    await user.click(screen.getByRole('button', { name: 'Define' }))
    expect(await screen.findByText('The study of the origin of words.')).toBeInTheDocument()
    expect(screen.getByText('etymology')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Copy' }))
    await waitFor(() => expect(writeText).toHaveBeenCalledWith('Etymology'))
    expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument()
    expect(screen.getByRole('dialog', { name: 'Selection actions' })).toBeInTheDocument()
  })
})

import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { DictationButton } from '../dictation-button'

describe('DictationButton', () => {
  beforeEach(() => {
    class MockSpeechRecognition {
      continuous = false
      interimResults = false
      lang = 'en-US'
      maxAlternatives = 1
      onstart: (() => void) | null = null
      onend: (() => void) | null = null
      onerror: ((event: { error: string }) => void) | null = null
      onresult: (() => void) | null = null
      start = vi.fn(() => {
        this.onstart?.()
      })
      stop = vi.fn()
      abort = vi.fn()
    }
    vi.stubGlobal('SpeechRecognition', MockSpeechRecognition)
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        getUserMedia: vi.fn().mockResolvedValue({ getTracks: () => [{ stop: vi.fn() }] }),
      },
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders start dictation button when supported', () => {
    render(<DictationButton onFinalResult={vi.fn()} />)
    expect(screen.getByRole('button', { name: 'Start dictation' })).toBeInTheDocument()
  })

  it('toggles aria-pressed when activated', async () => {
    const user = userEvent.setup()
    render(<DictationButton onFinalResult={vi.fn()} />)
    const btn = screen.getByRole('button', { name: 'Start dictation' })
    await user.click(btn)
    expect(screen.getByRole('button', { name: 'Stop dictation' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('returns null when SpeechRecognition is unavailable', () => {
    vi.unstubAllGlobals()
    const { container } = render(<DictationButton onFinalResult={vi.fn()} />)
    expect(container).toBeEmptyDOMElement()
  })
})

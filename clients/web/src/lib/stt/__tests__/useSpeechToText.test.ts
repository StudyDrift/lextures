import { renderHook, act } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { useSpeechToText } from '../useSpeechToText'

type MockRecognition = {
  continuous: boolean
  interimResults: boolean
  lang: string
  maxAlternatives: number
  onstart: (() => void) | null
  onend: (() => void) | null
  onerror: ((event: { error: string }) => void) | null
  onresult: ((event: { resultIndex: number; results: Array<{ isFinal: boolean; 0: { transcript: string } }> }) => void) | null
  start: ReturnType<typeof vi.fn>
  stop: ReturnType<typeof vi.fn>
  abort: ReturnType<typeof vi.fn>
}

function installMockSpeechRecognition() {
  const instances: MockRecognition[] = []
  class MockSpeechRecognition {
    continuous = false
    interimResults = false
    lang = 'en-US'
    maxAlternatives = 1
    onstart: (() => void) | null = null
    onend: (() => void) | null = null
    onerror: ((event: { error: string }) => void) | null = null
    onresult: ((event: {
      resultIndex: number
      results: Array<{ isFinal: boolean; 0: { transcript: string } }>
    }) => void) | null = null
    start = vi.fn(() => {
      this.onstart?.()
    })
    stop = vi.fn(() => {
      this.onend?.()
    })
    abort = vi.fn()
    constructor() {
      instances.push(this as unknown as MockRecognition)
    }
  }
  vi.stubGlobal('SpeechRecognition', MockSpeechRecognition)
  vi.stubGlobal('webkitSpeechRecognition', MockSpeechRecognition)
  Object.defineProperty(navigator, 'mediaDevices', {
    configurable: true,
    value: {
      getUserMedia: vi.fn().mockResolvedValue({ getTracks: () => [{ stop: vi.fn() }] }),
    },
  })
  return instances
}

describe('useSpeechToText', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('transitions idle → listening on start', async () => {
    installMockSpeechRecognition()
    const { result } = renderHook(() => useSpeechToText())
    expect(result.current.status).toBe('idle')
    await act(async () => {
      await result.current.start()
    })
    expect(result.current.status).toBe('listening')
    expect(result.current.liveAnnouncement).toContain('Dictation started')
  })

  it('calls onFinalResult when a final transcript arrives', async () => {
    const instances = installMockSpeechRecognition()
    const onFinalResult = vi.fn()
    const { result } = renderHook(() => useSpeechToText({ onFinalResult }))
    await act(async () => {
      await result.current.start()
    })
    act(() => {
      instances[0]?.onresult?.({
        resultIndex: 0,
        results: [{ isFinal: true, 0: { transcript: 'Hello world' } }],
      })
    })
    expect(onFinalResult).toHaveBeenCalledWith('Hello world')
  })

  it('returns supported=false when SpeechRecognition is missing', () => {
    const { result } = renderHook(() => useSpeechToText())
    expect(result.current.supported).toBe(false)
  })

  it('stop sets status to stopped', async () => {
    installMockSpeechRecognition()
    const { result } = renderHook(() => useSpeechToText())
    await act(async () => {
      await result.current.start()
    })
    act(() => {
      result.current.stop()
    })
    expect(result.current.status).toBe('stopped')
  })
})

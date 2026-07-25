import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  PINNED_SETTINGS_SAVE_DEBOUNCE_MS,
  PINNED_SETTINGS_UI_CAP,
} from '../../../lib/pinned-settings-copy'
import {
  usePinnedSettings,
  __resetPinnedHintDismissedForTests,
  __resetSuggestionsDismissedForTests,
} from '../use-pinned-settings'

const mockFetchDetailed = vi.fn()
const mockSave = vi.fn()
const mockToastError = vi.fn()
let ffPinnedSettings = true

vi.mock('../../../lib/pinned-settings-api', () => ({
  fetchPinnedSettingsDetailed: (...args: unknown[]) => mockFetchDetailed(...args),
  savePinnedSettings: (...args: unknown[]) => mockSave(...args),
}))

vi.mock('../../../lib/lms-toast', () => ({
  toastMutationError: (...args: unknown[]) => mockToastError(...args),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffPinnedSettings }),
}))

describe('usePinnedSettings', () => {
  beforeEach(() => {
    ffPinnedSettings = true
    mockFetchDetailed.mockReset()
    mockSave.mockReset()
    mockToastError.mockReset()
    __resetPinnedHintDismissedForTests()
    __resetSuggestionsDismissedForTests('quiz', false)
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: { assignment: [], quiz: [] },
    })
    mockSave.mockImplementation(async (_surface: string, keys: string[]) => ({
      assignment: [],
      quiz: keys,
    }))
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('status unavailable when flag is off and does not fetch', async () => {
    ffPinnedSettings = false
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    expect(result.current.status).toBe('unavailable')
    expect(result.current.enabled).toBe(false)
    expect(mockFetchDetailed).not.toHaveBeenCalled()
  })

  it('loads pins and becomes ready', async () => {
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: {
        assignment: [],
        quiz: ['quiz.presentation.lockdown-mode'],
      },
    })
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.status).toBe('ready'))
    expect(result.current.keys).toEqual(['quiz.presentation.lockdown-mode'])
    expect(result.current.resolved.map((d) => d.id)).toEqual([
      'quiz.presentation.lockdown-mode',
    ])
  })

  it('GET failure → status unavailable, empty keys (AC-10)', async () => {
    mockFetchDetailed.mockResolvedValue({
      ok: false,
      surfaces: { assignment: [], quiz: [] },
    })
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.status).toBe('unavailable'))
    expect(result.current.keys).toEqual([])
    expect(result.current.showFirstRunHint).toBe(false)
  })

  it('prunes unresolved keys from resolved but keeps them in keys (FR-18)', async () => {
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: {
        assignment: [],
        quiz: ['quiz.retired.bogus', 'quiz.scheduling.due-date'],
      },
    })
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.status).toBe('ready'))
    expect(result.current.keys).toEqual([
      'quiz.retired.bogus',
      'quiz.scheduling.due-date',
    ])
    expect(result.current.resolved.map((d) => d.id)).toEqual(['quiz.scheduling.due-date'])
  })

  it('optimistic pin + debounced save', async () => {
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.status).toBe('ready'))

    act(() => {
      result.current.pin('quiz.presentation.lockdown-mode')
    })
    expect(result.current.keys).toEqual(['quiz.presentation.lockdown-mode'])
    expect(mockSave).not.toHaveBeenCalled()

    await act(async () => {
      vi.advanceTimersByTime(PINNED_SETTINGS_SAVE_DEBOUNCE_MS + 10)
    })
    await waitFor(() => expect(mockSave).toHaveBeenCalledTimes(1))
    expect(mockSave).toHaveBeenCalledWith('quiz', ['quiz.presentation.lockdown-mode'])
  })

  it('reverts and toasts on save failure (AC-9)', async () => {
    mockSave.mockRejectedValue(new Error('server error'))
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.status).toBe('ready'))

    act(() => {
      result.current.pin('quiz.scheduling.due-date')
    })
    expect(result.current.keys).toEqual(['quiz.scheduling.due-date'])

    await act(async () => {
      vi.advanceTimersByTime(PINNED_SETTINGS_SAVE_DEBOUNCE_MS + 10)
    })
    await waitFor(() => expect(mockToastError).toHaveBeenCalled())
    expect(result.current.keys).toEqual([])
  })

  it('enforces UI cap of 8 (AC-7)', async () => {
    const eight = [
      'quiz.scheduling.due-date',
      'quiz.scheduling.visible-from',
      'quiz.scheduling.visible-until',
      'quiz.attempts-grading.unlimited-attempts',
      'quiz.attempts-grading.grade-policy',
      'quiz.attempts-grading.passing-score',
      'quiz.presentation.lockdown-mode',
      'quiz.access.access-code',
    ]
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: { assignment: [], quiz: eight },
    })
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.atCap).toBe(true))
    expect(result.current.keys).toHaveLength(PINNED_SETTINGS_UI_CAP)

    act(() => {
      result.current.pin('quiz.presentation.shuffle-questions')
    })
    expect(result.current.keys).toHaveLength(PINNED_SETTINGS_UI_CAP)
    expect(mockSave).not.toHaveBeenCalled()
  })

  it('moveByOffset reorders and announces', async () => {
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: {
        assignment: [],
        quiz: [
          'quiz.scheduling.due-date',
          'quiz.presentation.lockdown-mode',
          'quiz.access.access-code',
        ],
      },
    })
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.keys).toHaveLength(3))

    act(() => {
      result.current.moveByOffset('quiz.presentation.lockdown-mode', -1)
    })
    expect(result.current.keys[0]).toBe('quiz.presentation.lockdown-mode')
    // announce clears then sets on rAF
    await act(async () => {
      await Promise.resolve()
      vi.runAllTimers()
    })
    await waitFor(() => {
      expect(result.current.liveMessage).toMatch(/moved to position 1 of 3/i)
    })
  })

  it('suggestionsEligible when ready with zero pins and not dismissed (PS.4)', async () => {
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.status).toBe('ready'))
    // Suggestions replace the generic first-run hint when curated IDs resolve.
    expect(result.current.suggestionsEligible).toBe(true)
    expect(result.current.showFirstRunHint).toBe(false)
    expect(result.current.suggestedPins.length).toBeGreaterThan(0)

    act(() => {
      result.current.dismissSuggestions()
    })
    expect(result.current.suggestionsEligible).toBe(false)
  })

  it('showFirstRunHint only when suggestions are not eligible', async () => {
    __resetSuggestionsDismissedForTests('quiz', true)
    __resetPinnedHintDismissedForTests(false)
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.status).toBe('ready'))
    expect(result.current.suggestionsEligible).toBe(false)
    expect(result.current.showFirstRunHint).toBe(true)

    act(() => {
      result.current.dismissFirstRunHint()
    })
    expect(result.current.showFirstRunHint).toBe(false)
  })

  it('unpin removes key and force-opens section', async () => {
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: {
        assignment: [],
        quiz: ['quiz.presentation.lockdown-mode'],
      },
    })
    const { result } = renderHook(() => usePinnedSettings('quiz'))
    await waitFor(() => expect(result.current.keys).toHaveLength(1))

    act(() => {
      result.current.unpin('quiz.presentation.lockdown-mode')
    })
    expect(result.current.keys).toEqual([])
    expect(result.current.forceOpenSection).toBe('presentation')
  })
})

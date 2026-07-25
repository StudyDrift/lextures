import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  fetchPinnedSettings,
  fetchPinnedSettingsDetailed,
  savePinnedSettings,
} from '../pinned-settings-api'

vi.mock('../api', () => ({
  authorizedFetch: vi.fn(),
}))

import { authorizedFetch } from '../api'

const mockFetch = vi.mocked(authorizedFetch)

afterEach(() => {
  mockFetch.mockReset()
})

describe('fetchPinnedSettings', () => {
  it('returns parsed surfaces on 200', async () => {
    mockFetch.mockResolvedValue(
      new Response(
        JSON.stringify({
          surfaces: {
            assignment: ['assignment.scheduling.due-date'],
            quiz: ['quiz.presentation.lockdown-mode'],
          },
        }),
        { status: 200 },
      ),
    )
    const out = await fetchPinnedSettings()
    expect(out.assignment).toEqual(['assignment.scheduling.due-date'])
    expect(out.quiz).toEqual(['quiz.presentation.lockdown-mode'])
  })

  it('detailed fetch reports ok:true on success', async () => {
    mockFetch.mockResolvedValue(
      new Response(
        JSON.stringify({ surfaces: { assignment: [], quiz: ['quiz.scheduling.due-date'] } }),
        { status: 200 },
      ),
    )
    const out = await fetchPinnedSettingsDetailed()
    expect(out.ok).toBe(true)
    expect(out.surfaces.quiz).toEqual(['quiz.scheduling.due-date'])
  })

  it('detailed fetch reports ok:false on error', async () => {
    mockFetch.mockResolvedValue(new Response('nope', { status: 500 }))
    const out = await fetchPinnedSettingsDetailed()
    expect(out.ok).toBe(false)
    expect(out.surfaces).toEqual({ assignment: [], quiz: [] })
  })

  it('degrades to empty lists on 500 (AC-12)', async () => {
    mockFetch.mockResolvedValue(new Response('boom', { status: 500 }))
    const out = await fetchPinnedSettings()
    expect(out).toEqual({ assignment: [], quiz: [] })
  })

  it('degrades to empty lists on 404 (flag off)', async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'NOT_FOUND', message: 'off' } }), {
        status: 404,
      }),
    )
    const out = await fetchPinnedSettings()
    expect(out).toEqual({ assignment: [], quiz: [] })
  })

  it('degrades to empty lists on parse failure', async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify({ bad: true }), { status: 200 }))
    const out = await fetchPinnedSettings()
    expect(out).toEqual({ assignment: [], quiz: [] })
  })

  it('degrades to empty lists on network error', async () => {
    mockFetch.mockRejectedValue(new Error('network'))
    const out = await fetchPinnedSettings()
    expect(out).toEqual({ assignment: [], quiz: [] })
  })
})

describe('savePinnedSettings', () => {
  it('returns surfaces on success', async () => {
    mockFetch.mockResolvedValue(
      new Response(
        JSON.stringify({
          surfaces: { assignment: [], quiz: ['quiz.scheduling.due-date'] },
        }),
        { status: 200 },
      ),
    )
    const out = await savePinnedSettings('quiz', ['quiz.scheduling.due-date'])
    expect(out.quiz).toEqual(['quiz.scheduling.due-date'])
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/me/pinned-settings/quiz',
      expect.objectContaining({ method: 'PUT' }),
    )
  })

  it('rejects on non-2xx so caller can roll back', async () => {
    mockFetch.mockResolvedValue(
      new Response(
        JSON.stringify({ error: { code: 'INVALID_INPUT', message: 'too many' } }),
        { status: 400 },
      ),
    )
    await expect(savePinnedSettings('quiz', ['a.b.c'])).rejects.toThrow(/too many|Could not save/)
  })
})

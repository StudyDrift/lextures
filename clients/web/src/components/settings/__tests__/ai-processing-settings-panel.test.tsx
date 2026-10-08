import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AI_SETTINGS_PATH } from '../../../lib/ai-disclosure-i18n'
import { AiProcessingSettingsPanel } from '../ai-processing-settings-panel'

vi.mock('../../../lib/api', () => ({
  authorizedFetch: (input: string, init?: RequestInit) => globalThis.fetch(input, init),
}))

vi.mock('../../../lib/lms-toast', () => ({
  toastSaveOk: vi.fn(),
  toastMutationError: vi.fn(),
}))

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

describe('AiProcessingSettingsPanel', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('points "Manage AI settings" links at the account AI block', () => {
    expect(AI_SETTINGS_PATH).toBe('/settings/account#ai-settings')
  })

  it('shows the AI tutor opt-out and saves it', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input)
      if (url.endsWith('/ai-opt-out')) return json({ aiProcessingOptOut: false })
      if (url.endsWith('/ai-tutor-opt-out')) {
        if (init?.method === 'PUT') return json(JSON.parse(String(init.body)))
        return json({ aiTutorOptOut: false })
      }
      return json({}, 404)
    })
    const scrollIntoView = vi
      .spyOn(window.HTMLElement.prototype, 'scrollIntoView')
      .mockImplementation(() => {})

    render(
      <MemoryRouter initialEntries={[AI_SETTINGS_PATH]}>
        <AiProcessingSettingsPanel embedded />
      </MemoryRouter>,
    )

    const tutor = await screen.findByRole('checkbox', { name: /Turn off the AI tutor/i })
    expect(tutor).not.toBeChecked()
    expect(document.getElementById('ai-settings')).not.toBeNull()
    await waitFor(() => expect(scrollIntoView).toHaveBeenCalled())

    await userEvent.click(tutor)
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      const put = fetchMock.mock.calls.find(
        ([u, i]) => String(u).endsWith('/ai-tutor-opt-out') && i?.method === 'PUT',
      )
      expect(put).toBeTruthy()
      expect(JSON.parse(String(put?.[1]?.body))).toEqual({ aiTutorOptOut: true })
    })
  })

  it('hides the AI tutor opt-out when the tutor is unavailable', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.endsWith('/ai-opt-out')) return json({ aiProcessingOptOut: true })
      return json({ error: 'not found' }, 404)
    })

    render(
      <MemoryRouter>
        <AiProcessingSettingsPanel embedded />
      </MemoryRouter>,
    )

    expect(await screen.findByRole('checkbox', { name: /Opt out of AI processing/i })).toBeChecked()
    expect(screen.queryByRole('checkbox', { name: /Turn off the AI tutor/i })).toBeNull()
  })
})

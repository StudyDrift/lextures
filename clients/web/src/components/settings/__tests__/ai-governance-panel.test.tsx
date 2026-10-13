import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AiGovernancePanel } from '../ai-governance-panel'

vi.mock('../../../lib/api', () => ({
  authorizedFetch: (input: string, init?: RequestInit) => globalThis.fetch(input, init),
}))

vi.mock('../../../lib/lms-toast', () => ({
  toastSaveOk: vi.fn(),
  toastMutationError: vi.fn(),
}))

vi.mock('../adaptive-content-oversight-panel', () => ({
  AdaptiveContentOversightPanel: () => null,
}))

vi.mock('../../lms/adaptive-content/reports/admin-report-panel', () => ({
  AdminAdaptiveContentReportPanel: () => null,
}))

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function renderPanel() {
  return render(
    <MemoryRouter>
      <AiGovernancePanel />
    </MemoryRouter>,
  )
}

describe('AiGovernancePanel', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the loaded governance config', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async () =>
      json({ featuresEnabled: { ai_tutor: false }, allowedModels: ['gpt-4o'] }),
    )
    renderPanel()

    expect(await screen.findByRole('checkbox', { name: 'AI Tutor' })).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Notebook AI' })).toBeChecked()
    expect(screen.getByRole('textbox')).toHaveValue('gpt-4o')
    expect(screen.getByRole('button', { name: /save/i })).toBeEnabled()
  })

  it('shows an error instead of default toggles when the config fetch is forbidden', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockImplementation(async () => json({ error: { code: 'FORBIDDEN', message: 'nope' } }, 403))
    renderPanel()

    expect(await screen.findByRole('alert')).toHaveTextContent(/Could not load AI governance settings/i)
    expect(screen.queryByRole('checkbox')).toBeNull()
    expect(screen.queryByRole('button', { name: /save/i })).toBeNull()
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'PUT')).toBe(false)
  })

  it('shows the server error and retries on other failures', async () => {
    let calls = 0
    vi.spyOn(globalThis, 'fetch').mockImplementation(async () => {
      calls += 1
      if (calls === 1) return json({ error: { code: 'INTERNAL', message: 'Could not load AI config.' } }, 500)
      return json({ featuresEnabled: {}, allowedModels: null })
    })
    renderPanel()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Could not load AI config.')
    expect(screen.queryByRole('checkbox')).toBeNull()

    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() => expect(screen.getByRole('checkbox', { name: 'AI Tutor' })).toBeChecked())
    expect(screen.queryByRole('alert')).toBeNull()
  })
})

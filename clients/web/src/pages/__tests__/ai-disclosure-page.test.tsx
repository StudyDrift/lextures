import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi, afterEach } from 'vitest'
import AiDisclosurePage from '../ai-disclosure-page'

describe('AiDisclosurePage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders model cards from public API', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          version: '1',
          models: [{ id: 'm1', name: 'Test Model', provider: 'P', purposes: [], dataSent: 'x', retentionDays: 30, dpaStatus: 'ok', optOutPath: '/' }],
          features: [],
        }),
      }),
    )
    render(
      <MemoryRouter>
        <AiDisclosurePage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByText('Test Model')).toBeInTheDocument()
    })
  })
})

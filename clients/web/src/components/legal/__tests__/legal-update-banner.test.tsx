import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { LegalUpdateBanner } from '../legal-update-banner'
import * as legalApi from '../../../lib/legal-api'

describe('LegalUpdateBanner', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders nothing when no documents are pending', async () => {
    vi.spyOn(legalApi, 'fetchPendingLegalDocuments').mockResolvedValue([])
    const { container } = render(
      <MemoryRouter>
        <LegalUpdateBanner />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(container).toBeEmptyDOMElement()
    })
  })

  it('shows banner and acknowledges all pending documents', async () => {
    vi.spyOn(legalApi, 'fetchPendingLegalDocuments').mockResolvedValue([
      { document: 'privacy_policy', version: '2026-05-21', effectiveDate: '2026-05-21' },
    ])
    const ack = vi.spyOn(legalApi, 'acknowledgeLegalDocument').mockResolvedValue()
    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <LegalUpdateBanner />
      </MemoryRouter>,
    )

    await screen.findByRole('region', { name: /legal policy update/i })
    await user.click(screen.getByRole('button', { name: /i acknowledge/i }))

    await waitFor(() => {
      expect(ack).toHaveBeenCalledWith('privacy_policy', '2026-05-21')
    })
    await waitFor(() => {
      expect(screen.queryByRole('region', { name: /legal policy update/i })).not.toBeInTheDocument()
    })
  })
})

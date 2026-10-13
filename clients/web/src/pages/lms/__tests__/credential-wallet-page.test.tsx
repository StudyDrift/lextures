import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ loading: false, ffTranscripts: true }),
}))
vi.mock('../../../lib/wallet-api', () => ({
  fetchWallet: vi.fn(async () => ({ items: [] })),
  fetchWalletCollections: vi.fn(async () => []),
  createWalletCollection: vi.fn(),
  deleteWalletCollection: vi.fn(),
  downloadWalletExport: vi.fn(),
  fetchWalletCollectionAccess: vi.fn(async () => []),
  fetchWalletExport: vi.fn(),
  revokeWalletCollection: vi.fn(),
  startWalletExport: vi.fn(),
}))

import CredentialWalletPage from '../credential-wallet-page'

describe('CredentialWalletPage', () => {
  it('renders the "Credential wallet" heading only once (#787)', async () => {
    render(
      <MemoryRouter>
        <CredentialWalletPage />
      </MemoryRouter>,
    )
    expect(await screen.findByText(/No credentials yet/)).toBeInTheDocument()
    expect(screen.getAllByRole('heading', { name: 'Credential wallet' })).toHaveLength(1)
    expect(screen.getAllByText('Credential wallet')).toHaveLength(1)
  })
})

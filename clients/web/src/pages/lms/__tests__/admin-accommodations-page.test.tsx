import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import AdminAccommodationsPage from '../admin-accommodations-page'

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: () => false,
    loading: false,
  }),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    accommodationsEngineEnabled: false,
    loading: false,
  }),
}))

describe('AdminAccommodationsPage', () => {
  it('renders permission message when user cannot manage accommodations', () => {
    render(
      <MemoryRouter>
        <AdminAccommodationsPage />
      </MemoryRouter>,
    )
    expect(screen.getByText(/accessibility coordinator/i)).toBeInTheDocument()
  })
})

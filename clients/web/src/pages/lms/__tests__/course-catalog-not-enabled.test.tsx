import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

const perms = vi.hoisted(() => ({ admin: false }))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    ffCatalogIntegration: false,
    ffConsortiumSharing: false,
    ffCourseMarketplace: true,
  }),
}))
vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({ allows: () => perms.admin, loading: false }),
}))

import CourseCatalogPage from '../course-catalog'

function renderPage() {
  return render(
    <MemoryRouter>
      <CourseCatalogPage />
    </MemoryRouter>,
  )
}

describe('CourseCatalogPage when catalog is not enabled', () => {
  it('shows a friendly empty state with a marketplace link to non-admins', () => {
    perms.admin = false
    renderPage()
    expect(screen.queryByText(/Global platform/i)).toBeNull()
    expect(screen.getByRole('link', { name: /browse the marketplace/i })).toHaveAttribute('href', '/marketplace')
  })

  it('keeps the setup instructions for admins', () => {
    perms.admin = true
    renderPage()
    expect(screen.getByText(/Settings → Global platform/i)).toBeTruthy()
  })
})

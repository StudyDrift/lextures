import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PERM_REPORTS_VIEW } from '../../../lib/rbac-api'
import { ShellNavProvider } from '../shell-nav-context'
import { SideNavMainLinks } from '../side-nav-main-links'

const platformFeaturesMock = vi.fn(() => ({
  accommodationsEngineEnabled: false,
  ffEportfolio: false,
}))

vi.mock('../../../context/use-inbox-unread', () => ({
  useInboxUnreadCount: () => 2,
}))

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: (p: string) => p === PERM_REPORTS_VIEW,
    loading: false,
  }),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => platformFeaturesMock(),
}))

describe('SideNavMainLinks', () => {
  beforeEach(() => {
    platformFeaturesMock.mockReturnValue({
      accommodationsEngineEnabled: false,
      ffEportfolio: false,
    })
  })

  it('renders core navigation and unread badge when inbox has items', () => {
    render(
      <MemoryRouter>
        <ShellNavProvider>
          <SideNavMainLinks />
        </ShellNavProvider>
      </MemoryRouter>,
    )
    expect(screen.getByRole('link', { name: /^dashboard$/i })).toHaveAttribute('href', '/')
    expect(screen.getByRole('link', { name: /^courses$/i })).toHaveAttribute('href', '/courses')
    expect(screen.getByLabelText('2 unread')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /^reports$/i })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /^my portfolio$/i })).not.toBeInTheDocument()
  })

  it('shows My Portfolio when ePortfolio is enabled', () => {
    platformFeaturesMock.mockReturnValue({
      accommodationsEngineEnabled: false,
      ffEportfolio: true,
    })

    render(
      <MemoryRouter>
        <ShellNavProvider>
          <SideNavMainLinks />
        </ShellNavProvider>
      </MemoryRouter>,
    )

    expect(screen.getByRole('link', { name: /^my portfolio$/i })).toHaveAttribute('href', '/portfolios')
  })
})

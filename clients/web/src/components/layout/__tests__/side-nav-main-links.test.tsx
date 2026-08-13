import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PERM_MARKETING_CONTENT_VIEW, PERM_REPORTS_VIEW } from '../../../lib/rbac-api'
import { ShellNavProvider } from '../shell-nav-context'
import { SideNavMainLinks } from '../side-nav-main-links'

const allowsMock = vi.fn((p: string) => p === PERM_REPORTS_VIEW)

const platformFeaturesMock = vi.fn(() => ({
  accommodationsEngineEnabled: false,
  ffEportfolio: false,
  ragNotebookEnabled: true,
  ffCourseMarketplace: false,
  ffMarketingContent: false,
}))

vi.mock('../../../context/use-inbox-unread', () => ({
  useInboxUnreadCount: () => 2,
}))

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: (p: string) => allowsMock(p),
    loading: false,
  }),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => platformFeaturesMock(),
}))

describe('SideNavMainLinks', () => {
  beforeEach(() => {
    allowsMock.mockImplementation((p: string) => p === PERM_REPORTS_VIEW)
    platformFeaturesMock.mockReturnValue({
      accommodationsEngineEnabled: false,
      ffEportfolio: false,
      ragNotebookEnabled: true,
      ffCourseMarketplace: false,
      ffMarketingContent: false,
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
    expect(screen.getByRole('link', { name: /^ask ai$/i })).toHaveAttribute('href', '/ai')
    expect(screen.getByLabelText('2 unread')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /^reports$/i })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /^my portfolio$/i })).not.toBeInTheDocument()
  })

  it('hides Ask AI when notebook AI is disabled', () => {
    platformFeaturesMock.mockReturnValue({
      accommodationsEngineEnabled: false,
      ffEportfolio: false,
      ragNotebookEnabled: false,
      ffCourseMarketplace: false,
      ffMarketingContent: false,
    })

    render(
      <MemoryRouter>
        <ShellNavProvider>
          <SideNavMainLinks />
        </ShellNavProvider>
      </MemoryRouter>,
    )

    expect(screen.queryByRole('link', { name: /^ask ai$/i })).not.toBeInTheDocument()
  })

  it('shows My Portfolio when ePortfolio is enabled', () => {
    platformFeaturesMock.mockReturnValue({
      accommodationsEngineEnabled: false,
      ffEportfolio: true,
      ragNotebookEnabled: true,
      ffCourseMarketplace: false,
      ffMarketingContent: false,
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

  it('shows Marketplace when course marketplace is enabled', () => {
    platformFeaturesMock.mockReturnValue({
      accommodationsEngineEnabled: false,
      ffEportfolio: false,
      ragNotebookEnabled: true,
      ffCourseMarketplace: true,
      ffMarketingContent: false,
    })

    render(
      <MemoryRouter>
        <ShellNavProvider>
          <SideNavMainLinks />
        </ShellNavProvider>
      </MemoryRouter>,
    )

    expect(screen.getByRole('link', { name: /^marketplace$/i })).toHaveAttribute('href', '/marketplace')
    expect(screen.getByRole('link', { name: /^my purchases$/i })).toHaveAttribute('href', '/me/purchases')
  })

  it('hides Marketplace when course marketplace is disabled', () => {
    platformFeaturesMock.mockReturnValue({
      accommodationsEngineEnabled: false,
      ffEportfolio: false,
      ragNotebookEnabled: true,
      ffCourseMarketplace: false,
      ffMarketingContent: false,
    })

    render(
      <MemoryRouter>
        <ShellNavProvider>
          <SideNavMainLinks />
        </ShellNavProvider>
      </MemoryRouter>,
    )

    expect(screen.queryByRole('link', { name: /^marketplace$/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /^my purchases$/i })).not.toBeInTheDocument()
  })

  it('shows Marketing Content when the workspace flag and view permission are on', () => {
    allowsMock.mockImplementation((p: string) => p === PERM_MARKETING_CONTENT_VIEW)
    platformFeaturesMock.mockReturnValue({
      accommodationsEngineEnabled: false,
      ffEportfolio: false,
      ragNotebookEnabled: true,
      ffCourseMarketplace: false,
      ffMarketingContent: true,
    })

    render(
      <MemoryRouter>
        <ShellNavProvider>
          <SideNavMainLinks />
        </ShellNavProvider>
      </MemoryRouter>,
    )

    expect(screen.getByRole('link', { name: /^marketing content$/i })).toHaveAttribute(
      'href',
      '/admin/marketing-content',
    )
  })

  it('hides Marketing Content when the workspace flag is on but the viewer lacks permission', () => {
    platformFeaturesMock.mockReturnValue({
      accommodationsEngineEnabled: false,
      ffEportfolio: false,
      ragNotebookEnabled: true,
      ffCourseMarketplace: false,
      ffMarketingContent: true,
    })

    render(
      <MemoryRouter>
        <ShellNavProvider>
          <SideNavMainLinks />
        </ShellNavProvider>
      </MemoryRouter>,
    )

    expect(screen.queryByRole('link', { name: /^marketing content$/i })).not.toBeInTheDocument()
  })
})

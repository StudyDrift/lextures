import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { IntroCourseCard } from '../intro-course-card'
import * as hook from '../../../hooks/use-intro-course-progress'

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ introCourseEnabled: true, loading: false }),
}))

vi.mock('../../../hooks/use-intro-course-progress')

function renderCard() {
  return render(
    <MemoryRouter>
      <IntroCourseCard />
    </MemoryRouter>,
  )
}

describe('IntroCourseCard', () => {
  beforeEach(() => {
    vi.mocked(hook.useIntroCourseProgress).mockReset()
  })

  it('renders nothing when not enrolled', () => {
    vi.mocked(hook.useIntroCourseProgress).mockReturnValue({
      progress: {
        enrolled: false,
        modulesComplete: 0,
        modulesTotal: 7,
        percent: 0,
      },
      loading: false,
      error: false,
      refresh: vi.fn(),
    })
    const { container } = renderCard()
    expect(container).toBeEmptyDOMElement()
  })

  it('renders start-here state with progress and CTA', async () => {
    vi.mocked(hook.useIntroCourseProgress).mockReturnValue({
      progress: {
        enrolled: true,
        courseCode: 'C-WLCOME',
        modulesComplete: 0,
        modulesTotal: 7,
        percent: 0,
        nextItem: {
          slug: 'm1.welcome.dashboard',
          title: 'Dashboard tour',
          route: '/courses/C-WLCOME/modules/content/abc',
        },
      },
      loading: false,
      error: false,
      refresh: vi.fn(),
    })
    renderCard()
    expect(await screen.findByText('Start here')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /start the intro course/i })).toHaveAttribute(
      'href',
      '/courses/C-WLCOME/modules/content/abc',
    )
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '0')
  })

  it('renders fallback link on error', async () => {
    vi.mocked(hook.useIntroCourseProgress).mockReturnValue({
      progress: null,
      loading: false,
      error: true,
      refresh: vi.fn(),
    })
    renderCard()
    expect(await screen.findByRole('link', { name: /open the intro course/i })).toHaveAttribute(
      'href',
      '/courses/C-WLCOME',
    )
  })

  it('demotes to compact completed state', async () => {
    vi.mocked(hook.useIntroCourseProgress).mockReturnValue({
      progress: {
        enrolled: true,
        courseCode: 'C-WLCOME',
        modulesComplete: 7,
        modulesTotal: 7,
        percent: 100,
        completedAt: '2026-01-01T00:00:00Z',
      },
      loading: false,
      error: false,
      refresh: vi.fn(),
    })
    renderCard()
    expect(await screen.findByText('Onboarding complete')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /revisit/i })).toBeInTheDocument()
  })
})
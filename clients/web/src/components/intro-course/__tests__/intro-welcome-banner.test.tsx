import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { IntroWelcomeBanner } from '../intro-welcome-banner'
import * as api from '../../../lib/intro-course-api'
import * as hook from '../../../hooks/use-intro-course-progress'

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ introCourseEnabled: true, loading: false }),
}))

vi.mock('../../../hooks/use-intro-course-progress')
vi.mock('../../../lib/intro-course-api', async (importOriginal) => {
  const actual = await importOriginal<typeof api>()
  return {
    ...actual,
    dismissIntroWelcomeBanner: vi.fn().mockResolvedValue(undefined),
  }
})

describe('IntroWelcomeBanner', () => {
  beforeEach(() => {
    vi.mocked(hook.useIntroCourseProgress).mockReset()
    vi.mocked(api.dismissIntroWelcomeBanner).mockClear()
  })

  it('shows for first-login enrolled learners', async () => {
    vi.mocked(hook.useIntroCourseProgress).mockReturnValue({
      progress: {
        enrolled: true,
        modulesComplete: 0,
        modulesTotal: 7,
        percent: 0,
        nextItem: {
          slug: 'm1',
          title: 'Welcome',
          route: '/courses/C-WLCOME',
        },
      },
      loading: false,
      error: false,
      refresh: vi.fn(),
    })
    render(
      <MemoryRouter>
        <IntroWelcomeBanner />
      </MemoryRouter>,
    )
    expect(await screen.findByText(/welcome — start with the guided intro course/i)).toBeInTheDocument()
  })

  it('persists dismissal server-side', async () => {
    const refresh = vi.fn()
    vi.mocked(hook.useIntroCourseProgress).mockReturnValue({
      progress: {
        enrolled: true,
        modulesComplete: 0,
        modulesTotal: 7,
        percent: 0,
      },
      loading: false,
      error: false,
      refresh,
    })
    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <IntroWelcomeBanner />
      </MemoryRouter>,
    )
    await user.click(await screen.findByRole('button', { name: /dismiss welcome banner/i }))
    expect(api.dismissIntroWelcomeBanner).toHaveBeenCalled()
    expect(refresh).toHaveBeenCalled()
  })
})
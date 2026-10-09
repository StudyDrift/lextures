import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ImpersonationChrome } from '../impersonation-chrome'

const authMock = vi.hoisted(() => ({ token: null as string | null }))
const profileMock = vi.hoisted(() => vi.fn())

vi.mock('../../lib/auth', async (orig) => ({
  ...(await orig<typeof import('../../lib/auth')>()),
  getImpersonationToken: () => authMock.token,
}))

vi.mock('../../lib/impersonation', async (orig) => ({
  ...(await orig<typeof import('../../lib/impersonation')>()),
  fetchMeProfile: () => profileMock(),
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (_k: string, o?: { defaultValue?: string }) => o?.defaultValue ?? _k }),
}))

function renderChrome() {
  return render(
    <MemoryRouter>
      <ImpersonationChrome shellClassName="flex min-h-0 flex-1">
        <header>Header controls</header>
      </ImpersonationChrome>
    </MemoryRouter>,
  )
}

describe('ImpersonationChrome', () => {
  beforeEach(() => {
    authMock.token = null
    profileMock.mockReset()
  })

  it('renders only the shell when nobody is being impersonated', () => {
    const { container } = renderChrome()
    expect(screen.getByText('Header controls')).toBeInTheDocument()
    expect(container.querySelector('[role="status"]')).toBeNull()
  })

  it('keeps the Learn as banner in normal flow so it cannot cover the header', async () => {
    authMock.token = 'learner-token'
    profileMock.mockResolvedValue({ learningAs: true, displayName: 'QA Test Kid', email: 'kid@example.com' })
    renderChrome()
    const banner = await waitFor(() => screen.getByRole('status'))
    expect(banner).toHaveTextContent('Learning as QA Test Kid')
    expect(banner.className).not.toMatch(/\bfixed\b/)
    expect(banner.className).toMatch(/\bflex-wrap\b/)
    // Banner and shell are siblings inside one column so the shell is sized below the banner.
    const column = banner.parentElement as HTMLElement
    expect(column.className).toMatch(/\bflex-col\b/)
    expect(column).toContainElement(screen.getByText('Header controls'))
  })
})

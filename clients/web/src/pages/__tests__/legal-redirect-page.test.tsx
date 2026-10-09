import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from '../../app'
import { PermissionsProvider } from '../../context/permissions-provider'

describe('public /terms and /privacy routes', () => {
  const originalLocation = window.location

  afterEach(() => {
    Object.defineProperty(window, 'location', { value: originalLocation, configurable: true })
  })

  it.each([
    ['/terms', 'https://lextures.com/terms'],
    ['/privacy', 'https://lextures.com/privacy'],
  ])('redirects signed-out %s to the marketing site instead of /login', async (route, target) => {
    const replace = vi.fn()
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, replace },
      configurable: true,
    })
    render(
      <MemoryRouter initialEntries={[route]}>
        <PermissionsProvider>
          <App />
        </PermissionsProvider>
      </MemoryRouter>,
    )
    expect(await screen.findByRole('link', { name: target })).toHaveAttribute('href', target)
    expect(replace).toHaveBeenCalledWith(target)
  })
})

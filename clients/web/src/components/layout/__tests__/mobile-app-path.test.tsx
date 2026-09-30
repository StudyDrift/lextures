import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { IOS_APP_STORE_URL } from '../../../lib/mobile-apps'
import { ShellNavProvider } from '../shell-nav-context'
import { iosAppAccountMenuItem } from '../ios-app-account-menu-item'
import { MobileAppPath } from '../mobile-app-path'

describe('MobileAppPath', () => {
  it('links the App Store and notes that the browser works on a phone or tablet', () => {
    render(
      <ShellNavProvider>
        <MobileAppPath collapsed={false} />
      </ShellNavProvider>,
    )

    const link = screen.getByRole('link', { name: /get the ios app/i })
    expect(link).toHaveAttribute('href', IOS_APP_STORE_URL)
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveTextContent('App Store')
    expect(screen.getByText('Kids can learn on a phone or tablet in the browser.')).toBeInTheDocument()
    expect(screen.queryByText(/android|play store/i)).not.toBeInTheDocument()
  })

  it('keeps an accessible App Store link when the sidebar is collapsed', () => {
    render(
      <ShellNavProvider>
        <MobileAppPath collapsed />
      </ShellNavProvider>,
    )

    const link = screen.getByRole('link', { name: /get the ios app on the app store/i })
    expect(link).toHaveAttribute('href', IOS_APP_STORE_URL)
    expect(link).toHaveAccessibleName(/phone or tablet in the browser/i)
  })
})

describe('iosAppAccountMenuItem', () => {
  it('opens the App Store listing from the account menu', () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)
    const item = iosAppAccountMenuItem()
    render(<>{item.label}</>)

    expect(screen.getByText('Get the iOS app')).toBeInTheDocument()
    expect(screen.getByText('Kids can learn on a phone or tablet in the browser.')).toBeInTheDocument()
    item.onSelect?.()
    expect(open).toHaveBeenCalledWith(IOS_APP_STORE_URL, '_blank', 'noopener,noreferrer')
    open.mockRestore()
  })
})

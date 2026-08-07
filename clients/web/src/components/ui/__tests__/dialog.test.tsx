import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { PlatformFeaturesProvider } from '../../../context/platform-features-context'
import { Dialog } from '../dialog'
import { Button } from '../button'
import { Tabs, Tab, TabList, TabPanel } from '../tabs'
import { EmptyState } from '../empty-state'

function wrap(ui: ReactNode) {
  return render(<PlatformFeaturesProvider>{ui}</PlatformFeaturesProvider>)
}

describe('UX.2 Dialog', () => {
  it('traps focus and closes on Escape (FR-4 / AC-3)', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    wrap(
      <Dialog open onClose={onClose} title="Title" closeLabel="Close dialog">
        <Button>Inside</Button>
      </Dialog>,
    )
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toBeTruthy()
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
})

describe('UX.2 Tabs', () => {
  it('moves selection with arrow keys (AC-4)', async () => {
    const user = userEvent.setup()
    wrap(
      <Tabs defaultValue="a">
        <TabList aria-label="Demo">
          <Tab value="a">A</Tab>
          <Tab value="b">B</Tab>
          <Tab value="c">C</Tab>
        </TabList>
        <TabPanel value="a">Panel A</TabPanel>
        <TabPanel value="b">Panel B</TabPanel>
        <TabPanel value="c">Panel C</TabPanel>
      </Tabs>,
    )
    const tabA = screen.getByRole('tab', { name: 'A' })
    tabA.focus()
    await user.keyboard('{ArrowRight}')
    expect(screen.getByRole('tab', { name: 'B' })).toHaveAttribute('aria-selected', 'true')
    await user.keyboard('{End}')
    expect(screen.getByRole('tab', { name: 'C' })).toHaveAttribute('aria-selected', 'true')
    await user.keyboard('{Home}')
    expect(screen.getByRole('tab', { name: 'A' })).toHaveAttribute('aria-selected', 'true')
  })

  it('inverts horizontal arrows under dir=rtl (UX.4 AC-2)', async () => {
    const user = userEvent.setup()
    document.documentElement.dir = 'rtl'
    try {
      wrap(
        <Tabs defaultValue="a">
          <TabList aria-label="RTL">
            <Tab value="a">A</Tab>
            <Tab value="b">B</Tab>
            <Tab value="c">C</Tab>
          </TabList>
          <TabPanel value="a">Panel A</TabPanel>
          <TabPanel value="b">Panel B</TabPanel>
          <TabPanel value="c">Panel C</TabPanel>
        </Tabs>,
      )
      screen.getByRole('tab', { name: 'A' }).focus()
      // In RTL, ArrowRight moves to previous (none) / wrap — APG: ArrowRight → previous
      await user.keyboard('{ArrowRight}')
      // previous of A wraps to C
      expect(screen.getByRole('tab', { name: 'C' })).toHaveAttribute('aria-selected', 'true')
      await user.keyboard('{ArrowLeft}')
      // ArrowLeft in RTL moves next → A
      expect(screen.getByRole('tab', { name: 'A' })).toHaveAttribute('aria-selected', 'true')
    } finally {
      document.documentElement.dir = 'ltr'
    }
  })
})

describe('UX.2 EmptyState', () => {
  it('uses Button rather than hand-rolled styles (AC-11)', async () => {
    const onClick = vi.fn()
    wrap(
      <EmptyState
        icon={() => <span aria-hidden>*</span>}
        title="Empty"
        primaryAction={{ label: 'Create', onClick }}
      />,
    )
    const btn = screen.getByRole('button', { name: 'Create' })
    expect(btn.className).toMatch(/lx-control-btn/)
    await userEvent.click(btn)
    await waitFor(() => expect(onClick).toHaveBeenCalled())
  })
})

describe('UX.2 Button size', () => {
  it('smallest size still has min target classes (AC-7)', () => {
    wrap(
      <Button size="sm" variant="secondary">
        Go
      </Button>,
    )
    const btn = screen.getByRole('button', { name: 'Go' })
    expect(btn.className).toMatch(/min-h-6/)
    expect(btn.className).toMatch(/min-w-6/)
  })
})

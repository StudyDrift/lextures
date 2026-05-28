import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { PermissionsProvider } from '../../../context/permissions-provider'
import { CommandPaletteProvider } from '../command-palette-provider'

function renderWithPalette() {
  return render(
    <MemoryRouter>
      <PermissionsProvider>
        <CommandPaletteProvider>
          <button type="button" id="trigger">Open palette</button>
        </CommandPaletteProvider>
      </PermissionsProvider>
    </MemoryRouter>,
  )
}

describe('CommandPaletteDialog — accessibility', () => {
  it('dialog is not present when closed', () => {
    renderWithPalette()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('opens via Cmd+K and renders a dialog with correct role/label', async () => {
    renderWithPalette()
    await userEvent.keyboard('{Meta>}k{/Meta}')
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true')
    expect(screen.getByRole('dialog')).toHaveAttribute('aria-label', 'Command Palette')
  })

  it('search input has an accessible label', async () => {
    renderWithPalette()
    await userEvent.keyboard('{Meta>}k{/Meta}')
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    const input = screen.getByRole('searchbox', { name: /search/i })
    expect(input).toBeInTheDocument()
  })

  it('results listbox is present', async () => {
    renderWithPalette()
    await userEvent.keyboard('{Meta>}k{/Meta}')
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    expect(screen.getByRole('listbox', { name: /results/i })).toBeInTheDocument()
  })

  it('closes on Escape', async () => {
    renderWithPalette()
    await userEvent.keyboard('{Meta>}k{/Meta}')
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('live region exists for result count announcements', async () => {
    renderWithPalette()
    await userEvent.keyboard('{Meta>}k{/Meta}')
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    // The live status region should be present (sr-only, but in the DOM).
    const statuses = screen.getAllByRole('status')
    expect(statuses.length).toBeGreaterThan(0)
    const liveRegion = statuses.find((el) => el.getAttribute('aria-live') === 'polite')
    expect(liveRegion).toBeDefined()
  })
})

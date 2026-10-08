import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PERM_MARKETING_CONTENT_VIEW } from '../../../lib/rbac-api'
import { CommandPaletteProvider } from '../command-palette-provider'

const fetchSearchIndex = vi.fn()
const fetchSearchQuery = vi.fn()

vi.mock('../../../lib/search-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/search-api')>('../../../lib/search-api')
  return {
    ...actual,
    fetchSearchIndex: (...args: unknown[]) => fetchSearchIndex(...args),
    fetchSearchQuery: (...args: unknown[]) => fetchSearchQuery(...args),
  }
})

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: (permission: string) => permission === PERM_MARKETING_CONTENT_VIEW,
    loading: false,
  }),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    ffMarketingContent: true,
    ragNotebookEnabled: false,
    ffMotionOverlays: false,
  }),
}))

function renderPalette() {
  return render(
    <MemoryRouter>
      <CommandPaletteProvider>
        <button type="button">Open palette</button>
      </CommandPaletteProvider>
    </MemoryRouter>,
  )
}

async function openPalette() {
  renderPalette()
  await userEvent.keyboard('{Meta>}k{/Meta}')
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
  await waitFor(() => expect(screen.queryByText('Loading…')).not.toBeInTheDocument())
}

describe('Command palette Marketing Content', () => {
  beforeEach(() => {
    fetchSearchIndex.mockResolvedValue({
      courses: [{ courseCode: 'HEB101', title: 'Introduction to Ancient Hebrew' }],
      people: [],
    })
    fetchSearchQuery.mockResolvedValue({ groups: [], tookMs: 1 })
  })

  it('keeps Marketing Content in the empty-query hub', async () => {
    await openPalette()
    expect(screen.getByRole('option', { name: /marketing content/i })).toBeInTheDocument()
  })

  it('shows Marketing Content when the query matches its keywords', async () => {
    await openPalette()
    await userEvent.type(screen.getByRole('searchbox', { name: /search/i }), 'blog')
    expect(await screen.findByRole('option', { name: /marketing content/i })).toBeInTheDocument()
  })

  it('omits Marketing Content for a query that matches nothing', async () => {
    await openPalette()
    await userEvent.type(screen.getByRole('searchbox', { name: /search/i }), 'zzqx')
    expect(await screen.findByText('No results.')).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: /marketing content/i })).not.toBeInTheDocument()
  })

  it('does not pin Marketing Content ahead of an unrelated match', async () => {
    await openPalette()
    await userEvent.type(screen.getByRole('searchbox', { name: /search/i }), 'Hebrew')
    const matches = await screen.findAllByRole('option', { name: /introduction to ancient hebrew/i })
    expect(matches.some((option) => option.id === 'cmd-result-course:HEB101')).toBe(true)
    expect(screen.queryByRole('option', { name: /marketing content/i })).not.toBeInTheDocument()
  })
})

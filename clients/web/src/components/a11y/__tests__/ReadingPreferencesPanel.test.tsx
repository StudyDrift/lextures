import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { ReadingPreferencesPanel } from '../ReadingPreferencesPanel'
import { ReadingPreferencesContext } from '../../../context/reading-preferences-context'
import type { ReadingPreferences } from '../../../lib/reading-preferences'
import { defaultReadingPreferences } from '../../../lib/reading-preferences'

function makeContextValue(overrides: Partial<ReadingPreferences> = {}, update = vi.fn()) {
  return {
    prefs: { ...defaultReadingPreferences, ...overrides },
    loading: false,
    update,
  }
}

function renderPanel(
  props: { open: boolean; onClose: () => void },
  prefs: Partial<ReadingPreferences> = {},
  update = vi.fn(),
) {
  return render(
    <ReadingPreferencesContext.Provider value={makeContextValue(prefs, update)}>
      <ReadingPreferencesPanel {...props} />
    </ReadingPreferencesContext.Provider>,
  )
}

describe('ReadingPreferencesPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not render when closed', () => {
    renderPanel({ open: false, onClose: vi.fn() })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('renders dialog with correct role and label when open', () => {
    renderPanel({ open: true, onClose: vi.fn() })
    const dialog = screen.getByRole('dialog', { name: /reading preferences/i })
    expect(dialog).toBeInTheDocument()
    expect(dialog).toHaveAttribute('aria-modal', 'true')
  })

  it('shows heading "Reading Preferences"', () => {
    renderPanel({ open: true, onClose: vi.fn() })
    expect(screen.getByRole('heading', { name: /reading preferences/i })).toBeInTheDocument()
  })

  it('calls onClose when Escape is pressed', () => {
    const onClose = vi.fn()
    renderPanel({ open: true, onClose })
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls onClose when close button is clicked', () => {
    const onClose = vi.fn()
    renderPanel({ open: true, onClose })
    fireEvent.click(screen.getByRole('button', { name: /close reading preferences/i }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls update when a font radio is selected', () => {
    const update = vi.fn()
    renderPanel({ open: true, onClose: vi.fn() }, {}, update)
    const openDyslexicInput = screen.getByRole('radio', { name: /font: opendyslexic/i })
    fireEvent.click(openDyslexicInput)
    expect(update).toHaveBeenCalledWith({ fontFace: 'open-dyslexic' })
  })

  it('calls update when letter spacing is changed to Wide', () => {
    const update = vi.fn()
    renderPanel({ open: true, onClose: vi.fn() }, {}, update)
    const wideRadio = screen.getByRole('radio', { name: /letter spacing: wide$/i })
    fireEvent.click(wideRadio)
    expect(update).toHaveBeenCalledWith({ letterSpacing: 'wide' })
  })

  it('calls update when line height is changed to Taller', () => {
    const update = vi.fn()
    renderPanel({ open: true, onClose: vi.fn() }, {}, update)
    const tallerRadio = screen.getByRole('radio', { name: /line height: taller/i })
    fireEvent.click(tallerRadio)
    expect(update).toHaveBeenCalledWith({ lineHeight: 'taller' })
  })

  it('shows ruler toggle and calls update on click', () => {
    const update = vi.fn()
    renderPanel({ open: true, onClose: vi.fn() }, { rulerEnabled: false }, update)
    const toggle = screen.getByRole('switch', { name: /reading ruler/i })
    expect(toggle).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(toggle)
    expect(update).toHaveBeenCalledWith({ rulerEnabled: true })
  })

  it('shows ruler color options when ruler is enabled', () => {
    renderPanel({ open: true, onClose: vi.fn() }, { rulerEnabled: true })
    expect(screen.getByRole('radio', { name: /ruler colour: yellow tint/i })).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /ruler colour: grey tint/i })).toBeInTheDocument()
  })

  it('does not show ruler color options when ruler is disabled', () => {
    renderPanel({ open: true, onClose: vi.fn() }, { rulerEnabled: false })
    expect(screen.queryByRole('radio', { name: /ruler colour/i })).not.toBeInTheDocument()
  })

  it('shows skeleton loaders when loading', () => {
    render(
      <ReadingPreferencesContext.Provider value={{ prefs: defaultReadingPreferences, loading: true, update: vi.fn() }}>
        <ReadingPreferencesPanel open={true} onClose={vi.fn()} />
      </ReadingPreferencesContext.Provider>,
    )
    expect(screen.queryByRole('radio')).not.toBeInTheDocument()
  })

  it('current font is reflected in checked radio', () => {
    renderPanel({ open: true, onClose: vi.fn() }, { fontFace: 'atkinson' })
    const atkinsonRadio = screen.getByRole('radio', { name: /font: atkinson hyperlegible/i })
    expect(atkinsonRadio).toBeChecked()
    const defaultRadio = screen.getByRole('radio', { name: /font: default/i })
    expect(defaultRadio).not.toBeChecked()
  })
})

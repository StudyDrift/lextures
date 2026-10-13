import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const fetchLearnerSettings = vi.fn()
const updateLearnerSettings = vi.fn()
vi.mock('../../../lib/learner-settings-api', () => ({
  fetchLearnerSettings: () => fetchLearnerSettings(),
  updateLearnerSettings: (b: unknown) => updateLearnerSettings(b),
}))
vi.mock('../../../components/a11y/ReadingPreferencesPanel', () => ({
  ReadingPreferencesPanel: () => null,
}))

import KidSettingsPage from '../kid-settings'

describe('KidSettingsPage (#775)', () => {
  beforeEach(() => {
    fetchLearnerSettings.mockReset()
    updateLearnerSettings.mockReset()
    fetchLearnerSettings.mockResolvedValue({ displayName: 'Kiddo', avatarUrl: null, uiTheme: 'light' })
  })

  it('shows only the simplified sections', async () => {
    render(
      <MemoryRouter>
        <KidSettingsPage />
      </MemoryRouter>,
    )
    expect(await screen.findByDisplayValue('Kiddo')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /your name and picture/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /look and reading/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /sounds/i })).toBeInTheDocument()
    expect(screen.getByRole('switch', { name: /notification sounds/i })).toBeInTheDocument()
    const text = document.body.textContent?.toLowerCase() ?? ''
    for (const banned of ['email', 'password', 'privacy', 'ai provider', 'delete account', 'campus']) {
      expect(text).not.toContain(banned)
    }
  })
})

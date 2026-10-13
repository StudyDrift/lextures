import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import LearnersPage from '../learners-page'

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    loading: false,
    ffHomeschoolManagedLearners: true,
  }),
}))

vi.mock('../../../lib/managed-learners-api', () => ({
  listDependents: vi.fn().mockResolvedValue([]),
  listDependentCourses: vi.fn().mockResolvedValue([]),
  createDependent: vi.fn(),
  deleteDependent: vi.fn(),
  startLearnAs: vi.fn(),
}))

vi.mock('../../../lib/family-sign-in-api', () => ({
  listFamilyDevices: vi.fn().mockResolvedValue([]),
  registerThisFamilyDevice: vi.fn(),
  revokeFamilyDevice: vi.fn(),
}))

describe('Learners COPPA checkbox', () => {
  it('explains the recorded consent and AI limit', async () => {
    render(
      <MemoryRouter>
        <LearnersPage />
      </MemoryRouter>,
    )
    expect(await screen.findByLabelText('This learner is under 13 (COPPA)')).toBeInTheDocument()
    const help = document.getElementById('learner-under-13-help')
    expect(help?.textContent).toMatch(/recorded as the consenting parent/i)
    expect(help?.textContent).toMatch(/AI features stay off/i)
    expect(help?.textContent).toMatch(/adapted lessons wait for your approval/i)
    expect(screen.getByRole('link', { name: 'Learn more in the Trust Center' })).toHaveAttribute('href', '/trust')
  })
})

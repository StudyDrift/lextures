import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { I18nProvider } from '../../../context/i18n-provider'

const {
  fetchIntroCourseAdminStatus,
  fetchIntroCourseAdminAnalytics,
  resyncIntroCourse,
  startIntroCourseBackfill,
} = vi.hoisted(() => ({
  fetchIntroCourseAdminStatus: vi.fn(),
  fetchIntroCourseAdminAnalytics: vi.fn(),
  resyncIntroCourse: vi.fn(),
  startIntroCourseBackfill: vi.fn(),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    introCourseEnabled: true,
    loading: false,
    refresh: vi.fn(),
  }),
}))

vi.mock('../../../lib/intro-course-admin-api', () => ({
  fetchIntroCourseAdminStatus,
  fetchIntroCourseAdminAnalytics,
  resyncIntroCourse,
  startIntroCourseBackfill,
}))

vi.mock('../../../lib/api', () => ({
  authorizedFetch: vi.fn(),
}))

import { IntroCoursePanel } from '../intro-course-panel'

const mockStatus = {
  enabled: true,
  coursePresent: true,
  courseCode: 'C-WLCOME',
  contentVersion: 2,
  moduleCount: 7,
  availableLocales: ['en', 'es'],
  localeCoverage: { en: 1, es: 0.25 },
  backfill: { startedAt: null, completedAt: null, remaining: 3 },
}

const mockAnalytics = {
  enrolled: 12,
  completed: 5,
  completionRate: 5 / 12,
  perModuleFunnel: [
    { moduleSlug: 'm1.welcome', moduleTitle: 'Welcome', quizAttempted: 10, attemptRate: 10 / 12 },
  ],
  dropOffModuleSlug: 'm2.core-features',
  avgTimeToCompleteHours: 4.5,
}

function renderPanel() {
  return render(
    <I18nProvider>
      <IntroCoursePanel />
    </I18nProvider>,
  )
}

describe('IntroCoursePanel', () => {
  beforeEach(() => {
    fetchIntroCourseAdminStatus.mockReset()
    fetchIntroCourseAdminAnalytics.mockReset()
    resyncIntroCourse.mockReset()
    startIntroCourseBackfill.mockReset()
    fetchIntroCourseAdminStatus.mockResolvedValue(mockStatus)
    fetchIntroCourseAdminAnalytics.mockResolvedValue(mockAnalytics)
  })

  it('renders when perModuleFunnel is null from API', async () => {
    fetchIntroCourseAdminAnalytics.mockResolvedValue({
      ...mockAnalytics,
      perModuleFunnel: null,
    })
    renderPanel()
    await waitFor(() => {
      expect(screen.getByText('12')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /re-sync content/i })).toBeInTheDocument()
  })

  it('loads status and analytics', async () => {
    renderPanel()
    await waitFor(() => {
      expect(screen.getByText('7')).toBeInTheDocument()
    })
    expect(screen.getByText('12')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /re-sync content/i })).toBeInTheDocument()
  })

  it('confirms before re-sync', async () => {
    const user = userEvent.setup()
    renderPanel()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /re-sync content/i })).toBeEnabled()
    })
    await user.click(screen.getByRole('button', { name: /re-sync content/i }))
    expect(await screen.findByText(/re-sync intro course content/i)).toBeInTheDocument()
  })
})
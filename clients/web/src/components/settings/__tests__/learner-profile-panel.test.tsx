import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { I18nProvider } from '../../../context/i18n-provider'

const {
  fetchLearnerProfile,
  fetchLearnerProfileFacet,
  fetchLearnerProfileFacetEvidence,
  pauseLearnerProfile,
  resumeLearnerProfile,
  resetLearnerProfile,
  downloadLearnerProfileExport,
} = vi.hoisted(() => ({
  fetchLearnerProfile: vi.fn(),
  fetchLearnerProfileFacet: vi.fn(),
  fetchLearnerProfileFacetEvidence: vi.fn(),
  pauseLearnerProfile: vi.fn(),
  resumeLearnerProfile: vi.fn(),
  resetLearnerProfile: vi.fn(),
  downloadLearnerProfileExport: vi.fn(),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    gdprModuleEnabled: true,
    learnerProfileEnabled: true,
    loading: false,
  }),
}))

vi.mock('../../../lib/learner-profile-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/learner-profile-api')>(
    '../../../lib/learner-profile-api',
  )
  return {
    ...actual,
    fetchLearnerProfile,
    fetchLearnerProfileFacet,
    fetchLearnerProfileFacetEvidence,
    pauseLearnerProfile,
    resumeLearnerProfile,
    resetLearnerProfile,
    downloadLearnerProfileExport,
  }
})

import { LearnerProfilePanel } from '../learner-profile-panel'

const mockProfile = {
  status: 'active' as const,
  lastComputedAt: '2026-03-01T12:00:00Z',
  facets: [
    {
      facetKey: 'study_rhythm' as const,
      state: 'ok' as const,
      summary: {
        peakWindows: [{ dow: 'Tuesday', hourBucket: '20:00', share: 0.5 }],
        consistencyScore: 0.7,
      },
      confidence: 0.7,
      computedVersion: 1,
      updatedAt: '2026-03-01T12:00:00Z',
    },
  ],
}

const mockInsights = [
  {
    insightKey: 'peak_study_window',
    label: 'When you study most',
    value: {
      peakWindows: [{ dow: 'Tuesday', hourBucket: '20:00', share: 0.5 }],
    },
    confidence: 0.5,
    salience: 100,
  },
]

const mockEvidence = {
  peak_study_window: [
    {
      sourceKind: 'engagement_event',
      sourceTable: 'analytics.engagement_events',
      observationCount: 12,
      windowStart: '2026-01-01T00:00:00Z',
      windowEnd: '2026-03-01T00:00:00Z',
    },
  ],
}

function renderPanel() {
  return render(
    <MemoryRouter>
      <I18nProvider>
        <LearnerProfilePanel />
      </I18nProvider>
    </MemoryRouter>,
  )
}

describe('LearnerProfilePanel', () => {
  beforeEach(() => {
    fetchLearnerProfile.mockReset()
    fetchLearnerProfileFacet.mockReset()
    fetchLearnerProfileFacetEvidence.mockReset()
    pauseLearnerProfile.mockReset()
    resumeLearnerProfile.mockReset()
    resetLearnerProfile.mockReset()
    downloadLearnerProfileExport.mockReset()
    pauseLearnerProfile.mockResolvedValue('paused')
    resumeLearnerProfile.mockResolvedValue('active')
    resetLearnerProfile.mockResolvedValue('reset')
    downloadLearnerProfileExport.mockResolvedValue(undefined)
  })

  it('renders still-building empty state when profile has insufficient data', async () => {
    fetchLearnerProfile.mockResolvedValue({
      status: 'insufficient_data',
      facets: [],
    })

    renderPanel()

    await waitFor(() => {
      expect(screen.getByText(/your profile is still building/i)).toBeInTheDocument()
    })
  })

  it('renders facet section and lazy-loads evidence on expand', async () => {
    fetchLearnerProfile.mockResolvedValue(mockProfile)
    fetchLearnerProfileFacet.mockResolvedValue({
      facet: mockProfile.facets[0],
      insights: mockInsights,
    })
    fetchLearnerProfileFacetEvidence.mockResolvedValue(mockEvidence)

    renderPanel()

    await waitFor(() => {
      expect(screen.getByText(/when you study most/i)).toBeInTheDocument()
    })

    expect(fetchLearnerProfileFacetEvidence).not.toHaveBeenCalled()

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /why do you think this/i }))

    await waitFor(() => {
      expect(fetchLearnerProfileFacetEvidence).toHaveBeenCalledWith('study_rhythm')
    })
    expect(screen.getByRole('cell', { name: 'Study activity' })).toBeInTheDocument()
  })

  it('renders paused state', async () => {
    fetchLearnerProfile.mockResolvedValue({
      status: 'paused',
      facets: mockProfile.facets,
    })
    fetchLearnerProfileFacet.mockResolvedValue({
      facet: mockProfile.facets[0],
      insights: mockInsights,
    })

    renderPanel()

    await waitFor(() => {
      expect(screen.getByText(/profile updates are paused/i)).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /resume updates/i })).toBeEnabled()
  })

  it('enables manage controls when profile is loaded', async () => {
    fetchLearnerProfile.mockResolvedValue(mockProfile)
    fetchLearnerProfileFacet.mockResolvedValue({
      facet: mockProfile.facets[0],
      insights: mockInsights,
    })

    renderPanel()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /download profile/i })).toBeEnabled()
    })
    expect(screen.getByRole('button', { name: /pause updates/i })).toBeEnabled()
    expect(screen.getByRole('button', { name: /reset profile/i })).toBeEnabled()
  })
})
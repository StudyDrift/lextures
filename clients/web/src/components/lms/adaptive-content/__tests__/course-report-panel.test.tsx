import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { CourseReportPanel } from '../reports/course-report-panel'

vi.mock('../../../../lib/courses-api', () => ({
  fetchAdaptiveContentCourseReport: vi.fn(),
  exportAdaptiveContentCourseReportCsv: vi.fn(),
  refreshAdaptiveContentEffectiveness: vi.fn(),
}))

import { fetchAdaptiveContentCourseReport } from '../../../../lib/courses-api'

const fetchReport = vi.mocked(fetchAdaptiveContentCourseReport)

describe('CourseReportPanel', () => {
  beforeEach(() => {
    fetchReport.mockReset()
  })

  it('renders empty state when no ACE activity', async () => {
    fetchReport.mockResolvedValue({
      courseId: 'c1',
      courseCode: 'DEMO',
      empty: true,
      coverage: {
        eligibleContentItems: 0,
        adaptedUnits: 0,
        coveragePct: 0,
        studentsProfiled: 0,
        studentsServedVariant: 0,
        studentsHoldout: 0,
      },
      nUnits: 0,
      nActiveUnits: 0,
      nHelping: 0,
      nRegressing: 0,
      nInsufficient: 0,
      nNoEffect: 0,
      cost: {
        monthlyTokenBudget: 0,
        tokensUsedPeriod: 0,
        unlimited: true,
        periodStart: '2026-07-01',
      },
      unitsToReview: [],
      units: [],
      byMode: [],
      smallCellMinN: 5,
      minNPerArm: 10,
    })

    render(
      <MemoryRouter>
        <CourseReportPanel courseCode="DEMO" />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('ace-report-empty')).toBeInTheDocument()
    })
    expect(screen.getByText(/No adaptive data yet/i)).toBeInTheDocument()
  })

  it('shows KPIs and ranks regressing unit first', async () => {
    fetchReport.mockResolvedValue({
      courseId: 'c1',
      courseCode: 'DEMO',
      empty: false,
      coverage: {
        eligibleContentItems: 2,
        adaptedUnits: 1,
        coveragePct: 50,
        studentsProfiled: 4,
        studentsServedVariant: 3,
        studentsHoldout: 1,
      },
      meanLiftVsControl: -2,
      nUnits: 2,
      nActiveUnits: 2,
      nHelping: 0,
      nRegressing: 1,
      nInsufficient: 1,
      nNoEffect: 0,
      cost: {
        monthlyTokenBudget: 1000,
        tokensUsedPeriod: 200,
        budgetRemaining: 800,
        unlimited: false,
        periodStart: '2026-07-01',
      },
      unitsToReview: [
        {
          unitId: '11111111-1111-1111-1111-111111111111',
          reason: 'regressing',
          verdict: 'regressing',
          meanLift: -8,
          workspaceUrl: '/courses/DEMO/settings/adaptive-content?unit=11111111-1111-1111-1111-111111111111',
        },
      ],
      units: [
        {
          unitId: '11111111-1111-1111-1111-111111111111',
          nTreatment: 12,
          nHoldout: 12,
          treatmentMinusHoldout: -8,
          verdict: 'regressing',
          byMode: [],
          byVariant: [],
          smallCellMinN: 5,
          minNPerArm: 10,
        },
      ],
      byMode: [{ emphasisMode: 'remediate', n: 12, meanLift: -8 }],
      smallCellMinN: 5,
      minNPerArm: 10,
    })

    render(
      <MemoryRouter>
        <CourseReportPanel courseCode="DEMO" />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('ace-course-report')).toBeInTheDocument()
    })
    expect(screen.getByTestId('ace-report-kpis')).toBeInTheDocument()
    expect(screen.getByTestId('ace-report-top-review-unit')).toHaveTextContent(/Regressing/i)
    expect(screen.getByTestId('ace-report-export')).toBeInTheDocument()
  })
})

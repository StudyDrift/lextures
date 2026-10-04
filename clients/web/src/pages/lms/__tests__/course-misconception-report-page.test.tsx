import { render, screen } from '@testing-library/react'
import { MemoryRouter, Outlet, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CoursePublic } from '../../../lib/courses-api'
import CourseMisconceptionReportPage from '../course-misconception-report-page'

const fetchMisconceptionReport = vi.hoisted(() => vi.fn())

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: () => true,
    loading: false,
  }),
}))

vi.mock('../../../lib/courses-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/courses-api')>('../../../lib/courses-api')
  return {
    ...actual,
    fetchMisconceptionReport,
    postImportMisconceptionSeedLibrary: vi.fn(),
  }
})

function renderPage(course: CoursePublic | null) {
  function Layout() {
    return <Outlet context={{ course }} />
  }
  return render(
    <MemoryRouter initialEntries={['/courses/C-HUPCNF/misconception-report']}>
      <Routes>
        <Route path="/courses/:courseCode" element={<Layout />}>
          <Route path="misconception-report" element={<CourseMisconceptionReportPage />} />
        </Route>
      </Routes>
    </MemoryRouter>,
  )
}

describe('Course misconception report', () => {
  beforeEach(() => {
    fetchMisconceptionReport.mockReset()
    fetchMisconceptionReport.mockResolvedValue({ misconceptions: [] })
  })

  it('does not crash while course context is still null', () => {
    renderPage(null)
    expect(screen.getByRole('heading', { name: 'Misconception report' })).toBeInTheDocument()
    expect(screen.getByText('Loading course…')).toBeInTheDocument()
    expect(fetchMisconceptionReport).not.toHaveBeenCalled()
  })

  it('shows a feature-off state when the question bank is disabled', () => {
    renderPage({ courseCode: 'C-HUPCNF', questionBankEnabled: false } as CoursePublic)
    expect(screen.getByText(/question bank feature is disabled/i)).toBeInTheDocument()
    expect(fetchMisconceptionReport).not.toHaveBeenCalled()
  })

  it('loads an empty report when the question bank is on', async () => {
    renderPage({ courseCode: 'C-HUPCNF', questionBankEnabled: true } as CoursePublic)
    expect(await screen.findByText('No misconception events recorded yet.')).toBeInTheDocument()
    expect(fetchMisconceptionReport).toHaveBeenCalledWith('C-HUPCNF')
  })
})

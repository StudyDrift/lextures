import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CourseDocumentTitleProvider } from '../../../context/course-document-title-context'
import CourseModuleContentPage from '../course-module-content-page'

const preview = vi.hoisted(() => ({ mode: 'teacher' as 'teacher' | 'student' }))
const permissions = vi.hoisted(() => ({ allows: true }))

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: () => permissions.allows,
    loading: false,
  }),
}))

vi.mock('../../../lib/course-view-as', () => ({
  useCourseViewAs: () => preview.mode,
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    ffCeuTracking: false,
    aiStudyBuddyEnabled: false,
    aiConfigured: false,
    selfReflectionEnabled: false,
  }),
}))

vi.mock('../../../hooks/use-offline-content', () => ({
  useOfflineContent: () => ({
    status: 'uncached',
    saveForOffline: vi.fn(),
    removeFromOffline: vi.fn(),
    getCachedContent: vi.fn(),
  }),
}))

vi.mock('../../../components/content-page/content-page-reader', () => ({
  ContentPageReader: () => <div>Page body</div>,
}))

vi.mock('../../../components/feature-help/feature-help-trigger', () => ({
  FeatureHelpTrigger: () => null,
}))

vi.mock('../../../components/layout/reading-focus-toggle', () => ({
  ReadingFocusToggle: () => null,
}))

vi.mock('../../../components/a11y/read-aloud-controls', () => ({
  ReadAloudControls: () => null,
}))

vi.mock('../../../components/translation/course-content-locale-selector', () => ({
  CourseContentLocaleSelector: () => null,
}))

vi.mock('../../../lib/courses-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/courses-api')>(
    '../../../lib/courses-api',
  )
  return {
    ...actual,
    fetchCourse: vi.fn(async () => ({
      id: 'course-1',
      markdownThemePreset: 'classic',
      markdownThemeCustom: null,
      viewerEnrollmentRoles: ['teacher', 'test_student'],
      adaptivePathsEnabled: false,
    })),
    fetchModuleContentPage: vi.fn(async () => ({
      itemId: '5bc174c5-c76f-4946-b474-b9a51e6c8ef6',
      title: 'Why This Moment Feels Different',
      markdown: 'A short page.',
      dueAt: null,
      pointsWorth: null,
      assignmentGroupId: null,
      updatedAt: '2026-04-01T00:00:00Z',
      availableFrom: null,
      availableUntil: null,
      requiresAssignmentAccessCode: false,
      assignmentAccessCode: null,
      submissionAllowText: false,
      submissionAllowFileUpload: false,
      submissionAllowUrl: false,
      lateSubmissionPolicy: 'allow',
      latePenaltyPercent: null,
      rubric: null,
      blindGrading: false,
      identitiesRevealedAt: null,
      viewerCanRevealIdentities: false,
      moderatedGrading: false,
      moderationThresholdPct: null,
      moderatorUserId: null,
      provisionalGraderUserIds: [],
      originalityDetection: 'disabled',
      originalityStudentVisibility: 'hide',
    })),
    fetchContentPageMarkups: vi.fn(async () => []),
    postCourseContext: vi.fn(async () => undefined),
    fetchCourseStructure: vi.fn(async () => []),
  }
})

function renderPage() {
  return render(
    <CourseDocumentTitleProvider courseTitle="AI Essentials" defaultPageTitle="Content page">
      <MemoryRouter
        initialEntries={['/courses/C-HUPCNF/modules/content/5bc174c5-c76f-4946-b474-b9a51e6c8ef6']}
      >
        <Routes>
          <Route
            path="/courses/:courseCode/modules/content/:itemId"
            element={<CourseModuleContentPage />}
          />
        </Routes>
      </MemoryRouter>
    </CourseDocumentTitleProvider>,
  )
}

describe('Content page View as Test Student', () => {
  beforeEach(() => {
    preview.mode = 'teacher'
    permissions.allows = true
    const store = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => {
        store.set(key, value)
      },
      removeItem: (key: string) => {
        store.delete(key)
      },
      clear: () => {
        store.clear()
      },
    })
  })

  it('shows Edit for staff when preview is off', async () => {
    renderPage()
    expect(
      await screen.findByRole('heading', { name: 'Why This Moment Feels Different' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Edit' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save for offline' })).toBeNull()
  })

  it('hides Edit and shows learner controls while previewing', async () => {
    preview.mode = 'student'
    renderPage()
    expect(
      await screen.findByRole('heading', { name: 'Why This Moment Feels Different' }),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Save for offline' })).toBeInTheDocument()
    expect(screen.getByText('Page body')).toBeInTheDocument()
  })

  it('keeps learner controls for a real learner', async () => {
    permissions.allows = false
    renderPage()
    expect(
      await screen.findByRole('heading', { name: 'Why This Moment Feels Different' }),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Save for offline' })).toBeInTheDocument()
  })
})

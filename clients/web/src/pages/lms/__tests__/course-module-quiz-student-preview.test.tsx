import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CourseDocumentTitleProvider } from '../../../context/course-document-title-context'
import CourseModuleQuizPage from '../course-module-quiz-page'

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
    ffProctoringIntegration: false,
    aiConfigured: false,
    graderAgentEnabled: false,
  }),
}))

vi.mock('../../../components/feature-help/feature-help-trigger', () => ({
  FeatureHelpTrigger: () => null,
}))

vi.mock('../../../components/content-page/content-page-reader', () => ({
  ContentPageReader: () => <div>Quiz intro</div>,
}))

vi.mock('../../../lib/courses-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/courses-api')>(
    '../../../lib/courses-api',
  )
  return {
    ...actual,
    fetchCourse: vi.fn(async () => ({
      markdownThemePreset: 'classic',
      markdownThemeCustom: null,
      questionBankEnabled: false,
      lockdownModeEnabled: false,
      scheduleMode: 'fixed',
      relativeScheduleAnchorAt: null,
    })),
    fetchModuleQuiz: vi.fn(async () => ({
      itemId: '67e7f104-6691-42db-acdb-06787e4ae273',
      title: 'Module 1 Checkpoint',
      markdown: 'When you are ready, click Start Quiz below.',
      dueAt: null,
      availableFrom: null,
      availableUntil: null,
      unlimitedAttempts: false,
      maxAttempts: 1,
      gradeAttemptPolicy: 'latest' as const,
      passingScorePercent: null,
      pointsWorth: 10,
      lateSubmissionPolicy: 'allow' as const,
      latePenaltyPercent: null,
      timeLimitMinutes: null,
      timerPauseWhenTabHidden: false,
      perQuestionTimeLimitSeconds: null,
      showScoreTiming: 'immediate' as const,
      reviewVisibility: 'full' as const,
      reviewWhen: 'always' as const,
      oneQuestionAtATime: false,
      lockdownMode: 'standard' as const,
      shuffleQuestions: false,
      shuffleChoices: false,
      allowBackNavigation: true,
      requiresQuizAccessCode: false,
      adaptiveDifficulty: 'standard' as const,
      adaptiveTopicBalance: true,
      adaptiveStopRule: 'fixed_count' as const,
      randomQuestionPoolCount: null,
      questions: [],
      updatedAt: '2026-04-01T00:00:00Z',
      isAdaptive: false,
      adaptiveSystemPrompt: null,
      adaptiveSourceItemIds: null,
      adaptiveQuestionCount: 5,
      adaptiveDeliveryMode: 'ai' as const,
      assignmentGroupId: null,
    })),
    fetchReaderMarkups: vi.fn(async () => []),
    fetchCourseGradingSettings: vi.fn(async () => ({ assignmentGroups: [] })),
    fetchQuizAttemptsList: vi.fn(async () => ({ attempts: [] })),
    fetchCourseEnrollmentsList: vi.fn(async () => []),
  }
})

function renderQuiz() {
  return render(
    <CourseDocumentTitleProvider courseTitle="AI Essentials" defaultPageTitle="Quiz">
      <MemoryRouter
        initialEntries={[
          '/courses/C-HUPCNF/modules/quiz/67e7f104-6691-42db-acdb-06787e4ae273',
        ]}
      >
        <Routes>
          <Route
            path="/courses/:courseCode/modules/quiz/:itemId"
            element={<CourseModuleQuizPage />}
          />
        </Routes>
      </MemoryRouter>
    </CourseDocumentTitleProvider>,
  )
}

describe('Quiz View as Test Student', () => {
  beforeEach(() => {
    preview.mode = 'teacher'
    permissions.allows = true
  })

  it('shows Edit questions and More for staff when preview is off', async () => {
    renderQuiz()
    expect(await screen.findByRole('heading', { name: 'Module 1 Checkpoint' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Edit questions' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'More' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Start Quiz' })).toBeNull()
  })

  it('shows Start Quiz and hides staff controls while previewing', async () => {
    preview.mode = 'student'
    renderQuiz()
    expect(await screen.findByRole('heading', { name: 'Module 1 Checkpoint' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Start Quiz' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit questions' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'More' })).toBeNull()
  })

  it('keeps Start Quiz for a real learner', async () => {
    permissions.allows = false
    renderQuiz()
    expect(await screen.findByRole('heading', { name: 'Module 1 Checkpoint' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Start Quiz' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit questions' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'More' })).toBeNull()
  })
})

import { type ComponentProps } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { BrowserRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import {
  defaultQuizAdvancedSettings,
  type ModuleQuizPayload,
} from '../../../lib/courses-api'
import { server } from '../../../test/mocks/server'
import { setAccessToken } from '../../../lib/auth'
import { QuizStudentTakePanel } from '../quiz-student-take-panel'

function minimalQuiz(overrides: Partial<ModuleQuizPayload> = {}): ModuleQuizPayload {
  const base: ModuleQuizPayload = {
    itemId: 'item-1',
    title: 'Unit test quiz',
    markdown: '',
    dueAt: null,
    availableFrom: null,
    availableUntil: null,
    unlimitedAttempts: true,
    maxAttempts: 1,
    gradeAttemptPolicy: 'latest',
    passingScorePercent: null,
    pointsWorth: 10,
    lateSubmissionPolicy: 'allow',
    latePenaltyPercent: null,
    timeLimitMinutes: null,
    timerPauseWhenTabHidden: false,
    perQuestionTimeLimitSeconds: null,
    showScoreTiming: 'immediate',
    reviewVisibility: 'full',
    reviewWhen: 'always',
    oneQuestionAtATime: false,
    lockdownMode: 'standard',
    shuffleQuestions: false,
    shuffleChoices: false,
    allowBackNavigation: true,
    requiresQuizAccessCode: false,
    adaptiveDifficulty: 'standard',
    adaptiveTopicBalance: true,
    adaptiveStopRule: 'fixed_count',
    randomQuestionPoolCount: null,
    questions: [
      {
        id: 'q1',
        prompt: 'Pick Alpha',
        questionType: 'multiple_choice',
        choices: ['Alpha', 'Beta'],
        typeConfig: {},
        correctChoiceIndex: 0,
        multipleAnswer: false,
        answerWithImage: false,
        allowAnyAnswer: false,
        required: true,
        points: 1,
        estimatedMinutes: 2,
      },
    ],
    usesServerQuestionSampling: false,
    updatedAt: new Date().toISOString(),
    isAdaptive: false,
    adaptiveSystemPrompt: null,
    adaptiveSourceItemIds: null,
    adaptiveQuestionCount: 5,
    adaptiveDeliveryMode: 'ai',
    assignmentGroupId: null,
  }
  return { ...base, ...overrides }
}

describe('QuizStudentTakePanel', () => {
  function renderPanel(props: ComponentProps<typeof QuizStudentTakePanel>) {
    return render(
      <BrowserRouter>
        <QuizStudentTakePanel {...props} />
      </BrowserRouter>,
    )
  }

  it('renders nothing when closed', () => {
    const onClose = () => {}
    const { container } = renderPanel({
      open: false,
      onClose,
      courseCode: 'C-TEST',
      itemId: 'item-1',
      quiz: minimalQuiz(),
      advanced: defaultQuizAdvancedSettings(),
      oneQuestionAtATime: false,
      allowBackNavigation: true,
    })
    expect(container.firstChild).toBeNull()
  })

  it('renders page layout when mounted on the attempt route', () => {
    const onClose = () => {}
    renderPanel({
      layout: 'page',
      open: true,
      onClose,
      courseCode: 'C-TEST',
      itemId: 'item-1',
      quiz: minimalQuiz(),
      advanced: defaultQuizAdvancedSettings(),
      oneQuestionAtATime: false,
      allowBackNavigation: true,
    })
    expect(screen.getByRole('heading', { name: /begin quiz/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Begin' })).toBeInTheDocument()
    const scroll = document.querySelector('[data-quiz-take-scroll]')
    expect(scroll).toHaveClass('px-4', 'py-6', 'sm:px-8', 'md:px-10')
    expect(scroll?.firstElementChild).toHaveClass('mx-auto', 'max-w-3xl')
  })

  it('starts a standard attempt and shows the first question', async () => {
    const user = userEvent.setup()
    setAccessToken('test-token')

    server.use(
      http.post('http://localhost:8080/api/v1/courses/:courseCode/quizzes/:itemId/start', () =>
        HttpResponse.json({
          attemptId: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
          attemptNumber: 1,
          startedAt: new Date().toISOString(),
          lockdownMode: 'standard',
          hintsDisabled: false,
          backNavigationAllowed: true,
          currentQuestionIndex: 0,
          deadlineAt: null,
          reducedDistractionMode: false,
          retakePolicy: 'latest',
          maxAttempts: null,
          remainingAttempts: null,
        }),
      ),
      http.get('http://localhost:8080/api/v1/courses/:courseCode/quizzes/:itemId', ({ request }) => {
        const url = new URL(request.url)
        if (!url.searchParams.get('attemptId')) {
          return HttpResponse.json({ error: 'missing attempt' }, { status: 400 })
        }
        return HttpResponse.json({
          itemId: 'item-1',
          title: 'Unit test quiz',
          markdown: '',
          dueAt: null,
          availableFrom: null,
          availableUntil: null,
          unlimitedAttempts: true,
          maxAttempts: 1,
          gradeAttemptPolicy: 'latest',
          passingScorePercent: null,
          pointsWorth: 10,
          lateSubmissionPolicy: 'allow',
          latePenaltyPercent: null,
          timeLimitMinutes: null,
          timerPauseWhenTabHidden: false,
          perQuestionTimeLimitSeconds: null,
          showScoreTiming: 'immediate',
          reviewVisibility: 'full',
          reviewWhen: 'always',
          oneQuestionAtATime: false,
          lockdownMode: 'standard',
          shuffleQuestions: false,
          shuffleChoices: false,
          allowBackNavigation: true,
          requiresQuizAccessCode: false,
          adaptiveDifficulty: 'standard',
          adaptiveTopicBalance: true,
          adaptiveStopRule: 'fixed_count',
          randomQuestionPoolCount: null,
          questions: [
            {
              id: 'q1',
              prompt: 'Pick Alpha',
              questionType: 'multiple_choice',
              choices: ['Alpha', 'Beta'],
              typeConfig: {},
              correctChoiceIndex: null,
              multipleAnswer: false,
              answerWithImage: false,
              allowAnyAnswer: false,
              required: true,
              points: 1,
              estimatedMinutes: 2,
            },
          ],
          usesServerQuestionSampling: false,
          updatedAt: new Date().toISOString(),
          isAdaptive: false,
          adaptiveSystemPrompt: null,
          adaptiveSourceItemIds: null,
          adaptiveQuestionCount: 5,
          adaptiveDeliveryMode: 'ai',
          assignmentGroupId: null,
        })
      }),
    )

    const onClose = () => {}
    renderPanel({
      open: true,
      onClose,
      courseCode: 'C-TEST',
      itemId: 'item-1',
      quiz: minimalQuiz(),
      advanced: defaultQuizAdvancedSettings(),
      oneQuestionAtATime: false,
      allowBackNavigation: true,
    })

    await user.click(screen.getByRole('button', { name: /^Begin$/i }))

    await waitFor(() => {
      expect(screen.getByText(/Pick Alpha/i)).toBeInTheDocument()
    })
  })

  it('submits the authored choice index when choices are shuffled', async () => {
    const user = userEvent.setup()
    setAccessToken('test-token')
    const random = vi.spyOn(Math, 'random').mockReturnValue(0)
    let submitted: { responses?: { questionId: string; selectedChoiceIndex?: number }[] } | null = null

    server.use(
      http.post('http://localhost:8080/api/v1/courses/:courseCode/quizzes/:itemId/start', () =>
        HttpResponse.json({
          attemptId: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
          attemptNumber: 1,
          startedAt: new Date().toISOString(),
          lockdownMode: 'standard',
          hintsDisabled: false,
          backNavigationAllowed: true,
          currentQuestionIndex: 0,
          deadlineAt: null,
          reducedDistractionMode: false,
          retakePolicy: 'latest',
          maxAttempts: null,
          remainingAttempts: null,
        }),
      ),
      http.get('http://localhost:8080/api/v1/courses/:courseCode/quizzes/:itemId', () =>
        HttpResponse.json({
          ...minimalQuiz(),
          shuffleChoices: true,
          questions: [
            {
              id: 'q1',
              prompt: 'Which letter comes first?',
              questionType: 'multiple_choice',
              choices: ['mem', 'shin', 'lamed', 'vav'],
              typeConfig: {},
              correctChoiceIndex: null,
              multipleAnswer: false,
              answerWithImage: false,
              allowAnyAnswer: false,
              required: true,
              points: 1,
              estimatedMinutes: 1,
            },
          ],
        }),
      ),
      http.post('http://localhost:8080/api/v1/courses/:courseCode/quizzes/:itemId/submit', async ({ request }) => {
        submitted = (await request.json()) as typeof submitted
        return HttpResponse.json({
          attemptId: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
          pointsEarned: 1,
          pointsPossible: 1,
          scorePercent: 100,
        })
      }),
      http.get('http://localhost:8080/api/v1/courses/:courseCode/quizzes/:itemId/results', () =>
        HttpResponse.json({ questions: [] }),
      ),
    )

    renderPanel({
      open: true,
      onClose: () => {},
      courseCode: 'C-TEST',
      itemId: 'item-1',
      quiz: minimalQuiz({ shuffleChoices: true }),
      advanced: { ...defaultQuizAdvancedSettings(), shuffleChoices: true },
      oneQuestionAtATime: false,
      allowBackNavigation: true,
    })

    await user.click(screen.getByRole('button', { name: /^Begin$/i }))
    await screen.findByText(/Which letter comes first/i)
    // Math.random() === 0 rotates authored indexes to [1, 2, 3, 0], so "shin" is shown first.
    expect(screen.getAllByRole('radio').map((el) => el.closest('label')?.textContent)).toEqual([
      'shin',
      'lamed',
      'vav',
      'mem',
    ])
    await user.click(screen.getByRole('radio', { name: 'shin' }))
    await user.click(screen.getByRole('button', { name: 'Submit quiz' }))

    await waitFor(() => {
      expect(submitted?.responses?.[0]).toMatchObject({ questionId: 'q1', selectedChoiceIndex: 1 })
    })
    random.mockRestore()
  })
})

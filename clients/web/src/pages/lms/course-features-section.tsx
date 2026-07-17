import { useCallback, useEffect, useMemo, useState } from 'react'
import { FeatureToggleRow } from '../../components/settings/feature-toggle-row'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import { fetchCourseCanvasLink, patchCourseCanvasGradeSync, patchCourseFeatures } from '../../lib/courses-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import type { CoursePublic } from '../../lib/courses-api'

type Props = {
  courseCode: string
  course: CoursePublic
  onCourseUpdated: (c: CoursePublic) => void
}

export function CourseFeaturesSection({ courseCode, course, onCourseUpdated }: Props) {
  const { refresh } = useCourseNavFeatures()
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [query, setQuery] = useState('')
  const [canvasLinked, setCanvasLinked] = useState(false)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const link = await fetchCourseCanvasLink(courseCode)
        if (!cancelled) setCanvasLinked(Boolean(link.linked))
      } catch {
        if (!cancelled) setCanvasLinked(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode])

  const notebookEnabled = course.notebookEnabled !== false
  const feedEnabled = course.feedEnabled !== false
  const calendarEnabled = course.calendarEnabled !== false
  const questionBankEnabled = course.questionBankEnabled === true
  const lockdownModeEnabled = course.lockdownModeEnabled === true
  const standardsAlignmentEnabled = course.standardsAlignmentEnabled === true
  const adaptivePathsEnabled = course.adaptivePathsEnabled === true
  const srsEnabled = course.srsEnabled === true
  const diagnosticAssessmentsEnabled = course.diagnosticAssessmentsEnabled === true
  const hintScaffoldingEnabled = course.hintScaffoldingEnabled === true
  const misconceptionDetectionEnabled = course.misconceptionDetectionEnabled === true
  const sectionsEnabled = course.sectionsEnabled === true
  const discussionsEnabled = course.discussionsEnabled === true
  const collabDocsEnabled = course.collabDocsEnabled === true
  const liveSessionsEnabled = course.liveSessionsEnabled === true
  const officeHoursEnabled = course.officeHoursEnabled === true
  const aiTutorEnabled = course.aiTutorEnabled === true
  const multilingualMessagingEnabled = course.multilingualMessagingEnabled === true
  const filesEnabled = course.filesEnabled !== false
  const attendanceEnabled = course.attendanceEnabled === true
  const whiteboardEnabled = course.whiteboardEnabled === true
  const reportCardsEnabled = course.reportCardsEnabled === true
  const visualBoardsEnabled = course.visualBoardsEnabled === true
  const interactiveQuizzesEnabled = course.interactiveQuizzesEnabled === true
  const canvasGradeSyncEnabled = course.canvasGradeSyncEnabled === true

  const persistCanvasGradeSync = useCallback(
    async (enabled: boolean) => {
      setSaving(true)
      setMessage(null)
      setError(null)
      try {
        const link = await patchCourseCanvasGradeSync(courseCode, enabled)
        onCourseUpdated({
          ...course,
          canvasGradeSyncEnabled: Boolean(link.gradeSyncEnabled),
        })
        setMessage('Saved.')
        toastSaveOk('Course tools updated')
      } catch (e) {
        const msg = e instanceof Error ? e.message : 'Could not save.'
        setError(msg)
        toastMutationError(msg)
      } finally {
        setSaving(false)
      }
    },
    [course, courseCode, onCourseUpdated],
  )

  const persist = useCallback(
    async (patch: {
      notebookEnabled?: boolean
      feedEnabled?: boolean
      calendarEnabled?: boolean
      questionBankEnabled?: boolean
      lockdownModeEnabled?: boolean
      standardsAlignmentEnabled?: boolean
      adaptivePathsEnabled?: boolean
      srsEnabled?: boolean
      diagnosticAssessmentsEnabled?: boolean
      hintScaffoldingEnabled?: boolean
      misconceptionDetectionEnabled?: boolean
      sectionsEnabled?: boolean
      discussionsEnabled?: boolean
      collabDocsEnabled?: boolean
      liveSessionsEnabled?: boolean
      officeHoursEnabled?: boolean
      aiTutorEnabled?: boolean
      multilingualMessagingEnabled?: boolean
      filesEnabled?: boolean
      attendanceEnabled?: boolean
      whiteboardEnabled?: boolean
      reportCardsEnabled?: boolean
      visualBoardsEnabled?: boolean
      interactiveQuizzesEnabled?: boolean
    }) => {
      setSaving(true)
      setMessage(null)
      setError(null)
      try {
        const body = {
          notebookEnabled: patch.notebookEnabled ?? notebookEnabled,
          feedEnabled: patch.feedEnabled ?? feedEnabled,
          calendarEnabled: patch.calendarEnabled ?? calendarEnabled,
          questionBankEnabled: patch.questionBankEnabled ?? questionBankEnabled,
          lockdownModeEnabled: patch.lockdownModeEnabled ?? lockdownModeEnabled,
          standardsAlignmentEnabled: patch.standardsAlignmentEnabled ?? standardsAlignmentEnabled,
          adaptivePathsEnabled: patch.adaptivePathsEnabled ?? adaptivePathsEnabled,
          srsEnabled: patch.srsEnabled ?? srsEnabled,
          diagnosticAssessmentsEnabled:
            patch.diagnosticAssessmentsEnabled ?? diagnosticAssessmentsEnabled,
          hintScaffoldingEnabled: patch.hintScaffoldingEnabled ?? hintScaffoldingEnabled,
          misconceptionDetectionEnabled:
            patch.misconceptionDetectionEnabled ?? misconceptionDetectionEnabled,
          sectionsEnabled: patch.sectionsEnabled ?? sectionsEnabled,
          discussionsEnabled: patch.discussionsEnabled ?? discussionsEnabled,
          collabDocsEnabled: patch.collabDocsEnabled ?? collabDocsEnabled,
          liveSessionsEnabled: patch.liveSessionsEnabled ?? liveSessionsEnabled,
          officeHoursEnabled: patch.officeHoursEnabled ?? officeHoursEnabled,
          aiTutorEnabled: patch.aiTutorEnabled ?? aiTutorEnabled,
          multilingualMessagingEnabled: patch.multilingualMessagingEnabled ?? multilingualMessagingEnabled,
          filesEnabled: patch.filesEnabled ?? filesEnabled,
          attendanceEnabled: patch.attendanceEnabled ?? attendanceEnabled,
          whiteboardEnabled: patch.whiteboardEnabled ?? whiteboardEnabled,
          reportCardsEnabled: patch.reportCardsEnabled ?? reportCardsEnabled,
          visualBoardsEnabled: patch.visualBoardsEnabled ?? visualBoardsEnabled,
          interactiveQuizzesEnabled: patch.interactiveQuizzesEnabled ?? interactiveQuizzesEnabled,
        }
        const updated = await patchCourseFeatures(courseCode, body)
        onCourseUpdated(updated)
        await refresh()
        setMessage('Saved.')
        toastSaveOk('Course tools updated')
      } catch (e) {
        const msg = e instanceof Error ? e.message : 'Could not save.'
        setError(msg)
        toastMutationError(msg)
      } finally {
        setSaving(false)
      }
    },
    [
      adaptivePathsEnabled,
      srsEnabled,
      diagnosticAssessmentsEnabled,
      hintScaffoldingEnabled,
      misconceptionDetectionEnabled,
      sectionsEnabled,
      discussionsEnabled,
      collabDocsEnabled,
      liveSessionsEnabled,
      officeHoursEnabled,
      aiTutorEnabled,
      multilingualMessagingEnabled,
      filesEnabled,
      attendanceEnabled,
      whiteboardEnabled,
      reportCardsEnabled,
      visualBoardsEnabled,
      interactiveQuizzesEnabled,
      calendarEnabled,
      courseCode,
      feedEnabled,
      lockdownModeEnabled,
      notebookEnabled,
      onCourseUpdated,
      questionBankEnabled,
      refresh,
      standardsAlignmentEnabled,
    ],
  )

  const allFeatures = useMemo(
    () =>
      [
        {
          label: 'Adaptive learning paths',
          description:
            'Allow mastery-based branching between modules (requires learner model on the server). Instructors configure rules on each module in the course outline.',
          enabled: adaptivePathsEnabled,
          onToggle: () => void persist({ adaptivePathsEnabled: !adaptivePathsEnabled }),
        },
        {
          label: 'AI Tutor',
          description:
            'Conversational AI tutor side-panel available on all course pages — students can ask questions grounded in course context with a per-student monthly token budget.',
          enabled: aiTutorEnabled,
          onToggle: () => void persist({ aiTutorEnabled: !aiTutorEnabled }),
        },
        {
          label: 'Attendance',
          description:
            'Take roll call or run self-report check-ins; optionally add sessions to the gradebook.',
          enabled: attendanceEnabled,
          onToggle: () => void persist({ attendanceEnabled: !attendanceEnabled }),
        },
        {
          label: 'Calendar',
          description:
            'Month, week, and agenda views of assignment and content due dates for this course.',
          enabled: calendarEnabled,
          onToggle: () => void persist({ calendarEnabled: !calendarEnabled }),
        },
        ...(canvasLinked
          ? [
              {
                label: 'Canvas grade sync',
                description:
                  'When enabled, saving a grade in Lextures automatically pushes it back to the linked Canvas course. Requires a Canvas access token with grade-update permission saved in this browser (from Import settings).',
                enabled: canvasGradeSyncEnabled,
                onToggle: () => void persistCanvasGradeSync(!canvasGradeSyncEnabled),
              },
            ]
          : []),
        {
          label: 'Collaboration boards',
          description:
            'A shared wall where students post cards — text, images, links, and more — and react in real time.',
          enabled: visualBoardsEnabled,
          onToggle: () => void persist({ visualBoardsEnabled: !visualBoardsEnabled }),
        },
        {
          label: 'Live Quizzes',
          description:
            'Host live, game-based quizzes with join codes and leaderboards. Build reusable quiz kits for whole-class play.',
          enabled: interactiveQuizzesEnabled,
          onToggle: () => void persist({ interactiveQuizzesEnabled: !interactiveQuizzesEnabled }),
        },
        {
          label: 'Collaborative documents',
          description:
            'Real-time co-editing with Y.js CRDT — shared rich-text docs and whiteboards for group work and classroom brainstorming.',
          enabled: collabDocsEnabled,
          onToggle: () => void persist({ collabDocsEnabled: !collabDocsEnabled }),
        },
        {
          label: 'Course sections',
          description:
            'Split one course into multiple sections with separate rosters, section instructors, and optional per-section due dates. Disable to hide section APIs and UI.',
          enabled: sectionsEnabled,
          onToggle: () => void persist({ sectionsEnabled: !sectionsEnabled }),
        },
        {
          label: 'Discussion forums',
          description:
            'Threaded discussion boards with replies, upvotes, graded threads, and instructor moderation.',
          enabled: discussionsEnabled,
          onToggle: () => void persist({ discussionsEnabled: !discussionsEnabled }),
        },
        {
          label: 'Feed',
          description:
            'Course-wide channels and messages, including uploads and real-time updates.',
          enabled: feedEnabled,
          onToggle: () => void persist({ feedEnabled: !feedEnabled }),
        },
        {
          label: 'Files',
          description:
            'Course file space where instructors and students can upload, organize, and share documents, presentations, and other materials.',
          enabled: filesEnabled,
          onToggle: () => void persist({ filesEnabled: !filesEnabled }),
        },
        {
          label: 'Live sessions',
          description:
            'Virtual classroom meetings via Jitsi, BigBlueButton, Zoom, or other providers — shows the Live Sessions menu item and scheduling page.',
          enabled: liveSessionsEnabled,
          onToggle: () => void persist({ liveSessionsEnabled: !liveSessionsEnabled }),
        },
        {
          label: 'Misconception detection',
          description:
            'When tagged distractors are selected, record events, adjust mastery weighting, and show targeted remediation in quiz results (requires normalized question-bank options).',
          enabled: misconceptionDetectionEnabled,
          onToggle: () =>
            void persist({ misconceptionDetectionEnabled: !misconceptionDetectionEnabled }),
        },
        {
          label: 'Multilingual Messaging',
          description:
            'Show a Translate button on feed posts, discussion posts, and inbox messages so users can read content in their preferred language.',
          enabled: multilingualMessagingEnabled,
          onToggle: () =>
            void persist({ multilingualMessagingEnabled: !multilingualMessagingEnabled }),
        },
        {
          label: 'Notebook',
          description:
            'Personal notes workspace for this course (stored in the browser for each learner).',
          enabled: notebookEnabled,
          onToggle: () => void persist({ notebookEnabled: !notebookEnabled }),
        },
        {
          label: 'Office hours',
          description:
            'Let instructors define availability windows and students book 1-on-1 appointment slots — shows the Office Hours menu item.',
          enabled: officeHoursEnabled,
          onToggle: () => void persist({ officeHoursEnabled: !officeHoursEnabled }),
        },
        {
          label: 'Placement diagnostic',
          description:
            'Offer a short adaptive placement assessment after enrollment (requires DIAGNOSTIC_ASSESSMENTS_ENABLED on the server and a diagnostic configuration).',
          enabled: diagnosticAssessmentsEnabled,
          onToggle: () =>
            void persist({ diagnosticAssessmentsEnabled: !diagnosticAssessmentsEnabled }),
        },
        {
          label: 'Question bank',
          description:
            'Store quiz items in a reusable bank, optional random pools per attempt, and instructor-only bank APIs.',
          enabled: questionBankEnabled,
          onToggle: () => void persist({ questionBankEnabled: !questionBankEnabled }),
        },
        {
          label: 'Report cards',
          description:
            'Author district-formatted report cards with comment banks, narrative comments, and PDF release to the parent portal.',
          enabled: reportCardsEnabled,
          onToggle: () => void persist({ reportCardsEnabled: !reportCardsEnabled }),
        },
        {
          label: 'Quiz hints & worked examples',
          description:
            'Let learners request progressive hints and unlock worked examples during quizzes (question-bank items with UUID ids).',
          enabled: hintScaffoldingEnabled,
          onToggle: () => void persist({ hintScaffoldingEnabled: !hintScaffoldingEnabled }),
        },
        {
          label: 'Quiz lockdown / kiosk',
          description:
            'Lets instructors choose one-question-at-a-time or kiosk delivery on quizzes (server-enforced progression and optional focus-loss logging).',
          enabled: lockdownModeEnabled,
          onToggle: () => void persist({ lockdownModeEnabled: !lockdownModeEnabled }),
        },
        {
          label: 'Spaced repetition (review)',
          description:
            'Let learners use the global review queue for question-bank items you mark as SRS-eligible (requires SRS_PRACTICE_ENABLED on the server).',
          enabled: srsEnabled,
          onToggle: () => void persist({ srsEnabled: !srsEnabled }),
        },
        {
          label: 'Standards alignment',
          description:
            'Map concepts to Common Core / NGSS codes and view per-standard coverage for this course.',
          enabled: standardsAlignmentEnabled,
          onToggle: () => void persist({ standardsAlignmentEnabled: !standardsAlignmentEnabled }),
        },
        {
          label: 'Whiteboard',
          description:
            'Interactive canvas for teachers to draw diagrams, annotate concepts, and save named boards for later retrieval.',
          enabled: whiteboardEnabled,
          onToggle: () => void persist({ whiteboardEnabled: !whiteboardEnabled }),
        },
      ] as const,
    [
      adaptivePathsEnabled,
      aiTutorEnabled,
      attendanceEnabled,
      calendarEnabled,
      canvasGradeSyncEnabled,
      canvasLinked,
      collabDocsEnabled,
      sectionsEnabled,
      discussionsEnabled,
      feedEnabled,
      filesEnabled,
      liveSessionsEnabled,
      misconceptionDetectionEnabled,
      multilingualMessagingEnabled,
      notebookEnabled,
      officeHoursEnabled,
      diagnosticAssessmentsEnabled,
      questionBankEnabled,
      hintScaffoldingEnabled,
      lockdownModeEnabled,
      srsEnabled,
      standardsAlignmentEnabled,
      whiteboardEnabled,
      reportCardsEnabled,
      visualBoardsEnabled,
      interactiveQuizzesEnabled,
      persist,
      persistCanvasGradeSync,
    ],
  )

  const visibleFeatures = useMemo(() => {
    if (!query.trim()) return allFeatures
    const q = query.toLowerCase()
    return allFeatures.filter(
      (f) => f.label.toLowerCase().includes(q) || f.description.toLowerCase().includes(q),
    )
  }, [allFeatures, query])

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-900/5 dark:border-neutral-800 dark:bg-neutral-950">
      <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Course tools</h2>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        Turn tools on or off for everyone in this course. Disabled tools disappear from the course
        menu and cannot be used until you enable them again.
      </p>

      <div className="mt-3">
        <input
          type="search"
          placeholder="Search tools…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-2 focus:ring-indigo-300 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-indigo-500"
        />
      </div>

      <div className="mt-1 divide-y divide-slate-100 dark:divide-neutral-800">
        {visibleFeatures.length === 0 ? (
          <p className="py-6 text-center text-sm text-slate-400 dark:text-neutral-500">
            No tools match &ldquo;{query}&rdquo;
          </p>
        ) : (
          visibleFeatures.map((f) => (
            <FeatureToggleRow
              key={f.label}
              label={f.label}
              description={f.description}
              enabled={f.enabled}
              disabled={saving}
              onToggle={f.onToggle}
            />
          ))
        )}
      </div>

      {message && (
        <p className="mt-4 text-sm text-emerald-700 dark:text-emerald-400" role="status">
          {message}
        </p>
      )}
      {error && (
        <p className="mt-4 text-sm text-rose-700 dark:text-rose-400" role="status">
          {error}
        </p>
      )}
    </section>
  )
}

import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FeatureToggleRow } from '../../components/settings/feature-toggle-row'
import { useConfirm } from '../../components/use-confirm'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import {
  fetchAdaptiveContentSettings,
  fetchContentToolsCatalog,
  fetchContentToolsSettings,
  fetchCourseCanvasLink,
  patchCourseCanvasGradeSync,
  patchCourseFeatures,
  putAdaptiveContentSettings,
  putContentToolsSettings,
  type AdaptiveContentSettings,
  type ContentToolsCatalogTool,
  type ContentToolsSettings,
} from '../../lib/courses-api'
import { invalidateChecklist } from '../../lib/course-checklist-invalidate'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import type { CoursePublic } from '../../lib/courses-api'

type Props = {
  courseCode: string
  course: CoursePublic
  onCourseUpdated: (c: CoursePublic) => void
}

type CourseFeatureRow = {
  label: string
  description: string
  enabled: boolean
  disabled?: boolean
  disabledReason?: string
  onToggle: () => void
}

export function CourseFeaturesSection({ courseCode, course, onCourseUpdated }: Props) {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
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
  const modulesAiAssistantEnabled = course.modulesAiAssistantEnabled === true
  const multilingualMessagingEnabled = course.multilingualMessagingEnabled === true
  const filesEnabled = course.filesEnabled !== false
  const attendanceEnabled = course.attendanceEnabled === true
  const whiteboardEnabled = course.whiteboardEnabled === true
  const reportCardsEnabled = course.reportCardsEnabled === true
  const visualBoardsEnabled = course.visualBoardsEnabled === true
  const interactiveQuizzesEnabled = course.interactiveQuizzesEnabled === true
  const screenShareEnabled = course.screenShareEnabled === true
  const adaptiveContentEnabled = course.adaptiveContentEnabled === true
  const contentToolsEnabled = course.contentToolsEnabled !== false
  const canvasGradeSyncEnabled = course.canvasGradeSyncEnabled === true

  const [aceSettings, setAceSettings] = useState<AdaptiveContentSettings | null>(null)
  const [aceLoading, setAceLoading] = useState(false)
  const [aceSaving, setAceSaving] = useState(false)

  const [ctSettings, setCtSettings] = useState<ContentToolsSettings | null>(null)
  const [ctCatalog, setCtCatalog] = useState<ContentToolsCatalogTool[]>([])
  const [ctLoading, setCtLoading] = useState(false)
  const [ctSaving, setCtSaving] = useState(false)

  useEffect(() => {
    if (!adaptiveContentEnabled) {
      setAceSettings(null)
      return
    }
    let cancelled = false
    setAceLoading(true)
    void (async () => {
      try {
        const s = await fetchAdaptiveContentSettings(courseCode)
        if (!cancelled) setAceSettings(s)
      } catch {
        if (!cancelled) {
          setAceSettings({
            allowedAxes: ['emphasis', 'scaffolding', 'reading_level', 'misconception'],
            defaultStrategy: 'balanced',
            holdoutPercent: 0,
            monthlyTokenBudget: 0,
            requireInstructorApproval: false,
            studentOptoutAllowed: true,
          })
        }
      } finally {
        if (!cancelled) setAceLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [adaptiveContentEnabled, courseCode])

  useEffect(() => {
    if (!contentToolsEnabled) {
      setCtSettings(null)
      setCtCatalog([])
      return
    }
    let cancelled = false
    setCtLoading(true)
    void (async () => {
      try {
        const [settings, tools] = await Promise.all([
          fetchContentToolsSettings(courseCode),
          fetchContentToolsCatalog(courseCode),
        ])
        if (!cancelled) {
          setCtSettings(settings)
          setCtCatalog(tools)
        }
      } catch {
        if (!cancelled) {
          setCtSettings({
            allowedToolIds: [],
            studentResetAllowed: false,
            maxInstancesPerItem: 50,
            monthlyAiTokenBudget: 0,
            dailyAiCallsPerUser: 50,
            linkIngestionMode: 'public',
            linkHostAllowlist: [],
          })
          setCtCatalog([])
        }
      } finally {
        if (!cancelled) setCtLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [contentToolsEnabled, courseCode])

  const saveAceSettings = useCallback(async () => {
    if (!aceSettings) return
    setAceSaving(true)
    setError(null)
    try {
      const updated = await putAdaptiveContentSettings(courseCode, aceSettings)
      setAceSettings(updated)
      setMessage('Adaptive content settings saved.')
      toastSaveOk('Adaptive content settings saved')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Could not save adaptive content settings.'
      setError(msg)
      toastMutationError(msg)
    } finally {
      setAceSaving(false)
    }
  }, [aceSettings, courseCode])

  const saveCtSettings = useCallback(async () => {
    if (!ctSettings) return
    setCtSaving(true)
    setError(null)
    try {
      const updated = await putContentToolsSettings(courseCode, ctSettings)
      setCtSettings(updated)
      const tools = await fetchContentToolsCatalog(courseCode)
      setCtCatalog(tools)
      setMessage(t('course.features.contentTools.settingsSaved'))
      toastSaveOk(t('course.features.contentTools.settingsSaved'))
    } catch (e) {
      const msg =
        e instanceof Error ? e.message : t('course.features.contentTools.settingsSaveError')
      setError(msg)
      toastMutationError(msg)
    } finally {
      setCtSaving(false)
    }
  }, [ctSettings, courseCode, t])

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
      modulesAiAssistantEnabled?: boolean
      multilingualMessagingEnabled?: boolean
      filesEnabled?: boolean
      attendanceEnabled?: boolean
      whiteboardEnabled?: boolean
      reportCardsEnabled?: boolean
      visualBoardsEnabled?: boolean
      interactiveQuizzesEnabled?: boolean
      screenShareEnabled?: boolean
      adaptiveContentEnabled?: boolean
      contentToolsEnabled?: boolean
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
          modulesAiAssistantEnabled: patch.modulesAiAssistantEnabled ?? modulesAiAssistantEnabled,
          multilingualMessagingEnabled: patch.multilingualMessagingEnabled ?? multilingualMessagingEnabled,
          filesEnabled: patch.filesEnabled ?? filesEnabled,
          attendanceEnabled: patch.attendanceEnabled ?? attendanceEnabled,
          whiteboardEnabled: patch.whiteboardEnabled ?? whiteboardEnabled,
          reportCardsEnabled: patch.reportCardsEnabled ?? reportCardsEnabled,
          visualBoardsEnabled: patch.visualBoardsEnabled ?? visualBoardsEnabled,
          interactiveQuizzesEnabled: patch.interactiveQuizzesEnabled ?? interactiveQuizzesEnabled,
          screenShareEnabled: patch.screenShareEnabled ?? screenShareEnabled,
          adaptiveContentEnabled: patch.adaptiveContentEnabled ?? adaptiveContentEnabled,
          contentToolsEnabled: patch.contentToolsEnabled ?? contentToolsEnabled,
        }
        const updated = await patchCourseFeatures(courseCode, body)
        onCourseUpdated(updated)
        await refresh()
        invalidateChecklist(courseCode)
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
      modulesAiAssistantEnabled,
      multilingualMessagingEnabled,
      filesEnabled,
      attendanceEnabled,
      whiteboardEnabled,
      reportCardsEnabled,
      visualBoardsEnabled,
      interactiveQuizzesEnabled,
      screenShareEnabled,
      adaptiveContentEnabled,
      contentToolsEnabled,
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

  const allFeatures = useMemo((): CourseFeatureRow[] => {
    const rows: CourseFeatureRow[] = [
        {
          label: 'Adaptive Content',
          description:
            'Rewrite content per learner based on a pre-check, then measure improvement.',
          enabled: adaptiveContentEnabled,
          onToggle: () => {
            void persist({ adaptiveContentEnabled: !adaptiveContentEnabled })
          },
        },
        {
          label: 'Content Tools',
          description: t('course.features.contentTools.description'),
          enabled: contentToolsEnabled,
          onToggle: () => {
            void (async () => {
              if (contentToolsEnabled) {
                const ok = await confirm({
                  title: t('course.features.contentTools.title'),
                  description: t('course.features.contentTools.disableConfirm'),
                  variant: 'danger',
                })
                if (!ok) return
              }
              await persist({ contentToolsEnabled: !contentToolsEnabled })
            })()
          },
        },
        {
          label: 'Adaptive learning paths',
          description:
            'Allow mastery-based branching between modules (requires learner model on the server). Instructors configure rules on each module in the course outline.',
          enabled: adaptivePathsEnabled,
          onToggle: () => {
            void persist({ adaptivePathsEnabled: !adaptivePathsEnabled })
          },
        },
        {
          label: 'AI Tutor',
          description:
            'Conversational AI tutor side-panel available on all course pages — students can ask questions grounded in course context with a per-student monthly token budget.',
          enabled: aiTutorEnabled,
          onToggle: () => {
            void persist({ aiTutorEnabled: !aiTutorEnabled })
          },
        },
        {
          label: 'Modules AI assistant',
          description:
            'Chat on the Modules page to propose outline changes (new modules, items, renames, publish). Requires a configured AI provider to use.',
          enabled: modulesAiAssistantEnabled,
          onToggle: () => {
            void persist({ modulesAiAssistantEnabled: !modulesAiAssistantEnabled })
          },
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
          ? ([
              {
                label: 'Canvas grade sync',
                description:
                  'When enabled, saving a grade in Lextures automatically pushes it back to the linked Canvas course. Requires a Canvas access token with grade-update permission saved in this browser (from Import settings).',
                enabled: canvasGradeSyncEnabled,
                onToggle: () => {
                  void persistCanvasGradeSync(!canvasGradeSyncEnabled)
                },
              },
            ] satisfies CourseFeatureRow[])
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
          label: 'Screen sharing',
          description:
            'Share an entire screen to the classroom display and classmates without a cable. Instructor controls who may present.',
          enabled: screenShareEnabled,
          onToggle: () => void persist({ screenShareEnabled: !screenShareEnabled }),
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
      ]
    return rows
  }, [
      adaptiveContentEnabled,
      contentToolsEnabled,
      adaptivePathsEnabled,
      aiTutorEnabled,
      modulesAiAssistantEnabled,
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
      screenShareEnabled,
      persist,
      persistCanvasGradeSync,
      confirm,
      t,
  ])

  const visibleFeatures = useMemo(() => {
    if (!query.trim()) return allFeatures
    const q = query.toLowerCase()
    return allFeatures.filter(
      (f) => f.label.toLowerCase().includes(q) || f.description.toLowerCase().includes(q),
    )
  }, [allFeatures, query])

  return (
    <section
      className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-900/5 dark:border-neutral-800 dark:bg-neutral-950"
      data-focus-anchor="course.features.grid"
    >
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
              disabled={saving || Boolean(f.disabled)}
              disabledReason={f.disabledReason}
              onToggle={f.onToggle}
            />
          ))
        )}
      </div>

      <div className="mt-6 rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-neutral-800 dark:bg-neutral-900/50">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          Adaptive Content settings
        </h3>
        <p id="adaptive-content-settings-help" className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
          Course-wide defaults for which adaptation axes are allowed, cost budget, and holdout
          percent. Unit authoring, preview, and approval live under Settings → Adaptive Content.
        </p>
        {!adaptiveContentEnabled ? (
          <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">
            Turn on <span className="font-medium">Adaptive Content</span> to configure.
          </p>
        ) : aceLoading || !aceSettings ? (
          <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">Loading settings…</p>
        ) : (
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
              Default strategy
              <select
                className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                value={aceSettings.defaultStrategy}
                onChange={(e) =>
                  setAceSettings({ ...aceSettings, defaultStrategy: e.target.value })
                }
                aria-describedby="adaptive-content-settings-help"
              >
                <option value="gentle">gentle</option>
                <option value="balanced">balanced</option>
                <option value="aggressive">aggressive</option>
              </select>
            </label>
            <label className="flex flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
              Holdout percent (0–50)
              <input
                type="number"
                min={0}
                max={50}
                className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                value={aceSettings.holdoutPercent}
                onChange={(e) =>
                  setAceSettings({
                    ...aceSettings,
                    holdoutPercent: Number(e.target.value),
                  })
                }
              />
            </label>
            <label className="flex flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
              Monthly token budget (0 = unlimited)
              <input
                type="number"
                min={0}
                className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                value={aceSettings.monthlyTokenBudget}
                onChange={(e) =>
                  setAceSettings({
                    ...aceSettings,
                    monthlyTokenBudget: Number(e.target.value),
                  })
                }
              />
            </label>
            <div className="flex flex-col gap-2 justify-end">
              <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
                <input
                  type="checkbox"
                  checked={aceSettings.requireInstructorApproval}
                  onChange={(e) =>
                    setAceSettings({
                      ...aceSettings,
                      requireInstructorApproval: e.target.checked,
                    })
                  }
                />
                Require instructor approval
              </label>
              <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
                <input
                  type="checkbox"
                  checked={aceSettings.studentOptoutAllowed}
                  onChange={(e) =>
                    setAceSettings({
                      ...aceSettings,
                      studentOptoutAllowed: e.target.checked,
                    })
                  }
                />
                Allow student opt-out
              </label>
            </div>
            <div className="sm:col-span-2">
              <button
                type="button"
                disabled={aceSaving || saving}
                onClick={() => void saveAceSettings()}
                className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
              >
                {aceSaving ? 'Saving…' : 'Save Adaptive Content settings'}
              </button>
            </div>
          </div>
        )}
      </div>

      <div className="mt-6 rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-neutral-800 dark:bg-neutral-900/50">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          {t('course.features.contentTools.title')} settings
        </h3>
        <p id="content-tools-settings-help" className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
          {t('course.features.contentTools.allowlistHelp')}
        </p>
        {!contentToolsEnabled ? (
          <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">
            Turn on <span className="font-medium">{t('course.features.contentTools.title')}</span> to
            configure.
          </p>
        ) : ctLoading || !ctSettings ? (
          <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">Loading settings…</p>
        ) : (
          <div className="mt-3 grid gap-3">
            <fieldset>
              <legend className="text-xs font-medium text-slate-700 dark:text-neutral-300">
                {t('course.features.contentTools.allowlistLabel')}
              </legend>
              {ctCatalog.length === 0 ? (
                <p className="mt-2 text-sm text-slate-500 dark:text-neutral-400">
                  No tools in the catalog yet.
                </p>
              ) : (
                <ul className="mt-2 space-y-1.5">
                  {ctCatalog.map((tool) => {
                    const checked = ctSettings.allowedToolIds.includes(tool.id)
                    return (
                      <li key={tool.id}>
                        <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={(e) => {
                              const next = e.target.checked
                                ? [...ctSettings.allowedToolIds, tool.id]
                                : ctSettings.allowedToolIds.filter((id) => id !== tool.id)
                              setCtSettings({ ...ctSettings, allowedToolIds: next })
                            }}
                          />
                          <span className="font-mono text-xs">{tool.id}</span>
                          <span className="text-slate-400 dark:text-neutral-500">
                            ({tool.category})
                          </span>
                        </label>
                      </li>
                    )
                  })}
                </ul>
              )}
            </fieldset>
            <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
              <input
                type="checkbox"
                checked={ctSettings.studentResetAllowed}
                onChange={(e) =>
                  setCtSettings({
                    ...ctSettings,
                    studentResetAllowed: e.target.checked,
                  })
                }
              />
              {t('course.features.contentTools.studentResetAllowed')}
            </label>
            <label className="flex max-w-xs flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
              {t('course.features.contentTools.maxInstancesPerItem')}
              <input
                type="number"
                min={1}
                max={200}
                className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                value={ctSettings.maxInstancesPerItem}
                onChange={(e) =>
                  setCtSettings({
                    ...ctSettings,
                    maxInstancesPerItem: Number(e.target.value),
                  })
                }
                aria-describedby="content-tools-settings-help"
              />
            </label>
            <label className="flex max-w-xs flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
              {t('course.features.contentTools.linkIngestionMode')}
              <select
                className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                value={ctSettings.linkIngestionMode}
                onChange={(e) =>
                  setCtSettings({
                    ...ctSettings,
                    linkIngestionMode: e.target.value as ContentToolsSettings['linkIngestionMode'],
                  })
                }
                data-testid="ct-link-ingestion-mode"
              >
                <option value="public">{t('course.features.contentTools.linkIngestionPublic')}</option>
                <option value="allowlist">
                  {t('course.features.contentTools.linkIngestionAllowlist')}
                </option>
                <option value="off">{t('course.features.contentTools.linkIngestionOff')}</option>
              </select>
            </label>
            {ctSettings.linkIngestionMode === 'allowlist' ? (
              <label className="flex max-w-lg flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
                {t('course.features.contentTools.linkHostAllowlist')}
                <input
                  type="text"
                  className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                  value={ctSettings.linkHostAllowlist.join(', ')}
                  onChange={(e) =>
                    setCtSettings({
                      ...ctSettings,
                      linkHostAllowlist: e.target.value
                        .split(',')
                        .map((s) => s.trim())
                        .filter(Boolean),
                    })
                  }
                  placeholder="example.com, docs.example.org"
                />
              </label>
            ) : null}
            <label className="flex max-w-xs flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
              {t('course.features.contentTools.dailyAiCallsPerUser')}
              <input
                type="number"
                min={1}
                max={10000}
                className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                value={ctSettings.dailyAiCallsPerUser}
                onChange={(e) =>
                  setCtSettings({
                    ...ctSettings,
                    dailyAiCallsPerUser: Number(e.target.value),
                  })
                }
              />
            </label>
            <label className="flex max-w-xs flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
              {t('course.features.contentTools.monthlyAiTokenBudget')}
              <input
                type="number"
                min={0}
                className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                value={ctSettings.monthlyAiTokenBudget}
                onChange={(e) =>
                  setCtSettings({
                    ...ctSettings,
                    monthlyAiTokenBudget: Number(e.target.value),
                  })
                }
              />
            </label>
            <div>
              <button
                type="button"
                disabled={ctSaving || saving}
                onClick={() => void saveCtSettings()}
                className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
              >
                {ctSaving
                  ? t('course.features.contentTools.saving')
                  : t('course.features.contentTools.save')}
              </button>
            </div>
          </div>
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
      {ConfirmDialogHost}
    </section>
  )
}

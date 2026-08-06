import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { Pencil, FastForward, Download, CheckCircle, Loader2, Sparkles } from 'lucide-react'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { useOfflineContent } from '../../hooks/use-offline-content'
import { useOnlineStatus } from '../../hooks/use-online-status'
import {
  SyllabusBlockEditor,
  type ContentToolsFlushHandle,
} from '../../components/syllabus/syllabus-block-editor'
import { ContentPageReader } from '../../components/content-page/content-page-reader'
import { BuildContentPageWithAiModal } from '../../components/content-page/build-content-page-with-ai-modal'
import { markdownToSectionsForEditor, sectionsToMarkdown } from '../../components/syllabus/syllabus-section-markdown'
import { usePermissions } from '../../context/use-permissions'
import {
  buildContentPageWithAi,
  createContentToolInstance,
  defaultContentToolConfig,
  fetchContentPageMarkups,
  fetchCourse,
  fetchEnrollmentNext,
  fetchModuleContentPage,
  learnerCourseItemHref,
  patchModuleContentPage,
  postAdaptiveContentContest,
  postAdaptiveContentViewedOriginal,
  postCourseContext,
  putAdaptiveContentOptout,
  type AdaptiveServingMeta,
  type ContentPageMarkup,
  type CoursePublic,
  type DraftContentPageSection,
  type SyllabusSection,
} from '../../lib/courses-api'
import { serializeLexToolFenceBlock } from '../../lib/content-tools/lex-tool-fence'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import {
  type MarkdownThemeCustom,
  type ResolvedMarkdownTheme,
  resolveMarkdownTheme,
} from '../../lib/markdown-theme'
import { useLmsDarkMode } from '../../hooks/use-lms-dark-mode'
import { recordLastVisitedModuleItem } from '../../lib/last-visited-module-item'
import { ReflectionJournalPrompt } from '../../components/study-stats/reflection-journal-prompt'
import { ReadingFocusToggle } from '../../components/layout/reading-focus-toggle'
import { ReadAloudControls } from '../../components/a11y/read-aloud-controls'
import { AuthoringSaveFootprint } from '../../components/authoring-save-footprint'
import { FeatureHelpTrigger } from '../../components/feature-help/feature-help-trigger'
import { formatAbsolute } from '../../lib/format-datetime'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { permCourseItemCreate } from '../../lib/rbac-api'
import { LmsPage } from './lms-page'
import { ReadingLevelBadge } from '../../components/reading-level/reading-level-badge'
import { SimplifiedContentBanner } from '../../components/reading-level/simplified-content-banner'
import { ProfileRationaleChip } from '../../components/learner-profile/profile-rationale-chip'
import { SimplifyDiffDialog } from '../../components/reading-level/simplify-diff-dialog'
import { useSimplifiedContentView } from '../../components/reading-level/use-simplified-content-view'
import { useSimplifyDialog } from '../../components/reading-level/use-simplify-dialog'
import { AdaptedBanner } from '../../components/lms/adaptive-content/adapted-banner'
import { useAdaptedContentView } from '../../components/lms/adaptive-content/use-adapted-content-view'
import { usePrompt } from '../../components/use-prompt'
import {
  fetchItemReadingLevel,
  isReadingLevelEnabled,
  simplifyItemContent,
  type ReadingLevelInfo,
} from '../../lib/reading-level-api'
import { CourseContentLocaleSelector } from '../../components/translation/course-content-locale-selector'
import {
  altTextEnforcementFeatureEnabled,
  altTextHardBlockEnabled,
} from '../../lib/platform-features'
import { SeatTimeProgressBar } from '../../components/seat-time/seat-time-progress-bar'
import { useSeatTimeHeartbeat } from '../../hooks/use-seat-time-heartbeat'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { StudyBuddyWidget } from '../../components/notebook/study-buddy-widget'
import { summarizeSectionsAltText } from '../../lib/image-alt-validation'

function newLocalId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `local-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

export default function CourseModuleContentPage() {
  const { courseCode, itemId } = useParams<{ courseCode: string; itemId: string }>()
  const { allows, loading: permLoading } = usePermissions()
  const { ffCeuTracking, aiStudyBuddyEnabled, aiConfigured } = usePlatformFeatures()
  const { contentToolsEnabled } = useCourseNavFeatures()
  const { prompt, InputDialogHost } = usePrompt()

  const [title, setTitle] = useState('')
  const [markdown, setMarkdown] = useState('')
  const [updatedAt, setUpdatedAt] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [markups, setMarkups] = useState<ContentPageMarkup[]>([])

  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<SyllabusSection[]>([])
  const [buildAiOpen, setBuildAiOpen] = useState(false)
  const [ctInstancesReloadKey, setCtInstancesReloadKey] = useState(0)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [lastLocalAuthoringSave, setLastLocalAuthoringSave] = useState<string | null>(null)
  const [readingLevel, setReadingLevel] = useState<ReadingLevelInfo | null>(null)
  const [pagePayload, setPagePayload] = useState<{
    simplifiedForReadingLevel?: boolean
    originalMarkdown?: string | null
    readingLevelTargetFkgl?: number | null
    profileRationale?: import('../../lib/courses-api').ProfileRationale
    preferredAlternateItemId?: string | null
    adaptive?: AdaptiveServingMeta | null
  }>({})
  const [optoutBusy, setOptoutBusy] = useState(false)
  const [contestBusy, setContestBusy] = useState(false)
  const simplifyDlg = useSimplifyDialog()
  const readingLevelOn = isReadingLevelEnabled()
  const altTextOn = altTextEnforcementFeatureEnabled()
  const altTextHardBlock = altTextHardBlockEnabled()
  const draftAltCoverage = useMemo(
    () => (editing && altTextOn ? summarizeSectionsAltText(draft) : { withAlt: 0, total: 0, missing: [] }),
    [altTextOn, draft, editing],
  )
  const saveBlockedByAltText =
    altTextOn && altTextHardBlock && draftAltCoverage.missing.length > 0
  const [mdPreset, setMdPreset] = useState<string>('classic')
  const [mdCustom, setMdCustom] = useState<MarkdownThemeCustom | null>(null)
  const lmsUiDark = useLmsDarkMode()
  const mdTheme = useMemo(
    (): ResolvedMarkdownTheme => resolveMarkdownTheme(mdPreset, mdCustom, { lmsUiDark }),
    [mdPreset, mdCustom, lmsUiDark],
  )

  const contentLeaveSentRef = useRef(false)
  const contentOpenSentForRef = useRef<string | null>(null)
  const contentToolsFlushRef = useRef<ContentToolsFlushHandle | null>(null)
  const [courseProfile, setCourseProfile] = useState<CoursePublic | null>(null)
  const [nextNav, setNextNav] = useState<{ href: string; title: string; live: string } | null>(null)
  const [autoAdvance, setAutoAdvance] = useState(() => {
    return localStorage.getItem('lms_auto_advance') === 'true'
  })
  const [countdown, setCountdown] = useState<number | null>(null)
  const nextNavRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()
  const isOnline = useOnlineStatus()
  const { status: offlineStatus, saveForOffline } = useOfflineContent(itemId)

  useEffect(() => {
    localStorage.setItem('lms_auto_advance', String(autoAdvance))
  }, [autoAdvance])

  useEffect(() => {
    if (!autoAdvance || !nextNav || countdown !== null) {
      if (!autoAdvance) setCountdown(null)
      return
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          setCountdown(3)
        } else {
          setCountdown(null)
        }
      },
      { threshold: 0.5 },
    )

    if (nextNavRef.current) {
      observer.observe(nextNavRef.current)
    }

    return () => observer.disconnect()
  }, [autoAdvance, nextNav, countdown])

  useEffect(() => {
    if (countdown === null) return
    if (countdown <= 0) {
      if (nextNav) navigate(nextNav.href)
      return
    }

    const timer = setTimeout(() => {
      setCountdown(countdown - 1)
    }, 1000)

    return () => clearTimeout(timer)
  }, [countdown, nextNav, navigate])

  const canEdit = Boolean(
    courseCode && itemId && !permLoading && allows(permCourseItemCreate(courseCode)),
  )
  const seatTimeEnabled = ffCeuTracking && !canEdit && !editing && Boolean(itemId)
  useSeatTimeHeartbeat(itemId, seatTimeEnabled)

  const loadMarkups = useCallback(async () => {
    if (!courseCode || !itemId) return
    try {
      const list = await fetchContentPageMarkups(courseCode, itemId)
      setMarkups(list)
    } catch {
      setMarkups([])
    }
  }, [courseCode, itemId])

  const load = useCallback(async () => {
    if (!courseCode || !itemId) return
    setLoading(true)
    setLoadError(null)
    try {
      const [data, courseRow] = await Promise.all([
        fetchModuleContentPage(courseCode, itemId),
        fetchCourse(courseCode),
      ])
      setCourseProfile(courseRow)
      setTitle(data.title)
      setMarkdown(data.markdown)
      setUpdatedAt(data.updatedAt)
      setPagePayload({
        simplifiedForReadingLevel: data.simplifiedForReadingLevel,
        originalMarkdown: data.originalMarkdown,
        readingLevelTargetFkgl: data.readingLevelTargetFkgl,
        profileRationale: data.profileRationale,
        preferredAlternateItemId: data.preferredAlternateItemId,
        adaptive: data.adaptive ?? null,
      })
      // AC.6 entry-ticket gate: route to pre-assessment when required and not yet profiled.
      // Server only attaches `adaptive` for student viewers.
      if (data.adaptive?.requiresPreAssessment && data.adaptive.preAssessmentItemId) {
        navigate(
          `/courses/${encodeURIComponent(courseCode)}/modules/quiz/${encodeURIComponent(data.adaptive.preAssessmentItemId)}`,
          { replace: true },
        )
        return
      }
      if (readingLevelOn && courseCode && itemId) {
        void fetchItemReadingLevel(courseCode, itemId)
          .then(setReadingLevel)
          .catch(() => setReadingLevel(null))
      }
      setMdPreset(courseRow.markdownThemePreset)
      setMdCustom(courseRow.markdownThemeCustom)
      recordLastVisitedModuleItem(courseCode, {
        itemId,
        kind: 'content_page',
        title: data.title,
      })
      void loadMarkups()
      const openKey = `${courseCode}:${itemId}`
      if (contentOpenSentForRef.current !== openKey) {
        contentOpenSentForRef.current = openKey
        void postCourseContext(courseCode, {
          kind: 'content_open',
          structureItemId: itemId,
        }).catch(() => {})
      }
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : 'Could not load this page.')
      setTitle('')
      setMarkdown('')
      setUpdatedAt(null)
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId, loadMarkups, readingLevelOn, navigate])

  useEffect(() => {
    void load()
  }, [load])

  useCoursePageTitle(!loading && title ? title : null)

  useEffect(() => {
    if (
      canEdit ||
      !courseCode ||
      !itemId ||
      !courseProfile?.viewerStudentEnrollmentId ||
      !courseProfile.adaptivePathsEnabled
    ) {
      setNextNav(null)
      return
    }
    const enrollmentId = courseProfile.viewerStudentEnrollmentId
    let cancelled = false
    void (async () => {
      try {
        const n = await fetchEnrollmentNext(enrollmentId, {
          fromItemId: itemId,
        })
        if (cancelled) return
        const href = learnerCourseItemHref(courseCode!, n.item)
        const title = n.item.title?.trim() || 'Next'
        const live =
          n.skipReason?.trim() ||
          (n.fallback
            ? 'Continuing in course order.'
            : `Next: ${title}.`)
        setNextNav({ href, title, live })
      } catch {
        if (!cancelled) setNextNav(null)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [canEdit, courseCode, courseProfile, itemId])

  useEffect(() => {
    if (!courseCode || !itemId) return
    contentLeaveSentRef.current = false
    const sendLeave = (keepalive: boolean) => {
      if (contentLeaveSentRef.current) return
      contentLeaveSentRef.current = true
      void postCourseContext(
        courseCode,
        { kind: 'content_leave', structureItemId: itemId },
        { keepalive },
      ).catch(() => {})
    }
    const onPageHide = () => sendLeave(true)
    window.addEventListener('pagehide', onPageHide)
    return () => {
      window.removeEventListener('pagehide', onPageHide)
      sendLeave(false)
      const openKey = `${courseCode}:${itemId}`
      if (contentOpenSentForRef.current === openKey) {
        contentOpenSentForRef.current = null
      }
    }
  }, [courseCode, itemId])

  async function handleSaveForOffline() {
    if (!courseCode || !itemId || !title || !markdown) return
    await saveForOffline({
      id: itemId,
      course_id: courseCode,
      type: 'content_page',
      title,
      content: markdown,
      updated_at: updatedAt ?? new Date().toISOString(),
    })
  }

  function beginEdit() {
    setSaveError(null)
    setDraft(markdownToSectionsForEditor(markdown, newLocalId))
    setEditing(true)
  }

  function cancelEdit() {
    setSaveError(null)
    setBuildAiOpen(false)
    simplifyDlg.setOpen(false)
    setEditing(false)
    setDraft([])
  }

  /**
   * Apply AI draft sections. When tools are present, create content-tool instances
   * and embed ```lex-tool fences before updating the editor draft.
   */
  async function applyBuiltSections(sections: DraftContentPageSection[]) {
    if (!courseCode || !itemId) return

    const next: SyllabusSection[] = []
    let toolsCreated = 0
    let toolsFailed = 0

    for (const s of sections) {
      const sectionId = newLocalId()
      const bodyParts: string[] = []
      const prose = (s.markdown ?? '').trim()
      if (prose) bodyParts.push(prose)

      const tools = s.tools ?? []
      for (const tool of tools) {
        const toolId = String(tool.toolId ?? '').trim()
        if (!toolId) continue
        try {
          const config =
            tool.config && Object.keys(tool.config).length > 0
              ? tool.config
              : defaultContentToolConfig(toolId)
          const created = await createContentToolInstance(courseCode, {
            toolId,
            hostKind: 'content_page',
            structureItemId: itemId,
            sectionKey: sectionId,
            config,
          })
          bodyParts.push(
            serializeLexToolFenceBlock({
              instanceId: created.id,
              toolId: created.toolId,
              v: 1,
            }),
          )
          toolsCreated += 1
        } catch {
          toolsFailed += 1
        }
      }

      const markdownBody = bodyParts.join('\n\n').trim()
      if (!s.heading.trim() && !markdownBody) continue
      next.push({
        id: sectionId,
        heading: s.heading,
        markdown: markdownBody,
      })
    }

    if (next.length === 0) {
      throw new Error('No content sections were generated. Try a more specific description.')
    }

    setDraft(next)
    if (toolsCreated > 0) {
      setCtInstancesReloadKey((k) => k + 1)
    }
    setBuildAiOpen(false)
    if (toolsFailed > 0) {
      toastMutationError(
        toolsCreated > 0
          ? `Draft applied, but ${toolsFailed} interactive tool${toolsFailed === 1 ? '' : 's'} could not be placed.`
          : `Could not place interactive tools (${toolsFailed} failed). Prose draft was still applied.`,
      )
    }
  }

  async function save() {
    if (!courseCode || !itemId) return
    const body = sectionsToMarkdown(draft)
    setSaveError(null)
    setSaving(true)
    try {
      // Persist open/dirty content-tool configs (e.g. inline questions) before the page body.
      await contentToolsFlushRef.current?.flush()
      const data = await patchModuleContentPage(courseCode, itemId, {
        markdown: body,
        dueAt: null,
      })
      setMarkdown(data.markdown)
      setUpdatedAt(data.updatedAt)
      if (readingLevelOn && courseCode && itemId) {
        void fetchItemReadingLevel(courseCode, itemId)
          .then(setReadingLevel)
          .catch(() => setReadingLevel(null))
      }
      setLastLocalAuthoringSave(new Date().toISOString())
      setBuildAiOpen(false)
      simplifyDlg.setOpen(false)
      setEditing(false)
      setDraft([])
      void loadMarkups()
      toastSaveOk('Page saved')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Could not save.'
      setSaveError(msg)
      toastMutationError(msg)
    } finally {
      setSaving(false)
    }
  }

  const adaptiveMeta = pagePayload.adaptive
  const isAdaptedServe = Boolean(adaptiveMeta?.isAdapted && adaptiveMeta.canViewOriginal)
  // When adapted: markdown is the variant; originalMarkdown is the base (shipped for View original).
  const adaptedOriginal =
    isAdaptedServe && pagePayload.originalMarkdown
      ? pagePayload.originalMarkdown
      : markdown
  const adaptedView = useAdaptedContentView(
    isAdaptedServe ? markdown : undefined,
    adaptedOriginal,
    isAdaptedServe,
  )

  const originalForStudent =
    !isAdaptedServe && pagePayload.originalMarkdown && pagePayload.simplifiedForReadingLevel
      ? pagePayload.originalMarkdown
      : isAdaptedServe
        ? adaptedView.displayMarkdown
        : markdown
  const simplifiedView = useSimplifiedContentView(
    !isAdaptedServe && pagePayload.simplifiedForReadingLevel ? markdown : undefined,
    originalForStudent,
  )
  const displayMarkdown = isAdaptedServe
    ? adaptedView.displayMarkdown
    : simplifiedView.displayMarkdown

  const handleToggleAdaptedOriginal = useCallback(() => {
    const scrollY = typeof window !== 'undefined' ? window.scrollY : 0
    if (adaptedView.showingOriginal) {
      adaptedView.showAdapted()
    } else {
      adaptedView.showOriginal()
      if (courseCode && adaptiveMeta?.unitId) {
        void postAdaptiveContentViewedOriginal(courseCode, adaptiveMeta.unitId).catch(() => {})
      }
    }
    // Preserve scroll position after toggle (AC.6 a11y).
    requestAnimationFrame(() => {
      if (typeof window !== 'undefined') window.scrollTo(0, scrollY)
    })
  }, [adaptedView, courseCode, adaptiveMeta?.unitId])

  const handlePreferStandard = useCallback(async () => {
    if (!courseCode || optoutBusy) return
    setOptoutBusy(true)
    try {
      await putAdaptiveContentOptout(courseCode, true)
      await load()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not update preference.')
    } finally {
      setOptoutBusy(false)
    }
  }, [courseCode, load, optoutBusy])

  const handleReportAdaptation = useCallback(async () => {
    if (!courseCode || !adaptiveMeta?.unitId || contestBusy) return
    const reason = await prompt({
      title: 'Report this adaptation',
      description: 'What seems wrong about this adaptation? (optional)',
      confirmLabel: 'Submit report',
    })
    if (reason === null) return
    setContestBusy(true)
    try {
      await postAdaptiveContentContest(courseCode, adaptiveMeta.unitId, {
        reason: reason.trim() || undefined,
      })
      adaptedView.showOriginal()
      toastSaveOk('Thanks — your instructor will review this adaptation. Showing the original.')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not submit report.')
    } finally {
      setContestBusy(false)
    }
  }, [adaptedView, adaptiveMeta?.unitId, contestBusy, courseCode, prompt])

  if (!courseCode || !itemId) {
    return (
      <LmsPage title="Content page" description="">
        <p className="mt-6 text-sm text-fg-muted">Invalid link.</p>
      </LmsPage>
    )
  }

  const description = updatedAt == null ? '' : `Updated ${formatAbsolute(updatedAt)}`

  const backTo = `/courses/${encodeURIComponent(courseCode)}/modules`

  async function runSimplify(targetGrade: number) {
    if (!courseCode || !itemId) return
    const body = sectionsToMarkdown(draft)
    simplifyDlg.setOpen(true)
    simplifyDlg.setTargetFkgl(targetGrade)
    simplifyDlg.setOriginal(body)
    simplifyDlg.setSimplified('')
    simplifyDlg.setError(null)
    simplifyDlg.setLoading(true)
    try {
      const res = await simplifyItemContent(courseCode, itemId, targetGrade, body)
      simplifyDlg.setSimplified(res.simplified)
      simplifyDlg.setComputedFkgl(res.computedFkgl)
    } catch (e) {
      simplifyDlg.setError(e instanceof Error ? e.message : 'Simplification failed')
    } finally {
      simplifyDlg.setLoading(false)
    }
  }

  return (
    <LmsPage
      title={loading ? 'Content page' : title || 'Content page'}
      description={description}
      actions={
        editing ? (
          <div className="flex flex-wrap items-center gap-2">
            {canEdit ? <FeatureHelpTrigger topic="content-page" /> : null}
            <ReadingFocusToggle />
            {canEdit && aiConfigured ? (
              <button
                type="button"
                onClick={() => setBuildAiOpen(true)}
                disabled={saving}
                className="inline-flex items-center gap-2 rounded-xl border border-indigo-200 bg-indigo-50 px-3.5 py-2.5 text-sm font-semibold text-accent-fg shadow-sm transition-[background-color,color,border-color] hover:border-indigo-300 hover:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-indigo-900/60 dark:bg-indigo-950/40 dark:text-indigo-200 dark:hover:bg-indigo-950/70"
              >
                <Sparkles className="h-4 w-4" aria-hidden />
                Build with AI
              </button>
            ) : null}
            <button
              type="button"
              onClick={cancelEdit}
              disabled={saving}
              className="rounded-xl border border-border-strong bg-surface-raised px-4 py-2.5 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60"
            >
              Cancel
            </button>
            {readingLevelOn && readingLevel?.sufficient && readingLevel.fkgl != null ? (
              <ReadingLevelBadge
                fkgl={readingLevel.fkgl}
                sufficient
                aboveThreshold={readingLevel.aboveThreshold}
              />
            ) : null}
            {readingLevelOn ? (
              <select
                className="rounded-lg border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-overlay"
                defaultValue=""
                aria-label="Simplify to grade level"
                onChange={(e) => {
                  const v = e.target.value
                  e.target.value = ''
                  if (v) void runSimplify(Number(v))
                }}
              >
                <option value="">Simplify to…</option>
                {['K', '1', '2', '3', '4', '5', '6', '7', '8', '9', '10', '11', '12'].map((g, i) => (
                  <option key={g} value={String(i === 0 ? 0 : i)}>
                    Grade {g}
                  </option>
                ))}
              </select>
            ) : null}
            <button
              type="button"
              onClick={() => void save()}
              disabled={saving || saveBlockedByAltText}
              title={saveBlockedByAltText ? 'Add alt text to all images before saving' : undefined}
              className="rounded-xl bg-accent-solid px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        ) : canEdit ? (
          <div className="flex flex-wrap items-center gap-2">
            <FeatureHelpTrigger topic="content-page" />
            <ReadingFocusToggle />
            <button
              type="button"
              onClick={beginEdit}
              disabled={loading}
              className="inline-flex items-center gap-2 rounded-xl border border-border-strong bg-surface-raised px-4 py-2.5 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60"
            >
              <Pencil className="h-4 w-4" aria-hidden />
              Edit
            </button>
          </div>
        ) : (
          <div className="flex flex-wrap items-center gap-2">
            {courseCode ? <CourseContentLocaleSelector courseCode={courseCode} /> : null}
            {!loading && !loadError && isOnline && (
              <button
                type="button"
                onClick={() => void handleSaveForOffline()}
                disabled={offlineStatus === 'saving'}
                aria-label={
                  offlineStatus === 'cached' ? 'Saved for offline' : 'Save for offline'
                }
                title={
                  offlineStatus === 'cached'
                    ? 'Content saved for offline access'
                    : 'Save this page for offline access'
                }
                className="inline-flex items-center gap-1.5 rounded-lg border border-border-default bg-surface-raised px-3 py-2 text-xs font-medium text-fg-muted shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default dark:hover:bg-neutral-700"
              >
                {offlineStatus === 'saving' ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
                ) : offlineStatus === 'cached' ? (
                  <CheckCircle className="h-3.5 w-3.5 text-emerald-600" aria-hidden />
                ) : (
                  <Download className="h-3.5 w-3.5" aria-hidden />
                )}
                {offlineStatus === 'cached' ? 'Cached' : 'Save offline'}
              </button>
            )}
            <ReadingFocusToggle />
            <ReadAloudControls />
          </div>
        )
      }
    >
      <p className="mt-2 text-start text-sm">
        <Link to={backTo} className="font-medium text-accent-fg hover:text-indigo-500">
          ← Back to modules
        </Link>
      </p>

      {canEdit && !loading && !loadError ? (
        <div className="mt-6">
          <AuthoringSaveFootprint
            lastSavedIso={lastLocalAuthoringSave ?? updatedAt}
            saving={editing && saving}
            error={editing ? saveError : null}
            onRetry={editing ? () => void save() : undefined}
          />
        </div>
      ) : null}

      <div className="mx-auto w-full max-w-[96ch] min-w-0">
        {loadError && (
          <p className="mt-6 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800">
            {loadError}
          </p>
        )}
        {loading && <p className="mt-8 text-sm text-fg-muted">Loading…</p>}
        {!isOnline && offlineStatus === 'cached' && !loading && (
          <p className="mt-4 inline-flex items-center gap-1.5 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-1.5 text-xs font-medium text-emerald-800 dark:border-emerald-900/50 dark:bg-emerald-950/50 dark:text-emerald-300">
            <CheckCircle className="h-3.5 w-3.5" aria-hidden />
            Cached — viewing offline content
          </p>
        )}

        {!loading &&
          !loadError &&
          !editing &&
          !canEdit &&
          pagePayload.preferredAlternateItemId &&
          pagePayload.preferredAlternateItemId !== itemId && (
            <div className="mt-4 rounded-xl border border-violet-200 bg-violet-50/80 px-4 py-3 dark:border-violet-900/50 dark:bg-violet-950/30">
              {pagePayload.profileRationale ? (
                <ProfileRationaleChip rationale={pagePayload.profileRationale} />
              ) : null}
              <Link
                to={`/courses/${encodeURIComponent(courseCode)}/modules/content/${encodeURIComponent(pagePayload.preferredAlternateItemId)}`}
                className="mt-2 inline-flex text-sm font-semibold text-violet-800 underline underline-offset-2 dark:text-violet-200"
              >
                Open the version that fits how you learn
              </Link>
            </div>
          )}

        {!loading && !loadError && !editing && !canEdit && adaptedView.hasAdapted && (
          <AdaptedBanner
            adaptationReason={adaptiveMeta?.adaptationReason}
            showingOriginal={adaptedView.showingOriginal}
            canViewOriginal={Boolean(adaptiveMeta?.canViewOriginal)}
            optoutAllowed={Boolean(adaptiveMeta?.optoutAllowed) && !optoutBusy}
            onToggleOriginal={handleToggleAdaptedOriginal}
            onPreferStandard={
              adaptiveMeta?.optoutAllowed ? () => void handlePreferStandard() : undefined
            }
            onReportAdaptation={() => void handleReportAdaptation()}
            reportBusy={contestBusy}
          />
        )}

        {!loading && !loadError && !editing && !isAdaptedServe && simplifiedView.hasSimplified && (
          <SimplifiedContentBanner
            targetFkgl={pagePayload.readingLevelTargetFkgl ?? undefined}
            originalMarkdown={originalForStudent}
            simplifiedMarkdown={markdown}
            showingOriginal={simplifiedView.showingOriginal}
            onShowOriginal={simplifiedView.showOriginal}
            onShowSimplified={simplifiedView.showSimplified}
          />
        )}

        {!loading && !loadError && !editing && (
          <div className="mt-8 space-y-6 text-[1.0625rem] leading-relaxed">
            <ContentPageReader
              markdown={displayMarkdown}
              theme={mdTheme}
              markups={markups}
              onMarkupsChange={loadMarkups}
              courseCode={courseCode}
              markupTarget={{ variant: 'content_page', itemId }}
              contentTitle={title || 'Content page'}
            />
            {nextNav ? (
              <div
                ref={nextNavRef}
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border-default bg-surface-base px-4 py-3 dark:border-border-default/60"
                aria-live="polite"
                aria-atomic="true"
              >
                <div className="min-w-0">
                  <p className="text-sm text-fg-default">
                    <span className="font-medium">Suggested next:</span> {nextNav.title}
                    {nextNav.live && nextNav.live !== `Next: ${nextNav.title}.` ? (
                      <span className="mt-1 block text-xs text-fg-muted">
                        {nextNav.live}
                      </span>
                    ) : null}
                  </p>
                </div>
                <div className="flex items-center gap-4">
                  <label className="flex cursor-pointer select-none items-center gap-2">
                    <input
                      type="checkbox"
                      checked={autoAdvance}
                      onChange={(e) => setAutoAdvance(e.target.checked)}
                      className="h-4 w-4 rounded border-border-strong text-accent-fg focus:ring-indigo-500"
                    />
                    <span className="text-xs text-fg-muted">Auto-advance</span>
                  </label>
                  <Link
                    to={nextNav.href}
                    className="inline-flex items-center gap-2 rounded-lg bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
                  >
                    {countdown !== null ? (
                      <>
                        <FastForward className="h-4 w-4" />
                        Next in {countdown}…
                      </>
                    ) : (
                      'Continue'
                    )}
                  </Link>
                </div>
              </div>
            ) : null}
            {seatTimeEnabled && courseProfile?.id ? (
              <SeatTimeProgressBar courseId={courseProfile.id} />
            ) : null}
            {!editing && courseProfile?.id ? (
              <ReflectionJournalPrompt courseId={courseProfile.id} />
            ) : null}
          </div>
        )}
      </div>

      {!loading && !loadError && editing && (
        <div className="mt-6 -mx-6 md:-mx-8">
          {saveError && (
            <p className="mb-4 rounded-lg border border-rose-200 bg-rose-50 px-6 py-3 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/50 dark:text-rose-200 md:px-8">
              {saveError}
            </p>
          )}
          <div className="px-4 md:px-8">
            <SyllabusBlockEditor
              courseCode={courseCode}
              structureItemId={itemId}
              hostKind="content_page"
              instancesReloadKey={ctInstancesReloadKey}
              sections={draft}
              onChange={setDraft}
              disabled={saving}
              documentVariant="page"
              contentToolsFlushRef={contentToolsFlushRef}
            />
          </div>
        </div>
      )}

      <SimplifyDiffDialog
        open={simplifyDlg.open}
        original={simplifyDlg.original}
        simplified={simplifyDlg.simplified}
        targetFkgl={simplifyDlg.targetFkgl}
        computedFkgl={simplifyDlg.computedFkgl}
        loading={simplifyDlg.loading}
        error={simplifyDlg.error}
        onClose={() => simplifyDlg.setOpen(false)}
        onAccept={() => {
          setDraft(markdownToSectionsForEditor(simplifyDlg.simplified, newLocalId))
          simplifyDlg.setOpen(false)
        }}
      />
      {courseCode && itemId ? (
        <BuildContentPageWithAiModal
          open={buildAiOpen}
          existingMarkdown={sectionsToMarkdown(draft)}
          contentToolsAvailable={contentToolsEnabled}
          defaultIncludeTools
          description={
            contentToolsEnabled
              ? 'Describe what this page should cover. The draft can include prose and interactive tools (checks, flashcards, and more). Nothing is saved until you click Save.'
              : undefined
          }
          onClose={() => setBuildAiOpen(false)}
          onBuild={async ({ prompt, existingMarkdown, includeTools }) => {
            const { sections } = await buildContentPageWithAi(courseCode, itemId, {
              prompt,
              existingMarkdown: existingMarkdown || undefined,
              includeTools,
            })
            return sections
          }}
          onBuilt={applyBuiltSections}
        />
      ) : null}
      {aiStudyBuddyEnabled && courseCode ? <StudyBuddyWidget courseCode={courseCode} /> : null}
      {InputDialogHost}
    </LmsPage>
  )
}

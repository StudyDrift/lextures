import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'
import { X } from 'lucide-react'
import { formatDateTime } from '../../lib/format'
import {
  deleteSubmissionAnnotation,
  downloadSubmissionAttachmentsArchive,
  fetchModuleAssignmentMySubmission,
  fetchModuleAssignmentSubmissions,
  fetchProvisionalGrades,
  fetchSubmissionAnnotations,
  fetchSubmissionFeedbackMedia,
  fetchSubmissionOriginality,
  fetchSubmissionOriginalityEmbed,
  retrySubmissionOriginality,
  fetchSubmissionVersions,
  postProvisionalGrade,
  postRequestAssignmentRevision,
  postSubmissionAnnotation,
  revealModuleAssignmentIdentities,
  submitModuleAssignmentText,
  uploadModuleAssignmentSubmissionFile,
  type ModuleAssignmentSubmissionApi,
  type OriginalityReportApi,
  type OriginalityReportSummary,
  type PostSubmissionAnnotationInput,
  type SubmissionAnnotationApi,
  type SubmissionFeedbackMediaApi,
  submissionAttachmentsFromRow,
  type SubmissionAttachmentApi,
  type SubmissionVersionApi,
} from '../../lib/courses-api'
import { OriginalityBadge } from '../grading/originality-badge'
import { OriginalityReportViewer } from '../grading/originality-report-viewer'
import { getJwtSubject } from '../../lib/auth'
import { AnnotationCommentPanel } from './annotation-comment-panel'
import { AnnotationToolbar, type AnnotationTool } from './annotation-toolbar'
import { AnnotationViewer } from './annotation-viewer'
import { FeedbackMediaPlayerList } from './feedback-media-player'
import { FeedbackMediaRecorder } from './feedback-media-recorder'
import { FilePreviewBody } from '../file-preview'
import { detectPreviewType } from '../../lib/file-type'
import { SubmissionNavigator } from './submission-navigator'
import { useSpeedGraderHotkeys } from './speed-grader-shortcuts'
import {
  defaultSubmissionIndex,
  sortSubmissionsByStudentLabel,
  submissionsMatch,
  type GradedFilter,
} from './submission-navigator-utils'
import { ResizableSplitPane } from '../layout/resizable-split-pane'
import { SubmissionPreviewSidebar } from './submission-preview-sidebar'
import type { RubricDefinition } from '../../lib/courses-api'
import { QuizSpeedGraderBranch } from '../quiz/quiz-speed-grader-branch'

function submissionContentPath(contentPath?: string | null): string | null {
  const trimmed = contentPath?.trim()
  return trimmed || null
}

export type AssignmentAnnotationWorkbenchProps = {
  courseCode: string
  itemId: string
  /** `quiz` opens quiz attempt SpeedGrader; default is assignment submissions. */
  itemKind?: 'assignment' | 'quiz'
  /** Assignment title (in-app message / banners). */
  assignmentTitle?: string
  /** `staff` uses roster navigation; `student` loads only the viewer’s submission. */
  mode: 'staff' | 'student'
  submissionAllowsFile: boolean
  submissionAllowsText?: boolean
  submissionAllowsUrl?: boolean
  /**
   * When true, show document annotation (requires file upload allowed). Default: same as
   * `submissionAllowsFile` for backwards compatibility.
   */
  annotationsActive?: boolean
  /** Server `FEEDBACK_MEDIA_ENABLED` — A/V feedback (plan 3.2). */
  feedbackMediaEnabled?: boolean
  /** Plan 3.3 — show blind grading banner and anonymised labels. */
  blindGradingActive?: boolean
  /** Course creator may reveal identities (from assignment GET). */
  canRevealIdentities?: boolean
  /** Refresh assignment metadata after reveal. */
  onAfterRevealIdentities?: () => void
  /** Plan 3.4 — show provisional score entry for listed graders. */
  moderatedGradingActive?: boolean
  assignmentPointsWorth?: number | null
  assignmentRubric?: RubricDefinition | null
  provisionalGraderUserIds?: string[]
  /** Plan 3.5 — from assignment settings; when not `disabled`, originality API is polled. */
  originalityDetection?: 'disabled' | 'plagiarism' | 'ai' | 'both'
  /** Plan 3.13 — server `RESUBMISSION_WORKFLOW_ENABLED`. */
  resubmissionWorkflowEnabled?: boolean
  /** Staff preview-only mode opens in a near-full-screen modal instead of inline. */
  presentation?: 'inline' | 'modal'
  modalOpen?: boolean
  onModalClose?: () => void
  /** When set, staff navigation opens on this student's submission after load. */
  initialStudentUserId?: string | null
}

export function AssignmentAnnotationWorkbench(props: AssignmentAnnotationWorkbenchProps) {
  if (props.itemKind === 'quiz') {
    return (
      <QuizSpeedGraderBranch
        courseCode={props.courseCode}
        itemId={props.itemId}
        quizTitle={props.assignmentTitle}
        presentation={props.presentation}
        modalOpen={props.modalOpen}
        onModalClose={props.onModalClose}
        initialStudentUserId={props.initialStudentUserId}
      />
    )
  }

  return <AssignmentAnnotationWorkbenchInner {...props} />
}

function AssignmentAnnotationWorkbenchInner({
  courseCode,
  itemId,
  mode,
  submissionAllowsFile,
  submissionAllowsText = false,
  submissionAllowsUrl = false,
  annotationsActive: annotationsActiveProp,
  feedbackMediaEnabled = false,
  blindGradingActive = false,
  canRevealIdentities = false,
  onAfterRevealIdentities,
  moderatedGradingActive = false,
  assignmentPointsWorth = null,
  assignmentRubric = null,
  provisionalGraderUserIds = [],
  originalityDetection = 'disabled',
  assignmentTitle = 'Assignment',
  resubmissionWorkflowEnabled = false,
  presentation = 'inline',
  modalOpen = false,
  onModalClose,
  initialStudentUserId = null,
}: AssignmentAnnotationWorkbenchProps) {
  const annotationsActive = annotationsActiveProp ?? submissionAllowsFile
  const [panel, setPanel] = useState<'document' | 'media'>('document')
  const [mediaItems, setMediaItems] = useState<SubmissionFeedbackMediaApi[]>([])
  const [gradedFilter, setGradedFilter] = useState<GradedFilter>('all')
  const [submissions, setSubmissions] = useState<ModuleAssignmentSubmissionApi[]>([])
  const [idx, setIdx] = useState(0)
  const staffNavRef = useRef({ submissions, idx })
  staffNavRef.current = { submissions, idx }
  const [mine, setMine] = useState<ModuleAssignmentSubmissionApi | null>(null)
  const [annotations, setAnnotations] = useState<SubmissionAnnotationApi[]>([])
  const [tool, setTool] = useState<AnnotationTool>('highlight')
  const [colour, setColour] = useState('#FFFF00')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [provisionalInput, setProvisionalInput] = useState('')
  const [provisionalBusy, setProvisionalBusy] = useState(false)
  const [originalityReports, setOriginalityReports] = useState<OriginalityReportApi[] | null>(null)
  const [originalityViewerOpen, setOriginalityViewerOpen] = useState(false)
  const [originalityEmbedUrl, setOriginalityEmbedUrl] = useState<string | null>(null)
  const [originalitySummary, setOriginalitySummary] = useState<OriginalityReportSummary | null>(null)
  const [originalityViewSummaryOnly, setOriginalityViewSummaryOnly] = useState(false)
  const [submissionVersions, setSubmissionVersions] = useState<SubmissionVersionApi[]>([])
  const [viewVersionNumber, setViewVersionNumber] = useState<number | null>(null)
  const [revisionFormOpen, setRevisionFormOpen] = useState(false)
  const [revDueLocal, setRevDueLocal] = useState('')
  const [revFeedback, setRevFeedback] = useState('')
  const [revisionBusy, setRevisionBusy] = useState(false)
  const [draftText, setDraftText] = useState('')
  const [deadlineNow, setDeadlineNow] = useState(() => Date.now())
  const [selectedAttachmentId, setSelectedAttachmentId] = useState<string | null>(null)
  const [downloadAllBusy, setDownloadAllBusy] = useState(false)

  const originalityActive = originalityDetection !== 'disabled'

  const current: ModuleAssignmentSubmissionApi | null =
    mode === 'staff' ? (submissions[idx] ?? null) : mine
  const readOnly = mode === 'student'
  const failedOriginality = originalityReports?.some((r) => r.status === 'failed') ?? false

  async function onRetryOriginality() {
    if (!current?.id) return
    setBusy(true)
    try {
      const n = await retrySubmissionOriginality(courseCode, itemId, current.id)
      if (n === 0) {
        window.alert('No failed scans to retry.')
      }
      await reloadOriginality()
    } catch (e) {
      window.alert(e instanceof Error ? e.message : 'Retry failed.')
    } finally {
      setBusy(false)
    }
  }

  useEffect(() => {
    setViewVersionNumber(null)
    setSelectedAttachmentId(null)
  }, [current?.id])

  useEffect(() => {
    const submissionId = current?.id
    if (!resubmissionWorkflowEnabled || mode !== 'staff' || !submissionId) {
      setSubmissionVersions([])
      return
    }
    let c = true
    void (async () => {
      try {
        const v = await fetchSubmissionVersions(courseCode, itemId, submissionId)
        if (!c) return
        setSubmissionVersions(v)
        setViewVersionNumber((prev) => {
          if (prev == null) return v.length > 0 ? (v[v.length - 1]?.versionNumber ?? null) : null
          return v.some((x) => x.versionNumber === prev) ? prev : (v[v.length - 1]?.versionNumber ?? null)
        })
      } catch {
        if (c) setSubmissionVersions([])
      }
    })()
    return () => {
      c = false
    }
  }, [resubmissionWorkflowEnabled, courseCode, itemId, mode, current?.id])

  const versionForView = useMemo((): SubmissionVersionApi | null => {
    if (mode !== 'staff' || !resubmissionWorkflowEnabled || !current?.id || submissionVersions.length === 0) {
      return null
    }
    const n = viewVersionNumber ?? submissionVersions[submissionVersions.length - 1]?.versionNumber
    if (n == null) return null
    return submissionVersions.find((v) => v.versionNumber === n) ?? null
  }, [current?.id, mode, resubmissionWorkflowEnabled, submissionVersions, viewVersionNumber])

  const viewIsLatest =
    versionForView == null
      ? true
      : versionForView.versionNumber === (current?.versionNumber ?? 0)

  const displayAttachments = useMemo((): SubmissionAttachmentApi[] => {
    if (versionForView) {
      if (versionForView.attachmentFileId && versionForView.attachmentContentPath) {
        return [
          {
            fileId: versionForView.attachmentFileId,
            filename: versionForView.attachmentFilename ?? 'submission',
            mimeType: versionForView.attachmentMimeType ?? 'application/octet-stream',
            contentPath: versionForView.attachmentContentPath,
          },
        ]
      }
      return []
    }
    return submissionAttachmentsFromRow(current)
  }, [current, versionForView])

  const selectedAttachment = useMemo((): SubmissionAttachmentApi | null => {
    if (displayAttachments.length === 0) return null
    if (selectedAttachmentId) {
      const found = displayAttachments.find((file) => file.fileId === selectedAttachmentId)
      if (found) return found
    }
    return displayAttachments[0] ?? null
  }, [displayAttachments, selectedAttachmentId])

  const displayAttachmentFileId = selectedAttachment?.fileId ?? versionForView?.attachmentFileId ?? current?.attachmentFileId ?? null
  const displayFilePath = submissionContentPath(selectedAttachment?.contentPath)
  const displayMimeType = selectedAttachment?.mimeType ?? versionForView?.attachmentMimeType ?? current?.attachmentMimeType
  const displayFilename = selectedAttachment?.filename ?? versionForView?.attachmentFilename ?? current?.attachmentFilename ?? 'submission'
  const displayPreviewType = displayFilePath
    ? detectPreviewType(displayMimeType, displayFilename)
    : 'none'
  const displayBodyText = current?.bodyText?.trim() ?? ''
  const readOnlyDocument =
    readOnly || (mode === 'staff' && resubmissionWorkflowEnabled && !viewIsLatest)

  useEffect(() => {
    if (mode === 'student' && mine?.resubmissionRequested && mine.revisionDueAt) {
      const t = window.setInterval(() => setDeadlineNow(Date.now()), 15_000)
      return () => clearInterval(t)
    }
    return
  }, [mode, mine?.resubmissionRequested, mine?.revisionDueAt])

  const reloadStaffList = useCallback(async () => {
    if (mode !== 'staff') return
    setLoadError(null)
    try {
      const list = await fetchModuleAssignmentSubmissions(courseCode, itemId, { graded: gradedFilter })
      const sorted = sortSubmissionsByStudentLabel(list)
      const preserveCurrent = staffNavRef.current.submissions[staffNavRef.current.idx]
      setSubmissions(sorted)
      setIdx(() => {
        if (initialStudentUserId) {
          const targetIdx = sorted.findIndex((s) => s.submittedBy === initialStudentUserId)
          if (targetIdx >= 0) return targetIdx
        }
        if (preserveCurrent) {
          const nextIdx = sorted.findIndex((s) => submissionsMatch(s, preserveCurrent))
          if (nextIdx >= 0) return nextIdx
        }
        return defaultSubmissionIndex(sorted)
      })
    } catch (e) {
      setSubmissions([])
      setLoadError(e instanceof Error ? e.message : 'Could not load submissions.')
    }
  }, [courseCode, initialStudentUserId, itemId, gradedFilter, mode])

  const reloadMine = useCallback(async () => {
    if (mode !== 'student') return
    setLoadError(null)
    try {
      const row = await fetchModuleAssignmentMySubmission(courseCode, itemId)
      setMine(row)
    } catch (e) {
      setMine(null)
      setLoadError(e instanceof Error ? e.message : 'Could not load your submission.')
    }
  }, [courseCode, itemId, mode])

  useEffect(() => {
    if (presentation === 'modal' && !modalOpen) return
    if (mode === 'staff') void reloadStaffList()
    else void reloadMine()
  }, [mode, reloadMine, reloadStaffList, presentation, modalOpen])

  const handleGradeSaved = useCallback(() => {
    if (!current) return
    setSubmissions((prev) =>
      prev.map((sub) =>
        submissionsMatch(sub, current) ? { ...sub, isGraded: true } : sub
      )
    )
  }, [current])

  const handleGradeCleared = useCallback(() => {
    if (!current) return
    setSubmissions((prev) =>
      prev.map((sub) =>
        submissionsMatch(sub, current) ? { ...sub, isGraded: false } : sub
      )
    )
  }, [current])

  const myUid = getJwtSubject()
  const isListedGrader = Boolean(
    mode === 'staff' &&
      moderatedGradingActive &&
      myUid &&
      provisionalGraderUserIds.includes(myUid),
  )

  useEffect(() => {
    if (!isListedGrader || !current?.id) {
      setProvisionalInput('')
      return
    }
    let cancel = false
    void (async () => {
      try {
        const rows = await fetchProvisionalGrades(courseCode, itemId)
        const mine = rows.find((r) => r.submissionId === current.id && r.graderId === myUid)
        if (!cancel) setProvisionalInput(mine ? String(mine.score) : '')
      } catch {
        if (!cancel) setProvisionalInput('')
      }
    })()
    return () => {
      cancel = true
    }
  }, [courseCode, itemId, current?.id, isListedGrader, myUid])

  const reloadAnnotations = useCallback(async () => {
    if (!annotationsActive || !current?.id) {
      setAnnotations([])
      return
    }
    if (mode === 'staff' && resubmissionWorkflowEnabled && !viewIsLatest) {
      setAnnotations([])
      return
    }
    try {
      const list = await fetchSubmissionAnnotations(courseCode, itemId, current.id)
      setAnnotations(list)
    } catch {
      setAnnotations([])
    }
  }, [
    annotationsActive,
    courseCode,
    itemId,
    current?.id,
    mode,
    resubmissionWorkflowEnabled,
    viewIsLatest,
  ])

  const reloadMedia = useCallback(async () => {
    if (!feedbackMediaEnabled || !current?.id) {
      setMediaItems([])
      return
    }
    try {
      const list = await fetchSubmissionFeedbackMedia(courseCode, itemId, current.id)
      setMediaItems(list)
    } catch {
      setMediaItems([])
    }
  }, [courseCode, current?.id, feedbackMediaEnabled, itemId])

  useEffect(() => {
    void reloadAnnotations()
  }, [reloadAnnotations])

  useEffect(() => {
    void reloadMedia()
  }, [reloadMedia])

  const reloadOriginality = useCallback(async () => {
    if (!originalityActive || !current?.id) {
      setOriginalityReports(null)
      return
    }
    try {
      const reps = await fetchSubmissionOriginality(courseCode, itemId, current.id)
      setOriginalityReports(reps ?? [])
    } catch {
      setOriginalityReports([])
    }
  }, [courseCode, itemId, current?.id, originalityActive])

  useEffect(() => {
    void reloadOriginality()
  }, [reloadOriginality])

  useEffect(() => {
    if (!originalityActive || !current?.id) return
    const t = window.setInterval(() => void reloadOriginality(), 8000)
    return () => window.clearInterval(t)
  }, [current?.id, originalityActive, reloadOriginality])

  useEffect(() => {
    if (annotationsActive && !feedbackMediaEnabled) setPanel('document')
    if (!annotationsActive && feedbackMediaEnabled) setPanel('media')
  }, [annotationsActive, feedbackMediaEnabled])

  async function persistAnnotation(
    payload: PostSubmissionAnnotationInput,
  ): Promise<SubmissionAnnotationApi | null> {
    if (!current?.id || readOnlyDocument) return null
    setBusy(true)
    try {
      const created = await postSubmissionAnnotation(courseCode, itemId, current.id, payload)
      await reloadAnnotations()
      return created
    } catch (e) {
      window.alert(e instanceof Error ? e.message : 'Could not save annotation.')
      return null
    } finally {
      setBusy(false)
    }
  }

  // Create an annotation with no comment yet, then select it so the comment box opens — the
  // grader types the comment inline instead of through a blocking prompt.
  async function createAndSelect(
    payload: Omit<PostSubmissionAnnotationInput, 'clientId' | 'colour'>,
  ) {
    const created = await persistAnnotation({
      clientId: crypto.randomUUID(),
      colour,
      ...payload,
    })
    if (created) setSelectedId(created.id)
  }

  const onHighlightComplete = (page: number, rects: { x1: number; y1: number; x2: number; y2: number }[]) => {
    if (rects.length === 0) return
    void createAndSelect({ page, toolType: 'highlight', coordsJson: { rects } })
  }

  const onDrawComplete = (page: number, points: { x: number; y: number }[]) => {
    void createAndSelect({ page, toolType: 'draw', coordsJson: { points } })
  }

  const onPinComplete = (page: number, pt: { x: number; y: number }) => {
    void createAndSelect({ page, toolType: 'pin', coordsJson: pt })
  }

  const onTextBoxComplete = (page: number, rect: { x1: number; y1: number; x2: number; y2: number }) => {
    void createAndSelect({ page, toolType: 'text', coordsJson: rect })
  }

  // Re-post the same clientId so the server upserts the body in place (edit the comment on an
  // existing highlight/drawing/pin).
  async function onUpdateAnnotationBody(annotation: SubmissionAnnotationApi, body: string) {
    if (!current?.id || readOnlyDocument) return
    setBusy(true)
    try {
      await postSubmissionAnnotation(courseCode, itemId, current.id, {
        clientId: annotation.clientId,
        page: annotation.page,
        toolType: annotation.toolType as PostSubmissionAnnotationInput['toolType'],
        colour: annotation.colour,
        coordsJson: annotation.coordsJson,
        body: body.trim() || undefined,
      })
      await reloadAnnotations()
    } catch (e) {
      window.alert(e instanceof Error ? e.message : 'Could not save comment.')
    } finally {
      setBusy(false)
    }
  }

  async function onDeleteAnnotation(id: string) {
    if (!current?.id || readOnlyDocument) return
    if (!window.confirm('Delete this annotation?')) return
    setBusy(true)
    try {
      await deleteSubmissionAnnotation(courseCode, itemId, current.id, id)
      await reloadAnnotations()
    } catch (e) {
      window.alert(e instanceof Error ? e.message : 'Could not delete.')
    } finally {
      setBusy(false)
    }
  }

  async function onRequestRevision() {
    if (!current?.id) return
    setRevisionBusy(true)
    try {
      let revisionDueAt: string | null = null
      if (revDueLocal.trim()) {
        const d = new Date(revDueLocal)
        if (Number.isNaN(d.getTime())) {
          window.alert('Use a valid date and time for the revision deadline.')
          return
        }
        revisionDueAt = d.toISOString()
      }
      await postRequestAssignmentRevision(courseCode, itemId, current.id, {
        revisionDueAt,
        revisionFeedback: revFeedback.trim() || null,
      })
      setRevisionFormOpen(false)
      setRevDueLocal('')
      setRevFeedback('')
      await reloadStaffList()
    } catch (e) {
      window.alert(e instanceof Error ? e.message : 'Request failed.')
    } finally {
      setRevisionBusy(false)
    }
  }

  async function onSubmitStudentText() {
    if (mode !== 'student' || !submissionAllowsText) return
    const text = draftText.trim()
    if (!text) return
    setBusy(true)
    try {
      await submitModuleAssignmentText(courseCode, itemId, text)
      setDraftText('')
      await reloadMine()
    } catch (e) {
      window.alert(e instanceof Error ? e.message : 'Submit failed.')
    } finally {
      setBusy(false)
    }
  }

  async function onUploadStudentFile(file: File | null) {
    if (!file || mode !== 'student') return
    if (
      resubmissionWorkflowEnabled &&
      mine &&
      mine.attachmentFileId &&
      !mine.resubmissionRequested
    ) {
      window.alert(
        'Resubmission is not open. Your instructor must request a revision before you can upload a new file.',
      )
      return
    }
    setBusy(true)
    try {
      await uploadModuleAssignmentSubmissionFile(courseCode, itemId, file)
      await reloadMine()
    } catch (e) {
      window.alert(e instanceof Error ? e.message : 'Upload failed.')
    } finally {
      setBusy(false)
    }
  }

  async function onRevealIdentities() {
    if (
      !window.confirm(
        'You are about to unmask student identities for this assignment. This cannot be undone. Continue?',
      )
    ) {
      return
    }
    setBusy(true)
    try {
      try {
        await revealModuleAssignmentIdentities(courseCode, itemId, { force: false })
      } catch (e) {
        const msg = e instanceof Error ? e.message : ''
        if (
          msg.toLowerCase().includes('ungraded') &&
          window.confirm(
            'Some submissions are still ungraded. Reveal identities anyway? This cannot be undone.',
          )
        ) {
          await revealModuleAssignmentIdentities(courseCode, itemId, { force: true })
        } else {
          throw e
        }
      }
      onAfterRevealIdentities?.()
      if (mode === 'staff') void reloadStaffList()
    } catch (err) {
      window.alert(err instanceof Error ? err.message : 'Could not reveal identities.')
    } finally {
      setBusy(false)
    }
  }

  async function onOpenOriginalityReport() {
    if (!current?.id) return
    setBusy(true)
    try {
      const { embedUrl, summary } = await fetchSubmissionOriginalityEmbed(courseCode, itemId, current.id)
      if (embedUrl) {
        let url = embedUrl
        if (!/^https?:\/\//i.test(url)) {
          url = `${window.location.origin}${url.startsWith('/') ? '' : '/'}${url}`
        }
        setOriginalityEmbedUrl(url)
        setOriginalitySummary(summary)
        setOriginalityViewSummaryOnly(false)
        setOriginalityViewerOpen(true)
        return
      }
      if (summary) {
        setOriginalityEmbedUrl(null)
        setOriginalitySummary(summary)
        setOriginalityViewSummaryOnly(true)
        setOriginalityViewerOpen(true)
        return
      }
      window.alert('No originality report is available yet for this submission.')
    } catch (e) {
      window.alert(e instanceof Error ? e.message : 'No originality report is available yet.')
    } finally {
      setBusy(false)
    }
  }

  async function onDownloadAllAttachments() {
    if (!current?.id) return
    setDownloadAllBusy(true)
    try {
      await downloadSubmissionAttachmentsArchive(courseCode, itemId, current.id)
    } catch (e) {
      window.alert(e instanceof Error ? e.message : 'Download failed.')
    } finally {
      setDownloadAllBusy(false)
    }
  }

  const submissionReviewActive =
    mode === 'staff' && (submissionAllowsFile || submissionAllowsText || submissionAllowsUrl)
  const filePreviewActive = submissionAllowsFile || submissionAllowsText || submissionAllowsUrl
  const showDocPanel = annotationsActive || submissionReviewActive || (mode === 'student' && filePreviewActive)
  const showMediaPanel = feedbackMediaEnabled
  const both = showDocPanel && showMediaPanel
  const staffPreviewOnly = mode === 'staff' && filePreviewActive && !annotationsActive
  const staffGradingSidebarActive = mode === 'staff' && submissionReviewActive

  useSpeedGraderHotkeys({
    enabled: staffGradingSidebarActive && (presentation !== 'modal' || modalOpen),
    disabled: busy,
    submissions,
    index: idx,
    onIndexChange: setIdx,
  })
  const previewErrorVariant = presentation === 'modal' ? 'message-only' : 'standalone'
  const gradingSidebar = staffGradingSidebarActive ? (
    <SubmissionPreviewSidebar
      mode={mode}
      courseCode={courseCode}
      itemId={itemId}
      submissionId={current?.id ?? null}
      studentUserId={current?.submittedBy ?? null}
      rubric={assignmentRubric}
      maxPoints={assignmentPointsWorth}
      gradingDisabled={busy}
      files={displayAttachments}
      selectedFileId={selectedAttachment?.fileId ?? null}
      onSelectFile={setSelectedAttachmentId}
      submittedAt={current?.submittedAt}
      blindLabel={current?.blindLabel}
      onDownloadAll={displayAttachments.length > 1 ? onDownloadAllAttachments : undefined}
      downloadAllBusy={downloadAllBusy}
      onGradeSaved={handleGradeSaved}
      onGradeCleared={handleGradeCleared}
    />
  ) : null
  const studentFeedbackSidebar =
    mode === 'student' && current?.id ? (
      <SubmissionPreviewSidebar
        mode="student"
        courseCode={courseCode}
        itemId={itemId}
        submissionId={current.id}
        rubric={assignmentRubric}
        maxPoints={assignmentPointsWorth}
        files={displayAttachments}
        selectedFileId={selectedAttachment?.fileId ?? null}
        onSelectFile={setSelectedAttachmentId}
        submittedAt={current.submittedAt}
        onDownloadAll={displayAttachments.length > 1 ? onDownloadAllAttachments : undefined}
        downloadAllBusy={downloadAllBusy}
      />
    ) : null
  const modalTitleId = useId()
  const modalCloseRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (presentation !== 'modal' || !modalOpen) return
    const t = window.setTimeout(() => modalCloseRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [presentation, modalOpen])

  useEffect(() => {
    if (presentation !== 'modal' || !modalOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onModalClose?.()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [presentation, modalOpen, onModalClose])

  if (!showDocPanel && !showMediaPanel) {
    return null
  }

  if (presentation === 'modal' && !modalOpen) {
    return null
  }

  const sectionAriaLabel =
    mode === 'staff'
      ? annotationsActive
        ? 'Submission annotations'
        : 'Submission preview'
      : 'Your submission'

  const sectionTitle =
    mode === 'staff'
      ? annotationsActive
        ? 'SpeedGrader'
        : assignmentTitle
      : 'Your submission'

  const documentPreviewContent =
    displayFilePath ? (
      annotationsActive && (displayPreviewType === 'pdf' || displayPreviewType === 'image') ? (
        <AnnotationViewer
          filePath={displayFilePath}
          mimeType={displayMimeType ?? null}
          filename={displayFilename}
          readOnly={readOnlyDocument || staffPreviewOnly}
          fallbackVariant={previewErrorVariant}
          tool={tool}
          colour={colour}
          annotations={annotations}
          selectedId={selectedId}
          onSelectAnnotation={setSelectedId}
          onHighlightComplete={
            annotationsActive && !readOnlyDocument ? onHighlightComplete : undefined
          }
          onDrawComplete={
            annotationsActive && !readOnlyDocument ? onDrawComplete : undefined
          }
          onPinComplete={
            annotationsActive && !readOnlyDocument ? onPinComplete : undefined
          }
          onTextBoxComplete={
            annotationsActive && !readOnlyDocument ? onTextBoxComplete : undefined
          }
        />
      ) : (
        <FilePreviewBody
          filePath={displayFilePath}
          filename={displayFilename}
          mimeType={displayMimeType}
          errorVariant={previewErrorVariant}
          className="h-full min-h-[40vh]"
        />
      )
    ) : displayBodyText ? (
      <div className="h-full min-h-[40vh] overflow-y-auto rounded-lg border border-slate-200 bg-white px-4 py-4 text-sm text-slate-800 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100">
        <p className="whitespace-pre-wrap">{displayBodyText}</p>
      </div>
    ) : displayAttachmentFileId ? (
      <div className="flex h-full min-h-48 items-center justify-center rounded-lg border border-dashed border-amber-200 bg-amber-50 px-4 py-6 text-sm text-amber-950 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-100">
        <p>This submission references a file that could not be loaded. Try downloading from the panel on the right, or re-import submissions from Canvas.</p>
      </div>
    ) : (
      <div className="flex h-full min-h-48 items-center justify-center rounded-lg border border-dashed border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-900/60 dark:text-neutral-300">
        {mode === 'staff' ? (
          current?.submittedAt ? (
            <p>
              This student submitted on{' '}
              {formatDateTime(current.submittedAt, {
                dateStyle: 'medium',
                timeStyle: 'short',
              })}
              , but no submission content is on file yet.
            </p>
          ) : (
            <p>No submission from this student yet.</p>
          )
        ) : submissionAllowsText && mode === 'student' ? (
          <div className="w-full max-w-xl space-y-3 text-start">
            <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
              Your response
            </label>
            <textarea
              value={draftText}
              onChange={(e) => setDraftText(e.target.value)}
              rows={8}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-50"
              placeholder="Type your answer here…"
              disabled={busy}
            />
            <button
              type="button"
              onClick={() => void onSubmitStudentText()}
              disabled={busy || draftText.trim() === ''}
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {busy ? 'Submitting…' : 'Submit text'}
            </button>
          </div>
        ) : submissionAllowsFile ? (
          <p>Upload a file to submit this assignment.</p>
        ) : (
          <p>No submission on file yet.</p>
        )}
      </div>
    )

  // Shared markup surface: annotation toolbar above, document + comment list side by side.
  // Used by both the inline and modal (preview / gradebook) layouts so annotation works the
  // same everywhere.
  const annotationDocArea = (
    <div className="flex h-full min-h-0 flex-col gap-4">
      {annotationsActive && !readOnlyDocument ? (
        <AnnotationToolbar
          tool={tool}
          onToolChange={setTool}
          colour={colour}
          onColourChange={setColour}
          disabled={busy || !(current?.attachmentFileId || displayFilePath)}
          readOnly={readOnlyDocument}
        />
      ) : null}
      <div className="flex min-h-0 flex-1 flex-col gap-4 lg:flex-row lg:items-start">
        <div className="min-h-0 min-w-0 flex-1">{documentPreviewContent}</div>
        {annotationsActive ? (
          <AnnotationCommentPanel
            annotations={annotations}
            selectedId={selectedId}
            onSelect={setSelectedId}
            readOnly={readOnlyDocument}
            onDelete={readOnlyDocument ? undefined : onDeleteAnnotation}
            onUpdateBody={readOnlyDocument ? undefined : onUpdateAnnotationBody}
          />
        ) : null}
      </div>
    </div>
  )

  if (presentation === 'modal') {
    return (
      <div
        className="fixed inset-0 z-[500] flex items-center justify-center p-3 sm:p-6"
        role="presentation"
      >
        <button
          type="button"
          aria-label="Close submission preview backdrop"
          className="absolute inset-0 cursor-default border-0 bg-slate-950/55 p-0 backdrop-blur-[2px] dark:bg-black/80"
          onClick={onModalClose}
          tabIndex={-1}
        />
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby={modalTitleId}
          className="relative z-10 flex w-full max-w-[min(96vw,1600px)] flex-col overflow-hidden rounded-2xl border border-slate-300 bg-white shadow-[0_24px_80px_-12px_rgba(15,23,42,0.55)] ring-1 ring-slate-900/10 dark:border-neutral-500 dark:bg-neutral-900 dark:shadow-[0_24px_80px_-12px_rgba(0,0,0,0.85)] dark:ring-white/10"
          style={{ height: 'min(92vh, 1080px)', maxHeight: 'calc(100dvh - 1.5rem)' }}
        >
          <div className="flex shrink-0 flex-wrap items-center gap-3 border-b border-slate-200 bg-slate-50 px-4 py-3 dark:border-neutral-600 dark:bg-neutral-800">
            <h2
              id={modalTitleId}
              className="text-base font-semibold text-slate-900 dark:text-neutral-50"
            >
              {sectionTitle}
            </h2>
            {mode === 'staff' ? (
              <div className="flex flex-1 flex-wrap items-center justify-end gap-2">
                {canRevealIdentities ? (
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => void onRevealIdentities()}
                    className="rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-950 hover:bg-amber-100 disabled:opacity-50 dark:border-amber-800 dark:bg-amber-950/50 dark:text-amber-100 dark:hover:bg-amber-900/60"
                  >
                    Reveal identities
                  </button>
                ) : null}
                <SubmissionNavigator
                  submissions={submissions}
                  index={idx}
                  onIndexChange={setIdx}
                  gradedFilter={gradedFilter}
                  onGradedFilterChange={(f) => {
                    setGradedFilter(f)
                    setIdx(0)
                  }}
                  disabled={busy}
                  showShortcuts
                  anonymisedAriaLabel={
                    current?.blindLabel
                      ? `Anonymised student, label ${current.blindLabel}`
                      : undefined
                  }
                />
              </div>
            ) : null}
            <button
              ref={modalCloseRef}
              type="button"
              onClick={onModalClose}
              className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
              aria-label="Close submission preview"
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          {loadError ? (
            <p className="shrink-0 border-b border-rose-200 bg-rose-50 px-4 py-2 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200">
              {loadError}
            </p>
          ) : null}

          {mode === 'staff' && blindGradingActive ? (
            <p
              role="status"
              className="shrink-0 border-b border-indigo-200 bg-indigo-50/90 px-4 py-2 text-sm text-indigo-950 dark:border-indigo-900/60 dark:bg-indigo-950/40 dark:text-indigo-100"
            >
              Blind grading is active — student identities are hidden.
            </p>
          ) : null}

          <ResizableSplitPane
            storageKey="lextures:submission-grade-sidebar-width"
            primary={
              <div className="h-full min-h-[40vh] overflow-auto bg-slate-50 p-3 dark:bg-neutral-800/60">
                {annotationDocArea}
              </div>
            }
            secondary={gradingSidebar}
          />
        </div>
      </div>
    )
  }

  return (
    <section
      id="submission-preview"
      tabIndex={-1}
      aria-label={sectionAriaLabel}
      className="scroll-mt-20 mt-8 space-y-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-950"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0 space-y-2">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
            {sectionTitle}
          </h2>
          {originalityActive && originalityReports && originalityReports.length > 0 ? (
            <div className="flex flex-wrap items-center gap-2">
              <OriginalityBadge reports={originalityReports} />
              {mode === 'staff' ? (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void onOpenOriginalityReport()}
                  className="rounded-md border border-slate-200 bg-white px-2.5 py-1 text-xs font-semibold text-indigo-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-950 dark:text-indigo-300 dark:hover:bg-neutral-900"
                >
                  View report
                </button>
              ) : null}
              {mode === 'staff' && failedOriginality ? (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void onRetryOriginality()}
                  className="rounded-md border border-rose-300 bg-rose-50 px-2.5 py-1 text-xs font-semibold text-rose-900 hover:bg-rose-100 disabled:opacity-50 dark:border-rose-900 dark:bg-rose-950/50 dark:text-rose-100"
                >
                  Retry scan
                </button>
              ) : null}
            </div>
          ) : null}
        </div>
        {mode === 'staff' ? (
          <div className="flex flex-wrap items-center gap-2">
            {resubmissionWorkflowEnabled && current && current.submittedBy && (
              <button
                type="button"
                disabled={busy}
                onClick={() => {
                  setRevisionFormOpen((o) => !o)
                  if (!revDueLocal && !revFeedback) {
                    setRevDueLocal('')
                  }
                }}
                className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold text-slate-900 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800/80"
              >
                Request revision
              </button>
            )}
            {canRevealIdentities ? (
              <button
                type="button"
                disabled={busy}
                onClick={() => void onRevealIdentities()}
                className="rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-950 hover:bg-amber-100 disabled:opacity-50 dark:border-amber-800 dark:bg-amber-950/50 dark:text-amber-100 dark:hover:bg-amber-900/60"
              >
                Reveal identities
              </button>
            ) : null}
            <SubmissionNavigator
              submissions={submissions}
              index={idx}
              onIndexChange={setIdx}
              gradedFilter={gradedFilter}
              onGradedFilterChange={(f) => {
                setGradedFilter(f)
                setIdx(0)
              }}
              disabled={busy}
              showShortcuts
              anonymisedAriaLabel={
                current?.blindLabel
                  ? `Anonymised student, label ${current.blindLabel}`
                  : undefined
              }
            />
          </div>
        ) : null}
      </div>

      {revisionFormOpen && resubmissionWorkflowEnabled && mode === 'staff' && current?.id && (
        <div className="rounded-lg border border-slate-200 bg-slate-50/90 p-4 dark:border-neutral-600 dark:bg-neutral-900/60">
          <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">
            Request a revision: {assignmentTitle}
          </p>
          <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
            The student can resubmit a new file while a revision is open. Optional: set a resubmission
            deadline.
          </p>
          <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-end">
            <label className="text-sm text-slate-700 dark:text-neutral-200">
              <span className="mb-1 block text-xs font-semibold text-slate-500 dark:text-neutral-400">
                Resubmit by
              </span>
              <input
                type="datetime-local"
                className="rounded-md border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                value={revDueLocal}
                onChange={(e) => setRevDueLocal(e.target.value)}
                disabled={revisionBusy}
              />
            </label>
            <div className="min-w-0 flex-1">
              <span className="mb-1 block text-xs font-semibold text-slate-500 dark:text-neutral-400">
                Feedback
              </span>
              <textarea
                className="min-h-20 w-full rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                value={revFeedback}
                onChange={(e) => setRevFeedback(e.target.value)}
                rows={3}
                disabled={revisionBusy}
                placeholder="What should the student change before resubmitting?"
              />
            </div>
          </div>
          <div className="mt-3 flex flex-wrap gap-2">
            <button
              type="button"
              className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
              disabled={revisionBusy}
              onClick={() => void onRequestRevision()}
            >
              {revisionBusy ? 'Saving…' : 'Send revision request'}
            </button>
            <button
              type="button"
              className="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-900"
              onClick={() => {
                setRevisionFormOpen(false)
                setRevDueLocal('')
                setRevFeedback('')
              }}
              disabled={revisionBusy}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {resubmissionWorkflowEnabled && mode === 'staff' && submissionVersions.length > 0 && current?.id ? (
        <div
          className="flex flex-wrap gap-1 border-b border-slate-200 pb-1 dark:border-neutral-600"
          role="tablist"
          aria-label="Submission version"
        >
          {submissionVersions.map((v) => {
            const active = (viewVersionNumber ?? submissionVersions[submissionVersions.length - 1]?.versionNumber) === v.versionNumber
            return (
              <button
                key={v.versionNumber}
                type="button"
                role="tab"
                aria-selected={active}
                className={`rounded-t-md px-2.5 py-1.5 text-xs font-semibold ${
                  active
                    ? 'bg-slate-100 text-slate-900 dark:bg-neutral-800 dark:text-neutral-50'
                    : 'text-slate-600 hover:bg-slate-50 dark:text-neutral-400 dark:hover:bg-neutral-800/60'
                }`}
                onClick={() => setViewVersionNumber(v.versionNumber)}
                tabIndex={0}
              >
                Version {v.versionNumber}
              </button>
            )
          })}
        </div>
      ) : null}

      {originalityViewerOpen ? (
        <OriginalityReportViewer
          open={originalityViewerOpen}
          onClose={() => {
            setOriginalityViewerOpen(false)
            setOriginalityEmbedUrl(null)
            setOriginalitySummary(null)
            setOriginalityViewSummaryOnly(false)
          }}
          embedUrl={originalityEmbedUrl ?? ''}
          storedSummary={originalitySummary}
          viewStoredSummaryOnly={originalityViewSummaryOnly}
        />
      ) : null}

      {mode === 'staff' && blindGradingActive ? (
        <p
          role="status"
          className="rounded-lg border border-indigo-200 bg-indigo-50/90 px-3 py-2 text-sm text-indigo-950 dark:border-indigo-900/60 dark:bg-indigo-950/40 dark:text-indigo-100"
        >
          Blind grading is active — student identities are hidden. Use anonymised labels until you
          reveal identities.
        </p>
      ) : null}

      {isListedGrader && current ? (
        <div className="rounded-lg border border-slate-200 bg-slate-50/90 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900/60">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
            Provisional score
          </p>
          <div className="mt-2 flex flex-wrap items-end gap-2">
            <label className="text-sm text-slate-700 dark:text-neutral-200" htmlFor="prov-score">
              Points (0–{assignmentPointsWorth ?? '—'})
            </label>
            <input
              id="prov-score"
              type="number"
              min={0}
              max={assignmentPointsWorth ?? undefined}
              value={provisionalInput}
              onChange={(e) => setProvisionalInput(e.target.value)}
              className="w-28 rounded-md border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            />
            <button
              type="button"
              disabled={provisionalBusy}
              onClick={() => {
                const submissionId = current?.id
                if (!submissionId) return
                const n = Number(provisionalInput)
                if (!Number.isFinite(n) || n < 0) return
                setProvisionalBusy(true)
                void (async () => {
                  try {
                    await postProvisionalGrade(courseCode, itemId, submissionId, { score: n })
                  } finally {
                    setProvisionalBusy(false)
                  }
                })()
              }}
              className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
            >
              {provisionalBusy ? 'Saving…' : 'Save provisional'}
            </button>
          </div>
        </div>
      ) : null}

      {both ? (
        <div
          className="flex flex-wrap gap-1 border-b border-slate-200 pb-1 dark:border-neutral-600"
          role="tablist"
        >
          <button
            type="button"
            role="tab"
            aria-selected={panel === 'document'}
            className={`rounded-t-md px-3 py-1.5 text-sm font-medium ${
              panel === 'document'
                ? 'bg-slate-100 text-slate-900 dark:bg-neutral-800 dark:text-neutral-50'
                : 'text-slate-600 hover:bg-slate-50 dark:text-neutral-400 dark:hover:bg-neutral-800/60'
            }`}
            onClick={() => setPanel('document')}
          >
            Annotations
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={panel === 'media'}
            className={`rounded-t-md px-3 py-1.5 text-sm font-medium ${
              panel === 'media'
                ? 'bg-slate-100 text-slate-900 dark:bg-neutral-800 dark:text-neutral-50'
                : 'text-slate-600 hover:bg-slate-50 dark:text-neutral-400 dark:hover:bg-neutral-800/60'
            }`}
            onClick={() => setPanel('media')}
          >
            Media feedback
          </button>
        </div>
      ) : null}

      {loadError ? (
        <p className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200">
          {loadError}
        </p>
      ) : null}

      {resubmissionWorkflowEnabled && mode === 'student' && mine?.resubmissionRequested ? (
        <div
          className="rounded-lg border border-amber-200 bg-amber-50/95 px-3 py-2 text-sm text-amber-950 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-100"
          role="status"
        >
          <p className="font-semibold">Revision requested for “{assignmentTitle}”</p>
          {mine.revisionFeedback ? (
            <p className="mt-1 text-amber-950/90 dark:text-amber-100/90">{mine.revisionFeedback}</p>
          ) : null}
          {mine.revisionDueAt ? (
            <p className="mt-2" role="timer" aria-live="off">
              <span className="text-xs font-medium uppercase text-amber-900/80 dark:text-amber-200/80">
                Resubmit by:{' '}
              </span>
              {formatDateTime(mine.revisionDueAt, {
                dateStyle: 'medium',
                timeStyle: 'short',
              })}
              {Number.isFinite(new Date(mine.revisionDueAt).getTime() - deadlineNow) ? (
                <span className="ms-2 text-xs text-amber-900/70 dark:text-amber-200/80">
                  (
                  {new Date(mine.revisionDueAt).getTime() > deadlineNow
                    ? `${Math.max(0, Math.floor((new Date(mine.revisionDueAt).getTime() - deadlineNow) / 60000))} min left`
                    : 'deadline passed'}
                  )
                </span>
              ) : null}
            </p>
          ) : null}
        </div>
      ) : null}

      {showDocPanel && mode === 'student' && submissionAllowsFile ? (
        <div className="flex flex-wrap items-center gap-3">
          <label
            className={`text-sm font-medium ${mine?.resubmissionRequested || !mine?.attachmentFileId ? 'text-slate-700 dark:text-neutral-200' : 'text-slate-400 dark:text-neutral-500'}`}
          >
            <span className="me-2">
              {mine?.resubmissionRequested
                ? 'Resubmit file'
                : resubmissionWorkflowEnabled && mine?.attachmentFileId
                  ? 'Replace file (locked — revision not requested)'
                  : 'Upload file'}
            </span>
            <input
              type="file"
              accept=".pdf,image/png,image/jpeg,image/webp"
              disabled={
                busy ||
                (Boolean(resubmissionWorkflowEnabled) &&
                  Boolean(mine?.attachmentFileId) &&
                  !mine?.resubmissionRequested)
              }
              className="text-sm"
              onChange={(e) => void onUploadStudentFile(e.target.files?.[0] ?? null)}
            />
          </label>
        </div>
      ) : null}


      {showDocPanel && panel === 'document' ? (
        staffGradingSidebarActive && gradingSidebar ? (
          <div className="min-h-[min(70vh,720px)]">
            <ResizableSplitPane
              storageKey="lextures:submission-grade-sidebar-width-inline"
              primary={annotationDocArea}
              secondary={gradingSidebar}
            />
          </div>
        ) : (
          <>
            {annotationDocArea}
            {studentFeedbackSidebar ? (
              <div className="min-h-[min(40vh,480px)] overflow-hidden rounded-xl border border-slate-200 dark:border-neutral-600">
                {studentFeedbackSidebar}
              </div>
            ) : null}
          </>
        )
      ) : null}

      {showMediaPanel && (both ? panel === 'media' : true) && current?.id ? (
        <div className="space-y-4" aria-label="Instructor media feedback">
          {mode === 'staff' && current.id ? (
            <FeedbackMediaRecorder
              courseCode={courseCode}
              itemId={itemId}
              submissionId={current.id}
              onComplete={() => void reloadMedia()}
            />
          ) : null}
          <FeedbackMediaPlayerList
            courseCode={courseCode}
            itemId={itemId}
            submissionId={current.id}
            items={mediaItems}
            readOnly={readOnly}
            onChanged={() => void reloadMedia()}
          />
        </div>
      ) : null}
    </section>
  )
}

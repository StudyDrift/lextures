import { useEffect, useState } from 'react'
import { AssignmentAnnotationWorkbench } from '../../../components/annotation/assignment-annotation-workbench'
import {
  fetchCourse,
  fetchModuleAssignment,
  viewerIsCourseStaffEnrollment,
  type ModuleContentPagePayload,
} from '../../../lib/courses-api'

function submissionTypesAreSet(text: boolean, file: boolean, url: boolean): boolean {
  return text || file || url
}

export type GradebookSubmissionGradingModalState = {
  itemId: string
  studentUserId: string
  columnTitle: string
} | null

type GradebookSubmissionGradingModalProps = {
  open: GradebookSubmissionGradingModalState
  courseCode: string
  onClose: () => void
}

export function GradebookSubmissionGradingModal({
  open,
  courseCode,
  onClose,
}: GradebookSubmissionGradingModalProps) {
  const [loading, setLoading] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [assignment, setAssignment] = useState<ModuleContentPagePayload | null>(null)
  const [annotationsEnabled, setAnnotationsEnabled] = useState(false)
  const [feedbackMediaEnabled, setFeedbackMediaEnabled] = useState(false)
  const [resubmissionWorkflowEnabled, setResubmissionWorkflowEnabled] = useState(false)
  const [viewerIsCourseStaff, setViewerIsCourseStaff] = useState(false)

  useEffect(() => {
    if (!open) {
      setAssignment(null)
      setLoadError(null)
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setLoadError(null)
    void (async () => {
      try {
        const [assignmentRow, courseRow] = await Promise.all([
          fetchModuleAssignment(courseCode, open.itemId),
          fetchCourse(courseCode),
        ])
        if (cancelled) return
        setAssignment(assignmentRow)
        setAnnotationsEnabled(Boolean(courseRow.annotationsEnabled))
        setFeedbackMediaEnabled(Boolean(courseRow.feedbackMediaEnabled))
        setResubmissionWorkflowEnabled(Boolean(courseRow.resubmissionWorkflowEnabled))
        setViewerIsCourseStaff(viewerIsCourseStaffEnrollment(courseRow.viewerEnrollmentRoles))
      } catch (e: unknown) {
        if (cancelled) return
        setAssignment(null)
        setLoadError(e instanceof Error ? e.message : 'Could not load this assignment.')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode, open])

  if (!open) return null

  const assignmentAcceptsSubmissions = assignment
    ? submissionTypesAreSet(
        assignment.submissionAllowText,
        assignment.submissionAllowFileUpload,
        assignment.submissionAllowUrl,
      )
    : false
  const showWorkbench = Boolean(
    assignment && (assignmentAcceptsSubmissions || feedbackMediaEnabled),
  )
  const annotationsActive = Boolean(annotationsEnabled && assignment?.submissionAllowFileUpload)
  const blindGradingActive = Boolean(
    assignment?.blindGrading &&
      !assignment.identitiesRevealedAt &&
      viewerIsCourseStaff,
  )

  return (
    <>
      {loading ? (
        <div
          className="fixed inset-0 z-[500] flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-[2px] dark:bg-black/80"
          role="status"
          aria-live="polite"
        >
          <p className="rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-700 shadow-lg dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200">
            Loading assignment…
          </p>
        </div>
      ) : null}
      {loadError ? (
        <div
          className="fixed inset-0 z-[500] flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-[2px] dark:bg-black/80"
          role="dialog"
          aria-modal="true"
          aria-labelledby="gradebook-submission-error-title"
        >
          <div className="w-full max-w-md rounded-xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-600 dark:bg-neutral-900">
            <h2
              id="gradebook-submission-error-title"
              className="text-lg font-semibold text-slate-950 dark:text-neutral-100"
            >
              Could not open grading
            </h2>
            <p className="mt-2 text-sm text-red-600 dark:text-red-400" role="alert">
              {loadError}
            </p>
            <div className="mt-4 flex justify-end">
              <button
                type="button"
                className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-800 hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:bg-neutral-700/80"
                onClick={onClose}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      ) : null}
      {showWorkbench && assignment ? (
        <AssignmentAnnotationWorkbench
          key={`${open.itemId}:${open.studentUserId}`}
          courseCode={courseCode}
          itemId={open.itemId}
          assignmentTitle={assignment.title || open.columnTitle}
          mode="staff"
          submissionAllowsFile={assignment.submissionAllowFileUpload}
          submissionAllowsText={assignment.submissionAllowText}
          submissionAllowsUrl={assignment.submissionAllowUrl}
          annotationsActive={annotationsActive}
          feedbackMediaEnabled={feedbackMediaEnabled}
          resubmissionWorkflowEnabled={resubmissionWorkflowEnabled}
          blindGradingActive={blindGradingActive}
          canRevealIdentities={assignment.viewerCanRevealIdentities}
          onAfterRevealIdentities={() => {
            void fetchModuleAssignment(courseCode, open.itemId).then(setAssignment).catch(() => {})
          }}
          moderatedGradingActive={Boolean(assignment.moderatedGrading && viewerIsCourseStaff)}
          assignmentPointsWorth={assignment.pointsWorth}
          assignmentRubric={assignment.rubric}
          provisionalGraderUserIds={assignment.provisionalGraderUserIds ?? []}
          originalityDetection={assignment.originalityDetection}
          presentation="modal"
          modalOpen
          onModalClose={onClose}
          initialStudentUserId={open.studentUserId}
        />
      ) : null}
      {!loading && !loadError && assignment && !showWorkbench ? (
        <div
          className="fixed inset-0 z-[500] flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-[2px] dark:bg-black/80"
          role="dialog"
          aria-modal="true"
          aria-labelledby="gradebook-submission-empty-title"
        >
          <div className="w-full max-w-md rounded-xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-600 dark:bg-neutral-900">
            <h2
              id="gradebook-submission-empty-title"
              className="text-lg font-semibold text-slate-950 dark:text-neutral-100"
            >
              {assignment.title || open.columnTitle}
            </h2>
            <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
              This assignment does not accept submissions, so there is nothing to grade here.
            </p>
            <div className="mt-4 flex justify-end">
              <button
                type="button"
                className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-800 hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:bg-neutral-700/80"
                onClick={onClose}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  )
}

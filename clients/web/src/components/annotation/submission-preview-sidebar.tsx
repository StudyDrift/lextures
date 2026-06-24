import { useState } from 'react'
import { Files } from 'lucide-react'
import type { RubricDefinition, SubmissionAttachmentApi } from '../../lib/courses-api'
import { SubmissionFileDetailsPanel } from './submission-file-details-panel'
import { SubmissionGradingPanel } from './submission-grading-panel'

type SidebarTab = 'grade' | 'files'

type SubmissionPreviewSidebarProps = {
  mode: 'staff' | 'student'
  courseCode: string
  itemId: string
  submissionId: string | null
  studentUserId?: string | null
  rubric: RubricDefinition | null
  maxPoints: number | null
  gradingDisabled?: boolean
  gradeRefreshKey?: number
  files: SubmissionAttachmentApi[]
  selectedFileId: string | null
  onSelectFile: (fileId: string) => void
  submittedAt?: string | null
  blindLabel?: string | null
  onDownloadAll?: () => void
  downloadAllBusy?: boolean
  onGradeSaved?: () => void
  onGradeCleared?: () => void
}

export function SubmissionPreviewSidebar({
  mode,
  courseCode,
  itemId,
  submissionId,
  studentUserId = null,
  rubric,
  maxPoints,
  gradingDisabled = false,
  gradeRefreshKey = 0,
  files,
  selectedFileId,
  onSelectFile,
  submittedAt,
  blindLabel,
  onDownloadAll,
  downloadAllBusy = false,
  onGradeSaved,
  onGradeCleared,
}: SubmissionPreviewSidebarProps) {
  const [tab, setTab] = useState<SidebarTab>('grade')
  const hasRubric = Boolean(rubric && rubric.criteria.length > 0)

  if (mode !== 'staff') {
    return (
      <aside
        className="flex h-full min-h-0 w-full flex-col overflow-y-auto bg-slate-100 dark:bg-neutral-800"
        aria-label="Submission feedback"
      >
        {submissionId ? (
          <SubmissionGradingPanel
            mode="student"
            courseCode={courseCode}
            itemId={itemId}
            submissionId={submissionId}
            rubric={hasRubric ? rubric : null}
            maxPoints={maxPoints}
            disabled
          />
        ) : (
          <SubmissionFileDetailsPanel
            files={files}
            selectedFileId={selectedFileId}
            onSelectFile={onSelectFile}
            submittedAt={submittedAt}
            blindLabel={blindLabel}
            onDownloadAll={onDownloadAll}
            downloadAllBusy={downloadAllBusy}
          />
        )}
      </aside>
    )
  }

  return (
    <aside
      className="flex h-full min-h-0 w-full flex-col bg-slate-100 dark:bg-neutral-800"
      aria-label="Submission grading and details"
    >
      <div
        className="flex shrink-0 gap-1 border-b border-slate-200 bg-slate-100 p-2 dark:border-neutral-600 dark:bg-neutral-900/40"
        role="tablist"
        aria-label="Submission sidebar"
      >
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'grade'}
          onClick={() => setTab('grade')}
          className={`flex-1 rounded-lg px-3 py-2 text-sm font-semibold transition-[background-color,color,border-color] ${
            tab === 'grade'
              ? 'bg-white text-indigo-700 shadow-sm dark:bg-neutral-800 dark:text-indigo-300'
              : 'text-slate-600 hover:bg-white/70 dark:text-neutral-400 dark:hover:bg-neutral-800/60'
          }`}
        >
          Grade
          {hasRubric ? (
            <span className="ms-1.5 rounded-full bg-indigo-100 px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-indigo-700 dark:bg-indigo-950/70 dark:text-indigo-300">
              Rubric
            </span>
          ) : null}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'files'}
          onClick={() => setTab('files')}
          className={`flex flex-1 items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-sm font-semibold transition-[background-color,color,border-color] ${
            tab === 'files'
              ? 'bg-white text-indigo-700 shadow-sm dark:bg-neutral-800 dark:text-indigo-300'
              : 'text-slate-600 hover:bg-white/70 dark:text-neutral-400 dark:hover:bg-neutral-800/60'
          }`}
        >
          <Files className="h-4 w-4" aria-hidden="true" />
          Files
        </button>
      </div>

      {tab === 'grade' ? (
        <SubmissionGradingPanel
          key={submissionId ?? studentUserId ?? 'none'}
          mode={mode}
          courseCode={courseCode}
          itemId={itemId}
          submissionId={submissionId}
          studentUserId={studentUserId}
          rubric={rubric}
          maxPoints={maxPoints}
          disabled={gradingDisabled}
          gradeRefreshKey={gradeRefreshKey}
          autoFocusScore
          onGradeSaved={onGradeSaved}
          onGradeCleared={onGradeCleared}
        />
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto">
          <SubmissionFileDetailsPanel
            files={files}
            selectedFileId={selectedFileId}
            onSelectFile={onSelectFile}
            submittedAt={submittedAt}
            blindLabel={blindLabel}
            onDownloadAll={onDownloadAll}
            downloadAllBusy={downloadAllBusy}
          />
        </div>
      )}
    </aside>
  )
}
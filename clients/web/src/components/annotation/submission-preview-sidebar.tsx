import { useState } from 'react'
import { FileText } from 'lucide-react'
import type { RubricDefinition } from '../../lib/courses-api'
import { SubmissionFileDetailsPanel } from './submission-file-details-panel'
import { SubmissionGradingPanel } from './submission-grading-panel'

type SidebarTab = 'grade' | 'file'

type SubmissionPreviewSidebarProps = {
  mode: 'staff' | 'student'
  courseCode: string
  itemId: string
  submissionId: string | null
  rubric: RubricDefinition | null
  maxPoints: number | null
  gradingDisabled?: boolean
  filename: string
  filePath: string | null
  submittedAt?: string | null
  blindLabel?: string | null
  mimeType?: string | null
}

export function SubmissionPreviewSidebar({
  mode,
  courseCode,
  itemId,
  submissionId,
  rubric,
  maxPoints,
  gradingDisabled = false,
  filename,
  filePath,
  submittedAt,
  blindLabel,
  mimeType,
}: SubmissionPreviewSidebarProps) {
  const [tab, setTab] = useState<SidebarTab>('grade')
  const hasRubric = Boolean(rubric && rubric.criteria.length > 0)

  if (mode !== 'staff') {
    return (
      <aside
        className="flex w-full shrink-0 flex-col overflow-y-auto border-t border-slate-200 bg-slate-100 dark:border-neutral-600 dark:bg-neutral-800 lg:w-96 lg:border-t-0 lg:border-l xl:w-[26rem]"
        aria-label="Submission file details"
      >
        <SubmissionFileDetailsPanel
          filename={filename}
          filePath={filePath}
          submittedAt={submittedAt}
          blindLabel={blindLabel}
          mimeType={mimeType}
        />
      </aside>
    )
  }

  return (
    <aside
      className="flex min-h-0 w-full shrink-0 flex-col border-t border-slate-200 bg-slate-100 dark:border-neutral-600 dark:bg-neutral-800 lg:w-96 lg:border-t-0 lg:border-l xl:w-[28rem]"
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
          className={`flex-1 rounded-lg px-3 py-2 text-sm font-semibold transition ${
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
          aria-selected={tab === 'file'}
          onClick={() => setTab('file')}
          className={`flex flex-1 items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-sm font-semibold transition ${
            tab === 'file'
              ? 'bg-white text-indigo-700 shadow-sm dark:bg-neutral-800 dark:text-indigo-300'
              : 'text-slate-600 hover:bg-white/70 dark:text-neutral-400 dark:hover:bg-neutral-800/60'
          }`}
        >
          <FileText className="h-4 w-4" aria-hidden="true" />
          File
        </button>
      </div>

      {tab === 'grade' ? (
        <SubmissionGradingPanel
          courseCode={courseCode}
          itemId={itemId}
          submissionId={submissionId}
          rubric={rubric}
          maxPoints={maxPoints}
          disabled={gradingDisabled}
        />
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto">
          <SubmissionFileDetailsPanel
            filename={filename}
            filePath={filePath}
            submittedAt={submittedAt}
            blindLabel={blindLabel}
            mimeType={mimeType}
          />
        </div>
      )}
    </aside>
  )
}

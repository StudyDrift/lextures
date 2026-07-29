import { useId } from 'react'
import { ConfirmDialog } from '../../components/confirm-dialog'
import {
  courseExportImportStatLines,
  importModeLabel,
  importModeSummary,
  type CourseExportImportStats,
} from '../../lib/course-export-import-stats'
import type { CourseBundleImportMode } from '../../lib/courses-api'

export type CourseJsonImportConfirmDialogProps = {
  open: boolean
  fileName: string | null
  stats: CourseExportImportStats | null
  importMode: CourseBundleImportMode
  busy: boolean
  onClose: () => void
  onConfirm: () => void
}

export function CourseJsonImportConfirmDialog({
  open,
  fileName,
  stats,
  importMode,
  busy,
  onClose,
  onConfirm,
}: CourseJsonImportConfirmDialogProps) {
  const statsListId = useId()
  const pendingStatLines = stats ? courseExportImportStatLines(stats) : []

  return (
    <ConfirmDialog
      open={open}
      title="Confirm course import"
      variant={importMode === 'mergeAdd' ? 'default' : 'danger'}
      confirmLabel={busy ? 'Importing…' : 'Start import'}
      cancelLabel="Cancel"
      busy={busy}
      onClose={onClose}
      onConfirm={onConfirm}
      description={
        fileName && stats ? (
          <div className="space-y-3">
            <p>
              Import <span className="font-medium text-slate-800 dark:text-neutral-100">{fileName}</span>
              {stats.title ? (
                <>
                  {' '}
                  from <span className="font-medium text-slate-800 dark:text-neutral-100">{stats.title}</span>
                </>
              ) : null}
              {stats.sourceCourseCode ? (
                <>
                  {' '}
                  (
                  <code className="text-xs">{stats.sourceCourseCode}</code>
                  )
                </>
              ) : null}{' '}
              into this course using{' '}
              <span className="font-medium text-slate-800 dark:text-neutral-100">
                {importModeLabel(importMode)}
              </span>
              .
            </p>
            <p className="text-slate-500 dark:text-neutral-400">{importModeSummary(importMode)}</p>
            {pendingStatLines.length > 0 ? (
              <div>
                <p id={statsListId} className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                  In this file
                </p>
                <ul
                  aria-labelledby={statsListId}
                  className="mt-2 grid grid-cols-2 gap-2"
                >
                  {pendingStatLines.map((line) => (
                    <li
                      key={line.key}
                      className="flex items-baseline justify-between gap-2 rounded-lg border border-slate-200 bg-slate-50 px-2.5 py-1.5 dark:border-neutral-600 dark:bg-neutral-800/60"
                    >
                      <span className="text-xs text-slate-600 dark:text-neutral-300">{line.label}</span>
                      <span className="text-sm font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
                        {line.count}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            ) : (
              <p className="text-slate-500 dark:text-neutral-400">
                This file has no modules, bodies, syllabus sections, grading groups, or enrollments
                counted for preview. You can still import if the server accepts the format.
              </p>
            )}
            {stats.hasCourseSettings ? (
              <p className="text-xs text-slate-500 dark:text-neutral-500">
                Course settings (title, schedule, feature flags, appearance) are included and will
                {importMode === 'mergeAdd' ? ' not be applied in merge mode.' : ' be applied from the file.'}
              </p>
            ) : null}
          </div>
        ) : null
      }
    />
  )
}

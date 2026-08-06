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
              Import <span className="font-medium text-fg-default">{fileName}</span>
              {stats.title ? (
                <>
                  {' '}
                  from <span className="font-medium text-fg-default">{stats.title}</span>
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
              <span className="font-medium text-fg-default">
                {importModeLabel(importMode)}
              </span>
              .
            </p>
            <p className="text-fg-muted">{importModeSummary(importMode)}</p>
            {pendingStatLines.length > 0 ? (
              <div>
                <p id={statsListId} className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
                  In this file
                </p>
                <ul
                  aria-labelledby={statsListId}
                  className="mt-2 grid grid-cols-2 gap-2"
                >
                  {pendingStatLines.map((line) => (
                    <li
                      key={line.key}
                      className="flex items-baseline justify-between gap-2 rounded-lg border border-border-default bg-surface-base px-2.5 py-1.5 dark:border-border-default/60"
                    >
                      <span className="text-xs text-fg-muted">{line.label}</span>
                      <span className="text-sm font-semibold tabular-nums text-fg-default">
                        {line.count}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            ) : (
              <p className="text-fg-muted">
                This file has no modules, bodies, syllabus sections, grading groups, or enrollments
                counted for preview. You can still import if the server accepts the format.
              </p>
            )}
            {stats.hasCourseSettings ? (
              <p className="text-xs text-fg-subtle">
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

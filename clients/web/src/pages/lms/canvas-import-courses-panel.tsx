import { useId } from 'react'
import { Search, X } from 'lucide-react'
import { CanvasImportProgressLog } from '../../components/canvas/canvas-import-progress-log'
import { CanvasReadOnlyNotice } from '../../components/canvas/canvas-read-only-notice'
import type { CanvasImportProgressEntry } from '../../hooks/use-canvas-import-progress-log'
import type { CanvasCourseListItem } from '../../lib/courses-api'

export type CanvasImportCoursesStep = 'credentials' | 'select' | 'importing'

export type CanvasImportCoursesPanelProps = {
  step: CanvasImportCoursesStep
  busy: boolean
  importComplete: boolean
  error: string | null
  canvasBaseUrl: string
  canvasToken: string
  rememberCredentials: boolean
  courses: CanvasCourseListItem[] | null
  selected: Set<number>
  nameFilter: string
  hideUnpublished: boolean
  enableCanvasGradeSync: boolean
  importLog: CanvasImportProgressEntry[]
  filteredCourses: CanvasCourseListItem[]
  coursesToImport: CanvasCourseListItem[]
  selectedVisibleCount: number
  selectedHiddenCount: number
  onCanvasBaseUrlChange: (value: string) => void
  onCanvasTokenChange: (value: string) => void
  onRememberCredentialsChange: (value: boolean) => void
  onNameFilterChange: (value: string) => void
  onHideUnpublishedChange: (value: boolean) => void
  onEnableCanvasGradeSyncChange: (value: boolean) => void
  onToggleCourse: (id: number) => void
  onSelectAllVisible: () => void
  onClearSelection: () => void
  onConnect: () => void
  onBackToCredentials: () => void
  onImport: () => void
  onCancelImport: () => void
  onDismiss?: () => void
  showClose?: boolean
  compact?: boolean
  className?: string
}

function isCanvasCourseUnpublished(workflowState: string | undefined): boolean {
  return workflowState?.trim().toLowerCase() === 'unpublished'
}

export function CanvasImportCoursesPanel({
  step,
  busy,
  importComplete,
  error,
  canvasBaseUrl,
  canvasToken,
  rememberCredentials,
  courses,
  selected,
  nameFilter,
  hideUnpublished,
  enableCanvasGradeSync,
  importLog,
  filteredCourses,
  coursesToImport,
  selectedHiddenCount,
  onCanvasBaseUrlChange,
  onCanvasTokenChange,
  onRememberCredentialsChange,
  onNameFilterChange,
  onHideUnpublishedChange,
  onEnableCanvasGradeSyncChange,
  onToggleCourse,
  onSelectAllVisible,
  onClearSelection,
  onConnect,
  onBackToCredentials,
  onImport,
  onCancelImport,
  onDismiss,
  showClose = true,
  compact = false,
  className,
}: CanvasImportCoursesPanelProps) {
  const titleId = useId()
  const canDismiss = !busy || importComplete

  return (
    <div
      className={[
        'flex flex-col overflow-hidden',
        compact ? '' : 'rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900',
        step === 'importing' && !compact ? 'min-h-[32rem] max-h-[min(92vh,880px)]' : '',
        !compact ? 'max-h-[min(90vh,720px)]' : '',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <div className="flex items-start justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
        <div>
          <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
            Import from Canvas
          </h2>
          <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">
            {importComplete
              ? 'Import finished. Review the log below.'
              : step === 'credentials'
                ? 'Connect with your Canvas URL and personal access token.'
                : step === 'select'
                  ? 'Choose which courses to create in Lextures.'
                  : 'Creating courses and pulling content from Canvas…'}
          </p>
        </div>
        {showClose && onDismiss ? (
          <button
            type="button"
            disabled={!canDismiss}
            onClick={onDismiss}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
            aria-label="Close"
          >
            <X className="h-5 w-5" aria-hidden />
          </button>
        ) : null}
      </div>

      <div
        className={[
          'min-h-0 flex-1 px-5 py-4',
          step === 'importing' ? 'overflow-visible' : 'overflow-y-auto',
        ].join(' ')}
      >
        {error ? (
          <p className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
            {error}
          </p>
        ) : null}

        {step === 'credentials' ? (
          <div className="space-y-4">
            <CanvasReadOnlyNotice />
            <p className="text-sm text-slate-600 dark:text-neutral-400">
              Create a token in Canvas under Account → Settings → New Access Token (read-only
              scopes are sufficient). The token is sent to our server only for this session and is
              not stored on the server.
            </p>
            <label className="block">
              <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">
                Canvas base URL
              </span>
              <input
                type="url"
                value={canvasBaseUrl}
                onChange={(e) => onCanvasBaseUrlChange(e.target.value)}
                placeholder="https://yourschool.instructure.com"
                autoComplete="off"
                disabled={busy}
                className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-inner outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </label>
            <label className="block">
              <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">
                Access token
              </span>
              <input
                type="password"
                value={canvasToken}
                onChange={(e) => onCanvasTokenChange(e.target.value)}
                placeholder="Canvas API token"
                autoComplete="off"
                disabled={busy}
                className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-inner outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </label>
            <label className="flex cursor-pointer items-start gap-2 rounded-xl border border-slate-200 p-3 dark:border-neutral-600">
              <input
                type="checkbox"
                className="mt-0.5"
                checked={rememberCredentials}
                disabled={busy}
                onChange={(e) => onRememberCredentialsChange(e.target.checked)}
              />
              <span className="text-sm text-slate-700 dark:text-neutral-300">
                Remember URL and token in this browser
              </span>
            </label>
          </div>
        ) : null}

        {step === 'select' && courses ? (
          <div className="space-y-3">
            <CanvasReadOnlyNotice />
            <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50/80 p-3 dark:border-neutral-600 dark:bg-neutral-800/40">
              <label className="block">
                <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">
                  Search by name
                </span>
                <span className="relative mt-1 flex">
                  <Search
                    className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 dark:text-neutral-500"
                    aria-hidden
                  />
                  <input
                    type="search"
                    value={nameFilter}
                    onChange={(e) => onNameFilterChange(e.target.value)}
                    placeholder="Filter courses…"
                    autoComplete="off"
                    className="w-full rounded-xl border border-slate-200 bg-white py-2 ps-9 pe-3 text-sm text-slate-900 shadow-inner outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                  />
                </span>
              </label>
              <label className="flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  checked={hideUnpublished}
                  onChange={(e) => onHideUnpublishedChange(e.target.checked)}
                />
                <span className="text-sm text-slate-700 dark:text-neutral-300">
                  Hide unpublished courses
                </span>
              </label>
              <label className="flex cursor-pointer items-start gap-2 rounded-xl border border-sky-200 bg-sky-50/80 p-3 dark:border-sky-900/50 dark:bg-sky-950/30">
                <input
                  type="checkbox"
                  className="mt-0.5"
                  checked={enableCanvasGradeSync}
                  onChange={(e) => onEnableCanvasGradeSyncChange(e.target.checked)}
                />
                <span>
                  <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                    Sync grades back to Canvas when grading
                  </span>
                  <span className="mt-0.5 block text-xs text-slate-600 dark:text-neutral-400">
                    When enabled, saving a grade in Lextures automatically pushes it to Canvas.
                    Requires a Canvas token with grade-update permission (saved in this browser if
                    you chose Remember).
                  </span>
                </span>
              </label>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-sm text-slate-600 dark:text-neutral-400">
                {filteredCourses.length === courses.length
                  ? `${courses.length} course${courses.length === 1 ? '' : 's'}`
                  : `${filteredCourses.length} of ${courses.length} courses`}
                {coursesToImport.length > 0
                  ? ` · ${coursesToImport.length} selected for import`
                  : ''}
              </p>
              <div className="flex gap-2 text-sm">
                <button
                  type="button"
                  disabled={filteredCourses.length === 0}
                  className="font-medium text-indigo-600 hover:text-indigo-500 disabled:opacity-50 dark:text-indigo-400"
                  onClick={onSelectAllVisible}
                >
                  Select all shown
                </button>
                <span className="text-slate-300 dark:text-neutral-600" aria-hidden>
                  |
                </span>
                <button
                  type="button"
                  disabled={coursesToImport.length === 0}
                  className="font-medium text-indigo-600 hover:text-indigo-500 disabled:opacity-50 dark:text-indigo-400"
                  onClick={onClearSelection}
                >
                  Clear all
                </button>
              </div>
            </div>
            {selectedHiddenCount > 0 ? (
              <p className="text-xs text-amber-800 dark:text-amber-200">
                {selectedHiddenCount} selected course{selectedHiddenCount === 1 ? '' : 's'}{' '}
                {selectedHiddenCount === 1 ? 'is' : 'are'} hidden by your filters and will still be
                imported unless you clear selection.
              </p>
            ) : null}
            <ul className="max-h-64 space-y-1 overflow-y-auto rounded-xl border border-slate-200 p-2 dark:border-neutral-600">
              {filteredCourses.length === 0 ? (
                <li className="px-2 py-6 text-center text-sm text-slate-500 dark:text-neutral-400">
                  No courses match your filters.
                </li>
              ) : (
                filteredCourses.map((c) => (
                  <li key={c.id}>
                    <label className="flex cursor-pointer items-start gap-3 rounded-lg px-2 py-2 hover:bg-slate-50 dark:hover:bg-neutral-800/80">
                      <input
                        type="checkbox"
                        className="mt-1"
                        checked={selected.has(c.id)}
                        onChange={() => onToggleCourse(c.id)}
                      />
                      <span className="min-w-0 flex-1">
                        <span className="flex flex-wrap items-center gap-2">
                          <span className="text-sm font-medium text-slate-900 dark:text-neutral-100">
                            {c.name}
                          </span>
                          {isCanvasCourseUnpublished(c.workflowState) ? (
                            <span className="rounded-md bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-900 dark:bg-amber-950/60 dark:text-amber-200">
                              Unpublished
                            </span>
                          ) : null}
                        </span>
                        <span className="mt-0.5 block text-xs text-slate-500 dark:text-neutral-500">
                          ID {c.id}
                          {c.courseCode ? ` · ${c.courseCode}` : ''}
                          {c.termName ? ` · ${c.termName}` : ''}
                          {c.workflowState && !isCanvasCourseUnpublished(c.workflowState)
                            ? ` · ${c.workflowState}`
                            : ''}
                        </span>
                      </span>
                    </label>
                  </li>
                ))
              )}
            </ul>
            <p className="text-xs text-slate-500 dark:text-neutral-500">
              Check the courses you want, then import. Each one is created in Lextures and filled from
              Canvas (modules, assignments, quizzes, roster enrollments, grades, and settings).
              Learners with an email in Canvas get Lextures accounts when needed. Canvas is not
              modified.
            </p>
          </div>
        ) : null}

        {step === 'importing' ? (
          <div className="flex min-h-[280px] flex-col gap-4 py-2">
            <CanvasImportProgressLog
              entries={importLog}
              active={busy && !importComplete}
              maxHeightClassName="max-h-72"
              className="min-h-0 w-full"
            />
          </div>
        ) : null}
      </div>

      <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-4 dark:border-neutral-700">
        {step === 'credentials' ? (
          <>
            {onDismiss ? (
              <button
                type="button"
                disabled={busy}
                onClick={onDismiss}
                className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
              >
                Cancel
              </button>
            ) : null}
            <button
              type="button"
              disabled={busy}
              onClick={onConnect}
              className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-60"
            >
              {busy ? 'Connecting…' : 'Load courses'}
            </button>
          </>
        ) : null}
        {step === 'select' ? (
          <>
            <button
              type="button"
              disabled={busy}
              onClick={onBackToCredentials}
              className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              Back
            </button>
            <button
              type="button"
              disabled={busy || coursesToImport.length === 0}
              onClick={onImport}
              className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-60"
            >
              Import {coursesToImport.length} course{coursesToImport.length === 1 ? '' : 's'}
            </button>
          </>
        ) : null}
        {step === 'importing' && busy && !importComplete ? (
          <button
            type="button"
            onClick={onCancelImport}
            className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            Cancel import
          </button>
        ) : null}
        {importComplete && onDismiss ? (
          <button
            type="button"
            onClick={onDismiss}
            className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
          >
            Done
          </button>
        ) : null}
      </div>
    </div>
  )
}
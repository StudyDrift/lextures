import { useId, useLayoutEffect, useRef, type RefObject } from 'react'
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

const pressableButtonClassName =
  'motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96]'

const PANEL_RESIZE_MS = 300
const PANEL_RESIZE_EASING = 'cubic-bezier(0.2, 0, 0, 1)'

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false
  return (
    document.documentElement.classList.contains('reduced-motion') ||
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

function useAnimatedPanelResize(
  panelRef: RefObject<HTMLDivElement | null>,
  step: CanvasImportCoursesStep,
  enabled: boolean,
) {
  const prevStepRef = useRef(step)

  useLayoutEffect(() => {
    if (!enabled) return
    const node = panelRef.current
    if (!node) return
    if (prevStepRef.current === step) return
    prevStepRef.current = step
    if (prefersReducedMotion()) return

    const startHeight = node.getBoundingClientRect().height
    node.style.height = `${startHeight}px`
    node.style.overflow = 'hidden'

    const endHeight = node.scrollHeight
    node.style.transition = `height ${PANEL_RESIZE_MS}ms ${PANEL_RESIZE_EASING}`

    const raf = requestAnimationFrame(() => {
      node.style.height = `${endHeight}px`
    })

    const finish = () => {
      node.style.height = ''
      node.style.overflow = ''
      node.style.transition = ''
    }

    node.addEventListener('transitionend', finish, { once: true })
    const timeout = window.setTimeout(finish, PANEL_RESIZE_MS + 50)

    return () => {
      cancelAnimationFrame(raf)
      clearTimeout(timeout)
      node.removeEventListener('transitionend', finish)
    }
  }, [enabled, panelRef, step])
}

function isCanvasCourseUnpublished(workflowState: string | undefined): boolean {
  return workflowState?.trim().toLowerCase() === 'unpublished'
}

function courseCountLabel(filtered: number, total: number): string {
  if (filtered === total) {
    return `${total} course${total === 1 ? '' : 's'}`
  }
  return `${filtered} of ${total} courses`
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
  const panelRef = useRef<HTMLDivElement>(null)
  const canDismiss = !busy || importComplete
  const isExpandedStep = step !== 'credentials'

  useAnimatedPanelResize(panelRef, step, !compact)

  return (
    <div
      ref={panelRef}
      className={[
        'flex flex-col overflow-hidden',
        compact ? '' : 'rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900',
        !compact && isExpandedStep ? 'max-h-[min(92vh,880px)]' : '',
        step === 'importing' && !compact ? 'min-h-[32rem]' : '',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <div className="flex shrink-0 items-start justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
        <div className="min-w-0">
          <h2
            id={titleId}
            className="text-balance text-lg font-semibold text-slate-900 dark:text-neutral-100"
          >
            Import from Canvas
          </h2>
          <p className="mt-0.5 text-pretty text-sm text-slate-500 dark:text-neutral-400">
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
            className={[
              'relative -me-1 flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800',
              pressableButtonClassName,
            ].join(' ')}
            aria-label="Close"
          >
            <X className="h-5 w-5" aria-hidden />
          </button>
        ) : null}
      </div>

      <div
        className={[
          'min-h-0 flex-1 px-5 py-4',
          step === 'select' && courses ? 'flex flex-col overflow-hidden' : '',
          step === 'importing' ? 'overflow-visible' : step === 'select' && courses ? '' : 'overflow-y-auto',
        ].join(' ')}
      >
        {error ? (
          <p className="mb-4 shrink-0 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
            {error}
          </p>
        ) : null}

        {step === 'credentials' ? (
          <div className="space-y-3">
            <CanvasReadOnlyNotice variant="compact" />
            <p className="text-pretty text-sm text-slate-600 dark:text-neutral-400">
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
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-inner outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
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
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-inner outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </label>
            <label className="flex cursor-pointer items-start gap-2 rounded-lg p-3 shadow-[inset_0_0_0_1px_rgba(15,23,42,0.08)] dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)]">
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
          <div className="canvas-import-select-in flex min-h-[min(68vh,620px)] flex-1 flex-col gap-3">
            <div className="shrink-0 space-y-2">
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <span className="relative flex min-w-0 flex-1">
                  <Search
                    className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 dark:text-neutral-500"
                    aria-hidden
                  />
                  <input
                    type="search"
                    value={nameFilter}
                    onChange={(e) => onNameFilterChange(e.target.value)}
                    placeholder="Filter courses…"
                    aria-label="Filter courses by name"
                    autoComplete="off"
                    className="w-full rounded-lg border border-slate-200 bg-white py-2 ps-9 pe-3 text-sm text-slate-900 shadow-inner outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                  />
                </span>
                <label className="flex min-h-10 shrink-0 cursor-pointer items-center gap-2 px-1 sm:px-2">
                  <input
                    type="checkbox"
                    checked={hideUnpublished}
                    onChange={(e) => onHideUnpublishedChange(e.target.checked)}
                  />
                  <span className="text-sm whitespace-nowrap text-slate-700 dark:text-neutral-300">
                    Hide unpublished
                  </span>
                </label>
              </div>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="text-sm text-slate-600 dark:text-neutral-400">
                  <span className="tabular-nums">
                    {courseCountLabel(filteredCourses.length, courses.length)}
                  </span>
                  {coursesToImport.length > 0 ? (
                    <>
                      {' '}
                      ·{' '}
                      <span className="tabular-nums">
                        {coursesToImport.length} selected
                      </span>
                    </>
                  ) : null}
                </p>
                <div className="flex gap-1 text-sm">
                  <button
                    type="button"
                    disabled={filteredCourses.length === 0}
                    className={[
                      'min-h-10 rounded-lg px-2 font-medium text-indigo-600 hover:bg-indigo-50 hover:text-indigo-500 disabled:opacity-50 dark:text-indigo-400 dark:hover:bg-indigo-950/40',
                      pressableButtonClassName,
                    ].join(' ')}
                    onClick={onSelectAllVisible}
                  >
                    Select all shown
                  </button>
                  <button
                    type="button"
                    disabled={coursesToImport.length === 0}
                    className={[
                      'min-h-10 rounded-lg px-2 font-medium text-indigo-600 hover:bg-indigo-50 hover:text-indigo-500 disabled:opacity-50 dark:text-indigo-400 dark:hover:bg-indigo-950/40',
                      pressableButtonClassName,
                    ].join(' ')}
                    onClick={onClearSelection}
                  >
                    Clear all
                  </button>
                </div>
              </div>
              {selectedHiddenCount > 0 ? (
                <p className="text-xs text-amber-800 dark:text-amber-200">
                  <span className="tabular-nums">{selectedHiddenCount}</span> selected course
                  {selectedHiddenCount === 1 ? '' : 's'}{' '}
                  {selectedHiddenCount === 1 ? 'is' : 'are'} hidden by your filters and will still
                  be imported unless you clear selection.
                </p>
              ) : null}
            </div>

            <ul className="min-h-0 flex-1 space-y-0.5 overflow-y-auto rounded-lg bg-slate-50/70 p-1.5 shadow-[inset_0_0_0_1px_rgba(15,23,42,0.08)] dark:bg-neutral-800/40 dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)]">
              {filteredCourses.length === 0 ? (
                <li className="px-2 py-10 text-center text-sm text-slate-500 dark:text-neutral-400">
                  No courses match your filters.
                </li>
              ) : (
                filteredCourses.map((c) => (
                  <li key={c.id}>
                    <label className="flex min-h-10 cursor-pointer items-start gap-3 rounded-md px-2 py-2 hover:bg-white/80 dark:hover:bg-neutral-900/70">
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
                            <span className="rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-900 dark:bg-amber-950/60 dark:text-amber-200">
                              Unpublished
                            </span>
                          ) : null}
                        </span>
                        <span className="mt-0.5 block text-xs text-slate-500 dark:text-neutral-500">
                          <span className="tabular-nums">ID {c.id}</span>
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

            <div className="mt-auto grid shrink-0 gap-3 lg:grid-cols-2 lg:gap-4">
              <CanvasReadOnlyNotice variant="compact" />
              <label className="flex h-full cursor-pointer items-start gap-2 rounded-lg bg-slate-50/80 px-3 py-2.5 shadow-[inset_0_0_0_1px_rgba(15,23,42,0.08)] dark:bg-neutral-800/40 dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)]">
                <input
                  type="checkbox"
                  className="mt-0.5"
                  checked={enableCanvasGradeSync}
                  onChange={(e) => onEnableCanvasGradeSyncChange(e.target.checked)}
                />
                <span className="min-w-0">
                  <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                    Sync grades back to Canvas
                  </span>
                  <span className="mt-0.5 block text-pretty text-xs leading-relaxed text-slate-600 dark:text-neutral-400">
                    Push grades to Canvas when you save in Lextures. Requires a token with
                    grade-update permission.
                  </span>
                </span>
              </label>
            </div>
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

      <div className="flex shrink-0 flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-4 dark:border-neutral-700">
        {step === 'credentials' ? (
          <>
            {onDismiss ? (
              <button
                type="button"
                disabled={busy}
                onClick={onDismiss}
                className={[
                  'rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800',
                  pressableButtonClassName,
                ].join(' ')}
              >
                Cancel
              </button>
            ) : null}
            <button
              type="button"
              disabled={busy}
              onClick={onConnect}
              className={[
                'inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-60',
                pressableButtonClassName,
              ].join(' ')}
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
              className={[
                'rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800',
                pressableButtonClassName,
              ].join(' ')}
            >
              Back
            </button>
            <button
              type="button"
              disabled={busy || coursesToImport.length === 0}
              onClick={onImport}
              className={[
                'rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-60',
                pressableButtonClassName,
              ].join(' ')}
            >
              Import <span className="tabular-nums">{coursesToImport.length}</span> course
              {coursesToImport.length === 1 ? '' : 's'}
            </button>
          </>
        ) : null}
        {step === 'importing' && busy && !importComplete ? (
          <button
            type="button"
            onClick={onCancelImport}
            className={[
              'rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800',
              pressableButtonClassName,
            ].join(' ')}
          >
            Cancel import
          </button>
        ) : null}
        {importComplete && onDismiss ? (
          <button
            type="button"
            onClick={onDismiss}
            className={[
              'rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500',
              pressableButtonClassName,
            ].join(' ')}
          >
            Done
          </button>
        ) : null}
      </div>
    </div>
  )
}

import { GraduationCap } from 'lucide-react'
import { CanvasAccessTokenSettingsLink } from '../../components/canvas/canvas-access-token-settings-link'
import { CanvasReadOnlyNotice } from '../../components/canvas/canvas-read-only-notice'
import { CanvasImportProgressLog } from '../../components/canvas/canvas-import-progress-log'
import type { CanvasImportProgressEntry } from '../../hooks/use-canvas-import-progress-log'
import { BookLoader } from '../../components/quiz/book-loader'
import { clearCanvasImportCredentials } from '../../lib/canvas-import-credentials'
import type { CanvasImportInclude } from '../../lib/courses-api'

export type CourseCanvasImportPanelProps = {
  busy: boolean
  importing: boolean
  canvasBaseUrl: string
  canvasCourseId: string
  canvasToken: string
  canvasInclude: CanvasImportInclude
  canvasImportLog: CanvasImportProgressEntry[]
  rememberCanvasCredentials: boolean
  enableCanvasGradeSyncOnImport: boolean
  onCanvasBaseUrlChange: (value: string) => void
  onCanvasCourseIdChange: (value: string) => void
  onCanvasTokenChange: (value: string) => void
  onCanvasIncludeChange: (next: CanvasImportInclude) => void
  onRememberCredentialsChange: (value: boolean) => void
  onEnableGradeSyncChange: (value: boolean) => void
  onImport: () => void
}

const CANVAS_INCLUDE_OPTIONS = [
  ['modules', 'Modules', 'Outline, wiki pages, discussions, links, and other module items (not assignments/quizzes).'],
  ['assignments', 'Assignments', 'Assignment prompts, due dates, and submission settings.'],
  ['quizzes', 'Quizzes', 'Quizzes and questions when Canvas exposes them.'],
  [
    'enrollments',
    'Enrollments',
    'Active and invited roster; creates Lextures accounts from Canvas emails when needed.',
  ],
  [
    'grades',
    'Grades',
    'Per-learner scores on imported assignments and quizzes (max points + gradebook cells). Learners need an email in Canvas; accounts are created when missing.',
  ],
  ['settings', 'Settings', 'Course title, overview, dates, visibility, and syllabus sections.'],
  [
    'announcements',
    'Announcements',
    'Course announcements posted in Canvas, imported into the Feed #announcements channel.',
  ],
  [
    'files',
    'Files',
    'Course file folders and attachments from Canvas Files (shown on the course Files page).',
  ],
] as const

export function CourseCanvasImportPanel({
  busy,
  importing,
  canvasBaseUrl,
  canvasCourseId,
  canvasToken,
  canvasInclude,
  canvasImportLog,
  rememberCanvasCredentials,
  enableCanvasGradeSyncOnImport,
  onCanvasBaseUrlChange,
  onCanvasCourseIdChange,
  onCanvasTokenChange,
  onCanvasIncludeChange,
  onRememberCredentialsChange,
  onEnableGradeSyncChange,
  onImport,
}: CourseCanvasImportPanelProps) {
  return (
    <div className="mt-8 border-t border-slate-200 pt-8 dark:border-neutral-600">
      <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
        From Canvas LMS
      </h3>
      <CanvasReadOnlyNotice className="mt-3" />
      <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">
        Use a Canvas personal access token with read-only access. Choose what to pull below (all
        are on by default). We map Canvas into this course; roster members with an email in
        Canvas get a Lextures account when needed and are enrolled when enrollments are included.
        The token is sent once for the import (HTTPS); Lextures does not store it on the server.
        Imports run in the background — you can leave this page and refresh later; we notify you
        when the import finishes. You can optionally keep the URL and token in this browser for
        the next course import.
      </p>
      <fieldset className="mt-4 rounded-xl border border-slate-200 p-4 dark:border-neutral-600">
        <legend className="px-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
          Import from Canvas
        </legend>
        <div className="mt-2 grid gap-2 sm:grid-cols-2">
          {CANVAS_INCLUDE_OPTIONS.map(([key, label, hint]) => (
            <label
              key={key}
              className="flex cursor-pointer items-start gap-2 rounded-lg border border-transparent px-1 py-1 hover:border-slate-200 hover:bg-slate-50 dark:hover:border-neutral-600 dark:hover:bg-neutral-800/60"
            >
              <input
                type="checkbox"
                className="mt-0.5"
                checked={canvasInclude[key]}
                onChange={(e) =>
                  onCanvasIncludeChange({ ...canvasInclude, [key]: e.target.checked })
                }
              />
              <span>
                <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                  {label}
                </span>
                <span className="mt-0.5 block text-xs text-slate-500 dark:text-neutral-500">
                  {hint}
                </span>
              </span>
            </label>
          ))}
        </div>
      </fieldset>
      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        <label className="block sm:col-span-2">
          <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">
            Canvas base URL
          </span>
          <input
            type="url"
            value={canvasBaseUrl}
            onChange={(e) => onCanvasBaseUrlChange(e.target.value)}
            placeholder="https://yourschool.instructure.com"
            autoComplete="off"
            className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-inner outline-none ring-indigo-500/0 transition-[background-color,color,border-color] focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
          />
          <CanvasAccessTokenSettingsLink canvasBaseUrl={canvasBaseUrl} className="mt-1.5" />
        </label>
        <label className="block">
          <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">
            Canvas course ID
          </span>
          <input
            type="text"
            inputMode="numeric"
            value={canvasCourseId}
            onChange={(e) => onCanvasCourseIdChange(e.target.value)}
            placeholder="e.g. 1234567"
            autoComplete="off"
            className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-inner outline-none ring-indigo-500/0 transition-[background-color,color,border-color] focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
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
            className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-inner outline-none ring-indigo-500/0 transition-[background-color,color,border-color] focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
          />
        </label>
      </div>
      <label className="mt-3 flex cursor-pointer items-start gap-2 rounded-xl border border-slate-200 p-3 dark:border-neutral-600">
        <input
          type="checkbox"
          className="mt-0.5"
          checked={rememberCanvasCredentials}
          onChange={(e) => {
            const on = e.target.checked
            onRememberCredentialsChange(on)
            if (!on) {
              clearCanvasImportCredentials()
            }
          }}
        />
        <span>
          <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
            Save Canvas URL and access token on this device
          </span>
          <span className="mt-0.5 block text-xs text-slate-500 dark:text-neutral-500">
            Reuses the same connection when you import into other courses in Lextures. Stored
            only in this browser; avoid on shared computers.
          </span>
        </span>
      </label>
      <label className="mt-3 flex cursor-pointer items-start gap-2 rounded-xl border border-sky-200 bg-sky-50/80 p-3 dark:border-sky-900/50 dark:bg-sky-950/30">
        <input
          type="checkbox"
          className="mt-0.5"
          checked={enableCanvasGradeSyncOnImport}
          onChange={(e) => onEnableGradeSyncChange(e.target.checked)}
        />
        <span>
          <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
            Sync grades back to Canvas when grading
          </span>
          <span className="mt-0.5 block text-xs text-slate-600 dark:text-neutral-400">
            When enabled, saving a grade in Lextures automatically pushes it to Canvas. Your
            token needs permission to update grades in Canvas.
          </span>
        </span>
      </label>
      <p className="mt-3 text-xs text-slate-500 dark:text-neutral-500">
        In Canvas: Account or Profile → Settings → New access token. Use a token with
        permission to read the course, assignments, pages, quizzes, enrollments, and the course
        user list (roster).
      </p>
      <div className="mt-4">
        <button
          type="button"
          onClick={onImport}
          disabled={
            busy ||
            !canvasBaseUrl.trim() ||
            !canvasCourseId.trim() ||
            !canvasToken.trim()
          }
          aria-busy={importing}
          className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {importing ? (
            <span
              className="inline-flex shrink-0 items-center justify-center overflow-visible"
              aria-hidden
            >
              <span className="inline-flex origin-center translate-y-[4px] scale-[0.3]">
                <BookLoader className="![--quiz-book-loader-color:rgba(255,255,255,0.92)]" />
              </span>
            </span>
          ) : (
            <GraduationCap className="h-4 w-4 shrink-0" aria-hidden />
          )}
          {importing ? 'Importing from Canvas…' : 'Import from Canvas'}
        </button>
      </div>
      <CanvasImportProgressLog
        entries={canvasImportLog}
        active={importing}
        className="mt-3"
      />
    </div>
  )
}

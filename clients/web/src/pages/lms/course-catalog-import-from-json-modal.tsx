import { type ChangeEvent, type DragEvent, useEffect, useId, useRef, useState } from 'react'
import { CheckCircle2, FileJson, Upload, X } from 'lucide-react'
import { BookLoader } from '../../components/quiz/book-loader'
import { useInboxNotifications } from '../../context/use-push-notifications'
import { useBumpCoursesRevision } from '../../context/use-inbox-unread'
import {
  courseExportImportStatLines,
  summarizeCourseExportBundle,
  type CourseExportImportStats,
} from '../../lib/course-export-import-stats'
import {
  createCourse,
  postCourseImport,
  type CourseExportBundle,
  type CoursePublic,
} from '../../lib/courses-api'

type Props = {
  open: boolean
  onClose: () => void
  onImported: (course: CoursePublic) => void
}

type PendingJson = {
  fileName: string
  export: CourseExportBundle
  stats: CourseExportImportStats
}

type Step = 'pick' | 'review' | 'importing' | 'success'

const SUCCESS_CLOSE_MS = 1600

function descriptionFromExport(bundle: CourseExportBundle): string {
  const course = bundle.course
  if (course && typeof course === 'object' && !Array.isArray(course)) {
    const desc = (course as Record<string, unknown>).description
    if (typeof desc === 'string' && desc.trim()) return desc.trim()
  }
  return ''
}

function isJsonFile(file: File): boolean {
  if (file.type === 'application/json' || file.type === 'text/json') return true
  return /\.json$/i.test(file.name)
}

export function CourseCatalogImportFromJsonModal({ open, onClose, onImported }: Props) {
  if (!open) return null
  return <CourseCatalogImportFromJsonModalInner onClose={onClose} onImported={onImported} />
}

function CourseCatalogImportFromJsonModalInner({
  onClose,
  onImported,
}: Omit<Props, 'open'>) {
  const titleId = useId()
  const statusId = useId()
  const nameId = useId()
  const statsListId = useId()
  const fileRef = useRef<HTMLInputElement>(null)
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const bumpCoursesRevision = useBumpCoursesRevision()
  const { refresh: refreshInboxNotifications } = useInboxNotifications()

  const [step, setStep] = useState<Step>('pick')
  const [pending, setPending] = useState<PendingJson | null>(null)
  const [newTitle, setNewTitle] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [createdCourse, setCreatedCourse] = useState<CoursePublic | null>(null)
  const [dragOver, setDragOver] = useState(false)

  const busy = step === 'importing' || step === 'success'

  useEffect(() => {
    return () => {
      if (closeTimerRef.current) clearTimeout(closeTimerRef.current)
    }
  }, [])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key !== 'Escape' || busy) return
      e.preventDefault()
      onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [busy, onClose])

  function scheduleCloseAfterSuccess(course: CoursePublic) {
    if (closeTimerRef.current) clearTimeout(closeTimerRef.current)
    closeTimerRef.current = setTimeout(() => {
      onImported(course)
      bumpCoursesRevision()
      void refreshInboxNotifications()
      onClose()
    }, SUCCESS_CLOSE_MS)
  }

  async function processFile(file: File) {
    setError(null)
    setDragOver(false)
    try {
      if (!isJsonFile(file)) {
        throw new Error('Please drop a .json course export file.')
      }
      const text = await file.text()
      let parsed: unknown
      try {
        parsed = JSON.parse(text) as unknown
      } catch {
        throw new Error('That file is not valid JSON.')
      }
      const stats = summarizeCourseExportBundle(parsed)
      setPending({
        fileName: file.name,
        export: parsed as CourseExportBundle,
        stats,
      })
      setNewTitle(stats.title?.trim() || file.name.replace(/\.json$/i, '') || 'Imported course')
      setStep('review')
    } catch (err) {
      setPending(null)
      setStep('pick')
      setError(err instanceof Error ? err.message : 'Could not read import file.')
    }
  }

  async function onPickFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    await processFile(file)
  }

  function onDropZoneDragEnter(e: DragEvent) {
    e.preventDefault()
    e.stopPropagation()
    if (!busy) setDragOver(true)
  }

  function onDropZoneDragOver(e: DragEvent) {
    e.preventDefault()
    e.stopPropagation()
    if (busy) return
    e.dataTransfer.dropEffect = 'copy'
    setDragOver(true)
  }

  function onDropZoneDragLeave(e: DragEvent) {
    e.preventDefault()
    e.stopPropagation()
    if (e.currentTarget.contains(e.relatedTarget as Node | null)) return
    setDragOver(false)
  }

  function onDropZoneDrop(e: DragEvent) {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(false)
    if (busy) return
    const file = e.dataTransfer.files?.[0]
    if (!file) {
      setError('No file was dropped. Drop a .json course export file.')
      return
    }
    void processFile(file)
  }

  function clearFile() {
    if (busy) return
    setPending(null)
    setNewTitle('')
    setError(null)
    setDragOver(false)
    setStep('pick')
  }

  async function onSubmit() {
    if (!pending) return
    setError(null)
    const title = newTitle.trim()
    if (!title) {
      setError('Enter a name for the new course.')
      return
    }
    setStep('importing')
    try {
      const description = descriptionFromExport(pending.export) || title
      const created = await createCourse({ title, description })
      bumpCoursesRevision()
      await postCourseImport(created.courseCode, {
        mode: 'erase',
        export: pending.export,
      })
      setCreatedCourse(created)
      setStep('success')
      bumpCoursesRevision()
      void refreshInboxNotifications()
      scheduleCloseAfterSuccess(created)
    } catch (err) {
      setStep('review')
      setError(err instanceof Error ? err.message : 'Import failed.')
      void refreshInboxNotifications()
    }
  }

  const heading =
    step === 'success'
      ? 'Course created'
      : step === 'importing'
        ? 'Creating your course'
        : 'Import from JSON'

  const subheading =
    step === 'success' && createdCourse
      ? `${createdCourse.title} is in your catalog.`
      : step === 'importing'
        ? 'Creating the course and applying the export. This may take a moment.'
        : 'Create a new course from a Lextures JSON export file (same format as course settings export).'

  const pendingStatLines = pending ? courseExportImportStatLines(pending.stats) : []

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      aria-describedby={statusId}
      onClick={(e) => {
        if (e.target === e.currentTarget && !busy) onClose()
      }}
    >
      <div
        className="flex max-h-[min(92vh,720px)] w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
        onDragOver={(e) => {
          // Prevent the browser from navigating away if the file is dropped outside the drop zone.
          e.preventDefault()
        }}
        onDrop={(e) => {
          e.preventDefault()
        }}
      >
        <div className="flex items-start justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
          <div>
            <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              {heading}
            </h2>
            <p id={statusId} className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">
              {subheading}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
            aria-label="Close"
          >
            <X className="h-5 w-5" aria-hidden />
          </button>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
          {step === 'importing' ? (
            <div className="flex flex-col items-center justify-center gap-4 py-10 text-center" aria-live="polite">
              <div className="inline-flex origin-center scale-[0.45]">
                <BookLoader />
              </div>
              <p className="text-sm font-medium text-slate-800 dark:text-neutral-200">
                Importing {pending?.fileName ?? 'JSON export'}…
              </p>
              <p className="max-w-xs text-sm text-slate-500 dark:text-neutral-400">
                We will close this window when the course is ready.
              </p>
            </div>
          ) : null}

          {step === 'success' && createdCourse ? (
            <div className="flex flex-col items-center justify-center gap-3 py-10 text-center" aria-live="polite">
              <CheckCircle2 className="h-12 w-12 text-emerald-600 dark:text-emerald-400" aria-hidden />
              <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">{createdCourse.title}</p>
              <p className="text-sm text-slate-500 dark:text-neutral-400">
                Added to your catalog as <code className="text-xs">{createdCourse.courseCode}</code>.
              </p>
            </div>
          ) : null}

          {step === 'pick' || step === 'review' ? (
            <>
              {error ? (
                <p
                  className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200"
                  role="alert"
                >
                  {error}
                </p>
              ) : null}

              <input
                ref={fileRef}
                type="file"
                accept="application/json,.json"
                className="hidden"
                onChange={(e) => void onPickFile(e)}
              />

              {step === 'pick' ? (
                <div
                  onDragEnter={onDropZoneDragEnter}
                  onDragOver={onDropZoneDragOver}
                  onDragLeave={onDropZoneDragLeave}
                  onDrop={onDropZoneDrop}
                  className={`flex flex-col items-center gap-4 rounded-xl border-2 border-dashed px-4 py-10 text-center transition-[background-color,color,border-color] ${
                    dragOver
                      ? 'border-indigo-400 bg-indigo-50/80 dark:border-indigo-500 dark:bg-indigo-950/30'
                      : 'border-slate-300 bg-slate-50/80 dark:border-neutral-600 dark:bg-neutral-800/40'
                  }`}
                >
                  <FileJson
                    className={`h-10 w-10 ${
                      dragOver
                        ? 'text-indigo-500 dark:text-indigo-400'
                        : 'text-slate-400 dark:text-neutral-500'
                    }`}
                    aria-hidden
                  />
                  <div>
                    <p className="text-sm font-medium text-slate-800 dark:text-neutral-200">
                      {dragOver ? 'Drop JSON file here' : 'Drop a course export JSON file here'}
                    </p>
                    <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                      Or choose a file below. Same format as Download JSON export in course settings.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => fileRef.current?.click()}
                    className="inline-flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:bg-neutral-700"
                  >
                    <Upload className="h-4 w-4" aria-hidden />
                    Choose JSON file…
                  </button>
                </div>
              ) : null}

              {step === 'review' && pending ? (
                <div className="space-y-4">
                  <div className="flex items-start justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 dark:border-neutral-600 dark:bg-neutral-800/60">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-slate-900 dark:text-neutral-100">
                        {pending.fileName}
                      </p>
                      {pending.stats.sourceCourseCode ? (
                        <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                          Source code:{' '}
                          <code className="text-xs">{pending.stats.sourceCourseCode}</code>
                          {' '}
                          (informational only — a new course code is assigned)
                        </p>
                      ) : null}
                    </div>
                    <button
                      type="button"
                      onClick={clearFile}
                      className="shrink-0 text-xs font-semibold text-indigo-600 hover:text-indigo-500 dark:text-indigo-400"
                    >
                      Change file
                    </button>
                  </div>

                  <div>
                    <label
                      htmlFor={nameId}
                      className="block text-sm font-medium text-slate-800 dark:text-neutral-200"
                    >
                      New course name
                    </label>
                    <input
                      id={nameId}
                      type="text"
                      value={newTitle}
                      onChange={(e) => setNewTitle(e.target.value)}
                      placeholder="e.g. Intro to Biology — Spring 2027"
                      className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 shadow-sm outline-none placeholder:text-slate-400 focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
                    />
                  </div>

                  {pendingStatLines.length > 0 ? (
                    <div>
                      <p
                        id={statsListId}
                        className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400"
                      >
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
                            <span className="text-xs text-slate-600 dark:text-neutral-300">
                              {line.label}
                            </span>
                            <span className="text-sm font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
                              {line.count}
                            </span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  ) : (
                    <p className="text-sm text-slate-500 dark:text-neutral-400">
                      This file has no modules, bodies, syllabus sections, grading groups, or
                      enrollments counted for preview. You can still import if the server accepts the
                      format.
                    </p>
                  )}

                  {pending.stats.hasCourseSettings ? (
                    <p className="text-xs text-slate-500 dark:text-neutral-500">
                      Course settings from the file (schedule, feature flags, appearance) will be
                      applied after the new course is created.
                    </p>
                  ) : null}
                </div>
              ) : null}
            </>
          ) : null}
        </div>

        {step === 'review' ? (
          <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-4 dark:border-neutral-700">
            <button
              type="button"
              onClick={onClose}
              className="rounded-xl px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => void onSubmit()}
              disabled={!pending || !newTitle.trim()}
              className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              <FileJson className="h-4 w-4 shrink-0" aria-hidden />
              Create course
            </button>
          </div>
        ) : null}
      </div>
    </div>
  )
}

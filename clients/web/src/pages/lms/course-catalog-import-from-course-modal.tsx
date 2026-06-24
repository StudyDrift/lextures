import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { CheckCircle2, Copy, X } from 'lucide-react'
import { BookLoader } from '../../components/quiz/book-loader'
import { useInboxNotifications } from '../../context/use-push-notifications'
import { useBumpCoursesRevision } from '../../context/use-inbox-unread'
import {
  COURSE_COPY_INCLUDE_ALL,
  postCourseImportFromCourse,
  type CourseCopyInclude,
  type CoursePublic,
} from '../../lib/courses-api'
import { courseCatalogDisplayTitle } from './course-catalog-display'

type Props = {
  open: boolean
  courses: CoursePublic[]
  onClose: () => void
  onImported: (course: CoursePublic) => void
}

type Step = 'form' | 'importing' | 'success'

const SUCCESS_CLOSE_MS = 1600

const INCLUDE_OPTIONS: {
  key: keyof CourseCopyInclude
  label: string
  hint: string
}[] = [
  ['modules', 'Modules & pages', 'Outline, wiki pages, discussions, links, and other module items (not assignments/quizzes).'],
  ['assignments', 'Assignments', 'Assignment prompts, due dates, and submission settings.'],
  ['quizzes', 'Quizzes', 'Quizzes and question banks attached to module items.'],
  ['enrollments', 'Enrollments', 'Active roster members and their roles (you remain the teacher on the new course).'],
  ['grades', 'Grades', 'Gradebook scores for copied assignments and quizzes.'],
  ['settings', 'Settings', 'Syllabus, grading groups, dates, visibility, and course feature flags.'],
  ['files', 'Files', 'Course file folders and attachments from the Files page.'],
].map(([key, label, hint]) => ({
  key: key as keyof CourseCopyInclude,
  label,
  hint,
}))

export function CourseCatalogImportFromCourseModal({ open, courses, onClose, onImported }: Props) {
  if (!open) return null
  return <CourseCatalogImportFromCourseModalInner courses={courses} onClose={onClose} onImported={onImported} />
}

function CourseCatalogImportFromCourseModalInner({
  courses,
  onClose,
  onImported,
}: Omit<Props, 'open'>) {
  const titleId = useId()
  const statusId = useId()
  const sourceId = useId()
  const nameId = useId()
  const bumpCoursesRevision = useBumpCoursesRevision()
  const { refresh: refreshInboxNotifications } = useInboxNotifications()
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const [sourceCourseCode, setSourceCourseCode] = useState('')
  const [newTitle, setNewTitle] = useState('')
  const [include, setInclude] = useState<CourseCopyInclude>(COURSE_COPY_INCLUDE_ALL)
  const [step, setStep] = useState<Step>('form')
  const [error, setError] = useState<string | null>(null)
  const [createdCourse, setCreatedCourse] = useState<CoursePublic | null>(null)

  const busy = step === 'importing' || step === 'success'

  const sortedCourses = useMemo(
    () => [...courses].sort((a, b) => courseCatalogDisplayTitle(a).localeCompare(courseCatalogDisplayTitle(b))),
    [courses],
  )

  const selectedCourse = useMemo(
    () => sortedCourses.find((c) => c.courseCode === sourceCourseCode) ?? null,
    [sortedCourses, sourceCourseCode],
  )

  useEffect(() => {
    if (!selectedCourse) return
    setNewTitle((prev) => (prev.trim() ? prev : `${courseCatalogDisplayTitle(selectedCourse)} (copy)`))
  }, [selectedCourse])

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

  async function onSubmit() {
    setError(null)
    const code = sourceCourseCode.trim()
    const title = newTitle.trim()
    if (!code) {
      setError('Choose a course to copy from.')
      return
    }
    if (!title) {
      setError('Enter a name for the new course.')
      return
    }
    setStep('importing')
    try {
      const created = await postCourseImportFromCourse({
        sourceCourseCode: code,
        title,
        include,
      })
      setCreatedCourse(created)
      setStep('success')
      bumpCoursesRevision()
      void refreshInboxNotifications()
      scheduleCloseAfterSuccess(created)
    } catch (e) {
      setStep('form')
      setError(e instanceof Error ? e.message : 'Import failed.')
      void refreshInboxNotifications()
    }
  }

  const heading =
    step === 'success'
      ? 'Course created'
      : step === 'importing'
        ? 'Creating your course'
        : 'Import from another course'

  const subheading =
    step === 'success' && createdCourse
      ? `${createdCourse.title} is in your catalog.`
      : step === 'importing'
        ? 'Copying content into your new course. This may take a moment.'
        : 'Create a new course and copy selected content from one you already teach.'

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
      <div className="flex max-h-[min(92vh,720px)] w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
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
                Importing from {selectedCourse ? courseCatalogDisplayTitle(selectedCourse) : 'source course'}…
              </p>
              <p className="max-w-xs text-sm text-slate-500 dark:text-neutral-400">
                We will close this window and add a notification when the copy finishes.
              </p>
            </div>
          ) : null}

          {step === 'success' && createdCourse ? (
            <div className="flex flex-col items-center justify-center gap-3 py-10 text-center" aria-live="polite">
              <CheckCircle2 className="h-12 w-12 text-emerald-600 dark:text-emerald-400" aria-hidden />
              <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">{createdCourse.title}</p>
              <p className="text-sm text-slate-500 dark:text-neutral-400">
                Added to your catalog. Check the notification bell for details.
              </p>
            </div>
          ) : null}

          {step === 'form' ? (
            <>
              {error ? (
                <p className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200" role="alert">
                  {error}
                </p>
              ) : null}

              <label htmlFor={sourceId} className="block text-sm font-medium text-slate-800 dark:text-neutral-200">
                Copy from
              </label>
              <select
                id={sourceId}
                value={sourceCourseCode}
                onChange={(e) => setSourceCourseCode(e.target.value)}
                disabled={sortedCourses.length === 0}
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 shadow-sm outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
              >
                <option value="">Select a course…</option>
                {sortedCourses.map((c) => (
                  <option key={c.id} value={c.courseCode}>
                    {courseCatalogDisplayTitle(c)} ({c.courseCode})
                  </option>
                ))}
              </select>

              <label htmlFor={nameId} className="mt-4 block text-sm font-medium text-slate-800 dark:text-neutral-200">
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

              <fieldset className="mt-5 rounded-xl border border-slate-200 p-4 dark:border-neutral-600">
                <legend className="px-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
                  What to import
                </legend>
                <div className="mt-2 grid gap-2">
                  {INCLUDE_OPTIONS.map(({ key, label, hint }) => (
                    <label
                      key={key}
                      className="flex cursor-pointer items-start gap-2 rounded-lg border border-transparent px-1 py-1 hover:border-slate-200 hover:bg-slate-50 dark:hover:border-neutral-600 dark:hover:bg-neutral-800/60"
                    >
                      <input
                        type="checkbox"
                        className="mt-0.5"
                        checked={include[key]}
                        onChange={(e) => setInclude((prev) => ({ ...prev, [key]: e.target.checked }))}
                      />
                      <span>
                        <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">{label}</span>
                        <span className="mt-0.5 block text-xs text-slate-500 dark:text-neutral-500">{hint}</span>
                      </span>
                    </label>
                  ))}
                </div>
              </fieldset>
            </>
          ) : null}
        </div>

        {step === 'form' ? (
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
              disabled={!sourceCourseCode.trim() || !newTitle.trim()}
              className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              <Copy className="h-4 w-4 shrink-0" aria-hidden />
              Create course
            </button>
          </div>
        ) : null}
      </div>
    </div>
  )
}
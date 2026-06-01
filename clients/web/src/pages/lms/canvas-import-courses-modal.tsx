import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'
import { Search, X } from 'lucide-react'
import { CanvasReadOnlyNotice } from '../../components/canvas/canvas-read-only-notice'
import { BookLoader } from '../../components/quiz/book-loader'
import {
  clearCanvasImportCredentials,
  loadCanvasImportCredentials,
  saveCanvasImportCredentials,
} from '../../lib/canvas-import-credentials'
import {
  CANVAS_IMPORT_CANCELLED_MESSAGE,
  CANVAS_IMPORT_INCLUDE_ALL,
  createCourse,
  fetchCanvasCourses,
  postCourseImportCanvas,
  type CanvasCourseListItem,
} from '../../lib/courses-api'

type Step = 'credentials' | 'select' | 'importing'

function isCanvasCourseUnpublished(workflowState: string | undefined): boolean {
  return workflowState?.trim().toLowerCase() === 'unpublished'
}

function courseMatchesNameFilter(course: CanvasCourseListItem, query: string): boolean {
  const q = query.trim().toLowerCase()
  if (!q) return true
  const haystack = [course.name, course.courseCode, course.termName, String(course.id)]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
  return haystack.includes(q)
}

type Props = {
  open: boolean
  onClose: () => void
  onImported?: () => void
}

export function CanvasImportCoursesModal({ open, onClose, onImported }: Props) {
  const titleId = useId()
  const [step, setStep] = useState<Step>('credentials')
  const [canvasBaseUrl, setCanvasBaseUrl] = useState('')
  const [canvasToken, setCanvasToken] = useState('')
  const [rememberCredentials, setRememberCredentials] = useState(false)
  const [courses, setCourses] = useState<CanvasCourseListItem[] | null>(null)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [importStatus, setImportStatus] = useState<string | null>(null)
  const [importDone, setImportDone] = useState<{
    ok: number
    failed: string[]
    cancelled: boolean
  } | null>(null)
  const importCancelledRef = useRef(false)
  const activeCourseImportAbortRef = useRef<AbortController | null>(null)
  const [nameFilter, setNameFilter] = useState('')
  const [hideUnpublished, setHideUnpublished] = useState(false)

  const filteredCourses = useMemo(() => {
    if (!courses) return []
    return courses.filter((c) => {
      if (hideUnpublished && isCanvasCourseUnpublished(c.workflowState)) return false
      return courseMatchesNameFilter(c, nameFilter)
    })
  }, [courses, hideUnpublished, nameFilter])

  const coursesToImport = useMemo(() => {
    if (!courses) return []
    return courses.filter((c) => selected.has(c.id))
  }, [courses, selected])

  const selectedVisibleCount = useMemo(
    () => filteredCourses.filter((c) => selected.has(c.id)).length,
    [filteredCourses, selected],
  )

  const selectedHiddenCount = coursesToImport.length - selectedVisibleCount

  const reset = useCallback(() => {
    setStep('credentials')
    setCourses(null)
    setSelected(new Set())
    setBusy(false)
    setError(null)
    setImportStatus(null)
    setImportDone(null)
    setNameFilter('')
    setHideUnpublished(false)
    importCancelledRef.current = false
    activeCourseImportAbortRef.current?.abort()
    activeCourseImportAbortRef.current = null
  }, [])

  useEffect(() => {
    if (!open) return
    reset()
    const saved = loadCanvasImportCredentials()
    if (!saved) return
    setCanvasBaseUrl(saved.canvasBaseUrl)
    setCanvasToken(saved.accessToken)
    setRememberCredentials(true)
  }, [open, reset])

  useEffect(() => {
    if (!hideUnpublished || !courses?.length) return
    setSelected((prev) => {
      let changed = false
      const next = new Set(prev)
      for (const id of prev) {
        const course = courses.find((c) => c.id === id)
        if (course && isCanvasCourseUnpublished(course.workflowState)) {
          next.delete(id)
          changed = true
        }
      }
      return changed ? next : prev
    })
  }, [hideUnpublished, courses])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !busy) {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, busy, onClose])

  async function onConnect() {
    setError(null)
    const base = canvasBaseUrl.trim()
    const token = canvasToken.trim()
    if (!base || !token) {
      setError('Canvas base URL and access token are required.')
      return
    }
    setBusy(true)
    try {
      const list = await fetchCanvasCourses({ canvasBaseUrl: base, accessToken: token })
      setCourses(list)
      setSelected(new Set())
      setStep('select')
      if (rememberCredentials) {
        saveCanvasImportCredentials(base, token)
      } else {
        clearCanvasImportCredentials()
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load Canvas courses.')
    } finally {
      setBusy(false)
    }
  }

  function toggleCourse(id: number) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function selectAllVisible() {
    if (!filteredCourses.length) return
    setSelected(new Set(filteredCourses.map((c) => c.id)))
  }

  function clearSelection() {
    setSelected(new Set())
  }

  function requestCancelImport() {
    importCancelledRef.current = true
    activeCourseImportAbortRef.current?.abort()
    setImportStatus('Stopping import…')
  }

  async function onImport() {
    if (coursesToImport.length === 0) return
    setError(null)
    setImportDone(null)
    importCancelledRef.current = false
    activeCourseImportAbortRef.current = null
    setStep('importing')
    setBusy(true)
    const base = canvasBaseUrl.trim()
    const token = canvasToken.trim()
    const toImport = coursesToImport
    let ok = 0
    const failed: string[] = []

    for (let i = 0; i < toImport.length; i++) {
      if (importCancelledRef.current) break
      const canvasCourse = toImport[i]!
      setImportStatus(`Importing ${i + 1} of ${toImport.length}: ${canvasCourse.name}`)
      try {
        const created = await createCourse({
          title: canvasCourse.name,
          description: canvasCourse.courseCode?.trim() || canvasCourse.name,
        })
        if (importCancelledRef.current) break
        const courseAbort = new AbortController()
        activeCourseImportAbortRef.current = courseAbort
        await postCourseImportCanvas(
          created.courseCode,
          {
            mode: 'erase',
            canvasBaseUrl: base,
            canvasCourseId: String(canvasCourse.id),
            accessToken: token,
            include: CANVAS_IMPORT_INCLUDE_ALL,
          },
          (message) => setImportStatus(`${canvasCourse.name}: ${message}`),
          { signal: courseAbort.signal },
        )
        ok++
      } catch (e) {
        const msg = e instanceof Error ? e.message : 'Import failed'
        if (msg === CANVAS_IMPORT_CANCELLED_MESSAGE || importCancelledRef.current) {
          break
        }
        failed.push(`${canvasCourse.name}: ${msg}`)
      } finally {
        activeCourseImportAbortRef.current = null
      }
    }

    const cancelled = importCancelledRef.current
    setImportDone({ ok, failed, cancelled })
    setImportStatus(null)
    setBusy(false)
    if (ok > 0) onImported?.()
    if (!rememberCredentials) setCanvasToken('')
  }

  if (!open) return null

  const canClose = !busy

  return (
    <div className="fixed inset-0 z-[400] flex items-center justify-center p-4" role="presentation">
      <button
        type="button"
        aria-label="Close dialog"
        disabled={!canClose}
        className="absolute inset-0 cursor-default border-0 bg-black/45 p-0 disabled:cursor-not-allowed"
        onClick={() => {
          if (canClose) onClose()
        }}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className={[
          'relative flex w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900',
          step === 'importing'
            ? 'min-h-[32rem] max-h-[min(92vh,880px)]'
            : 'max-h-[min(90vh,720px)]',
        ].join(' ')}
      >
        <div className="flex items-start justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
          <div>
            <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              Import from Canvas
            </h2>
            <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">
              {step === 'credentials'
                ? 'Connect with your Canvas URL and personal access token.'
                : step === 'select'
                  ? 'Choose which courses to create in Lextures.'
                  : 'Creating courses and pulling content from Canvas…'}
            </p>
          </div>
          <button
            type="button"
            disabled={!canClose}
            onClick={() => {
              if (canClose) onClose()
            }}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
            aria-label="Close"
          >
            <X className="h-5 w-5" aria-hidden />
          </button>
        </div>

        <div
          className={[
            'min-h-0 flex-1 px-5 py-4',
            step === 'importing' ? 'overflow-visible' : 'overflow-y-auto',
          ].join(' ')}
        >
          {error && (
            <p className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
              {error}
            </p>
          )}

          {step === 'credentials' && (
            <div className="space-y-4">
              <CanvasReadOnlyNotice />
              <p className="text-sm text-slate-600 dark:text-neutral-400">
                Create a token in Canvas under Account → Settings → New Access Token (read-only
                scopes are sufficient). The token is sent to our server only for this session and
                is not stored on the server.
              </p>
              <label className="block">
                <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">
                  Canvas base URL
                </span>
                <input
                  type="url"
                  value={canvasBaseUrl}
                  onChange={(e) => setCanvasBaseUrl(e.target.value)}
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
                  onChange={(e) => setCanvasToken(e.target.value)}
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
                  onChange={(e) => {
                    const on = e.target.checked
                    setRememberCredentials(on)
                    if (!on) clearCanvasImportCredentials()
                  }}
                />
                <span className="text-sm text-slate-700 dark:text-neutral-300">
                  Remember URL and token in this browser
                </span>
              </label>
            </div>
          )}

          {step === 'select' && courses && (
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
                      onChange={(e) => setNameFilter(e.target.value)}
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
                    onChange={(e) => setHideUnpublished(e.target.checked)}
                  />
                  <span className="text-sm text-slate-700 dark:text-neutral-300">
                    Hide unpublished courses
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
                    onClick={selectAllVisible}
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
                    onClick={clearSelection}
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
                          onChange={() => toggleCourse(c.id)}
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
                Check the courses you want, then import. Each one is created in Lextures and filled
                from Canvas (modules, assignments, quizzes, roster enrollments, grades, and settings).
                Learners with an email in Canvas get Lextures accounts when needed. Canvas is not
                modified.
              </p>
            </div>
          )}

          {step === 'importing' && (
            <div className="flex min-h-[280px] flex-col items-center justify-center gap-6 overflow-visible py-10">
              {busy && !importDone && (
                <>
                  <div className="overflow-visible px-6 pt-8" aria-hidden>
                    <span className="inline-flex origin-center scale-[0.65] sm:scale-[0.75]">
                      <BookLoader className="![--quiz-book-loader-color:rgb(79,70,229)] dark:![--quiz-book-loader-color:rgb(129,140,248)]" />
                    </span>
                  </div>
                  {importStatus && (
                    <p
                      className="canvas-import-status-in max-w-sm text-center text-sm text-slate-600 dark:text-neutral-400"
                      role="status"
                      aria-live="polite"
                    >
                      {importStatus}
                    </p>
                  )}
                </>
              )}
              {importDone && (
                <div className="w-full space-y-3 text-sm">
                  {importDone.cancelled ? (
                    <p className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 dark:border-neutral-600 dark:bg-neutral-800/60 dark:text-neutral-200">
                      Import stopped.
                      {importDone.ok > 0
                        ? ` ${importDone.ok} course${importDone.ok === 1 ? ' was' : 's were'} imported before you cancelled.`
                        : ' No courses were fully imported.'}
                    </p>
                  ) : importDone.ok > 0 ? (
                    <p className="rounded-xl border border-emerald-200 bg-emerald-50 px-3 py-2 text-emerald-900 dark:border-emerald-900/50 dark:bg-emerald-950/40 dark:text-emerald-200">
                      Imported {importDone.ok} course{importDone.ok === 1 ? '' : 's'} successfully.
                    </p>
                  ) : null}
                  {importDone.failed.length > 0 && (
                    <div className="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-amber-950 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100">
                      <p className="font-medium">Some imports failed:</p>
                      <ul className="mt-2 list-inside list-disc space-y-1 text-xs">
                        {importDone.failed.map((line) => (
                          <li key={line}>{line}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-4 dark:border-neutral-700">
          {step === 'credentials' && (
            <>
              <button
                type="button"
                disabled={busy}
                onClick={onClose}
                className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => void onConnect()}
                className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-60"
              >
                {busy ? 'Connecting…' : 'Load courses'}
              </button>
            </>
          )}
          {step === 'select' && (
            <>
              <button
                type="button"
                disabled={busy}
                onClick={() => {
                  setStep('credentials')
                  setCourses(null)
                  setNameFilter('')
                  setHideUnpublished(false)
                }}
                className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
              >
                Back
              </button>
              <button
                type="button"
                disabled={busy || coursesToImport.length === 0}
                onClick={() => void onImport()}
                className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-60"
              >
                Import {coursesToImport.length} course{coursesToImport.length === 1 ? '' : 's'}
              </button>
            </>
          )}
          {step === 'importing' && busy && !importDone && (
            <button
              type="button"
              onClick={requestCancelImport}
              className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              Cancel import
            </button>
          )}
          {step === 'importing' && importDone && (
            <button
              type="button"
              onClick={onClose}
              className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
            >
              Done
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

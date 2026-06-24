/* eslint-disable react-refresh/only-export-components -- context module exports provider + hooks */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from 'react'
import {
  clearCanvasImportCredentials,
  loadCanvasImportCredentials,
  saveCanvasImportCredentials,
} from '../lib/canvas-import-credentials'
import { useBumpCoursesRevision } from './use-inbox-unread'
import { useCanvasImportProgressLog } from '../hooks/use-canvas-import-progress-log'
import {
  CANVAS_IMPORT_CANCELLED_MESSAGE,
  CANVAS_IMPORT_INCLUDE_ALL,
  createCourse,
  fetchCanvasCourses,
  postCourseImportCanvas,
  type CanvasCourseListItem,
} from '../lib/courses-api'
import { BookLoader } from '../components/quiz/book-loader'
import {
  CanvasImportCoursesPanel,
  type CanvasImportCoursesStep,
} from '../pages/lms/canvas-import-courses-panel'

const AUTO_MINIMIZE_MS = 5000
const COLLAPSE_ANIMATION_MS = 520

export type CanvasImportPresentation = 'closed' | 'fullscreen' | 'collapsing' | 'minimized' | 'popover'

function courseMatchesNameFilter(course: CanvasCourseListItem, query: string): boolean {
  const q = query.trim().toLowerCase()
  if (!q) return true
  const haystack = [course.name, course.courseCode, course.termName, String(course.id)]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
  return haystack.includes(q)
}

function isCanvasCourseUnpublished(workflowState: string | undefined): boolean {
  return workflowState?.trim().toLowerCase() === 'unpublished'
}

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false
  return (
    document.documentElement.classList.contains('reduced-motion') ||
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

type CanvasImportContextValue = {
  presentation: CanvasImportPresentation
  step: CanvasImportCoursesStep
  isSessionActive: boolean
  isImporting: boolean
  importComplete: boolean
  anchorRef: RefObject<HTMLButtonElement | null>
  open: () => void
  dismiss: () => void
  togglePopover: () => void
  panelProps: React.ComponentProps<typeof CanvasImportCoursesPanel>
}

const CanvasImportContext = createContext<CanvasImportContextValue | null>(null)

export function useCanvasImport() {
  const ctx = useContext(CanvasImportContext)
  if (!ctx) {
    throw new Error('useCanvasImport must be used within CanvasImportProvider')
  }
  return ctx
}

export function useCanvasImportOptional() {
  return useContext(CanvasImportContext)
}

type CollapseFrame = {
  left: number
  top: number
  width: number
  height: number
  transform: string
  opacity: number
  borderRadius: string
  transition: string
}

export function CanvasImportProvider({ children }: { children: ReactNode }) {
  const bumpCoursesRevision = useBumpCoursesRevision()
  const anchorRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const collapseTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const minimizeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const [presentation, setPresentation] = useState<CanvasImportPresentation>('closed')
  const [collapseFrame, setCollapseFrame] = useState<CollapseFrame | null>(null)
  const [backdropOpacity, setBackdropOpacity] = useState(1)

  const [step, setStep] = useState<CanvasImportCoursesStep>('credentials')
  const [canvasBaseUrl, setCanvasBaseUrl] = useState('')
  const [canvasToken, setCanvasToken] = useState('')
  const [rememberCredentials, setRememberCredentials] = useState(false)
  const [courses, setCourses] = useState<CanvasCourseListItem[] | null>(null)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [busy, setBusy] = useState(false)
  const [importComplete, setImportComplete] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { entries: importLog, append: appendImportLog, clear: clearImportLog } =
    useCanvasImportProgressLog()
  const importCancelledRef = useRef(false)
  const activeCourseImportAbortsRef = useRef<Set<AbortController>>(new Set())
  const presentationRef = useRef(presentation)
  presentationRef.current = presentation
  const [nameFilter, setNameFilter] = useState('')
  const [hideUnpublished, setHideUnpublished] = useState(false)
  const [enableCanvasGradeSync, setEnableCanvasGradeSync] = useState(false)

  const clearTimers = useCallback(() => {
    if (collapseTimerRef.current) {
      clearTimeout(collapseTimerRef.current)
      collapseTimerRef.current = null
    }
    if (minimizeTimerRef.current) {
      clearTimeout(minimizeTimerRef.current)
      minimizeTimerRef.current = null
    }
  }, [])

  const resetSession = useCallback(() => {
    clearTimers()
    setPresentation('closed')
    setCollapseFrame(null)
    setBackdropOpacity(1)
    setStep('credentials')
    setCourses(null)
    setSelected(new Set())
    setBusy(false)
    setImportComplete(false)
    setError(null)
    clearImportLog()
    setNameFilter('')
    setHideUnpublished(false)
    setEnableCanvasGradeSync(false)
    importCancelledRef.current = false
    for (const controller of activeCourseImportAbortsRef.current) {
      controller.abort()
    }
    activeCourseImportAbortsRef.current.clear()
  }, [clearImportLog, clearTimers])

  const dismiss = useCallback(() => {
    if (busy && !importComplete) return
    resetSession()
  }, [busy, importComplete, resetSession])

  const open = useCallback(() => {
    resetSession()
    const saved = loadCanvasImportCredentials()
    if (saved) {
      setCanvasBaseUrl(saved.canvasBaseUrl)
      setCanvasToken(saved.accessToken)
      setRememberCredentials(true)
    }
    setPresentation('fullscreen')
  }, [resetSession])

  const togglePopover = useCallback(() => {
    if (presentation === 'popover') {
      if (importComplete) {
        resetSession()
      } else {
        setPresentation('minimized')
      }
      return
    }
    if (presentation === 'minimized' || presentation === 'collapsing') {
      setPresentation('popover')
    }
  }, [importComplete, presentation, resetSession])

  const startCollapse = useCallback(() => {
    if (presentation !== 'fullscreen' || step !== 'importing') return

    const panel = panelRef.current
    const anchor = anchorRef.current
    if (!panel || !anchor || prefersReducedMotion()) {
      setPresentation('minimized')
      return
    }

    const from = panel.getBoundingClientRect()
    const to = anchor.getBoundingClientRect()
    const scaleX = Math.max(to.width / from.width, 0.08)
    const scaleY = Math.max(to.height / from.height, 0.08)
    const dx = to.left + to.width / 2 - (from.left + from.width / 2)
    const dy = to.top + to.height / 2 - (from.top + from.height / 2)

    setCollapseFrame({
      left: from.left,
      top: from.top,
      width: from.width,
      height: from.height,
      transform: 'translate(0px, 0px) scale(1)',
      opacity: 1,
      borderRadius: '16px',
      transition: 'none',
    })
    setPresentation('collapsing')
    setBackdropOpacity(0)

    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        setCollapseFrame({
          left: from.left,
          top: from.top,
          width: from.width,
          height: from.height,
          transform: `translate(${dx}px, ${dy}px) scale(${scaleX}, ${scaleY})`,
          opacity: 0.15,
          borderRadius: '12px',
          transition: `transform ${COLLAPSE_ANIMATION_MS}ms cubic-bezier(0.4, 0, 0.2, 1), opacity ${COLLAPSE_ANIMATION_MS}ms ease, border-radius ${COLLAPSE_ANIMATION_MS}ms ease`,
        })
      })
    })

    minimizeTimerRef.current = setTimeout(() => {
      setPresentation('minimized')
      setCollapseFrame(null)
      setBackdropOpacity(1)
    }, COLLAPSE_ANIMATION_MS + 40)
  }, [presentation, step])

  useEffect(() => {
    if (presentation !== 'fullscreen' || step !== 'importing' || !busy) {
      clearTimers()
      return
    }
    collapseTimerRef.current = setTimeout(() => {
      startCollapse()
    }, AUTO_MINIMIZE_MS)
    return clearTimers
  }, [busy, clearTimers, presentation, startCollapse, step])

  useEffect(() => {
    if (presentation !== 'fullscreen') return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && (!busy || importComplete)) {
        e.preventDefault()
        dismiss()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [busy, dismiss, importComplete, presentation])

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

  const onConnect = useCallback(async () => {
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
  }, [canvasBaseUrl, canvasToken, rememberCredentials])

  const requestCancelImport = useCallback(() => {
    importCancelledRef.current = true
    for (const controller of activeCourseImportAbortsRef.current) {
      controller.abort()
    }
    activeCourseImportAbortsRef.current.clear()
    appendImportLog('Stopping import…')
  }, [appendImportLog])

  const runImport = useCallback(async () => {
    if (coursesToImport.length === 0) return
    setError(null)
    clearImportLog()
    importCancelledRef.current = false
    for (const controller of activeCourseImportAbortsRef.current) {
      controller.abort()
    }
    activeCourseImportAbortsRef.current.clear()
    setStep('importing')
    setBusy(true)
    setImportComplete(false)
    const base = canvasBaseUrl.trim()
    const token = canvasToken.trim()
    const toImport = coursesToImport

    const results = await Promise.all(
      toImport.map(async (canvasCourse, i) => {
        if (importCancelledRef.current) return false
        appendImportLog(`Importing ${i + 1} of ${toImport.length}: ${canvasCourse.name}`)
        try {
          const created = await createCourse({
            title: canvasCourse.name,
            description: canvasCourse.courseCode?.trim() || canvasCourse.name,
          })
          bumpCoursesRevision()
          if (importCancelledRef.current) return false
          const courseAbort = new AbortController()
          activeCourseImportAbortsRef.current.add(courseAbort)
          try {
            await postCourseImportCanvas(
              created.courseCode,
              {
                mode: 'erase',
                canvasBaseUrl: base,
                canvasCourseId: String(canvasCourse.id),
                accessToken: token,
                include: CANVAS_IMPORT_INCLUDE_ALL,
                ...(enableCanvasGradeSync ? { canvasGradeSyncEnabled: true } : {}),
              },
              (message) => appendImportLog(`${canvasCourse.name}: ${message}`),
              {
                signal: courseAbort.signal,
                onCoursesUpdated: bumpCoursesRevision,
              },
            )
            return true
          } finally {
            activeCourseImportAbortsRef.current.delete(courseAbort)
          }
        } catch (e) {
          const msg = e instanceof Error ? e.message : 'Import failed'
          if (msg === CANVAS_IMPORT_CANCELLED_MESSAGE || importCancelledRef.current) {
            return false
          }
          appendImportLog(`${canvasCourse.name}: ${msg}`)
          return false
        }
      }),
    )

    const ok = results.filter(Boolean).length
    setBusy(false)
    if (ok > 0) bumpCoursesRevision()
    if (!rememberCredentials) setCanvasToken('')

    if (importCancelledRef.current) {
      appendImportLog('Import cancelled.')
      setImportComplete(true)
      return
    }

    if (ok > 0) {
      setImportComplete(true)
      appendImportLog(`Finished importing ${ok} course${ok === 1 ? '' : 's'}.`)
      if (presentationRef.current === 'fullscreen') {
        resetSession()
      }
      return
    }

    setError('No courses were imported successfully.')
    setImportComplete(true)
  }, [
    appendImportLog,
    bumpCoursesRevision,
    canvasBaseUrl,
    canvasToken,
    clearImportLog,
    coursesToImport,
    enableCanvasGradeSync,
    rememberCredentials,
    resetSession,
  ])

  const onRememberCredentialsChange = useCallback((on: boolean) => {
    setRememberCredentials(on)
    if (!on) clearCanvasImportCredentials()
  }, [])

  const onToggleCourse = useCallback((id: number) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const onSelectAllVisible = useCallback(() => {
    if (!filteredCourses.length) return
    setSelected(new Set(filteredCourses.map((c) => c.id)))
  }, [filteredCourses])

  const onClearSelection = useCallback(() => setSelected(new Set()), [])

  const onBackToCredentials = useCallback(() => {
    setStep('credentials')
    setCourses(null)
    setNameFilter('')
    setHideUnpublished(false)
  }, [])

  const onImport = useCallback(() => void runImport(), [runImport])

  const panelProps = useMemo(
    (): React.ComponentProps<typeof CanvasImportCoursesPanel> => ({
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
      selectedVisibleCount,
      selectedHiddenCount,
      onCanvasBaseUrlChange: setCanvasBaseUrl,
      onCanvasTokenChange: setCanvasToken,
      onRememberCredentialsChange,
      onNameFilterChange: setNameFilter,
      onHideUnpublishedChange: setHideUnpublished,
      onEnableCanvasGradeSyncChange: setEnableCanvasGradeSync,
      onToggleCourse,
      onSelectAllVisible,
      onClearSelection,
      onConnect: () => void onConnect(),
      onBackToCredentials,
      onImport,
      onCancelImport: requestCancelImport,
      onDismiss: dismiss,
      showClose: true,
    }),
    [
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
      selectedVisibleCount,
      selectedHiddenCount,
      onRememberCredentialsChange,
      onToggleCourse,
      onSelectAllVisible,
      onClearSelection,
      onConnect,
      onBackToCredentials,
      onImport,
      requestCancelImport,
      dismiss,
    ],
  )

  const value = useMemo(
    (): CanvasImportContextValue => ({
      presentation,
      step,
      isSessionActive: presentation !== 'closed',
      isImporting: step === 'importing' && busy && !importComplete,
      importComplete,
      anchorRef,
      open,
      dismiss,
      togglePopover,
      panelProps,
    }),
    [presentation, step, busy, importComplete, open, dismiss, togglePopover, panelProps],
  )

  return (
    <CanvasImportContext.Provider value={value}>
      {children}
      <CanvasImportOverlay
        presentation={presentation}
        panelRef={panelRef}
        collapseFrame={collapseFrame}
        backdropOpacity={backdropOpacity}
        panelProps={panelProps}
        onDismiss={dismiss}
        busy={busy}
        importComplete={importComplete}
      />
    </CanvasImportContext.Provider>
  )
}

function CanvasImportOverlay({
  presentation,
  panelRef,
  collapseFrame,
  backdropOpacity,
  panelProps,
  onDismiss,
  busy,
  importComplete,
}: {
  presentation: CanvasImportPresentation
  panelRef: RefObject<HTMLDivElement | null>
  collapseFrame: CollapseFrame | null
  backdropOpacity: number
  panelProps: React.ComponentProps<typeof CanvasImportCoursesPanel>
  onDismiss: () => void
  busy: boolean
  importComplete: boolean
}) {
  const canDismissBackdrop = !busy || importComplete

  if (presentation === 'fullscreen') {
    return (
      <div className="fixed inset-0 z-[400] flex items-center justify-center p-4" role="presentation">
        <button
          type="button"
          aria-label="Close dialog"
          disabled={!canDismissBackdrop}
          className="absolute inset-0 cursor-default border-0 bg-black/45 p-0 motion-safe:transition-opacity duration-300 disabled:cursor-not-allowed"
          style={{ opacity: backdropOpacity }}
          onClick={() => {
            if (canDismissBackdrop) onDismiss()
          }}
        />
        <div
          ref={panelRef}
          role="dialog"
          aria-modal="true"
          className="relative flex w-full max-w-lg flex-col"
        >
          <CanvasImportCoursesPanel {...panelProps} />
        </div>
      </div>
    )
  }

  if (presentation === 'collapsing' && collapseFrame) {
    return (
      <div className="pointer-events-none fixed inset-0 z-[400]" aria-hidden>
        <div
          ref={panelRef}
          className="pointer-events-none fixed overflow-hidden border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
          style={{
            left: collapseFrame.left,
            top: collapseFrame.top,
            width: collapseFrame.width,
            height: collapseFrame.height,
            transform: collapseFrame.transform,
            opacity: collapseFrame.opacity,
            borderRadius: collapseFrame.borderRadius,
            transition: collapseFrame.transition,
            transformOrigin: 'center center',
          }}
        >
          <CanvasImportCoursesPanel {...panelProps} compact showClose={false} />
        </div>
      </div>
    )
  }

  return null
}

export function CanvasImportHeaderWidget() {
  const ctx = useCanvasImportOptional()
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!ctx || ctx.presentation !== 'popover') return
    const { togglePopover } = ctx
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) togglePopover()
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') togglePopover()
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [ctx])

  if (!ctx) return null

  const { presentation, step, isSessionActive, isImporting, importComplete, anchorRef, togglePopover, panelProps } =
    ctx

  const showWidget =
    isSessionActive &&
    (step === 'importing' ||
      presentation === 'minimized' ||
      presentation === 'popover' ||
      presentation === 'collapsing')

  if (!showWidget) return null

  const popoverOpen = presentation === 'popover'
  const anchorVisible =
    presentation === 'minimized' || presentation === 'popover' || presentation === 'collapsing'

  return (
    <div
      ref={rootRef}
      className={[
        'relative shrink-0 transition-opacity duration-300',
        anchorVisible ? 'opacity-100' : 'pointer-events-none opacity-0',
      ].join(' ')}
      aria-hidden={!anchorVisible}
    >
      <button
        ref={anchorRef}
        type="button"
        data-testid="canvas-import-anchor"
        tabIndex={anchorVisible ? 0 : -1}
        aria-label={
          importComplete
            ? 'Canvas import finished. Open import details.'
            : isImporting
              ? 'Canvas import in progress. Open import details.'
              : 'Canvas import. Open details.'
        }
        aria-expanded={popoverOpen}
        aria-haspopup="dialog"
        onClick={togglePopover}
        className={`relative inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl transition-[background-color,color,border-color] focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 ${
          popoverOpen
            ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
            : 'text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800'
        }`}
      >
        {importComplete ? (
          <span className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-emerald-500 text-[11px] font-bold text-white">
            ✓
          </span>
        ) : (
          <span className="inline-flex h-5 w-5 items-center justify-center overflow-visible" aria-hidden>
            <BookLoader className="quiz-book-loader-icon ![--quiz-book-loader-color:rgb(79,70,229)] dark:![--quiz-book-loader-color:rgb(129,140,248)]" />
          </span>
        )}
      </button>

      {popoverOpen ? (
        <div
          role="dialog"
          aria-modal="true"
          aria-label="Canvas import"
          data-testid="canvas-import-popover"
          className="absolute end-0 top-full z-50 mt-1 flex w-[min(32rem,calc(100vw-2rem))] max-h-[min(85vh,720px)] flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-900 dark:shadow-black/40"
        >
          <CanvasImportCoursesPanel {...panelProps} compact className="max-h-[min(85vh,720px)]" />
        </div>
      ) : null}
    </div>
  )
}
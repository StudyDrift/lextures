import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from 'react'

const DEFAULT_SIDEBAR_WIDTH = 320
const MIN_SIDEBAR_WIDTH = 260
const MIN_CANVAS_WIDTH = 320
const MAX_SIDEBAR_RATIO = 0.6
const STORAGE_KEY = 'block-editor-sidebar-width'

type BlockEditorShellProps = {
  /** Main canvas (blocks). */
  children: ReactNode
  /** Right settings column (tabs + panels). */
  sidebar: ReactNode
  className?: string
  /** localStorage key for persisted sidebar width. */
  widthStorageKey?: string
  defaultSidebarWidth?: number
  minSidebarWidth?: number
}

function clampSidebarWidth(
  width: number,
  containerWidth: number,
  minSidebarWidth: number,
  minCanvasWidth: number,
): number {
  if (containerWidth <= 0) {
    return Math.max(minSidebarWidth, width)
  }
  const maxSidebar = Math.max(
    minSidebarWidth,
    Math.min(containerWidth * MAX_SIDEBAR_RATIO, containerWidth - minCanvasWidth),
  )
  return Math.max(minSidebarWidth, Math.min(maxSidebar, width))
}

function storageGet(key: string): string | null {
  try {
    return globalThis.localStorage?.getItem(key) ?? null
  } catch {
    return null
  }
}

function storageSet(key: string, value: string): void {
  try {
    globalThis.localStorage?.setItem(key, value)
  } catch {
    /* ignore quota / private mode / missing storage */
  }
}

function readStoredWidth(storageKey: string, fallback: number, minSidebarWidth: number): number {
  const raw = storageGet(storageKey)
  if (!raw) return fallback
  const parsed = Number(raw)
  if (!Number.isFinite(parsed)) return fallback
  return clampSidebarWidth(parsed, 1600, minSidebarWidth, MIN_CANVAS_WIDTH)
}

/**
 * Two-column Gutenberg-style layout: scrollable canvas + resizable settings sidebar.
 * On md+ the sidebar width can be dragged horizontally; preference is persisted.
 */
export function BlockEditorShell({
  children,
  sidebar,
  className,
  widthStorageKey = STORAGE_KEY,
  defaultSidebarWidth = DEFAULT_SIDEBAR_WIDTH,
  minSidebarWidth = MIN_SIDEBAR_WIDTH,
}: BlockEditorShellProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const handleId = useId()
  const sidebarId = useId()
  const [sidebarWidth, setSidebarWidth] = useState(() =>
    readStoredWidth(widthStorageKey, defaultSidebarWidth, minSidebarWidth),
  )
  const [dragging, setDragging] = useState(false)

  const updateWidthFromPointer = useCallback(
    (clientX: number) => {
      const container = containerRef.current
      if (!container) return
      const rect = container.getBoundingClientRect()
      // Sidebar is the trailing flex column (visual right in LTR and RTL flex-row).
      const next = clampSidebarWidth(
        rect.right - clientX,
        rect.width,
        minSidebarWidth,
        MIN_CANVAS_WIDTH,
      )
      setSidebarWidth(next)
    },
    [minSidebarWidth],
  )

  useEffect(() => {
    if (!dragging) return
    const previousCursor = document.body.style.cursor
    const previousUserSelect = document.body.style.userSelect
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    return () => {
      document.body.style.cursor = previousCursor
      document.body.style.userSelect = previousUserSelect
    }
  }, [dragging])

  useEffect(() => {
    storageSet(widthStorageKey, String(Math.round(sidebarWidth)))
  }, [sidebarWidth, widthStorageKey])

  useEffect(() => {
    const container = containerRef.current
    if (!container) return
    const observer = new ResizeObserver(() => {
      setSidebarWidth((width) => {
        const containerWidth = container.getBoundingClientRect().width
        // Mobile stacks; only clamp when the two-column layout is active.
        if (containerWidth < 768) return width
        return clampSidebarWidth(width, containerWidth, minSidebarWidth, MIN_CANVAS_WIDTH)
      })
    })
    observer.observe(container)
    return () => observer.disconnect()
  }, [minSidebarWidth])

  const endDrag = useCallback((target: HTMLDivElement, pointerId: number) => {
    try {
      if (target.hasPointerCapture(pointerId)) {
        target.releasePointerCapture(pointerId)
      }
    } catch {
      /* ignore */
    }
    setDragging(false)
  }, [])

  const onHandlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      if (e.button !== 0) return
      e.preventDefault()
      e.currentTarget.setPointerCapture(e.pointerId)
      setDragging(true)
      updateWidthFromPointer(e.clientX)
    },
    [updateWidthFromPointer],
  )

  const onHandlePointerMove = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      if (!e.currentTarget.hasPointerCapture(e.pointerId)) return
      updateWidthFromPointer(e.clientX)
    },
    [updateWidthFromPointer],
  )

  const onHandleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      const step = e.shiftKey ? 48 : 16
      if (e.key === 'ArrowLeft') {
        e.preventDefault()
        setSidebarWidth((w) => {
          const container = containerRef.current
          const containerWidth = container?.getBoundingClientRect().width ?? 1600
          return clampSidebarWidth(w + step, containerWidth, minSidebarWidth, MIN_CANVAS_WIDTH)
        })
      } else if (e.key === 'ArrowRight') {
        e.preventDefault()
        setSidebarWidth((w) => {
          const container = containerRef.current
          const containerWidth = container?.getBoundingClientRect().width ?? 1600
          return clampSidebarWidth(w - step, containerWidth, minSidebarWidth, MIN_CANVAS_WIDTH)
        })
      } else if (e.key === 'Home') {
        e.preventDefault()
        setSidebarWidth(minSidebarWidth)
      } else if (e.key === 'End') {
        e.preventDefault()
        setSidebarWidth(() => {
          const container = containerRef.current
          const containerWidth = container?.getBoundingClientRect().width ?? 1600
          return clampSidebarWidth(
            containerWidth * MAX_SIDEBAR_RATIO,
            containerWidth,
            minSidebarWidth,
            MIN_CANVAS_WIDTH,
          )
        })
      }
    },
    [minSidebarWidth],
  )

  return (
    <div
      ref={containerRef}
      className={
        className ??
        'block-editor-root flex min-h-[min(70vh,720px)] w-full flex-col overflow-hidden rounded-xl border border-slate-200/90 bg-[#f0f0f0] shadow-sm shadow-slate-900/5 dark:border-border-default dark:bg-surface-base dark:shadow-black/40 md:min-h-[min(75vh,880px)]'
      }
      style={{ '--block-editor-sidebar-width': `${sidebarWidth}px` } as CSSProperties}
    >
      <div className="flex min-h-0 min-w-0 flex-1 flex-col md:flex-row">
        <div
          role="region"
          aria-label="Block editor canvas"
          className="min-h-0 min-w-0 flex-1 overflow-y-auto bg-[#f0f0f0] dark:bg-surface-base"
        >
          {children}
        </div>

        <div
          id={handleId}
          role="separator"
          aria-orientation="vertical"
          aria-controls={sidebarId}
          aria-valuenow={Math.round(sidebarWidth)}
          aria-valuemin={minSidebarWidth}
          aria-valuemax={720}
          aria-label="Resize editor settings panel"
          tabIndex={0}
          onPointerDown={onHandlePointerDown}
          onPointerMove={onHandlePointerMove}
          onPointerUp={(e) => endDrag(e.currentTarget, e.pointerId)}
          onPointerCancel={(e) => endDrag(e.currentTarget, e.pointerId)}
          onLostPointerCapture={() => setDragging(false)}
          onKeyDown={onHandleKeyDown}
          className={`relative hidden shrink-0 touch-none md:block ${dragging ? 'z-20' : 'z-10'}`}
          style={{ width: 0 }}
          data-testid="block-editor-sidebar-resize"
        >
          <div
            className={`absolute inset-y-0 -start-2 flex w-4 cursor-col-resize items-center justify-center ${ dragging ? 'bg-indigo-500/10' : '' }`}
            aria-hidden="true"
          >
            <div
              className={`h-full w-px transition-colors ${ dragging ? 'bg-indigo-500' : 'bg-slate-300 hover:bg-indigo-400 dark:bg-neutral-600 dark:hover:bg-indigo-400' }`}
            />
            {/* Grip dots for discoverability */}
            <div
              className={`pointer-events-none absolute top-1/2 flex -translate-y-1/2 flex-col gap-1 rounded-full px-0.5 py-1.5 ${ dragging ? 'bg-indigo-500 text-white' : 'bg-slate-200 text-fg-muted hover:bg-slate-300 dark:bg-neutral-700 dark:text-fg-muted' }`}
            >
              <span className="block size-0.5 rounded-full bg-current" />
              <span className="block size-0.5 rounded-full bg-current" />
              <span className="block size-0.5 rounded-full bg-current" />
            </div>
          </div>
        </div>

        <aside
          id={sidebarId}
          aria-label="Editor settings"
          className="flex max-h-[min(42vh,420px)] min-h-0 w-full shrink-0 flex-col border-t border-slate-200/90 bg-surface-raised dark:border-border-default dark:bg-surface-raised md:max-h-none md:w-[var(--block-editor-sidebar-width)] md:border-s md:border-t-0"
        >
          {sidebar}
        </aside>
      </div>
    </div>
  )
}

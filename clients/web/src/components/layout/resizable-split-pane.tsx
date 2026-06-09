import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from 'react'

const DEFAULT_SECONDARY_WIDTH = 448
const MIN_SECONDARY_WIDTH = 280
const MIN_PRIMARY_WIDTH = 360
const MAX_SECONDARY_RATIO = 0.65

type ResizableSplitPaneProps = {
  primary: ReactNode
  secondary: ReactNode
  defaultSecondaryWidth?: number
  minSecondaryWidth?: number
  minPrimaryWidth?: number
  storageKey?: string
  className?: string
}

function clampSecondaryWidth(
  width: number,
  containerWidth: number,
  minSecondaryWidth: number,
  minPrimaryWidth: number,
): number {
  const maxSecondary = Math.max(
    minSecondaryWidth,
    Math.min(containerWidth * MAX_SECONDARY_RATIO, containerWidth - minPrimaryWidth),
  )
  return Math.max(minSecondaryWidth, Math.min(maxSecondary, width))
}

function readStoredWidth(storageKey: string | undefined, fallback: number): number {
  if (!storageKey) return fallback
  try {
    const raw = localStorage.getItem(storageKey)
    if (!raw) return fallback
    const parsed = Number(raw)
    if (!Number.isFinite(parsed)) return fallback
    return clampSecondaryWidth(parsed, 1600, MIN_SECONDARY_WIDTH, MIN_PRIMARY_WIDTH)
  } catch {
    return fallback
  }
}

export function ResizableSplitPane({
  primary,
  secondary,
  defaultSecondaryWidth = DEFAULT_SECONDARY_WIDTH,
  minSecondaryWidth = MIN_SECONDARY_WIDTH,
  minPrimaryWidth = MIN_PRIMARY_WIDTH,
  storageKey,
  className,
}: ResizableSplitPaneProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const handleId = useId()
  const [secondaryWidth, setSecondaryWidth] = useState(() =>
    readStoredWidth(storageKey, defaultSecondaryWidth),
  )
  const [dragging, setDragging] = useState(false)

  const updateWidthFromPointer = useCallback(
    (clientX: number) => {
      const container = containerRef.current
      if (!container) return
      const rect = container.getBoundingClientRect()
      const next = clampSecondaryWidth(
        rect.right - clientX,
        rect.width,
        minSecondaryWidth,
        minPrimaryWidth,
      )
      setSecondaryWidth(next)
    },
    [minPrimaryWidth, minSecondaryWidth],
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
    if (!storageKey) return
    try {
      localStorage.setItem(storageKey, String(Math.round(secondaryWidth)))
    } catch {
      /* ignore quota / private mode */
    }
  }, [secondaryWidth, storageKey])

  useEffect(() => {
    const container = containerRef.current
    if (!container) return
    const observer = new ResizeObserver(() => {
      setSecondaryWidth((width) => {
        const containerWidth = container.getBoundingClientRect().width
        return clampSecondaryWidth(width, containerWidth, minSecondaryWidth, minPrimaryWidth)
      })
    })
    observer.observe(container)
    return () => observer.disconnect()
  }, [minPrimaryWidth, minSecondaryWidth])

  const endDrag = useCallback(
    (target: HTMLDivElement, pointerId: number) => {
      try {
        if (target.hasPointerCapture(pointerId)) {
          target.releasePointerCapture(pointerId)
        }
      } catch {
        /* ignore */
      }
      setDragging(false)
    },
    [],
  )

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

  return (
    <div
      ref={containerRef}
      className={`flex min-h-0 flex-1 flex-col lg:flex-row ${className ?? ''}`}
      style={{ '--split-secondary-width': `${secondaryWidth}px` } as CSSProperties}
    >
      <div className="min-h-[40vh] min-w-0 flex-1 overflow-auto border-b border-slate-200 bg-white dark:border-neutral-600 dark:bg-neutral-800 lg:min-h-0 lg:border-b-0">
        {primary}
      </div>

      <div
        id={handleId}
        role="separator"
        aria-orientation="vertical"
        aria-controls="resizable-split-pane-secondary"
        aria-valuenow={Math.round(secondaryWidth)}
        aria-valuemin={minSecondaryWidth}
        aria-valuemax={720}
        aria-label="Resize submission viewer and grading panel"
        tabIndex={0}
        onPointerDown={onHandlePointerDown}
        onPointerMove={onHandlePointerMove}
        onPointerUp={(e) => endDrag(e.currentTarget, e.pointerId)}
        onPointerCancel={(e) => endDrag(e.currentTarget, e.pointerId)}
        onLostPointerCapture={() => setDragging(false)}
        onKeyDown={(e) => {
          const step = e.shiftKey ? 48 : 16
          if (e.key === 'ArrowLeft') {
            e.preventDefault()
            setSecondaryWidth((w) => {
              const container = containerRef.current
              if (!container) return w + step
              return clampSecondaryWidth(
                w + step,
                container.getBoundingClientRect().width,
                minSecondaryWidth,
                minPrimaryWidth,
              )
            })
          } else if (e.key === 'ArrowRight') {
            e.preventDefault()
            setSecondaryWidth((w) => {
              const container = containerRef.current
              if (!container) return w - step
              return clampSecondaryWidth(
                w - step,
                container.getBoundingClientRect().width,
                minSecondaryWidth,
                minPrimaryWidth,
              )
            })
          }
        }}
        className={`relative hidden shrink-0 touch-none lg:block ${
          dragging ? 'z-20' : 'z-10'
        }`}
        style={{ width: 0 }}
      >
        <div
          className={`absolute inset-y-0 -start-2 flex w-4 cursor-col-resize items-center justify-center ${
            dragging ? 'bg-indigo-500/10' : ''
          }`}
          aria-hidden="true"
        >
          <div
            className={`h-full w-px transition-colors ${
              dragging
                ? 'bg-indigo-500'
                : 'bg-slate-300 hover:bg-indigo-400 dark:bg-neutral-600 dark:hover:bg-indigo-400'
            }`}
          />
        </div>
      </div>

      <div
        id="resizable-split-pane-secondary"
        className="flex min-h-0 w-full shrink-0 flex-col border-t border-slate-200 dark:border-neutral-600 lg:w-[var(--split-secondary-width)] lg:border-t-0 lg:border-l"
      >
        {secondary}
      </div>
    </div>
  )
}

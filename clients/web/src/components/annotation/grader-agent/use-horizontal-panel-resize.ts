import { useCallback, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'

type UseHorizontalPanelResizeOptions = {
  defaultWidth: number
  minWidth: number
  maxWidth: number
}

export function clampResizeWidth(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

export function useHorizontalPanelResize({
  defaultWidth,
  minWidth,
  maxWidth,
}: UseHorizontalPanelResizeOptions) {
  const [width, setWidth] = useState(defaultWidth)
  const dragRef = useRef<{ startX: number; startWidth: number } | null>(null)

  const endDrag = useCallback((target: EventTarget & { releasePointerCapture?: (id: number) => void }, pointerId: number) => {
    dragRef.current = null
    target.releasePointerCapture?.(pointerId)
  }, [])

  const onPointerDown = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      event.preventDefault()
      dragRef.current = { startX: event.clientX, startWidth: width }
      event.currentTarget.setPointerCapture(event.pointerId)
    },
    [width],
  )

  const onPointerMove = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      if (!dragRef.current) return
      const deltaX = dragRef.current.startX - event.clientX
      setWidth(clampResizeWidth(dragRef.current.startWidth + deltaX, minWidth, maxWidth))
    },
    [maxWidth, minWidth],
  )

  const onPointerUp = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      if (!dragRef.current) return
      endDrag(event.currentTarget, event.pointerId)
    },
    [endDrag],
  )

  const onPointerCancel = onPointerUp

  return {
    width,
    resizeHandleProps: {
      onPointerDown,
      onPointerMove,
      onPointerUp,
      onPointerCancel,
    },
  }
}
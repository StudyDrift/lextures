import { useCallback, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'

type UseVerticalPanelResizeOptions = {
  defaultHeight: number
  minHeight: number
  maxHeight: number
}

export function clampResizeHeight(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

export function useVerticalPanelResize({
  defaultHeight,
  minHeight,
  maxHeight,
}: UseVerticalPanelResizeOptions) {
  const [height, setHeight] = useState(defaultHeight)
  const dragRef = useRef<{ startY: number; startHeight: number } | null>(null)

  const endDrag = useCallback((target: EventTarget & { releasePointerCapture?: (id: number) => void }, pointerId: number) => {
    dragRef.current = null
    target.releasePointerCapture?.(pointerId)
  }, [])

  const onPointerDown = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      event.preventDefault()
      dragRef.current = { startY: event.clientY, startHeight: height }
      event.currentTarget.setPointerCapture(event.pointerId)
    },
    [height],
  )

  const onPointerMove = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      if (!dragRef.current) return
      const deltaY = dragRef.current.startY - event.clientY
      setHeight(clampResizeHeight(dragRef.current.startHeight + deltaY, minHeight, maxHeight))
    },
    [maxHeight, minHeight],
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
    height,
    resizeHandleProps: {
      onPointerDown,
      onPointerMove,
      onPointerUp,
      onPointerCancel,
    },
  }
}
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type MouseEvent as ReactMouseEvent,
  type PointerEvent as ReactPointerEvent,
} from 'react'
import {
  eraseFromElements,
  pickElement,
  redrawWhiteboard,
  translateElement,
} from '../../lib/whiteboard/canvas-core'
import {
  WHITEBOARD_COLORS,
  WHITEBOARD_ERASER_SIZES,
  WHITEBOARD_STROKE_WIDTHS,
  type DrawEl,
  type WhiteboardTool,
} from '../../lib/whiteboard/types'

type UseWhiteboardCanvasOptions = {
  elements: DrawEl[]
  onElementsChange: (elements: DrawEl[]) => void
  disabled?: boolean
}

export function useWhiteboardCanvas({ elements, onElementsChange, disabled = false }: UseWhiteboardCanvasOptions) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const [draft, setDraft] = useState<DrawEl | null>(null)
  const [tool, setTool] = useState<WhiteboardTool>('pen')
  const [color, setColor] = useState<string>(WHITEBOARD_COLORS[0])
  const [strokeWidth, setStrokeWidth] = useState<number>(WHITEBOARD_STROKE_WIDTHS[1])
  const [eraserSize, setEraserSize] = useState<number>(WHITEBOARD_ERASER_SIZES[1])
  const [eraserCursorPos, setEraserCursorPos] = useState<[number, number] | null>(null)
  const [selectedIdx, setSelectedIdx] = useState<number | null>(null)
  const [dragInfo, setDragInfo] = useState<{
    idx: number
    startPos: [number, number]
    origEl: DrawEl
  } | null>(null)
  const [isDrawing, setIsDrawing] = useState(false)
  const [origin, setOrigin] = useState<[number, number]>([0, 0])

  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')

  const elementsRef = useRef(elements)
  elementsRef.current = elements

  const setElements = useCallback(
    (updater: DrawEl[] | ((prev: DrawEl[]) => DrawEl[])) => {
      if (disabled) return
      const next = typeof updater === 'function' ? updater(elementsRef.current) : updater
      onElementsChange(next)
    },
    [disabled, onElementsChange],
  )

  useEffect(() => {
    if (!isDrawing) {
      setDraft(null)
      setDragInfo(null)
    }
  }, [elements, isDrawing])

  const resizeCanvas = useCallback(() => {
    const canvas = canvasRef.current
    const container = containerRef.current
    if (!canvas || !container) return
    const { width, height } = container.getBoundingClientRect()
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (ctx) redrawWhiteboard(ctx, width, height, elements, draft, isDark, selectedIdx)
  }, [elements, draft, isDark, selectedIdx])

  useEffect(() => {
    resizeCanvas()
    const obs = new ResizeObserver(() => resizeCanvas())
    if (containerRef.current) obs.observe(containerRef.current)
    return () => obs.disconnect()
  }, [resizeCanvas])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    redrawWhiteboard(ctx, canvas.width, canvas.height, elements, draft, isDark, selectedIdx)
  }, [elements, draft, isDark, selectedIdx])

  useEffect(() => {
    setIsDrawing(false)
    setDraft(null)
    setDragInfo(null)
    if (tool !== 'select') setSelectedIdx(null)
  }, [tool])

  const getPos = useCallback(
    (e: ReactPointerEvent<HTMLCanvasElement> | ReactMouseEvent<HTMLCanvasElement>): [number, number] => {
      const canvas = canvasRef.current!
      const rect = canvas.getBoundingClientRect()
      return [e.clientX - rect.left, e.clientY - rect.top]
    },
    [],
  )

  const onPointerDown = useCallback(
    (e: ReactPointerEvent<HTMLCanvasElement>) => {
      if (disabled) return
      e.stopPropagation()
      const [x, y] = getPos(e)
      ;(e.target as HTMLCanvasElement).setPointerCapture(e.pointerId)

      if (tool === 'select') {
        const idx = pickElement(elements, x, y)
        setSelectedIdx(idx >= 0 ? idx : null)
        if (idx >= 0) {
          setDragInfo({ idx, startPos: [x, y], origEl: elements[idx] })
          setIsDrawing(true)
        }
        return
      }

      setOrigin([x, y])
      setIsDrawing(true)

      if (tool === 'pen') {
        setDraft({ type: 'stroke', color, width: strokeWidth, pts: [[x, y]] })
      } else if (tool === 'eraser') {
        setElements((prev) => eraseFromElements(prev, x, y, eraserSize))
      }
    },
    [color, disabled, elements, eraserSize, getPos, setElements, strokeWidth, tool],
  )

  const onPointerMove = useCallback(
    (e: ReactPointerEvent<HTMLCanvasElement>) => {
      if (disabled) return
      e.stopPropagation()
      const [x, y] = getPos(e)

      if (tool === 'eraser') {
        setEraserCursorPos([x, y])
      }

      if (!isDrawing) return

      if (tool === 'select') {
        if (dragInfo) {
          const dx = x - dragInfo.startPos[0]
          const dy = y - dragInfo.startPos[1]
          setElements((prev) =>
            prev.map((el, i) => (i === dragInfo.idx ? translateElement(dragInfo.origEl, dx, dy) : el)),
          )
        }
        return
      }

      if (tool === 'eraser') {
        const canvas = canvasRef.current!
        const rect = canvas.getBoundingClientRect()
        const coalesced = e.nativeEvent.getCoalescedEvents?.() ?? []
        const pts: [number, number][] =
          coalesced.length > 0
            ? coalesced.map((ce) => [ce.clientX - rect.left, ce.clientY - rect.top])
            : [[x, y]]
        setElements((prev) => {
          let els = prev
          for (const [px, py] of pts) {
            els = eraseFromElements(els, px, py, eraserSize)
          }
          return els
        })
        return
      }

      const [ox, oy] = origin

      if (tool === 'pen') {
        setDraft((prev) => {
          if (!prev || prev.type !== 'stroke') return prev
          return { ...prev, pts: [...prev.pts, [x, y]] }
        })
        return
      }

      if (tool === 'line') {
        setDraft({ type: 'line', color, width: strokeWidth, x1: ox, y1: oy, x2: x, y2: y })
      } else if (tool === 'rect') {
        setDraft({ type: 'rect', color, width: strokeWidth, x: ox, y: oy, w: x - ox, h: y - oy })
      } else if (tool === 'circle') {
        setDraft({
          type: 'circle',
          color,
          width: strokeWidth,
          cx: ox,
          cy: oy,
          rx: (x - ox) / 2,
          ry: (y - oy) / 2,
        })
      } else if (tool === 'triangle') {
        const mx = (ox + x) / 2
        setDraft({
          type: 'triangle',
          color,
          width: strokeWidth,
          x1: mx,
          y1: oy,
          x2: x,
          y2: y,
          x3: ox,
          y3: y,
        })
      }
    },
    [color, disabled, dragInfo, eraserSize, getPos, isDrawing, origin, setElements, strokeWidth, tool],
  )

  const onPointerUp = useCallback(() => {
    if (disabled || !isDrawing) return
    setIsDrawing(false)
    setDragInfo(null)
    if (draft) {
      setElements((prev) => [...prev, draft])
      setDraft(null)
    }
  }, [disabled, draft, isDrawing, setElements])

  const clearCanvas = useCallback(() => {
    setElements([])
    setDraft(null)
    setSelectedIdx(null)
  }, [setElements])

  const exportPng = useCallback((filename = 'whiteboard') => {
    const canvas = canvasRef.current
    if (!canvas) return
    const a = document.createElement('a')
    a.href = canvas.toDataURL('image/png')
    a.download = `${filename}.png`
    a.click()
  }, [])

  const cursor =
    tool === 'select'
      ? dragInfo
        ? 'cursor-grabbing'
        : 'cursor-grab'
      : tool === 'eraser'
        ? 'cursor-none'
        : 'cursor-crosshair'

  return {
    canvasRef,
    containerRef,
    tool,
    setTool,
    color,
    setColor,
    strokeWidth,
    setStrokeWidth,
    eraserSize,
    setEraserSize,
    eraserCursorPos,
    setEraserCursorPos,
    clearCanvas,
    exportPng,
    onPointerDown,
    onPointerMove,
    onPointerUp,
    cursor,
  }
}

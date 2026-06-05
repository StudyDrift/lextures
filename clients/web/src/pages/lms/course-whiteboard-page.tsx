import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type MouseEvent as ReactMouseEvent,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from 'react'
import { useParams } from 'react-router-dom'
import {
  Circle,
  Download,
  Eraser,
  FolderOpen,
  Minus,
  MousePointer2,
  Pencil,
  Save,
  Square,
  Trash2,
  Triangle,
} from 'lucide-react'
import {
  createWhiteboard,
  deleteWhiteboard,
  listWhiteboards,
  updateWhiteboard,
  type WhiteboardRow,
} from '../../lib/courses-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

// ---------------------------------------------------------------------------
// Drawing data model
// ---------------------------------------------------------------------------

type StrokeEl = { type: 'stroke'; color: string; width: number; pts: [number, number][] }
type RectEl = { type: 'rect'; color: string; width: number; x: number; y: number; w: number; h: number }
type CircleEl = { type: 'circle'; color: string; width: number; cx: number; cy: number; rx: number; ry: number }
type TriangleEl = { type: 'triangle'; color: string; width: number; x1: number; y1: number; x2: number; y2: number; x3: number; y3: number }
type LineEl = { type: 'line'; color: string; width: number; x1: number; y1: number; x2: number; y2: number }

type DrawEl = StrokeEl | RectEl | CircleEl | TriangleEl | LineEl

type Tool = 'select' | 'pen' | 'line' | 'rect' | 'circle' | 'triangle' | 'eraser'

const COLORS = ['#1e293b', '#ef4444', '#f97316', '#eab308', '#22c55e', '#3b82f6', '#a855f7', '#ec4899', '#ffffff']
const STROKE_WIDTHS = [2, 4, 8]
const ERASER_SIZES = [8, 16, 32]
const GRID_SPACING = 24

// ---------------------------------------------------------------------------
// Canvas helpers
// ---------------------------------------------------------------------------

function drawGrid(ctx: CanvasRenderingContext2D, w: number, h: number, dark: boolean) {
  ctx.save()
  ctx.fillStyle = dark ? '#171717' : '#ffffff'
  ctx.fillRect(0, 0, w, h)
  ctx.fillStyle = dark ? 'rgba(255,255,255,0.18)' : 'rgba(0,0,0,0.18)'
  for (let x = GRID_SPACING; x < w; x += GRID_SPACING) {
    for (let y = GRID_SPACING; y < h; y += GRID_SPACING) {
      ctx.beginPath()
      ctx.arc(x, y, 1, 0, Math.PI * 2)
      ctx.fill()
    }
  }
  ctx.restore()
}

function drawElement(ctx: CanvasRenderingContext2D, el: DrawEl) {
  ctx.save()
  ctx.strokeStyle = el.color
  ctx.fillStyle = 'transparent'
  ctx.lineWidth = el.width
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  switch (el.type) {
    case 'stroke': {
      if (el.pts.length < 2) break
      ctx.beginPath()
      ctx.moveTo(el.pts[0][0], el.pts[0][1])
      for (let i = 1; i < el.pts.length; i++) ctx.lineTo(el.pts[i][0], el.pts[i][1])
      ctx.stroke()
      break
    }
    case 'rect': {
      ctx.beginPath()
      ctx.strokeRect(el.x, el.y, el.w, el.h)
      break
    }
    case 'circle': {
      ctx.beginPath()
      ctx.ellipse(el.cx, el.cy, Math.abs(el.rx), Math.abs(el.ry), 0, 0, Math.PI * 2)
      ctx.stroke()
      break
    }
    case 'triangle': {
      ctx.beginPath()
      ctx.moveTo(el.x1, el.y1)
      ctx.lineTo(el.x2, el.y2)
      ctx.lineTo(el.x3, el.y3)
      ctx.closePath()
      ctx.stroke()
      break
    }
    case 'line': {
      ctx.beginPath()
      ctx.moveTo(el.x1, el.y1)
      ctx.lineTo(el.x2, el.y2)
      ctx.stroke()
      break
    }
  }
  ctx.restore()
}

// ---------------------------------------------------------------------------
// Eraser hit-testing & partial erase
// ---------------------------------------------------------------------------

function distToSegment(px: number, py: number, ax: number, ay: number, bx: number, by: number): number {
  const dx = bx - ax
  const dy = by - ay
  if (dx === 0 && dy === 0) return Math.hypot(px - ax, py - ay)
  const t = Math.max(0, Math.min(1, ((px - ax) * dx + (py - ay) * dy) / (dx * dx + dy * dy)))
  return Math.hypot(px - (ax + t * dx), py - (ay + t * dy))
}

function effectiveEraserRadius(radius: number, lineWidth: number): number {
  return radius + lineWidth / 2
}

function lerp(a: number, b: number, t: number): number {
  return a + (b - a) * t
}

/** Parameter values t ∈ [0,1] where segment (x1,y1)→(x2,y2) meets circle (cx,cy,r). */
function segmentCircleIntersections(
  cx: number,
  cy: number,
  r: number,
  x1: number,
  y1: number,
  x2: number,
  y2: number,
): number[] {
  const dx = x2 - x1
  const dy = y2 - y1
  const fx = x1 - cx
  const fy = y1 - cy
  const a = dx * dx + dy * dy
  if (a === 0) return Math.hypot(fx, fy) <= r ? [0] : []
  const b = 2 * (fx * dx + fy * dy)
  const c = fx * fx + fy * fy - r * r
  const disc = b * b - 4 * a * c
  if (disc < 0) return []
  const sd = Math.sqrt(disc)
  const t1 = (-b - sd) / (2 * a)
  const t2 = (-b + sd) / (2 * a)
  const out: number[] = []
  if (t1 >= 0 && t1 <= 1) out.push(t1)
  if (t2 >= 0 && t2 <= 1 && Math.abs(t2 - t1) > 1e-9) out.push(t2)
  return out.sort((u, v) => u - v)
}

function hitTestStroke(stroke: StrokeEl, px: number, py: number, radius: number): boolean {
  const r = effectiveEraserRadius(radius, stroke.width)
  for (let i = 0; i < stroke.pts.length; i++) {
    const [ex, ey] = stroke.pts[i]
    if (Math.hypot(ex - px, ey - py) <= r) return true
    if (i > 0) {
      const [ax, ay] = stroke.pts[i - 1]
      if (distToSegment(px, py, ax, ay, ex, ey) <= r) return true
    }
  }
  return false
}

/** Convert any drawable element into one or more polylines for partial erasing. */
function elementToStrokes(el: DrawEl): StrokeEl[] {
  const { color, width } = el
  switch (el.type) {
    case 'stroke':
      return [el]
    case 'line':
      return [{ type: 'stroke', color, width, pts: [[el.x1, el.y1], [el.x2, el.y2]] }]
    case 'rect': {
      const x1 = el.x
      const y1 = el.y
      const x2 = el.x + el.w
      const y2 = el.y + el.h
      const edge = (a: [number, number], b: [number, number]): StrokeEl => ({ type: 'stroke', color, width, pts: [a, b] })
      return [
        edge([x1, y1], [x2, y1]),
        edge([x2, y1], [x2, y2]),
        edge([x2, y2], [x1, y2]),
        edge([x1, y2], [x1, y1]),
      ]
    }
    case 'triangle': {
      const edge = (a: [number, number], b: [number, number]): StrokeEl => ({ type: 'stroke', color, width, pts: [a, b] })
      return [
        edge([el.x1, el.y1], [el.x2, el.y2]),
        edge([el.x2, el.y2], [el.x3, el.y3]),
        edge([el.x3, el.y3], [el.x1, el.y1]),
      ]
    }
    case 'circle': {
      const segments = 36
      const strokes: StrokeEl[] = []
      let prev: [number, number] | null = null
      for (let i = 0; i <= segments; i++) {
        const angle = (2 * Math.PI * i) / segments
        const pt: [number, number] = [el.cx + el.rx * Math.cos(angle), el.cy + el.ry * Math.sin(angle)]
        if (prev) strokes.push({ type: 'stroke', color, width, pts: [prev, pt] })
        prev = pt
      }
      return strokes
    }
  }
}

function hitTest(el: DrawEl, px: number, py: number, radius: number): boolean {
  switch (el.type) {
    case 'stroke':
      return hitTestStroke(el, px, py, radius)
    case 'line':
      return distToSegment(px, py, el.x1, el.y1, el.x2, el.y2) <= effectiveEraserRadius(radius, el.width)
    case 'rect': {
      const x1 = Math.min(el.x, el.x + el.w)
      const x2 = Math.max(el.x, el.x + el.w)
      const y1 = Math.min(el.y, el.y + el.h)
      const y2 = Math.max(el.y, el.y + el.h)
      const r = effectiveEraserRadius(radius, el.width)
      return (
        distToSegment(px, py, x1, y1, x2, y1) <= r ||
        distToSegment(px, py, x2, y1, x2, y2) <= r ||
        distToSegment(px, py, x2, y2, x1, y2) <= r ||
        distToSegment(px, py, x1, y2, x1, y1) <= r
      )
    }
    case 'circle': {
      const rx = Math.abs(el.rx) || 1
      const ry = Math.abs(el.ry) || 1
      const norm = Math.hypot((px - el.cx) / rx, (py - el.cy) / ry)
      return Math.abs(norm - 1) * Math.min(rx, ry) <= effectiveEraserRadius(radius, el.width)
    }
    case 'triangle': {
      const r = effectiveEraserRadius(radius, el.width)
      return (
        distToSegment(px, py, el.x1, el.y1, el.x2, el.y2) <= r ||
        distToSegment(px, py, el.x2, el.y2, el.x3, el.y3) <= r ||
        distToSegment(px, py, el.x3, el.y3, el.x1, el.y1) <= r
      )
    }
  }
}

function pushUniquePoint(cur: [number, number][], x: number, y: number) {
  const n = cur.length
  if (n === 0 || cur[n - 1][0] !== x || cur[n - 1][1] !== y) cur.push([x, y])
}

/** Split a stroke at the eraser circle, clipping segments rather than dropping dense points. */
function splitStroke(stroke: StrokeEl, px: number, py: number, radius: number): StrokeEl[] {
  const pts = stroke.pts
  if (pts.length < 2) return pts.length === 1 && hitTestStroke(stroke, px, py, radius) ? [] : [stroke]

  const r = effectiveEraserRadius(radius, stroke.width)
  const inside = (x: number, y: number) => Math.hypot(x - px, y - py) <= r
  const result: StrokeEl[] = []
  let cur: [number, number][] = []

  const flush = () => {
    if (cur.length >= 2) result.push({ ...stroke, pts: cur })
    cur = []
  }

  const first = pts[0]
  if (!inside(first[0], first[1])) cur.push(first)

  for (let i = 1; i < pts.length; i++) {
    const [x0, y0] = pts[i - 1]
    const [x1, y1] = pts[i]
    const aIn = inside(x0, y0)
    const bIn = inside(x1, y1)

    if (aIn && bIn) {
      flush()
      continue
    }

    const hits = segmentCircleIntersections(px, py, r, x0, y0, x1, y1)

    if (!aIn && !bIn) {
      if (hits.length >= 2) {
        pushUniquePoint(cur, lerp(x0, x1, hits[0]), lerp(y0, y1, hits[0]))
        flush()
        pushUniquePoint(cur, lerp(x0, x1, hits[1]), lerp(y0, y1, hits[1]))
      } else {
        pushUniquePoint(cur, x1, y1)
      }
      continue
    }

    const t = hits[0] ?? (aIn ? 0 : 1)
    const bx = lerp(x0, x1, t)
    const by = lerp(y0, y1, t)

    if (!aIn && bIn) {
      pushUniquePoint(cur, bx, by)
      flush()
    } else {
      flush()
      pushUniquePoint(cur, bx, by)
      pushUniquePoint(cur, x1, y1)
    }
  }

  flush()
  return result
}

/** Partial erase: clip every drawable at the eraser circle boundary. */
function eraseFromElements(elements: DrawEl[], px: number, py: number, radius: number): DrawEl[] {
  const out: DrawEl[] = []
  for (const el of elements) {
    for (const stroke of elementToStrokes(el)) {
      out.push(...splitStroke(stroke, px, py, radius))
    }
  }
  return out
}

// ---------------------------------------------------------------------------
// Select-mode helpers
// ---------------------------------------------------------------------------

function translateElement(el: DrawEl, dx: number, dy: number): DrawEl {
  switch (el.type) {
    case 'stroke': return { ...el, pts: el.pts.map(([x, y]) => [x + dx, y + dy] as [number, number]) }
    case 'rect':   return { ...el, x: el.x + dx, y: el.y + dy }
    case 'circle': return { ...el, cx: el.cx + dx, cy: el.cy + dy }
    case 'line':   return { ...el, x1: el.x1 + dx, y1: el.y1 + dy, x2: el.x2 + dx, y2: el.y2 + dy }
    case 'triangle': return { ...el, x1: el.x1+dx, y1: el.y1+dy, x2: el.x2+dx, y2: el.y2+dy, x3: el.x3+dx, y3: el.y3+dy }
  }
}

/** Returns the index of the topmost element whose outline is within 8px of (px,py), or -1. */
function pickElement(elements: DrawEl[], px: number, py: number): number {
  for (let i = elements.length - 1; i >= 0; i--) {
    if (hitTest(elements[i], px, py, 8)) return i
  }
  return -1
}

function getBoundingBox(el: DrawEl): { x: number; y: number; w: number; h: number } | null {
  switch (el.type) {
    case 'stroke': {
      if (!el.pts.length) return null
      let [minX, minY, maxX, maxY] = [Infinity, Infinity, -Infinity, -Infinity]
      for (const [x, y] of el.pts) { minX=Math.min(minX,x); minY=Math.min(minY,y); maxX=Math.max(maxX,x); maxY=Math.max(maxY,y) }
      return { x: minX, y: minY, w: maxX - minX, h: maxY - minY }
    }
    case 'rect': {
      const x = Math.min(el.x, el.x + el.w), y = Math.min(el.y, el.y + el.h)
      return { x, y, w: Math.abs(el.w), h: Math.abs(el.h) }
    }
    case 'circle':
      return { x: el.cx - Math.abs(el.rx), y: el.cy - Math.abs(el.ry), w: 2*Math.abs(el.rx), h: 2*Math.abs(el.ry) }
    case 'line': {
      const x = Math.min(el.x1, el.x2), y = Math.min(el.y1, el.y2)
      return { x, y, w: Math.abs(el.x2 - el.x1), h: Math.abs(el.y2 - el.y1) }
    }
    case 'triangle': {
      const xs = [el.x1, el.x2, el.x3], ys = [el.y1, el.y2, el.y3]
      const x = Math.min(...xs), y = Math.min(...ys)
      return { x, y, w: Math.max(...xs) - x, h: Math.max(...ys) - y }
    }
  }
}

// ---------------------------------------------------------------------------
// Redraw
// ---------------------------------------------------------------------------

function redraw(
  ctx: CanvasRenderingContext2D,
  w: number,
  h: number,
  elements: DrawEl[],
  draft: DrawEl | null,
  dark: boolean,
  selectedIdx?: number | null,
) {
  drawGrid(ctx, w, h, dark)
  for (const el of elements) drawElement(ctx, el)
  if (draft) drawElement(ctx, draft)
  // Selection highlight
  if (selectedIdx != null && selectedIdx >= 0 && elements[selectedIdx]) {
    const bb = getBoundingBox(elements[selectedIdx])
    if (bb) {
      ctx.save()
      ctx.strokeStyle = '#6366f1'
      ctx.lineWidth = 1.5
      ctx.setLineDash([4, 3])
      ctx.strokeRect(bb.x - 6, bb.y - 6, bb.w + 12, bb.h + 12)
      ctx.restore()
    }
  }
}

// ---------------------------------------------------------------------------
// Save dialog
// ---------------------------------------------------------------------------

function SaveDialog({
  initialTitle,
  onSave,
  onClose,
}: {
  initialTitle: string
  onSave: (title: string) => void
  onClose: () => void
}) {
  const [title, setTitle] = useState(initialTitle)
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        className="w-80 rounded-2xl bg-white p-6 shadow-xl dark:bg-neutral-900"
        onClick={(e) => e.stopPropagation()}
      >
        <p className="mb-3 text-sm font-semibold text-slate-900 dark:text-neutral-100">Save whiteboard</p>
        <input
          autoFocus
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Whiteboard name"
          className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100"
          onKeyDown={(e) => {
            if (e.key === 'Enter' && title.trim()) onSave(title.trim())
            if (e.key === 'Escape') onClose()
          }}
        />
        <div className="mt-4 flex gap-2 justify-end">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={!title.trim()}
            onClick={() => onSave(title.trim())}
            className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Load panel
// ---------------------------------------------------------------------------

function LoadPanel({
  boards,
  onLoad,
  onDelete,
  onClose,
}: {
  boards: WhiteboardRow[]
  onLoad: (b: WhiteboardRow) => void
  onDelete: (id: string) => void
  onClose: () => void
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        className="w-96 max-h-[70vh] overflow-y-auto rounded-2xl bg-white p-6 shadow-xl dark:bg-neutral-900"
        onClick={(e) => e.stopPropagation()}
      >
        <p className="mb-3 text-sm font-semibold text-slate-900 dark:text-neutral-100">Load whiteboard</p>
        {boards.length === 0 ? (
          <p className="text-sm text-slate-500 dark:text-neutral-400">No saved whiteboards yet.</p>
        ) : (
          <ul className="divide-y divide-slate-100 dark:divide-neutral-800">
            {boards.map((b) => (
              <li key={b.id} className="flex items-center justify-between gap-2 py-2">
                <button
                  type="button"
                  onClick={() => onLoad(b)}
                  className="flex-1 text-left text-sm text-slate-800 hover:text-indigo-600 dark:text-neutral-200 dark:hover:text-indigo-400"
                >
                  {b.title}
                </button>
                <button
                  type="button"
                  onClick={() => onDelete(b.id)}
                  className="rounded p-1 text-slate-400 hover:text-rose-500"
                  title="Delete"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="mt-4 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Popover group — shows trigger; hover/click reveals options panel to the right
// ---------------------------------------------------------------------------

function PopoverGroup({ trigger, children }: { trigger: ReactNode; children: ReactNode }) {
  const [open, setOpen] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const show = () => {
    if (timerRef.current) clearTimeout(timerRef.current)
    setOpen(true)
  }
  const hide = () => {
    timerRef.current = setTimeout(() => setOpen(false), 80)
  }

  return (
    <div className="relative" onMouseEnter={show} onMouseLeave={hide}>
      <div onClick={() => setOpen((o) => !o)}>{trigger}</div>
      {open && (
        <div
          className="absolute left-full top-0 z-50 ml-2 rounded-xl border border-slate-200 bg-white p-2 shadow-lg dark:border-neutral-800 dark:bg-neutral-950"
          onMouseEnter={show}
          onMouseLeave={hide}
        >
          {children}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function CourseWhiteboardPage() {
  const { courseCode = '' } = useParams<{ courseCode: string }>()

  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const [elements, setElements] = useState<DrawEl[]>([])
  const [draft, setDraft] = useState<DrawEl | null>(null)
  const [tool, setTool] = useState<Tool>('pen')
  const [color, setColor] = useState(COLORS[0])
  const [strokeWidth, setStrokeWidth] = useState(STROKE_WIDTHS[1])
  const [eraserSize, setEraserSize] = useState(ERASER_SIZES[1])
  const [eraserCursorPos, setEraserCursorPos] = useState<[number, number] | null>(null)
  const [selectedIdx, setSelectedIdx] = useState<number | null>(null)
  const [dragInfo, setDragInfo] = useState<{
    idx: number
    startPos: [number, number]
    origEl: DrawEl
  } | null>(null)
  const [isDrawing, setIsDrawing] = useState(false)
  const [origin, setOrigin] = useState<[number, number]>([0, 0])

  const [boards, setBoards] = useState<WhiteboardRow[]>([])
  const [currentBoard, setCurrentBoard] = useState<WhiteboardRow | null>(null)
  const [showSave, setShowSave] = useState(false)
  const [showLoad, setShowLoad] = useState(false)
  const [saving, setSaving] = useState(false)

  const isDark = document.documentElement.classList.contains('dark')

  // Size canvas to container
  const resizeCanvas = useCallback(() => {
    const canvas = canvasRef.current
    const container = containerRef.current
    if (!canvas || !container) return
    const { width, height } = container.getBoundingClientRect()
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (ctx) redraw(ctx, width, height, elements, draft, isDark, selectedIdx)
  }, [elements, draft, isDark, selectedIdx])

  useEffect(() => {
    resizeCanvas()
    const obs = new ResizeObserver(() => resizeCanvas())
    if (containerRef.current) obs.observe(containerRef.current)
    return () => obs.disconnect()
  }, [resizeCanvas])

  // Redraw whenever elements/draft/selection change
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    redraw(ctx, canvas.width, canvas.height, elements, draft, isDark, selectedIdx)
  }, [elements, draft, isDark, selectedIdx])

  // Reset drawing state when tool changes (prevents eraser/state leaking across tools)
  useEffect(() => {
    setIsDrawing(false)
    setDraft(null)
    setDragInfo(null)
    if (tool !== 'select') setSelectedIdx(null)
  }, [tool])

  // Load board list on mount
  useEffect(() => {
    listWhiteboards(courseCode)
      .then(setBoards)
      .catch(() => {})
  }, [courseCode])

  // --------------- pointer helpers ---------------
  function getPos(e: ReactPointerEvent<HTMLCanvasElement> | ReactMouseEvent<HTMLCanvasElement>): [number, number] {
    const canvas = canvasRef.current!
    const rect = canvas.getBoundingClientRect()
    return [e.clientX - rect.left, e.clientY - rect.top]
  }

  function onPointerDown(e: ReactPointerEvent<HTMLCanvasElement>) {
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
  }

  function onPointerMove(e: ReactPointerEvent<HTMLCanvasElement>) {
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
      // getCoalescedEvents gives all sub-frame pointer positions the browser recorded,
      // so we erase only where the mouse actually was without artificial interpolation.
      const coalesced = e.nativeEvent.getCoalescedEvents?.() ?? []
      const pts: [number, number][] = coalesced.length > 0
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
      setDraft({ type: 'circle', color, width: strokeWidth, cx: ox, cy: oy, rx: (x - ox) / 2, ry: (y - oy) / 2 })
    } else if (tool === 'triangle') {
      const mx = (ox + x) / 2
      setDraft({ type: 'triangle', color, width: strokeWidth, x1: mx, y1: oy, x2: x, y2: y, x3: ox, y3: y })
    }
  }

  function onPointerUp() {
    if (!isDrawing) return
    setIsDrawing(false)
    setDragInfo(null)
    if (draft) {
      setElements((prev) => [...prev, draft])
      setDraft(null)
    }
  }

  // --------------- actions ---------------
  function clearCanvas() {
    setElements([])
    setDraft(null)
    setCurrentBoard(null)
  }

  function exportPng() {
    const canvas = canvasRef.current
    if (!canvas) return
    const a = document.createElement('a')
    a.href = canvas.toDataURL('image/png')
    a.download = `${currentBoard?.title ?? 'whiteboard'}.png`
    a.click()
  }

  async function handleSave(title: string) {
    setSaving(true)
    setShowSave(false)
    try {
      let saved: WhiteboardRow
      if (currentBoard) {
        saved = await updateWhiteboard(courseCode, currentBoard.id, title, elements as unknown[])
        setBoards((prev) => prev.map((b) => (b.id === saved.id ? saved : b)))
      } else {
        saved = await createWhiteboard(courseCode, title, elements as unknown[])
        setBoards((prev) => [saved, ...prev])
      }
      setCurrentBoard(saved)
      toastSaveOk('Whiteboard saved')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not save')
    } finally {
      setSaving(false)
    }
  }

  function handleLoad(b: WhiteboardRow) {
    setShowLoad(false)
    setCurrentBoard(b)
    const data = Array.isArray(b.canvasData) ? (b.canvasData as DrawEl[]) : []
    setElements(data)
    setDraft(null)
  }

  async function handleDelete(id: string) {
    try {
      await deleteWhiteboard(courseCode, id)
      setBoards((prev) => prev.filter((b) => b.id !== id))
      if (currentBoard?.id === id) {
        setCurrentBoard(null)
        setElements([])
      }
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not delete')
    }
  }

  // --------------- toolbar items ---------------
  const tools: { id: Tool; icon: React.ReactNode; label: string }[] = [
    { id: 'select', icon: <MousePointer2 className="h-5 w-5" />, label: 'Select' },
    { id: 'pen', icon: <Pencil className="h-5 w-5" />, label: 'Pen' },
    { id: 'line', icon: <Minus className="h-5 w-5" />, label: 'Line' },
    { id: 'rect', icon: <Square className="h-5 w-5" />, label: 'Rectangle' },
    { id: 'circle', icon: <Circle className="h-5 w-5" />, label: 'Circle' },
    { id: 'triangle', icon: <Triangle className="h-5 w-5" />, label: 'Triangle' },
  ]

  const cursor =
    tool === 'select' ? (dragInfo ? 'cursor-grabbing' : 'cursor-grab') :
    tool === 'eraser' ? 'cursor-none' :
    'cursor-crosshair'

  return (
    <div className="flex h-[calc(100vh-4rem)] overflow-hidden">
      {/* Left toolbar */}
      <div className="flex w-14 flex-col items-center gap-1 border-r border-slate-200 bg-white py-3 dark:border-neutral-800 dark:bg-neutral-950">
        {/* Tool buttons — collapsed to selected, popover on hover/click */}
        {(() => {
          const activeTool = tools.find((t) => t.id === tool) ?? tools[0]
          return (
            <PopoverGroup
              trigger={
                <button
                  type="button"
                  title={activeTool.label}
                  className="flex h-9 w-9 items-center justify-center rounded-lg transition-colors bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300"
                >
                  {activeTool.icon}
                </button>
              }
            >
              <div className="flex flex-col gap-1">
                {tools.map((t) => (
                  <button
                    key={t.id}
                    type="button"
                    title={t.label}
                    onClick={() => setTool(t.id)}
                    className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                      tool === t.id
                        ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300'
                        : 'text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200'
                    }`}
                  >
                    {t.icon}
                  </button>
                ))}
              </div>
            </PopoverGroup>
          )
        })()}

        {/* Eraser — collapsed to icon, popover shows 3 sizes */}
        <PopoverGroup
          trigger={
            <button
              type="button"
              title="Eraser"
              onClick={() => setTool('eraser')}
              className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                tool === 'eraser'
                  ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300'
                  : 'text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200'
              }`}
            >
              <Eraser className="h-5 w-5" />
            </button>
          }
        >
          <div className="flex flex-col gap-1">
            {ERASER_SIZES.map((s) => (
              <button
                key={s}
                type="button"
                title={`Eraser ${s}px`}
                onClick={() => { setEraserSize(s); setTool('eraser') }}
                className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                  eraserSize === s && tool === 'eraser'
                    ? 'bg-indigo-100 dark:bg-indigo-950'
                    : 'hover:bg-slate-100 dark:hover:bg-neutral-800'
                }`}
              >
                <span
                  className="rounded-full border border-slate-400 bg-white dark:border-neutral-500 dark:bg-neutral-800"
                  style={{ width: s, height: s }}
                />
              </button>
            ))}
          </div>
        </PopoverGroup>

        <div className="my-1 h-px w-8 bg-slate-200 dark:bg-neutral-800" />

        {/* Stroke width — collapsed to selected, popover on hover/click */}
        <PopoverGroup
          trigger={
            <button
              type="button"
              title={`Stroke ${strokeWidth}px`}
              className="flex h-9 w-9 items-center justify-center rounded-lg transition-colors bg-indigo-100 dark:bg-indigo-950"
            >
              <span
                className="rounded-full bg-slate-700 dark:bg-neutral-300"
                style={{ width: strokeWidth * 2, height: strokeWidth * 2 }}
              />
            </button>
          }
        >
          <div className="flex flex-col gap-1">
            {STROKE_WIDTHS.map((w) => (
              <button
                key={w}
                type="button"
                title={`Stroke ${w}px`}
                onClick={() => setStrokeWidth(w)}
                className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                  strokeWidth === w
                    ? 'bg-indigo-100 dark:bg-indigo-950'
                    : 'hover:bg-slate-100 dark:hover:bg-neutral-800'
                }`}
              >
                <span
                  className="rounded-full bg-slate-700 dark:bg-neutral-300"
                  style={{ width: w * 2, height: w * 2 }}
                />
              </button>
            ))}
          </div>
        </PopoverGroup>

        <div className="my-1 h-px w-8 bg-slate-200 dark:bg-neutral-800" />

        {/* Color — collapsed to selected, popover on hover/click */}
        <PopoverGroup
          trigger={
            <button
              type="button"
              title={color}
              className="flex h-7 w-7 items-center justify-center rounded-full ring-2 ring-indigo-500 ring-offset-1 scale-110 transition-transform"
              style={{ backgroundColor: color, border: color === '#ffffff' ? '1px solid #e2e8f0' : undefined }}
            />
          }
        >
          <div
            className="grid gap-2 p-1"
            style={{ gridTemplateColumns: 'repeat(3, 1.75rem)' }}
          >
            {COLORS.map((c) => (
              <button
                key={c}
                type="button"
                title={c}
                onClick={() => setColor(c)}
                className={`h-7 w-7 rounded-full transition-transform ${
                  color === c ? 'ring-2 ring-indigo-500 ring-offset-1 scale-110' : 'hover:scale-110'
                }`}
                style={{ backgroundColor: c, border: c === '#ffffff' ? '1px solid #e2e8f0' : undefined }}
              />
            ))}
          </div>
        </PopoverGroup>

        <div className="my-1 h-px w-8 bg-slate-200 dark:bg-neutral-800" />

        {/* Actions */}
        <button
          type="button"
          title="Clear canvas"
          onClick={clearCanvas}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 hover:text-rose-500 dark:text-neutral-400 dark:hover:bg-neutral-800"
        >
          <Trash2 className="h-5 w-5" />
        </button>
        <button
          type="button"
          title="Export PNG"
          onClick={exportPng}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800"
        >
          <Download className="h-5 w-5" />
        </button>
        <button
          type="button"
          title="Load whiteboard"
          onClick={() => setShowLoad(true)}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800"
        >
          <FolderOpen className="h-5 w-5" />
        </button>
        <button
          type="button"
          title="Save whiteboard"
          disabled={saving}
          onClick={() => setShowSave(true)}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 hover:text-indigo-600 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
        >
          <Save className="h-5 w-5" />
        </button>
      </div>

      {/* Canvas area */}
      <div ref={containerRef} className="relative flex-1 overflow-hidden">
        {currentBoard && (
          <div className="pointer-events-none absolute left-4 top-3 z-10 text-xs text-slate-400 dark:text-neutral-500">
            {currentBoard.title}
          </div>
        )}
        <canvas
          ref={canvasRef}
          className={`touch-none ${cursor}`}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={onPointerUp}
          onPointerLeave={() => setEraserCursorPos(null)}
        />
        {tool === 'eraser' && eraserCursorPos && (
          <div
            className="pointer-events-none absolute rounded-full border border-slate-500 bg-white/15 dark:border-slate-400"
            style={{
              left: eraserCursorPos[0] - eraserSize,
              top: eraserCursorPos[1] - eraserSize,
              width: eraserSize * 2,
              height: eraserSize * 2,
            }}
          />
        )}
      </div>

      {/* Dialogs */}
      {showSave && (
        <SaveDialog
          initialTitle={currentBoard?.title ?? ''}
          onSave={(t) => void handleSave(t)}
          onClose={() => setShowSave(false)}
        />
      )}
      {showLoad && (
        <LoadPanel
          boards={boards}
          onLoad={handleLoad}
          onDelete={(id) => void handleDelete(id)}
          onClose={() => setShowLoad(false)}
        />
      )}
    </div>
  )
}

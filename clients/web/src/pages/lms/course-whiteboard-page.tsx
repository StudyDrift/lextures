import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type MouseEvent as ReactMouseEvent,
  type PointerEvent as ReactPointerEvent,
} from 'react'
import { useParams } from 'react-router-dom'
import {
  Circle,
  Download,
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

type Tool = 'select' | 'pen' | 'line' | 'rect' | 'circle' | 'triangle'

const COLORS = ['#1e293b', '#ef4444', '#f97316', '#eab308', '#22c55e', '#3b82f6', '#a855f7', '#ec4899', '#ffffff']
const STROKE_WIDTHS = [2, 4, 8]
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

function redraw(
  ctx: CanvasRenderingContext2D,
  w: number,
  h: number,
  elements: DrawEl[],
  draft: DrawEl | null,
  dark: boolean,
) {
  drawGrid(ctx, w, h, dark)
  for (const el of elements) drawElement(ctx, el)
  if (draft) drawElement(ctx, draft)
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
  const [strokeWidth, setStrokeWidth] = useState(STROKE_WIDTHS[0])
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
    if (ctx) redraw(ctx, width, height, elements, draft, isDark)
  }, [elements, draft, isDark])

  useEffect(() => {
    resizeCanvas()
    const obs = new ResizeObserver(() => resizeCanvas())
    if (containerRef.current) obs.observe(containerRef.current)
    return () => obs.disconnect()
  }, [resizeCanvas])

  // Redraw whenever elements/draft change
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    redraw(ctx, canvas.width, canvas.height, elements, draft, isDark)
  }, [elements, draft, isDark])

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
    if (tool === 'select') return
    const [x, y] = getPos(e)
    setOrigin([x, y])
    setIsDrawing(true)
    ;(e.target as HTMLCanvasElement).setPointerCapture(e.pointerId)

    if (tool === 'pen') {
      setDraft({ type: 'stroke', color, width: strokeWidth, pts: [[x, y]] })
    }
  }

  function onPointerMove(e: ReactPointerEvent<HTMLCanvasElement>) {
    if (!isDrawing) return
    const [x, y] = getPos(e)
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
  }

  function onPointerUp() {
    if (!isDrawing) return
    setIsDrawing(false)
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

  const cursor = tool === 'select' ? 'cursor-default' : tool === 'pen' ? 'cursor-crosshair' : 'cursor-crosshair'

  return (
    <div className="flex h-[calc(100vh-4rem)] overflow-hidden">
      {/* Left toolbar */}
      <div className="flex w-14 flex-col items-center gap-1 border-r border-slate-200 bg-white py-3 dark:border-neutral-800 dark:bg-neutral-950">
        {/* Tool buttons */}
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

        <div className="my-1 h-px w-8 bg-slate-200 dark:bg-neutral-800" />

        {/* Stroke widths */}
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

        <div className="my-1 h-px w-8 bg-slate-200 dark:bg-neutral-800" />

        {/* Colors */}
        {COLORS.map((c) => (
          <button
            key={c}
            type="button"
            title={c}
            onClick={() => setColor(c)}
            className={`flex h-7 w-7 items-center justify-center rounded-full transition-transform ${
              color === c ? 'ring-2 ring-indigo-500 ring-offset-1 scale-110' : 'hover:scale-110'
            }`}
            style={{ backgroundColor: c, border: c === '#ffffff' ? '1px solid #e2e8f0' : undefined }}
          />
        ))}

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
        />
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

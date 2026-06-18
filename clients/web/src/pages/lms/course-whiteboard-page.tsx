import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Trash2 } from 'lucide-react'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { useWhiteboardCanvas } from '../../components/whiteboard/use-whiteboard-canvas'
import { WhiteboardToolbar } from '../../components/whiteboard/whiteboard-toolbar'
import {
  createWhiteboard,
  deleteWhiteboard,
  listWhiteboards,
  updateWhiteboard,
  type WhiteboardRow,
} from '../../lib/courses-api'
import type { DrawEl } from '../../lib/whiteboard/types'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

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

export default function CourseWhiteboardPage() {
  const { courseCode = '' } = useParams<{ courseCode: string }>()

  const [elements, setElements] = useState<DrawEl[]>([])
  const [boards, setBoards] = useState<WhiteboardRow[]>([])
  const [currentBoard, setCurrentBoard] = useState<WhiteboardRow | null>(null)
  const [showSave, setShowSave] = useState(false)
  const [showLoad, setShowLoad] = useState(false)
  const [saving, setSaving] = useState(false)

  const onElementsChange = useCallback((next: DrawEl[]) => {
    setElements(next)
  }, [])

  const wb = useWhiteboardCanvas({ elements, onElementsChange })

  useEffect(() => {
    listWhiteboards(courseCode)
      .then(setBoards)
      .catch(() => {})
  }, [courseCode])

  useCoursePageTitle(currentBoard?.title ?? null)

  function clearCanvas() {
    wb.clearCanvas()
    setCurrentBoard(null)
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

  return (
    <div className="flex h-[calc(100vh-4rem)] overflow-hidden">
      <WhiteboardToolbar
        tool={wb.tool}
        onToolChange={wb.setTool}
        color={wb.color}
        onColorChange={wb.setColor}
        strokeWidth={wb.strokeWidth}
        onStrokeWidthChange={wb.setStrokeWidth}
        eraserSize={wb.eraserSize}
        onEraserSizeChange={wb.setEraserSize}
        onClear={clearCanvas}
        onExportPng={() => wb.exportPng(currentBoard?.title ?? 'whiteboard')}
        onLoad={() => setShowLoad(true)}
        onSave={() => setShowSave(true)}
        saving={saving}
      />

      <div ref={wb.containerRef} className="relative flex-1 overflow-hidden">
        {currentBoard ? (
          <div className="pointer-events-none absolute left-4 top-3 z-10 text-xs text-slate-400 dark:text-neutral-500">
            {currentBoard.title}
          </div>
        ) : null}
        <canvas
          ref={wb.canvasRef}
          className={`touch-none ${wb.cursor}`}
          onPointerDown={wb.onPointerDown}
          onPointerMove={wb.onPointerMove}
          onPointerUp={wb.onPointerUp}
          onPointerLeave={() => wb.setEraserCursorPos(null)}
        />
        {wb.tool === 'eraser' && wb.eraserCursorPos ? (
          <div
            className="pointer-events-none absolute rounded-full border border-slate-500 bg-white/15 dark:border-slate-400"
            style={{
              left: wb.eraserCursorPos[0] - wb.eraserSize,
              top: wb.eraserCursorPos[1] - wb.eraserSize,
              width: wb.eraserSize * 2,
              height: wb.eraserSize * 2,
            }}
          />
        ) : null}
      </div>

      {showSave ? (
        <SaveDialog
          initialTitle={currentBoard?.title ?? ''}
          onSave={(t) => void handleSave(t)}
          onClose={() => setShowSave(false)}
        />
      ) : null}
      {showLoad ? (
        <LoadPanel
          boards={boards}
          onLoad={handleLoad}
          onDelete={(id) => void handleDelete(id)}
          onClose={() => setShowLoad(false)}
        />
      ) : null}
    </div>
  )
}

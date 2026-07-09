import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  pointerWithin,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { CSS } from '@dnd-kit/utilities'
import { ChevronLeft, ChevronRight, GripVertical } from 'lucide-react'
import type { CoursePublic } from '../../lib/courses-api'
import type { KanbanColumnLabels } from '../../lib/course-catalog-settings-api'
import { formatRelativeCompact } from '../../lib/format-datetime'
import { CourseCatalogStatusPill } from '../../components/ui/status-vocabulary'
import { CourseCatalogNicknameEditor } from './course-catalog-nickname-editor'
import { CourseCatalogPinButton } from './course-catalog-pin-button'
import {
  buildKanbanBoardState,
  courseCatalogStatusLabel,
  isUserCatalogHidden,
  resolveKanbanColumn,
} from './course-catalog-status'
import { isKanbanColumnId, KANBAN_COLUMN_IDS, type KanbanColumnId } from '../../lib/course-catalog-types'

const KANBAN_COLUMN_HINTS: Record<KanbanColumnId, string> = {
  todo: 'Draft and upcoming courses',
  'in-progress': 'Active courses',
  done: 'Ended courses',
  hidden: 'Hidden from your catalog or outside visibility window',
}

function columnDropId(columnId: KanbanColumnId): string {
  return `column:${columnId}`
}

function parseColumnDropId(id: string): KanbanColumnId | null {
  if (!id.startsWith('column:')) return null
  const col = id.slice('column:'.length)
  return isKanbanColumnId(col) ? col : null
}

function boardToColumnIds(board: Record<KanbanColumnId, CoursePublic[]>): Record<KanbanColumnId, string[]> {
  return {
    todo: board.todo.map((c) => c.id),
    'in-progress': board['in-progress'].map((c) => c.id),
    done: board.done.map((c) => c.id),
    hidden: board.hidden.filter((c) => isUserCatalogHidden(c)).map((c) => c.id),
  }
}

function computeNextBoard(
  board: Record<KanbanColumnId, CoursePublic[]>,
  courseId: string,
  targetColumn: KanbanColumnId,
  targetIndex?: number,
): Record<KanbanColumnId, CoursePublic[]> {
  const next: Record<KanbanColumnId, CoursePublic[]> = {
    todo: [...board.todo],
    'in-progress': [...board['in-progress']],
    done: [...board.done],
    hidden: [...board.hidden],
  }
  let moving: CoursePublic | undefined
  for (const col of KANBAN_COLUMN_IDS) {
    const idx = next[col].findIndex((c) => c.id === courseId)
    if (idx >= 0) {
      moving = next[col][idx]
      next[col].splice(idx, 1)
      break
    }
  }
  if (!moving) return board
  const insertAt =
    targetIndex == null || targetIndex < 0 || targetIndex > next[targetColumn].length
      ? next[targetColumn].length
      : targetIndex
  next[targetColumn].splice(insertAt, 0, moving)
  return next
}

function formatCourseTermLabel(course: CoursePublic): string {
  return course.term?.name?.trim() || '—'
}

function KanbanDraggableCard({
  course,
  onNicknameChange,
  onPinnedChange,
  overlay = false,
}: {
  course: CoursePublic
  onNicknameChange: (courseId: string, nickname: string | null) => void
  onPinnedChange: (courseId: string, pinned: boolean) => void
  overlay?: boolean
}) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: course.id,
    data: { courseId: course.id },
  })
  const courseHref = `/courses/${encodeURIComponent(course.courseCode)}`
  const lifecycleLabel = courseCatalogStatusLabel(course)
  const showLifecyclePill = resolveKanbanColumn(course) === 'hidden'

  return (
    <article
      ref={overlay ? undefined : setNodeRef}
      style={overlay ? undefined : { transform: CSS.Translate.toString(transform) }}
      className={[
        'rounded-lg border border-slate-200 bg-white shadow-sm shadow-slate-900/5 dark:border-neutral-600 dark:bg-neutral-900',
        isDragging && !overlay ? 'opacity-40' : '',
        overlay ? 'cursor-grabbing shadow-md ring-2 ring-indigo-400/40' : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <div className="flex items-start gap-1 p-3">
        <button
          type="button"
          className="mt-0.5 inline-flex h-6 w-5 shrink-0 cursor-grab items-center justify-center rounded text-slate-400 hover:bg-slate-100 hover:text-slate-600 active:cursor-grabbing dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
          aria-label={`Drag ${course.title}`}
          {...listeners}
          {...attributes}
        >
          <GripVertical className="h-4 w-4" aria-hidden />
        </button>
        <div className="min-w-0 flex-1">
          <div className="flex items-start gap-1">
            <div className="min-w-0 flex-1">
              <CourseCatalogNicknameEditor
                course={course}
                titleClassName="text-sm font-semibold leading-snug text-slate-900 dark:text-neutral-100"
                onNicknameChange={onNicknameChange}
              />
            </div>
            <CourseCatalogPinButton
              course={course}
              variant="inline"
              className="h-7 w-7"
              onPinnedChange={onPinnedChange}
            />
          </div>
          <Link
            to={courseHref}
            className="mt-2 inline-block text-xs font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-300 dark:hover:text-indigo-200"
          >
            Open course
          </Link>
          {showLifecyclePill ? (
            <div className="mt-2">
              <CourseCatalogStatusPill label={lifecycleLabel} />
            </div>
          ) : null}
          <dl className="mt-2 space-y-1 text-xs text-slate-500 dark:text-neutral-400">
            <div className="flex justify-between gap-2">
              <dt className="sr-only">Term</dt>
              <dd className="truncate">{formatCourseTermLabel(course)}</dd>
            </div>
            <div className="flex justify-between gap-2">
              <dt className="sr-only">Last edited</dt>
              <dd className="whitespace-nowrap">{formatRelativeCompact(course.updatedAt)}</dd>
            </div>
          </dl>
        </div>
      </div>
    </article>
  )
}

function KanbanColumnDropZone({
  columnId,
  children,
}: {
  columnId: KanbanColumnId
  children: ReactNode
}) {
  const { setNodeRef, isOver } = useDroppable({ id: columnDropId(columnId) })
  return (
    <div
      ref={setNodeRef}
      className={[
        'flex min-h-[6rem] flex-col gap-2 rounded-lg transition-colors',
        isOver ? 'bg-indigo-50/80 ring-1 ring-indigo-300/60 dark:bg-indigo-950/20 dark:ring-indigo-500/30' : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {children}
    </div>
  )
}

type KanbanColumnProps = {
  columnId: KanbanColumnId
  title: string
  hint: string
  courses: CoursePublic[]
  collapsed?: boolean
  onToggleCollapsed?: () => void
  onTitleChange: (columnId: KanbanColumnId, title: string) => void
  onNicknameChange: (courseId: string, nickname: string | null) => void
  onPinnedChange: (courseId: string, pinned: boolean) => void
}

function KanbanColumn({
  columnId,
  title,
  hint,
  courses,
  collapsed = false,
  onToggleCollapsed,
  onTitleChange,
  onNicknameChange,
  onPinnedChange,
}: KanbanColumnProps) {
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState(title)

  useEffect(() => {
    if (!editingTitle) setTitleDraft(title)
  }, [title, editingTitle])

  if (collapsed && onToggleCollapsed) {
    return (
      <div className="flex w-11 shrink-0 flex-col self-stretch">
        <button
          type="button"
          onClick={onToggleCollapsed}
          className="flex h-full min-h-[12rem] flex-col items-center justify-start gap-3 rounded-xl border border-slate-200 bg-slate-100/90 px-1.5 py-3 text-slate-600 transition-[background-color,color,border-color] hover:border-slate-300 hover:bg-slate-100 dark:border-neutral-700 dark:bg-neutral-800/90 dark:text-neutral-300 dark:hover:border-neutral-600 dark:hover:bg-neutral-800"
          aria-label={`Expand ${title} column (${courses.length} courses). ${hint}`}
          title={`${title} (${courses.length})`}
        >
          <ChevronLeft className="h-4 w-4 shrink-0" aria-hidden />
          <span
            className="text-xs font-semibold uppercase tracking-wide [writing-mode:vertical-rl] rotate-180"
            aria-hidden
          >
            {title}
          </span>
          <span className="rounded-full bg-slate-200 px-1.5 py-0.5 text-[10px] font-semibold text-slate-700 dark:bg-neutral-700 dark:text-neutral-200">
            {courses.length}
          </span>
        </button>
      </div>
    )
  }

  return (
    <section
      className="flex w-72 shrink-0 flex-col rounded-xl border border-slate-200 bg-slate-100/90 dark:border-neutral-700 dark:bg-neutral-800/90"
      aria-label={`${title} (${courses.length})`}
    >
      <header className="flex items-start justify-between gap-2 border-b border-slate-200/80 px-3 py-3 dark:border-neutral-700">
        <div className="min-w-0 flex-1">
          {editingTitle ? (
            <input
              value={titleDraft}
              maxLength={80}
              onChange={(e) => setTitleDraft(e.target.value)}
              onPointerDown={(e) => e.stopPropagation()}
              onKeyDown={(e) => {
                e.stopPropagation()
                if (e.key === 'Enter') {
                  e.currentTarget.blur()
                }
                if (e.key === 'Escape') {
                  setTitleDraft(title)
                  setEditingTitle(false)
                }
              }}
              onBlur={() => {
                setEditingTitle(false)
                onTitleChange(columnId, titleDraft.trim() || title)
              }}
              className="w-full rounded-md border border-slate-200 bg-white px-2 py-1 text-sm font-semibold text-slate-900 outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
              aria-label={`Rename ${title} column`}
            />
          ) : (
            <button
              type="button"
              onClick={() => setEditingTitle(true)}
              className="text-start text-sm font-semibold text-slate-900 hover:text-indigo-600 dark:text-neutral-100 dark:hover:text-indigo-300"
              title="Rename column"
            >
              {title}
            </button>
          )}
          <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">{hint}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <span className="rounded-full bg-slate-200 px-2 py-0.5 text-xs font-semibold text-slate-700 dark:bg-neutral-700 dark:text-neutral-200">
            {courses.length}
          </span>
          {onToggleCollapsed ? (
            <button
              type="button"
              onClick={onToggleCollapsed}
              className="inline-flex h-7 w-7 items-center justify-center rounded-md text-slate-500 transition-[background-color,color,border-color] hover:bg-slate-200/80 hover:text-slate-800 dark:text-neutral-400 dark:hover:bg-neutral-700 dark:hover:text-neutral-100"
              aria-label={`Collapse ${title} column`}
              title={`Collapse ${title}`}
            >
              <ChevronRight className="h-4 w-4" aria-hidden />
            </button>
          ) : null}
        </div>
      </header>
      <div className="flex-1 overflow-y-auto p-3">
        <KanbanColumnDropZone columnId={columnId}>
          {courses.length === 0 ? (
            <p className="rounded-lg border border-dashed border-slate-300 px-3 py-6 text-center text-xs text-slate-500 dark:border-neutral-600 dark:text-neutral-400">
              Drop courses here
            </p>
          ) : (
            courses.map((course) => (
              <KanbanDraggableCard
                key={course.id}
                course={course}
                onNicknameChange={onNicknameChange}
                onPinnedChange={onPinnedChange}
              />
            ))
          )}
        </KanbanColumnDropZone>
      </div>
    </section>
  )
}

type Props = {
  courses: CoursePublic[]
  columnLabels: KanbanColumnLabels
  hiddenColumnExpanded: boolean
  onHiddenColumnExpandedChange: (expanded: boolean) => void
  onColumnLabelsChange: (labels: KanbanColumnLabels) => void
  onNicknameChange: (courseId: string, nickname: string | null) => void
  onPinnedChange: (courseId: string, pinned: boolean) => void
  onHiddenChange: (courseId: string, hidden: boolean) => void
  onBoardChange: (columns: Record<KanbanColumnId, string[]>) => Promise<void>
}

export function CourseCatalogKanbanBoard({
  courses,
  columnLabels,
  hiddenColumnExpanded,
  onHiddenColumnExpandedChange,
  onColumnLabelsChange,
  onNicknameChange,
  onPinnedChange,
  onHiddenChange,
  onBoardChange,
}: Props) {
  const courseById = useMemo(() => new Map(courses.map((c) => [c.id, c])), [courses])
  const [board, setBoard] = useState<Record<KanbanColumnId, CoursePublic[]>>(() => buildKanbanBoardState(courses))
  const [activeCourseId, setActiveCourseId] = useState<string | null>(null)
  const [boardError, setBoardError] = useState<string | null>(null)
  const [savingBoard, setSavingBoard] = useState(false)

  useEffect(() => {
    setBoard(buildKanbanBoardState(courses))
  }, [courses])

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }))

  const persistBoard = useCallback(
    async (nextBoard: Record<KanbanColumnId, CoursePublic[]>) => {
      setBoardError(null)
      setSavingBoard(true)
      try {
        await onBoardChange(boardToColumnIds(nextBoard))
      } catch (e: unknown) {
        setBoardError(e instanceof Error ? e.message : 'Could not save kanban board.')
        setBoard(buildKanbanBoardState(courses))
      } finally {
        setSavingBoard(false)
      }
    },
    [courses, onBoardChange],
  )

  const onDragStart = useCallback((event: DragStartEvent) => {
    setActiveCourseId(String(event.active.id))
  }, [])

  const onDragEnd = useCallback(
    (event: DragEndEvent) => {
      setActiveCourseId(null)
      const courseId = String(event.active.id)
      const over = event.over
      if (!over) return

      let targetColumn = parseColumnDropId(String(over.id))
      let targetIndex: number | undefined
      if (!targetColumn) {
        for (const col of KANBAN_COLUMN_IDS) {
          const idx = board[col].findIndex((c) => c.id === String(over.id))
          if (idx >= 0) {
            targetColumn = col
            targetIndex = idx
            break
          }
        }
      }

      if (!targetColumn) return
      const previousColumn = (() => {
        for (const col of KANBAN_COLUMN_IDS) {
          if (board[col].some((c) => c.id === courseId)) return col
        }
        return null
      })()
      const nextBoard = computeNextBoard(board, courseId, targetColumn, targetIndex)
      setBoard(nextBoard)
      const moving = courseById.get(courseId)
      if (moving) {
        if (targetColumn === 'hidden' && !isUserCatalogHidden(moving)) {
          onHiddenChange(courseId, true)
        } else if (previousColumn === 'hidden' && targetColumn !== 'hidden' && isUserCatalogHidden(moving)) {
          onHiddenChange(courseId, false)
        }
      }
      void persistBoard(nextBoard)
    },
    [board, courseById, onHiddenChange, persistBoard],
  )

  const activeCourse = activeCourseId ? courseById.get(activeCourseId) : undefined

  return (
    <div className="mt-8">
      {boardError ? (
        <p className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/50 dark:text-rose-200">
          {boardError}
        </p>
      ) : null}
      {savingBoard ? (
        <p className="mb-4 text-xs text-slate-500 dark:text-neutral-400" role="status">
          Saving board…
        </p>
      ) : null}
      <DndContext
        sensors={sensors}
        collisionDetection={pointerWithin}
        onDragStart={onDragStart}
        onDragEnd={onDragEnd}
        onDragCancel={() => setActiveCourseId(null)}
      >
        <div className="overflow-x-auto pb-2">
          <div className="flex min-h-[20rem] items-stretch gap-4">
            {KANBAN_COLUMN_IDS.map((columnId) => (
              <KanbanColumn
                key={columnId}
                columnId={columnId}
                title={columnLabels[columnId]}
                hint={KANBAN_COLUMN_HINTS[columnId]}
                courses={board[columnId]}
                collapsed={columnId === 'hidden' && !hiddenColumnExpanded}
                onToggleCollapsed={
                  columnId === 'hidden' ? () => onHiddenColumnExpandedChange(!hiddenColumnExpanded) : undefined
                }
                onTitleChange={(id, nextTitle) => {
                  onColumnLabelsChange({ ...columnLabels, [id]: nextTitle })
                }}
                onNicknameChange={onNicknameChange}
                onPinnedChange={onPinnedChange}
              />
            ))}
          </div>
        </div>
        <DragOverlay dropAnimation={null}>
          {activeCourse ? (
            <KanbanDraggableCard
              course={activeCourse}
              onNicknameChange={onNicknameChange}
              onPinnedChange={onPinnedChange}
              overlay
            />
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  )
}

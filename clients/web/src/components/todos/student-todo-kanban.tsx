import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  TouchSensor,
  closestCorners,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { CSS } from '@dnd-kit/utilities'
import { CalendarDays, CheckCircle2 } from 'lucide-react'
import { formatDueShort } from '../../lib/course-calendar-utils'
import {
  boardToColumnKeys,
  buildStudentTodoBoard,
  computeNextStudentTodoBoard,
} from '../../lib/student-todo-utils'
import { saveStudentTodoBoard } from '../../lib/student-todo-board-api'
import {
  filterItemsForWeeks,
  normalizeWeekOffsets,
  openCountLabelForWeeks,
  weekLabelForItem,
} from '../../lib/student-todo-week'
import { relativeWeekStartKey } from '../../lib/use-relative-week-now'
import {
  STUDENT_TODO_COLUMN_IDS,
  STUDENT_TODO_COLUMN_LABELS,
  STUDENT_TODO_COLUMN_SHORT_LABELS,
  STUDENT_TODO_WEEKDAY_COLUMN_IDS,
  type StudentTodoColumnId,
  type StudentTodoItem,
  type StudentTodoPlacement,
  type StudentTodoWeekdayColumnId,
} from '../../lib/student-todo-types'

const BOARD_SHELL_SHADOW =
  'shadow-[0px_0px_0px_1px_rgba(0,0,0,0.06),0px_1px_2px_-1px_rgba(0,0,0,0.06),0px_2px_4px_0px_rgba(0,0,0,0.04)] dark:shadow-[0_0_0_1px_rgba(255,255,255,0.08)]'

const CARD_SHADOW =
  'shadow-[0px_0px_0px_1px_rgba(0,0,0,0.05),0px_1px_2px_0px_rgba(0,0,0,0.04)] dark:shadow-[0_0_0_1px_rgba(255,255,255,0.07)]'

const CARD_SHADOW_HOVER =
  'hover:shadow-[0px_0px_0px_1px_rgba(0,0,0,0.07),0px_2px_6px_0px_rgba(0,0,0,0.06)] dark:hover:shadow-[0_0_0_1px_rgba(255,255,255,0.1)]'

const WEEKDAY_INDEX_TO_COLUMN: Record<number, StudentTodoWeekdayColumnId> = {
  0: 'sun',
  1: 'mon',
  2: 'tue',
  3: 'wed',
  4: 'thu',
  5: 'fri',
  6: 'sat',
}

function columnDropId(columnId: StudentTodoColumnId): string {
  return `column:${columnId}`
}

function parseColumnDropId(id: string): StudentTodoColumnId | null {
  if (!id.startsWith('column:')) return null
  const col = id.slice('column:'.length)
  return (STUDENT_TODO_COLUMN_IDS as readonly string[]).includes(col) ? (col as StudentTodoColumnId) : null
}

function todayColumnId(now = new Date()): StudentTodoWeekdayColumnId {
  return WEEKDAY_INDEX_TO_COLUMN[now.getDay()] ?? 'mon'
}

function boardToPlacements(
  board: Record<StudentTodoColumnId, StudentTodoItem[]>,
): StudentTodoPlacement[] {
  const out: StudentTodoPlacement[] = []
  for (const columnId of STUDENT_TODO_COLUMN_IDS) {
    board[columnId].forEach((item, sortOrder) => {
      out.push({ itemKey: item.key, columnId, sortOrder })
    })
  }
  return out
}

function resolveDropTarget(
  board: Record<StudentTodoColumnId, StudentTodoItem[]>,
  overId: string,
): { columnId: StudentTodoColumnId; index?: number } | null {
  const columnId = parseColumnDropId(overId)
  if (columnId) return { columnId }

  for (const col of STUDENT_TODO_COLUMN_IDS) {
    const idx = board[col].findIndex((item) => item.key === overId)
    if (idx >= 0) return { columnId: col, index: idx }
  }
  return null
}

type TodoDraggableCardProps = {
  item: StudentTodoItem
  overlay?: boolean
  isDragActive?: boolean
  weekBadge?: string | null
}

function TodoDraggableCard({ item, overlay = false, isDragActive = false, weekBadge }: TodoDraggableCardProps) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: item.key,
    data: { itemKey: item.key, type: 'todo-item' },
  })
  const suppressNavRef = useRef(false)
  const wasDraggingRef = useRef(false)

  useEffect(() => {
    if (wasDraggingRef.current && !isDragging) {
      suppressNavRef.current = true
    }
    wasDraggingRef.current = isDragging
  }, [isDragging])

  const cardBody = (
    <>
      <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-900 group-hover:text-indigo-700 dark:text-neutral-100 dark:group-hover:text-indigo-300">
        {item.title}
      </p>
      <div className="mt-1 flex min-w-0 items-center gap-1.5">
        <p className="min-w-0 truncate text-[11px] text-slate-500 dark:text-neutral-400">{item.courseTitle}</p>
        {weekBadge ? (
          <span className="shrink-0 rounded bg-slate-100 px-1 py-px text-[10px] font-medium text-slate-500 dark:bg-neutral-800 dark:text-neutral-400">
            {weekBadge}
          </span>
        ) : null}
      </div>
      {item.dueAt ? (
        <p className="mt-1.5 flex items-center gap-1 text-[11px] text-slate-400 dark:text-neutral-500">
          <CalendarDays className="h-3 w-3 shrink-0 opacity-70" aria-hidden />
          <span className="truncate">{formatDueShort(item.dueAt)}</span>
        </p>
      ) : item.kind === 'notebook_task' ? (
        <p className="mt-1.5 text-[11px] text-slate-400 dark:text-neutral-500">Notebook task</p>
      ) : null}
    </>
  )

  return (
    <article
      ref={overlay ? undefined : setNodeRef}
      style={overlay ? undefined : { transform: CSS.Translate.toString(transform) }}
      className={[
        'group touch-manipulation rounded-lg bg-white transition-[box-shadow,opacity,transform] duration-150 ease-out dark:bg-neutral-900',
        CARD_SHADOW,
        isDragging && !overlay ? 'opacity-30' : '',
        overlay
          ? 'cursor-grabbing shadow-[0px_0px_0px_1px_rgba(0,0,0,0.08),0px_8px_24px_-4px_rgba(0,0,0,0.12)] ring-2 ring-indigo-400/40'
          : ['cursor-grab active:scale-[0.98] active:cursor-grabbing', CARD_SHADOW_HOVER].join(' '),
      ]
        .filter(Boolean)
        .join(' ')}
      {...(overlay ? {} : { ...listeners, ...attributes })}
    >
      {overlay ? (
        <div className="p-2.5">{cardBody}</div>
      ) : (
        <Link
          to={item.href}
          className="block p-2.5"
          draggable={false}
          onClick={(e) => {
            if (isDragActive || isDragging || suppressNavRef.current) {
              e.preventDefault()
              suppressNavRef.current = false
            }
          }}
        >
          {cardBody}
        </Link>
      )}
    </article>
  )
}

type TodoKanbanColumnProps = {
  columnId: StudentTodoColumnId
  title: string
  shortTitle: string
  items: StudentTodoItem[]
  isDragActive: boolean
  isDropTarget: boolean
  isToday?: boolean
  variant: 'weekday' | 'done'
  showWeekBadge?: boolean
  now: Date
}

function TodoKanbanColumn({
  columnId,
  title,
  shortTitle,
  items,
  isDragActive,
  isDropTarget,
  isToday = false,
  variant,
  showWeekBadge = false,
  now,
}: TodoKanbanColumnProps) {
  const isEmpty = items.length === 0
  const isDone = variant === 'done'
  const { setNodeRef, isOver } = useDroppable({
    id: columnDropId(columnId),
    data: { columnId, type: 'todo-column' },
  })

  const headerLabel = isDone ? title : shortTitle

  return (
    <section
      ref={setNodeRef}
      className={[
        'flex h-full w-[10.5rem] shrink-0 flex-col overflow-hidden rounded-xl sm:w-[11.5rem]',
        isEmpty && !isDropTarget ? 'opacity-80' : 'opacity-100',
        isOver || isDropTarget
          ? 'ring-2 ring-indigo-400/45 dark:ring-indigo-500/35'
          : '',
      ]
        .filter(Boolean)
        .join(' ')}
      aria-label={`${title} (${items.length})`}
    >
      <header
        className={[
          'flex shrink-0 items-center justify-between gap-2 rounded-t-xl px-2.5 py-2',
          isDone
            ? 'bg-emerald-600/90 text-white dark:bg-emerald-700/90'
            : isToday
              ? 'bg-indigo-600/90 text-white dark:bg-indigo-600/90'
              : 'bg-slate-100/95 text-slate-600 dark:bg-neutral-800/95 dark:text-neutral-300',
        ].join(' ')}
      >
        <div className="min-w-0">
          <h3 className="truncate text-xs font-semibold tracking-wide uppercase">
            {isDone ? (
              <span className="inline-flex items-center gap-1">
                <CheckCircle2 className="h-3.5 w-3.5 shrink-0" aria-hidden />
                {headerLabel}
              </span>
            ) : (
              headerLabel
            )}
          </h3>
          {isToday && !isDone ? (
            <p className="text-[10px] font-medium text-indigo-100">Today</p>
          ) : null}
        </div>
        <span
          className={[
            'shrink-0 rounded-md px-1.5 py-0.5 text-[11px] font-semibold tabular-nums',
            isDone || isToday
              ? 'bg-white/20 text-white'
              : 'bg-white/70 text-slate-700 dark:bg-neutral-900/50 dark:text-neutral-200',
          ].join(' ')}
        >
          {items.length}
        </span>
      </header>

      <div
        className={[
          'flex min-h-[10rem] flex-1 flex-col overflow-y-auto overscroll-contain rounded-b-xl p-2',
          isDone
            ? 'bg-emerald-50/60 dark:bg-emerald-950/20'
            : isEmpty
              ? 'bg-slate-50/50 dark:bg-neutral-900/30'
              : 'bg-white/80 dark:bg-neutral-900/50',
          isOver ? 'bg-indigo-50/50 dark:bg-indigo-950/15' : '',
        ]
          .filter(Boolean)
          .join(' ')}
      >
        {isEmpty ? (
          <div className="flex flex-1 items-center justify-center px-1 py-8 text-center">
            <p className="text-[11px] font-medium text-slate-400 dark:text-neutral-500">
              {isDragActive ? 'Drop here' : '—'}
            </p>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {items.map((item) => (
              <TodoDraggableCard
                key={item.key}
                item={item}
                isDragActive={isDragActive}
                weekBadge={showWeekBadge ? weekLabelForItem(item, now) : null}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

export type StudentTodoKanbanProps = {
  items: StudentTodoItem[]
  placements: StudentTodoPlacement[]
  weekOffsets: number[]
  now: Date
  onItemMovedToDone?: (item: StudentTodoItem) => void | Promise<void>
}

export function StudentTodoKanban({ items, placements, weekOffsets, now, onItemMovedToDone }: StudentTodoKanbanProps) {
  const weekStartKey = useMemo(() => relativeWeekStartKey(now), [now])
  const normalizedWeekOffsets = useMemo(() => normalizeWeekOffsets(weekOffsets), [weekOffsets])
  const showWeekBadge = normalizedWeekOffsets.length > 1

  const filteredItems = useMemo(
    () => filterItemsForWeeks(items, normalizedWeekOffsets, now),
    [items, normalizedWeekOffsets, now, weekStartKey],
  )
  const itemByKey = useMemo(
    () => new Map(filteredItems.map((item) => [item.key, item])),
    [filteredItems],
  )
  const savedPlacementsRef = useRef(placements)
  const [board, setBoard] = useState(() => buildStudentTodoBoard(filteredItems, placements))
  const [activeItemKey, setActiveItemKey] = useState<string | null>(null)
  const [overColumnId, setOverColumnId] = useState<StudentTodoColumnId | null>(null)
  const [boardError, setBoardError] = useState<string | null>(null)
  const [savingBoard, setSavingBoard] = useState(false)
  const today = useMemo(() => todayColumnId(now), [now, weekStartKey])

  const totalOpen = useMemo(
    () => STUDENT_TODO_WEEKDAY_COLUMN_IDS.reduce((sum, id) => sum + board[id].length, 0),
    [board],
  )

  const doneCount = board.done.length

  useEffect(() => {
    savedPlacementsRef.current = placements
    setBoard(buildStudentTodoBoard(filteredItems, placements))
  }, [filteredItems, placements])

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 150, tolerance: 8 } }),
  )

  const persistBoard = useCallback(
    async (nextBoard: Record<StudentTodoColumnId, StudentTodoItem[]>, movedToDone?: StudentTodoItem) => {
      setBoardError(null)
      setSavingBoard(true)
      try {
        if (movedToDone && onItemMovedToDone) {
          await onItemMovedToDone(movedToDone)
        }
        await saveStudentTodoBoard(boardToColumnKeys(nextBoard))
        savedPlacementsRef.current = boardToPlacements(nextBoard)
      } catch (e: unknown) {
        setBoardError(e instanceof Error ? e.message : 'Could not save todo board.')
        setBoard(buildStudentTodoBoard(filteredItems, savedPlacementsRef.current))
      } finally {
        setSavingBoard(false)
      }
    },
    [filteredItems, onItemMovedToDone],
  )

  const onDragStart = useCallback((event: DragStartEvent) => {
    setActiveItemKey(String(event.active.id))
    setOverColumnId(null)
  }, [])

  const onDragOver = useCallback(
    (event: DragOverEvent) => {
      const over = event.over
      if (!over) {
        setOverColumnId(null)
        return
      }
      const target = resolveDropTarget(board, String(over.id))
      setOverColumnId(target?.columnId ?? null)
    },
    [board],
  )

  const onDragEnd = useCallback(
    (event: DragEndEvent) => {
      setActiveItemKey(null)
      setOverColumnId(null)
      const itemKey = String(event.active.id)
      const over = event.over
      if (!over) return

      const target = resolveDropTarget(board, String(over.id))
      if (!target) return

      const previousColumn = STUDENT_TODO_COLUMN_IDS.find((col) =>
        board[col].some((item) => item.key === itemKey),
      )
      const nextBoard = computeNextStudentTodoBoard(board, itemKey, target.columnId, target.index)
      const movedItem = itemByKey.get(itemKey)
      const movedToDone = target.columnId === 'done' && previousColumn !== 'done' ? movedItem : undefined
      setBoard(nextBoard)
      void persistBoard(nextBoard, movedToDone)
    },
    [board, itemByKey, persistBoard],
  )

  const onDragCancel = useCallback(() => {
    setActiveItemKey(null)
    setOverColumnId(null)
  }, [])

  const activeItem = activeItemKey ? itemByKey.get(activeItemKey) : undefined
  const isDragActive = activeItemKey !== null

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="text-balance text-sm text-slate-600 dark:text-neutral-300">
            <span className="text-2xl font-semibold tracking-tight text-slate-950 tabular-nums dark:text-neutral-50">
              {totalOpen}
            </span>{' '}
            <span className="text-slate-500 dark:text-neutral-400">{openCountLabelForWeeks(normalizedWeekOffsets)}</span>
          </p>
          {doneCount > 0 ? (
            <p className="mt-0.5 text-xs text-emerald-700 tabular-nums dark:text-emerald-400">
              {doneCount} finished
            </p>
          ) : null}
        </div>
        <p className="max-w-xs text-pretty text-xs leading-relaxed text-slate-500 dark:text-neutral-400">
          Drag cards between days. Drop notebook tasks on Done to mark them complete.
        </p>
      </div>

      {boardError ? (
        <p className="mb-3 rounded-xl bg-rose-50 px-4 py-3 text-sm text-rose-800 shadow-[inset_0_0_0_1px_rgba(225,29,72,0.2)] dark:bg-rose-950/40 dark:text-rose-200">
          {boardError}
        </p>
      ) : null}
      {savingBoard ? (
        <p className="mb-3 text-xs text-slate-500 dark:text-neutral-400" role="status">
          Saving…
        </p>
      ) : null}

      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={onDragStart}
        onDragOver={onDragOver}
        onDragEnd={onDragEnd}
        onDragCancel={onDragCancel}
      >
        <div
          className={[
            'flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl bg-slate-100/70 p-2 dark:bg-neutral-950/40',
            BOARD_SHELL_SHADOW,
          ].join(' ')}
        >
          <div className="flex min-h-[min(28rem,calc(100vh-13rem))] gap-2 overflow-x-auto overscroll-x-contain pb-1">
            {STUDENT_TODO_WEEKDAY_COLUMN_IDS.map((columnId) => (
              <TodoKanbanColumn
                key={columnId}
                columnId={columnId}
                title={STUDENT_TODO_COLUMN_LABELS[columnId]}
                shortTitle={STUDENT_TODO_COLUMN_SHORT_LABELS[columnId]}
                items={board[columnId]}
                isDragActive={isDragActive}
                isDropTarget={overColumnId === columnId}
                isToday={normalizedWeekOffsets.includes(0) && columnId === today}
                variant="weekday"
                showWeekBadge={showWeekBadge}
                now={now}
              />
            ))}
            <div
              className="mx-0.5 w-px shrink-0 self-stretch bg-slate-300/70 dark:bg-neutral-600/70"
              aria-hidden
            />
            <TodoKanbanColumn
              columnId="done"
              title={STUDENT_TODO_COLUMN_LABELS.done}
              shortTitle={STUDENT_TODO_COLUMN_SHORT_LABELS.done}
              items={board.done}
              isDragActive={isDragActive}
              isDropTarget={overColumnId === 'done'}
              variant="done"
              showWeekBadge={showWeekBadge}
              now={now}
            />
          </div>
        </div>

        <DragOverlay dropAnimation={null}>
          {activeItem ? (
            <TodoDraggableCard
              item={activeItem}
              overlay
              isDragActive
              weekBadge={showWeekBadge ? weekLabelForItem(activeItem, now) : null}
            />
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  )
}
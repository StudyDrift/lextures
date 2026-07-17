import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  closestCorners,
  pointerWithin,
  useDroppable,
  useSensor,
  useSensors,
  type CollisionDetection,
  type DragCancelEvent,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { SortableContext, rectSortingStrategy, useSortable } from '@dnd-kit/sortable'
import { useCoursePins } from '../../context/course-pinned-context'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { heroImageObjectStyle } from '../../lib/hero-image-position'
import { listDragStyle, listDropAnimation } from '../../lib/list-motion'
import { usePrefersReducedMotion } from '../../lib/motion'
import type { PinnedCourseSummary } from '../../lib/course-catalog-settings-api'
import {
  computeNextPinRows,
  flatPinnedRows,
  newRowDropId,
  pinRowsEqual,
  resolvePinDropTarget,
  rowDropId,
} from '../../lib/pinned-courses-layout'
import { useShellNav } from './use-shell-nav'
import { SideNavTooltip } from './side-nav-tooltip'
import { CourseHeroImage } from '../course-hero-image'

function pinnedCourseTitle(course: PinnedCourseSummary): string {
  return course.catalogNickname?.trim() || course.title
}

const gridColsMap: Record<number, string> = {
  1: 'grid-cols-1',
  2: 'grid-cols-2',
  3: 'grid-cols-3',
  4: 'grid-cols-4',
}

const NEW_ROW_DROP_ID = newRowDropId()

/** Prefer the new-row slot and row gutters over sortable tile collisions. */
const pinCollisionDetection: CollisionDetection = (args) => {
  // Anything at or below the top of the new-row zone counts as a new-row drop.
  // The zone is full width, so a Y-only check (with no lower bound) keeps it
  // reachable even when dragging the last tile of the bottom row — where the
  // nearest sortable corner would otherwise always win — and tolerates
  // overshooting past the thin zone.
  const newRowRect = args.droppableRects.get(NEW_ROW_DROP_ID)
  const pointerY = args.pointerCoordinates?.y
  if (newRowRect && pointerY != null && pointerY >= newRowRect.top) {
    return [{ id: NEW_ROW_DROP_ID }]
  }

  const cornerHits = closestCorners(args)
  const sortableHit = cornerHits.find((collision) => !String(collision.id).startsWith('row:'))
  if (sortableHit) return [sortableHit]

  const rowHit = pointerWithin(args).find(
    (collision) => String(collision.id).startsWith('row:') && collision.id !== NEW_ROW_DROP_ID,
  )
  if (rowHit) return [rowHit]

  return cornerHits
}

type PinnedCourseTileVisualProps = {
  course: PinnedCourseSummary
  active: boolean
  sideNavCollapsed: boolean
  overlay?: boolean
  isPlaceholder?: boolean
  isDragActive?: boolean
  onSuppressNavClick?: () => boolean
  onConsumeNavSuppress?: () => void
}

function PinnedCourseTileVisual({
  course,
  active,
  sideNavCollapsed,
  overlay = false,
  isPlaceholder = false,
  isDragActive = false,
  onSuppressNavClick,
  onConsumeNavSuppress,
}: PinnedCourseTileVisualProps) {
  const href = `/courses/${encodeURIComponent(course.courseCode)}`
  const title = pinnedCourseTitle(course)

  return (
    <div
      className={[
        isPlaceholder ? 'pointer-events-none' : '',
        overlay ? 'scale-[1.06]' : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <NavLink
        to={href}
        aria-label={title}
        aria-current={active ? 'page' : undefined}
        draggable={false}
        tabIndex={isPlaceholder ? -1 : undefined}
        onClick={(event) => {
          if (onSuppressNavClick?.()) {
            event.preventDefault()
            onConsumeNavSuppress?.()
          }
        }}
        className={[
          'group relative block overflow-hidden rounded-xl ring-1 ring-black/[0.06] hover:ring-indigo-400/50 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:ring-white/10 dark:hover:ring-indigo-400/40',
          sideNavCollapsed ? 'h-9 w-9' : 'h-10 w-full',
          active ? 'ring-2 ring-indigo-500 dark:ring-indigo-400' : '',
          !sideNavCollapsed && !overlay && !isPlaceholder
            ? 'cursor-grab touch-none active:cursor-grabbing'
            : '',
          overlay
            ? 'cursor-grabbing shadow-lg shadow-black/20 ring-2 ring-indigo-400/50'
            : '',
          isDragActive && !overlay ? 'pointer-events-none' : '',
        ]
          .filter(Boolean)
          .join(' ')}
      >
        <CourseHeroImage
          src={course.heroImageUrl ?? '/course-card-hero.png'}
          size="catalog-thumb"
          alt=""
          draggable={false}
          loading="lazy"
          decoding="async"
          className="h-full w-full object-cover"
          style={heroImageObjectStyle(course.heroImageObjectPosition)}
        />
        <span
          className={[
            'pointer-events-none absolute inset-0 bg-gradient-to-t from-black/35 to-transparent opacity-0 transition-opacity group-hover:opacity-100',
            active ? 'opacity-100' : '',
          ]
            .filter(Boolean)
            .join(' ')}
          aria-hidden
        />
      </NavLink>
    </div>
  )
}

type SortablePinnedCourseTileProps = {
  course: PinnedCourseSummary
  active: boolean
  flashPinnedCourseId: string | null
  sideNavCollapsed: boolean
  isDragActive: boolean
}

function SortablePinnedCourseTile({
  course,
  active,
  flashPinnedCourseId,
  sideNavCollapsed,
  isDragActive,
}: SortablePinnedCourseTileProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: course.id,
    disabled: sideNavCollapsed,
  })
  const reduceMotion = usePrefersReducedMotion()
  const { ffMotionLists } = usePlatformFeatures()
  const suppressNavRef = useRef(false)
  const wasDraggingRef = useRef(false)

  useEffect(() => {
    if (wasDraggingRef.current && !isDragging) {
      suppressNavRef.current = true
    }
    wasDraggingRef.current = isDragging
  }, [isDragging])

  const style: CSSProperties = {
    ...listDragStyle({
      transform,
      transition,
      isDragging,
      reduceMotion,
      enabled: ffMotionLists,
    }),
    // Keep the sortable placeholder transparent while the overlay shows the lift.
    opacity: isDragging ? 0 : undefined,
  }

  const title = pinnedCourseTitle(course)

  return (
    <SideNavTooltip content={title} hoverWhenExpanded instant={flashPinnedCourseId === course.id}>
      <div ref={setNodeRef} className="min-w-0" style={style} {...listeners} {...attributes}>
        <PinnedCourseTileVisual
          course={course}
          active={active}
          sideNavCollapsed={sideNavCollapsed}
          isPlaceholder={isDragging}
          isDragActive={isDragActive}
          onSuppressNavClick={() => isDragActive || isDragging || suppressNavRef.current}
          onConsumeNavSuppress={() => {
            suppressNavRef.current = false
          }}
        />
      </div>
    </SideNavTooltip>
  )
}

function PinnedCourseRow({
  rowIndex,
  courses,
  sideNavCollapsed,
  flashPinnedCourseId,
  activeCourseIds,
  isDragActive,
}: {
  rowIndex: number
  courses: PinnedCourseSummary[]
  sideNavCollapsed: boolean
  flashPinnedCourseId: string | null
  activeCourseIds: Set<string>
  isDragActive: boolean
}) {
  const { setNodeRef, isOver } = useDroppable({
    id: rowDropId(rowIndex),
    disabled: sideNavCollapsed,
  })
  const cols = Math.min(Math.max(courses.length, 1), 4)
  const gridColsClass = gridColsMap[cols] ?? 'grid-cols-4'

  return (
    <div
      ref={setNodeRef}
      className={[
        sideNavCollapsed ? 'flex flex-col items-center gap-1.5' : `grid ${gridColsClass} gap-1.5`,
        isOver && isDragActive ? 'rounded-xl bg-indigo-50/70 ring-1 ring-indigo-300/50 dark:bg-indigo-950/20 dark:ring-indigo-500/30' : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {courses.map((course) => (
        <SortablePinnedCourseTile
          key={course.id}
          course={course}
          active={activeCourseIds.has(course.id)}
          flashPinnedCourseId={flashPinnedCourseId}
          sideNavCollapsed={sideNavCollapsed}
          isDragActive={isDragActive}
        />
      ))}
    </div>
  )
}

/**
 * Fixed-height hit area below the grid for dropping into a new row. The height
 * stays constant whether or not it's hovered so the drop zone never shifts out
 * from under the pointer (which would otherwise cause hover flicker).
 */
function NewPinnedRowDropZone({ active }: { active: boolean }) {
  const { setNodeRef } = useDroppable({ id: NEW_ROW_DROP_ID })
  return (
    <div ref={setNodeRef} className="flex h-12 w-full items-center" aria-hidden>
      <div
        className={[
          'h-10 w-full rounded-xl transition-colors duration-150',
          active
            ? 'bg-indigo-500/10 ring-1 ring-indigo-300/50 dark:bg-indigo-400/10 dark:ring-indigo-500/30'
            : 'bg-indigo-500/5 dark:bg-indigo-400/5',
        ].join(' ')}
      />
    </div>
  )
}

export function SideNavPinnedCourses() {
  const { pinnedRows, loading, flashPinnedCourseId, reorderPinnedRows } = useCoursePins()
  const { sideNavCollapsed } = useShellNav()
  const { ffMotionLists } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const location = useLocation()
  const dropAnimation = listDropAnimation({ reduceMotion, enabled: ffMotionLists })
  const [activeDragId, setActiveDragId] = useState<string | null>(null)
  const [hoveringNewRow, setHoveringNewRow] = useState(false)
  const [localRows, setLocalRows] = useState<PinnedCourseSummary[][]>([])
  const localRowsRef = useRef(localRows)
  const hoveringNewRowRef = useRef(false)
  const activeDragIdRef = useRef<string | null>(null)

  useEffect(() => {
    localRowsRef.current = localRows
  }, [localRows])

  useEffect(() => {
    hoveringNewRowRef.current = hoveringNewRow
  }, [hoveringNewRow])

  useEffect(() => {
    activeDragIdRef.current = activeDragId
  }, [activeDragId])

  useEffect(() => {
    if (activeDragId) return
    setLocalRows(pinnedRows)
  }, [pinnedRows, activeDragId])

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 6 },
    }),
  )

  const sortableIds = useMemo(() => flatPinnedRows(localRows).map((course) => course.id), [localRows])

  const activeCourseIds = useMemo(() => {
    const ids = new Set<string>()
    for (const row of localRows) {
      for (const course of row) {
        const href = `/courses/${encodeURIComponent(course.courseCode)}`
        if (location.pathname === href || location.pathname.startsWith(`${href}/`)) {
          ids.add(course.id)
        }
      }
    }
    return ids
  }, [localRows, location.pathname])

  const activeCourse = useMemo(() => {
    if (!activeDragId) return null
    for (const row of localRows) {
      const found = row.find((course) => course.id === activeDragId)
      if (found) return found
    }
    return null
  }, [activeDragId, localRows])

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveDragId(String(event.active.id))
    setHoveringNewRow(false)
  }, [])

  const handleDragOver = useCallback(
    (event: DragOverEvent) => {
      const { active, over } = event
      if (!over) {
        setHoveringNewRow(false)
        return
      }

      const overId = String(over.id)
      if (overId === NEW_ROW_DROP_ID) {
        setHoveringNewRow(true)
        return
      }

      setHoveringNewRow(false)
      if (active.id === over.id) return

      const courseId = String(active.id)
      setLocalRows((current) => {
        const target = resolvePinDropTarget(current, overId)
        if (!target || target.kind === 'new-row') return current
        const next = computeNextPinRows(current, courseId, target)
        return pinRowsEqual(current, next) ? current : next
      })
    },
    [],
  )

  const handleDragEnd = useCallback(
    (_event: DragEndEvent) => {
      const draggedId = activeDragIdRef.current
      let finalRows = localRowsRef.current

      if (hoveringNewRowRef.current && draggedId) {
        finalRows = computeNextPinRows(finalRows, draggedId, { kind: 'new-row' })
        setLocalRows(finalRows)
      }

      setActiveDragId(null)
      setHoveringNewRow(false)

      if (!pinRowsEqual(finalRows, pinnedRows)) {
        // Commit synchronously: reorderPinnedRows updates pinnedRows optimistically
        // before awaiting the API, so the effect that syncs localRows from
        // pinnedRows on drag end sees the new order and doesn't snap back.
        void reorderPinnedRows(finalRows).catch(() => {
          setLocalRows(pinnedRows)
        })
      }
    },
    [pinnedRows, reorderPinnedRows],
  )

  const handleDragCancel = useCallback((_event: DragCancelEvent) => {
    setActiveDragId(null)
    setHoveringNewRow(false)
    setLocalRows(pinnedRows)
  }, [pinnedRows])

  if (loading || localRows.length === 0) return null

  const isDragging = activeDragId !== null

  const rowsContent = (
    <div
      className={[
        'shrink-0 px-3 py-3',
        sideNavCollapsed ? 'flex flex-col items-center gap-1.5' : 'flex flex-col gap-1.5',
      ].join(' ')}
      aria-label="Pinned courses"
    >
      {localRows.map((row, rowIndex) => (
        <PinnedCourseRow
          key={`pin-row-${rowIndex}`}
          rowIndex={rowIndex}
          courses={row}
          sideNavCollapsed={sideNavCollapsed}
          flashPinnedCourseId={flashPinnedCourseId}
          activeCourseIds={activeCourseIds}
          isDragActive={isDragging}
        />
      ))}
    </div>
  )

  if (sideNavCollapsed) return rowsContent

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={pinCollisionDetection}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
      onDragCancel={handleDragCancel}
    >
      <SortableContext items={sortableIds} strategy={rectSortingStrategy}>
        {rowsContent}
      </SortableContext>
      {isDragging ? (
        <div className="shrink-0 px-3 pb-3">
          <NewPinnedRowDropZone active={hoveringNewRow} />
        </div>
      ) : null}
      <DragOverlay dropAnimation={dropAnimation}>
        {activeCourse ? (
          <PinnedCourseTileVisual
            course={activeCourse}
            active={activeCourseIds.has(activeCourse.id)}
            sideNavCollapsed={false}
            overlay
          />
        ) : null}
      </DragOverlay>
    </DndContext>
  )
}
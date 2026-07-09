import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type MouseEvent as ReactMouseEvent,
  type MutableRefObject,
  type PointerEvent as ReactPointerEvent,
  type SVGProps,
} from 'react'
import { Link } from 'react-router-dom'
import {
  closestCorners,
  DndContext,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  rectSortingStrategy,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { BookOpen, Eye, EyeOff, Plus } from 'lucide-react'
import { KeyboardSensor as SharedKeyboardSensor, defaultKeyboardSensorOptions } from '../../lib/dnd/keyboardSensorConfig'
import { useCanvasImport } from '../../context/canvas-import-context'
import { CourseCatalogImportFromCourseModal } from './course-catalog-import-from-course-modal'
import { CourseCatalogImportMenu } from './course-catalog-import-menu'
import {
  CourseCatalogViewMenu,
} from './course-catalog-view-menu'
import { CourseCatalogKanbanBoard } from './course-catalog-kanban'
import { courseCatalogStatusLabel } from './course-catalog-status'
import { courseCatalogDescriptionBlurb, courseCatalogDisplayTitle } from './course-catalog-display'
import { CourseCatalogNicknameEditor } from './course-catalog-nickname-editor'
import { CourseCatalogPinButton } from './course-catalog-pin-button'
import { CourseCatalogActionsMenu } from './course-catalog-actions-menu'
import {
  buildCatalogSections,
  catalogEmptyStateKind,
  countUserHiddenCourses,
  filterCatalogCourses,
  isUserCatalogHidden,
  type CatalogSection,
} from './course-catalog-hidden'
import {
  DEFAULT_KANBAN_COLUMN_LABELS,
  fetchCourseCatalogSettings,
  migrateLegacyCourseCatalogLocalStorage,
  putCourseCatalogSettings,
  putCourseKanbanBoard,
  type KanbanColumnLabels,
} from '../../lib/course-catalog-settings-api'
import type { CourseCatalogView, KanbanColumnId } from '../../lib/course-catalog-types'
import { EmptyState } from '../../components/ui/empty-state'
import { CoursesCatalogSkeleton } from '../../components/ui/lms-content-skeletons'
import { LmsPage } from './lms-page'
import { usePermissions } from '../../context/use-permissions'
import { useBumpCoursesRevision, useCoursesRevision } from '../../context/use-inbox-unread'
import { authorizedFetch } from '../../lib/api'
import { putCourseCatalogOrder, type CoursePublic, fetchOrgTerms, fetchOrgType, type OrgTerm, type OrgType } from '../../lib/courses-api'
import { decodeJwtPayload } from '../../lib/jwt-payload'
import { getAccessToken } from '../../lib/auth'
import { readApiErrorMessage } from '../../lib/errors'
import { heroImageObjectStyle } from '../../lib/hero-image-position'
import { formatRelativeCompact } from '../../lib/format-datetime'
import { CourseCatalogStatusPill } from '../../components/ui/status-vocabulary'
import { canCreateCourses } from '../../lib/rbac-api'
import { CourseHeroImage } from '../../components/course-hero-image'
import type { CourseHeroImageSize } from '../../lib/course-hero-image-url'
import { CourseEnrollmentInvitationActions } from '../../components/enrollment/course-enrollment-invitation-actions'

export type { CoursePublic } from '../../lib/courses-api'

type CatalogNicknameChangeHandler = (courseId: string, nickname: string | null) => void
type CatalogPinnedChangeHandler = (courseId: string, pinned: boolean) => void
type CatalogHiddenChangeHandler = (courseId: string, hidden: boolean) => void
type CatalogInvitationResolvedHandler = (courseId: string, approved: boolean) => void

function CourseCatalogHiddenBadge() {
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
      <EyeOff className="h-3 w-3" aria-hidden />
      Hidden
    </span>
  )
}

function catalogRevealedHiddenClass(course: CoursePublic, showHidden: boolean): string {
  return showHidden && isUserCatalogHidden(course) ? 'opacity-70' : ''
}

function courseInvitationPending(course: CoursePublic): boolean {
  return Boolean(
    course.viewerEnrollmentInvitationPending && course.viewerPendingEnrollmentId,
  )
}

function CreateCourseIcon({ className, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" className={className} {...props}>
      {/* Left page */}
      <path
        d="M12 20.5C10.4 19.7 8.4 19.5 6 19.5C4.3 19.5 3 19.7 2 20V5.5C3 5.2 4.3 5 6 5C8.5 5 10.5 5.3 12 6V20.5Z"
        fill="#e0e7ff"
        stroke="#6366f1"
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
      {/* Right page */}
      <path
        d="M12 20.5C13.6 19.7 15.6 19.5 18 19.5C19.7 19.5 21 19.7 22 20V5.5C21 5.2 19.7 5 18 5C15.5 5 13.5 5.3 12 6V20.5Z"
        fill="#c7d2fe"
        stroke="#6366f1"
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
      {/* Lines on left page */}
      <path d="M4.5 10H9.5" stroke="#6366f1" strokeWidth="1" strokeLinecap="round" />
      <path d="M4.5 13H9.5" stroke="#6366f1" strokeWidth="1" strokeLinecap="round" />
      <path d="M4.5 16H7" stroke="#6366f1" strokeWidth="1" strokeLinecap="round" />
      {/* Plus on right page */}
      <path d="M18 10.5V15.5" stroke="#6366f1" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M15.5 13H20.5" stroke="#6366f1" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

const COURSE_GRID_SORT_ID = 'course-catalog-grid'
const CATALOG_DRAG_ACTIVATION_DISTANCE_SQ = 8 * 8

type CatalogPointerDragGuard = {
  x: number
  y: number
  moved: boolean
}

function catalogCardPointerDownCapture(
  guard: MutableRefObject<CatalogPointerDragGuard>,
  e: ReactPointerEvent<HTMLElement>,
) {
  guard.current = { x: e.clientX, y: e.clientY, moved: false }
}

function shouldSuppressCatalogLinkClick(opts: {
  pointerGuard?: MutableRefObject<CatalogPointerDragGuard>
  justFinishedDraggingRef?: MutableRefObject<boolean>
  isDragging?: boolean
  catalogDragActive?: boolean
  suppressAfterDragRef?: MutableRefObject<boolean>
}): boolean {
  return Boolean(
    opts.isDragging ||
      opts.catalogDragActive ||
      opts.suppressAfterDragRef?.current ||
      opts.justFinishedDraggingRef?.current ||
      opts.pointerGuard?.current.moved,
  )
}

function consumeCatalogLinkClickSuppress(opts: {
  pointerGuard?: MutableRefObject<CatalogPointerDragGuard>
  justFinishedDraggingRef?: MutableRefObject<boolean>
  suppressAfterDragRef?: MutableRefObject<boolean>
}) {
  if (opts.pointerGuard) opts.pointerGuard.current.moved = false
  if (opts.justFinishedDraggingRef) opts.justFinishedDraggingRef.current = false
  if (opts.suppressAfterDragRef) opts.suppressAfterDragRef.current = false
}

function useCatalogSortablePointerGuard(isDragging: boolean) {
  const pointerGuardRef = useRef<CatalogPointerDragGuard>({ x: 0, y: 0, moved: false })
  const justFinishedDraggingRef = useRef(false)
  const wasDraggingRef = useRef(false)

  useEffect(() => {
    if (isDragging) {
      pointerGuardRef.current.moved = true
      justFinishedDraggingRef.current = false
    } else if (wasDraggingRef.current) {
      pointerGuardRef.current.moved = true
      justFinishedDraggingRef.current = true
    }
    wasDraggingRef.current = isDragging
  }, [isDragging])

  const onPointerDownCapture = useCallback((e: ReactPointerEvent<HTMLElement>) => {
    justFinishedDraggingRef.current = false
    catalogCardPointerDownCapture(pointerGuardRef, e)
    const pointerId = e.pointerId

    const onWindowPointerMove = (ev: PointerEvent) => {
      if (ev.pointerId !== pointerId) return
      if (pointerGuardRef.current.moved) return
      const dx = ev.clientX - pointerGuardRef.current.x
      const dy = ev.clientY - pointerGuardRef.current.y
      if (dx * dx + dy * dy >= CATALOG_DRAG_ACTIVATION_DISTANCE_SQ) {
        pointerGuardRef.current.moved = true
      }
    }

    const endWindowTracking = (ev: PointerEvent) => {
      if (ev.pointerId !== pointerId) return
      window.removeEventListener('pointermove', onWindowPointerMove)
      window.removeEventListener('pointerup', endWindowTracking)
      window.removeEventListener('pointercancel', endWindowTracking)
    }

    window.addEventListener('pointermove', onWindowPointerMove)
    window.addEventListener('pointerup', endWindowTracking)
    window.addEventListener('pointercancel', endWindowTracking)
  }, [])

  return { pointerGuardRef, justFinishedDraggingRef, onPointerDownCapture }
}

type SortableCourseProps = {
  listeners: Record<string, unknown>
  setNodeRef: (node: HTMLElement | null) => void
  style: CSSProperties
  isDragging: boolean
  pointerGuardRef: MutableRefObject<CatalogPointerDragGuard>
  justFinishedDraggingRef: MutableRefObject<boolean>
  onPointerDownCapture: (e: ReactPointerEvent<HTMLElement>) => void
}

type CatalogCourseDragProps = {
  catalogDragActive: boolean
  suppressNavigateAfterDragRef: MutableRefObject<boolean>
}

function formatEditedAgo(iso: string): string {
  return `Edited ${formatRelativeCompact(iso)}`
}

function formatCourseTermLabel(course: CoursePublic): string {
  return course.term?.name?.trim() || '—'
}

const CATALOG_LIST_HERO_FRAME =
  'relative aspect-[7/5] w-28 shrink-0 overflow-hidden rounded-lg bg-slate-100 dark:bg-neutral-800'

function CatalogCourseHero({
  course,
  size,
  className,
}: {
  course: CoursePublic
  size: CourseHeroImageSize
  className: string
}) {
  return (
    <CourseHeroImage
      data-lex-hero
      src={course.heroImageUrl ?? '/course-card-hero.png'}
      size={size}
      alt=""
      draggable={false}
      loading="lazy"
      decoding="async"
      className={className}
      style={heroImageObjectStyle(course.heroImageObjectPosition)}
    />
  )
}

function catalogViewUsesGrid(view: CourseCatalogView): boolean {
  return view === 'cards' || view === 'gallery'
}

function CourseCard({
  course,
  sortable,
  catalogDragActive,
  suppressNavigateAfterDragRef,
  showHiddenRevealed,
  renameRequest,
  onNicknameChange,
  onPinnedChange,
  onHiddenChange,
  onInvitationResolved,
}: {
  course: CoursePublic
  catalogDragActive: boolean
  suppressNavigateAfterDragRef: MutableRefObject<boolean>
  showHiddenRevealed: boolean
  renameRequest?: number
  onNicknameChange: CatalogNicknameChangeHandler
  onPinnedChange: CatalogPinnedChangeHandler
  onHiddenChange: CatalogHiddenChangeHandler
  onInvitationResolved?: CatalogInvitationResolvedHandler
  sortable?: SortableCourseProps
}) {
  const courseHref = `/courses/${encodeURIComponent(course.courseCode)}`
  const badgeLabel = courseCatalogStatusLabel(course)
  const descriptionBlurb = courseCatalogDescriptionBlurb(course)
  const displayTitle = courseCatalogDisplayTitle(course)
  const invitationPending = courseInvitationPending(course)

  const heroBlock = (
    <>
      <CatalogCourseHero course={course} size="catalog-card" className="h-40 w-full object-cover" />
      <div
        className="pointer-events-none absolute inset-0 bg-gradient-to-t from-black/80 via-black/25 to-transparent"
        aria-hidden
      />
      <span className="absolute start-3 top-3">
        <CourseCatalogStatusPill label={invitationPending ? 'Invitation' : badgeLabel} />
      </span>
      {!invitationPending ? (
        <span className="absolute end-3 top-3 z-10 flex items-center gap-1">
          <CourseCatalogPinButton course={course} onPinnedChange={onPinnedChange} />
          <CourseCatalogActionsMenu
            course={course}
            variant="overlay"
            onPinnedChange={onPinnedChange}
            onHiddenChange={onHiddenChange}
          />
        </span>
      ) : null}
      <div className="absolute inset-x-0 bottom-0 p-4 pt-10">
        <h2 className="text-lg font-semibold leading-snug tracking-tight text-white drop-shadow-sm line-clamp-2">
          {displayTitle}
        </h2>
      </div>
    </>
  )

  const invitationMutedClass = invitationPending ? 'opacity-60 grayscale' : ''

  const suppressLinkClick = () =>
    shouldSuppressCatalogLinkClick({
      pointerGuard: sortable?.pointerGuardRef,
      justFinishedDraggingRef: sortable?.justFinishedDraggingRef,
      isDragging: sortable?.isDragging,
      catalogDragActive,
      suppressAfterDragRef: suppressNavigateAfterDragRef,
    })

  const onCatalogLinkClick = (e: ReactMouseEvent<HTMLAnchorElement>) => {
    if (!suppressLinkClick()) return
    e.preventDefault()
    e.stopPropagation()
    consumeCatalogLinkClickSuppress({
      pointerGuard: sortable?.pointerGuardRef,
      justFinishedDraggingRef: sortable?.justFinishedDraggingRef,
      suppressAfterDragRef: suppressNavigateAfterDragRef,
    })
  }

  return (
    <article
      ref={sortable?.setNodeRef}
      style={sortable?.style}
      className={[
        'flex h-full flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-900/5 transition-shadow dark:border-neutral-700 dark:bg-neutral-900',
        catalogRevealedHiddenClass(course, showHiddenRevealed),
        sortable ? 'touch-none cursor-grab active:cursor-grabbing' : '',
        sortable?.isDragging ? 'shadow-md shadow-slate-900/10 ring-2 ring-indigo-400/40' : '',
      ]
        .filter(Boolean)
        .join(' ')}
      onPointerDownCapture={sortable?.onPointerDownCapture}
      {...(sortable ? sortable.listeners : {})}
    >
      {showHiddenRevealed && isUserCatalogHidden(course) ? (
        <div className="border-b border-slate-100 px-4 py-2 dark:border-neutral-800">
          <CourseCatalogHiddenBadge />
        </div>
      ) : null}
      <div className={invitationMutedClass}>
        {invitationPending ? (
          <div className="relative block" aria-label={`${displayTitle} — invitation pending`}>
            {heroBlock}
          </div>
        ) : (
          <Link
            to={courseHref}
            className="relative block focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-indigo-500"
            aria-label={`Open ${displayTitle}`}
            onClick={onCatalogLinkClick}
          >
            {heroBlock}
          </Link>
        )}
      </div>

      <div className="flex flex-1 flex-col justify-end px-5 pb-4 pt-3">
        <div className={invitationMutedClass}>
          <CourseCatalogNicknameEditor
            course={course}
            compact
            openRequest={renameRequest}
            onNicknameChange={onNicknameChange}
          />
        </div>
        {invitationPending && course.viewerPendingEnrollmentId ? (
          <>
            <p className="mt-3 text-start text-sm text-slate-600 dark:text-neutral-400">
              You have been invited to this course. Approve to join or decline to remove the invitation.
            </p>
            <CourseEnrollmentInvitationActions
              courseCode={course.courseCode}
              enrollmentId={course.viewerPendingEnrollmentId}
              onResolved={(approved) => onInvitationResolved?.(course.id, approved)}
            />
          </>
        ) : (
          <>
            {descriptionBlurb ? (
              <p className="mt-3 text-start text-sm leading-snug text-slate-600 line-clamp-4 dark:text-neutral-400">
                {descriptionBlurb}
              </p>
            ) : null}
            <p className="mt-3 text-start text-xs text-slate-400 dark:text-neutral-500">
              {formatEditedAgo(course.updatedAt)}
            </p>
          </>
        )}
      </div>
    </article>
  )
}

function SortableCourseCard({
  course,
  catalogDragActive,
  suppressNavigateAfterDragRef,
  showHiddenRevealed,
  onNicknameChange,
  onPinnedChange,
  onHiddenChange,
  onInvitationResolved,
}: {
  course: CoursePublic
} & CatalogCourseDragProps & {
  showHiddenRevealed: boolean
  onNicknameChange: CatalogNicknameChangeHandler
  onPinnedChange: CatalogPinnedChangeHandler
  onHiddenChange: CatalogHiddenChangeHandler
  onInvitationResolved?: CatalogInvitationResolvedHandler
}) {
  const { listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: course.id,
  })
  const { pointerGuardRef, justFinishedDraggingRef, onPointerDownCapture } =
    useCatalogSortablePointerGuard(isDragging)
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.92 : undefined,
    zIndex: isDragging ? 20 : undefined,
  }

  return (
    <div className="h-full min-h-0">
      <CourseCard
        course={course}
        catalogDragActive={catalogDragActive}
        suppressNavigateAfterDragRef={suppressNavigateAfterDragRef}
        showHiddenRevealed={showHiddenRevealed}
        onNicknameChange={onNicknameChange}
        onPinnedChange={onPinnedChange}
        onHiddenChange={onHiddenChange}
        onInvitationResolved={onInvitationResolved}
        sortable={{
          listeners: listeners as Record<string, unknown>,
          setNodeRef,
          style,
          isDragging,
          pointerGuardRef,
          justFinishedDraggingRef,
          onPointerDownCapture,
        }}
      />
    </div>
  )
}

function CourseListRow({
  course,
  sortable,
  catalogDragActive,
  suppressNavigateAfterDragRef,
  showHiddenRevealed,
  renameRequest,
  onRenameRequest,
  onNicknameChange,
  onPinnedChange,
  onHiddenChange,
  onInvitationResolved,
}: {
  course: CoursePublic
  catalogDragActive: boolean
  suppressNavigateAfterDragRef: MutableRefObject<boolean>
  showHiddenRevealed: boolean
  renameRequest?: number
  onRenameRequest?: () => void
  onNicknameChange: CatalogNicknameChangeHandler
  onPinnedChange: CatalogPinnedChangeHandler
  onHiddenChange: CatalogHiddenChangeHandler
  onInvitationResolved?: CatalogInvitationResolvedHandler
  sortable?: SortableCourseProps
}) {
  const courseHref = `/courses/${encodeURIComponent(course.courseCode)}`
  const badgeLabel = courseCatalogStatusLabel(course)
  const descriptionBlurb = courseCatalogDescriptionBlurb(course)
  const displayTitle = courseCatalogDisplayTitle(course)
  const invitationPending = courseInvitationPending(course)
  const invitationMutedClass = invitationPending ? 'opacity-60 grayscale' : ''

  const suppressLinkClick = () =>
    shouldSuppressCatalogLinkClick({
      pointerGuard: sortable?.pointerGuardRef,
      justFinishedDraggingRef: sortable?.justFinishedDraggingRef,
      isDragging: sortable?.isDragging,
      catalogDragActive,
      suppressAfterDragRef: suppressNavigateAfterDragRef,
    })

  const onCatalogLinkClick = (e: ReactMouseEvent<HTMLAnchorElement>) => {
    if (!suppressLinkClick()) return
    e.preventDefault()
    e.stopPropagation()
    consumeCatalogLinkClickSuppress({
      pointerGuard: sortable?.pointerGuardRef,
      justFinishedDraggingRef: sortable?.justFinishedDraggingRef,
      suppressAfterDragRef: suppressNavigateAfterDragRef,
    })
  }

  return (
    <article
      ref={sortable?.setNodeRef}
      style={sortable?.style}
      className={[
        'flex overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm shadow-slate-900/5 transition-shadow dark:border-neutral-700 dark:bg-neutral-900',
        catalogRevealedHiddenClass(course, showHiddenRevealed),
        sortable ? 'touch-none cursor-grab active:cursor-grabbing' : '',
        sortable?.isDragging ? 'shadow-md shadow-slate-900/10 ring-2 ring-indigo-400/40' : '',
      ]
        .filter(Boolean)
        .join(' ')}
      onPointerDownCapture={sortable?.onPointerDownCapture}
      {...(sortable ? sortable.listeners : {})}
    >
      <div className="flex min-w-0 flex-1 items-stretch gap-4 p-3 sm:p-4">
        <div className={invitationMutedClass}>
        {invitationPending ? (
          <div className={CATALOG_LIST_HERO_FRAME} aria-hidden>
            <CatalogCourseHero
              course={course}
              size="catalog-list"
              className="absolute inset-0 h-full w-full object-cover"
            />
          </div>
        ) : (
          <Link
            to={courseHref}
            className={`${CATALOG_LIST_HERO_FRAME} focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500`}
            aria-label={`Open ${displayTitle}`}
            onClick={onCatalogLinkClick}
          >
            <CatalogCourseHero
              course={course}
              size="catalog-list"
              className="absolute inset-0 h-full w-full object-cover"
            />
          </Link>
        )}
        </div>
        <div className="flex min-w-0 flex-1 flex-col justify-center gap-1">
          <div className={`flex flex-wrap items-center gap-2 ${invitationMutedClass}`}>
            <CourseCatalogNicknameEditor
              course={course}
              titleClassName="text-base font-semibold leading-snug text-slate-900 line-clamp-1 dark:text-neutral-100"
              openRequest={renameRequest}
              onNicknameChange={onNicknameChange}
            />
            <CourseCatalogStatusPill label={invitationPending ? 'Invitation' : badgeLabel} />
            {showHiddenRevealed && isUserCatalogHidden(course) ? <CourseCatalogHiddenBadge /> : null}
          </div>
          {invitationPending && course.viewerPendingEnrollmentId ? (
            <CourseEnrollmentInvitationActions
              compact
              courseCode={course.courseCode}
              enrollmentId={course.viewerPendingEnrollmentId}
              onResolved={(approved) => onInvitationResolved?.(course.id, approved)}
            />
          ) : descriptionBlurb ? (
            <Link
              to={courseHref}
              className="text-start text-sm leading-snug text-slate-600 line-clamp-2 hover:text-indigo-600 dark:text-neutral-400 dark:hover:text-indigo-300"
              onClick={onCatalogLinkClick}
            >
              {descriptionBlurb}
            </Link>
          ) : null}
          {!invitationPending ? (
            <p className="text-start text-xs text-slate-400 dark:text-neutral-500">{formatEditedAgo(course.updatedAt)}</p>
          ) : null}
        </div>
        {!invitationPending ? (
          <div className="flex shrink-0 items-start gap-1 pt-1">
            <CourseCatalogPinButton course={course} variant="inline" onPinnedChange={onPinnedChange} />
            <CourseCatalogActionsMenu
              course={course}
              onPinnedChange={onPinnedChange}
              onHiddenChange={onHiddenChange}
              onRenameRequest={onRenameRequest}
            />
          </div>
        ) : null}
      </div>
    </article>
  )
}

function SortableCourseListRow({
  course,
  catalogDragActive,
  suppressNavigateAfterDragRef,
  showHiddenRevealed,
  onNicknameChange,
  onPinnedChange,
  onHiddenChange,
  onInvitationResolved,
}: {
  course: CoursePublic
} & CatalogCourseDragProps & {
  showHiddenRevealed: boolean
  onNicknameChange: CatalogNicknameChangeHandler
  onPinnedChange: CatalogPinnedChangeHandler
  onHiddenChange: CatalogHiddenChangeHandler
  onInvitationResolved?: CatalogInvitationResolvedHandler
}) {
  const [renameRequest, setRenameRequest] = useState(0)
  const { listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: course.id,
  })
  const { pointerGuardRef, justFinishedDraggingRef, onPointerDownCapture } =
    useCatalogSortablePointerGuard(isDragging)
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.92 : undefined,
    zIndex: isDragging ? 20 : undefined,
  }

  return (
    <CourseListRow
      course={course}
      catalogDragActive={catalogDragActive}
      suppressNavigateAfterDragRef={suppressNavigateAfterDragRef}
      showHiddenRevealed={showHiddenRevealed}
      renameRequest={renameRequest}
      onRenameRequest={() => setRenameRequest((n) => n + 1)}
      onNicknameChange={onNicknameChange}
      onPinnedChange={onPinnedChange}
      onHiddenChange={onHiddenChange}
      onInvitationResolved={onInvitationResolved}
      sortable={{
        listeners: listeners as Record<string, unknown>,
        setNodeRef,
        style,
        isDragging,
        pointerGuardRef,
        justFinishedDraggingRef,
        onPointerDownCapture,
      }}
    />
  )
}

function CourseGalleryTile({
  course,
  sortable,
  catalogDragActive,
  suppressNavigateAfterDragRef,
  showHiddenRevealed,
  onNicknameChange,
  onPinnedChange,
  onHiddenChange,
  onInvitationResolved,
}: {
  course: CoursePublic
  catalogDragActive: boolean
  suppressNavigateAfterDragRef: MutableRefObject<boolean>
  showHiddenRevealed: boolean
  onNicknameChange: CatalogNicknameChangeHandler
  onPinnedChange: CatalogPinnedChangeHandler
  onHiddenChange: CatalogHiddenChangeHandler
  onInvitationResolved?: CatalogInvitationResolvedHandler
  sortable?: SortableCourseProps
}) {
  const courseHref = `/courses/${encodeURIComponent(course.courseCode)}`
  const badgeLabel = courseCatalogStatusLabel(course)
  const displayTitle = courseCatalogDisplayTitle(course)
  const invitationPending = courseInvitationPending(course)
  const invitationMutedClass = invitationPending ? 'opacity-60 grayscale' : ''

  const suppressLinkClick = () =>
    shouldSuppressCatalogLinkClick({
      pointerGuard: sortable?.pointerGuardRef,
      justFinishedDraggingRef: sortable?.justFinishedDraggingRef,
      isDragging: sortable?.isDragging,
      catalogDragActive,
      suppressAfterDragRef: suppressNavigateAfterDragRef,
    })

  const onCatalogLinkClick = (e: ReactMouseEvent<HTMLAnchorElement>) => {
    if (!suppressLinkClick()) return
    e.preventDefault()
    e.stopPropagation()
    consumeCatalogLinkClickSuppress({
      pointerGuard: sortable?.pointerGuardRef,
      justFinishedDraggingRef: sortable?.justFinishedDraggingRef,
      suppressAfterDragRef: suppressNavigateAfterDragRef,
    })
  }

  return (
    <article
      ref={sortable?.setNodeRef}
      style={sortable?.style}
      className={[
        'overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm shadow-slate-900/5 transition-shadow dark:border-neutral-700 dark:bg-neutral-900',
        catalogRevealedHiddenClass(course, showHiddenRevealed),
        sortable ? 'touch-none cursor-grab active:cursor-grabbing' : '',
        sortable?.isDragging ? 'shadow-md shadow-slate-900/10 ring-2 ring-indigo-400/40' : '',
      ]
        .filter(Boolean)
        .join(' ')}
      onPointerDownCapture={sortable?.onPointerDownCapture}
      {...(sortable ? sortable.listeners : {})}
    >
      {showHiddenRevealed && isUserCatalogHidden(course) ? (
        <div className="border-b border-slate-100 px-3 py-1.5 dark:border-neutral-800">
          <CourseCatalogHiddenBadge />
        </div>
      ) : null}
      <div className={invitationMutedClass}>
      {invitationPending ? (
        <div className="relative block aspect-[4/3]" aria-hidden>
          <CatalogCourseHero
            course={course}
            size="catalog-gallery"
            className="absolute inset-0 h-full w-full object-cover"
          />
        </div>
      ) : (
      <Link
        to={courseHref}
        className="relative block aspect-[4/3] focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-indigo-500"
        aria-label={`Open ${displayTitle}`}
        onClick={onCatalogLinkClick}
      >
        <CatalogCourseHero
          course={course}
          size="catalog-gallery"
          className="absolute inset-0 h-full w-full object-cover"
        />
        <div
          className="pointer-events-none absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"
          aria-hidden
        />
        <span className="absolute start-2 top-2">
          <CourseCatalogStatusPill label={invitationPending ? 'Invitation' : badgeLabel} />
        </span>
        {!invitationPending ? (
          <span className="absolute end-2 top-2 z-10 flex items-center gap-1">
            <CourseCatalogPinButton course={course} onPinnedChange={onPinnedChange} />
            <CourseCatalogActionsMenu
              course={course}
              variant="overlay"
              onPinnedChange={onPinnedChange}
              onHiddenChange={onHiddenChange}
            />
          </span>
        ) : null}
        <h2 className="absolute inset-x-0 bottom-0 p-3 text-sm font-semibold leading-snug text-white drop-shadow-sm line-clamp-2">
          {displayTitle}
        </h2>
      </Link>
      )}
      </div>
      <div className="border-t border-slate-100 px-3 py-2 dark:border-neutral-800">
        <div className={invitationMutedClass}>
          <CourseCatalogNicknameEditor course={course} compact onNicknameChange={onNicknameChange} />
        </div>
        {invitationPending && course.viewerPendingEnrollmentId ? (
          <CourseEnrollmentInvitationActions
            compact
            courseCode={course.courseCode}
            enrollmentId={course.viewerPendingEnrollmentId}
            onResolved={(approved) => onInvitationResolved?.(course.id, approved)}
          />
        ) : null}
      </div>
    </article>
  )
}

function SortableCourseGalleryTile({
  course,
  catalogDragActive,
  suppressNavigateAfterDragRef,
  showHiddenRevealed,
  onNicknameChange,
  onPinnedChange,
  onHiddenChange,
  onInvitationResolved,
}: {
  course: CoursePublic
} & CatalogCourseDragProps & {
  showHiddenRevealed: boolean
  onNicknameChange: CatalogNicknameChangeHandler
  onPinnedChange: CatalogPinnedChangeHandler
  onHiddenChange: CatalogHiddenChangeHandler
  onInvitationResolved?: CatalogInvitationResolvedHandler
}) {
  const { listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: course.id,
  })
  const { pointerGuardRef, justFinishedDraggingRef, onPointerDownCapture } =
    useCatalogSortablePointerGuard(isDragging)
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.92 : undefined,
    zIndex: isDragging ? 20 : undefined,
  }

  return (
    <CourseGalleryTile
      course={course}
      catalogDragActive={catalogDragActive}
      suppressNavigateAfterDragRef={suppressNavigateAfterDragRef}
      showHiddenRevealed={showHiddenRevealed}
      onNicknameChange={onNicknameChange}
      onPinnedChange={onPinnedChange}
      onHiddenChange={onHiddenChange}
      onInvitationResolved={onInvitationResolved}
      sortable={{
        listeners: listeners as Record<string, unknown>,
        setNodeRef,
        style,
        isDragging,
        pointerGuardRef,
        justFinishedDraggingRef,
        onPointerDownCapture,
      }}
    />
  )
}

function CourseCatalogTableHeader() {
  return (
    <div
      className="grid grid-cols-[minmax(0,2.2fr)_minmax(5.5rem,auto)_minmax(0,1.1fr)_minmax(5.5rem,auto)_minmax(4.5rem,auto)] gap-3 border-b border-slate-200 px-4 py-2.5 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:border-neutral-700 dark:text-neutral-400"
      aria-hidden
    >
      <span>Title</span>
      <span>Status</span>
      <span>Term</span>
      <span>Edited</span>
      <span>Code</span>
    </div>
  )
}

function CourseTableRow({
  course,
  sortable,
  catalogDragActive,
  suppressNavigateAfterDragRef,
  showHiddenRevealed,
  renameRequest,
  onRenameRequest,
  onNicknameChange,
  onPinnedChange,
  onHiddenChange,
  onInvitationResolved,
}: {
  course: CoursePublic
  catalogDragActive: boolean
  suppressNavigateAfterDragRef: MutableRefObject<boolean>
  showHiddenRevealed: boolean
  renameRequest?: number
  onRenameRequest?: () => void
  onNicknameChange: CatalogNicknameChangeHandler
  onPinnedChange: CatalogPinnedChangeHandler
  onHiddenChange: CatalogHiddenChangeHandler
  onInvitationResolved?: CatalogInvitationResolvedHandler
  sortable?: SortableCourseProps
}) {
  const courseHref = `/courses/${encodeURIComponent(course.courseCode)}`
  const badgeLabel = courseCatalogStatusLabel(course)
  const invitationPending = courseInvitationPending(course)
  const invitationMutedClass = invitationPending ? 'opacity-60 grayscale' : ''

  const suppressLinkClick = () =>
    shouldSuppressCatalogLinkClick({
      pointerGuard: sortable?.pointerGuardRef,
      justFinishedDraggingRef: sortable?.justFinishedDraggingRef,
      isDragging: sortable?.isDragging,
      catalogDragActive,
      suppressAfterDragRef: suppressNavigateAfterDragRef,
    })

  const onCatalogLinkClick = (e: ReactMouseEvent<HTMLAnchorElement>) => {
    if (!suppressLinkClick()) return
    e.preventDefault()
    e.stopPropagation()
    consumeCatalogLinkClickSuppress({
      pointerGuard: sortable?.pointerGuardRef,
      justFinishedDraggingRef: sortable?.justFinishedDraggingRef,
      suppressAfterDragRef: suppressNavigateAfterDragRef,
    })
  }

  return (
    <article
      ref={sortable?.setNodeRef}
      style={sortable?.style}
      className={[
        'grid grid-cols-[minmax(0,2.2fr)_minmax(5.5rem,auto)_minmax(0,1.1fr)_minmax(5.5rem,auto)_minmax(4.5rem,auto)] gap-3 border-b border-slate-100 px-4 py-3 text-sm last:border-b-0 dark:border-neutral-800',
        catalogRevealedHiddenClass(course, showHiddenRevealed),
        sortable ? 'touch-none cursor-grab bg-white active:cursor-grabbing dark:bg-neutral-900' : 'bg-white dark:bg-neutral-900',
        sortable?.isDragging ? 'relative z-20 shadow-md ring-2 ring-indigo-400/40' : '',
      ]
        .filter(Boolean)
        .join(' ')}
      onPointerDownCapture={sortable?.onPointerDownCapture}
      {...(sortable ? sortable.listeners : {})}
    >
      <div className="flex min-w-0 items-start gap-2">
        <div className="min-w-0 flex-1">
          <div className={`flex flex-wrap items-center gap-2 ${invitationMutedClass}`}>
            <CourseCatalogNicknameEditor
              course={course}
              titleClassName="font-semibold text-slate-900 dark:text-neutral-100"
              openRequest={renameRequest}
              onNicknameChange={onNicknameChange}
            />
            {showHiddenRevealed && isUserCatalogHidden(course) ? <CourseCatalogHiddenBadge /> : null}
          </div>
          {invitationPending && course.viewerPendingEnrollmentId ? (
            <CourseEnrollmentInvitationActions
              compact
              courseCode={course.courseCode}
              enrollmentId={course.viewerPendingEnrollmentId}
              onResolved={(approved) => onInvitationResolved?.(course.id, approved)}
            />
          ) : (
            <Link
              to={courseHref}
              className={`mt-1 inline-block text-xs font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-300 dark:hover:text-indigo-200 ${invitationMutedClass}`}
              onClick={onCatalogLinkClick}
            >
              Open course
            </Link>
          )}
        </div>
        {!invitationPending ? (
          <div className="flex shrink-0 items-start gap-1">
            <CourseCatalogPinButton course={course} variant="inline" onPinnedChange={onPinnedChange} />
            <CourseCatalogActionsMenu
              course={course}
              onPinnedChange={onPinnedChange}
              onHiddenChange={onHiddenChange}
              onRenameRequest={onRenameRequest}
            />
          </div>
        ) : null}
      </div>
      <div className={`self-center ${invitationMutedClass}`}>
        <CourseCatalogStatusPill label={invitationPending ? 'Invitation' : badgeLabel} />
      </div>
      <span className={`self-center truncate text-slate-600 dark:text-neutral-400 ${invitationMutedClass}`}>
        {formatCourseTermLabel(course)}
      </span>
      <span className={`self-center whitespace-nowrap text-xs text-slate-500 dark:text-neutral-400 ${invitationMutedClass}`}>
        {formatRelativeCompact(course.updatedAt)}
      </span>
      <span className={`self-center truncate font-mono text-xs text-slate-500 dark:text-neutral-400 ${invitationMutedClass}`}>
        {course.courseCode}
      </span>
    </article>
  )
}

function SortableCourseTableRow({
  course,
  catalogDragActive,
  suppressNavigateAfterDragRef,
  showHiddenRevealed,
  onNicknameChange,
  onPinnedChange,
  onHiddenChange,
  onInvitationResolved,
}: {
  course: CoursePublic
} & CatalogCourseDragProps & {
  showHiddenRevealed: boolean
  onNicknameChange: CatalogNicknameChangeHandler
  onPinnedChange: CatalogPinnedChangeHandler
  onHiddenChange: CatalogHiddenChangeHandler
  onInvitationResolved?: CatalogInvitationResolvedHandler
}) {
  const [renameRequest, setRenameRequest] = useState(0)
  const { listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: course.id,
  })
  const { pointerGuardRef, justFinishedDraggingRef, onPointerDownCapture } =
    useCatalogSortablePointerGuard(isDragging)
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.92 : undefined,
    zIndex: isDragging ? 20 : undefined,
  }

  return (
    <CourseTableRow
      course={course}
      catalogDragActive={catalogDragActive}
      suppressNavigateAfterDragRef={suppressNavigateAfterDragRef}
      showHiddenRevealed={showHiddenRevealed}
      renameRequest={renameRequest}
      onRenameRequest={() => setRenameRequest((n) => n + 1)}
      onNicknameChange={onNicknameChange}
      onPinnedChange={onPinnedChange}
      onHiddenChange={onHiddenChange}
      onInvitationResolved={onInvitationResolved}
      sortable={{
        listeners: listeners as Record<string, unknown>,
        setNodeRef,
        style,
        isDragging,
        pointerGuardRef,
        justFinishedDraggingRef,
        onPointerDownCapture,
      }}
    />
  )
}

export default function Courses() {
  const { allows, loading: permLoading } = usePermissions()
  const showCourseCreateActions = canCreateCourses(allows, permLoading)
  const coursesRevision = useCoursesRevision()
  const bumpCoursesRevision = useBumpCoursesRevision()
  const { open: openCanvasImport } = useCanvasImport()
  const [importFromCourseOpen, setImportFromCourseOpen] = useState(false)
  const [courses, setCourses] = useState<CoursePublic[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [termFilter, setTermFilter] = useState<string>('')
  const [termList, setTermList] = useState<OrgTerm[]>([])
  const [gradeLevelFilter, setGradeLevelFilter] = useState<string>('')
  const [showHidden, setShowHidden] = useState(false)
  const [catalogView, setCatalogView] = useState<CourseCatalogView>('cards')
  const [kanbanColumnLabels, setKanbanColumnLabels] = useState<KanbanColumnLabels>(DEFAULT_KANBAN_COLUMN_LABELS)
  const [hiddenColumnExpanded, setHiddenColumnExpanded] = useState(false)
  const [orgType, setOrgType] = useState<OrgType>('higher-ed')
  const orgId = decodeJwtPayload(getAccessToken())?.org_id ?? ''

  useEffect(() => {
    if (!orgId) return
    let cancelled = false
    void fetchOrgTerms(orgId)
      .then((t) => {
        if (!cancelled) setTermList(t)
      })
      .catch(() => {
        if (!cancelled) setTermList([])
      })
    return () => {
      cancelled = true
    }
  }, [orgId])

  useEffect(() => {
    if (!orgId) return
    let cancelled = false
    void fetchOrgType(orgId)
      .then((t) => {
        if (!cancelled) setOrgType(t)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [orgId])
  /** After a catalog drag, the browser may emit a click on the card link; block that navigation. */
  const suppressNavigateAfterDragRef = useRef(false)
  const [catalogDragActive, setCatalogDragActive] = useState(false)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        await migrateLegacyCourseCatalogLocalStorage()
        const settings = await fetchCourseCatalogSettings()
        if (cancelled) return
        setCatalogView(settings.view)
        setKanbanColumnLabels(settings.kanbanColumnLabels)
        setHiddenColumnExpanded(settings.hiddenColumnExpanded)
      } catch {
        /* keep defaults */
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const handleCatalogViewChange = useCallback((next: CourseCatalogView) => {
    setCatalogView(next)
    void putCourseCatalogSettings({ view: next }).catch(() => {
      setError('Could not save catalog view preference.')
    })
  }, [])

  const handleNicknameChange = useCallback((courseId: string, nickname: string | null) => {
    setCourses((prev) =>
      prev?.map((course) => (course.id === courseId ? { ...course, catalogNickname: nickname } : course)) ?? prev,
    )
  }, [])

  const handlePinnedChange = useCallback((courseId: string, pinned: boolean) => {
    setCourses(
      (prev) =>
        prev?.map((course) => (course.id === courseId ? { ...course, catalogPinned: pinned } : course)) ?? prev,
    )
  }, [])

  const handleHiddenChange = useCallback((courseId: string, hidden: boolean) => {
    setCourses(
      (prev) =>
        prev?.map((course) =>
          course.id === courseId
            ? {
                ...course,
                catalogHidden: hidden,
                catalogPinned: hidden ? false : course.catalogPinned,
                kanbanColumnId: hidden ? null : course.kanbanColumnId,
                kanbanSortOrder: hidden ? null : course.kanbanSortOrder,
              }
            : course,
        ) ?? prev,
    )
  }, [])

  const handleInvitationResolved = useCallback((courseId: string, approved: boolean) => {
    if (!approved) {
      setCourses((prev) => prev?.filter((course) => course.id !== courseId) ?? prev)
      return
    }
    setCourses(
      (prev) =>
        prev?.map((course) =>
          course.id === courseId
            ? {
                ...course,
                viewerEnrollmentInvitationPending: false,
                viewerPendingEnrollmentId: null,
              }
            : course,
        ) ?? prev,
    )
  }, [])

  const handleKanbanColumnLabelsChange = useCallback((labels: KanbanColumnLabels) => {
    setKanbanColumnLabels(labels)
    void putCourseCatalogSettings({ kanbanColumnLabels: labels }).catch(() => {
      setError('Could not save kanban column names.')
    })
  }, [])

  const handleHiddenColumnExpandedChange = useCallback((expanded: boolean) => {
    setHiddenColumnExpanded(expanded)
    void putCourseCatalogSettings({ hiddenColumnExpanded: expanded }).catch(() => {
      setError('Could not save kanban column preference.')
    })
  }, [])

  const handleKanbanBoardChange = useCallback(async (columns: Record<KanbanColumnId, string[]>) => {
    await putCourseKanbanBoard(columns)
    setCourses((prev) => {
      if (!prev) return prev
      const placementById = new Map<string, { columnId: KanbanColumnId; sortOrder: number }>()
      const hiddenIds = new Set(columns.hidden ?? [])
      for (const columnId of Object.keys(columns) as KanbanColumnId[]) {
        if (columnId === 'hidden') continue
        columns[columnId].forEach((courseId, sortOrder) => {
          placementById.set(courseId, { columnId, sortOrder })
        })
      }
      return prev.map((course) => {
        const hidden = hiddenIds.has(course.id)
        const placement = placementById.get(course.id)
        if (hidden) {
          return {
            ...course,
            catalogHidden: true,
            catalogPinned: false,
            kanbanColumnId: null,
            kanbanSortOrder: null,
          }
        }
        if (!placement) {
          return {
            ...course,
            catalogHidden: false,
            kanbanColumnId: null,
            kanbanSortOrder: null,
          }
        }
        return {
          ...course,
          catalogHidden: false,
          kanbanColumnId: placement.columnId,
          kanbanSortOrder: placement.sortOrder,
        }
      })
    })
  }, [])

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      setError(null)
      try {
        const params = new URLSearchParams()
        if (termFilter) params.set('term_id', termFilter)
        if (gradeLevelFilter) params.set('grade_level', gradeLevelFilter)
        const qs = params.toString() ? `?${params.toString()}` : ''
        const res = await authorizedFetch(`/api/v1/courses${qs}`)
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok) {
          setCourses([])
          setError(readApiErrorMessage(raw))
          return
        }
        const data = raw as { courses?: CoursePublic[] }
        if (!cancelled) setCourses(data.courses ?? [])
      } catch {
        if (!cancelled) {
          setCourses([])
          setError('Could not load courses. Is the API running?')
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [termFilter, gradeLevelFilter, coursesRevision])

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(SharedKeyboardSensor, defaultKeyboardSensorOptions),
  )

  const hiddenCount = useMemo(() => countUserHiddenCourses(courses ?? []), [courses])
  const visibleCourses = useMemo(
    () => filterCatalogCourses(courses ?? [], showHidden),
    [courses, showHidden],
  )
  const courseIds = useMemo(() => visibleCourses.map((c) => c.id), [visibleCourses])
  const emptyStateKind = useMemo(
    () => catalogEmptyStateKind(courses, showHidden),
    [courses, showHidden],
  )

  const catalogSections = useMemo((): CatalogSection[] | null => {
    if (!courses?.length || catalogView === 'status') return null
    return buildCatalogSections(courses, termList, { termFilter, showHidden })
  }, [courses, termFilter, termList, catalogView, showHidden])

  const clearSuppressNavigateAfterDragSoon = useCallback(() => {
    window.setTimeout(() => {
      suppressNavigateAfterDragRef.current = false
    }, 300)
  }, [])

  const finishCatalogDrag = useCallback(() => {
    suppressNavigateAfterDragRef.current = true
    clearSuppressNavigateAfterDragSoon()
    window.setTimeout(() => {
      setCatalogDragActive(false)
    }, 0)
  }, [clearSuppressNavigateAfterDragSoon])

  const handleDragStart = useCallback((_event: DragStartEvent) => {
    suppressNavigateAfterDragRef.current = true
    setCatalogDragActive(true)
  }, [])

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event
      finishCatalogDrag()
      if (!over || active.id === over.id || visibleCourses.length === 0) return
      setError(null)
      const oldIndex = visibleCourses.findIndex((c) => c.id === active.id)
      const newIndex = visibleCourses.findIndex((c) => c.id === over.id)
      if (oldIndex < 0 || newIndex < 0) return
      const previous = courses
      const reorderedVisible = arrayMove(visibleCourses, oldIndex, newIndex)
      const visibleIdOrder = reorderedVisible.map((c) => c.id)
      const hiddenTail = (courses ?? []).filter((c) => !visibleIdOrder.includes(c.id))
      const next = [...reorderedVisible, ...hiddenTail]
      setCourses(next)
      void putCourseCatalogOrder(next.map((c) => c.id)).catch(() => {
        setCourses(previous)
        setError('Could not save course order. Try again.')
      })
    },
    [courses, finishCatalogDrag, visibleCourses],
  )

  const handleDragCancel = useCallback(() => {
    finishCatalogDrag()
  }, [finishCatalogDrag])

  const sortStrategy = catalogViewUsesGrid(catalogView) ? rectSortingStrategy : verticalListSortingStrategy

  const renderSortableCourse = useCallback(
    (course: CoursePublic) => {
      const shared = {
        catalogDragActive,
        suppressNavigateAfterDragRef,
        showHiddenRevealed: showHidden,
        onNicknameChange: handleNicknameChange,
        onPinnedChange: handlePinnedChange,
        onHiddenChange: handleHiddenChange,
        onInvitationResolved: handleInvitationResolved,
      }
      switch (catalogView) {
        case 'cards':
          return <SortableCourseCard key={course.id} course={course} {...shared} />
        case 'gallery':
          return <SortableCourseGalleryTile key={course.id} course={course} {...shared} />
        case 'table':
          return <SortableCourseTableRow key={course.id} course={course} {...shared} />
        case 'list':
          return <SortableCourseListRow key={course.id} course={course} {...shared} />
        case 'status':
          return null
        default: {
          const _exhaustive: never = catalogView
          return _exhaustive
        }
      }
    },
    [
      catalogDragActive,
      catalogView,
      handleHiddenChange,
      handleInvitationResolved,
      handleNicknameChange,
      handlePinnedChange,
      showHidden,
    ],
  )

  const renderCourseItems = useCallback(
    (items: CoursePublic[], marginClass = 'mt-4') => {
      switch (catalogView) {
        case 'cards':
          return (
            <div className={`${marginClass} grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4`}>
              {items.map((c) => renderSortableCourse(c))}
            </div>
          )
        case 'gallery':
          return (
            <div
              className={`${marginClass} grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6`}
            >
              {items.map((c) => renderSortableCourse(c))}
            </div>
          )
        case 'table':
          return (
            <div
              className={`${marginClass} overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900`}
            >
              <div className="min-w-[42rem]">
                <CourseCatalogTableHeader />
                {items.map((c) => renderSortableCourse(c))}
              </div>
            </div>
          )
        case 'list':
          return <div className={`${marginClass} flex flex-col gap-2`}>{items.map((c) => renderSortableCourse(c))}</div>
        case 'status':
          return null
        default: {
          const _exhaustive: never = catalogView
          return _exhaustive
        }
      }
    },
    [catalogView, renderSortableCourse],
  )

  return (
    <LmsPage
      title="Courses"
      description="Browse and open your enrolled courses. Drag to reorder your catalog."
      actions={
        <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
          {showCourseCreateActions ? (
            <CourseCatalogImportMenu
              onImportCanvas={openCanvasImport}
              onImportFromCourse={() => setImportFromCourseOpen(true)}
            />
          ) : null}
          <CourseCatalogViewMenu value={catalogView} onChange={handleCatalogViewChange} />
          {showCourseCreateActions ? (
            <Link
              to="/courses/create"
              className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500"
            >
              <Plus className="h-4 w-4" aria-hidden />
              New course
            </Link>
          ) : null}
        </div>
      }
    >
      {error && (
        <p className="mt-6 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800">
          {error}
        </p>
      )}

      <div className="mt-6 flex flex-wrap items-end gap-4">
        {hiddenCount > 0 ? (
          <button
            type="button"
            aria-pressed={showHidden}
            onClick={() => setShowHidden((value) => !value)}
            className={[
              'inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-semibold shadow-sm transition-[background-color,color,border-color]',
              showHidden
                ? 'border-indigo-300 bg-indigo-50 text-indigo-700 dark:border-indigo-500/40 dark:bg-indigo-950/40 dark:text-indigo-200'
                : 'border-slate-200 bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:border-neutral-600 dark:hover:bg-neutral-800',
            ].join(' ')}
          >
            {showHidden ? <Eye className="h-4 w-4" aria-hidden /> : <EyeOff className="h-4 w-4" aria-hidden />}
            Show hidden ({hiddenCount})
          </button>
        ) : null}
        {orgId && termList.length > 0 && (
          <div className="min-w-48 max-w-sm flex-1">
            <label htmlFor="course-catalog-term-filter" className="text-sm font-medium text-slate-700 dark:text-neutral-200">
              Term
            </label>
            <select
              id="course-catalog-term-filter"
              value={termFilter}
              onChange={(e) => setTermFilter(e.target.value)}
              className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 shadow-sm outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
              aria-label="Filter courses by academic term"
            >
              <option value="">All terms</option>
              {termList.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>
        )}
        {orgType === 'k-12' && (
          <div className="min-w-48 max-w-sm flex-1">
            <label htmlFor="course-catalog-grade-filter" className="text-sm font-medium text-slate-700 dark:text-neutral-200">
              Grade level
            </label>
            <select
              id="course-catalog-grade-filter"
              value={gradeLevelFilter}
              onChange={(e) => setGradeLevelFilter(e.target.value)}
              className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 shadow-sm outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
              aria-label="Filter courses by grade level"
            >
              <option value="">All grade levels</option>
              <option value="K">Kindergarten</option>
              <option value="1">Grade 1</option>
              <option value="2">Grade 2</option>
              <option value="3">Grade 3</option>
              <option value="4">Grade 4</option>
              <option value="5">Grade 5</option>
              <option value="6">Grade 6</option>
              <option value="7">Grade 7</option>
              <option value="8">Grade 8</option>
              <option value="9">Grade 9</option>
              <option value="10">Grade 10</option>
              <option value="11">Grade 11</option>
              <option value="12">Grade 12</option>
              <option value="K-2">K–2 (multi-grade)</option>
              <option value="3-5">3–5 (multi-grade)</option>
              <option value="6-8">6–8 (multi-grade)</option>
              <option value="9-12">9–12 (multi-grade)</option>
              <option value="K-12">K–12 (all grades)</option>
            </select>
          </div>
        )}
      </div>

      {courses === null && !error && <CoursesCatalogSkeleton />}

      {emptyStateKind === 'none' && !error && (
        <div className="mt-8">
          {showCourseCreateActions ? (
            <EmptyState
              icon={CreateCourseIcon}
              title="Create your first course"
              body="You do not have any courses in your catalog yet. Add a title and description to get started, then invite learners from the course dashboard."
              primaryAction={{ label: 'New course', to: '/courses/create' }}
            />
          ) : (
            <EmptyState
              icon={BookOpen}
              title="No courses yet"
              body="You are not enrolled in any published courses. When an instructor adds you, the course will appear here."
            />
          )}
        </div>
      )}

      {emptyStateKind === 'all-hidden' && !error && (
        <div className="mt-8">
          <EmptyState
            icon={EyeOff}
            title="All your courses are hidden"
            body="You hid every course from this page. Show hidden courses to bring them back, or open a course from a direct link."
            primaryAction={{
              label: 'Show hidden',
              onClick: () => setShowHidden(true),
            }}
          />
        </div>
      )}

      {courses && courses.length > 0 && catalogView === 'status' && (
        <CourseCatalogKanbanBoard
          courses={courses}
          columnLabels={kanbanColumnLabels}
          hiddenColumnExpanded={hiddenColumnExpanded}
          onHiddenColumnExpandedChange={handleHiddenColumnExpandedChange}
          onColumnLabelsChange={handleKanbanColumnLabelsChange}
          onNicknameChange={handleNicknameChange}
          onPinnedChange={handlePinnedChange}
          onHiddenChange={handleHiddenChange}
          onBoardChange={handleKanbanBoardChange}
        />
      )}

      {emptyStateKind === 'has-visible' && catalogView !== 'status' && catalogSections && (
        <DndContext
          id={COURSE_GRID_SORT_ID}
          sensors={sensors}
          collisionDetection={closestCorners}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={handleDragCancel}
        >
          <SortableContext items={courseIds} strategy={sortStrategy}>
            <div className="mt-8 space-y-10">
              {catalogSections.map((sec) => (
                <section key={sec.key} aria-labelledby={`cat-${sec.key}`}>
                  <h2
                    id={`cat-${sec.key}`}
                    className="text-base font-semibold text-slate-900 dark:text-neutral-100"
                  >
                    {sec.title}
                  </h2>
                  {renderCourseItems(sec.items)}
                </section>
              ))}
            </div>
          </SortableContext>
        </DndContext>
      )}

      {emptyStateKind === 'has-visible' && catalogView !== 'status' && !catalogSections && (
        <DndContext
          id={COURSE_GRID_SORT_ID}
          sensors={sensors}
          collisionDetection={closestCorners}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={handleDragCancel}
        >
          <SortableContext items={courseIds} strategy={sortStrategy}>
            {renderCourseItems(visibleCourses, 'mt-8')}
          </SortableContext>
        </DndContext>
      )}

      <CourseCatalogImportFromCourseModal
        open={importFromCourseOpen}
        courses={courses ?? []}
        onClose={() => setImportFromCourseOpen(false)}
        onImported={() => {
          bumpCoursesRevision()
        }}
      />
    </LmsPage>
  )
}

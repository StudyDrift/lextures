import { Pin } from 'lucide-react'
import { toPinnedSummary, useCoursePins } from '../../context/course-pinned-context'
import type { CoursePublic } from '../../lib/courses-api'

type PinButtonCourse = Pick<
  CoursePublic,
  'id' | 'courseCode' | 'title' | 'heroImageUrl' | 'heroImageObjectPosition' | 'catalogNickname' | 'catalogPinned'
>

export function CourseCatalogPinButton({
  course,
  className = '',
  variant = 'overlay',
  onPinnedChange,
}: {
  course: PinButtonCourse
  className?: string
  /** `overlay` for hero images; `inline` for list/table/kanban rows. */
  variant?: 'overlay' | 'inline'
  onPinnedChange?: (courseId: string, pinned: boolean) => void
}) {
  const { togglePin, togglingCourseId } = useCoursePins()
  const pinned = Boolean(course.catalogPinned)
  const displayTitle = course.catalogNickname?.trim() || course.title
  const busy = togglingCourseId === course.id

  return (
    <button
      type="button"
      disabled={busy}
      onClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        const nextPinned = !pinned
        onPinnedChange?.(course.id, nextPinned)
        void togglePin(course.id, nextPinned, nextPinned ? toPinnedSummary(course) : undefined).catch(() => {
          onPinnedChange?.(course.id, pinned)
        })
      }}
      onPointerDown={(e) => e.stopPropagation()}
      aria-label={pinned ? `Unpin ${displayTitle}` : `Pin ${displayTitle} to sidebar`}
      aria-pressed={pinned}
      title={pinned ? 'Unpin from sidebar' : 'Pin to sidebar'}
      className={[
        'inline-flex h-8 w-8 items-center justify-center rounded-full transition focus:outline-none disabled:opacity-60',
        variant === 'overlay'
          ? [
              'backdrop-blur-sm focus-visible:ring-2 focus-visible:ring-white/80',
              pinned
                ? 'bg-white/95 text-indigo-600 shadow-sm ring-1 ring-white/60 dark:bg-neutral-900/95 dark:text-indigo-300 dark:ring-white/10'
                : 'bg-black/35 text-white/90 hover:bg-black/50 hover:text-white dark:bg-black/45 dark:hover:bg-black/60',
            ].join(' ')
          : [
              'focus-visible:ring-2 focus-visible:ring-indigo-400/40',
              pinned
                ? 'bg-indigo-50 text-indigo-600 ring-1 ring-indigo-200/80 dark:bg-indigo-950/50 dark:text-indigo-300 dark:ring-indigo-500/30'
                : 'text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:text-neutral-500 dark:hover:bg-neutral-800 dark:hover:text-neutral-200',
            ].join(' '),
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <Pin
        className="h-4 w-4"
        strokeWidth={pinned ? 2.25 : 1.75}
        fill={pinned ? 'currentColor' : 'none'}
        aria-hidden
      />
    </button>
  )
}
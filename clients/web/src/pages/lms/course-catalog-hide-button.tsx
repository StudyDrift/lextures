import { Eye, EyeOff } from 'lucide-react'
import { useCourseHidden } from '../../context/course-hidden-context'
import type { CoursePublic } from '../../lib/courses-api'


type HideButtonCourse = Pick<CoursePublic, 'id' | 'title' | 'catalogNickname' | 'catalogHidden' | 'catalogPinned'>

export function CourseCatalogHideButton({
  course,
  className = '',
  onHiddenChange,
}: {
  course: HideButtonCourse
  className?: string
  onHiddenChange?: (courseId: string, hidden: boolean) => void
}) {
  const { toggleHidden, togglingCourseId } = useCourseHidden()
  const hidden = Boolean(course.catalogHidden)
  const displayTitle = course.catalogNickname?.trim() || course.title
  const busy = togglingCourseId === course.id

  return (
    <button
      type="button"
      disabled={busy}
      onClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        const nextHidden = !hidden
        onHiddenChange?.(course.id, nextHidden)
        void toggleHidden(course.id, nextHidden, course.catalogPinned).catch(() => {
          onHiddenChange?.(course.id, hidden)
        })
      }}
      onPointerDown={(e) => e.stopPropagation()}
      aria-label={
        hidden
          ? `Show ${displayTitle} in your catalog`
          : `Hide ${displayTitle} from your catalog`
      }
      className={[
        'inline-flex h-8 w-full items-center gap-2 rounded-md px-2 text-start text-sm transition-colors hover:bg-slate-50 disabled:opacity-60 dark:hover:bg-neutral-700',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {hidden ? (
        <Eye className="h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
      ) : (
        <EyeOff className="h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
      )}
      <span>{hidden ? 'Show in catalog' : 'Hide from catalog'}</span>
    </button>
  )
}
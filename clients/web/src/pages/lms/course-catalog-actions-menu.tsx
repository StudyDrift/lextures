import { useEffect, useId, useRef, useState } from 'react'
import { MoreHorizontal, Pencil, Pin } from 'lucide-react'
import type { CoursePublic } from '../../lib/courses-api'
import { toPinnedSummary, useCoursePins } from '../../context/course-pinned-context'

import { CourseCatalogHideButton } from './course-catalog-hide-button'

type ActionsMenuCourse = Pick<
  CoursePublic,
  'id' | 'courseCode' | 'title' | 'catalogNickname' | 'catalogPinned' | 'catalogHidden' | 'heroImageUrl' | 'heroImageObjectPosition'
>

type Props = {
  course: ActionsMenuCourse
  variant?: 'overlay' | 'inline'
  onPinnedChange?: (courseId: string, pinned: boolean) => void
  onHiddenChange?: (courseId: string, hidden: boolean) => void
  onRenameRequest?: () => void
  className?: string
}

export function CourseCatalogActionsMenu({
  course,
  variant = 'inline',
  onPinnedChange,
  onHiddenChange,
  onRenameRequest,
  className = '',
}: Props) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()
  const { togglePin, togglingCourseId } = useCoursePins()
  const displayTitle = course.catalogNickname?.trim() || course.title
  const pinned = Boolean(course.catalogPinned)
  const pinBusy = togglingCourseId === course.id

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  const triggerClass =
    variant === 'overlay'
      ? 'bg-black/35 text-white/90 backdrop-blur-sm hover:bg-black/50 hover:text-white dark:bg-black/45 dark:hover:bg-black/60'
      : 'text-fg-subtle hover:bg-surface-sunken hover:text-fg-muted dark:hover:bg-surface-overlay dark:hover:text-fg-default'

  return (
    <div ref={rootRef} className={`relative ${className}`}>
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={`Actions for ${displayTitle}`}
        onClick={(e) => {
          e.preventDefault()
          e.stopPropagation()
          setOpen((value) => !value)
        }}
        onPointerDown={(e) => e.stopPropagation()}
        className={[
          'inline-flex h-8 w-8 items-center justify-center rounded-full transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-400/40',
          triggerClass,
        ].join(' ')}
      >
        <MoreHorizontal className="h-4 w-4" aria-hidden />
      </button>

      {open ? (
        <div
          id={menuId}
          role="menu"
          aria-label={`Actions for ${displayTitle}`}
          className="absolute end-0 z-50 mt-1 min-w-[12rem] overflow-hidden rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg shadow-slate-900/10 dark:border-border-default dark:bg-surface-overlay dark:shadow-black/40"
          onPointerDown={(e) => e.stopPropagation()}
        >
          <button
            type="button"
            role="menuitem"
            disabled={pinBusy}
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              const nextPinned = !pinned
              onPinnedChange?.(course.id, nextPinned)
              void togglePin(course.id, nextPinned, nextPinned ? toPinnedSummary(course) : undefined).catch(() => {
                onPinnedChange?.(course.id, pinned)
              })
              setOpen(false)
            }}
            className="flex w-full items-center gap-2 px-2 py-2 text-start text-sm transition-colors hover:bg-surface-base disabled:opacity-60 dark:hover:bg-neutral-700"
          >
            <Pin className="h-4 w-4 shrink-0 text-fg-muted" aria-hidden />
            <span>{pinned ? 'Unpin from sidebar' : 'Pin to sidebar'}</span>
          </button>
          {onRenameRequest ? (
            <button
              type="button"
              role="menuitem"
              onClick={(e) => {
                e.preventDefault()
                e.stopPropagation()
                onRenameRequest()
                setOpen(false)
              }}
              className="flex w-full items-center gap-2 px-2 py-2 text-start text-sm transition-colors hover:bg-surface-base dark:hover:bg-neutral-700"
            >
              <Pencil className="h-4 w-4 shrink-0 text-fg-muted" aria-hidden />
              <span>Rename</span>
            </button>
          ) : null}
          <div role="none" className="my-1 border-t border-border-subtle dark:border-border-default" />
          <div role="presentation" className="px-0.5">
            <CourseCatalogHideButton
              course={course}
              onHiddenChange={(courseId, hidden) => {
                onHiddenChange?.(courseId, hidden)
                if (hidden) setOpen(false)
              }}
            />
          </div>
        </div>
      ) : null}
    </div>
  )
}
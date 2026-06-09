import { useEffect, useId, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react'
import { Pencil } from 'lucide-react'
import type { CoursePublic } from '../../lib/courses-api'
import { putCourseCatalogNickname } from '../../lib/course-catalog-settings-api'
import { courseCatalogDisplayTitle, courseCatalogHasNickname } from './course-catalog-display'

function stopDragKeyboardPropagation(e: KeyboardEvent | PointerEvent) {
  e.stopPropagation()
}

type Props = {
  course: CoursePublic
  className?: string
  titleClassName?: string
  /** Hero/tile already shows the display title — keep footer to official title + edit only. */
  compact?: boolean
  onNicknameChange: (courseId: string, nickname: string | null) => void
}

export function CourseCatalogNicknameEditor({
  course,
  className = '',
  titleClassName = '',
  compact = false,
  onNicknameChange,
}: Props) {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState(course.catalogNickname ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const inputId = useId()
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!open) return
    inputRef.current?.focus()
    inputRef.current?.select()
  }, [open])

  async function save() {
    const trimmed = draft.trim()
    const next = trimmed.length > 0 ? trimmed : null
    const current = course.catalogNickname?.trim() || null
    if (next === current) {
      setOpen(false)
      setError(null)
      return
    }
    setSaving(true)
    setError(null)
    try {
      await putCourseCatalogNickname(course.id, next)
      onNicknameChange(course.id, next)
      setOpen(false)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Could not save nickname.')
    } finally {
      setSaving(false)
    }
  }

  const displayTitle = courseCatalogDisplayTitle(course)
  const hasNickname = courseCatalogHasNickname(course)

  const editButton = (
    <button
      type="button"
      onClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        setDraft(course.catalogNickname ?? '')
        setError(null)
        setOpen(true)
      }}
      className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-500 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
      aria-label={`Edit nickname for ${course.title}`}
      title="Edit nickname"
    >
      <Pencil className="h-3.5 w-3.5" aria-hidden />
    </button>
  )

  return (
    <div className={className}>
      {compact ? (
        <div className="flex items-center gap-1.5">
          {hasNickname ? (
            <p className="min-w-0 flex-1 text-xs text-slate-500 line-clamp-1 dark:text-neutral-400">{course.title}</p>
          ) : (
            <span className="min-w-0 flex-1 text-xs text-slate-500 dark:text-neutral-400">Add nickname</span>
          )}
          {editButton}
        </div>
      ) : (
        <>
          <div className="flex items-start gap-1.5">
            <span className={titleClassName}>{displayTitle}</span>
            {editButton}
          </div>
          {hasNickname ? (
            <p className="mt-0.5 text-xs text-slate-500 line-clamp-1 dark:text-neutral-400">{course.title}</p>
          ) : null}
        </>
      )}
      {open ? (
        <div
          className="mt-2 space-y-2 rounded-lg border border-slate-200 bg-white p-2 shadow-sm dark:border-neutral-600 dark:bg-neutral-900"
          onClick={(e) => {
            e.preventDefault()
            e.stopPropagation()
          }}
          onPointerDown={stopDragKeyboardPropagation}
          onKeyDown={stopDragKeyboardPropagation}
        >
          <label htmlFor={inputId} className="sr-only">
            Nickname for {course.title}
          </label>
          <input
            id={inputId}
            ref={inputRef}
            value={draft}
            disabled={saving}
            maxLength={120}
            placeholder={course.title}
            onChange={(e) => setDraft(e.target.value)}
            onPointerDown={stopDragKeyboardPropagation}
            onKeyDown={(e) => {
              stopDragKeyboardPropagation(e)
              if (e.key === 'Enter') {
                e.preventDefault()
                void save()
              }
              if (e.key === 'Escape') {
                e.preventDefault()
                setOpen(false)
                setError(null)
              }
            }}
            className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
          />
          {error ? <p className="text-xs text-rose-600 dark:text-rose-300">{error}</p> : null}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              disabled={saving}
              onClick={() => {
                setOpen(false)
                setError(null)
              }}
              className="rounded-md px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={saving}
              onClick={() => void save()}
              className="rounded-md bg-indigo-600 px-2 py-1 text-xs font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
            >
              Save
            </button>
          </div>
        </div>
      ) : null}
    </div>
  )
}

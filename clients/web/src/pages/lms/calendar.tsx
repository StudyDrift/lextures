import { useCallback, useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { authorizedFetch } from '../../lib/api'
import { parseCalendarDateFromQuery } from '../../lib/command-palette-go-to'
import { readApiErrorMessage } from '../../lib/errors'
import {
  fetchCourseStructure,
  type CoursePublic,
  type CourseStructureItem,
} from '../../lib/courses-api'
import { CalendarActionsMenu } from './calendar-actions-menu'
import { CalendarCoursesViewMenu } from './calendar-courses-view-menu'
import { CourseCalendar, type CourseCalendarAssignment } from './course-calendar'
import { LmsPage } from './lms-page'

const LS_DISABLED_KEY = 'lextures.globalCalendar.disabledCourseIds'

function readDisabledCourseIdsFromStorage(): string[] | null {
  try {
    const raw = window.localStorage.getItem(LS_DISABLED_KEY)
    if (raw == null) return null
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return null
    return parsed.filter((x): x is string => typeof x === 'string')
  } catch {
    return null
  }
}

function writeDisabledCourseIdsToStorage(ids: string[]) {
  try {
    window.localStorage.setItem(LS_DISABLED_KEY, JSON.stringify(ids))
  } catch {
    /* ignore quota / private mode */
  }
}

function mergeDisabledIds(eligible: CoursePublic[], stored: string[] | null): Set<string> {
  const eligibleIds = new Set(eligible.map((c) => c.id))
  if (stored === null) {
    return new Set()
  }
  const out = new Set<string>()
  for (const id of stored) {
    if (eligibleIds.has(id)) out.add(id)
  }
  return out
}

function structureToAssignments(
  course: CoursePublic,
  items: CourseStructureItem[],
  paletteIndex: number,
): CourseCalendarAssignment[] {
  const isDueCalendarItem = (
    i: CourseStructureItem,
  ): i is CourseStructureItem & {
    kind: 'content_page' | 'assignment' | 'quiz'
    dueAt: string
  } =>
    (i.kind === 'content_page' || i.kind === 'assignment' || i.kind === 'quiz') && Boolean(i.dueAt)

  const title = course.title.trim() || course.courseCode
  return items.filter(isDueCalendarItem).map((i) => ({
    id: i.id,
    title: i.title,
    dueAt: i.dueAt,
    kind: i.kind,
    pointsWorth: i.pointsWorth,
    pointsPossible: i.pointsPossible,
    isAdaptive: i.isAdaptive,
    linkCourseCode: course.courseCode,
    courseTitle: title,
    courseLabel: course.courseCode,
    paletteIndex,
  }))
}

export default function Calendar() {
  const [searchParams] = useSearchParams()
  const rawDate = searchParams.get('date')?.trim() ?? ''
  const dateKey = useMemo(() => parseCalendarDateFromQuery(rawDate), [rawDate])

  const [courses, setCourses] = useState<CoursePublic[] | null>(null)
  const [coursesError, setCoursesError] = useState<string | null>(null)
  const [disabledCourseIds, setDisabledCourseIds] = useState<Set<string>>(() => new Set())

  const [structureByCourseId, setStructureByCourseId] = useState<
    Record<string, CourseStructureItem[] | null>
  >({})
  const [structureErrors, setStructureErrors] = useState<Record<string, string>>({})
  const [structuresLoading, setStructuresLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      setCoursesError(null)
      try {
        const res = await authorizedFetch('/api/v1/courses')
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok) {
          if (!cancelled) {
            setCourses([])
            setCoursesError(readApiErrorMessage(raw))
            setDisabledCourseIds(new Set())
          }
          return
        }
        const data = raw as { courses?: CoursePublic[] }
        const list = data.courses ?? []
        if (!cancelled) {
          setCourses(list)
          const eligible = list.filter((c) => !c.archived && c.calendarEnabled !== false)
          setDisabledCourseIds(mergeDisabledIds(eligible, readDisabledCourseIdsFromStorage()))
        }
      } catch {
        if (!cancelled) {
          setCourses([])
          setCoursesError('Could not load courses.')
          setDisabledCourseIds(new Set())
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const eligibleCourses = useMemo(
    () => (courses ?? []).filter((c) => !c.archived && c.calendarEnabled !== false),
    [courses],
  )

  const enabledCourses = useMemo(
    () => eligibleCourses.filter((c) => !disabledCourseIds.has(c.id)),
    [eligibleCourses, disabledCourseIds],
  )

  const paletteIndexByCourseId = useMemo(() => {
    const m = new Map<string, number>()
    eligibleCourses.forEach((c, i) => m.set(c.id, i))
    return m
  }, [eligibleCourses])

  useEffect(() => {
    let cancelled = false
    const targets = enabledCourses

    void (async () => {
      if (targets.length === 0) {
        await Promise.resolve()
        if (cancelled) return
        setStructuresLoading(false)
        setStructureByCourseId({})
        setStructureErrors({})
        return
      }

      setStructuresLoading(true)
      setStructureErrors({})

      const next: Record<string, CourseStructureItem[] | null> = {}
      const errs: Record<string, string> = {}

      await Promise.all(
        targets.map(async (c) => {
          try {
            const items = await fetchCourseStructure(c.courseCode)
            if (!cancelled) next[c.id] = items
          } catch (e) {
            if (!cancelled) {
              next[c.id] = null
              errs[c.id] = e instanceof Error ? e.message : 'Could not load calendar.'
            }
          }
        }),
      )

      if (cancelled) return
      setStructureByCourseId(next)
      setStructureErrors(errs)
      setStructuresLoading(false)
    })()

    return () => {
      cancelled = true
    }
  }, [enabledCourses])

  const mergedAssignments: CourseCalendarAssignment[] = useMemo(() => {
    const out: CourseCalendarAssignment[] = []
    for (const c of enabledCourses) {
      const items = structureByCourseId[c.id]
      if (!items) continue
      const pi = paletteIndexByCourseId.get(c.id) ?? 0
      out.push(...structureToAssignments(c, items, pi))
    }
    return out
  }, [enabledCourses, structureByCourseId, paletteIndexByCourseId])

  const hasAnyLoadedStructure = useMemo(
    () => enabledCourses.some((c) => Array.isArray(structureByCourseId[c.id])),
    [enabledCourses, structureByCourseId],
  )

  const setCourseEnabled = useCallback((courseId: string, enabled: boolean) => {
    setDisabledCourseIds((prev) => {
      const next = new Set(prev)
      if (enabled) next.delete(courseId)
      else next.add(courseId)
      writeDisabledCourseIdsToStorage([...next])
      return next
    })
  }, [])

  const showAllCourses = useCallback(() => {
    setDisabledCourseIds(() => {
      writeDisabledCourseIdsToStorage([])
      return new Set()
    })
  }, [])

  const hideAllCourses = useCallback(() => {
    setDisabledCourseIds(() => {
      const all = eligibleCourses.map((c) => c.id)
      writeDisabledCourseIdsToStorage(all)
      return new Set(all)
    })
  }, [eligibleCourses])

  const representativeCourseCode = enabledCourses[0]?.courseCode ?? ''

  return (
    <LmsPage
      title="Calendar"
      description="Month, week, and to-do views across your courses. Use View to choose which courses appear on the calendar."
      fillHeight
      actions={
        <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
          {eligibleCourses.length > 0 ? (
            <CalendarCoursesViewMenu
              courses={eligibleCourses}
              disabledCourseIds={disabledCourseIds}
              structureErrors={structureErrors}
              onCourseEnabledChange={setCourseEnabled}
              onShowAll={showAllCourses}
              onHideAll={hideAllCourses}
            />
          ) : null}
          <CalendarActionsMenu scope="global" />
        </div>
      }
    >
      {coursesError && (
        <p className="mt-6 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/50 dark:text-rose-200">
          {coursesError}
        </p>
      )}
      {courses === null && !coursesError && (
        <p className="mt-8 text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
      )}
      {courses && courses.length === 0 && !coursesError && (
        <p className="mt-8 text-sm text-slate-600 dark:text-neutral-300">No courses on your account yet.</p>
      )}
      {courses && courses.length > 0 && eligibleCourses.length === 0 && !coursesError && (
        <p className="mt-8 text-sm text-slate-600 dark:text-neutral-300">
          No enrolled courses have the calendar tool enabled.
        </p>
      )}
      {eligibleCourses.length > 0 ? (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          {enabledCourses.length === 0 ? (
            <p className="rounded-xl border border-dashed border-slate-200 bg-slate-50/80 px-4 py-8 text-center text-sm text-slate-600 dark:border-neutral-700 dark:bg-neutral-950/40 dark:text-neutral-300">
              Turn on at least one course to load its schedule.
            </p>
          ) : structuresLoading && !hasAnyLoadedStructure ? (
            <p className="text-sm text-slate-500 dark:text-neutral-400">Loading calendars…</p>
          ) : (
            <CourseCalendar
              courseCode={representativeCourseCode}
              assignments={mergedAssignments}
              canRescheduleDueByDrag={false}
              initialDateKey={dateKey}
            />
          )}
        </div>
      ) : null}
    </LmsPage>
  )
}

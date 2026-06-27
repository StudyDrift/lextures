import { useCallback, useEffect, useMemo, useState } from 'react'
import { StudentTodoWeekPicker } from '../../components/todos/student-todo-week-picker'
import { StudentTodoViewPicker } from '../../components/todos/student-todo-view-picker'
import { authorizedFetch } from '../../lib/api'
import { mapPool } from '../../lib/async-pool'
import { readApiErrorMessage } from '../../lib/errors'
import {
  courseGradebookViewPermission,
  fetchCourse,
  fetchCourseGradingBacklog,
  fetchCourseStructure,
  viewerIsCourseStaffEnrollment,
  type CoursePublic,
  type CourseStructureItem,
} from '../../lib/courses-api'
import { usePermissions } from '../../context/use-permissions'
import {
  GradingBacklogList,
  type GradingBacklogItem,
} from '../../components/dashboard/grading-backlog-list'
import { StudentTodoKanban } from '../../components/todos/student-todo-kanban'
import { fetchNotebookTasks, patchNotebookTask } from '../../lib/notebook-tasks-api'
import { NOTEBOOK_TASKS_CHANGED } from '../../lib/notebook-task-sync'
import { markNotebookTaskComplete } from '../../lib/student-notebook-storage'
import { fetchStudentTodoBoardPlacements } from '../../lib/student-todo-board-api'
import { collectStudentTodoItems } from '../../lib/student-todo-utils'
import type { StudentTodoItem, StudentTodoPlacement } from '../../lib/student-todo-types'
import { readStoredWeekOffsets, storeWeekOffsets } from '../../lib/student-todo-week'
import { readStoredCollapseEmpty, storeCollapseEmpty } from '../../lib/student-todo-view'
import { useRelativeWeekNow } from '../../lib/use-relative-week-now'
import { LmsPage } from './lms-page'

function hasStudentRole(roles: readonly string[] | undefined): boolean {
  if (!roles?.length) return false
  return roles.some((r) => r.trim().toLowerCase() === 'student')
}

export default function TodosPage() {
  const { allows, loading: permLoading } = usePermissions()
  const { now } = useRelativeWeekNow()
  const [weekOffsets, setWeekOffsets] = useState(() => readStoredWeekOffsets())
  const [collapseEmpty, setCollapseEmpty] = useState(() => readStoredCollapseEmpty())

  useEffect(() => {
    storeWeekOffsets(weekOffsets)
  }, [weekOffsets])

  useEffect(() => {
    storeCollapseEmpty(collapseEmpty)
  }, [collapseEmpty])

  const [courses, setCourses] = useState<CoursePublic[] | null>(null)
  const [coursesError, setCoursesError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [studentCourses, setStudentCourses] = useState<CoursePublic[]>([])
  const [staffCourses, setStaffCourses] = useState<CoursePublic[]>([])
  const [todoItems, setTodoItems] = useState<StudentTodoItem[]>([])
  const [placements, setPlacements] = useState<StudentTodoPlacement[]>([])
  const [gradingBacklog, setGradingBacklog] = useState<GradingBacklogItem[]>([])

  const load = useCallback(async () => {
    setLoading(true)
    setLoadError(null)
    setCoursesError(null)

    try {
      const res = await authorizedFetch('/api/v1/courses')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setCourses([])
        setCoursesError(readApiErrorMessage(raw))
        setStudentCourses([])
        setStaffCourses([])
        setTodoItems([])
        setPlacements([])
        setGradingBacklog([])
        return
      }

      const list = ((raw as { courses?: CoursePublic[] }).courses ?? []).filter((c) => !c.archived)
      setCourses(list)

      const enriched = await mapPool(list, 4, async (course) => {
        try {
          return await fetchCourse(course.courseCode)
        } catch {
          return course
        }
      })

      const students = enriched.filter((c) => hasStudentRole(c.viewerEnrollmentRoles))
      const staff = enriched.filter((c) => viewerIsCourseStaffEnrollment(c.viewerEnrollmentRoles))
      setStudentCourses(students)
      setStaffCourses(staff)

      const [structures, tasks, boardPlacements, backlogRows] = await Promise.all([
        students.length > 0
          ? mapPool(students, 3, async (course) => {
              try {
                const items = await fetchCourseStructure(course.courseCode)
                return { courseCode: course.courseCode, items }
              } catch {
                return { courseCode: course.courseCode, items: [] as CourseStructureItem[] }
              }
            })
          : Promise.resolve([] as { courseCode: string; items: CourseStructureItem[] }[]),
        students.length > 0 ? fetchNotebookTasks().catch(() => []) : Promise.resolve([]),
        students.length > 0 ? fetchStudentTodoBoardPlacements().catch(() => []) : Promise.resolve([]),
        staff.length > 0
          ? mapPool(
              staff.filter((course) => allows(courseGradebookViewPermission(course.courseCode))),
              3,
              async (course) => {
                try {
                  const items = await fetchCourseGradingBacklog(course.courseCode)
                  return items.map((item) => ({
                    itemId: item.itemId ?? item.assignmentId,
                    itemType: item.itemType ?? 'assignment',
                    assignmentId: item.assignmentId,
                    assignmentTitle: item.assignmentTitle,
                    ungradedCount: item.ungradedCount,
                    courseCode: course.courseCode,
                    courseTitle: course.title,
                  }))
                } catch {
                  return [] as GradingBacklogItem[]
                }
              },
            )
          : Promise.resolve([] as GradingBacklogItem[][]),
      ])

      const structureMap = Object.fromEntries(structures.map((row) => [row.courseCode, row.items]))
      setPlacements(boardPlacements)
      setTodoItems(
        collectStudentTodoItems({
          studentCourses: students,
          structureByCourseCode: structureMap,
          notebookTasks: tasks,
        }),
      )
      setGradingBacklog(backlogRows.flat())
    } catch (e: unknown) {
      setLoadError(e instanceof Error ? e.message : 'Could not load todos.')
    } finally {
      setLoading(false)
    }
  }, [allows])

  useEffect(() => {
    if (permLoading) return
    void load()
  }, [load, permLoading])

  useEffect(() => {
    function onTasksChanged() {
      void load()
    }
    window.addEventListener(NOTEBOOK_TASKS_CHANGED, onTasksChanged)
    return () => window.removeEventListener(NOTEBOOK_TASKS_CHANGED, onTasksChanged)
  }, [load])

  const showStudentBoard = studentCourses.length > 0
  const showGradingList = staffCourses.length > 0

  const description = useMemo(() => {
    if (showStudentBoard && showGradingList) {
      return 'Plan your week as a student and work through grading as an instructor.'
    }
    if (showStudentBoard) {
      return 'Plan your week day by day. Drag items between columns to reschedule your work.'
    }
    if (showGradingList) {
      return 'Assignments and quizzes waiting for your feedback.'
    }
    return 'Enroll in a course to see student todos or staff grading work here.'
  }, [showGradingList, showStudentBoard])

  const onItemMovedToDone = useCallback(async (item: StudentTodoItem) => {
    if (item.kind !== 'notebook_task' || !item.notebookTaskId || !item.notebookPageId) return
    await patchNotebookTask(item.notebookTaskId, { completed: true })
    markNotebookTaskComplete(item.courseCode, item.notebookPageId, item.notebookTaskId)
    setTodoItems((prev) => prev.filter((row) => row.key !== item.key))
    window.dispatchEvent(new Event(NOTEBOOK_TASKS_CHANGED))
  }, [])

  return (
    <LmsPage
      title="Todos"
      description={description}
      fillHeight
      actions={
        showStudentBoard && !loading ? (
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <StudentTodoViewPicker collapseEmpty={collapseEmpty} onCollapseEmptyChange={setCollapseEmpty} />
            <StudentTodoWeekPicker value={weekOffsets} onChange={setWeekOffsets} now={now} />
          </div>
        ) : undefined
      }
    >
      {coursesError ? (
        <p className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100">
          {coursesError}
        </p>
      ) : null}
      {loadError ? (
        <p className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/50 dark:text-rose-200">
          {loadError}
        </p>
      ) : null}

      {loading ? (
        <p className="text-sm text-slate-500 dark:text-neutral-400" role="status">
          Loading todos…
        </p>
      ) : null}

      {!loading && showStudentBoard ? (
        <section className="flex min-h-0 flex-1 flex-col" aria-label="Student todo board">
          <StudentTodoKanban
            items={todoItems}
            placements={placements}
            weekOffsets={weekOffsets}
            now={now}
            collapseEmpty={collapseEmpty}
            onItemMovedToDone={onItemMovedToDone}
          />
        </section>
      ) : null}

      {!loading && showGradingList ? (
        <section
          className={showStudentBoard ? 'mt-10 border-t border-slate-200 pt-8 dark:border-neutral-700' : ''}
          aria-label="Grading todo list"
        >
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
            Needs grading
          </h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
            Open SpeedGrader from any row to grade ungraded submissions.
          </p>
          <div className="mt-4">
            <GradingBacklogList
              items={gradingBacklog}
              showCourse
              emptyMessage="Nothing waiting for grading across your courses."
            />
          </div>
        </section>
      ) : null}

      {!loading && !showStudentBoard && !showGradingList && courses !== null ? (
        <p className="rounded-xl border border-slate-200 bg-slate-50/80 px-4 py-3 text-sm text-slate-600 dark:border-neutral-700 dark:bg-neutral-900/50 dark:text-neutral-300">
          Join a course as a student to plan your week here, or as teaching staff to see grading work.
        </p>
      ) : null}
    </LmsPage>
  )
}
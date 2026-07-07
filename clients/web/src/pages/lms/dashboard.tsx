import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { formatDate, formatDateTime } from '../../lib/format'
import { Link, useNavigate } from 'react-router-dom'
import {
  ArrowRight,
  BookOpen,
  CalendarDays,
  ClipboardList,
  Inbox,
  Megaphone,
  MessageCircle,
  Sparkles,
  Flame,
  ChevronDown,
  ChevronUp,
  FolderOpen,
  GraduationCap,
} from 'lucide-react'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { mapPool } from '../../lib/async-pool'
import { fetchFeedChannels, fetchFeedMessages } from '../../lib/course-feed-api'
import { getCourseViewAs } from '../../lib/course-view-as'
import { getAccountType, getJwtSubject } from '../../lib/auth'
import {
  courseGradebookViewPermission,
  fetchCourse,
  fetchCourseGradebookGrid,
  fetchCourseGradingBacklog,
  fetchCourseMyGrades,
  fetchCourseStructure,
  fetchLearnerRecommendations,
  fetchLearnerReviewStats,
  postRecommendationEvent,
  viewerIsCourseStaffEnrollment,
  viewerShouldShowMyGradesNav,
  type CourseGradebookGridResponse,
  type CourseMyGradesResponse,
  type CoursePublic,
  type CourseStructureItem,
  type RecommendationItem,
  type ReviewStatsPayload,
} from '../../lib/courses-api'
import { getMostRecentLastVisited, hrefForLastVisited } from '../../lib/last-visited-module-item'
import { hrefForRecommendationItem, surfaceLabel } from '../../lib/recommendation-nav'
import { DeadlineDateTime } from '../../components/timezone/deadline-datetime'
import { useInboxUnreadCount, useCoursesRevision } from '../../context/use-inbox-unread'
import { useCourseFeedUnread } from '../../context/use-course-feed-unread'
import { usePermissions } from '../../context/use-permissions'
import { canCreateCourses } from '../../lib/rbac-api'
import {
  computeCourseFinalPercent,
  formatFinalPercent,
  type AssignmentGroupWeight,
  type GradebookColumnForFinal,
} from './gradebook/compute-course-final-percent'
import { DashboardCourseSectionSkeleton, DashboardLoadingSkeleton } from '../../components/ui/lms-content-skeletons'
import {
  GradingBacklogList,
  type GradingBacklogItem,
} from '../../components/dashboard/grading-backlog-list'
import { NotebookTasksCard } from '../../components/dashboard/notebook-tasks-card'
import { SelfPacedDashboardSection } from '../../components/self-paced/self-paced-dashboard-section'
import { DegreeProgressCard } from '../../components/dashboard/degree-progress-card'
import { RecentCertificatesCard } from '../../components/credentials/recent-certificates-card'
import { DashboardLearningPathsCard } from '../../components/dashboard/dashboard-learning-paths-card'
import { ConsentPrompt } from '../../components/research/consent-prompt'
import { EnrollmentStateBadge } from '../../components/enrollment/enrollment-state-badge'
import type { EnrollmentState } from '../../lib/enrollment-state-api'
import { StudyStatsCard } from '../../components/study-stats/study-stats-card'
import { GamificationDashboardCard } from '../../components/gamification/gamification-dashboard-card'
import { StartHereCard } from '../../components/onboarding/start-here-card'
import { DailyGoalProgressCard } from '../../components/study-reminders/daily-goal-progress-card'
import { StudyBuddyPromptsCard, StudyBuddyWidget } from '../../components/notebook/study-buddy-widget'
import { LmsPage } from './lms-page'
import { fetchCatalogSchedule, type ScheduleEntry } from '../../lib/catalog-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { fetchCalendarEvents, type AcademicCalendarEvent } from '../../lib/courses-api'
import { scheduleIdleTask } from '../../lib/schedule-idle'
import { splitCoursesForPrefetch } from '../../lib/dashboard-course-prefetch'

function startOfWeekMonday(now = new Date()): Date {
  const d = new Date(now)
  const day = d.getDay()
  const diff = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + diff)
  d.setHours(0, 0, 0, 0)
  return d
}

function endOfWeekSunday(start: Date): Date {
  const e = new Date(start)
  e.setDate(start.getDate() + 6)
  e.setHours(23, 59, 59, 999)
  return e
}

function weekProgressFraction(now = new Date()): number {
  const start = startOfWeekMonday(now).getTime()
  const end = endOfWeekSunday(startOfWeekMonday(now)).getTime()
  const t = now.getTime()
  if (end <= start) return 0
  return Math.max(0, Math.min(1, (t - start) / (end - start)))
}

function hasStudentRole(roles: readonly string[] | undefined): boolean {
  if (!roles?.length) return false
  return roles.some((r) => r.trim().toLowerCase() === 'student')
}

function feedSnippet(body: string, max = 140): string {
  const t = body
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/[#*_`[\]]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
  if (!t) return ''
  return t.length <= max ? t : `${t.slice(0, max - 1)}…`
}

function countEmptyGradeCells(grid: CourseGradebookGridResponse): number {
  const { students, columns, grades = {} } = grid
  let n = 0
  for (const col of columns) {
    if (col.maxPoints == null || col.maxPoints <= 0) continue
    for (const s of students) {
      const cell = grades[s.userId]?.[col.id]
      if (cell == null || String(cell).trim() === '') n++
    }
  }
  return n
}

function courseSectionSubtitle(course: CoursePublic, t: TFunction<'dashboard'>): string | null {
  if (course.sectionsEnabled !== true) return null
  const code = course.viewerSectionCode?.trim()
  if (!code) return null
  const name = course.viewerSectionName?.trim()
  return name ? t('dashboard.section.withName', { code, name }) : t('dashboard.section.codeOnly', { code })
}

function dueThisWeekItems(
  structure: CourseStructureItem[],
  weekStart: Date,
  weekEnd: Date,
): CourseStructureItem[] {
  const t0 = weekStart.getTime()
  const t1 = weekEnd.getTime()
  const out: CourseStructureItem[] = []
  for (const it of structure) {
    if (!it.dueAt) continue
    if (it.kind !== 'assignment' && it.kind !== 'quiz' && it.kind !== 'content_page') continue
    const t = new Date(it.dueAt).getTime()
    if (Number.isNaN(t) || t < t0 || t > t1) continue
    out.push(it)
  }
  out.sort((a, b) => new Date(a.dueAt!).getTime() - new Date(b.dueAt!).getTime())
  return out
}

function gradeSnippetForItem(
  my: CourseMyGradesResponse | null,
  itemId: string,
  t: TFunction<'dashboard'>,
): { label: string; pct: number } | null {
  if (!my) return null
  const col = my.columns.find((c) => c.id === itemId)
  if (!col || col.maxPoints == null || col.maxPoints <= 0) return null
  const raw = my.grades[itemId]
  const earned = raw != null && String(raw).trim() !== '' ? Number.parseFloat(String(raw).replace(/,/g, '')) : NaN
  if (!Number.isFinite(earned)) return { label: t('dashboard.grades.notSubmitted'), pct: 0 }
  const pct = Math.max(0, Math.min(100, (earned / col.maxPoints) * 100))
  return { label: t('dashboard.grades.points', { earned, max: col.maxPoints }), pct }
}

type AnnouncementPreview = {
  courseCode: string
  courseTitle: string
  channelName: string
  snippet: string
  author: string
  createdAt: string
  pinned: boolean
}

async function loadAnnouncementPreview(
  course: CoursePublic,
  t: TFunction<'dashboard'>,
): Promise<AnnouncementPreview | null> {
  if (course.feedEnabled === false) return null
  try {
    const channels = await fetchFeedChannels(course.courseCode)
    if (!channels.length) return null
    const sorted = [...channels].sort((a, b) => a.sortOrder - b.sortOrder)
    const preferred =
      sorted.find((c) => c.name.toLowerCase().includes('announce')) ??
      sorted.find((c) => c.name.toLowerCase() === 'general') ??
      sorted[0]
    const messages = await fetchFeedMessages(course.courseCode, preferred.id)
    const roots = messages.filter((m) => !m.parentMessageId)
    const viewer = getJwtSubject()?.toLowerCase() ?? ''
    const ranked = [...roots].sort((a, b) => {
      const ap = a.pinnedAt ? 1 : 0
      const bp = b.pinnedAt ? 1 : 0
      if (ap !== bp) return bp - ap
      return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    })
    const pick =
      ranked.find((m) => m.authorUserId.toLowerCase() !== viewer) ?? ranked[0] ?? null
    if (!pick) return null
    const snip = feedSnippet(pick.body)
    if (!snip) return null
    return {
      courseCode: course.courseCode,
      courseTitle: course.title,
      channelName: preferred.name,
      snippet: snip,
      author: pick.authorDisplayName?.trim() || pick.authorEmail || t('dashboard.announcements.authorFallback'),
      createdAt: pick.createdAt,
      pinned: Boolean(pick.pinnedAt),
    }
  } catch {
    return null
  }
}

async function loadStudentRow(course: CoursePublic, t: TFunction<'dashboard'>) {
  let structure: CourseStructureItem[] = []
  let myGrades: CourseMyGradesResponse | null = null
  try {
    structure = await fetchCourseStructure(course.courseCode)
  } catch {
    structure = []
  }
  const preview = getCourseViewAs(course.courseCode)
  if (viewerShouldShowMyGradesNav(course.viewerEnrollmentRoles, preview)) {
    try {
      myGrades = await fetchCourseMyGrades(course.courseCode)
    } catch {
      myGrades = null
    }
  }
  let announcement: AnnouncementPreview | null = null
  if (course.feedEnabled !== false) {
    announcement = await loadAnnouncementPreview(course, t)
  }
  return { course, structure, myGrades, announcement }
}

async function loadStaffRow(
  course: CoursePublic,
  allows: (permission: string) => boolean,
) {
  const code = course.courseCode
  let emptyGradeCells: number | null = null
  let gradingBacklog: GradingBacklogItem[] = []
  if (allows(courseGradebookViewPermission(code))) {
    try {
      const grid = await fetchCourseGradebookGrid(code)
      emptyGradeCells = countEmptyGradeCells(grid)
    } catch {
      emptyGradeCells = null
    }
    try {
      const backlog = await fetchCourseGradingBacklog(code)
      gradingBacklog = backlog.map((item) => ({
        itemId: item.itemId ?? item.assignmentId,
        itemType: item.itemType ?? 'assignment',
        assignmentId: item.assignmentId,
        assignmentTitle: item.assignmentTitle,
        ungradedCount: item.ungradedCount,
        courseCode: code,
        courseTitle: course.title,
      }))
    } catch {
      gradingBacklog = []
    }
  }
  return { course, emptyGradeCells, gradingBacklog }
}

export default function Dashboard() {
  const { t } = useTranslation('dashboard')
  const navigate = useNavigate()
  useEffect(() => {
    if (getAccountType() === 'parent') {
      navigate('/parent', { replace: true })
    }
  }, [navigate])
  const { allows, loading: permLoading } = usePermissions()
  const showCourseCreateActions = canCreateCourses(allows, permLoading)
  const inboxUnread = useInboxUnreadCount()
  const coursesRevision = useCoursesRevision()
  const { totalFeedUnread } = useCourseFeedUnread()
  const {
    ffCatalogIntegration,
    ffEnrollmentStateMachine,
    ffAcademicCalendar,
    ffEportfolio,
    ffCoCurricularTranscript,
    ffCeuTracking,
    ffAdvisingIntegration,
    ffLearningPaths,
    ffCompletionCredentials,
    ffGamification,
    ffStudyReminders,
    aiStudyBuddyEnabled,
    ffResearchConsent,
  } = usePlatformFeatures()

  const [catalog, setCatalog] = useState<CoursePublic[] | null>(null)
  const [schedule, setSchedule] = useState<ScheduleEntry[]>([])
  const [catalogError, setCatalogError] = useState<string | null>(null)
  const [courses, setCourses] = useState<CoursePublic[] | null>(null)
  const [detailsLoading, setDetailsLoading] = useState(false)
  const [detailError, setDetailError] = useState<string | null>(null)
  const [deferredStudentCourses, setDeferredStudentCourses] = useState<CoursePublic[]>([])
  const [deferredStaffCourses, setDeferredStaffCourses] = useState<CoursePublic[]>([])
  const [loadingMoreCourses, setLoadingMoreCourses] = useState(false)

  const [studentRows, setStudentRows] = useState<
    {
      course: CoursePublic
      structure: CourseStructureItem[]
      myGrades: CourseMyGradesResponse | null
      announcement: AnnouncementPreview | null
    }[]
  >([])
  const [staffRows, setStaffRows] = useState<
    {
      course: CoursePublic
      emptyGradeCells: number | null
      gradingBacklog: GradingBacklogItem[]
    }[]
  >([])

  const [reviewStats, setReviewStats] = useState<ReviewStatsPayload | null>(null)
  const [whatsNextRaw, setWhatsNextRaw] = useState<{
    course: CoursePublic
    primary: RecommendationItem | null
    chips: RecommendationItem[]
    degraded: boolean
  } | null>(null)

  const [upcomingCalendarEvents, setUpcomingCalendarEvents] = useState<AcademicCalendarEvent[]>([])

  const [collapsedSections, setCollapsedSections] = useState<Record<string, boolean>>(() => {
    try {
      const saved = localStorage.getItem('dashboard_collapsed_sections')
      return saved ? JSON.parse(saved) : {}
    } catch {
      return {}
    }
  })

  useEffect(() => {
    try {
      localStorage.setItem('dashboard_collapsed_sections', JSON.stringify(collapsedSections))
    } catch {
      // Ignore storage errors
    }
  }, [collapsedSections])

  const toggleSection = (id: string) => {
    setCollapsedSections((prev) => ({ ...prev, [id]: !prev[id] }))
  }

  const whatsNext = useMemo(() => {
    const uid = getJwtSubject()
    const top = studentRows[0]?.course
    if (!uid || !top || !whatsNextRaw) return null
    return whatsNextRaw.course.id === top.id ? whatsNextRaw : null
  }, [whatsNextRaw, studentRows])

  const studyBuddyCourseCode = studentRows[0]?.course?.courseCode ?? null

  const detailGenRef = useRef(0)

  useEffect(() => {
    const uid = getJwtSubject()
    if (!uid) return
    let cancelled = false
    const cancelIdle = scheduleIdleTask(() => {
      void (async () => {
        try {
          const s = await fetchLearnerReviewStats(uid)
          if (!cancelled) setReviewStats(s)
        } catch {
          if (!cancelled) setReviewStats(null)
        }
      })()
    })
    return () => {
      cancelled = true
      cancelIdle()
    }
  }, [])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      await Promise.resolve()
      if (cancelled) return
      setCatalogError(null)
      try {
        const res = await authorizedFetch('/api/v1/courses')
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok) {
          if (!cancelled) {
            setCatalog([])
            setCatalogError(readApiErrorMessage(raw))
          }
          return
        }
        const data = raw as { courses?: CoursePublic[] }
        const nextCatalog = data.courses ?? []
        if (!cancelled) {
          setCatalog(nextCatalog)
          setCourses(nextCatalog)
          performance.mark('dashboard:catalog-loaded')
        }
      } catch {
        if (!cancelled) {
          setCatalog([])
          setCatalogError(t('dashboard.errors.loadCourses'))
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [coursesRevision, t])

  useEffect(() => {
    if (!ffCatalogIntegration) {
      setSchedule([])
      return
    }
    let cancelled = false
    const cancelIdle = scheduleIdleTask(() => {
      void fetchCatalogSchedule()
        .then((entries) => {
          if (!cancelled) setSchedule(entries)
        })
        .catch(() => {
          if (!cancelled) setSchedule([])
        })
    })
    return () => {
      cancelled = true
      cancelIdle()
    }
  }, [ffCatalogIntegration, coursesRevision])

  useEffect(() => {
    if (catalog === null || permLoading) return
    const gen = ++detailGenRef.current
    const list = catalog

    void (async () => {
      await Promise.resolve()
      if (detailGenRef.current !== gen) return
      setDetailError(null)
      setStudentRows([])
      setStaffRows([])
      setDeferredStudentCourses([])
      setDeferredStaffCourses([])

      if (!list.length) {
        setCourses([])
        setDetailsLoading(false)
        return
      }

      setDetailsLoading(true)

      try {
        const enriched = await mapPool(list, 4, async (c) => {
          try {
            return await fetchCourse(c.courseCode)
          } catch {
            return c
          }
        })
        if (detailGenRef.current !== gen) return
        setCourses(enriched)
        performance.mark('dashboard:courses-enriched')

        const studentCourses = enriched.filter((c) => hasStudentRole(c.viewerEnrollmentRoles))
        const staffCourses = enriched.filter((c) => viewerIsCourseStaffEnrollment(c.viewerEnrollmentRoles))

        const { initial: initialStudent, deferred: deferredStudent } = splitCoursesForPrefetch(studentCourses)
        const { initial: initialStaff, deferred: deferredStaff } = splitCoursesForPrefetch(staffCourses)

        const sRows = await mapPool(initialStudent, 3, (course) => loadStudentRow(course, t))
        if (detailGenRef.current !== gen) return

        const tRows = await mapPool(initialStaff, 3, (course) => loadStaffRow(course, allows))
        if (detailGenRef.current !== gen) return

        setStudentRows(sRows)
        setStaffRows(tRows)
        setDeferredStudentCourses(deferredStudent)
        setDeferredStaffCourses(deferredStaff)
        setDetailsLoading(false)
        performance.mark('dashboard:rows-loaded')
      } catch {
        if (detailGenRef.current !== gen) return
        setDetailError(t('dashboard.errors.loadDetails'))
        setCourses(list)
        setDetailsLoading(false)
      }
    })()
  }, [catalog, permLoading, allows, t])

  const loadMoreCourses = () => {
    if (loadingMoreCourses) return
    const pendingStudent = deferredStudentCourses
    const pendingStaff = deferredStaffCourses
    if (pendingStudent.length === 0 && pendingStaff.length === 0) return

    setLoadingMoreCourses(true)
    void (async () => {
      try {
        const [moreStudent, moreStaff] = await Promise.all([
          mapPool(pendingStudent, 3, (course) => loadStudentRow(course, t)),
          mapPool(pendingStaff, 3, (course) => loadStaffRow(course, allows)),
        ])
        setStudentRows((prev) => [...prev, ...moreStudent])
        setStaffRows((prev) => [...prev, ...moreStaff])
        setDeferredStudentCourses([])
        setDeferredStaffCourses([])
      } finally {
        setLoadingMoreCourses(false)
      }
    })()
  }

  useEffect(() => {
    const uid = getJwtSubject()
    if (!uid || studentRows.length === 0) {
      return
    }
    const { course } = studentRows[0]
    let cancelled = false
    const cancelIdle = scheduleIdleTask(() => {
      void (async () => {
        try {
          const surfaces = ['continue', 'review', 'strengthen', 'challenge'] as const
          const results = await Promise.all(
            surfaces.map((s) => fetchLearnerRecommendations(uid, course.id, s, { limit: 4 })),
          )
          if (cancelled) return
          const merged: RecommendationItem[] = []
          let degraded = false
          for (const r of results) {
            merged.push(...r.recommendations)
            if (r.degraded) degraded = true
          }
          merged.sort((a, b) => b.score - a.score)
          const primary = merged[0] ?? null
          const chips = merged.slice(1, 4)
          setWhatsNextRaw({ course, primary, chips, degraded })
          if (primary) {
            void postRecommendationEvent({
              courseId: course.id,
              itemId: primary.itemId,
              surface: primary.surface,
              eventType: 'impression',
              rank: 0,
            }).catch(() => {})
          }
        } catch {
          if (!cancelled) setWhatsNextRaw(null)
        }
      })()
    })
    return () => {
      cancelled = true
      cancelIdle()
    }
  }, [studentRows])

  useEffect(() => {
    if (!ffAcademicCalendar || !courses?.length) return
    let cancelled = false
    const cancelIdle = scheduleIdleTask(() => {
      void (async () => {
        const today = new Date().toISOString().slice(0, 10)
        const pairs = new Map<string, string | undefined>()
        for (const c of courses) {
          if (c.orgId) pairs.set(c.orgId, c.termId ?? undefined)
        }
        const fetches = Array.from(pairs.entries()).map(([orgId, termId]) =>
          fetchCalendarEvents(orgId, termId).catch(() => [] as AcademicCalendarEvent[]),
        )
        const results = await Promise.all(fetches)
        if (cancelled) return
        const all = results.flat().filter((e) => e.startDate >= today)
        all.sort((a, b) => a.startDate.localeCompare(b.startDate))
        setUpcomingCalendarEvents(all.slice(0, 5))
      })()
    })
    return () => {
      cancelled = true
      cancelIdle()
    }
  }, [ffAcademicCalendar, courses])

  const weekStart = useMemo(() => startOfWeekMonday(), [])
  const weekEnd = useMemo(() => endOfWeekSunday(weekStart), [weekStart])
  const weekFrac = useMemo(() => weekProgressFraction(), [])

  const courseCodes = useMemo(() => (courses ?? []).map((c) => c.courseCode), [courses])

  const courseTitles = useMemo(() => {
    const out: Record<string, string> = {}
    for (const c of courses ?? []) {
      out[c.courseCode] = c.title
    }
    return out
  }, [courses])

  /** Read on each render so returning from a module picks up the latest `localStorage` write. */
  const continueTarget =
    courseCodes.length > 0 ? getMostRecentLastVisited(courseCodes) : null

  const announcements = useMemo(() => {
    const list = studentRows.map((r) => r.announcement).filter(Boolean) as AnnouncementPreview[]
    list.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    return list.slice(0, 4)
  }, [studentRows])

  const anyStudentExperience = studentRows.length > 0
  const anyStaffExperience = staffRows.length > 0

  const allGradingBacklog = useMemo(
    () =>
      staffRows
        .flatMap((row) => row.gradingBacklog)
        .sort((a, b) => b.ungradedCount - a.ungradedCount),
    [staffRows],
  )

  const hasCourses = (catalog?.length ?? 0) > 0
  const showInitialLoading = catalog === null
  const deferredCourseCount = deferredStudentCourses.length + deferredStaffCourses.length

  return (
    <LmsPage
      title={t('dashboard.title')}
      description={t('dashboard.description')}
    >
      {catalogError && (
        <p className="mt-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-100">
          {catalogError}
        </p>
      )}
      {detailError && (
        <p className="mt-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100">
          {detailError}
        </p>
      )}

      {showInitialLoading && !catalogError && <DashboardLoadingSkeleton />}

      {catalog && catalog.length === 0 && !catalogError && (
        <div className="mt-10 rounded-2xl border border-slate-200 bg-slate-50/80 px-6 py-8 text-center dark:border-neutral-700 dark:bg-neutral-900/50">
          <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">{t('dashboard.empty.noCourses')}</p>
          <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
            {showCourseCreateActions
              ? t('dashboard.empty.joinOrCreate')
              : t('dashboard.empty.joinOnly')}
          </p>
          <div className="mt-5 flex flex-wrap justify-center gap-3">
            <Link
              to="/courses"
              className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500"
            >
              {t('dashboard.empty.browseCourses')}
              <ArrowRight className="h-4 w-4" aria-hidden />
            </Link>
          </div>
        </div>
      )}

      {hasCourses && (
        <div data-onboarding="dashboard-main" className="mt-8 space-y-10">
          <section aria-label={t('dashboard.quickLinks.ariaLabel')}>
            <div className="flex flex-wrap gap-3">
              <Link
                to="/inbox"
                className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800"
              >
                <Inbox className="h-4 w-4 text-indigo-500" aria-hidden />
                {t('dashboard.quickLinks.inbox')}
                {inboxUnread > 0 ? (
                  <span className="rounded-full bg-indigo-600 px-2 py-0.5 text-xs font-semibold text-white">
                    {inboxUnread > 99 ? '99+' : inboxUnread}
                  </span>
                ) : null}
              </Link>
              <Link
                to="/courses"
                className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800"
              >
                <BookOpen className="h-4 w-4 text-indigo-500" aria-hidden />
                {t('dashboard.quickLinks.allCourses')}
              </Link>
              {ffEportfolio ? (
                <Link
                  to="/portfolios"
                  className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800"
                >
                  <FolderOpen className="h-4 w-4 text-indigo-500" aria-hidden />
                  {t('dashboard.quickLinks.myPortfolio')}
                </Link>
              ) : null}
              {ffAdvisingIntegration ? (
                <Link
                  to="/advising-notes"
                  className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800"
                >
                  <GraduationCap className="h-4 w-4 text-indigo-500" aria-hidden />
                  {t('dashboard.quickLinks.advisingNotes')}
                </Link>
              ) : null}
              {totalFeedUnread > 0 ? (
                <span className="inline-flex items-center gap-2 rounded-xl border border-teal-200 bg-teal-50 px-4 py-2.5 text-sm font-medium text-teal-900 dark:border-teal-900/50 dark:bg-teal-950/40 dark:text-teal-50">
                  <MessageCircle className="h-4 w-4" aria-hidden />
                  {t('dashboard.quickLinks.feedUnread', { count: totalFeedUnread })}
                </span>
              ) : (
                <span className="inline-flex items-center gap-2 rounded-xl border border-slate-100 bg-slate-50 px-4 py-2.5 text-xs text-slate-500 dark:border-neutral-800 dark:bg-neutral-900/40 dark:text-neutral-500">
                  <MessageCircle className="h-4 w-4" aria-hidden />
                  {t('dashboard.quickLinks.feedUnreadHint')}
                </span>
              )}
            </div>
          </section>

          <StartHereCard />

          {detailsLoading && <DashboardCourseSectionSkeleton />}

          {whatsNext && anyStudentExperience && (
            <section aria-label={t('dashboard.whatsNext.ariaLabel')}>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                {t('dashboard.whatsNext.title')}
              </h2>
              {whatsNext.primary ? (
                <article
                  role="article"
                  aria-label={t('dashboard.whatsNext.recommendedAria', {
                    title: whatsNext.primary.title,
                    surface: surfaceLabel(whatsNext.primary.surface),
                  })}
                  className="mt-3 rounded-2xl border border-violet-100 bg-gradient-to-br from-violet-50/90 to-white p-5 shadow-sm dark:border-violet-900/40 dark:from-violet-950/30 dark:to-neutral-900"
                >
                  <div className="flex flex-wrap items-center gap-2 text-xs font-medium text-violet-800 dark:text-violet-200">
                    <Sparkles className="h-4 w-4 shrink-0" aria-hidden />
                    <span>{whatsNext.course.title}</span>
                    <span className="rounded-full bg-violet-100 px-2 py-0.5 text-violet-900 dark:bg-violet-900/50 dark:text-violet-100">
                      {surfaceLabel(whatsNext.primary.surface)}
                    </span>
                  </div>
                  <p className="mt-2 text-lg font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
                    {whatsNext.primary.title}
                  </p>
                  <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">{whatsNext.primary.reason}</p>
                  {whatsNext.degraded ? (
                    <p className="mt-2 text-xs text-amber-800 dark:text-amber-200">
                      {t('dashboard.whatsNext.degraded')}
                    </p>
                  ) : null}
                  <Link
                    to={hrefForRecommendationItem(whatsNext.course.courseCode, whatsNext.primary)}
                    onClick={() => {
                      const p = whatsNext.primary
                      if (p == null) return
                      void postRecommendationEvent({
                        courseId: whatsNext.course.id,
                        itemId: p.itemId,
                        surface: p.surface,
                        eventType: 'click',
                        rank: 0,
                      }).catch(() => {})
                    }}
                    className="mt-4 inline-flex items-center gap-2 rounded-xl bg-violet-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-violet-500"
                  >
                    {t('dashboard.whatsNext.go')}
                    <ArrowRight className="h-4 w-4" aria-hidden />
                  </Link>
                  {whatsNext.chips.length > 0 ? (
                    <div className="mt-4 flex flex-wrap gap-2 border-t border-violet-100 pt-4 dark:border-violet-900/40">
                      {whatsNext.chips.map((c, idx) => (
                        <Link
                          key={`${c.itemId}-${c.surface}-${idx}`}
                          to={hrefForRecommendationItem(whatsNext.course.courseCode, c)}
                          onClick={() => {
                            void postRecommendationEvent({
                              courseId: whatsNext.course.id,
                              itemId: c.itemId,
                              surface: c.surface,
                              eventType: 'click',
                              rank: idx + 1,
                            }).catch(() => {})
                          }}
                          className="inline-flex items-center gap-1 rounded-lg border border-violet-200 bg-white px-2.5 py-1 text-xs font-medium text-violet-900 shadow-sm hover:bg-violet-50 dark:border-violet-800 dark:bg-neutral-900 dark:text-violet-100 dark:hover:bg-violet-950/40"
                        >
                          <span className="text-violet-600 dark:text-violet-300">{surfaceLabel(c.surface)}</span>
                          <span className="max-w-[10rem] truncate">{c.title}</span>
                        </Link>
                      ))}
                    </div>
                  ) : null}
                </article>
              ) : (
                <p className="mt-3 rounded-2xl border border-slate-200 bg-slate-50/80 px-4 py-4 text-sm text-slate-700 dark:border-neutral-700 dark:bg-neutral-900/50 dark:text-neutral-200">
                  {t('dashboard.whatsNext.caughtUp', { courseTitle: whatsNext.course.title })}
                </p>
              )}
            </section>
          )}

          <StudyStatsCard />

          {ffStudyReminders && anyStudentExperience ? <DailyGoalProgressCard /> : null}

          {aiStudyBuddyEnabled && studyBuddyCourseCode && anyStudentExperience ? (
            <StudyBuddyPromptsCard courseCode={studyBuddyCourseCode} />
          ) : null}

          {ffGamification && anyStudentExperience ? <GamificationDashboardCard /> : null}

          {ffCompletionCredentials && anyStudentExperience ? <RecentCertificatesCard /> : null}

          {ffLearningPaths && anyStudentExperience ? <DashboardLearningPathsCard /> : null}

          {ffAdvisingIntegration && anyStudentExperience ? <DegreeProgressCard /> : null}

          {ffResearchConsent && anyStudentExperience ? <ConsentPrompt /> : null}

          {ffCeuTracking ? (
            <section aria-label={t('dashboard.continuingEducation.ariaLabel')}>
              <div className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-teal-100 bg-teal-50/80 px-5 py-4 dark:border-teal-900/40 dark:bg-teal-950/30">
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                    {t('dashboard.continuingEducation.title')}
                  </p>
                  <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
                    {t('dashboard.continuingEducation.description')}
                  </p>
                </div>
                <Link
                  to="/me/ce-transcript"
                  className="inline-flex items-center gap-1 rounded-lg bg-teal-600 px-3 py-2 text-sm font-medium text-white hover:bg-teal-700"
                >
                  {t('dashboard.continuingEducation.transcriptLink')}
                  <ArrowRight className="h-4 w-4" aria-hidden />
                </Link>
              </div>
            </section>
          ) : null}

          {ffCoCurricularTranscript ? (
            <section aria-label={t('dashboard.achievements.ariaLabel')}>
              <div className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-violet-100 bg-violet-50/80 px-5 py-4 dark:border-violet-900/40 dark:bg-violet-950/30">
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                    {t('dashboard.achievements.title')}
                  </p>
                  <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
                    {t('dashboard.achievements.description')}
                  </p>
                </div>
                <Link
                  to="/me/ccr"
                  className="inline-flex items-center gap-1 rounded-lg bg-violet-600 px-3 py-2 text-sm font-medium text-white hover:bg-violet-700"
                >
                  {t('dashboard.achievements.openCcr')}
                  <ArrowRight className="h-4 w-4" aria-hidden />
                </Link>
              </div>
            </section>
          ) : null}

          {ffCompletionCredentials ? (
            <section aria-label={t('dashboard.credentials.ariaLabel')}>
              <div className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-emerald-100 bg-emerald-50/80 px-5 py-4 dark:border-emerald-900/40 dark:bg-emerald-950/30">
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                    {t('dashboard.credentials.title')}
                  </p>
                  <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
                    {t('dashboard.credentials.description')}
                  </p>
                </div>
                <Link
                  to="/me/credentials"
                  className="inline-flex items-center gap-1 rounded-lg bg-emerald-600 px-3 py-2 text-sm font-medium text-white hover:bg-emerald-700"
                >
                  {t('dashboard.credentials.viewLink')}
                  <ArrowRight className="h-4 w-4" aria-hidden />
                </Link>
              </div>
            </section>
          ) : null}

          <NotebookTasksCard courseTitles={courseTitles} />

          {reviewStats != null && (
            <section aria-label={t('dashboard.review.ariaLabel')}>
              <div className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-amber-100 bg-amber-50/80 px-5 py-4 dark:border-amber-900/40 dark:bg-amber-950/30">
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                    {t('dashboard.review.title')}
                  </p>
                  <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
                    {reviewStats.dueToday > 0
                      ? t('dashboard.review.dueToday', { count: reviewStats.dueToday })
                      : t('dashboard.review.noneDue')}
                    {reviewStats.streak > 0 ? (
                      <span className="ms-2 inline-flex items-center gap-1 font-medium text-amber-800 dark:text-amber-200">
                        <Flame className="h-3.5 w-3.5" aria-hidden />
                        {t('dashboard.review.streak', { count: reviewStats.streak })}
                      </span>
                    ) : null}
                  </p>
                </div>
                <Link
                  to="/review"
                  className="inline-flex shrink-0 items-center gap-2 rounded-xl bg-amber-700 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-amber-600"
                >
                  {reviewStats.dueToday > 0
                    ? t('dashboard.review.startReview')
                    : t('dashboard.review.openReview')}
                  <ArrowRight className="h-4 w-4" aria-hidden />
                </Link>
              </div>
            </section>
          )}

          {ffAcademicCalendar && upcomingCalendarEvents.length > 0 && (
            <section aria-label={t('dashboard.upcomingDates.ariaLabel')}>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                {t('dashboard.upcomingDates.title')}
              </h2>
              <ul className="mt-4 space-y-2">
                {upcomingCalendarEvents.map((ev) => (
                  <li
                    key={ev.id}
                    className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
                  >
                    <span className="font-medium text-slate-900 dark:text-neutral-100">{ev.eventName}</span>
                    <span className="text-slate-500 dark:text-neutral-400">
                      <time dateTime={ev.startDate}>{ev.startDate}</time>
                      {ev.endDate && ev.endDate !== ev.startDate && (
                        <> – <time dateTime={ev.endDate}>{ev.endDate}</time></>
                      )}
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          )}

          {ffCatalogIntegration && schedule.length > 0 && (
            <section aria-label={t('dashboard.schedule.ariaLabel')}>
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                  {t('dashboard.schedule.title')}
                </h2>
                <Link
                  to="/catalog"
                  className="text-xs font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                >
                  {t('dashboard.schedule.browseCatalog')}
                </Link>
              </div>
              <ul className="mt-4 space-y-3">
                {schedule.map((entry) => {
                  const sec = entry.section
                  const mp = sec.meetingPattern
                  const meeting =
                    mp?.days && mp.startTime
                      ? `${mp.days} ${mp.startTime}${mp.endTime ? `–${mp.endTime}` : ''}`
                      : '—'
                  const href = entry.courseCode
                    ? `/courses/${encodeURIComponent(entry.courseCode)}`
                    : '/catalog'
                  const regLabel =
                    entry.registrationStatus === 'registered'
                      ? t('dashboard.schedule.registration.registered')
                      : entry.registrationStatus === 'waitlisted'
                        ? t('dashboard.schedule.registration.waitlisted')
                        : entry.registrationStatus === 'auditing'
                          ? t('dashboard.schedule.registration.auditing')
                          : entry.registrationStatus
                  return (
                    <li key={sec.id}>
                      <Link
                        to={href}
                        className="flex flex-col gap-2 rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-sm transition-[background-color,color,border-color] hover:border-indigo-200 dark:border-neutral-700 dark:bg-neutral-900 dark:hover:border-indigo-800 sm:flex-row sm:items-center sm:justify-between"
                      >
                        <div className="min-w-0">
                          <p className="text-xs font-medium text-slate-500 dark:text-neutral-400">
                            {sec.subject} {sec.courseNumber}
                            {sec.sectionNumber ? ` · ${sec.sectionNumber}` : ''}
                            {sec.crn ? t('dashboard.schedule.crn', { crn: sec.crn }) : ''}
                          </p>
                          <p className="truncate font-semibold text-slate-900 dark:text-neutral-100">
                            {entry.courseTitle ?? sec.title}
                          </p>
                          <p className="mt-0.5 flex items-center gap-1 text-xs text-slate-500 dark:text-neutral-400">
                            <CalendarDays className="h-3.5 w-3.5 shrink-0" aria-hidden />
                            {meeting}
                            {sec.room ? ` · ${sec.room}` : ''}
                          </p>
                        </div>
                        <span className="inline-flex shrink-0 items-center rounded-full bg-indigo-50 px-2.5 py-1 text-xs font-medium text-indigo-700 dark:bg-indigo-950/50 dark:text-indigo-200">
                          {regLabel}
                        </span>
                      </Link>
                    </li>
                  )
                })}
              </ul>
            </section>
          )}

          <SelfPacedDashboardSection />

          {continueTarget && (
            <section aria-label={t('dashboard.continue.ariaLabel')}>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                {t('dashboard.continue.title')}
              </h2>
              <div className="mt-3 rounded-2xl border border-indigo-100 bg-gradient-to-br from-indigo-50/90 to-white p-5 shadow-sm dark:border-indigo-900/40 dark:from-indigo-950/40 dark:to-neutral-900">
                <p className="text-xs font-medium text-indigo-700 dark:text-indigo-200">
                  {courses?.find((c) => c.courseCode === continueTarget.courseCode)?.title ??
                    continueTarget.courseCode}
                </p>
                <p className="mt-1 text-lg font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
                  {continueTarget.title}
                </p>
                <Link
                  to={hrefForLastVisited(continueTarget.courseCode, continueTarget.kind, continueTarget.itemId)}
                  className="mt-4 inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500"
                >
                  {t('dashboard.continue.button')}
                  <ArrowRight className="h-4 w-4" aria-hidden />
                </Link>
              </div>
            </section>
          )}

          {anyStudentExperience && (
            <section aria-label={t('dashboard.learning.ariaLabel')}>
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                  {t('dashboard.learning.title')}
                </h2>
                <button
                  onClick={() => toggleSection('student-overview')}
                  className="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-neutral-800 dark:hover:text-neutral-300"
                  aria-label={
                    collapsedSections['student-overview']
                      ? t('dashboard.learning.expandAria')
                      : t('dashboard.learning.collapseAria')
                  }
                >
                  {collapsedSections['student-overview'] ? (
                    <ChevronDown className="h-4 w-4" />
                  ) : (
                    <ChevronUp className="h-4 w-4" />
                  )}
                </button>
              </div>

              {!collapsedSections['student-overview'] && (
                <>
                  <div className="mt-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
                    <div className="flex flex-wrap items-end justify-between gap-3">
                      <div>
                        <p className="text-base font-semibold text-slate-900 dark:text-neutral-100">
                          {t('dashboard.learning.dueThisWeek')}
                        </p>
                        <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                          {formatDate(weekStart, { dateStyle: 'medium' })} –{' '}
                          {formatDate(weekEnd, { dateStyle: 'medium' })}
                        </p>
                      </div>
                      <div className="min-w-[140px] flex-1 max-w-xs">
                        <div className="flex justify-between text-[0.65rem] font-medium uppercase tracking-wide text-slate-400 dark:text-neutral-500">
                          <span>{t('dashboard.learning.weekLabel')}</span>
                          <span>{Math.round(weekFrac * 100)}%</span>
                        </div>
                        <div className="mt-1 h-2 overflow-hidden rounded-full bg-slate-100 dark:bg-neutral-800">
                          <div
                            className="h-full rounded-full bg-indigo-500 motion-safe:transition-[width]"
                            style={{ width: `${Math.round(weekFrac * 100)}%` }}
                          />
                        </div>
                      </div>
                    </div>
                    <ul className="mt-4 space-y-3">
                      {studentRows
                        .flatMap((row) => {
                          const dues = dueThisWeekItems(row.structure, weekStart, weekEnd)
                          return dues.map((it) => ({ row, it }))
                        })
                        .slice(0, 24)
                        .map(({ row, it }) => {
                          const g = gradeSnippetForItem(row.myGrades, it.id, t)
                          const base = `/courses/${encodeURIComponent(row.course.courseCode)}`
                          const href =
                            it.kind === 'quiz'
                              ? `${base}/modules/quiz/${encodeURIComponent(it.id)}`
                              : it.kind === 'assignment'
                                ? `${base}/modules/assignment/${encodeURIComponent(it.id)}`
                                : `${base}/modules/content/${encodeURIComponent(it.id)}`
                          return (
                            <li key={`${row.course.courseCode}-${it.id}`}>
                              <Link
                                to={href}
                                className="flex flex-col gap-1 rounded-xl border border-slate-100 px-3 py-3 transition-[background-color,color,border-color] hover:border-indigo-200 hover:bg-indigo-50/40 dark:border-neutral-800 dark:hover:border-indigo-900/50 dark:hover:bg-indigo-950/20 sm:flex-row sm:items-center sm:justify-between"
                              >
                                <div className="min-w-0">
                                  <p className="text-xs font-medium text-slate-500 dark:text-neutral-400">
                                    {row.course.title}
                                    {courseSectionSubtitle(row.course, t)
                                      ? ` · ${courseSectionSubtitle(row.course, t)}`
                                      : ''}
                                  </p>
                                  <p className="truncate text-sm font-semibold text-slate-900 dark:text-neutral-100">
                                    {it.title}
                                  </p>
                                  <p className="mt-0.5 flex items-center gap-1 text-xs text-slate-500 dark:text-neutral-400">
                                    <CalendarDays className="h-3.5 w-3.5 shrink-0" aria-hidden />
                                    <DeadlineDateTime
                                      iso={it.dueAt!}
                                      courseTimezone={row.course.courseTimezone}
                                    />
                                  </p>
                                </div>
                                {g ? (
                                  <div className="flex shrink-0 flex-col items-start gap-1 sm:items-end">
                                    <span className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                                      {g.label}
                                    </span>
                                    <div className="h-1.5 w-full min-w-[96px] overflow-hidden rounded-full bg-slate-100 dark:bg-neutral-800 sm:w-28">
                                      <div
                                        className="h-full rounded-full bg-emerald-500"
                                        style={{ width: `${Math.round(g.pct)}%` }}
                                      />
                                    </div>
                                  </div>
                                ) : (
                                  <span className="text-xs text-slate-400 dark:text-neutral-500">—</span>
                                )}
                              </Link>
                            </li>
                          )
                        })}
                    </ul>
                    {studentRows.every((r) => dueThisWeekItems(r.structure, weekStart, weekEnd).length === 0) && (
                      <p className="mt-2 text-sm text-slate-500 dark:text-neutral-400">
                        {t('dashboard.learning.nothingDue')}{' '}
                        <Link
                          className="font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                          to="/calendar"
                        >
                          {t('dashboard.learning.openCalendar')}
                        </Link>{' '}
                        {t('dashboard.learning.fullSchedule')}
                      </p>
                    )}
                  </div>

                  <div className="mt-6 grid gap-4 md:grid-cols-2">
                    {studentRows.map((row) => {
                      const held = new Set(row.myGrades?.heldGradeItemIds ?? [])
                      const cols: GradebookColumnForFinal[] = (row.myGrades?.columns ?? [])
                        .filter((c) => !held.has(c.id))
                        .map((c) => ({
                          id: c.id,
                          maxPoints: c.maxPoints,
                          assignmentGroupId: c.assignmentGroupId ?? null,
                          neverDrop: c.neverDrop === true,
                          replaceWithFinal: c.replaceWithFinal === true,
                          dueAt: c.dueAt ?? null,
                        }))
                      const weights: AssignmentGroupWeight[] = (row.myGrades?.assignmentGroups ?? []).map((g) => ({
                        id: g.id,
                        weightPercent: g.weightPercent,
                        dropLowest: g.dropLowest,
                        dropHighest: g.dropHighest,
                        replaceLowestWithFinal: g.replaceLowestWithFinal,
                      }))
                      const exc: Record<string, boolean> = {}
                      for (const [k, v] of Object.entries(row.myGrades?.gradeStatuses ?? {})) {
                        if (v === 'excused') exc[k] = true
                      }
                      const finalPct = computeCourseFinalPercent(cols, row.myGrades?.grades ?? {}, weights, exc)
                      const base = `/courses/${encodeURIComponent(row.course.courseCode)}`
                      return (
                        <div
                          key={row.course.courseCode}
                          className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
                        >
                          <div className="flex flex-wrap items-center gap-2">
                            <Link
                              to={base}
                              className="text-base font-semibold text-slate-900 hover:text-indigo-600 dark:text-neutral-100 dark:hover:text-indigo-300"
                            >
                              {row.course.title}
                            </Link>
                            {ffEnrollmentStateMachine &&
                            row.course.viewerEnrollmentState &&
                            row.course.viewerEnrollmentState !== 'active' ? (
                              <EnrollmentStateBadge
                                state={row.course.viewerEnrollmentState as EnrollmentState}
                                changedAt={row.course.viewerEnrollmentStateChangedAt}
                              />
                            ) : null}
                          </div>
                          {courseSectionSubtitle(row.course, t) ? (
                            <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                              {courseSectionSubtitle(row.course, t)}
                            </p>
                          ) : null}
                          <p className="mt-3 text-xs font-medium uppercase tracking-wide text-slate-400 dark:text-neutral-500">
                            {t('dashboard.learning.courseGrade')}
                          </p>
                          <p className="mt-1 text-2xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
                            {row.myGrades ? formatFinalPercent(finalPct) : '—'}
                          </p>
                          <div className="mt-4 flex flex-wrap gap-2">
                            <Link
                              to={`${base}/modules`}
                              className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                            >
                              <ClipboardList className="h-3.5 w-3.5" aria-hidden />
                              {t('dashboard.learning.modules')}
                            </Link>
                            <Link
                              to={`${base}/feed`}
                              className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                            >
                              <MessageCircle className="h-3.5 w-3.5" aria-hidden />
                              {t('dashboard.learning.feed')}
                            </Link>
                            <Link
                              to={`${base}/my-grades`}
                              className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                            >
                              <Sparkles className="h-3.5 w-3.5" aria-hidden />
                              {t('dashboard.learning.grades')}
                            </Link>
                          </div>
                        </div>
                      )
                    })}
                  </div>

                  {announcements.length > 0 && (
                    <div className="mt-8">
                      <h3 className="text-sm font-semibold text-slate-800 dark:text-neutral-100">
                        {t('dashboard.learning.announcementsTitle')}
                      </h3>
                      <ul className="mt-3 space-y-3">
                        {announcements.map((a) => (
                          <li
                            key={`${a.courseCode}-${a.createdAt}`}
                            className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
                          >
                            <div className="flex flex-wrap items-center gap-2">
                              <Megaphone className="h-4 w-4 text-amber-500" aria-hidden />
                              <Link
                                to={`/courses/${encodeURIComponent(a.courseCode)}/feed`}
                                className="text-sm font-semibold text-slate-900 hover:text-indigo-600 dark:text-neutral-100 dark:hover:text-indigo-300"
                              >
                                {a.courseTitle}
                              </Link>
                              {a.pinned ? (
                                <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[0.65rem] font-semibold uppercase tracking-wide text-amber-900 dark:bg-amber-950/60 dark:text-amber-100">
                                  {t('dashboard.learning.pinned')}
                                </span>
                              ) : null}
                            </div>
                            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                              {a.channelName} · {a.author} ·{' '}
                              {formatDateTime(a.createdAt, {
                                dateStyle: 'medium',
                                timeStyle: 'short',
                              })}
                            </p>
                            <p className="mt-2 text-sm leading-relaxed text-slate-700 dark:text-neutral-200">
                              {a.snippet}
                            </p>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </>
              )}
            </section>
          )}

          {anyStaffExperience && (
            <section aria-label={t('dashboard.teaching.ariaLabel')}>
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                  {t('dashboard.teaching.title')}
                </h2>
                <button
                  onClick={() => toggleSection('teaching-overview')}
                  className="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-neutral-800 dark:hover:text-neutral-300"
                  aria-label={
                    collapsedSections['teaching-overview']
                      ? t('dashboard.teaching.expandAria')
                      : t('dashboard.teaching.collapseAria')
                  }
                >
                  {collapsedSections['teaching-overview'] ? (
                    <ChevronDown className="h-4 w-4" />
                  ) : (
                    <ChevronUp className="h-4 w-4" />
                  )}
                </button>
              </div>

              {!collapsedSections['teaching-overview'] && (
                <>
                  {allGradingBacklog.length > 0 ? (
                    <div className="mt-4 rounded-2xl border border-amber-100 bg-amber-50/50 p-5 shadow-sm dark:border-amber-900/40 dark:bg-amber-950/20">
                      <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                        {t('dashboard.teaching.needsGrading')}
                      </h3>
                      <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
                        {t('dashboard.teaching.needsGradingHint')}
                      </p>
                      <div className="mt-3">
                        <GradingBacklogList items={allGradingBacklog} showCourse />
                      </div>
                    </div>
                  ) : null}
                  <div className="mt-4 grid gap-4 md:grid-cols-2">
                  {staffRows.map((row) => {
                    const base = `/courses/${encodeURIComponent(row.course.courseCode)}`
                    return (
                      <div
                        key={row.course.courseCode}
                        className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
                      >
                        <Link
                          to={base}
                          className="text-base font-semibold text-slate-900 hover:text-indigo-600 dark:text-neutral-100 dark:hover:text-indigo-300"
                        >
                          {row.course.title}
                        </Link>
                        {courseSectionSubtitle(row.course, t) ? (
                          <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                            {courseSectionSubtitle(row.course, t)}
                          </p>
                        ) : null}
                        {row.gradingBacklog.length > 0 ? (
                          <div className="mt-4">
                            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                              {t('dashboard.teaching.ungradedSubmissions')}
                            </p>
                            <div className="mt-2">
                              <GradingBacklogList items={row.gradingBacklog} />
                            </div>
                          </div>
                        ) : (
                          <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">
                            {t('dashboard.teaching.noUngraded')}
                          </p>
                        )}
                        <dl className="mt-4 space-y-3 text-sm">
                          <div className="flex justify-between gap-3">
                            <dt className="text-slate-500 dark:text-neutral-400">
                              {t('dashboard.teaching.gradebookGaps')}
                            </dt>
                            <dd className="font-semibold text-slate-900 dark:text-neutral-100">
                              {row.emptyGradeCells == null ? (
                                <span className="text-slate-400">{t('dashboard.teaching.noAccess')}</span>
                              ) : (
                                <>
                                  {row.emptyGradeCells}{' '}
                                  <span className="font-normal text-slate-500 dark:text-neutral-400">
                                    {t('dashboard.teaching.emptyCells')}
                                  </span>
                                </>
                              )}
                            </dd>
                          </div>
                        </dl>
                        <div className="mt-4 flex flex-wrap gap-2">
                          <Link
                            to={`${base}/gradebook`}
                            className="inline-flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500"
                          >
                            {t('dashboard.teaching.openGradebook')}
                          </Link>
                          <Link
                            to={`${base}/modules`}
                            className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                          >
                            {t('dashboard.teaching.modules')}
                          </Link>
                        </div>
                      </div>
                    )
                  })}
                </div>
                </>
              )}
            </section>
          )}

          {!anyStudentExperience && !anyStaffExperience && hasCourses && (
            <p className="text-sm text-slate-500 dark:text-neutral-400">
              {t('dashboard.noEnrollments')}
            </p>
          )}

          {deferredCourseCount > 0 && (
            <div className="flex justify-center pt-2">
              <button
                type="button"
                onClick={loadMoreCourses}
                disabled={loadingMoreCourses}
                className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:border-slate-300 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800"
              >
                {loadingMoreCourses
                  ? t('dashboard.loadMore.loading')
                  : t('dashboard.loadMore.button', { count: deferredCourseCount })}
              </button>
            </div>
          )}
        </div>
      )}
      {aiStudyBuddyEnabled && studyBuddyCourseCode ? (
        <StudyBuddyWidget courseCode={studyBuddyCourseCode} />
      ) : null}
    </LmsPage>
  )
}

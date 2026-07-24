/* eslint-disable react-hooks/set-state-in-effect -- sync breadcrumb async labels and cache when the route or course changes */
import { useEffect, useMemo, useState } from 'react'
import { ChevronRight } from 'lucide-react'
import { Link, matchPath, useLocation, useSearchParams } from 'react-router-dom'
import {
  fetchCourse,
  fetchCourseStructure,
  type CourseStructureItem,
} from '../../lib/courses-api'
import { listCourseFiles, type FolderBreadcrumb } from '../../lib/course-files-api'
import {
  courseSettingsSectionFromPathname,
  settingsViewFromPathname,
  type CourseSettingsSection,
} from './side-nav-path-utils'

const courseTitleCache = new Map<string, string>()
const structureCache = new Map<string, CourseStructureItem[]>()
const fileFolderCache = new Map<string, FolderBreadcrumb[]>()

type Crumb = { key: string; label: string; to?: string }

const COURSE_SETTINGS_LABEL: Record<CourseSettingsSection, string> = {
  general: 'General',
  grading: 'Grading',
  'grading-agents': 'Grading agents',
  plagiarism: 'Plagiarism',
  outcomes: 'Outcomes',
  badges: 'Badges',
  features: 'Features',
  accessibility: 'Accessibility',
  translations: 'Translations',
  sections: 'Sections',
  'import-export': 'Import / export',
  blueprint: 'Blueprint',
  archive: 'Archive',
}

function findItemModuleTitles(
  items: CourseStructureItem[],
  itemId: string,
): { itemTitle: string; moduleTitle: string | null } {
  const byId = new Map(items.map((i) => [i.id, i]))
  const item = byId.get(itemId)
  if (!item) return { itemTitle: 'Item', moduleTitle: null }
  let moduleTitle: string | null = null
  let cur: CourseStructureItem | undefined = item
  while (cur) {
    if (cur.kind === 'module') {
      moduleTitle = cur.title
      break
    }
    cur = cur.parentId ? byId.get(cur.parentId) : undefined
  }
  return { itemTitle: item.title, moduleTitle }
}

function settingsSubLabel(view: ReturnType<typeof settingsViewFromPathname>): string {
  switch (view) {
    case 'account':
      return 'Account'
    case 'notifications':
      return 'Notifications'
    case 'learner-profile':
      return 'Learner Profile'
    case 'roles':
      return 'Roles and Permissions'
    case 'ai-models':
      return 'Models'
    case 'ai-prompts':
      return 'System Prompts'
    case 'ai-reports':
      return 'Reports'
    case 'ai-governance':
      return 'Governance'
    case 'lti-tools':
      return 'LTI tools'
    case 'platform':
      return 'Global platform'
    case 'intro-course':
      return 'Intro course'
    case 'organizations':
      return 'Organizations'
    case 'org-units':
      return 'Org structure'
    case 'org-branding':
      return 'Organization branding'
    case 'terms':
      return 'Academic terms'
    case 'scim-provisioning':
      return 'SCIM provisioning'
    case 'cloud-providers':
      return 'Cloud file pickers'
    case 'oer-providers':
      return 'OER library'
    default:
      return 'Account'
  }
}

/** Static trail from pathname only (course title / structure filled elsewhere). */
function staticCrumbsFromPathname(pathname: string, courseCode: string | null): Crumb[] {
  if (pathname === '/') return [{ key: 'dash', label: 'Dashboard' }]

  if (pathname === '/courses') return [{ key: 'courses', label: 'Courses' }]
  if (pathname === '/courses/create') {
    return [
      { key: 'courses', label: 'Courses', to: '/courses' },
      { key: 'create', label: 'Create course' },
    ]
  }

  if (pathname === '/notebooks') return [{ key: 'notebooks', label: 'Notebooks' }]
  if (pathname === '/calendar') return [{ key: 'cal', label: 'Calendar' }]
  if (pathname === '/inbox') return [{ key: 'inbox', label: 'Inbox' }]
  if (pathname === '/reports') return [{ key: 'reports', label: 'Reports' }]

  if (pathname === '/admin/accommodations/audit') {
    return [
      { key: 'admin', label: 'Admin' },
      { key: 'acc', label: 'Accommodations', to: '/admin/accommodations' },
      { key: 'audit', label: 'Audit report' },
    ]
  }

  if (pathname === '/admin/accommodations') {
    return [
      { key: 'admin', label: 'Admin' },
      { key: 'acc', label: 'Accommodations' },
    ]
  }

  if (pathname === '/admin/bookstore') {
    return [
      { key: 'admin', label: 'Admin' },
      { key: 'bookstore', label: 'Bookstore integration' },
    ]
  }

  if (pathname.startsWith('/settings')) {
    const view = settingsViewFromPathname(pathname)
    return [
      { key: 'uset', label: 'User settings', to: '/settings/account' },
      { key: 'leaf', label: settingsSubLabel(view) },
    ]
  }

  if (!courseCode) return []

  const enc = encodeURIComponent(courseCode)
  const base = `/courses/${enc}`

  const courseCrumb = (label: string, withLink: boolean): Crumb => ({
    key: 'course',
    label,
    to: withLink ? base : undefined,
  })

  const onCourseSettings =
    pathname === `${base}/settings` || pathname.startsWith(`${base}/settings/`)
  if (courseCode && onCourseSettings) {
    const section = courseSettingsSectionFromPathname(pathname)
    const sectionLabel = COURSE_SETTINGS_LABEL[section]
    return [
      courseCrumb(courseCode, true),
      { key: 'settings', label: 'Settings', to: `${base}/settings/general` },
      { key: 'sec', label: sectionLabel },
    ]
  }

  if (pathname === base || pathname === `${base}/`) {
    return [courseCrumb(courseCode, false)]
  }

  if (pathname === `${base}/feed`) {
    return [courseCrumb(courseCode, true), { key: 'feed', label: 'Feed' }]
  }
  if (pathname === `${base}/discussions`) {
    return [courseCrumb(courseCode, true), { key: 'disc', label: 'Discussions' }]
  }
  if (pathname === `${base}/syllabus`) {
    return [courseCrumb(courseCode, true), { key: 'syl', label: 'Syllabus' }]
  }
  if (pathname.startsWith(`${base}/files`)) {
    return [courseCrumb(courseCode, true), { key: 'files', label: 'Files' }]
  }
  if (pathname === `${base}/modules`) {
    return [courseCrumb(courseCode, true), { key: 'mod', label: 'Modules' }]
  }
  if (pathname === `${base}/questions`) {
    return [courseCrumb(courseCode, true), { key: 'qb', label: 'Question bank' }]
  }
  if (pathname === `${base}/notebook`) {
    return [courseCrumb(courseCode, true), { key: 'nb', label: 'Notebook' }]
  }
  if (pathname === `${base}/calendar`) {
    return [courseCrumb(courseCode, true), { key: 'cal', label: 'Calendar' }]
  }
  if (pathname === `${base}/live`) {
    return [courseCrumb(courseCode, true), { key: 'live', label: 'Live Sessions' }]
  }
  if (pathname === `${base}/office-hours`) {
    return [courseCrumb(courseCode, true), { key: 'office-hours', label: 'Office Hours' }]
  }
  if (pathname === `${base}/my-grades`) {
    return [courseCrumb(courseCode, true), { key: 'mg', label: 'My grades' }]
  }
  if (pathname === `${base}/gradebook`) {
    return [courseCrumb(courseCode, true), { key: 'gb', label: 'Gradebook' }]
  }
  if (pathname === `${base}/reports`) {
    return [courseCrumb(courseCode, true), { key: 'reports', label: 'Reports' }]
  }
  if (pathname === `${base}/standards-gradebook`) {
    return [courseCrumb(courseCode, true), { key: 'sgb', label: 'Standards gradebook' }]
  }
  if (pathname === `${base}/standards-coverage`) {
    return [courseCrumb(courseCode, true), { key: 'st', label: 'Standards coverage' }]
  }
  if (pathname === `${base}/enrollments`) {
    return [courseCrumb(courseCode, true), { key: 'enr', label: 'Enrollments' }]
  }

  const modItem = matchModuleItemRoute(pathname)
  if (modItem && modItem.code === courseCode) {
    return [
      courseCrumb(courseCode, true),
      { key: 'modules', label: 'Modules', to: `${base}/modules` },
      { key: 'modname', label: '\u00a0', to: `${base}/modules` },
      { key: 'item', label: '…' },
    ]
  }

  return []
}

function mergeCourseTitle(crumbs: Crumb[], courseTitle: string | null, courseCode: string): Crumb[] {
  return crumbs.map((c) =>
    c.key === 'course' ? { ...c, label: courseTitle?.trim() || courseCode } : c,
  )
}

function mergeModuleItem(
  crumbs: Crumb[],
  moduleTitle: string | null,
  itemTitle: string,
): Crumb[] {
  return crumbs.map((c) => {
    if (c.key === 'modname') {
      if (moduleTitle) return { ...c, label: moduleTitle, to: c.to }
      return { ...c, label: '', to: undefined }
    }
    if (c.key === 'item') return { ...c, label: itemTitle }
    return c
  }).filter((c) => c.label.trim().length > 0)
}

function mergeFileFolderCrumbs(
  crumbs: Crumb[],
  courseCode: string,
  breadcrumbs: FolderBreadcrumb[],
): Crumb[] {
  if (!breadcrumbs.length) return crumbs
  const filesBase = `/courses/${encodeURIComponent(courseCode)}/files`
  return [
    ...crumbs.map((c) => (c.key === 'files' ? { ...c, to: filesBase } : c)),
    ...breadcrumbs.map((b, i) => ({
      key: `ff-${b.id}`,
      label: b.name,
      to:
        i < breadcrumbs.length - 1
          ? `${filesBase}?folder=${encodeURIComponent(b.id)}`
          : undefined,
    })),
  ]
}

const MODULE_ITEM_PATTERNS = [
  '/courses/:courseCode/modules/content/:itemId',
  '/courses/:courseCode/modules/assignment/:itemId',
  '/courses/:courseCode/modules/quiz/:itemId',
  '/courses/:courseCode/modules/external-link/:itemId',
  '/courses/:courseCode/modules/h5p/:itemId',
  '/courses/:courseCode/modules/scorm/:itemId',
  '/courses/:courseCode/modules/lti/:itemId',
  '/courses/:courseCode/modules/vibe-activity/:itemId',
] as const

function matchModuleItemRoute(pathname: string): { code: string; id: string } | null {
  const normalized = pathname.replace(/\/attempt\/?$/, '')
  for (const p of MODULE_ITEM_PATTERNS) {
    const m = matchPath({ path: p, end: true }, normalized)
    if (m?.params.itemId && m.params.courseCode) {
      return { code: m.params.courseCode, id: m.params.itemId }
    }
  }
  return null
}

export function TopBarBreadcrumbs() {
  const { pathname } = useLocation()
  const [searchParams] = useSearchParams()
  const folderId = searchParams.get('folder')
  const courseCode = useMemo(() => {
    const m = matchPath({ path: '/courses/:courseCode/*', end: false }, pathname)
    const code = m?.params.courseCode
    return code && code !== 'create' ? code : null
  }, [pathname])

  const [courseTitle, setCourseTitle] = useState<string | null>(() =>
    courseCode ? courseTitleCache.get(courseCode) ?? null : null,
  )

  const [itemTrail, setItemTrail] = useState<{ moduleTitle: string | null; itemTitle: string } | null>(
    null,
  )

  const [fileFolderTrail, setFileFolderTrail] = useState<FolderBreadcrumb[] | null>(null)

  useEffect(() => {
    if (!courseCode) {
      setCourseTitle(null)
      return
    }
    const cached = courseTitleCache.get(courseCode)
    if (cached) {
      setCourseTitle(cached)
      return
    }
    let cancelled = false
    void (async () => {
      try {
        const c = await fetchCourse(courseCode)
        if (cancelled) return
        courseTitleCache.set(courseCode, c.title)
        setCourseTitle(c.title)
      } catch {
        if (!cancelled) setCourseTitle(null)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode])

  useEffect(() => {
    const picked = matchModuleItemRoute(pathname)
    if (!picked?.code || !picked.id) {
      setItemTrail(null)
      return
    }
    const { code, id } = picked
    const cached = structureCache.get(code)
    if (cached) {
      setItemTrail(findItemModuleTitles(cached, id))
      return
    }
    let cancelled = false
    void (async () => {
      try {
        const items = await fetchCourseStructure(code)
        if (cancelled) return
        structureCache.set(code, items)
        setItemTrail(findItemModuleTitles(items, id))
      } catch {
        if (!cancelled) setItemTrail({ moduleTitle: null, itemTitle: 'Item' })
      }
    })()
    return () => {
      cancelled = true
    }
  }, [pathname])

  useEffect(() => {
    if (!courseCode) {
      setFileFolderTrail(null)
      return
    }
    const filesBase = `/courses/${encodeURIComponent(courseCode)}/files`
    if (!pathname.startsWith(`${filesBase}`)) {
      setFileFolderTrail(null)
      return
    }
    if (!folderId) {
      setFileFolderTrail([])
      return
    }
    const cacheKey = `${courseCode}:${folderId}`
    const cached = fileFolderCache.get(cacheKey)
    if (cached) {
      setFileFolderTrail(cached)
      return
    }
    let cancelled = false
    void (async () => {
      try {
        const contents = await listCourseFiles(courseCode, folderId)
        if (cancelled) return
        const trail = contents.breadcrumbs ?? []
        fileFolderCache.set(cacheKey, trail)
        setFileFolderTrail(trail)
      } catch {
        if (!cancelled) setFileFolderTrail([])
      }
    })()
    return () => {
      cancelled = true
    }
  }, [pathname, courseCode, folderId])

  const crumbs = useMemo(() => {
    let base = staticCrumbsFromPathname(pathname, courseCode)
    if (courseCode && base.some((c) => c.key === 'course')) {
      base = mergeCourseTitle(base, courseTitle, courseCode)
    }
    if (itemTrail && base.some((c) => c.key === 'item')) {
      base = mergeModuleItem(base, itemTrail.moduleTitle, itemTrail.itemTitle)
    }
    if (base.some((c) => c.key === 'files') && folderId) {
      if (fileFolderTrail === null) {
        const filesBase = `/courses/${encodeURIComponent(courseCode!)}/files`
        base = [
          ...base.map((c) => (c.key === 'files' ? { ...c, to: filesBase } : c)),
          { key: 'ff-loading', label: '…' },
        ]
      } else if (fileFolderTrail.length > 0) {
        base = mergeFileFolderCrumbs(base, courseCode!, fileFolderTrail)
      }
    }
    return base
  }, [pathname, courseCode, courseTitle, itemTrail, folderId, fileFolderTrail])

  if (!crumbs.length) return null

  return (
    <nav aria-label="Breadcrumb" className="min-w-0 flex-1 basis-0 overflow-hidden ps-1 sm:ps-0">
      <ol className="m-0 flex list-none items-center gap-0.5 p-0 text-xs text-slate-600 sm:text-sm dark:text-neutral-400">
        {crumbs.map((c, i) => {
          const last = i === crumbs.length - 1
          return (
            <li key={c.key + String(i)} className="flex min-w-0 items-center gap-0.5">
              {i > 0 ? (
                <ChevronRight
                  className="h-3.5 w-3.5 shrink-0 text-slate-300 dark:text-neutral-600"
                  aria-hidden
                />
              ) : null}
              {last || !c.to ? (
                <span
                  className={`truncate ${last ? 'font-medium text-slate-900 dark:text-neutral-100' : ''}`}
                  aria-current={last ? 'page' : undefined}
                >
                  {c.label}
                </span>
              ) : (
                <Link
                  to={c.to}
                  className="truncate transition-[background-color,color,border-color] hover:text-indigo-600 dark:hover:text-indigo-400"
                >
                  {c.label}
                </Link>
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}

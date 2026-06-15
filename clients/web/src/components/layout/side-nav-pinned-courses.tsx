import { NavLink, useLocation } from 'react-router-dom'
import { useCoursePins } from '../../context/course-pinned-context'
import { heroImageObjectStyle } from '../../lib/hero-image-position'
import type { PinnedCourseSummary } from '../../lib/course-catalog-settings-api'
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

export function SideNavPinnedCourses() {
  const { pinnedCourses, loading, flashPinnedCourseId } = useCoursePins()
  const { sideNavCollapsed } = useShellNav()
  const location = useLocation()

  if (loading || pinnedCourses.length === 0) return null

  const cols = Math.min(pinnedCourses.length, 4)
  const gridColsClass = gridColsMap[cols] ?? 'grid-cols-4'

  return (
    <div
      className={`shrink-0 px-3 py-3 ${sideNavCollapsed ? 'flex flex-col items-center gap-1.5' : `grid ${gridColsClass} gap-1.5`}`}
      aria-label="Pinned courses"
    >
      {pinnedCourses.map((course) => {
        const href = `/courses/${encodeURIComponent(course.courseCode)}`
        const active =
          location.pathname === href || location.pathname.startsWith(`${href}/`)
        const title = pinnedCourseTitle(course)

        return (
          <SideNavTooltip
            key={course.id}
            content={title}
            hoverWhenExpanded
            instant={flashPinnedCourseId === course.id}
          >
            <NavLink
              to={href}
              aria-label={title}
              aria-current={active ? 'page' : undefined}
              className={[
                'group relative block overflow-hidden rounded-xl ring-1 ring-black/[0.06] transition hover:ring-indigo-400/50 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:ring-white/10 dark:hover:ring-indigo-400/40',
                sideNavCollapsed ? 'h-9 w-9' : 'h-10 w-full',
                active ? 'ring-2 ring-indigo-500 dark:ring-indigo-400' : '',
              ]
                .filter(Boolean)
                .join(' ')}
            >
              <CourseHeroImage
                src={course.heroImageUrl ?? '/course-card-hero.png'}
                alt=""
                draggable={false}
                loading="lazy"
                decoding="async"
                className="h-full w-full object-cover"
                style={heroImageObjectStyle(course.heroImageObjectPosition)}
              />
              <span
                className={[
                  'pointer-events-none absolute inset-0 bg-gradient-to-t from-black/35 to-transparent opacity-0 transition group-hover:opacity-100',
                  active ? 'opacity-100' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
                aria-hidden
              />
            </NavLink>
          </SideNavTooltip>
        )
      })}
    </div>
  )
}
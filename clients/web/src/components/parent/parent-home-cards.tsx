import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, CheckCircle2, Clock, GraduationCap } from 'lucide-react'
import { formatDateTime } from '../../lib/format'
import { learnerCourseItemHref } from '../../lib/courses-api'
import { formatTimeSpent, parentProgressHref } from '../../lib/dashboard-view-preference'
import type { ParentHomeChild, ParentHomeCourse, ParentHomeWorkItem } from '../../lib/parent-api'
import { LinkButton } from '../ui/link-button'

function WorkRow({ item, done }: { item: ParentHomeWorkItem; done?: boolean }) {
  const { t } = useTranslation('parent')
  const when = done ? item.completedAt : item.dueAt
  return (
    <li className="flex flex-wrap items-baseline justify-between gap-2 py-1.5 text-body-sm">
      <Link
        to={learnerCourseItemHref(item.courseCode, { kind: item.kind, id: item.itemId })}
        className="min-w-0 truncate font-medium text-fg-default underline-offset-2 hover:underline"
      >
        {item.title}
        <span className="ms-1 text-caption font-normal text-fg-default">· {item.courseTitle}</span>
      </Link>
      <span className="flex shrink-0 items-center gap-1 text-caption text-fg-default">
        {!done && item.completed && (
          <span className="inline-flex items-center gap-1 rounded bg-success-surface px-1.5 py-0.5 font-medium text-success-fg">
            <CheckCircle2 className="h-3 w-3" aria-hidden />
            {t('parent.home.turnedIn')}
          </span>
        )}
        {item.overdue && (
          <span className="inline-flex items-center gap-1 rounded bg-danger-surface px-1.5 py-0.5 font-medium text-danger-fg">
            <AlertTriangle className="h-3 w-3" aria-hidden />
            {t('parent.home.overdue')}
          </span>
        )}
        {when && (
          <time dateTime={when}>
            {done
              ? t('parent.home.completedAt', { date: formatDateTime(when) })
              : t('parent.grades.due', { date: formatDateTime(when) })}
          </time>
        )}
      </span>
    </li>
  )
}

function GradeRow({ course }: { course: ParentHomeCourse }) {
  const { t } = useTranslation('parent')
  const pct =
    course.gradePercent == null ? t('parent.home.noGrade') : `${course.gradePercent.toFixed(1)}%`
  const label = (
    <>
      <span className="min-w-0 truncate">{course.title}</span>
      <span className="shrink-0 font-medium tabular-nums">{pct}</span>
    </>
  )
  return (
    <li className="py-1.5 text-body-sm">
      {course.enrollmentId ? (
        <Link
          to={parentProgressHref(course.courseCode, course.enrollmentId)}
          className="flex justify-between gap-2 text-fg-default underline-offset-2 hover:underline"
        >
          {label}
        </Link>
      ) : (
        <span className="flex justify-between gap-2 text-fg-default">{label}</span>
      )}
    </li>
  )
}

function Section({
  title,
  icon,
  children,
}: {
  title: string
  icon: ReactNode
  children: ReactNode
}) {
  return (
    <div>
      <h4 className="flex items-center gap-1.5 text-overline font-semibold text-fg-default">
        {icon}
        {title}
      </h4>
      {children}
    </div>
  )
}

export function ParentHomeChildCard({ child }: { child: ParentHomeChild }) {
  const { t } = useTranslation('parent')
  const headingId = `parent-home-child-${child.studentUserId}`
  return (
    <section
      aria-labelledby={headingId}
      data-testid="parent-home-card"
      className="flex flex-col gap-4 rounded-xl border border-border-default bg-surface-raised p-4 shadow-sm"
    >
      <h3 id={headingId} className="text-subtitle font-semibold text-fg-default">
        {child.displayName}
      </h3>
      {child.courses.length === 0 ? (
        <div className="flex flex-col items-start gap-2">
          <p className="text-body-sm text-fg-muted">{t('parent.home.noCourses')}</p>
          <LinkButton to="/courses" variant="secondary" size="sm">
            {t('parent.home.addCourse')}
          </LinkButton>
        </div>
      ) : (
        <>
          <Section
            title={t('parent.home.dueThisWeek', { count: child.dueThisWeek.length })}
            icon={<Clock className="h-3.5 w-3.5" aria-hidden />}
          >
            {child.dueThisWeek.length === 0 ? (
              <p className="py-1.5 text-body-sm text-fg-muted">{t('parent.home.nothingDue')}</p>
            ) : (
              <ul className="divide-y divide-border-subtle">
                {child.dueThisWeek.map((it) => (
                  <WorkRow key={`${it.courseCode}-${it.itemId}`} item={it} />
                ))}
              </ul>
            )}
          </Section>
          <Section
            title={t('parent.home.completedThisWeek', { count: child.completedThisWeek.length })}
            icon={<CheckCircle2 className="h-3.5 w-3.5" aria-hidden />}
          >
            {child.completedThisWeek.length === 0 ? (
              <p className="py-1.5 text-body-sm text-fg-muted">{t('parent.home.nothingCompleted')}</p>
            ) : (
              <ul className="divide-y divide-border-subtle">
                {child.completedThisWeek.map((it) => (
                  <WorkRow key={`${it.courseCode}-${it.itemId}`} item={it} done />
                ))}
              </ul>
            )}
          </Section>
          <Section
            title={t('parent.home.grades')}
            icon={<GraduationCap className="h-3.5 w-3.5" aria-hidden />}
          >
            <ul className="divide-y divide-border-subtle">
              {child.courses.map((c) => (
                <GradeRow key={c.courseCode} course={c} />
              ))}
            </ul>
          </Section>
        </>
      )}
      <p className="flex items-center gap-1.5 border-t border-border-subtle pt-3 text-body-sm text-fg-muted">
        <Clock className="h-4 w-4" aria-hidden />
        {t('parent.home.timeSpent', { time: formatTimeSpent(child.timeSpentSeconds) })}
      </p>
    </section>
  )
}

/** One card per child; stacks to a single column on mobile. */
export function ParentHomeCards({ items }: { items: ParentHomeChild[] }) {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2" data-testid="parent-home-cards">
      {items.map((c) => (
        <ParentHomeChildCard key={c.studentUserId} child={c} />
      ))}
    </div>
  )
}

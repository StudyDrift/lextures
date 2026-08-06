import { useEffect, useMemo } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  COURSE_DESIGN_RESEARCH_HREF,
  listChecklistHelpRefs,
  resolveChecklistHelp,
} from '../../lib/checklist-help'
import { courseDesignResearchHref } from '../../lib/checklist-research-anchors'
import { courseChecklistI18n } from '../../lib/course-checklist-i18n'
import { LmsPage } from '../lms/lms-page'

/**
 * Support URL destination for HelpRef `course-checklist#<slug>` (CC.10 FR-3).
 * Renders every catalog entry so deep links from the help popover resolve.
 * Lives inside AppShell for side nav + top bar.
 */
export default function CourseChecklistHelpPage() {
  const { hash } = useLocation()
  const entries = useMemo(() => {
    return listChecklistHelpRefs()
      .map((ref) => resolveChecklistHelp(ref))
      .filter((e): e is NonNullable<typeof e> => e != null)
      .sort((a, b) => a.title.localeCompare(b.title))
  }, [])

  useEffect(() => {
    const id = hash.replace(/^#/, '')
    if (!id) return
    requestAnimationFrame(() => {
      document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }, [hash])

  return (
    <LmsPage
      title="Designing a good course"
      description="Guidance for every course checklist item: what the check looks at, why it matters, how to satisfy it, and when dismissal is reasonable."
      actions={
        <Link
          to={COURSE_DESIGN_RESEARCH_HREF}
          className="inline-flex min-h-11 items-center rounded-lg border border-border-default bg-surface-raised px-3 text-sm font-medium text-fg-muted hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
        >
          Rule-to-standard mapping
        </Link>
      }
    >
      <ul className="max-w-3xl space-y-6">
        {entries.map((entry) => {
          const slug = entry.helpRef.includes('#') ? entry.helpRef.split('#')[1] : entry.helpRef
          return (
            <li
              key={entry.helpRef}
              id={slug}
              className="scroll-mt-6 rounded-xl border border-border-default bg-surface-raised p-5 shadow-sm dark:border-border-default dark:bg-surface-raised"
            >
              <h2 className="text-base font-semibold text-fg-default">
                {entry.title}
              </h2>
              <dl className="mt-3 space-y-3 text-sm text-fg-muted">
                <div>
                  <dt className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
                    {courseChecklistI18n.helpWhat}
                  </dt>
                  <dd className="mt-1">{entry.what}</dd>
                </div>
                <div>
                  <dt className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
                    {courseChecklistI18n.helpWhy}
                  </dt>
                  <dd className="mt-1">{entry.why}</dd>
                </div>
                <div>
                  <dt className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
                    {courseChecklistI18n.helpHow}
                  </dt>
                  <dd className="mt-1">{entry.how}</dd>
                </div>
                <div>
                  <dt className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
                    {courseChecklistI18n.helpWhenDismiss}
                  </dt>
                  <dd className="mt-1">{entry.whenToDismiss}</dd>
                </div>
              </dl>
              {entry.sources.length > 0 ? (
                <ul className="mt-3 flex flex-wrap gap-1.5">
                  {entry.sources.map((src) => (
                    <li key={src}>
                      <Link
                        to={courseDesignResearchHref(src)}
                        className="rounded bg-surface-sunken px-1.5 py-0.5 text-[11px] font-medium text-fg-muted underline-offset-2 hover:underline dark:bg-surface-overlay dark:text-fg-muted"
                      >
                        {src}
                      </Link>
                    </li>
                  ))}
                </ul>
              ) : null}
            </li>
          )
        })}
      </ul>
    </LmsPage>
  )
}

import { useMemo } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { ChecklistResearchBody } from '../components/checklist/checklist-research-body'
import { COURSE_DESIGN_RESEARCH_HREF } from '../lib/checklist-help'
import { buildSourceIndex } from '../lib/checklist-research-anchors'
import { LmsPage } from './lms/lms-page'

/**
 * Full-page support URL for the checklist rule-to-standard mapping (CC.10 FR-4).
 * Source chips open an in-page dialog with the same anchors; this route is for deep links.
 */
export default function CourseDesignResearchPage() {
  const { hash } = useLocation()
  const focusSource = useMemo(() => {
    const id = hash.replace(/^#/, '')
    if (!id.startsWith('src-')) return null
    const entry = buildSourceIndex().find((e) => e.anchorId === id)
    return entry?.source ?? null
  }, [hash])

  return (
    <LmsPage
      title="Course-design research"
      description="Where checklist rules come from: the rule-to-standard mapping for Quality Matters, OSCQR, NSQ, UDL, and WCAG."
      actions={
        <Link
          to="/help/course-checklist"
          className="inline-flex min-h-11 items-center rounded-lg border border-slate-200 bg-white px-3 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
        >
          Checklist help hub
        </Link>
      }
    >
      <ChecklistResearchBody className="max-w-3xl" focusSource={focusSource} />
      <p className="mt-10 max-w-3xl text-xs text-slate-500 dark:text-neutral-500">
        Canonical path: <code className="font-mono">{COURSE_DESIGN_RESEARCH_HREF}</code>
        {focusSource ? (
          <>
            {' '}
            · focused on <strong>{focusSource}</strong>
          </>
        ) : null}
      </p>
    </LmsPage>
  )
}

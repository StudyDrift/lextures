// Dashboard section listing the learner's self-paced courses with a progress ring
// and a "Resume" CTA (plan 15.2, FR-4/FR-5).
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowRight } from 'lucide-react'
import {
  fetchSelfPacedEnrollments,
  formatProgressLabel,
  type SelfPacedEnrollment,
} from '../../lib/self-paced-api'

/** Small SVG progress ring with an accessible value. */
function ProgressRing({ percent }: { percent: number }) {
  const clamped = Math.max(0, Math.min(100, Math.round(percent)))
  const r = 16
  const c = 2 * Math.PI * r
  const offset = c - (clamped / 100) * c
  return (
    <svg
      width={44}
      height={44}
      viewBox="0 0 44 44"
      role="img"
      aria-label={formatProgressLabel(clamped)}
      className="shrink-0"
    >
      <circle cx={22} cy={22} r={r} fill="none" stroke="currentColor" strokeWidth={4} className="text-slate-200 dark:text-slate-700" />
      <circle
        cx={22}
        cy={22}
        r={r}
        fill="none"
        stroke="currentColor"
        strokeWidth={4}
        strokeLinecap="round"
        strokeDasharray={c}
        strokeDashoffset={offset}
        transform="rotate(-90 22 22)"
        className="text-emerald-500"
      />
      <text x={22} y={26} textAnchor="middle" className="fill-slate-700 text-[10px] font-semibold dark:fill-slate-200">
        {clamped}%
      </text>
    </svg>
  )
}

export function SelfPacedDashboardSection() {
  const [rows, setRows] = useState<SelfPacedEnrollment[]>([])
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    let cancelled = false
    fetchSelfPacedEnrollments()
      .then((r) => {
        if (!cancelled) {
          setRows(r)
          setLoaded(true)
        }
      })
      .catch(() => {
        if (!cancelled) setLoaded(true)
      })
    return () => {
      cancelled = true
    }
  }, [])

  if (!loaded || rows.length === 0) return null

  return (
    <section aria-label="Self-paced courses">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        Self-paced
      </h2>
      <ul className="mt-3 space-y-3">
        {rows.map((row) => (
          <li
            key={row.enrollmentId}
            className="flex items-center gap-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800"
          >
            <ProgressRing percent={row.progressPercent} />
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-slate-900 dark:text-neutral-50">
                {row.title}
              </p>
              <p className="text-xs text-slate-500 dark:text-neutral-400">
                {row.completed
                  ? 'Completed'
                  : `${row.completedItems} of ${row.totalItems} items`}
              </p>
            </div>
            <Link
              to={`/courses/${encodeURIComponent(row.courseCode)}/modules`}
              aria-label={`Resume ${row.title}`}
              className="inline-flex shrink-0 items-center gap-1 rounded-xl bg-emerald-600 px-3 py-2 text-xs font-semibold text-white transition-[background-color,color,border-color] hover:bg-emerald-500"
            >
              {row.completed ? 'Review' : 'Resume'}
              <ArrowRight className="h-3.5 w-3.5" aria-hidden />
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}

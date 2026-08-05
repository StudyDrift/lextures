import { useMemo, useRef, useState, useEffect, type UIEvent } from 'react'
import { Link } from 'react-router-dom'
import type {
  ChecklistEvidence,
  ChecklistNavTarget,
} from '../../../lib/course-checklist-api-schemas'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import { hrefForTarget } from '../../../lib/use-focus-anchor'

const VIRTUALIZE_THRESHOLD = 100
const ROW_HEIGHT = 44
const VIEWPORT_ROWS = 12

type ChecklistEvidenceTableProps = {
  evidence: ChecklistEvidence
  fallbackTarget?: ChecklistNavTarget | null
  courseCode?: string
  onRowNavigate?: (route: string) => void
}

function resolveHref(
  rowTarget: ChecklistNavTarget | null | undefined,
  fallback: ChecklistNavTarget | null | undefined,
  courseCode?: string,
): string | null {
  const t = rowTarget ?? fallback
  if (!t?.route) return null
  const href = hrefForTarget(
    {
      route: t.route,
      anchor: t.anchor,
      entityKey: t.entityKey,
    },
    courseCode ? { courseCode } : undefined,
  )
  return href || null
}

export function ChecklistEvidenceTable({
  evidence,
  fallbackTarget,
  courseCode,
  onRowNavigate,
}: ChecklistEvidenceTableProps) {
  const rows = evidence.rows
  const virtualize = rows.length > VIRTUALIZE_THRESHOLD
  const [scrollTop, setScrollTop] = useState(0)
  const scrollerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setScrollTop(0)
    if (scrollerRef.current) scrollerRef.current.scrollTop = 0
  }, [rows.length])

  const windowed = useMemo(() => {
    if (!virtualize) return { start: 0, end: rows.length, offsetY: 0, totalH: 0 }
    const start = Math.max(0, Math.floor(scrollTop / ROW_HEIGHT) - 2)
    const end = Math.min(rows.length, start + VIEWPORT_ROWS + 4)
    return {
      start,
      end,
      offsetY: start * ROW_HEIGHT,
      totalH: rows.length * ROW_HEIGHT,
    }
  }, [virtualize, scrollTop, rows.length])

  const visibleRows = virtualize ? rows.slice(windowed.start, windowed.end) : rows
  const truncatedAt = evidence.truncatedAt ?? null

  const onScroll = (e: UIEvent<HTMLDivElement>) => {
    setScrollTop(e.currentTarget.scrollTop)
  }

  const table = (
    <table className="w-full min-w-[20rem] border-collapse text-sm">
      <thead>
        <tr className="border-b border-slate-200 text-start dark:border-neutral-700">
          {evidence.columns.map((col) => (
            <th
              key={col}
              scope="col"
              className="px-3 py-2 text-start text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400"
            >
              {col}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {virtualize ? <tr style={{ height: windowed.offsetY }} aria-hidden><td colSpan={evidence.columns.length} /></tr> : null}
        {visibleRows.map((row, idx) => {
          const route = resolveHref(row.target, fallbackTarget, courseCode)
          const cells = [row.label, row.sublabel ?? '', row.status, '']
          const key = `${row.label}-${windowed.start + idx}`
          return (
            <tr
              key={key}
              className="border-b border-slate-100 last:border-0 dark:border-neutral-800 max-sm:block max-sm:border max-sm:rounded-lg max-sm:mb-2 max-sm:p-2"
            >
              {evidence.columns.map((col, colIdx) => {
                const value = cells[colIdx] ?? ''
                const isFirst = colIdx === 0
                return (
                  <td
                    key={col}
                    className="px-3 py-2 text-slate-800 dark:text-neutral-200 max-sm:block max-sm:px-0 max-sm:py-1"
                    data-label={col}
                  >
                    {isFirst && route ? (
                      <Link
                        to={route}
                        className="inline-flex min-h-11 items-center font-medium text-amber-800 underline-offset-2 hover:underline dark:text-amber-300"
                        onClick={() => onRowNavigate?.(route)}
                      >
                        {row.label}
                      </Link>
                    ) : isFirst ? (
                      <span className="font-medium">{row.label}</span>
                    ) : (
                      <span className="max-sm:before:content-[attr(data-label)_':_'] max-sm:before:font-semibold max-sm:before:text-slate-500">
                        {value}
                      </span>
                    )}
                  </td>
                )
              })}
            </tr>
          )
        })}
        {virtualize ? (
          <tr style={{ height: Math.max(0, windowed.totalH - windowed.offsetY - visibleRows.length * ROW_HEIGHT) }} aria-hidden>
            <td colSpan={evidence.columns.length} />
          </tr>
        ) : null}
      </tbody>
    </table>
  )

  return (
    <div className="mt-3 overflow-x-auto rounded-lg border border-slate-200 dark:border-neutral-700">
      {truncatedAt != null ? (
        <p className="border-b border-slate-200 px-3 py-2 text-xs text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
          {courseChecklistI18n.evidenceTruncated(rows.length, truncatedAt)}
        </p>
      ) : null}
      {virtualize ? (
        <div
          ref={scrollerRef}
          className="max-h-[528px] overflow-y-auto"
          onScroll={onScroll}
        >
          {table}
        </div>
      ) : (
        table
      )}
    </div>
  )
}

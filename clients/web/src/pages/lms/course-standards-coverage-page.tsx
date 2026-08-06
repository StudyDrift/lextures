import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  fetchCourse,
  fetchCourseStandardsCoverage,
  type CoursePublic,
  type StandardCoverageItem,
} from '../../lib/courses-api'

type SortKey = 'code' | 'questionCount' | 'coverageStatus' | 'averageMastery'

function csvEscape(s: string): string {
  if (s.includes('"') || s.includes(',') || s.includes('\n')) {
    return `"${s.replace(/"/g, '""')}"`
  }
  return s
}

export default function CourseStandardsCoveragePage() {
  const { courseCode: rawCode } = useParams()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''

  const [course, setCourse] = useState<CoursePublic | null>(null)
  const [rows, setRows] = useState<StandardCoverageItem[]>([])
  const [framework, setFramework] = useState('ccss-math')
  const [grade, setGrade] = useState('6')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [sortKey, setSortKey] = useState<SortKey>('code')
  const [sortDir, setSortDir] = useState<'ascending' | 'descending'>('ascending')

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    setError(null)
    try {
      const c = await fetchCourse(courseCode)
      setCourse(c)
      if (c.standardsAlignmentEnabled !== true) {
        setRows([])
        setError('Turn on Standards alignment in Course settings → Course tools to use this report.')
        return
      }
      const data = await fetchCourseStandardsCoverage(courseCode, {
        framework: framework.trim(),
        grade: grade.trim() || undefined,
      })
      setRows(data.standards)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load coverage.')
      setRows([])
    } finally {
      setLoading(false)
    }
  }, [courseCode, framework, grade])

  useEffect(() => {
    void load()
  }, [load])

  const sorted = useMemo(() => {
    const copy = [...rows]
    const dir = sortDir === 'ascending' ? 1 : -1
    copy.sort((a, b) => {
      let cmp = 0
      if (sortKey === 'code') {
        cmp = a.code.localeCompare(b.code)
      } else if (sortKey === 'questionCount') {
        cmp = a.questionCount - b.questionCount
      } else if (sortKey === 'coverageStatus') {
        cmp = a.coverageStatus.localeCompare(b.coverageStatus)
      } else {
        const av = a.averageMastery ?? -1
        const bv = b.averageMastery ?? -1
        cmp = av - bv
      }
      return cmp * dir
    })
    return copy
  }, [rows, sortDir, sortKey])

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === 'ascending' ? 'descending' : 'ascending'))
    } else {
      setSortKey(key)
      setSortDir('ascending')
    }
  }

  const exportCsv = () => {
    const header = [
      'code',
      'shortCode',
      'description',
      'gradeBand',
      'questionCount',
      'averageMastery',
      'coverageStatus',
      'superseded',
    ]
    const lines = [header.join(',')]
    for (const r of sorted) {
      lines.push(
        [
          csvEscape(r.code),
          csvEscape(r.shortCode ?? ''),
          csvEscape(r.description),
          csvEscape(r.gradeBand ?? ''),
          String(r.questionCount),
          r.averageMastery == null ? '' : String(r.averageMastery),
          csvEscape(r.coverageStatus),
          r.superseded ? 'true' : 'false',
        ].join(','),
      )
    }
    const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `standards-coverage-${courseCode}-${framework}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const sortAria = (key: SortKey): 'ascending' | 'descending' | 'none' =>
    sortKey === key ? sortDir : 'none'

  return (
    <div data-focus-anchor="standards.coverage.grid" className="mx-auto max-w-6xl px-4 py-6">
      <header className="mb-6">
        <h1 className="text-xl font-semibold text-fg-default">
          Standards coverage
        </h1>
        {course && (
          <p className="mt-1 text-sm text-fg-muted">{course.title}</p>
        )}
        <p className="mt-2 max-w-3xl text-sm text-fg-muted">
          Per-standard question counts and class-average mastery (from tagged concepts). Enable the
          feature under Settings if this page is unavailable.
        </p>
      </header>

      <div className="mb-4 flex flex-wrap items-end gap-3">
        <label className="flex flex-col text-xs font-medium text-fg-muted">
          Framework code
          <input
            className="mt-1 rounded-md border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-raised"
            value={framework}
            onChange={(e) => setFramework(e.target.value)}
            aria-label="Framework code"
          />
        </label>
        <label className="flex flex-col text-xs font-medium text-fg-muted">
          Grade band
          <input
            className="mt-1 rounded-md border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-raised"
            value={grade}
            onChange={(e) => setGrade(e.target.value)}
            aria-label="Grade band filter"
          />
        </label>
        <button
          type="button"
          className="rounded-md bg-accent-solid px-3 py-2 text-sm font-medium text-white hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
          onClick={() => void load()}
        >
          Refresh
        </button>
        <button
          type="button"
          className="rounded-md border border-border-strong px-3 py-2 text-sm font-medium text-fg-default hover:bg-surface-base focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:border-border-default dark:text-fg-default dark:hover:bg-surface-raised"
          onClick={exportCsv}
          disabled={sorted.length === 0}
        >
          Export CSV
        </button>
      </div>

      {loading && <p className="text-sm text-fg-muted">Loading…</p>}
      {error && (
        <p className="text-sm text-rose-700 dark:text-rose-400" role="alert">
          {error}
        </p>
      )}

      {!loading && !error && sorted.length === 0 && (
        <p className="text-sm text-fg-muted">
          No standards aligned yet. Tag your concepts to standards to enable coverage reporting.
        </p>
      )}

      {!loading && sorted.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-border-default dark:border-border-subtle">
          <table className="min-w-full divide-y divide-slate-200 text-start text-sm dark:divide-neutral-800">
            <thead className="bg-surface-base">
              <tr>
                <th scope="col" className="px-3 py-2 font-semibold text-fg-default">
                  <button
                    type="button"
                    className="rounded px-1 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                    onClick={() => toggleSort('code')}
                    aria-sort={sortAria('code')}
                  >
                    Code
                  </button>
                </th>
                <th scope="col" className="px-3 py-2 font-semibold text-fg-default">
                  Description
                </th>
                <th scope="col" className="px-3 py-2 font-semibold text-fg-default">
                  <button
                    type="button"
                    className="rounded px-1 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                    onClick={() => toggleSort('questionCount')}
                    aria-sort={sortAria('questionCount')}
                  >
                    Questions
                  </button>
                </th>
                <th scope="col" className="px-3 py-2 font-semibold text-fg-default">
                  <button
                    type="button"
                    className="rounded px-1 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                    onClick={() => toggleSort('averageMastery')}
                    aria-sort={sortAria('averageMastery')}
                  >
                    Avg mastery
                  </button>
                </th>
                <th scope="col" className="px-3 py-2 font-semibold text-fg-default">
                  <button
                    type="button"
                    className="rounded px-1 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                    onClick={() => toggleSort('coverageStatus')}
                    aria-sort={sortAria('coverageStatus')}
                  >
                    Coverage
                  </button>
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
              {sorted.map((r) => (
                <tr key={r.standardCodeId} tabIndex={0} className="focus-within:bg-surface-base dark:focus-within:bg-neutral-900/60">
                  <td className="whitespace-nowrap px-3 py-2 font-mono text-xs text-fg-default">
                    <span aria-label={`Standard code ${r.code}`}>{r.shortCode ?? r.code}</span>
                    {r.superseded ? (
                      <span className="ms-2 rounded bg-amber-100 px-1.5 text-xs text-amber-900 dark:bg-amber-900/40 dark:text-amber-100">
                        superseded
                      </span>
                    ) : null}
                  </td>
                  <td className="max-w-md px-3 py-2 text-fg-muted">
                    {r.description}
                  </td>
                  <td className="px-3 py-2 tabular-nums text-fg-default">
                    {r.questionCount}
                  </td>
                  <td className="px-3 py-2 tabular-nums text-fg-default">
                    {r.averageMastery == null ? '—' : `${Math.round(r.averageMastery * 100)}%`}
                  </td>
                  <td className="px-3 py-2 text-fg-default">{r.coverageStatus}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

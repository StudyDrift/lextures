import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolInstanceUsage,
  fetchContentToolsInstances,
  type ContentToolInstance,
} from '../../lib/courses-api'
import { ToolResponsesPanel } from '../../components/content-tools/instructor/tool-responses-panel'
import { SourcesPanel } from '../../components/content-tools/instructor/sources-panel'

type Row = ContentToolInstance & {
  learnersWithState: number
  learnersCompleted: number
}

export default function CourseContentToolsInsights() {
  const { courseCode = '' } = useParams()
  const { t } = useTranslation('contentTools')
  const [rows, setRows] = useState<Row[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [openInstanceId, setOpenInstanceId] = useState<string | null>(null)

  useEffect(() => {
    if (!courseCode) return
    let cancelled = false
    setLoading(true)
    void (async () => {
      try {
        const instances = await fetchContentToolsInstances(courseCode, {})
        const withUsage = await Promise.all(
          instances.map(async (inst) => {
            try {
              const usage = await fetchContentToolInstanceUsage(courseCode, inst.id)
              return {
                ...inst,
                learnersWithState: usage.learnersWithState,
                learnersCompleted: usage.learnersCompleted,
              }
            } catch {
              return { ...inst, learnersWithState: 0, learnersCompleted: 0 }
            }
          }),
        )
        if (!cancelled) setRows(withUsage)
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load tools.')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode])

  return (
    <div className="mx-auto max-w-5xl p-4" data-testid="content-tools-insights">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-slate-900 dark:text-neutral-100">
            {t('contentTools.instructor.insightsTitle')}
          </h1>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
            {t('contentTools.instructor.insightsHelp')}
          </p>
        </div>
        <Link
          to={`/courses/${encodeURIComponent(courseCode)}/whats-working`}
          className="text-xs text-sky-700 underline dark:text-sky-300"
        >
          What&apos;s working
        </Link>
      </div>
      {loading ? <p className="text-sm text-slate-500">Loading…</p> : null}
      {error ? (
        <p className="text-sm text-rose-600" role="alert">
          {error}
        </p>
      ) : null}
      {!loading && rows.length === 0 ? (
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          {t('contentTools.instructor.emptyRoster')}
        </p>
      ) : null}
      {rows.length > 0 ? (
        <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-neutral-700">
          <table className="min-w-full text-sm">
            <thead className="bg-slate-50 text-left dark:bg-neutral-900">
              <tr>
                <th className="px-3 py-2" scope="col">
                  Tool
                </th>
                <th className="px-3 py-2" scope="col">
                  Started
                </th>
                <th className="px-3 py-2" scope="col">
                  Completed
                </th>
                <th className="px-3 py-2" scope="col">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id} className="border-t border-slate-100 dark:border-neutral-800">
                  <td className="px-3 py-2">
                    {row.title || row.toolId}
                    <span className="mt-0.5 block text-xs text-slate-500">{row.toolId}</span>
                  </td>
                  <td className="px-3 py-2">{row.learnersWithState}</td>
                  <td className="px-3 py-2">{row.learnersCompleted}</td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-3">
                      <button
                        type="button"
                        className="text-xs font-medium text-sky-700 underline dark:text-sky-300"
                        onClick={() => setOpenInstanceId(row.id)}
                      >
                        {t('contentTools.instructor.openResponses')}
                      </button>
                      <button
                        type="button"
                        className="text-xs font-medium text-sky-700 underline dark:text-sky-300"
                        data-testid={`ct-open-sources-${row.id}`}
                        onClick={() => setOpenInstanceId(row.id)}
                      >
                        {t('contentTools.context.sourcesTitle')}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      {openInstanceId ? (
        <ToolResponsesPanel
          open
          courseCode={courseCode}
          instanceId={openInstanceId}
          itemId={rows.find((r) => r.id === openInstanceId)?.structureItemId ?? undefined}
          onClose={() => setOpenInstanceId(null)}
        />
      ) : null}
      {openInstanceId && rows.find((r) => r.id === openInstanceId)?.structureItemId ? (
        <SourcesPanel
          courseCode={courseCode}
          itemId={rows.find((r) => r.id === openInstanceId)!.structureItemId!}
          instanceId={openInstanceId}
        />
      ) : null}
    </div>
  )
}

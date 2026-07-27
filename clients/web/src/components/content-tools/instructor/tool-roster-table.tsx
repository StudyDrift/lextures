import { useTranslation } from 'react-i18next'
import type { ContentToolRosterStateRow } from '../../../lib/courses-api'

export type ToolRosterTableProps = {
  rows: ContentToolRosterStateRow[]
  loading?: boolean
  empty?: boolean
  onSelect: (row: ContentToolRosterStateRow) => void
  onResetOne: (row: ContentToolRosterStateRow) => void
  sortKey?: 'displayName' | 'status' | 'interactionCount'
  sortDir?: 'asc' | 'desc'
  onSort?: (key: 'displayName' | 'status' | 'interactionCount') => void
}

function ariaSort(
  active: boolean,
  dir: 'asc' | 'desc',
): 'ascending' | 'descending' | 'none' {
  if (!active) return 'none'
  return dir === 'asc' ? 'ascending' : 'descending'
}

export function ToolRosterTable({
  rows,
  loading = false,
  empty = false,
  onSelect,
  onResetOne,
  sortKey = 'displayName',
  sortDir = 'asc',
  onSort,
}: ToolRosterTableProps) {
  const { t } = useTranslation('contentTools')

  if (loading) {
    return (
      <div
        className="animate-pulse space-y-2 rounded-2xl border border-slate-200 p-4 dark:border-neutral-700"
        data-testid="tool-roster-loading"
      >
        <div className="h-4 w-1/3 rounded bg-slate-200 dark:bg-neutral-700" />
        <div className="h-8 rounded bg-slate-100 dark:bg-neutral-800" />
        <div className="h-8 rounded bg-slate-100 dark:bg-neutral-800" />
      </div>
    )
  }

  if (empty || rows.length === 0) {
    return (
      <p className="text-sm text-slate-600 dark:text-neutral-300" data-testid="tool-roster-empty">
        {t('contentTools.instructor.emptyRoster')}
      </p>
    )
  }

  return (
    <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-neutral-700">
      <table className="min-w-full text-sm" data-testid="tool-roster-table">
        <caption className="sr-only">{t('contentTools.instructor.rosterCaption')}</caption>
        <thead className="bg-slate-50 text-left dark:bg-neutral-900">
          <tr>
            {(
              [
                ['displayName', t('contentTools.instructor.colLearner')],
                ['status', t('contentTools.instructor.colStatus')],
                ['interactionCount', t('contentTools.instructor.colInteractions')],
              ] as const
            ).map(([key, label]) => (
              <th
                key={key}
                scope="col"
                aria-sort={ariaSort(sortKey === key, sortDir)}
                className="px-3 py-2 font-semibold text-slate-700 dark:text-neutral-200"
              >
                {onSort ? (
                  <button
                    type="button"
                    className="hover:underline"
                    onClick={() => onSort(key)}
                  >
                    {label}
                  </button>
                ) : (
                  label
                )}
              </th>
            ))}
            <th scope="col" className="px-3 py-2 font-semibold text-slate-700 dark:text-neutral-200">
              {t('contentTools.instructor.colScore')}
            </th>
            <th scope="col" className="px-3 py-2 font-semibold text-slate-700 dark:text-neutral-200">
              {t('contentTools.instructor.colActions')}
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.enrollmentId}
              className="border-t border-slate-100 dark:border-neutral-800"
              data-enrollment-id={row.enrollmentId}
            >
              <td className="px-3 py-2 text-slate-900 dark:text-neutral-100">{row.displayName}</td>
              <td className="px-3 py-2 text-slate-700 dark:text-neutral-300">
                {row.status.replace(/_/g, ' ')}
              </td>
              <td className="px-3 py-2 text-slate-700 dark:text-neutral-300">
                {row.interactionCount}
              </td>
              <td className="px-3 py-2 text-slate-700 dark:text-neutral-300">
                {row.score ? `${row.score.raw}/${row.score.max}` : '—'}
              </td>
              <td className="px-3 py-2">
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    className="text-xs font-medium text-sky-700 underline dark:text-sky-300"
                    onClick={() => onSelect(row)}
                  >
                    {t('contentTools.instructor.viewDetail')}
                  </button>
                  <button
                    type="button"
                    className="text-xs font-medium text-rose-700 underline dark:text-rose-300"
                    onClick={() => onResetOne(row)}
                    disabled={row.status === 'not_started' && row.interactionCount === 0}
                  >
                    {t('contentTools.reset.actionOne')}
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

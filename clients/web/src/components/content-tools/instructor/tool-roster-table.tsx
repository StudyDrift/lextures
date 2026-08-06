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
        className="animate-pulse space-y-2 rounded-2xl border border-border-default p-4 dark:border-border-default"
        data-testid="tool-roster-loading"
      >
        <div className="h-4 w-1/3 rounded bg-slate-200 dark:bg-neutral-700" />
        <div className="h-8 rounded bg-surface-sunken" />
        <div className="h-8 rounded bg-surface-sunken" />
      </div>
    )
  }

  if (empty || rows.length === 0) {
    return (
      <p className="text-sm text-fg-muted" data-testid="tool-roster-empty">
        {t('contentTools.instructor.emptyRoster')}
      </p>
    )
  }

  return (
    <div className="overflow-x-auto rounded-2xl border border-border-default">
      <table className="min-w-full text-sm" data-testid="tool-roster-table">
        <caption className="sr-only">{t('contentTools.instructor.rosterCaption')}</caption>
        <thead className="bg-surface-base text-left dark:bg-surface-raised">
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
                className="px-3 py-2 font-semibold text-fg-default"
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
            <th scope="col" className="px-3 py-2 font-semibold text-fg-default">
              {t('contentTools.instructor.colScore')}
            </th>
            <th scope="col" className="px-3 py-2 font-semibold text-fg-default">
              {t('contentTools.instructor.colActions')}
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.enrollmentId}
              className="border-t border-border-subtle"
              data-enrollment-id={row.enrollmentId}
            >
              <td className="px-3 py-2 text-fg-default">{row.displayName}</td>
              <td className="px-3 py-2 text-fg-muted">
                {row.status.replace(/_/g, ' ')}
              </td>
              <td className="px-3 py-2 text-fg-muted">
                {row.interactionCount}
              </td>
              <td className="px-3 py-2 text-fg-muted">
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

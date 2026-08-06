import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolStateResets,
  restoreContentToolStateReset,
  type ContentToolStateResetSnapshot,
} from '../../../lib/courses-api'

export type ResetHistoryListProps = {
  courseCode: string
  instanceId?: string
  enrollmentId?: string
  refreshKey?: number
}

export function ResetHistoryList({
  courseCode,
  instanceId,
  enrollmentId,
  refreshKey = 0,
}: ResetHistoryListProps) {
  const { t } = useTranslation('contentTools')
  const [items, setItems] = useState<ContentToolStateResetSnapshot[]>([])
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    void fetchContentToolStateResets(courseCode, { instanceId, enrollmentId })
      .then((list) => {
        if (!cancelled) setItems(list)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load history.')
      })
    return () => {
      cancelled = true
    }
  }, [courseCode, instanceId, enrollmentId, refreshKey])

  async function onRestore(id: string) {
    setBusyId(id)
    setError(null)
    try {
      await restoreContentToolStateReset(courseCode, id)
      const list = await fetchContentToolStateResets(courseCode, { instanceId, enrollmentId })
      setItems(list)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Restore failed.')
    } finally {
      setBusyId(null)
    }
  }

  if (items.length === 0 && !error) {
    return (
      <p className="text-xs text-fg-muted">
        {t('contentTools.reset.historyEmpty')}
      </p>
    )
  }

  return (
    <div className="space-y-2" data-testid="reset-history-list">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
        {t('contentTools.reset.historyTitle')}
      </h3>
      {error ? (
        <p className="text-xs text-rose-600" role="alert">
          {error}
        </p>
      ) : null}
      <ul className="space-y-2">
        {items.map((item) => (
          <li
            key={item.id}
            className="rounded-md border border-border-default px-3 py-2 text-xs dark:border-border-default"
          >
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="font-medium text-fg-default">
                  {item.scope.replace(/_/g, ' ')} · {new Date(item.resetAt).toLocaleString()}
                </p>
                <p className="text-fg-muted">
                  {item.priorStatus}
                  {item.restoredAt
                    ? ` · ${t('contentTools.reset.restored')}`
                    : ` · ${t('contentTools.reset.restorableUntil', {
                        date: new Date(item.purgeAfter).toLocaleDateString(),
                      })}`}
                </p>
              </div>
              {!item.restoredAt ? (
                <button
                  type="button"
                  disabled={busyId === item.id}
                  className="text-sky-700 underline dark:text-sky-300"
                  onClick={() => void onRestore(item.id)}
                >
                  {t('contentTools.reset.restore')}
                </button>
              ) : null}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

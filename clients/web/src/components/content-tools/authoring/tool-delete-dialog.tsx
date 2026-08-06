import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolInstanceUsage,
  type ToolInstanceUsage,
} from '../../../lib/courses-api'

export type ToolDeleteDialogProps = {
  open: boolean
  courseCode: string
  instanceId: string | null
  onClose: () => void
  onArchive: () => void | Promise<void>
  onDeletePermanently: () => void | Promise<void>
}

export function ToolDeleteDialog({
  open,
  courseCode,
  instanceId,
  onClose,
  onArchive,
  onDeletePermanently,
}: ToolDeleteDialogProps) {
  const { t } = useTranslation('contentTools')
  const [usage, setUsage] = useState<ToolInstanceUsage | null>(null)
  const [loading, setLoading] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open || !instanceId) return
    let cancelled = false
    setLoading(true)
    setError(null)
    setUsage(null)
    void fetchContentToolInstanceUsage(courseCode, instanceId)
      .then((u) => {
        if (!cancelled) setUsage(u)
      })
      .catch(() => {
        if (!cancelled) {
          setUsage({
            instanceId,
            learnersWithState: 0,
            learnersCompleted: 0,
            lastInteractionAt: null,
            referencedInBody: true,
          })
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, courseCode, instanceId])

  if (!open || !instanceId) return null

  const learners = usage?.learnersWithState ?? 0

  return (
    <div
      className="fixed inset-0 z-[80] flex items-center justify-center bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="content-tool-delete-title"
    >
      <div className="w-full max-w-md rounded-lg border border-border-default bg-surface-raised p-4 shadow-xl dark:border-border-default dark:bg-surface-raised">
        <h2
          id="content-tool-delete-title"
          className="text-sm font-semibold text-fg-default"
        >
          {t('contentTools.authoring.deleteTitle')}
        </h2>
        {loading ? (
          <p className="mt-2 text-xs text-fg-muted">
            {t('contentTools.authoring.loadingUsage')}
          </p>
        ) : learners > 0 ? (
          <p className="mt-2 text-sm text-fg-muted">
            {t('contentTools.authoring.deleteWithUsage', { count: learners })}
          </p>
        ) : (
          <p className="mt-2 text-sm text-fg-muted">
            {t('contentTools.authoring.deleteNoUsage')}
          </p>
        )}
        {error ? (
          <p className="mt-2 text-xs text-rose-600 dark:text-rose-400" role="alert">
            {error}
          </p>
        ) : null}
        <div className="mt-4 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={onClose}
            className="rounded-md px-3 py-1.5 text-xs font-medium text-fg-muted hover:bg-surface-sunken dark:text-fg-default dark:hover:bg-surface-overlay"
          >
            {t('contentTools.authoring.cancel')}
          </button>
          <button
            type="button"
            disabled={busy || loading}
            onClick={() => {
              setBusy(true)
              void Promise.resolve(onDeletePermanently())
                .catch((err) => {
                  setError(err instanceof Error ? err.message : String(err))
                })
                .finally(() => setBusy(false))
            }}
            className="rounded-md px-3 py-1.5 text-xs font-medium text-rose-700 hover:bg-rose-50 dark:text-rose-300 dark:hover:bg-rose-950/40"
          >
            {t('contentTools.authoring.deletePermanently')}
          </button>
          <button
            type="button"
            disabled={busy || loading}
            autoFocus
            onClick={() => {
              setBusy(true)
              void Promise.resolve(onArchive())
                .catch((err) => {
                  setError(err instanceof Error ? err.message : String(err))
                })
                .finally(() => setBusy(false))
            }}
            className="rounded-md bg-slate-800 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-700 dark:bg-neutral-200 dark:text-neutral-900"
          >
            {t('contentTools.authoring.archive')}
          </button>
        </div>
      </div>
    </div>
  )
}

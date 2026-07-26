import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolStateDetail,
  type ContentToolStateDetail,
} from '../../../lib/courses-api'

export type ToolStateDetailDrawerProps = {
  open: boolean
  courseCode: string
  instanceId: string
  enrollmentId: string | null
  onClose: () => void
  onReset: (enrollmentId: string) => void
}

export function ToolStateDetailDrawer({
  open,
  courseCode,
  instanceId,
  enrollmentId,
  onClose,
  onReset,
}: ToolStateDetailDrawerProps) {
  const { t } = useTranslation('contentTools')
  const [detail, setDetail] = useState<ContentToolStateDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open || !enrollmentId) return
    let cancelled = false
    setLoading(true)
    setError(null)
    setDetail(null)
    void fetchContentToolStateDetail(courseCode, instanceId, enrollmentId)
      .then((d) => {
        if (!cancelled) setDetail(d)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load detail.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, courseCode, instanceId, enrollmentId])

  if (!open || !enrollmentId) return null

  return (
    <div
      className="fixed inset-0 z-[80] flex justify-end bg-slate-900/40"
      role="dialog"
      aria-modal="true"
      aria-labelledby="tool-state-detail-title"
      data-testid="tool-state-detail-drawer"
    >
      <button type="button" className="flex-1" aria-label={t('contentTools.instructor.close')} onClick={onClose} />
      <aside className="flex h-full w-full max-w-md flex-col border-l border-slate-200 bg-white p-4 shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex items-start justify-between gap-2">
          <h2
            id="tool-state-detail-title"
            className="text-sm font-semibold text-slate-900 dark:text-neutral-100"
          >
            {detail?.displayName ?? t('contentTools.instructor.learnerDetail')}
          </h2>
          <button
            type="button"
            className="text-xs text-slate-500 underline dark:text-neutral-400"
            onClick={onClose}
          >
            {t('contentTools.instructor.close')}
          </button>
        </div>
        {loading ? (
          <p className="mt-3 text-sm text-slate-500">{t('contentTools.instructor.loadingDetail')}</p>
        ) : error ? (
          <p className="mt-3 text-sm text-rose-600" role="alert">
            {error}
          </p>
        ) : detail ? (
          <div className="mt-3 space-y-3 overflow-y-auto text-sm">
            <p className="text-slate-700 dark:text-neutral-300">{detail.summary}</p>
            <pre className="overflow-x-auto rounded-md bg-slate-50 p-2 text-xs dark:bg-neutral-950">
              {JSON.stringify(detail.state.state ?? detail.state.stateJson ?? {}, null, 2)}
            </pre>
            <button
              type="button"
              className="rounded-md bg-rose-600 px-3 py-1.5 text-xs font-medium text-white"
              onClick={() => onReset(enrollmentId)}
            >
              {t('contentTools.reset.actionOne')}
            </button>
          </div>
        ) : null}
      </aside>
    </div>
  )
}

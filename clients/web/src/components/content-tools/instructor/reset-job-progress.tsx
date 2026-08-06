import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolResetJob,
  type ContentToolResetJobStatus,
} from '../../../lib/courses-api'

export type ResetJobProgressProps = {
  courseCode: string
  jobId: string
  onDone?: (job: ContentToolResetJobStatus) => void
}

export function ResetJobProgress({ courseCode, jobId, onDone }: ResetJobProgressProps) {
  const { t } = useTranslation('contentTools')
  const [job, setJob] = useState<ContentToolResetJobStatus | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    let timer: number | undefined
    async function poll() {
      try {
        const next = await fetchContentToolResetJob(courseCode, jobId)
        if (cancelled) return
        setJob(next)
        if (next.status === 'succeeded' || next.status === 'failed' || next.status === 'cancelled') {
          onDone?.(next)
          return
        }
        timer = window.setTimeout(() => {
          void poll()
        }, 1000)
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to poll job.')
      }
    }
    void poll()
    return () => {
      cancelled = true
      if (timer) window.clearTimeout(timer)
    }
  }, [courseCode, jobId, onDone])

  const total = job?.totalRows ?? 0
  const processed = job?.processedRows ?? 0
  const pct = total > 0 ? Math.min(100, Math.round((processed / total) * 100)) : 0

  return (
    <div className="space-y-2" data-testid="reset-job-progress">
      <p className="text-sm text-fg-muted">
        {t('contentTools.reset.jobProgress', {
          processed,
          total,
          status: job?.status ?? 'queued',
        })}
      </p>
      <div
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct}
        aria-label={t('contentTools.reset.jobProgressLabel')}
        className="h-2 overflow-hidden rounded bg-slate-200 dark:bg-neutral-700"
      >
        <div className="h-full bg-sky-600 transition-all" style={{ width: `${pct}%` }} />
      </div>
      {error ? (
        <p className="text-xs text-rose-600" role="alert">
          {error}
        </p>
      ) : null}
      {job?.error ? (
        <p className="text-xs text-rose-600" role="alert">
          {job.error}
        </p>
      ) : null}
    </div>
  )
}

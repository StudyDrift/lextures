import { Fragment, useCallback, useEffect, useId, useRef, useState } from 'react'
import {
  fetchScheduleHistory,
  fetchScheduledJobsOverview,
  setScheduledJobEnabled,
  triggerScheduledJob,
  type ScheduledJob,
  type ScheduleHistoryRow,
} from '../../lib/scheduler-api'
import { ScheduledJobActionsMenu } from './scheduled-job-actions-menu'

function fmt(ts?: string | null): string {
  return ts ? new Date(ts).toLocaleString() : '—'
}

function isInFlightStatus(status?: string | null): boolean {
  if (!status) return false
  const normalized = status.toLowerCase()
  return (
    normalized === 'pending' ||
    normalized === 'running' ||
    normalized === 'queued' ||
    normalized === 'in_progress' ||
    normalized === 'triggered'
  )
}

function statusBadgeClass(status?: string | null): string {
  const base = 'inline-flex rounded-full px-2 py-0.5 text-xs font-medium'
  if (!status) return `${base} bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-300`
  const normalized = status.toLowerCase()
  if (
    normalized === 'completed' ||
    normalized === 'succeeded' ||
    normalized === 'success' ||
    normalized === 'done'
  ) {
    return `${base} bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-200`
  }
  if (
    normalized === 'failed' ||
    normalized === 'dead_letter' ||
    normalized === 'error' ||
    normalized === 'cancelled'
  ) {
    return `${base} bg-rose-100 text-rose-800 dark:bg-rose-950 dark:text-rose-200`
  }
  if (isInFlightStatus(status)) {
    return `${base} bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-200`
  }
  return `${base} bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-300`
}

export default function ScheduledJobs() {
  const titleId = useId()
  const [jobs, setJobs] = useState<ScheduledJob[]>([])
  const [backgroundJobsEnabled, setBackgroundJobsEnabled] = useState(true)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<string | null>(null)
  const [history, setHistory] = useState<ScheduleHistoryRow[]>([])
  const [watchingJob, setWatchingJob] = useState<string | null>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const stopPolling = useCallback(() => {
    if (pollRef.current) {
      clearInterval(pollRef.current)
      pollRef.current = null
    }
    setWatchingJob(null)
  }, [])

  const refreshOverview = useCallback(async () => {
    const overview = await fetchScheduledJobsOverview()
    setJobs(overview.jobs)
    setBackgroundJobsEnabled(overview.backgroundJobsEnabled)
    return overview
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      await refreshOverview()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load scheduled jobs.')
    } finally {
      setLoading(false)
    }
  }, [refreshOverview])

  useEffect(() => {
    void load()
    return () => stopPolling()
  }, [load, stopPolling])

  const refreshHistory = useCallback(async (jobName: string) => {
    setHistory(await fetchScheduleHistory(jobName))
  }, [])

  const startPolling = useCallback(
    (jobName: string) => {
      stopPolling()
      setWatchingJob(jobName)
      let attempts = 0
      pollRef.current = setInterval(() => {
        attempts += 1
        void (async () => {
          try {
            const overview = await refreshOverview()
            const job = overview.jobs.find((row) => row.name === jobName)
            if (expanded === jobName) {
              await refreshHistory(jobName)
            }
            if (!job || !isInFlightStatus(job.lastStatus) || attempts >= 15) {
              stopPolling()
            }
          } catch {
            if (attempts >= 15) stopPolling()
          }
        })()
      }, 2000)
    },
    [expanded, refreshHistory, refreshOverview, stopPolling],
  )

  async function toggle(job: ScheduledJob) {
    setBusy(job.name)
    setError(null)
    try {
      await setScheduledJobEnabled(job.name, !job.enabled)
      await refreshOverview()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update job.')
    } finally {
      setBusy(null)
    }
  }

  async function trigger(job: ScheduledJob) {
    setBusy(job.name)
    setError(null)
    try {
      await triggerScheduledJob(job.name)
      await refreshOverview()
      if (expanded === job.name) {
        await refreshHistory(job.name)
      }
      startPolling(job.name)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to trigger job.')
    } finally {
      setBusy(null)
    }
  }

  async function showHistory(job: ScheduledJob) {
    if (expanded === job.name) {
      setExpanded(null)
      setHistory([])
      return
    }
    setExpanded(job.name)
    setHistory([])
    try {
      await refreshHistory(job.name)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load history.')
    }
  }

  return (
    <div className="mx-auto max-w-6xl p-6">
      <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        Scheduled jobs
      </h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
        Cron-scheduled background jobs. Each trigger enqueues a job onto the durable queue and is
        recorded in run history.
      </p>

      {!loading && !backgroundJobsEnabled ? (
        <div
          role="status"
          className="mt-4 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100"
        >
          The background job worker is not running, so triggered jobs stay pending. Set{' '}
          <code className="rounded bg-white/80 px-1 py-0.5 font-mono text-xs dark:bg-neutral-900">
            BACKGROUND_JOBS_ENABLED=1
          </code>{' '}
          in <code className="font-mono text-xs">server/.env</code> and restart the API (enabled by
          default when <code className="font-mono text-xs">APP_ENV=local</code>).
        </div>
      ) : null}

      {watchingJob ? (
        <p role="status" className="mt-4 text-sm text-slate-600 dark:text-slate-400">
          Waiting for <span className="font-mono text-xs">{watchingJob}</span> to finish…
        </p>
      ) : null}

      {error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      {loading ? (
        <p className="mt-6 text-sm text-slate-500">Loading…</p>
      ) : jobs.length === 0 ? (
        <p className="mt-6 text-sm text-slate-500">No scheduled jobs configured.</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
          <table className="min-w-full text-left text-sm" aria-label="Scheduled jobs">
            <caption className="sr-only">Scheduled background jobs</caption>
            <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-slate-400">
              <tr>
                <th scope="col" className="px-4 py-2 font-medium">
                  Name
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Schedule
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Last run
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Last status
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Next run
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Enabled
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {jobs.map((job) => (
                <Fragment key={job.name}>
                  <tr className="border-t border-slate-100 dark:border-neutral-800">
                    <td className="px-4 py-3">
                      <div className="font-medium text-slate-900 dark:text-slate-100">{job.name}</div>
                      {job.description ? (
                        <div className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                          {job.description}
                        </div>
                      ) : null}
                    </td>
                    <td className="px-4 py-3">
                      <code className="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs text-slate-800 dark:bg-neutral-800 dark:text-neutral-200">
                        {job.spec}
                      </code>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">{fmt(job.lastRun)}</td>
                    <td className="px-4 py-3">
                      {job.lastStatus ? (
                        <span className={statusBadgeClass(job.lastStatus)}>{job.lastStatus}</span>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      {job.enabled ? fmt(job.nextRun) : (
                        <span className="text-slate-500 dark:text-neutral-400">disabled</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={
                          job.enabled
                            ? 'inline-flex rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:bg-emerald-950 dark:text-emerald-200'
                            : 'inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-neutral-800 dark:text-neutral-300'
                        }
                      >
                        {job.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <ScheduledJobActionsMenu
                        disabled={busy === job.name}
                        enabled={job.enabled}
                        historyOpen={expanded === job.name}
                        onToggleEnabled={() => void toggle(job)}
                        onTrigger={() => void trigger(job)}
                        onToggleHistory={() => void showHistory(job)}
                      />
                    </td>
                  </tr>
                  {expanded === job.name ? (
                    <tr className="border-t border-slate-100 bg-slate-50/80 dark:border-neutral-800 dark:bg-neutral-950/50">
                      <td colSpan={7} className="px-4 py-4">
                        {history.length === 0 ? (
                          <p className="text-sm text-slate-500">No run history.</p>
                        ) : (
                          <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white dark:border-neutral-800 dark:bg-neutral-900">
                            <table
                              className="min-w-full text-left text-sm"
                              aria-label={`${job.name} history`}
                            >
                              <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-slate-400">
                                <tr>
                                  <th scope="col" className="px-4 py-2 font-medium">
                                    Triggered at
                                  </th>
                                  <th scope="col" className="px-4 py-2 font-medium">
                                    Status
                                  </th>
                                  <th scope="col" className="px-4 py-2 font-medium">
                                    Error
                                  </th>
                                </tr>
                              </thead>
                              <tbody>
                                {history.map((h) => (
                                  <tr
                                    key={h.id}
                                    className="border-t border-slate-100 dark:border-neutral-800"
                                  >
                                    <td className="whitespace-nowrap px-4 py-2">{fmt(h.triggeredAt)}</td>
                                    <td className="px-4 py-2">
                                      <span className={statusBadgeClass(h.status)}>{h.status}</span>
                                    </td>
                                    <td className="max-w-md px-4 py-2 font-mono text-xs text-slate-600 dark:text-neutral-400">
                                      {h.errorLog ?? '—'}
                                    </td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          </div>
                        )}
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

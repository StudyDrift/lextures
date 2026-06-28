import { Fragment, useEffect, useId, useState } from 'react'
import {
  fetchScheduledJobs,
  fetchScheduleHistory,
  setScheduledJobEnabled,
  triggerScheduledJob,
  type ScheduledJob,
  type ScheduleHistoryRow,
} from '../../lib/scheduler-api'

function fmt(ts?: string | null): string {
  return ts ? new Date(ts).toLocaleString() : '—'
}

export default function ScheduledJobs() {
  const titleId = useId()
  const [jobs, setJobs] = useState<ScheduledJob[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<string | null>(null)
  const [history, setHistory] = useState<ScheduleHistoryRow[]>([])

  async function load() {
    setLoading(true)
    setError(null)
    try {
      setJobs(await fetchScheduledJobs())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load scheduled jobs.')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  async function toggle(job: ScheduledJob) {
    setBusy(job.name)
    setError(null)
    try {
      await setScheduledJobEnabled(job.name, !job.enabled)
      await load()
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
      await load()
      if (expanded === job.name) {
        setHistory(await fetchScheduleHistory(job.name))
      }
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
      setHistory(await fetchScheduleHistory(job.name))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load history.')
    }
  }

  return (
    <main aria-labelledby={titleId}>
      <h1 id={titleId}>Scheduled Jobs</h1>
      <p>
        Cron-scheduled background jobs. Each trigger enqueues a job onto the durable queue and is
        recorded in run history.
      </p>

      {error && (
        <div role="alert" style={{ color: 'red' }}>
          {error}
        </div>
      )}

      {loading && <p>Loading…</p>}

      {!loading && jobs.length === 0 && <p>No scheduled jobs configured.</p>}

      {!loading && jobs.length > 0 && (
        <table aria-label="Scheduled jobs">
          <thead>
            <tr>
              <th scope="col">Name</th>
              <th scope="col">Schedule</th>
              <th scope="col">Last run</th>
              <th scope="col">Last status</th>
              <th scope="col">Next run</th>
              <th scope="col">Enabled</th>
              <th scope="col">Actions</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((job) => (
              <Fragment key={job.name}>
                <tr>
                  <td>
                    <strong>{job.name}</strong>
                    <div>{job.description}</div>
                  </td>
                  <td>
                    <code>{job.spec}</code>
                  </td>
                  <td>{fmt(job.lastRun)}</td>
                  <td>{job.lastStatus ?? '—'}</td>
                  <td>{job.enabled ? fmt(job.nextRun) : 'disabled'}</td>
                  <td>
                    <button
                      type="button"
                      disabled={busy === job.name}
                      aria-pressed={job.enabled}
                      onClick={() => void toggle(job)}
                    >
                      {job.enabled ? 'Disable' : 'Enable'}
                    </button>
                  </td>
                  <td>
                    <button
                      type="button"
                      disabled={busy === job.name}
                      onClick={() => void trigger(job)}
                    >
                      Trigger now
                    </button>{' '}
                    <button
                      type="button"
                      aria-expanded={expanded === job.name}
                      onClick={() => void showHistory(job)}
                    >
                      History
                    </button>
                  </td>
                </tr>
                {expanded === job.name && (
                  <tr>
                    <td colSpan={7}>
                      {history.length === 0 ? (
                        <p>No run history.</p>
                      ) : (
                        <table aria-label={`${job.name} history`}>
                          <thead>
                            <tr>
                              <th scope="col">Triggered at</th>
                              <th scope="col">Status</th>
                              <th scope="col">Error</th>
                            </tr>
                          </thead>
                          <tbody>
                            {history.map((h) => (
                              <tr key={h.id}>
                                <td>{fmt(h.triggeredAt)}</td>
                                <td>{h.status}</td>
                                <td>{h.errorLog ?? '—'}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
      )}
    </main>
  )
}

import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

async function apiJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await authorizedFetch(path, init)
  if (!res.ok) {
    throw new Error(await readApiErrorMessage(res))
  }
  if (res.status === 204) {
    return undefined as T
  }
  return res.json() as Promise<T>
}

export interface ScheduledJob {
  name: string
  spec: string
  jobType: string
  description: string
  enabled: boolean
  lastRun?: string | null
  lastStatus?: string | null
  nextRun?: string | null
}

export interface ScheduleHistoryRow {
  id: number
  jobName: string
  triggeredAt: string
  jobId?: string | null
  status: string
  errorLog?: string | null
  notes?: string | null
}

export async function fetchScheduledJobs(): Promise<ScheduledJob[]> {
  const res = await apiJson<{ jobs: ScheduledJob[] }>('/api/v1/admin/scheduler')
  return res.jobs ?? []
}

export async function fetchScheduleHistory(name: string): Promise<ScheduleHistoryRow[]> {
  const res = await apiJson<{ history: ScheduleHistoryRow[] }>(
    `/api/v1/admin/scheduler/${encodeURIComponent(name)}/history`,
  )
  return res.history ?? []
}

export async function setScheduledJobEnabled(name: string, enabled: boolean): Promise<void> {
  await apiJson(
    `/api/v1/admin/scheduler/${encodeURIComponent(name)}/${enabled ? 'enable' : 'disable'}`,
    { method: 'POST' },
  )
}

export async function triggerScheduledJob(name: string): Promise<string> {
  const res = await apiJson<{ jobId: string }>(
    `/api/v1/admin/scheduler/${encodeURIComponent(name)}/trigger`,
    { method: 'POST' },
  )
  return res.jobId
}

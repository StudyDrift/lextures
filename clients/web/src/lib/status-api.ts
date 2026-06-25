import { apiUrl } from './api'

export type StatusIncident = {
  id: string
  name: string
  status: string
  impact: string
}

export type StatusSummary = {
  pageUrl: string
  status: string
  incidents: StatusIncident[]
  configured: boolean
}

const POLL_INTERVAL_MS = 5 * 60 * 1000

export const STATUS_POLL_INTERVAL_MS = POLL_INTERVAL_MS

export async function fetchStatusSummary(): Promise<StatusSummary> {
  const res = await fetch(apiUrl('/api/v1/status-summary'))
  if (!res.ok) {
    throw new Error(`Failed to load status summary (${res.status})`)
  }
  return res.json() as Promise<StatusSummary>
}
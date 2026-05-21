export type IncidentSeverity = 'low' | 'medium' | 'high' | 'critical'
export type IncidentStatus = 'resolved' | 'monitoring'

export type Incident = {
  date: string
  severity: IncidentSeverity
  summary: string
  impact: string
  resolvedDate: string | null
  status: IncidentStatus
}

/** Incidents are ordered most-recent first. */
export const INCIDENTS: Incident[] = [
  {
    date: '2026-01-15',
    severity: 'low',
    summary: 'Planned maintenance overran scheduled window',
    impact: 'Read access degraded for approximately 500 users for 45 minutes.',
    resolvedDate: '2026-01-15',
    status: 'resolved',
  },
]

import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type ResearchParticipation = 'opt_in' | 'opt_out'
export type ResearchParticipationSetting = { orgId: string; participation: ResearchParticipation | null; resolved: boolean; updatedAt?: string }

async function decode(res: Response): Promise<ResearchParticipationSetting> {
  const body: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(body))
  return body as ResearchParticipationSetting
}

export async function getResearchParticipation(orgId: string) {
  return decode(await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/settings/research-participation`))
}

export async function setResearchParticipation(orgId: string, participation: ResearchParticipation) {
  return decode(await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/settings/research-participation`, {
    method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ participation }),
  }))
}


import { useEffect, useState } from 'react'
import { getAccessToken } from '../../lib/auth'
import { decodeJwtPayload } from '../../lib/jwt-payload'
import { authorizedFetch } from '../../lib/api'
import { getResearchParticipation, setResearchParticipation, type ResearchParticipation } from '../../lib/research-participation-api'
import { Button, Card, Field, InlineAlert, Select } from '../ui'

export function ResearchParticipationPanel() {
  const jwtOrgId = decodeJwtPayload(getAccessToken())?.org_id ?? ''
  const [orgId, setOrgId] = useState(jwtOrgId)
  const [orgs, setOrgs] = useState<{ id: string; name: string }[]>([])
  const [value, setValue] = useState<ResearchParticipation | ''>('')
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  useEffect(() => {
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/admin/orgs?limit=200')
        if (!res.ok) return
        const body = await res.json() as { organizations?: { id: string; name: string }[] }
        setOrgs(body.organizations ?? [])
        if (!orgId && body.organizations?.[0]) setOrgId(body.organizations[0].id)
      } catch { /* tenant admins can still use their token organization */ }
    })()
  }, [orgId])

  useEffect(() => {
    if (!orgId) return
    setLoading(true); setMessage(null)
    void getResearchParticipation(orgId).then(s => setValue(s.participation ?? '')).catch(e => setMessage(e instanceof Error ? e.message : 'Could not load the setting.')).finally(() => setLoading(false))
  }, [orgId])

  async function save() {
    if (!orgId || !value) return
    setLoading(true); setMessage(null)
    try { await setResearchParticipation(orgId, value); setMessage('Research participation preference saved and audit logged.') }
    catch (e) { setMessage(e instanceof Error ? e.message : 'Could not save the setting.') }
    finally { setLoading(false) }
  }

  return <Card className="mt-8 p-6">
    <h2 className="text-lg font-semibold text-fg-default">Aggregate research participation</h2>
    <p className="mt-2 text-sm text-fg-muted">Choose whether de-identified records from this organization may be included in future aggregate research. An unresolved organization is excluded. Opting out applies to every future extract.</p>
    <p className="mt-2 text-sm"><a className="text-accent-fg underline" href="https://lextures.com/resources/research/methodology">Read the public privacy and methodology standard</a>.</p>
    <div className="mt-5 grid gap-4 md:grid-cols-2">
      {orgs.length > 1 && <Field label="Organization"><Select value={orgId} onChange={e => setOrgId(e.target.value)}>{orgs.map(org => <option key={org.id} value={org.id}>{org.name}</option>)}</Select></Field>}
      <Field label="Participation decision" description={!value ? 'No decision recorded; records are excluded.' : undefined}>
        <Select value={value} onChange={e => setValue(e.target.value as ResearchParticipation | '')} disabled={loading}>
          <option value="">Not resolved — exclude</option><option value="opt_in">Participate</option><option value="opt_out">Opt out</option>
        </Select>
      </Field>
    </div>
    {message && <InlineAlert className="mt-4" tone={message.startsWith('Research') ? 'success' : 'danger'}>{message}</InlineAlert>}
    <Button className="mt-5" onClick={() => void save()} disabled={loading || !orgId || !value}>{loading ? 'Saving…' : 'Save decision'}</Button>
  </Card>
}

import { useCallback, useEffect, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { useSearchParams } from 'react-router-dom'
import {
  activateConsortiumAgreement,
  createConsortiumAgreement,
  listConsortiumAgreements,
  type ConsortiumAgreement,
} from '../../lib/consortium-api'

function statusLabel(status: ConsortiumAgreement['status']): string {
  switch (status) {
    case 'pending':
      return 'Pending'
    case 'active':
      return 'Active'
    case 'terminated':
      return 'Terminated'
    default: {
      const _exhaustive: never = status
      return _exhaustive
    }
  }
}

export default function ConsortiumAgreementsPage() {
  const { ffConsortiumSharing, loading: featuresLoading } = usePlatformFeatures()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId') ?? ''
  const [agreements, setAgreements] = useState<ConsortiumAgreement[]>([])
  const [guestOrgId, setGuestOrgId] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      setAgreements(await listConsortiumAgreements(orgId))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load agreements.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    if (featuresLoading || !ffConsortiumSharing || !orgId) return
    void load()
  }, [ffConsortiumSharing, featuresLoading, load, orgId])

  if (!ffConsortiumSharing && !featuresLoading) {
    return (
      <div className="p-8">
        <h1 className="text-xl font-semibold">Consortium sharing</h1>
        <p className="mt-2 text-sm text-slate-600">Enable consortium sharing in Settings → Global platform.</p>
      </div>
    )
  }

  if (!orgId) {
    return (
      <div className="p-8">
        <h1 className="text-xl font-semibold">Consortium agreements</h1>
        <p className="mt-2 text-sm text-slate-600">
          Add an <code className="text-xs">?orgId=</code> query parameter with your institution UUID.
        </p>
      </div>
    )
  }

  async function onCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!guestOrgId.trim()) return
    try {
      await createConsortiumAgreement({ hostOrgId: orgId, guestOrgId: guestOrgId.trim(), status: 'pending' })
      setGuestOrgId('')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed.')
    }
  }

  async function onActivate(id: string) {
    try {
      await activateConsortiumAgreement(id)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Activation failed.')
    }
  }

  return (
    <div className="mx-auto max-w-3xl p-6 md:p-8">
      <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-50">Consortium agreements</h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Manage cross-institutional course sharing with partner campuses.
      </p>

      {error ? (
        <p className="mt-4 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800">{error}</p>
      ) : null}

      <form onSubmit={(e) => void onCreate(e)} className="mt-6 space-y-3 rounded-xl border border-slate-200 p-4 dark:border-neutral-700">
        <h2 className="text-sm font-semibold">Invite partner institution</h2>
        <label className="block text-sm">
          Guest organization ID
          <input
            value={guestOrgId}
            onChange={(e) => setGuestOrgId(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            placeholder="UUID of partner org"
          />
        </label>
        <button type="submit" className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white">
          Create agreement
        </button>
      </form>

      <section className="mt-8">
        <h2 className="text-sm font-semibold">Agreements</h2>
        {loading ? <p className="mt-2 text-sm text-slate-500">Loading…</p> : null}
        {!loading && agreements.length === 0 ? (
          <p className="mt-2 text-sm text-slate-500">No consortium agreements yet.</p>
        ) : null}
        <ul className="mt-3 space-y-2">
          {agreements.map((a) => (
            <li
              key={a.id}
              className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700"
            >
              <div>
                <span className="font-medium">{a.hostOrgName ?? a.hostOrgId}</span>
                <span className="mx-1 text-slate-400">↔</span>
                <span className="font-medium">{a.guestOrgName ?? a.guestOrgId}</span>
                <span className="ml-2 text-slate-500">({statusLabel(a.status)})</span>
              </div>
              {a.status === 'pending' && a.guestOrgId === orgId ? (
                <button
                  type="button"
                  onClick={() => void onActivate(a.id)}
                  className="rounded bg-emerald-600 px-2 py-1 text-xs font-semibold text-white"
                >
                  Confirm as guest
                </button>
              ) : null}
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}

import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { authorizedFetch } from '../lib/api'

const API = '/api/v1/internal/ops'

interface TierStatus {
  tier: string
  lastSuccessAt?: string
  walLagSeconds?: number
  nextScheduledAt?: string
  lastError?: string
  healthy: boolean
}

interface BackupStatus {
  targets: {
    postgresRpoMinutes: number
    postgresRtoMinutes: number
    objectStorageRpoHours: number
  }
  tiers: TierStatus[]
  alerts: { tier: string; reason: string }[]
  restoreDrills: {
    id: string
    drillDate: string
    pass?: boolean
    rpoAchievedMinutes?: number
    rtoAchievedMinutes?: number
  }[]
}

export default function BackupOpsAdminPage() {
  const [status, setStatus] = useState<BackupStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await authorizedFetch(`${API}/backup-status`)
      if (res.status === 404) {
        setError('Backup module is not enabled on this environment.')
        return
      }
      if (res.status === 403) {
        setError('You need Global Admin / compliance:backup:admin permission.')
        return
      }
      if (!res.ok) {
        setError('Failed to load backup status.')
        return
      }
      setStatus((await res.json()) as BackupStatus)
    } catch {
      setError('Network error loading backup status.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    document.title = 'Backup & restore — Lextures'
    void load()
  }, [load])

  async function recordDrill(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setMessage(null)
    const form = new FormData(e.currentTarget)
    const now = new Date()
    const body = {
      drillDate: String(form.get('drillDate') || now.toISOString().slice(0, 10)),
      backupTimestamp: String(form.get('backupTimestamp') || now.toISOString()),
      restoreStart: String(form.get('restoreStart') || now.toISOString()),
      restoreEnd: now.toISOString(),
      rpoAchievedMinutes: Number(form.get('rpoMinutes') || 0),
      rtoAchievedMinutes: Number(form.get('rtoMinutes') || 0),
      pass: form.get('pass') === 'on',
      smokeTestOutput: String(form.get('smokeOutput') || ''),
      notes: String(form.get('notes') || ''),
    }
    const res = await authorizedFetch(`${API}/restore-drill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!res.ok) {
      setMessage('Could not record restore drill.')
      return
    }
    setMessage('Restore drill recorded.')
    void load()
  }

  if (loading) {
    return <p className="p-6 text-slate-600 dark:text-neutral-400">Loading backup status…</p>
  }

  if (error) {
    return (
      <div className="p-6 max-w-3xl">
        <p role="alert" className="text-red-700 dark:text-red-300">{error}</p>
        <Link to="/settings/account" className="mt-4 inline-block text-sm underline">Back to settings</Link>
      </div>
    )
  }

  return (
    <div className="p-6 max-w-4xl space-y-8">
      <div>
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-50">Backup &amp; restore</h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          RPO/RTO ops dashboard. Targets: Postgres RPO ≤ {status?.targets.postgresRpoMinutes} min, RTO ≤ {status?.targets.postgresRtoMinutes} min.
        </p>
      </div>

      {status && status.alerts.length > 0 && (
        <div role="alert" className="rounded-lg border border-amber-300 bg-amber-50 dark:bg-amber-950/40 p-4 text-sm">
          <p className="font-medium text-amber-900 dark:text-amber-200">Active alerts</p>
          <ul className="mt-2 list-disc ps-5 space-y-1">
            {status.alerts.map((a) => (
              <li key={`${a.tier}-${a.reason}`}>{a.tier}: {a.reason}</li>
            ))}
          </ul>
        </div>
      )}

      {status && (
        <section className="grid gap-4 sm:grid-cols-2">
          {status.tiers.map((t) => (
            <div key={t.tier} className="rounded-lg border border-slate-200 dark:border-neutral-800 p-4">
              <h2 className="font-medium capitalize">{t.tier.replace('_', ' ')}</h2>
              <dl className="mt-2 text-sm space-y-1">
                <div><dt className="inline text-slate-500">Last success: </dt><dd className="inline">{t.lastSuccessAt ?? '—'}</dd></div>
                {t.walLagSeconds != null && (
                  <div><dt className="inline text-slate-500">WAL lag (s): </dt><dd className="inline">{t.walLagSeconds}</dd></div>
                )}
                <div><dt className="inline text-slate-500">Next scheduled: </dt><dd className="inline">{t.nextScheduledAt ?? '—'}</dd></div>
                <div><dt className="inline text-slate-500">Healthy: </dt><dd className="inline">{t.healthy ? 'yes' : 'no'}</dd></div>
              </dl>
            </div>
          ))}
        </section>
      )}

      {status && status.restoreDrills.length > 0 && (
        <section>
          <h2 className="text-lg font-medium">Restore drill history</h2>
          <ul className="mt-2 text-sm space-y-2">
            {status.restoreDrills.map((d) => (
              <li key={d.id} className="border-b border-slate-100 dark:border-neutral-800 pb-2">
                {d.drillDate} — {d.pass ? 'PASS' : d.pass === false ? 'FAIL' : 'pending'}
                {d.rpoAchievedMinutes != null && ` · RPO ${d.rpoAchievedMinutes}m`}
                {d.rtoAchievedMinutes != null && ` · RTO ${d.rtoAchievedMinutes}m`}
              </li>
            ))}
          </ul>
        </section>
      )}

      <section>
        <h2 className="text-lg font-medium">Record restore drill</h2>
        <form onSubmit={(e) => void recordDrill(e)} className="mt-3 space-y-3 text-sm max-w-md">
          <label className="flex flex-col gap-1">
            Drill date
            <input name="drillDate" type="date" className="border rounded px-2 py-1 dark:bg-neutral-900" />
          </label>
          <label className="flex items-center gap-2">
            <input name="pass" type="checkbox" defaultChecked />
            Drill passed smoke tests
          </label>
          <label className="flex flex-col gap-1">
            RPO achieved (minutes)
            <input name="rpoMinutes" type="number" min={0} defaultValue={45} className="border rounded px-2 py-1 dark:bg-neutral-900" />
          </label>
          <label className="flex flex-col gap-1">
            RTO achieved (minutes)
            <input name="rtoMinutes" type="number" min={0} defaultValue={90} className="border rounded px-2 py-1 dark:bg-neutral-900" />
          </label>
          <label className="flex flex-col gap-1">
            Smoke test output
            <textarea name="smokeOutput" rows={2} className="border rounded px-2 py-1 dark:bg-neutral-900" placeholder="grade reads, quiz attempts, auth: ok" />
          </label>
          <label className="flex flex-col gap-1">
            Notes
            <textarea name="notes" rows={2} className="border rounded px-2 py-1 dark:bg-neutral-900" />
          </label>
          <button type="submit" className="rounded bg-slate-900 text-white px-3 py-1.5 dark:bg-neutral-100 dark:text-neutral-900">
            Save drill
          </button>
        </form>
        {message && <p className="mt-2 text-sm text-emerald-700 dark:text-emerald-300">{message}</p>}
      </section>
    </div>
  )
}

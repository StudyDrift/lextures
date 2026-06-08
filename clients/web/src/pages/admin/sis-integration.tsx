import { useCallback, useEffect, useId, useState, type FormEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createSISConnection,
  fetchSISConnections,
  fetchSISSyncLogs,
  HE_SIS_VENDORS,
  testSISConnection,
  triggerSISSync,
  vendorLabel,
  type SISConnection,
  type SISSyncLog,
  type SISVendor,
} from '../../lib/sis-api'

export default function SisIntegrationPage() {
  const titleId = useId()
  const vendorId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId') ?? ''
  const { ffSisIntegration, loading: featuresLoading } = usePlatformFeatures()
  const [connections, setConnections] = useState<SISConnection[]>([])
  const [logs, setLogs] = useState<SISSyncLog[]>([])
  const [vendor, setVendor] = useState<SISVendor>('banner')
  const [baseUrl, setBaseUrl] = useState('https://banner.university.example.edu')
  const [clientIdRef, setClientIdRef] = useState('secrets/sis-client-id')
  const [clientSecretRef, setClientSecretRef] = useState('secrets/sis-client-secret')
  const [loading, setLoading] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const [conns, syncLogs] = await Promise.all([
        fetchSISConnections(orgId),
        fetchSISSyncLogs(orgId),
      ])
      setConnections(conns.filter((c) => c.market === 'he'))
      setLogs(syncLogs)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load SIS data.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    if (featuresLoading || !ffSisIntegration || !orgId) return
    void load()
  }, [featuresLoading, ffSisIntegration, load, orgId])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    if (!orgId) return
    setBusyId('create')
    setError(null)
    setMessage(null)
    try {
      await createSISConnection(orgId, {
        vendor,
        baseUrl: baseUrl.trim(),
        clientIdRef: clientIdRef.trim(),
        clientSecretRef: clientSecretRef.trim(),
      })
      setMessage('SIS connection created.')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create connection.')
    } finally {
      setBusyId(null)
    }
  }

  async function handleSync(conn: SISConnection) {
    if (!orgId) return
    setBusyId(conn.id)
    setError(null)
    setMessage(null)
    try {
      const result = await triggerSISSync(orgId, conn.id)
      setMessage(`Sync finished with status: ${result.status}.`)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sync failed.')
    } finally {
      setBusyId(null)
    }
  }

  async function handleTest(conn: SISConnection) {
    if (!orgId) return
    setBusyId(`test-${conn.id}`)
    setError(null)
    setMessage(null)
    try {
      const result = await testSISConnection(orgId, conn.id)
      setMessage(result.message)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Connection test failed.')
    } finally {
      setBusyId(null)
    }
  }

  if (featuresLoading) {
    return (
      <main className="mx-auto max-w-4xl p-6">
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">
          Student Information System
        </h1>
        <p className="mt-6 text-sm" role="status">
          Loading…
        </p>
      </main>
    )
  }

  if (!ffSisIntegration) {
    return (
      <main className="mx-auto max-w-4xl p-6">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          SIS integration is not enabled on this platform. Enable{' '}
          <strong>SIS integration</strong> in Settings → Global platform.
        </p>
      </main>
    )
  }

  if (!orgId) {
    return (
      <main className="mx-auto max-w-4xl p-6">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Add an <code className="text-xs">?orgId=</code> query parameter with your institution
          organization id.
        </p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-4xl p-6">
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        Student Information System
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Connect Banner, Workday Student, Colleague, or other higher-ed SIS platforms for nightly
        roster sync and grade passback.
      </p>

      {loading ? (
        <p className="mt-6 text-sm" role="status">
          Loading connections…
        </p>
      ) : null}
      {error ? (
        <p className="mt-4 text-sm text-rose-700 dark:text-rose-200" role="alert">
          {error}
        </p>
      ) : null}
      {message ? (
        <p className="mt-4 text-sm text-emerald-700 dark:text-emerald-200" role="status">
          {message}
        </p>
      ) : null}

      <section className="mt-8 rounded-lg border border-slate-200 p-4 dark:border-neutral-700">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          Add SIS connection
        </h2>
        <form className="mt-4 grid gap-4 sm:grid-cols-2" onSubmit={(e) => void handleCreate(e)}>
          <div className="sm:col-span-2">
            <label htmlFor={vendorId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Adapter type
            </label>
            <select
              id={vendorId}
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
              value={vendor}
              onChange={(e) => setVendor(e.target.value as SISVendor)}
            >
              {HE_SIS_VENDORS.map((v) => (
                <option key={v.value} value={v.value}>
                  {v.label}
                </option>
              ))}
            </select>
          </div>
          <div className="sm:col-span-2">
            <label htmlFor="sis-base-url" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Endpoint URL
            </label>
            <input
              id="sis-base-url"
              type="url"
              required
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
            />
          </div>
          <div>
            <label htmlFor="sis-client-id" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Client ID secret ref
            </label>
            <input
              id="sis-client-id"
              required
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
              value={clientIdRef}
              onChange={(e) => setClientIdRef(e.target.value)}
            />
          </div>
          <div>
            <label htmlFor="sis-client-secret" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Client secret ref
            </label>
            <input
              id="sis-client-secret"
              required
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
              value={clientSecretRef}
              onChange={(e) => setClientSecretRef(e.target.value)}
            />
          </div>
          <div className="sm:col-span-2">
            <button
              type="submit"
              disabled={busyId === 'create'}
              className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
            >
              {busyId === 'create' ? 'Saving…' : 'Add connection'}
            </button>
          </div>
        </form>
      </section>

      <section className="mt-8" aria-labelledby="sis-connections-heading">
        <h2 id="sis-connections-heading" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          Configured connections
        </h2>
        {connections.length === 0 && !loading ? (
          <p className="mt-3 text-sm text-slate-600 dark:text-neutral-400">
            No higher-ed SIS connections yet. Add Banner, Workday, or Colleague above.
          </p>
        ) : (
          <ul className="mt-3 space-y-3">
            {connections.map((conn) => (
              <li
                key={conn.id}
                className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-slate-200 p-4 dark:border-neutral-700"
              >
                <div>
                  <p className="font-medium text-slate-900 dark:text-neutral-100">
                    {vendorLabel(conn.vendor)}
                  </p>
                  <p className="text-xs text-slate-500 dark:text-neutral-400">{conn.baseUrl}</p>
                  <p className="text-xs text-slate-500 dark:text-neutral-400">
                    Schedule: {conn.syncSchedule}
                    {conn.lastSyncAt ? ` · Last sync ${conn.lastSyncAt}` : ''}
                  </p>
                </div>
                <div className="flex gap-2">
                  <button
                    type="button"
                    disabled={busyId != null}
                    onClick={() => void handleTest(conn)}
                    className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
                  >
                    {busyId === `test-${conn.id}` ? 'Testing…' : 'Test'}
                  </button>
                  <button
                    type="button"
                    disabled={busyId != null}
                    onClick={() => void handleSync(conn)}
                    className="rounded-md bg-slate-800 px-3 py-1.5 text-sm text-white dark:bg-neutral-200 dark:text-neutral-900"
                  >
                    {busyId === conn.id ? 'Syncing…' : 'Sync now'}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="mt-8" aria-labelledby="sis-logs-heading">
        <h2 id="sis-logs-heading" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          Sync history
        </h2>
        {logs.length === 0 && !loading ? (
          <p className="mt-3 text-sm text-slate-600 dark:text-neutral-400">No sync runs yet.</p>
        ) : (
          <div className="mt-3 overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-neutral-700">
                  <th className="py-2 pr-4 font-medium">Started</th>
                  <th className="py-2 pr-4 font-medium">Status</th>
                  <th className="py-2 font-medium">Connection</th>
                </tr>
              </thead>
              <tbody>
                {logs.slice(0, 10).map((log) => (
                  <tr key={log.id} className="border-b border-slate-100 dark:border-neutral-800">
                    <td className="py-2 pr-4">{log.startedAt}</td>
                    <td className="py-2 pr-4">{log.status}</td>
                    <td className="py-2 font-mono text-xs">{log.connectionId.slice(0, 8)}…</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </main>
  )
}

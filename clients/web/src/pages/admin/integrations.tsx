import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  disconnectIntegration,
  fetchIntegrations,
  startConnect,
  type IntegrationConnection,
  type IntegrationProvider,
} from '../../lib/integrations-api'

const PROVIDER_BLURB: Record<IntegrationProvider, string> = {
  google_classroom:
    'Import class rosters, co-teachers, and assignments from Google Classroom, with optional recurring roster sync.',
  microsoft_teams: 'Keep your Lextures course roster in sync with a Microsoft Teams Education class.',
  canva: 'Embed Canva for Education designs directly into module items.',
}

function formatTimestamp(iso?: string): string {
  if (!iso) return 'never'
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

export default function IntegrationsAdminPage() {
  const titleId = useId()
  const [searchParams] = useSearchParams()
  const [integrations, setIntegrations] = useState<IntegrationConnection[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setIntegrations(await fetchIntegrations())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load integrations.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  // Surface the result of an OAuth round-trip (?connected= / ?error=).
  useEffect(() => {
    const connected = searchParams.get('connected')
    const err = searchParams.get('error')
    if (connected) setMessage(`Connected ${connected.replace(/_/g, ' ')}.`)
    if (err) setError(`Connection failed: ${err.replace(/_/g, ' ')}.`)
  }, [searchParams])

  async function handleConnect(provider: IntegrationProvider) {
    setBusy(`connect-${provider}`)
    setError(null)
    setMessage(null)
    try {
      const url = await startConnect(provider)
      window.location.assign(url)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start the connection flow.')
      setBusy(null)
    }
  }

  async function handleDisconnect(conn: IntegrationConnection) {
    if (!conn.id) return
    setBusy(`disconnect-${conn.id}`)
    setError(null)
    setMessage(null)
    try {
      await disconnectIntegration(conn.id)
      setMessage(`Disconnected ${conn.displayName}. Imported content is retained.`)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to disconnect integration.')
    } finally {
      setBusy(null)
    }
  }

  return (
    <main className="mx-auto max-w-4xl p-6" aria-labelledby={titleId}>
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        Integrations
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Connect Lextures to the tools you already use. Imports are read-only and OAuth tokens are
        stored encrypted.
      </p>

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

      {loading ? (
        <p className="mt-6 text-sm" role="status">
          Loading integrations…
        </p>
      ) : (
        <ul className="mt-8 grid gap-4 sm:grid-cols-2" data-testid="integration-grid">
          {integrations.map((conn) => (
            <li
              key={conn.id ?? conn.provider}
              data-testid={`integration-card-${conn.provider}`}
              className="rounded-lg border border-slate-200 p-4 dark:border-neutral-700"
            >
              <div className="flex items-center justify-between gap-2">
                <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                  {conn.displayName}
                </h2>
                <span
                  className={
                    conn.connected
                      ? 'rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:bg-emerald-900 dark:text-emerald-100'
                      : 'rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-neutral-800 dark:text-neutral-300'
                  }
                  data-testid={`integration-status-${conn.provider}`}
                >
                  {conn.connected ? 'Connected' : 'Not connected'}
                </span>
              </div>
              <p className="mt-2 text-xs text-slate-600 dark:text-neutral-400">
                {PROVIDER_BLURB[conn.provider] ?? ''}
              </p>
              {conn.connected ? (
                <dl className="mt-3 text-xs text-slate-600 dark:text-neutral-400">
                  <div className="flex justify-between gap-2">
                    <dt>Last synced</dt>
                    <dd>{formatTimestamp(conn.lastSyncedAt)}</dd>
                  </div>
                  {conn.lastSyncError ? (
                    <div className="mt-1 text-rose-700 dark:text-rose-300" role="status">
                      Sync unavailable — {conn.lastSyncError}
                    </div>
                  ) : null}
                </dl>
              ) : null}
              <div className="mt-4">
                {conn.connected ? (
                  <button
                    type="button"
                    onClick={() => void handleDisconnect(conn)}
                    disabled={busy === `disconnect-${conn.id}`}
                    className="rounded border border-rose-300 px-3 py-1.5 text-sm font-medium text-rose-700 hover:bg-rose-50 disabled:opacity-50 dark:border-rose-700 dark:text-rose-200"
                  >
                    {busy === `disconnect-${conn.id}` ? 'Disconnecting…' : 'Disconnect'}
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => void handleConnect(conn.provider)}
                    disabled={busy === `connect-${conn.provider}`}
                    className="rounded bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {busy === `connect-${conn.provider}` ? 'Connecting…' : 'Connect'}
                  </button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  )
}

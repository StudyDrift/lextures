import { useCallback, useEffect, useId, useState } from 'react'
import { fetchInstalledApps, revokeInstalledApp, type InstalledApp } from '../../lib/marketplace-api'

function formatDate(iso: string | null | undefined): string {
  if (!iso) return '—'
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

function InstalledRow({
  app,
  onRevoke,
  busy,
}: {
  app: InstalledApp
  onRevoke: (id: string) => void
  busy: boolean
}) {
  return (
    <tr style={{ borderBottom: '1px solid #f3f4f6' }}>
      <td style={{ padding: '0.75rem 0.5rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          {app.appLogoUrl && (
            <img
              src={app.appLogoUrl}
              alt=""
              width={28}
              height={28}
              style={{ borderRadius: '0.375rem', objectFit: 'cover' }}
            />
          )}
          <div>
            <div style={{ fontWeight: 600, fontSize: '0.875rem' }}>{app.appName}</div>
            <div style={{ fontSize: '0.75rem', color: '#6b7280' }}>{app.appSlug}</div>
          </div>
        </div>
      </td>
      <td style={{ padding: '0.75rem 0.5rem', fontSize: '0.8rem' }}>
        {app.grantedScopes.length > 0 ? (
          <ul style={{ margin: 0, padding: 0, listStyle: 'none', display: 'flex', flexWrap: 'wrap', gap: '0.25rem' }}>
            {app.grantedScopes.map((s) => (
              <li
                key={s}
                style={{
                  background: '#f3f4f6',
                  borderRadius: '0.25rem',
                  padding: '0.1rem 0.35rem',
                  fontFamily: 'monospace',
                  fontSize: '0.75rem',
                }}
              >
                {s}
              </li>
            ))}
          </ul>
        ) : (
          '—'
        )}
      </td>
      <td style={{ padding: '0.75rem 0.5rem', fontSize: '0.8rem', color: '#374151' }}>
        {formatDate(app.installedAt)}
      </td>
      <td style={{ padding: '0.75rem 0.5rem', fontSize: '0.8rem', color: '#374151' }}>
        {formatDate(app.lastUsedAt)}
      </td>
      <td style={{ padding: '0.75rem 0.5rem', textAlign: 'right' }}>
        <button
          type="button"
          disabled={busy}
          onClick={() => onRevoke(app.id)}
          style={{
            padding: '0.375rem 0.75rem',
            fontSize: '0.8rem',
            color: '#dc2626',
            border: '1px solid #dc2626',
            background: '#fff',
            borderRadius: '0.375rem',
            cursor: busy ? 'not-allowed' : 'pointer',
            opacity: busy ? 0.5 : 1,
          }}
          aria-label={`Revoke access for ${app.appName}`}
        >
          Revoke
        </button>
      </td>
    </tr>
  )
}

export default function MarketplaceInstalledPage() {
  const headingId = useId()
  const [apps, setApps] = useState<InstalledApp[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setApps(await fetchInstalledApps())
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load installed apps.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void load() }, [load])

  async function handleRevoke(id: string) {
    if (!window.confirm('Revoke access for this app? Its tokens will be immediately invalidated.')) return
    setBusy(id)
    try {
      await revokeInstalledApp(id)
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to revoke app.')
    } finally {
      setBusy(null)
    }
  }

  return (
    <main
      aria-labelledby={headingId}
      style={{ maxWidth: '72rem', margin: '0 auto', padding: '2rem 1rem' }}
    >
      <header style={{ marginBottom: '1.5rem' }}>
        <h1 id={headingId} style={{ fontSize: '1.5rem', fontWeight: 700 }}>
          Installed Apps
        </h1>
        <p style={{ color: '#6b7280', fontSize: '0.875rem', marginTop: '0.25rem' }}>
          Manage third-party apps that have been granted access to your organization.
        </p>
      </header>

      {error && (
        <div role="alert" style={{ color: '#dc2626', background: '#fef2f2', padding: '0.75rem', borderRadius: '0.375rem', marginBottom: '1rem', fontSize: '0.875rem' }}>
          {error}
        </div>
      )}

      {loading && <p>Loading installed apps&hellip;</p>}

      {!loading && apps.length === 0 && (
        <p style={{ color: '#6b7280' }}>No apps are installed in your organization.</p>
      )}

      {!loading && apps.length > 0 && (
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.875rem' }}>
            <thead>
              <tr style={{ textAlign: 'left', borderBottom: '2px solid #e5e7eb' }}>
                <th style={{ padding: '0.5rem' }}>App</th>
                <th style={{ padding: '0.5rem' }}>Granted Scopes</th>
                <th style={{ padding: '0.5rem' }}>Installed</th>
                <th style={{ padding: '0.5rem' }}>Last Used</th>
                <th style={{ padding: '0.5rem', textAlign: 'right' }}>Action</th>
              </tr>
            </thead>
            <tbody>
              {apps.map((a) => (
                <InstalledRow
                  key={a.id}
                  app={a}
                  onRevoke={(id) => void handleRevoke(id)}
                  busy={busy === a.id}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </main>
  )
}

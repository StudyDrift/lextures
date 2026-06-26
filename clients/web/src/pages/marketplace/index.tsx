import { useEffect, useId, useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchMarketplaceApps, type MarketplaceApp } from '../../lib/marketplace-api'

function AppCard({ app }: { app: MarketplaceApp }) {
  return (
    <article
      style={{
        border: '1px solid #e5e7eb',
        borderRadius: '0.5rem',
        padding: '1.5rem',
        display: 'flex',
        flexDirection: 'column',
        gap: '0.75rem',
        background: '#fff',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
        {app.logoUrl ? (
          <img
            src={app.logoUrl}
            alt={`${app.name} logo`}
            width={40}
            height={40}
            style={{ borderRadius: '0.5rem', objectFit: 'cover' }}
          />
        ) : (
          <div
            aria-hidden="true"
            style={{
              width: 40,
              height: 40,
              borderRadius: '0.5rem',
              background: '#e5e7eb',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '1.25rem',
              color: '#6b7280',
            }}
          >
            {app.name[0]?.toUpperCase()}
          </div>
        )}
        <div>
          <h3 style={{ margin: 0, fontSize: '1rem', fontWeight: 600 }}>{app.name}</h3>
        </div>
      </div>

      <p style={{ margin: 0, fontSize: '0.875rem', color: '#374151', flexGrow: 1 }}>
        {app.description}
      </p>

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.375rem' }}>
        {app.requestedScopes.map((s) => (
          <span
            key={s}
            style={{
              fontSize: '0.75rem',
              background: '#f3f4f6',
              color: '#374151',
              borderRadius: '0.25rem',
              padding: '0.125rem 0.375rem',
            }}
          >
            {s}
          </span>
        ))}
      </div>

      <Link
        to={`/marketplace/${app.slug}`}
        style={{
          display: 'inline-block',
          padding: '0.5rem 1rem',
          background: '#2563eb',
          color: '#fff',
          borderRadius: '0.375rem',
          textDecoration: 'none',
          fontSize: '0.875rem',
          textAlign: 'center',
          fontWeight: 500,
        }}
      >
        View &amp; Install
      </Link>
    </article>
  )
}

export default function MarketplacePage() {
  const headingId = useId()
  const [apps, setApps] = useState<MarketplaceApp[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    fetchMarketplaceApps()
      .then(setApps)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed to load apps.'))
      .finally(() => setLoading(false))
  }, [])

  return (
    <main aria-labelledby={headingId} style={{ maxWidth: '72rem', margin: '0 auto', padding: '2rem 1rem' }}>
      <header style={{ marginBottom: '2rem' }}>
        <h1 id={headingId} style={{ fontSize: '1.875rem', fontWeight: 700 }}>
          App Marketplace
        </h1>
        <p style={{ color: '#6b7280', marginTop: '0.5rem' }}>
          Browse and install third-party integrations for your organization.
        </p>
      </header>

      {loading && <p>Loading apps&hellip;</p>}
      {error && (
        <div role="alert" style={{ color: '#dc2626', padding: '1rem', background: '#fef2f2', borderRadius: '0.375rem' }}>
          {error}
        </div>
      )}

      {!loading && !error && apps.length === 0 && (
        <p style={{ color: '#6b7280' }}>No apps are available in the marketplace yet.</p>
      )}

      {!loading && apps.length > 0 && (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
            gap: '1.5rem',
          }}
        >
          {apps.map((app) => (
            <AppCard key={app.id} app={app} />
          ))}
        </div>
      )}
    </main>
  )
}

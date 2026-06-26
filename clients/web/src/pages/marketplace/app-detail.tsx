import { useEffect, useId, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { fetchMarketplaceApp, type MarketplaceApp } from '../../lib/marketplace-api'

export default function MarketplaceAppDetailPage() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const headingId = useId()
  const [app, setApp] = useState<MarketplaceApp | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!slug) return
    setLoading(true)
    fetchMarketplaceApp(slug)
      .then((a) => {
        if (!a) {
          void navigate('/marketplace', { replace: true })
          return
        }
        setApp(a)
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed to load app.'))
      .finally(() => setLoading(false))
  }, [slug, navigate])

  if (loading) return <main style={{ padding: '2rem' }}>Loading&hellip;</main>

  if (error)
    return (
      <main style={{ padding: '2rem' }}>
        <div role="alert" style={{ color: '#dc2626' }}>
          {error}
        </div>
      </main>
    )

  if (!app) return null

  const installUrl = `/oauth/authorize?client_id=${encodeURIComponent(app.id)}&scope=${encodeURIComponent(app.requestedScopes.join(' '))}`

  return (
    <main
      aria-labelledby={headingId}
      style={{ maxWidth: '48rem', margin: '0 auto', padding: '2rem 1rem' }}
    >
      <nav style={{ marginBottom: '1rem' }}>
        <Link to="/marketplace" style={{ color: '#2563eb', textDecoration: 'none', fontSize: '0.875rem' }}>
          &larr; Back to Marketplace
        </Link>
      </nav>

      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '1.5rem' }}>
        {app.logoUrl ? (
          <img
            src={app.logoUrl}
            alt={`${app.name} logo`}
            width={64}
            height={64}
            style={{ borderRadius: '0.75rem', objectFit: 'cover' }}
          />
        ) : (
          <div
            aria-hidden="true"
            style={{
              width: 64,
              height: 64,
              borderRadius: '0.75rem',
              background: '#e5e7eb',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '2rem',
              color: '#6b7280',
            }}
          >
            {app.name[0]?.toUpperCase()}
          </div>
        )}
        <h1 id={headingId} style={{ margin: 0, fontSize: '1.5rem', fontWeight: 700 }}>
          {app.name}
        </h1>
      </div>

      <p style={{ color: '#374151', marginBottom: '2rem', fontSize: '1rem', lineHeight: '1.6' }}>
        {app.description}
      </p>

      {app.requestedScopes.length > 0 && (
        <section aria-labelledby="perms-heading" style={{ marginBottom: '2rem' }}>
          <h2 id="perms-heading" style={{ fontSize: '1rem', fontWeight: 600, marginBottom: '0.75rem' }}>
            Required Permissions
          </h2>
          <ul style={{ listStyle: 'none', padding: 0, margin: 0, display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
            {app.requestedScopes.map((scope) => (
              <li
                key={scope}
                style={{
                  background: '#f3f4f6',
                  borderRadius: '0.375rem',
                  padding: '0.5rem 0.75rem',
                  fontSize: '0.875rem',
                  fontFamily: 'monospace',
                }}
              >
                {scope}
              </li>
            ))}
          </ul>
        </section>
      )}

      <Link
        to={installUrl}
        style={{
          display: 'inline-block',
          padding: '0.75rem 1.5rem',
          background: '#2563eb',
          color: '#fff',
          borderRadius: '0.375rem',
          textDecoration: 'none',
          fontWeight: 600,
          fontSize: '1rem',
        }}
      >
        Install App
      </Link>
    </main>
  )
}

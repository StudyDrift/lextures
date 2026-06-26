import { useEffect, useId, useState } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { fetchOAuthConsentInfo, type OAuthConsentInfo, type ScopeInfo } from '../../lib/marketplace-api'

function ScopeRow({ scope }: { scope: ScopeInfo }) {
  return (
    <li
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: '0.5rem',
        padding: '0.625rem 0',
        borderBottom: '1px solid #f3f4f6',
      }}
    >
      {scope.isWrite && (
        <span
          aria-label="Write permission warning"
          title="This permission allows the app to modify data"
          style={{ color: '#f59e0b', fontSize: '1rem', flexShrink: 0 }}
        >
          ⚠
        </span>
      )}
      <div>
        <span style={{ fontSize: '0.875rem', color: '#111827', display: 'block' }}>
          {scope.label}
        </span>
        <span style={{ fontSize: '0.75rem', color: '#6b7280', fontFamily: 'monospace' }}>
          {scope.scope}
        </span>
      </div>
    </li>
  )
}

export default function OAuthConsentPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const headingId = useId()

  const clientId = searchParams.get('client_id') ?? ''
  const redirectUri = searchParams.get('redirect_uri') ?? ''
  const scope = searchParams.get('scope') ?? ''
  const codeChallenge = searchParams.get('code_challenge') ?? ''

  const [info, setInfo] = useState<OAuthConsentInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!clientId || !redirectUri || !codeChallenge) {
      setError('Missing required OAuth parameters.')
      setLoading(false)
      return
    }
    fetchOAuthConsentInfo({ clientId, redirectUri, scope, codeChallenge })
      .then(setInfo)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed to load app info.'))
      .finally(() => setLoading(false))
  }, [clientId, redirectUri, scope, codeChallenge])

  function handleAllow() {
    if (!info) return
    // Redirect back with the signed state as the authorization code
    const cbUrl = new URL(redirectUri)
    cbUrl.searchParams.set('code', info.state)
    cbUrl.searchParams.set('state', searchParams.get('state') ?? '')
    window.location.href = cbUrl.toString()
  }

  function handleDeny() {
    const cbUrl = new URL(redirectUri)
    cbUrl.searchParams.set('error', 'access_denied')
    cbUrl.searchParams.set('state', searchParams.get('state') ?? '')
    window.location.href = cbUrl.toString()
  }

  if (loading)
    return (
      <main style={{ padding: '2rem', textAlign: 'center' }}>
        Loading authorization details&hellip;
      </main>
    )

  if (error)
    return (
      <main style={{ maxWidth: '24rem', margin: '4rem auto', padding: '0 1rem' }}>
        <div role="alert" style={{ color: '#dc2626', background: '#fef2f2', padding: '1rem', borderRadius: '0.5rem' }}>
          {error}
        </div>
        <button
          type="button"
          onClick={() => void navigate(-1)}
          style={{ marginTop: '1rem', padding: '0.5rem 1rem', cursor: 'pointer' }}
        >
          Go Back
        </button>
      </main>
    )

  if (!info) return null

  return (
    <main
      aria-labelledby={headingId}
      style={{
        maxWidth: '26rem',
        margin: '4rem auto',
        padding: '0 1rem',
      }}
    >
      <div
        style={{
          border: '1px solid #e5e7eb',
          borderRadius: '0.75rem',
          padding: '2rem',
          background: '#fff',
        }}
      >
        <header style={{ textAlign: 'center', marginBottom: '1.5rem' }}>
          {info.appLogoUrl && (
            <img
              src={info.appLogoUrl}
              alt={`${info.appName} logo`}
              width={56}
              height={56}
              style={{ borderRadius: '0.75rem', objectFit: 'cover', marginBottom: '0.75rem' }}
            />
          )}
          <h1 id={headingId} style={{ fontSize: '1.125rem', fontWeight: 700, margin: 0 }}>
            {info.appName} wants access
          </h1>
          <p style={{ fontSize: '0.875rem', color: '#6b7280', margin: '0.5rem 0 0' }}>
            Review the permissions this app is requesting for your organization.
          </p>
        </header>

        {info.scopes.length > 0 && (
          <section aria-label="Requested permissions">
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {info.scopes.map((s) => (
                <ScopeRow key={s.scope} scope={s} />
              ))}
            </ul>
          </section>
        )}

        {info.scopes.length === 0 && (
          <p style={{ color: '#6b7280', fontSize: '0.875rem', textAlign: 'center' }}>
            This app requests no special permissions.
          </p>
        )}

        <div style={{ display: 'flex', gap: '0.75rem', marginTop: '1.5rem' }}>
          <button
            type="button"
            onClick={handleAllow}
            style={{
              flex: 1,
              padding: '0.625rem',
              background: '#2563eb',
              color: '#fff',
              border: 'none',
              borderRadius: '0.375rem',
              fontWeight: 600,
              fontSize: '0.9375rem',
              cursor: 'pointer',
            }}
          >
            Allow
          </button>
          <button
            type="button"
            onClick={handleDeny}
            style={{
              flex: 1,
              padding: '0.625rem',
              background: '#fff',
              color: '#374151',
              border: '1px solid #d1d5db',
              borderRadius: '0.375rem',
              fontWeight: 500,
              fontSize: '0.9375rem',
              cursor: 'pointer',
            }}
          >
            Deny
          </button>
        </div>

        <p style={{ fontSize: '0.75rem', color: '#9ca3af', marginTop: '1rem', textAlign: 'center' }}>
          You are granting access on behalf of your entire organization.
        </p>
      </div>
    </main>
  )
}

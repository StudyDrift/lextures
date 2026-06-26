import { useCallback, useEffect, useId, useState } from 'react'
import {
  createDeveloperApp,
  fetchDeveloperApps,
  type CreateAppInput,
  type CreateAppResult,
  type DeveloperApp,
} from '../../lib/marketplace-api'

function NewSecretBanner({ app, onDismiss }: { app: CreateAppResult; onDismiss: () => void }) {
  return (
    <div
      role="alert"
      style={{
        background: '#f0fdf4',
        border: '1px solid #86efac',
        borderRadius: '0.5rem',
        padding: '1rem',
        marginBottom: '1.5rem',
      }}
    >
      <strong style={{ display: 'block', marginBottom: '0.5rem' }}>
        App &ldquo;{app.name}&rdquo; created!
      </strong>
      <p style={{ margin: '0 0 0.5rem', fontSize: '0.875rem', color: '#166534' }}>
        Copy your client secret now — it will not be shown again.
      </p>
      <div
        style={{
          background: '#dcfce7',
          padding: '0.5rem 0.75rem',
          borderRadius: '0.375rem',
          fontFamily: 'monospace',
          fontSize: '0.85rem',
          wordBreak: 'break-all',
        }}
      >
        <strong>Client ID:</strong> {app.clientId}
        <br />
        <strong>Client Secret:</strong> {app.clientSecret}
      </div>
      <button
        type="button"
        onClick={onDismiss}
        style={{ marginTop: '0.75rem', fontSize: '0.875rem', cursor: 'pointer' }}
      >
        Dismiss
      </button>
    </div>
  )
}

function AppRow({ app }: { app: DeveloperApp }) {
  return (
    <tr>
      <td style={{ padding: '0.75rem 0.5rem' }}>
        <div style={{ fontWeight: 600 }}>{app.name}</div>
        <div style={{ fontSize: '0.75rem', color: '#6b7280' }}>{app.slug}</div>
      </td>
      <td style={{ padding: '0.75rem 0.5rem', fontFamily: 'monospace', fontSize: '0.8rem' }}>
        {app.clientId}
      </td>
      <td style={{ padding: '0.75rem 0.5rem', fontSize: '0.875rem' }}>
        {app.requestedScopes.join(', ') || '—'}
      </td>
      <td style={{ padding: '0.75rem 0.5rem' }}>
        <span
          style={{
            fontSize: '0.75rem',
            padding: '0.125rem 0.5rem',
            borderRadius: '9999px',
            background: app.published ? '#d1fae5' : '#f3f4f6',
            color: app.published ? '#065f46' : '#374151',
          }}
        >
          {app.published ? 'Published' : 'Draft'}
        </span>
      </td>
    </tr>
  )
}

const DEFAULT_FORM: CreateAppInput = {
  name: '',
  slug: '',
  description: '',
  redirectUris: [''],
  requestedScopes: [],
}

const AVAILABLE_SCOPES = [
  'courses:read',
  'courses:write',
  'enrollments:read',
  'users:read',
  'grades:read',
  'grades:write',
  'assignments:read',
  'pii:read',
]

export default function DeveloperPortalPage() {
  const headingId = useId()
  const formId = useId()
  const [apps, setApps] = useState<DeveloperApp[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<CreateAppInput>(DEFAULT_FORM)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [newApp, setNewApp] = useState<CreateAppResult | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      setApps(await fetchDeveloperApps())
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load apps.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void load() }, [load])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setFormError(null)
    try {
      const result = await createDeveloperApp({
        ...form,
        redirectUris: form.redirectUris.filter((u) => u.trim() !== ''),
      })
      setNewApp(result)
      setShowForm(false)
      setForm(DEFAULT_FORM)
      await load()
    } catch (e: unknown) {
      setFormError(e instanceof Error ? e.message : 'Failed to create app.')
    } finally {
      setSubmitting(false)
    }
  }

  function toggleScope(scope: string) {
    setForm((f) => ({
      ...f,
      requestedScopes: f.requestedScopes.includes(scope)
        ? f.requestedScopes.filter((s) => s !== scope)
        : [...f.requestedScopes, scope],
    }))
  }

  return (
    <main
      aria-labelledby={headingId}
      style={{ maxWidth: '64rem', margin: '0 auto', padding: '2rem 1rem' }}
    >
      <header style={{ marginBottom: '2rem', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '1rem' }}>
        <div>
          <h1 id={headingId} style={{ fontSize: '1.5rem', fontWeight: 700, margin: 0 }}>
            Developer Portal
          </h1>
          <p style={{ color: '#6b7280', marginTop: '0.25rem', fontSize: '0.875rem' }}>
            Register and manage your Lextures marketplace apps.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setShowForm((v) => !v)}
          style={{
            padding: '0.5rem 1rem',
            background: '#2563eb',
            color: '#fff',
            border: 'none',
            borderRadius: '0.375rem',
            fontWeight: 500,
            cursor: 'pointer',
          }}
        >
          {showForm ? 'Cancel' : 'Register New App'}
        </button>
      </header>

      {newApp && (
        <NewSecretBanner app={newApp} onDismiss={() => setNewApp(null)} />
      )}

      {showForm && (
        <section
          aria-label="Register new app"
          style={{
            border: '1px solid #e5e7eb',
            borderRadius: '0.5rem',
            padding: '1.5rem',
            marginBottom: '2rem',
            background: '#fff',
          }}
        >
          <h2 style={{ fontSize: '1rem', fontWeight: 600, marginBottom: '1rem' }}>
            New Application
          </h2>
          {formError && (
            <div role="alert" style={{ color: '#dc2626', marginBottom: '1rem', fontSize: '0.875rem' }}>
              {formError}
            </div>
          )}
          <form id={formId} onSubmit={(e) => void handleSubmit(e)} style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
            <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.875rem' }}>
              App Name *
              <input
                required
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                style={{ padding: '0.5rem', border: '1px solid #d1d5db', borderRadius: '0.375rem' }}
              />
            </label>
            <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.875rem' }}>
              Slug * (used in URL, lowercase letters and hyphens only)
              <input
                required
                pattern="[a-z0-9-]+"
                value={form.slug}
                onChange={(e) => setForm((f) => ({ ...f, slug: e.target.value }))}
                style={{ padding: '0.5rem', border: '1px solid #d1d5db', borderRadius: '0.375rem' }}
              />
            </label>
            <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.875rem' }}>
              Description
              <textarea
                rows={3}
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                style={{ padding: '0.5rem', border: '1px solid #d1d5db', borderRadius: '0.375rem', resize: 'vertical' }}
              />
            </label>
            <fieldset style={{ border: '1px solid #d1d5db', borderRadius: '0.375rem', padding: '0.75rem' }}>
              <legend style={{ fontSize: '0.875rem', fontWeight: 500, padding: '0 0.25rem' }}>
                Redirect URIs *
              </legend>
              {form.redirectUris.map((uri, i) => (
                <div key={i} style={{ display: 'flex', gap: '0.5rem', marginBottom: '0.5rem' }}>
                  <input
                    type="url"
                    value={uri}
                    onChange={(e) => {
                      const next = [...form.redirectUris]
                      next[i] = e.target.value
                      setForm((f) => ({ ...f, redirectUris: next }))
                    }}
                    style={{ flex: 1, padding: '0.375rem 0.5rem', border: '1px solid #d1d5db', borderRadius: '0.375rem', fontSize: '0.875rem' }}
                    placeholder="https://yourapp.com/callback"
                  />
                  {form.redirectUris.length > 1 && (
                    <button
                      type="button"
                      onClick={() => setForm((f) => ({ ...f, redirectUris: f.redirectUris.filter((_, j) => j !== i) }))}
                      style={{ color: '#dc2626', border: 'none', background: 'none', cursor: 'pointer' }}
                      aria-label="Remove redirect URI"
                    >
                      ✕
                    </button>
                  )}
                </div>
              ))}
              <button
                type="button"
                onClick={() => setForm((f) => ({ ...f, redirectUris: [...f.redirectUris, ''] }))}
                style={{ fontSize: '0.8rem', color: '#2563eb', border: 'none', background: 'none', cursor: 'pointer', padding: 0 }}
              >
                + Add another URI
              </button>
            </fieldset>
            <fieldset style={{ border: '1px solid #d1d5db', borderRadius: '0.375rem', padding: '0.75rem' }}>
              <legend style={{ fontSize: '0.875rem', fontWeight: 500, padding: '0 0.25rem' }}>
                Requested Scopes
              </legend>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
                {AVAILABLE_SCOPES.map((scope) => (
                  <label
                    key={scope}
                    style={{ display: 'flex', alignItems: 'center', gap: '0.375rem', fontSize: '0.8rem', cursor: 'pointer' }}
                  >
                    <input
                      type="checkbox"
                      checked={form.requestedScopes.includes(scope)}
                      onChange={() => toggleScope(scope)}
                    />
                    {scope}
                  </label>
                ))}
              </div>
            </fieldset>
            <div style={{ display: 'flex', gap: '0.75rem' }}>
              <button
                type="submit"
                disabled={submitting}
                style={{
                  padding: '0.5rem 1.25rem',
                  background: '#2563eb',
                  color: '#fff',
                  border: 'none',
                  borderRadius: '0.375rem',
                  fontWeight: 500,
                  cursor: submitting ? 'not-allowed' : 'pointer',
                  opacity: submitting ? 0.6 : 1,
                }}
              >
                {submitting ? 'Creating…' : 'Create App'}
              </button>
            </div>
          </form>
        </section>
      )}

      {loading && <p>Loading your apps&hellip;</p>}
      {error && (
        <div role="alert" style={{ color: '#dc2626', fontSize: '0.875rem' }}>
          {error}
        </div>
      )}

      {!loading && !error && apps.length === 0 && !showForm && (
        <p style={{ color: '#6b7280' }}>
          You have no registered apps yet. Click &ldquo;Register New App&rdquo; to get started.
        </p>
      )}

      {apps.length > 0 && (
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.875rem' }}>
            <thead>
              <tr style={{ textAlign: 'left', borderBottom: '2px solid #e5e7eb' }}>
                <th style={{ padding: '0.5rem' }}>App</th>
                <th style={{ padding: '0.5rem' }}>Client ID</th>
                <th style={{ padding: '0.5rem' }}>Scopes</th>
                <th style={{ padding: '0.5rem' }}>Status</th>
              </tr>
            </thead>
            <tbody>
              {apps.map((a) => (
                <AppRow key={a.id} app={a} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </main>
  )
}

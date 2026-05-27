import { useCallback, useEffect, useId, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createAdminLRSEndpoint,
  fetchAdminLRSEndpoints,
  testAdminLRSEndpoint,
  updateAdminLRSEndpoint,
  type LRSEndpoint,
} from '../../lib/lrs-api'

export function LRSSettingsPanel() {
  const { xapiEmissionEnabled, loading: featuresLoading } = usePlatformFeatures()
  const labelId = useId()
  const urlId = useId()
  const userId = useId()
  const passId = useId()
  const [endpoints, setEndpoints] = useState<LRSEndpoint[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [testMsg, setTestMsg] = useState<string | null>(null)
  const [form, setForm] = useState({
    label: '',
    endpointUrl: '',
    authType: 'basic' as 'basic' | 'oauth2',
    username: '',
    password: '',
    enabled: true,
  })

  const load = useCallback(async () => {
    if (!xapiEmissionEnabled) {
      setEndpoints([])
      setLoading(false)
      setError(null)
      return
    }
    setLoading(true)
    setError(null)
    try {
      setEndpoints(await fetchAdminLRSEndpoints())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load LRS settings.')
    } finally {
      setLoading(false)
    }
  }, [xapiEmissionEnabled])

  useEffect(() => {
    if (featuresLoading) return
    void load()
  }, [load, featuresLoading])

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    try {
      await createAdminLRSEndpoint(form)
      setForm({ label: '', endpointUrl: '', authType: 'basic', username: '', password: '', enabled: true })
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save endpoint.')
    } finally {
      setSaving(false)
    }
  }

  if (featuresLoading) {
    return (
      <section>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
          Learning Record Stores
        </h2>
        <p className="mt-4 text-sm text-slate-500">Loading…</p>
      </section>
    )
  }

  if (!xapiEmissionEnabled) {
    return (
      <section>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
          Learning Record Stores
        </h2>
        <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">
          xAPI emission is not enabled for this platform. Turn on{' '}
          <span className="font-medium">xAPI / Caliper emission</span> under Platform settings to
          configure external LRS endpoints.
        </p>
      </section>
    )
  }

  return (
    <section>
      <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
        Learning Record Stores
      </h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
        Forward xAPI statements to external LRS endpoints. Credentials are encrypted at rest and
        never returned in API responses.
      </p>

      {loading && <p className="mt-4 text-sm text-slate-500">Loading…</p>}
      {error && (
        <p className="mt-4 text-sm text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      )}
      {testMsg && (
        <p className="mt-4 text-sm text-emerald-800 dark:text-emerald-200" role="status">
          {testMsg}
        </p>
      )}

      {!loading && endpoints.length > 0 && (
        <ul className="mt-4 divide-y divide-slate-200 rounded-xl border border-slate-200 dark:divide-neutral-700 dark:border-neutral-700">
          {endpoints.map((ep) => (
            <li key={ep.id} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
              <div>
                <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">
                  {ep.label || ep.endpointUrl}
                </p>
                <p className="text-xs text-slate-500 dark:text-neutral-400">{ep.endpointUrl}</p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold dark:border-neutral-600"
                  onClick={async () => {
                    setTestMsg(null)
                    try {
                      const r = await testAdminLRSEndpoint(ep.id)
                      setTestMsg(r.message)
                    } catch (err) {
                      setTestMsg(err instanceof Error ? err.message : 'Test failed.')
                    }
                  }}
                >
                  Test
                </button>
                <label className="inline-flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={ep.enabled}
                    onChange={async (ev) => {
                      await updateAdminLRSEndpoint(ep.id, { enabled: ev.target.checked })
                      await load()
                    }}
                  />
                  Enabled
                </label>
              </div>
            </li>
          ))}
        </ul>
      )}

      <form className="mt-6 space-y-4 rounded-xl border border-slate-200 p-4 dark:border-neutral-700" onSubmit={handleAdd}>
        <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Add LRS endpoint</h3>
        <div>
          <label htmlFor={labelId} className="text-sm font-medium">
            Label
          </label>
          <input
            id={labelId}
            className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
            value={form.label}
            onChange={(e) => setForm((f) => ({ ...f, label: e.target.value }))}
          />
        </div>
        <div>
          <label htmlFor={urlId} className="text-sm font-medium">
            Endpoint URL
          </label>
          <input
            id={urlId}
            required
            type="url"
            className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
            value={form.endpointUrl}
            onChange={(e) => setForm((f) => ({ ...f, endpointUrl: e.target.value }))}
          />
        </div>
        <div>
          <label htmlFor={userId} className="text-sm font-medium">
            Username (Basic auth)
          </label>
          <input
            id={userId}
            autoComplete="username"
            className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
            value={form.username}
            onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
          />
        </div>
        <div>
          <label htmlFor={passId} className="text-sm font-medium">
            Password
          </label>
          <input
            id={passId}
            type="password"
            autoComplete="new-password"
            className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
            value={form.password}
            onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
          />
        </div>
        <button
          type="submit"
          disabled={saving}
          className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Add endpoint'}
        </button>
      </form>
    </section>
  )
}

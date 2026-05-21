import { useCallback, useEffect, useState } from 'react'
import {
  fetchAdminCloudProviders,
  putAdminCloudProvider,
  type CloudProviderSetting,
} from '../../lib/courses-api'

const PROVIDER_META: Record<string, { label: string; description: string }> = {
  google_drive: {
    label: 'Google Drive',
    description: 'Allow instructors and students to link files from Google Drive using the Google Picker API.',
  },
  onedrive: {
    label: 'Microsoft OneDrive',
    description: 'Allow instructors and students to link files from OneDrive and SharePoint.',
  },
  dropbox: {
    label: 'Dropbox',
    description: 'Allow instructors and students to link files from Dropbox using the Dropbox Chooser.',
  },
}

export function CloudProvidersPanel() {
  const [providers, setProviders] = useState<CloudProviderSetting[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const list = await fetchAdminCloudProviders()
      setProviders(list)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load cloud provider settings.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function handleToggle(provider: string, enabled: boolean) {
    setSaveError(null)
    setSaving(provider)
    try {
      await putAdminCloudProvider(provider, enabled)
      setProviders((prev) =>
        prev.map((p) => (p.provider === provider ? { ...p, enabled } : p)),
      )
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Could not update provider setting.')
    } finally {
      setSaving(null)
    }
  }

  return (
    <section>
      <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
        Cloud file picker integrations
      </h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
        Enable cloud storage providers so instructors and students can link files directly from
        their cloud accounts. Users authenticate with their own accounts — no credentials are
        stored on the server.
      </p>

      {loading && (
        <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
      )}
      {error && (
        <p className="mt-4 text-sm text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      )}
      {saveError && (
        <p className="mt-4 text-sm text-rose-700 dark:text-rose-300" role="alert">
          {saveError}
        </p>
      )}

      {!loading && !error && (
        <ul className="mt-4 divide-y divide-slate-200 rounded-xl border border-slate-200 dark:divide-neutral-700 dark:border-neutral-700">
          {providers.map((p) => {
            const meta = PROVIDER_META[p.provider]
            if (!meta) return null
            const isSaving = saving === p.provider
            return (
              <li key={p.provider} className="flex items-start gap-4 px-4 py-4">
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">
                    {meta.label}
                  </p>
                  <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                    {meta.description}
                  </p>
                </div>
                <button
                  type="button"
                  role="switch"
                  aria-checked={p.enabled}
                  aria-label={`${p.enabled ? 'Disable' : 'Enable'} ${meta.label}`}
                  disabled={isSaving}
                  onClick={() => void handleToggle(p.provider, !p.enabled)}
                  className={`relative mt-0.5 inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:opacity-60 ${
                    p.enabled
                      ? 'bg-indigo-600'
                      : 'bg-slate-200 dark:bg-neutral-600'
                  }`}
                >
                  <span
                    className={`pointer-events-none block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform ${
                      p.enabled ? 'translate-x-5' : 'translate-x-0'
                    }`}
                  />
                </button>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}

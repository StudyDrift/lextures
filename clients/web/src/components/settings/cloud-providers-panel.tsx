import { useCallback, useEffect, useState } from 'react'
import {
  fetchAdminCloudProviders,
  putAdminCloudProvider,
  type CloudProviderId,
  type CloudProviderSetting,
} from '../../lib/cloud-providers-api'

const PROVIDER_META: Record<CloudProviderId, { label: string; description: string }> = {
  google_drive: {
    label: 'Google Drive',
    description: 'Allow instructors and students to link or import files from Google Drive using the Google Picker API.',
  },
  onedrive: {
    label: 'Microsoft OneDrive',
    description: 'Allow instructors and students to link or import files from OneDrive and SharePoint.',
  },
  dropbox: {
    label: 'Dropbox',
    description: 'Allow instructors and students to link or import files from Dropbox using the Dropbox Chooser.',
  },
}

type ProviderDraft = {
  enabled: boolean
  clientId: string
  apiKey: string
  appKey: string
}

function draftFromSetting(p: CloudProviderSetting): ProviderDraft {
  return {
    enabled: p.enabled,
    clientId: p.clientId,
    apiKey: p.apiKey,
    appKey: p.appKey,
  }
}

export function CloudProvidersPanel() {
  const [providers, setProviders] = useState<CloudProviderSetting[]>([])
  const [drafts, setDrafts] = useState<Partial<Record<CloudProviderId, ProviderDraft>>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState<CloudProviderId | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const list = await fetchAdminCloudProviders()
      setProviders(list)
      const nextDrafts: Partial<Record<CloudProviderId, ProviderDraft>> = {}
      for (const p of list) {
        nextDrafts[p.provider] = draftFromSetting(p)
      }
      setDrafts(nextDrafts)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load cloud provider settings.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  function updateDraft(provider: CloudProviderId, patch: Partial<ProviderDraft>) {
    setDrafts((prev) => ({
      ...prev,
      [provider]: { ...(prev[provider] ?? { enabled: false, clientId: '', apiKey: '', appKey: '' }), ...patch },
    }))
  }

  async function handleSave(provider: CloudProviderId) {
    const draft = drafts[provider]
    if (!draft) return
    setSaveError(null)
    setSaving(provider)
    try {
      await putAdminCloudProvider(provider, {
        enabled: draft.enabled,
        clientId: draft.clientId,
        apiKey: draft.apiKey,
        appKey: draft.appKey,
      })
      setProviders((prev) =>
        prev.map((p) =>
          p.provider === provider
            ? {
                ...p,
                enabled: draft.enabled,
                clientId: draft.clientId,
                apiKey: draft.apiKey,
                appKey: draft.appKey,
              }
            : p,
        ),
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
        Enable cloud storage providers and register the OAuth or SDK credentials required by each
        picker. Users authenticate with their own accounts — no user tokens are stored on the server.
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
            const draft = drafts[p.provider]
            if (!meta || !draft) return null
            const isSaving = saving === p.provider
            return (
              <li key={p.provider} className="px-4 py-4">
                <div className="flex items-start gap-4">
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
                    aria-checked={draft.enabled}
                    aria-label={`${draft.enabled ? 'Disable' : 'Enable'} ${meta.label}`}
                    disabled={isSaving}
                    onClick={() => updateDraft(p.provider, { enabled: !draft.enabled })}
                    className={`relative mt-0.5 inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:opacity-60 ${
                      draft.enabled
                        ? 'bg-indigo-600'
                        : 'bg-slate-200 dark:bg-neutral-600'
                    }`}
                  >
                    <span
                      className={`pointer-events-none block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform ${
                        draft.enabled ? 'translate-x-5' : 'translate-x-0'
                      }`}
                    />
                  </button>
                </div>

                <div className="mt-4 grid gap-3 sm:grid-cols-2">
                  {(p.provider === 'google_drive' || p.provider === 'onedrive') && (
                    <label className="block text-xs">
                      <span className="font-medium text-slate-600 dark:text-neutral-300">
                        OAuth client ID
                      </span>
                      <input
                        type="text"
                        value={draft.clientId}
                        disabled={isSaving}
                        onChange={(e) => updateDraft(p.provider, { clientId: e.target.value })}
                        placeholder={p.provider === 'google_drive' ? '123456789.apps.googleusercontent.com' : 'Azure application (client) ID'}
                        className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                      />
                    </label>
                  )}
                  {p.provider === 'google_drive' && (
                    <label className="block text-xs">
                      <span className="font-medium text-slate-600 dark:text-neutral-300">
                        API key (Picker developer key)
                      </span>
                      <input
                        type="text"
                        value={draft.apiKey}
                        disabled={isSaving}
                        onChange={(e) => updateDraft(p.provider, { apiKey: e.target.value })}
                        placeholder="AIza…"
                        className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                      />
                    </label>
                  )}
                  {p.provider === 'dropbox' && (
                    <label className="block text-xs sm:col-span-2">
                      <span className="font-medium text-slate-600 dark:text-neutral-300">
                        App key
                      </span>
                      <input
                        type="text"
                        value={draft.appKey}
                        disabled={isSaving}
                        onChange={(e) => updateDraft(p.provider, { appKey: e.target.value })}
                        placeholder="Dropbox Chooser app key"
                        className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                      />
                    </label>
                  )}
                </div>

                <div className="mt-3 flex justify-end">
                  <button
                    type="button"
                    disabled={isSaving}
                    onClick={() => void handleSave(p.provider)}
                    className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {isSaving ? 'Saving…' : 'Save'}
                  </button>
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}

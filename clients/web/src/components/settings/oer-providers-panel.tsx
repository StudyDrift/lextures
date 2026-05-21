import { useCallback, useEffect, useState } from 'react'
import {
  fetchAdminOERProviders,
  putAdminOERProvider,
  type OERProviderId,
  type OERProviderSetting,
} from '../../lib/oer-api'

const PROVIDER_META: Record<OERProviderId, { label: string; description: string }> = {
  oer_commons: {
    label: 'OER Commons',
    description: 'Search open educational resources from OER Commons (K-12 and higher ed).',
  },
  merlot: {
    label: 'MERLOT',
    description: 'Higher-education materials from the MERLOT collection.',
  },
  openstax: {
    label: 'OpenStax',
    description: 'Free OpenStax textbooks and chapter-level links.',
  },
}

export function OERProvidersPanel() {
  const [providers, setProviders] = useState<OERProviderSetting[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setProviders(await fetchAdminOERProviders())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load OER provider settings.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function handleToggle(provider: OERProviderId, enabled: boolean) {
    setSaving(provider)
    try {
      await putAdminOERProvider(provider, enabled)
      setProviders((prev) => prev.map((p) => (p.provider === provider ? { ...p, enabled } : p)))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not update provider.')
    } finally {
      setSaving(null)
    }
  }

  return (
    <section>
      <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">OER library sources</h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
        Enable or disable open educational resource catalogs shown in the course module editor.
      </p>
      {loading && <p className="mt-4 text-sm text-slate-500">Loading…</p>}
      {error && (
        <p className="mt-4 text-sm text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      )}
      {!loading && !error && (
        <ul className="mt-4 divide-y divide-slate-200 rounded-xl border border-slate-200 dark:divide-neutral-700 dark:border-neutral-700">
          {providers.map((p) => {
            const meta = PROVIDER_META[p.provider]
            if (!meta) return null
            return (
              <li key={p.provider} className="flex items-start gap-4 px-4 py-4">
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">{meta.label}</p>
                  <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">{meta.description}</p>
                </div>
                <label className="flex shrink-0 items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={p.enabled}
                    disabled={saving === p.provider}
                    onChange={(e) => void handleToggle(p.provider, e.target.checked)}
                  />
                  <span className="text-slate-600 dark:text-neutral-300">
                    {saving === p.provider ? 'Saving…' : p.enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </label>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}

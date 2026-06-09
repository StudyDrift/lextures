import { useCallback, useEffect, useId, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { authorizedFetch } from '../../lib/api'

const API_KEY_PLACEHOLDER = '••••••••••••'

type LibraryConfig = {
  ezproxyPrefix: string
  domainPatterns: string[]
  almaApiBaseUrl: string
  hasAlmaApiKey: boolean
  updatedAt?: string
}

async function fetchLibraryConfig(): Promise<LibraryConfig> {
  const res = await authorizedFetch('/api/v1/admin/library/config')
  if (!res.ok) throw new Error('Failed to load library config')
  return res.json() as Promise<LibraryConfig>
}

async function saveLibraryConfig(body: {
  ezproxyPrefix: string
  domainPatterns: string[]
  almaApiBaseUrl: string
  almaApiKey: string
}): Promise<LibraryConfig> {
  const res = await authorizedFetch('/api/v1/admin/library/config', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error('Failed to save library config')
  return res.json() as Promise<LibraryConfig>
}

export default function LibraryIntegrationPage() {
  const titleId = useId()
  const { ffLibraryIntegration, loading: featuresLoading } = usePlatformFeatures()

  const [config, setConfig] = useState<LibraryConfig | null>(null)
  const [ezproxyPrefix, setEzproxyPrefix] = useState('')
  const [domainPatternsRaw, setDomainPatternsRaw] = useState('')
  const [almaApiBaseUrl, setAlmaApiBaseUrl] = useState('')
  const [almaApiKey, setAlmaApiKey] = useState('')

  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const cfg = await fetchLibraryConfig()
      setConfig(cfg)
      setEzproxyPrefix(cfg.ezproxyPrefix)
      setDomainPatternsRaw(cfg.domainPatterns.join('\n'))
      setAlmaApiBaseUrl(cfg.almaApiBaseUrl)
      setAlmaApiKey(cfg.hasAlmaApiKey ? API_KEY_PLACEHOLDER : '')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load settings.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffLibraryIntegration) return
    void load()
  }, [featuresLoading, ffLibraryIntegration, load])

  async function handleSave() {
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const patterns = domainPatternsRaw
        .split('\n')
        .map((p) => p.trim())
        .filter(Boolean)
      const apiKey = almaApiKey === API_KEY_PLACEHOLDER ? API_KEY_PLACEHOLDER : almaApiKey
      const cfg = await saveLibraryConfig({
        ezproxyPrefix: ezproxyPrefix.trim(),
        domainPatterns: patterns,
        almaApiBaseUrl: almaApiBaseUrl.trim(),
        almaApiKey: apiKey,
      })
      setConfig(cfg)
      setAlmaApiKey(cfg.hasAlmaApiKey ? API_KEY_PLACEHOLDER : '')
      setSaved(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save settings.')
    } finally {
      setSaving(false)
    }
  }

  if (featuresLoading) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">Library Integration</h1>
        <p className="mt-2 text-sm text-slate-500">Loading…</p>
      </main>
    )
  }

  if (!ffLibraryIntegration) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">Library Integration</h1>
        <p className="mt-2 text-sm text-slate-500">
          Library integration is not enabled for this platform. Contact your system administrator.
        </p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-3xl p-6" aria-labelledby={titleId}>
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        Library / E-Reserves Integration
      </h1>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        Configure EZproxy URL rewriting, Alma catalog search, and Leganto reading list settings.
      </p>

      {loading && <p className="mt-4 text-sm text-slate-500">Loading…</p>}

      {!loading && (
        <div className="mt-6 space-y-6">
          {/* EZproxy */}
          <section className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-700 dark:bg-neutral-800">
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">EZproxy Configuration</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              All external links matching the configured domain patterns will be automatically rewritten
              through your institution&apos;s EZproxy server.
            </p>
            <div className="mt-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                  EZproxy prefix URL
                </label>
                <input
                  type="url"
                  value={ezproxyPrefix}
                  onChange={(e) => setEzproxyPrefix(e.target.value)}
                  placeholder="https://ezproxy.university.edu"
                  className="mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                  Domain patterns (one per line)
                </label>
                <textarea
                  value={domainPatternsRaw}
                  onChange={(e) => setDomainPatternsRaw(e.target.value)}
                  rows={4}
                  placeholder={'journals.sagepub.com\n*.springer.com\nelsevier.com'}
                  className="mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                />
                <p className="mt-1 text-xs text-slate-500">
                  Use <code className="font-mono">*.example.com</code> for wildcard subdomains.
                </p>
              </div>
            </div>
          </section>

          {/* Alma */}
          <section className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-700 dark:bg-neutral-800">
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
              Ex Libris Alma API
            </h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Enables the library catalog search widget in the module editor.
            </p>
            <div className="mt-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                  Alma API base URL
                </label>
                <input
                  type="url"
                  value={almaApiBaseUrl}
                  onChange={(e) => setAlmaApiBaseUrl(e.target.value)}
                  placeholder="https://api-eu.hosted.exlibrisgroup.com"
                  className="mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                  Alma API key
                </label>
                <input
                  type="password"
                  value={almaApiKey}
                  onChange={(e) => setAlmaApiKey(e.target.value)}
                  placeholder={config?.hasAlmaApiKey ? API_KEY_PLACEHOLDER : 'Enter API key…'}
                  className="mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                />
                {config?.hasAlmaApiKey && (
                  <p className="mt-1 text-xs text-slate-500">
                    API key is set. Leave unchanged to keep the existing key.
                  </p>
                )}
              </div>
            </div>
          </section>

          {error && (
            <p role="alert" className="text-sm text-red-600 dark:text-red-400">
              {error}
            </p>
          )}
          {saved && (
            <p role="status" className="text-sm text-green-600 dark:text-green-400">
              Settings saved.
            </p>
          )}

          <div className="flex justify-end">
            <button
              type="button"
              onClick={() => void handleSave()}
              disabled={saving || loading}
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {saving ? 'Saving…' : 'Save settings'}
            </button>
          </div>
        </div>
      )}
    </main>
  )
}

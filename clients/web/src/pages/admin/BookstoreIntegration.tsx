import { useCallback, useEffect, useId, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { authorizedFetch } from '../../lib/api'

type BookstoreConfig = {
  defaultProvider: 'vitalsource' | 'redshelf'
  vitalsourceToolId: string | null
  redshelfToolId: string | null
  updatedAt?: string
}

async function fetchBookstoreConfig(): Promise<BookstoreConfig> {
  const res = await authorizedFetch('/api/v1/admin/bookstore/config')
  if (!res.ok) throw new Error('Failed to load bookstore config')
  return res.json() as Promise<BookstoreConfig>
}

async function saveBookstoreConfig(body: {
  defaultProvider: string
  vitalsourceToolId: string | null
  redshelfToolId: string | null
}): Promise<BookstoreConfig> {
  const res = await authorizedFetch('/api/v1/admin/bookstore/config', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error('Failed to save bookstore config')
  return res.json() as Promise<BookstoreConfig>
}

export default function BookstoreIntegrationPage() {
  const titleId = useId()
  const { ffBookstoreIntegration, loading: featuresLoading } = usePlatformFeatures()

  const [defaultProvider, setDefaultProvider] = useState<'vitalsource' | 'redshelf'>('vitalsource')
  const [vitalsourceToolId, setVitalsourceToolId] = useState('')
  const [redshelfToolId, setRedshelfToolId] = useState('')

  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const cfg = await fetchBookstoreConfig()
      setDefaultProvider(cfg.defaultProvider)
      setVitalsourceToolId(cfg.vitalsourceToolId ?? '')
      setRedshelfToolId(cfg.redshelfToolId ?? '')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load settings.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffBookstoreIntegration) return
    void load()
  }, [featuresLoading, ffBookstoreIntegration, load])

  async function handleSave() {
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const cfg = await saveBookstoreConfig({
        defaultProvider,
        vitalsourceToolId: vitalsourceToolId.trim() || null,
        redshelfToolId: redshelfToolId.trim() || null,
      })
      setDefaultProvider(cfg.defaultProvider)
      setVitalsourceToolId(cfg.vitalsourceToolId ?? '')
      setRedshelfToolId(cfg.redshelfToolId ?? '')
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
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">Bookstore Integration</h1>
        <p className="mt-2 text-sm text-slate-500">Loading…</p>
      </main>
    )
  }

  if (!ffBookstoreIntegration) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">Bookstore Integration</h1>
        <p className="mt-2 text-sm text-slate-500">
          Bookstore integration is not enabled for this platform. Contact your system administrator.
        </p>
      </main>
    )
  }

  const fieldClasses =
    'mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100'

  return (
    <main className="mx-auto max-w-3xl p-6" aria-labelledby={titleId}>
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        Bookstore / Textbook Integration
      </h1>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        Configure VitalSource Bridge and RedShelf BookShelf Inclusive Access delivery. Register the
        provider LTI 1.3 tools under Settings → LTI tools, then select them here.
      </p>

      {loading && <p className="mt-4 text-sm text-slate-500">Loading…</p>}

      {!loading && (
        <div className="mt-6 space-y-6">
          <section className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-700 dark:bg-neutral-800">
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Default provider</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Used as the default when an instructor adds a textbook resource.
            </p>
            <div className="mt-4">
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Provider
              </label>
              <select
                value={defaultProvider}
                onChange={(e) => setDefaultProvider(e.target.value as 'vitalsource' | 'redshelf')}
                className={fieldClasses}
              >
                <option value="vitalsource">VitalSource Bridge</option>
                <option value="redshelf">RedShelf BookShelf</option>
              </select>
            </div>
          </section>

          <section className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-700 dark:bg-neutral-800">
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
              Registered LTI tools
            </h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Paste the external-tool ID (UUID) for each registered bookstore LTI 1.3 tool.
            </p>
            <div className="mt-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                  VitalSource tool ID
                </label>
                <input
                  type="text"
                  value={vitalsourceToolId}
                  onChange={(e) => setVitalsourceToolId(e.target.value)}
                  placeholder="00000000-0000-0000-0000-000000000000"
                  className={`${fieldClasses} font-mono`}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                  RedShelf tool ID
                </label>
                <input
                  type="text"
                  value={redshelfToolId}
                  onChange={(e) => setRedshelfToolId(e.target.value)}
                  placeholder="00000000-0000-0000-0000-000000000000"
                  className={`${fieldClasses} font-mono`}
                />
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

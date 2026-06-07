import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  CONTENT_FILTER_SECRET_PLACEHOLDER,
  fetchContentFilterSettings,
  patchContentFilterSettings,
  type ContentFilterSettings,
} from '../../lib/content-filter-api'
import { apiUrl } from '../../lib/api'

export default function ContentFilterSettingsPage() {
  const titleId = useId()
  const ggKeyId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId') ?? ''
  const { ffContentFilterIntegration, loading: featuresLoading } = usePlatformFeatures()
  const [settings, setSettings] = useState<ContentFilterSettings | null>(null)
  const [goGuardianEnabled, setGoGuardianEnabled] = useState(false)
  const [securlyEnabled, setSecurlyEnabled] = useState(false)
  const [apiKey, setApiKey] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const load = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const row = await fetchContentFilterSettings(orgId)
      setSettings(row)
      setGoGuardianEnabled(row.goGuardianEnabled)
      setSecurlyEnabled(row.securlyEnabled)
      setApiKey(row.hasGoGuardianApiKey ? CONTENT_FILTER_SECRET_PLACEHOLDER : '')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load settings.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    if (featuresLoading || !ffContentFilterIntegration || !orgId) return
    void load()
  }, [featuresLoading, ffContentFilterIntegration, load, orgId])

  async function handleSave() {
    if (!orgId) return
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const body: Parameters<typeof patchContentFilterSettings>[1] = {
        goGuardianEnabled,
        securlyEnabled,
      }
      if (apiKey === '') {
        body.clearGoGuardianApiKey = true
      } else if (apiKey !== CONTENT_FILTER_SECRET_PLACEHOLDER) {
        body.goGuardianApiKey = apiKey
      }
      const row = await patchContentFilterSettings(orgId, body)
      setSettings(row)
      setApiKey(row.hasGoGuardianApiKey ? CONTENT_FILTER_SECRET_PLACEHOLDER : '')
      setSaved(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save settings.')
    } finally {
      setSaving(false)
    }
  }

  const allowlistHref = apiUrl(settings?.allowlistUrl ?? '/.well-known/content-filter-allowlist.json')

  if (featuresLoading) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">
          Content filter integration
        </h1>
        <p className="mt-6 text-sm" role="status">
          Loading…
        </p>
      </main>
    )
  }

  if (!ffContentFilterIntegration) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Content filter integration is not enabled on this platform. Enable{' '}
          <strong>Content filter integration</strong> in Settings → Global platform.
        </p>
      </main>
    )
  }

  if (!orgId) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Add an <code className="text-xs">?orgId=</code> query parameter with your district organization
          id.
        </p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-3xl p-6">
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        Content filter integration
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Configure GoGuardian and Securly so Lextures loads on district Chromebooks without being
        blocked by web-content filters.
      </p>

      {loading ? (
        <p className="mt-6 text-sm" role="status">
          Loading settings…
        </p>
      ) : null}

      {error ? (
        <p className="mt-4 text-sm text-red-700 dark:text-red-400" role="alert">
          {error}
        </p>
      ) : null}

      {saved ? (
        <p className="mt-4 text-sm text-green-700 dark:text-green-400" role="status">
          Settings saved.
        </p>
      ) : null}

      <section className="mt-8 space-y-6" aria-labelledby={titleId}>
        <div className="rounded-lg border border-slate-200 p-4 dark:border-neutral-700">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">URL allowlist</h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Share this JSON document with your district IT team to allowlist Lextures domains in
            GoGuardian, Securly, or other filters.
          </p>
          <a
            href={allowlistHref}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-3 inline-block text-sm font-medium text-indigo-600 hover:underline dark:text-indigo-400"
          >
            Download allowlist JSON
          </a>
        </div>

        <div className="rounded-lg border border-slate-200 p-4 dark:border-neutral-700">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">GoGuardian</h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            When enabled, Lextures emits educational activity events with a hashed student id so
            GoGuardian can apply per-student policies.
          </p>
          <label className="mt-4 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={goGuardianEnabled}
              onChange={(e) => setGoGuardianEnabled(e.target.checked)}
            />
            Enable GoGuardian integration
          </label>
          <div className="mt-4">
            <label htmlFor={ggKeyId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              GoGuardian API key
            </label>
            <input
              id={ggKeyId}
              type="password"
              autoComplete="off"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="Paste API key from GoGuardian admin console"
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            />
            {settings?.hasGoGuardianApiKey ? (
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-500">
                A key is stored. Leave the placeholder to keep it, or clear the field to remove.
              </p>
            ) : null}
          </div>
        </div>

        <div className="rounded-lg border border-slate-200 p-4 dark:border-neutral-700">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Securly</h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Include Lextures in the Securly educational app catalog via the allowlist document.
          </p>
          <label className="mt-4 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={securlyEnabled}
              onChange={(e) => setSecurlyEnabled(e.target.checked)}
            />
            Enable Securly catalog entry
          </label>
        </div>

        <button
          type="button"
          onClick={() => void handleSave()}
          disabled={saving || loading}
          className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Save settings'}
        </button>
      </section>
    </main>
  )
}

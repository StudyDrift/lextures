import { useCallback, useEffect, useId, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchAdminTranscriptsConfig,
  saveAdminTranscriptsConfig,
  type TranscriptsConfig,
} from '../../lib/transcripts-api'

const SECRET_PLACEHOLDER = '••••••••••••'

export function TranscriptsSettingsPanel() {
  const { ffTranscripts, loading: featuresLoading } = usePlatformFeatures()
  const urlId = useId()
  const secretId = useId()
  const [config, setConfig] = useState<TranscriptsConfig | null>(null)
  const [webhookUrl, setWebhookUrl] = useState('')
  const [webhookSecret, setWebhookSecret] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const load = useCallback(async () => {
    if (!ffTranscripts) {
      setConfig(null)
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const cfg = await fetchAdminTranscriptsConfig()
      setConfig(cfg)
      setWebhookUrl(cfg.webhookUrl)
      setWebhookSecret(cfg.hasWebhookSecret ? SECRET_PLACEHOLDER : '')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load transcripts settings.')
    } finally {
      setLoading(false)
    }
  }, [ffTranscripts])

  useEffect(() => {
    if (featuresLoading) return
    void load()
  }, [load, featuresLoading])

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const payload: { webhookUrl: string; webhookSecret?: string } = {
        webhookUrl: webhookUrl.trim(),
      }
      if (webhookSecret.trim() && webhookSecret !== SECRET_PLACEHOLDER) {
        payload.webhookSecret = webhookSecret.trim()
      }
      const cfg = await saveAdminTranscriptsConfig(payload)
      setConfig(cfg)
      setWebhookSecret(cfg.hasWebhookSecret ? SECRET_PLACEHOLDER : '')
      setSaved(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save settings.')
    } finally {
      setSaving(false)
    }
  }

  if (featuresLoading) {
    return (
      <section>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Transcripts</h2>
        <p className="mt-4 text-sm text-slate-500">Loading…</p>
      </section>
    )
  }

  if (!ffTranscripts) {
    return (
      <section>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Transcripts</h2>
        <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">
          Transcripts is not enabled for this platform. Turn on{' '}
          <span className="font-medium">Transcripts</span> under Global platform settings.
        </p>
      </section>
    )
  }

  return (
    <section>
      <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Transcripts</h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
        Register your institution&apos;s webhook URL. When a student requests an official transcript,
        Lextures sends a POST request to this endpoint with the student&apos;s information.
      </p>

      {error && (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
      {saved && (
        <p role="status" className="mt-4 text-sm text-emerald-600 dark:text-emerald-400">
          Settings saved.
        </p>
      )}

      {loading ? (
        <p className="mt-4 text-sm text-slate-500">Loading configuration…</p>
      ) : (
        <form onSubmit={handleSave} className="mt-6 space-y-4">
          <div>
            <label htmlFor={urlId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Webhook URL
            </label>
            <input
              id={urlId}
              type="url"
              required
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
              placeholder="https://sis.example.edu/api/transcript-requests"
              className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
            />
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              POST-only endpoint. Receives JSON with requestId, requestedAt, and student profile fields.
            </p>
          </div>
          <div>
            <label htmlFor={secretId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Webhook secret (optional)
            </label>
            <input
              id={secretId}
              type="password"
              autoComplete="off"
              value={webhookSecret}
              onChange={(e) => setWebhookSecret(e.target.value)}
              placeholder={config?.hasWebhookSecret ? SECRET_PLACEHOLDER : 'HMAC signing secret'}
              className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
            />
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              When set, requests include an <code className="font-mono">X-Lextures-Signature</code>{' '}
              header (HMAC-SHA256 of the body).
            </p>
          </div>
          <button
            type="submit"
            disabled={saving}
            className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save configuration'}
          </button>
        </form>
      )}
    </section>
  )
}

import { type FormEvent, useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Loader2, Save } from 'lucide-react'
import { fetchAdminSettings, putAdminSettings, type AdminSettings } from '../../lib/admin-console-api'

export default function AdminSettingsPage() {
  const formId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [form, setForm] = useState<AdminSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setForm(await fetchAdminSettings(orgId))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load settings.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    void load()
  }, [load])

  async function onSave(e: FormEvent) {
    e.preventDefault()
    if (!form) return
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      setForm(await putAdminSettings(form, orgId))
      setMessage('Settings saved.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save settings.')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <p className="text-sm text-fg-muted">Loading settings…</p>
  if (!form) return null

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg-default dark:text-slate-100">Settings</h1>
      <p className="mt-1 text-sm text-fg-muted dark:text-fg-subtle">
        Organization name, branding, timezone, and locale.
      </p>

      <form id={formId} onSubmit={(e) => void onSave(e)} className="mt-6 max-w-xl space-y-4">
        <label className="block text-sm">
          <span className="text-fg-muted dark:text-fg-subtle">Organization name</span>
          <input
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            className="mt-1 w-full rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
          />
        </label>
        <label className="block text-sm">
          <span className="text-fg-muted dark:text-fg-subtle">Subdomain slug</span>
          <input value={form.slug} readOnly className="mt-1 w-full rounded-lg border border-border-default bg-surface-base px-3 py-2 dark:border-border-subtle dark:bg-surface-base" />
        </label>
        <label className="block text-sm">
          <span className="text-fg-muted dark:text-fg-subtle">Primary brand color</span>
          <input
            type="color"
            value={form.primaryColor}
            onChange={(e) => setForm({ ...form, primaryColor: e.target.value })}
            className="mt-1 h-10 w-20 cursor-pointer rounded border border-border-strong"
          />
        </label>
        <label className="block text-sm">
          <span className="text-fg-muted dark:text-fg-subtle">Timezone</span>
          <input
            value={form.timezone ?? ''}
            onChange={(e) => setForm({ ...form, timezone: e.target.value })}
            placeholder="America/New_York"
            className="mt-1 w-full rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
          />
        </label>
        <label className="block text-sm">
          <span className="text-fg-muted dark:text-fg-subtle">Locale</span>
          <input
            value={form.locale ?? ''}
            onChange={(e) => setForm({ ...form, locale: e.target.value })}
            placeholder="en-US"
            className="mt-1 w-full rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
          />
        </label>
        <label className="block text-sm">
          <span className="text-fg-muted dark:text-fg-subtle">Email from display name</span>
          <input
            value={form.customEmailDisplayName ?? ''}
            onChange={(e) => setForm({ ...form, customEmailDisplayName: e.target.value })}
            className="mt-1 w-full rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
          />
        </label>

        <button
          type="submit"
          disabled={saving}
          className="inline-flex items-center gap-2 rounded-lg bg-accent-solid px-4 py-2 text-sm font-medium text-white hover:bg-accent disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <Save className="h-4 w-4" aria-hidden />}
          Save
        </button>
      </form>

      {message ? <p className="mt-3 text-sm text-success-fg">{message}</p> : null}
      {error ? (
        <p role="alert" className="mt-3 text-sm text-danger-fg">
          {error}
        </p>
      ) : null}
    </div>
  )
}

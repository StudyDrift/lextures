import { type FormEvent, useCallback, useEffect, useId, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Loader2, Megaphone, Save, Trash2 } from 'lucide-react'
import {
  createAdminBanner,
  deleteAdminBanner,
  fetchAdminBanners,
  type MaintenanceBanner,
  updateAdminBanner,
} from '../../lib/banner-api'
import { fetchAdminConsoleCapabilities } from '../../lib/admin-console-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

type BannerForm = {
  scope: 'global' | 'org'
  message: string
  severity: 'info' | 'warning' | 'error'
  ctaText: string
  ctaUrl: string
  startsAt: string
  expiresAt: string
}

const emptyForm = (): BannerForm => ({
  scope: 'org',
  message: '',
  severity: 'warning',
  ctaText: '',
  ctaUrl: '',
  startsAt: '',
  expiresAt: '',
})

function toLocalInputValue(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function fromLocalInputValue(value: string): string | undefined {
  if (!value.trim()) return undefined
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return undefined
  return d.toISOString()
}

export default function AdminBannersPage() {
  const formId = useId()
  const { maintenanceBannerEnabled } = usePlatformFeatures()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [form, setForm] = useState<BannerForm>(emptyForm)
  const [existing, setExisting] = useState<MaintenanceBanner[]>([])
  const [isGlobalAdmin, setIsGlobalAdmin] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const previewBanner = useMemo<MaintenanceBanner | null>(() => {
    if (!form.message.trim()) return null
    return {
      id: 'preview',
      scope: form.scope,
      message: form.message.trim(),
      severity: form.severity,
      ctaText: form.ctaText.trim() || undefined,
      ctaUrl: form.ctaUrl.trim() || undefined,
      isActive: true,
      updatedAt: new Date().toISOString(),
    }
  }, [form])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const caps = await fetchAdminConsoleCapabilities()
      setIsGlobalAdmin(caps.isGlobalAdmin)
      const list = await fetchAdminBanners(orgId, caps.isGlobalAdmin ? undefined : undefined)
      setExisting(list)
      const active = list.find((b) => b.isActive)
      if (active) {
        setForm({
          scope: active.scope,
          message: active.message,
          severity: active.severity,
          ctaText: active.ctaText ?? '',
          ctaUrl: active.ctaUrl ?? '',
          startsAt: toLocalInputValue(active.startsAt),
          expiresAt: toLocalInputValue(active.expiresAt),
        })
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load banners.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    void load()
  }, [load])

  async function onPublish(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const body = {
        scope: form.scope,
        message: form.message.trim(),
        severity: form.severity,
        ctaText: form.ctaText.trim() || undefined,
        ctaUrl: form.ctaUrl.trim() || undefined,
        startsAt: fromLocalInputValue(form.startsAt),
        expiresAt: fromLocalInputValue(form.expiresAt),
      }
      const active = existing.find((b) => b.isActive && b.scope === form.scope)
      if (active) {
        await updateAdminBanner(active.id, { ...body, isActive: true }, orgId)
      } else {
        await createAdminBanner(body, orgId)
      }
      setMessage('Banner published.')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to publish banner.')
    } finally {
      setSaving(false)
    }
  }

  async function onClear() {
    const active = existing.find((b) => b.isActive)
    if (!active) {
      setForm(emptyForm())
      return
    }
    setSaving(true)
    setError(null)
    try {
      await deleteAdminBanner(active.id, orgId)
      setForm(emptyForm())
      setMessage('Banner cleared.')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear banner.')
    } finally {
      setSaving(false)
    }
  }

  if (!maintenanceBannerEnabled) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <p className="text-sm text-slate-500">
          Maintenance banners are disabled. Enable them in Settings → Global platform.
        </p>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <p className="text-sm text-slate-500">Loading banners…</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="flex items-center gap-2">
        <Megaphone className="h-5 w-5 text-indigo-600" aria-hidden />
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">Maintenance notices</h1>
      </div>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
        Publish a site-wide or organization notice visible to users until dismissed or expired.
      </p>

      {previewBanner ? (
        <div className="mt-4 overflow-hidden rounded-lg border border-slate-200 dark:border-neutral-800">
          <p className="bg-slate-50 px-3 py-1 text-xs font-medium uppercase tracking-wide text-slate-500 dark:bg-neutral-950">
            Preview
          </p>
          <aside
            role="status"
            className={`flex items-start gap-3 border-b px-4 py-2 text-sm ${
              previewBanner.severity === 'error'
                ? 'border-red-200 bg-red-50 text-red-950 dark:border-red-900/60 dark:bg-red-950/50 dark:text-red-100'
                : previewBanner.severity === 'warning'
                  ? 'border-amber-200 bg-amber-50 text-amber-950 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-100'
                  : 'border-sky-200 bg-sky-50 text-sky-950 dark:border-sky-900/60 dark:bg-sky-950/40 dark:text-sky-100'
            }`}
          >
            <p className="flex-1">{previewBanner.message}</p>
          </aside>
        </div>
      ) : null}

      <form id={formId} onSubmit={(e) => void onPublish(e)} className="mt-6 max-w-xl space-y-4">
        {isGlobalAdmin ? (
          <label className="block text-sm">
            <span className="text-slate-600 dark:text-slate-400">Scope</span>
            <select
              value={form.scope}
              onChange={(e) => setForm({ ...form, scope: e.target.value as 'global' | 'org' })}
              className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
            >
              <option value="org">Organization only</option>
              <option value="global">Global (all organizations)</option>
            </select>
          </label>
        ) : null}
        <label className="block text-sm">
          <span className="text-slate-600 dark:text-slate-400">Message</span>
          <textarea
            value={form.message}
            onChange={(e) => setForm({ ...form, message: e.target.value })}
            maxLength={500}
            rows={3}
            required
            placeholder="Scheduled maintenance Sunday 2am–4am UTC"
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
          />
        </label>
        <label className="block text-sm">
          <span className="text-slate-600 dark:text-slate-400">Severity</span>
          <select
            value={form.severity}
            onChange={(e) => setForm({ ...form, severity: e.target.value as BannerForm['severity'] })}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
          >
            <option value="info">Info</option>
            <option value="warning">Warning</option>
            <option value="error">Error</option>
          </select>
        </label>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block text-sm">
            <span className="text-slate-600 dark:text-slate-400">Start (optional)</span>
            <input
              type="datetime-local"
              value={form.startsAt}
              onChange={(e) => setForm({ ...form, startsAt: e.target.value })}
              className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </label>
          <label className="block text-sm">
            <span className="text-slate-600 dark:text-slate-400">Expires (optional)</span>
            <input
              type="datetime-local"
              value={form.expiresAt}
              onChange={(e) => setForm({ ...form, expiresAt: e.target.value })}
              className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </label>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block text-sm">
            <span className="text-slate-600 dark:text-slate-400">CTA label (optional)</span>
            <input
              value={form.ctaText}
              onChange={(e) => setForm({ ...form, ctaText: e.target.value })}
              className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </label>
          <label className="block text-sm">
            <span className="text-slate-600 dark:text-slate-400">CTA URL (optional)</span>
            <input
              type="url"
              value={form.ctaUrl}
              onChange={(e) => setForm({ ...form, ctaUrl: e.target.value })}
              className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </label>
        </div>
        {message ? <p className="text-sm text-green-700 dark:text-green-400">{message}</p> : null}
        {error ? <p className="text-sm text-red-600 dark:text-red-400">{error}</p> : null}
        <div className="flex flex-wrap gap-2">
          <button
            type="submit"
            disabled={saving}
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-60"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            Publish
          </button>
          <button
            type="button"
            disabled={saving}
            onClick={() => void onClear()}
            className="inline-flex items-center gap-2 rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:text-slate-200 dark:hover:bg-neutral-900"
          >
            <Trash2 className="h-4 w-4" />
            Clear active banner
          </button>
        </div>
      </form>
    </div>
  )
}

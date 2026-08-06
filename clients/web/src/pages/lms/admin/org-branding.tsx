import { type FormEvent, useCallback, useEffect, useId, useState } from 'react'
import { Loader2, Palette, Save } from 'lucide-react'
import { resolveOrgBrandAssetUrl } from '../../../lib/branding-url'
import { decodeJwtPayload } from '../../../lib/jwt-payload'
import { getAccessToken } from '../../../lib/auth'
import { authorizedFetch } from '../../../lib/api'
import { readApiErrorMessage } from '../../../lib/errors'
import { toastMutationError, toastSaveOk } from '../../../lib/lms-toast'
import { oklchToHex, parseOklch } from '../../../lib/tokens/oklch'
import { OrgBrandingAccentField } from './org-branding-accent-field'
import { OrgBrandingPreview } from './org-branding-preview'

type BrandingResponse = {
  logoUrl: string | null
  faviconUrl: string | null
  primaryColor: string
  secondaryColor: string
  customDomain: string | null
  customEmailDisplayName: string | null
  contrastWarningPrimary: boolean
  contrastRatioPrimary: number | null
  accentOklch?: string | null
  derivedRamp?: Record<string, string>
  tokensVersion?: number
}

function contrastOk(ratio: number) {
  return ratio >= 4.5
}

/**
 * Settings — organization branding (plan 5.7). Requires org unit admin or global admin.
 */
export default function OrgBranding() {
  const formId = useId()
  const [orgId, setOrgId] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [form, setForm] = useState<BrandingResponse>({
    logoUrl: null,
    faviconUrl: null,
    primaryColor: '#4F46E5',
    secondaryColor: '#7C3AED',
    customDomain: null,
    customEmailDisplayName: null,
    contrastWarningPrimary: false,
    contrastRatioPrimary: null,
    accentOklch: null,
    derivedRamp: {},
    tokensVersion: 1,
  })
  const [previewLogoUrl, setPreviewLogoUrl] = useState<string | null>(null)
  const [accentInput, setAccentInput] = useState('')
  const [accentHex, setAccentHex] = useState('#4F46E5')

  const jwtOrg = decodeJwtPayload(getAccessToken())?.org_id ?? null
  useEffect(() => {
    if (jwtOrg) setOrgId(jwtOrg)
  }, [jwtOrg])

  const load = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/branding`)
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setError(readApiErrorMessage(raw))
        return
      }
      const b = raw as BrandingResponse
      setForm(b)
      setPreviewLogoUrl(resolveOrgBrandAssetUrl(b.logoUrl))
      const a = b.accentOklch?.trim() || ''
      setAccentInput(a)
      if (a) {
        const p = parseOklch(a)
        if (p) setAccentHex(oklchToHex(p))
      } else {
        setAccentHex(b.primaryColor || '#4F46E5')
      }
    } catch {
      setError('Could not load branding.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    void load()
  }, [load])

  async function onSave(e: FormEvent) {
    e.preventDefault()
    if (!orgId) return
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/branding`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          logoUrl: form.logoUrl,
          faviconUrl: form.faviconUrl,
          primaryColor: form.primaryColor,
          secondaryColor: form.secondaryColor,
          customDomain: form.customDomain,
          customEmailDisplayName: form.customEmailDisplayName,
          accentOklch: accentInput.trim() ? accentInput.trim() : null,
        }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        const errBody = raw as {
          error?: string
          failingPairs?: { fg: string; bg: string; ratio: number; required: number }[]
          suggestion?: string | null
          message?: string
        }
        if (res.status === 422 && errBody.error === 'brand_accent_contrast') {
          const pairs = (errBody.failingPairs ?? [])
            .map((p) => `${p.fg} on ${p.bg} (${p.ratio}:1 < ${p.required}:1)`)
            .join('; ')
          const msg = `Accent fails contrast: ${pairs || 'AA'}${
            errBody.suggestion ? `. Try ${errBody.suggestion}` : ''
          }`
          setError(msg)
          toastMutationError(msg)
          return
        }
        setError(readApiErrorMessage(raw))
        toastMutationError(readApiErrorMessage(raw))
        return
      }
      const b = raw as BrandingResponse
      setForm(b)
      setPreviewLogoUrl(resolveOrgBrandAssetUrl(b.logoUrl))
      const a = b.accentOklch?.trim() || ''
      setAccentInput(a)
      setMessage('Saved.')
      toastSaveOk('Branding saved')
    } catch {
      setError('Could not save.')
      toastMutationError('Could not save.')
    } finally {
      setSaving(false)
    }
  }

  async function upload(kind: 'logo' | 'favicon', file: File) {
    if (!orgId) return
    const fd = new FormData()
    fd.set('file', file)
    const res = await authorizedFetch(
      `/api/v1/orgs/${encodeURIComponent(orgId)}/branding/${kind}`,
      { method: 'POST', body: fd },
    )
    const raw: unknown = await res.json().catch(() => ({}))
    if (!res.ok) {
      toastMutationError(readApiErrorMessage(raw))
      return
    }
    const u = raw as { url?: string }
    if (u.url && kind === 'logo') {
      setForm((f) => ({ ...f, logoUrl: u.url ?? null }))
      setPreviewLogoUrl(resolveOrgBrandAssetUrl(u.url))
    }
    if (u.url && kind === 'favicon') {
      setForm((f) => ({ ...f, faviconUrl: u.url ?? null }))
    }
    toastSaveOk(kind === 'logo' ? 'Logo uploaded' : 'Favicon uploaded')
  }

  const approxWarn =
    form.contrastWarningPrimary ||
    (form.contrastRatioPrimary != null && !contrastOk(form.contrastRatioPrimary))

  if (loading) {
    return <p className="mt-8 text-sm text-fg-muted">Loading…</p>
  }
  if (!orgId) {
    return (
      <p className="mt-8 text-sm text-warning-fg">No organization id on your session.</p>
    )
  }

  return (
    <div className="mt-4 grid gap-8 lg:grid-cols-2">
      <form id={formId} className="space-y-6" onSubmit={onSave}>
        <div>
          <label className="mb-2 block text-sm font-medium text-fg-default">Logo</label>
          <input
            type="file"
            accept="image/png,image/jpeg,image/gif,image/svg+xml"
            aria-label="Upload organization logo"
            className="block w-full text-sm text-fg-muted file:me-4 file:rounded-lg file:border file:border-border-default file:bg-surface-raised file:px-3 file:py-2 file:text-sm file:font-medium dark:text-fg-muted dark:file:border-border-default dark:file:bg-surface-overlay"
            onChange={(e) => {
              const f = e.target.files?.[0]
              if (f) void upload('logo', f)
              e.target.value = ''
            }}
          />
          <p className="mt-1 text-xs text-fg-muted">
            PNG, JPEG, GIF, or SVG. Shown on the sign-in page and navigation.
          </p>
        </div>
        <div>
          <label className="mb-2 block text-sm font-medium text-fg-default">Favicon</label>
          <input
            type="file"
            accept="image/png,image/jpeg,image/gif,image/svg+xml"
            aria-label="Upload favicon"
            className="block w-full text-sm text-fg-muted file:me-4 file:rounded-lg file:border file:border-border-default file:bg-surface-raised file:px-3 file:py-2 file:text-sm file:font-medium dark:text-fg-muted dark:file:border-border-default dark:file:bg-surface-overlay"
            onChange={(e) => {
              const f = e.target.files?.[0]
              if (f) void upload('favicon', f)
              e.target.value = ''
            }}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label
              className="mb-2 flex items-center gap-2 text-sm font-medium text-fg-default"
              htmlFor={`${formId}-primary`}
            >
              <Palette className="h-4 w-4" aria-hidden />
              Primary color
            </label>
            <div className="flex gap-2">
              <input
                id={`${formId}-primary`}
                type="color"
                value={form.primaryColor}
                onChange={(e) => setForm((f) => ({ ...f, primaryColor: e.target.value }))}
                className="h-10 w-14 cursor-pointer rounded border border-border-default bg-surface-raised dark:border-border-default"
              />
              <input
                type="text"
                value={form.primaryColor}
                onChange={(e) => setForm((f) => ({ ...f, primaryColor: e.target.value }))}
                className="flex-1 rounded-lg border border-border-default bg-surface-raised px-3 py-2 font-mono text-sm dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
                autoComplete="off"
              />
            </div>
          </div>
          <div>
            <label
              className="mb-2 block text-sm font-medium text-fg-default"
              htmlFor={`${formId}-secondary`}
            >
              Secondary color
            </label>
            <div className="flex gap-2">
              <input
                id={`${formId}-secondary`}
                type="color"
                value={form.secondaryColor}
                onChange={(e) => setForm((f) => ({ ...f, secondaryColor: e.target.value }))}
                className="h-10 w-14 cursor-pointer rounded border border-border-default bg-surface-raised dark:border-border-default"
              />
              <input
                type="text"
                value={form.secondaryColor}
                onChange={(e) => setForm((f) => ({ ...f, secondaryColor: e.target.value }))}
                className="flex-1 rounded-lg border border-border-default bg-surface-raised px-3 py-2 font-mono text-sm dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
                autoComplete="off"
              />
            </div>
          </div>
        </div>
        {approxWarn ? (
          <div
            role="status"
            className="rounded-lg border border-warning-border bg-warning-surface px-3 py-2 text-sm text-warning-fg"
          >
            This color may not meet WCAG AA contrast requirements against white (need 4.5:1 for
            normal text).
            {form.contrastRatioPrimary != null && (
              <span className="ms-1">
                Current ratio (approx.): {form.contrastRatioPrimary.toFixed(2)}:1
              </span>
            )}
          </div>
        ) : null}
        <OrgBrandingAccentField
          formId={formId}
          accentInput={accentInput}
          accentHex={accentHex}
          onAccentInputChange={setAccentInput}
          onAccentHexChange={setAccentHex}
          onClear={() => {
            setAccentInput('')
            setAccentHex('#4F46E5')
          }}
        />
        <div>
          <label
            className="mb-2 block text-sm font-medium text-fg-default"
            htmlFor={`${formId}-domain`}
          >
            Custom domain
          </label>
          <input
            id={`${formId}-domain`}
            type="text"
            value={form.customDomain ?? ''}
            onChange={(e) =>
              setForm((f) => ({
                ...f,
                customDomain: e.target.value.trim() ? e.target.value : null,
              }))
            }
            placeholder="lms.yourschool.edu"
            className="w-full rounded-lg border border-border-default bg-surface-raised px-3 py-2 text-sm dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
          />
          <p className="mt-1 text-xs text-fg-muted">
            Point DNS to this service; map the hostname to your org after it resolves.
          </p>
        </div>
        <div>
          <label
            className="mb-2 block text-sm font-medium text-fg-default"
            htmlFor={`${formId}-emailname`}
          >
            Email display name
          </label>
          <input
            id={`${formId}-emailname`}
            type="text"
            value={form.customEmailDisplayName ?? ''}
            onChange={(e) =>
              setForm((f) => ({
                ...f,
                customEmailDisplayName: e.target.value.trim() ? e.target.value : null,
              }))
            }
            placeholder="Your District Name"
            className="w-full rounded-lg border border-border-default bg-surface-raised px-3 py-2 text-sm dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
          />
          <p className="mt-1 text-xs text-fg-muted">
            Shown as the sender name on password-reset and similar emails.
          </p>
        </div>
        {error ? (
          <p className="text-sm text-rose-600 dark:text-rose-400" role="alert">
            {error}
          </p>
        ) : null}
        {message ? (
          <p className="text-sm text-emerald-700 dark:text-emerald-400" role="status">
            {message}
          </p>
        ) : null}
        <button
          type="submit"
          disabled={saving}
          className="inline-flex items-center gap-2 rounded-xl bg-accent-solid px-4 py-2.5 text-sm font-semibold text-fg-on-accent shadow-sm transition-[background-color,color,border-color] hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {saving ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
          ) : (
            <Save className="h-4 w-4" aria-hidden />
          )}
          Save branding
        </button>
      </form>
      <OrgBrandingPreview primaryColor={form.primaryColor} previewLogoUrl={previewLogoUrl} />
    </div>
  )
}

import { useEffect, useId, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchInstallPreview,
  installTool,
  type InstallPreview,
} from '../../lib/content-tool-marketplace-api'
import { authorizedFetch } from '../../lib/api'

export default function ToolMarketplaceDetailPage() {
  const { t } = useTranslation('contentTools')
  const titleId = useId()
  const { toolId = '' } = useParams()
  const { ffContentToolMarketplace } = usePlatformFeatures()
  const [preview, setPreview] = useState<InstallPreview | null>(null)
  const [orgId, setOrgId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [installed, setInstalled] = useState(false)

  useEffect(() => {
    if (!ffContentToolMarketplace || !toolId) return
    let cancelled = false
    void (async () => {
      try {
        const meRes = await authorizedFetch('/api/v1/me')
        if (!meRes.ok) throw new Error('Failed to load profile')
        const me = (await meRes.json()) as { orgId?: string; org?: { id?: string } }
        const oid = me.org?.id || me.orgId || null
        if (!oid) throw new Error('No organization')
        if (cancelled) return
        setOrgId(oid)
        const p = await fetchInstallPreview(oid, toolId)
        if (!cancelled) setPreview(p)
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [ffContentToolMarketplace, toolId])

  if (!ffContentToolMarketplace) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <p className="text-sm">{t('contentTools.marketplace.disabled')}</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl p-6" data-testid="tool-marketplace-detail">
      <h1 id={titleId} className="text-2xl font-semibold text-fg-default dark:text-slate-100">
        {preview?.displayName || toolId}
      </h1>
      {error ? (
        <p className="mt-4 text-sm text-rose-700" role="alert">
          {error}
        </p>
      ) : null}
      {preview ? (
        <section className="mt-6" aria-labelledby={titleId}>
          <h2 className="text-base font-medium">{t('contentTools.marketplace.consentTitle')}</h2>
          <p className="mt-1 text-sm text-fg-muted">{t('contentTools.marketplace.consentHelp')}</p>
          <ul className="mt-3 list-disc space-y-1 ps-5 text-sm" data-testid="tool-consent-capabilities">
            {preview.capabilities.map((c) => (
              <li key={c.capability}>{c.plainLanguage}</li>
            ))}
          </ul>
          {preview.hosts.length > 0 ? (
            <div className="mt-4">
              <h3 className="text-sm font-medium">{t('contentTools.marketplace.hostsTitle')}</h3>
              <ul className="mt-1 list-disc ps-5 text-sm">
                {preview.hosts.map((h) => (
                  <li key={h}>{h}</li>
                ))}
              </ul>
            </div>
          ) : null}
          <button
            type="button"
            className="mt-6 rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-surface-sunken dark:text-fg-default"
            data-testid="tool-install-confirm"
            disabled={busy || !orgId || installed}
            onClick={() => {
              if (!orgId) return
              setBusy(true)
              setError(null)
              void installTool(orgId, { toolId, consented: true })
                .then(() => setInstalled(true))
                .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Install failed'))
                .finally(() => setBusy(false))
            }}
          >
            {installed
              ? t('contentTools.marketplace.installed')
              : t('contentTools.marketplace.install')}
          </button>
        </section>
      ) : null}
    </div>
  )
}

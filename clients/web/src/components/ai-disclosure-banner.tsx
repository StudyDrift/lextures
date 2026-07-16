import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { authorizedFetch } from '../lib/api'
import { aiDisclosureI18n, aiDisclosureProviderPhrase } from '../lib/ai-disclosure-i18n'
import { providerLabel } from '../lib/ai-providers'
import { usePlatformFeatures } from '../context/platform-features-context'

type Props = {
  featureKey: string
  modelLabel?: string
  /** Override active provider display name; defaults from platform features. */
  providerLabel?: string
}

export function AiDisclosureBanner({ featureKey, modelLabel, providerLabel: providerLabelProp }: Props) {
  const [visible, setVisible] = useState(false)
  const [busy, setBusy] = useState(false)
  const { aiProvidersConfigured } = usePlatformFeatures()

  const activeProviderLabel = useMemo(() => {
    if (providerLabelProp?.trim()) return providerLabelProp.trim()
    const first = aiProvidersConfigured?.[0]
    return first ? providerLabel(first) : undefined
  }, [providerLabelProp, aiProvidersConfigured])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/settings/ai-disclosure/acknowledgements')
        if (!res.ok) {
          return
        }
        const data = (await res.json()) as { features?: string[] }
        if (!cancelled) {
          const acked = data.features ?? []
          setVisible(!acked.includes(featureKey))
        }
      } catch {
        /* ignore — banner is optional UX */
      }
    })()
    return () => {
      cancelled = true
    }
  }, [featureKey])

  const acknowledge = useCallback(async () => {
    setBusy(true)
    try {
      const res = await authorizedFetch('/api/v1/settings/ai-disclosure/acknowledgements', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ featureKey }),
      })
      if (res.ok || res.status === 204) {
        setVisible(false)
      }
    } finally {
      setBusy(false)
    }
  }, [featureKey])

  if (!visible) return null

  const via = aiDisclosureProviderPhrase(activeProviderLabel)

  return (
    <div
      className="mb-4 rounded-xl border border-indigo-200 bg-indigo-50 px-4 py-3 text-sm text-indigo-950 dark:border-indigo-800 dark:bg-indigo-950/40 dark:text-indigo-100"
      role="region"
      aria-label={aiDisclosureI18n.bannerTitle}
    >
      <p className="font-semibold">{aiDisclosureI18n.bannerTitle}</p>
      <p className="mt-1">
        {modelLabel ? (
          <>
            This feature uses <strong>{modelLabel}</strong> via {via}.{' '}
          </>
        ) : (
          <>This feature uses an AI model via {via}. </>
        )}
        {aiDisclosureI18n.bannerBody}
      </p>
      <div className="mt-3 flex flex-wrap items-center gap-3">
        <button
          type="button"
          disabled={busy}
          onClick={() => void acknowledge()}
          className="rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
        >
          {aiDisclosureI18n.bannerUnderstand}
        </button>
        <Link to="/settings/account" className="text-xs font-medium underline underline-offset-2">
          {aiDisclosureI18n.bannerOptOutLink}
        </Link>
        <Link to="/ai-disclosure" className="text-xs font-medium underline underline-offset-2">
          {aiDisclosureI18n.fullDisclosureLink}
        </Link>
      </div>
    </div>
  )
}

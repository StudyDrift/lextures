import { useCallback, useEffect, useState } from 'react'
import {
  CLOUD_PROVIDER_LABELS,
  fetchCloudProviders,
  type CloudProviderId,
  type ConfiguredCloudProvider,
} from '../lib/cloud-providers-api'
import { createCloudPicker, type PickedFile } from '../services/cloud-picker'

type CloudPickerButtonsProps = {
  onPicked: (file: PickedFile) => void
  disabled?: boolean
  label?: string
}

export function CloudPickerButtons({
  onPicked,
  disabled,
  label = 'Or link from cloud storage',
}: CloudPickerButtonsProps) {
  const [providers, setProviders] = useState<ConfiguredCloudProvider[]>([])
  const [loading, setLoading] = useState(true)
  const [picking, setPicking] = useState<CloudProviderId | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    void fetchCloudProviders()
      .then((list) => {
        if (!cancelled) setProviders(list)
      })
      .catch(() => {
        if (!cancelled) setProviders([])
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const handlePick = useCallback(async (provider: CloudProviderId, config: ConfiguredCloudProvider) => {
    setError(null)
    setPicking(provider)
    try {
      const picker = createCloudPicker(provider, config, 'link')
      const file = await picker.pick()
      if (file) onPicked(file)
    } catch (e) {
      setError(e instanceof Error ? e.message : `Could not open ${CLOUD_PROVIDER_LABELS[provider]} picker.`)
    } finally {
      setPicking(null)
    }
  }, [onPicked])

  if (loading || providers.length === 0) return null

  return (
    <div>
      <p className="mb-2 text-xs font-medium text-slate-600 dark:text-neutral-300">
        {label}
      </p>
      <div className="flex flex-wrap gap-2">
        {providers.map((provider) => (
          <button
            key={provider.provider}
            type="button"
            disabled={disabled || picking !== null}
            onClick={() => void handlePick(provider.provider, provider)}
            aria-haspopup="dialog"
            className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700"
          >
            {picking === provider.provider ? 'Opening…' : CLOUD_PROVIDER_LABELS[provider.provider]}
          </button>
        ))}
      </div>
      {error && (
        <p className="mt-2 text-xs text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}

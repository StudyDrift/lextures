import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { ChevronDown, Download } from 'lucide-react'
import {
  CLOUD_PROVIDER_LABELS,
  fetchCloudProviders,
  type CloudProviderId,
  type ConfiguredCloudProvider,
} from '../lib/cloud-providers-api'
import { createCloudPicker, downloadPickedFile } from '../services/cloud-picker'

type CloudImportMenuProps = {
  disabled?: boolean
  onImportFile: (file: File) => void | Promise<void>
  onError?: (message: string) => void
  className?: string
}

export function CloudImportMenu({
  disabled = false,
  onImportFile,
  onError,
  className,
}: CloudImportMenuProps) {
  const [providers, setProviders] = useState<ConfiguredCloudProvider[]>([])
  const [loading, setLoading] = useState(true)
  const [open, setOpen] = useState(false)
  const [picking, setPicking] = useState<CloudProviderId | null>(null)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

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

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  const handlePick = useCallback(async (provider: CloudProviderId, config: ConfiguredCloudProvider) => {
    setOpen(false)
    setPicking(provider)
    try {
      const picker = createCloudPicker(provider, config, 'import')
      const picked = await picker.pick()
      if (!picked) return
      const file = await downloadPickedFile(picked)
      await onImportFile(file)
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Could not import from cloud storage.'
      onError?.(message)
    } finally {
      setPicking(null)
    }
  }, [onError, onImportFile])

  if (loading || providers.length === 0) return null

  const isBusy = disabled || picking !== null

  return (
    <div ref={rootRef} className={`relative inline-block ${className ?? ''}`}>
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        disabled={isBusy}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
      >
        {picking ? (
          'Importing…'
        ) : (
          <>
            <Download className="h-4 w-4" aria-hidden />
            Import
            <ChevronDown
              className={`h-4 w-4 shrink-0 transition ${open ? 'rotate-180' : ''}`}
              aria-hidden
            />
          </>
        )}
      </button>
      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Import from cloud storage"
          className="absolute end-0 z-50 mt-1 min-w-[12rem] overflow-hidden rounded-lg border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
        >
          {providers.map((provider) => (
            <button
              key={provider.provider}
              type="button"
              role="menuitem"
              aria-haspopup="dialog"
              onClick={() => void handlePick(provider.provider, provider)}
              className="flex w-full flex-col gap-0.5 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:hover:bg-neutral-800"
            >
              <span className="font-medium text-slate-900 dark:text-neutral-100">
                {CLOUD_PROVIDER_LABELS[provider.provider]}
              </span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">
                Import from {CLOUD_PROVIDER_LABELS[provider.provider]}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

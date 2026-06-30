import { useCallback, useEffect, useState } from 'react'
import { AlertTriangle, Info, X, XCircle } from 'lucide-react'
import {
  BANNER_POLL_INTERVAL_MS,
  dismissBanner,
  fetchActiveBanner,
  isBannerDismissed,
  type MaintenanceBanner,
} from '../lib/banner-api'
import { usePlatformFeatures } from '../context/platform-features-context'

function severityLabel(severity: MaintenanceBanner['severity']): string {
  switch (severity) {
    case 'error':
      return 'Error'
    case 'warning':
      return 'Warning'
    default:
      return 'Information'
  }
}

function bannerTone(severity: MaintenanceBanner['severity']): string {
  switch (severity) {
    case 'error':
      return 'border-red-200 bg-red-50 text-red-950 dark:border-red-900/60 dark:bg-red-950/50 dark:text-red-100'
    case 'warning':
      return 'border-amber-200 bg-amber-50 text-amber-950 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-100'
    default:
      return 'border-sky-200 bg-sky-50 text-sky-950 dark:border-sky-900/60 dark:bg-sky-950/40 dark:text-sky-100'
  }
}

function SeverityIcon({ severity }: { severity: MaintenanceBanner['severity'] }) {
  switch (severity) {
    case 'error':
      return <XCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
    case 'warning':
      return <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
    default:
      return <Info className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
  }
}

type StatusBannerProps = {
  orgSlug?: string | null
}

export function StatusBanner({ orgSlug = null }: StatusBannerProps) {
  const { maintenanceBannerEnabled } = usePlatformFeatures()
  const [banner, setBanner] = useState<MaintenanceBanner | null>(null)
  const [hidden, setHidden] = useState(false)

  const loadBanner = useCallback(async () => {
    try {
      const next = await fetchActiveBanner(orgSlug)
      if (!next) {
        setBanner(null)
        setHidden(false)
        return
      }
      setBanner(next)
      setHidden(isBannerDismissed(next))
    } catch {
      setBanner(null)
    }
  }, [orgSlug])

  useEffect(() => {
    if (!maintenanceBannerEnabled) {
      setBanner(null)
      return
    }
    void loadBanner()
    const timer = window.setInterval(() => {
      void loadBanner()
    }, BANNER_POLL_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [loadBanner, maintenanceBannerEnabled])

  if (!maintenanceBannerEnabled || !banner || hidden) {
    return null
  }

  function onDismiss() {
    dismissBanner(banner!)
    setHidden(true)
  }

  const placementClass =
    'max-md:fixed max-md:inset-x-0 max-md:bottom-0 max-md:z-50 max-md:border-t max-md:shadow-lg md:border-b'

  return (
    <aside
      role="status"
      aria-live="polite"
      aria-atomic="true"
      className={`flex items-start gap-3 px-4 py-2 text-sm ${bannerTone(banner.severity)} ${placementClass}`}
    >
      <SeverityIcon severity={banner.severity} />
      <div className="min-w-0 flex-1">
        <p>
          <span className="sr-only">{severityLabel(banner.severity)}: </span>
          {banner.message}
          {banner.ctaUrl ? (
            <>
              {' '}
              <a
                href={banner.ctaUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium underline underline-offset-2"
              >
                {banner.ctaText?.trim() || 'Learn more'}
              </a>
            </>
          ) : null}
        </p>
      </div>
      <button
        type="button"
        onClick={onDismiss}
        className="rounded p-1 transition-[background-color,color,border-color] hover:bg-black/5 dark:hover:bg-white/10"
        aria-label="Dismiss maintenance notice"
      >
        <X className="h-4 w-4" aria-hidden />
      </button>
    </aside>
  )
}

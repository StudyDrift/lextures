import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { FeatureHelpMedia } from '../../lib/feature-help-content'

type LoadState = 'loading' | 'ready' | 'error'

export function FeatureHelpMediaRegion({
  media,
  active,
}: {
  media: FeatureHelpMedia
  /** When false, defer fetching until the help panel is open. */
  active: boolean
}) {
  const { t } = useTranslation('common')
  const [loadState, setLoadState] = useState<LoadState>('loading')
  const alt = t(media.altKey)

  useEffect(() => {
    if (!active) {
      setLoadState('loading')
      return
    }
    setLoadState('loading')
  }, [active, media.src])

  if (!active) return null

  if (loadState === 'error') return null

  return (
    <figure className="relative mb-4 aspect-video w-full overflow-hidden rounded-xl border border-slate-200 bg-slate-100 dark:border-neutral-700 dark:bg-neutral-800/80">
      {loadState === 'loading' ? (
        <div
          className="absolute inset-0 animate-pulse bg-gradient-to-br from-slate-100 via-slate-50 to-slate-100 dark:from-neutral-800 dark:via-neutral-900 dark:to-neutral-800"
          aria-hidden
        />
      ) : null}
      {/* eslint-disable-next-line jsx-a11y/media-has-caption -- silent decorative loop; alt text is the accessible equivalent */}
      <video
        className={loadState === 'ready' ? 'h-full w-full object-cover' : 'sr-only'}
        src={active ? media.src : undefined}
        muted
        loop
        playsInline
        controls
        preload="none"
        aria-label={alt}
        onLoadedData={() => setLoadState('ready')}
        onError={() => setLoadState('error')}
      />
      <figcaption className="sr-only">{alt}</figcaption>
    </figure>
  )
}
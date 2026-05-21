import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { authorizedFetch } from '../lib/api'
import { h5pI18n } from '../lib/h5p-i18n'

export type H5PPlayerProps = {
  courseCode: string
  packageId: string
  title: string
  renderUrl: string
  downloadUrl?: string
  ready: boolean
}

/** Sandboxed iframe player for H5P packages (plan 8.12). */
export function H5PPlayer({
  courseCode,
  packageId,
  title,
  renderUrl,
  downloadUrl,
  ready,
}: H5PPlayerProps) {
  const [loadError, setLoadError] = useState(false)
  const [visible, setVisible] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const labelId = useId()

  useEffect(() => {
    const el = containerRef.current
    if (!el || !ready) return
    const obs = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          setVisible(true)
          obs.disconnect()
        }
      },
      { rootMargin: '120px' },
    )
    obs.observe(el)
    return () => obs.disconnect()
  }, [ready])

  const forwardXAPI = useCallback(
    async (statement: unknown) => {
      try {
        await authorizedFetch('/api/v1/xapi/statements', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            courseCode,
            packageId,
            statement,
          }),
        })
      } catch {
        /* non-fatal */
      }
    },
    [courseCode, packageId],
  )

  useEffect(() => {
    function onMessage(ev: MessageEvent) {
      if (ev.data?.type === 'h5p-xapi' && ev.data.packageId === packageId && ev.data.statement) {
        void forwardXAPI(ev.data.statement)
      }
    }
    window.addEventListener('message', onMessage)
    return () => window.removeEventListener('message', onMessage)
  }, [forwardXAPI, packageId])

  if (!ready) {
    return (
      <div className="rounded-xl border border-amber-200/80 bg-amber-50/90 px-4 py-6 text-sm text-amber-900 dark:border-amber-500/35 dark:bg-amber-950/50 dark:text-amber-100">
        <p>{h5pI18n.loading}</p>
        {downloadUrl ? (
          <a href={downloadUrl} className="mt-2 inline-block font-medium underline">
            {h5pI18n.downloadFallback}
          </a>
        ) : null}
      </div>
    )
  }

  if (loadError) {
    return (
      <div className="rounded-xl border border-red-200/80 bg-red-50/90 px-4 py-6 text-sm text-red-900 dark:border-red-500/35 dark:bg-red-950/50 dark:text-red-100">
        <p>{h5pI18n.error}</p>
        {downloadUrl ? (
          <a href={downloadUrl} className="mt-2 inline-block font-medium underline">
            {h5pI18n.downloadFallback}
          </a>
        ) : null}
      </div>
    )
  }

  return (
    <div ref={containerRef} className="relative min-h-[320px] w-full">
      {!visible ? (
        <p className="text-sm text-slate-500 dark:text-neutral-400" aria-live="polite">
          {h5pI18n.loading}
        </p>
      ) : (
        <iframe
          title={`Interactive activity: ${title}`}
          aria-labelledby={labelId}
          src={renderUrl}
          sandbox="allow-scripts allow-same-origin"
          className="h-[min(70vh,640px)] w-full rounded-xl border border-slate-200/80 bg-white dark:border-neutral-600 dark:bg-neutral-900"
          onError={() => setLoadError(true)}
        />
      )}
      <span id={labelId} className="sr-only">
        {title}
      </span>
    </div>
  )
}

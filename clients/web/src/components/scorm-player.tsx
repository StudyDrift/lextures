import { useCallback, useEffect, useState } from 'react'
import { authorizedFetch } from '../lib/api'

export type ScormPlayerProps = {
  courseCode: string
  scoId: string
  title: string
  renderUrl: string
  downloadUrl?: string
  ready: boolean
  hasResume?: boolean
}

/** Sandboxed iframe player for SCORM packages (plan 2.14). */
export function ScormPlayer({
  title,
  renderUrl,
  downloadUrl,
  ready,
  hasResume,
}: ScormPlayerProps) {
  const [loadError, setLoadError] = useState(false)

  if (!ready) {
    return (
      <div className="rounded-xl border border-amber-200/80 bg-amber-50/90 px-4 py-6 text-sm text-amber-900 dark:border-amber-500/35 dark:bg-amber-950/50 dark:text-amber-100">
        <p>Package is still being prepared…</p>
        {downloadUrl ? (
          <a href={downloadUrl} className="mt-2 inline-block font-medium underline">
            Download package
          </a>
        ) : null}
      </div>
    )
  }

  if (loadError) {
    return (
      <div className="rounded-xl border border-red-200/80 bg-red-50/90 px-4 py-6 text-sm text-red-900 dark:border-red-500/35 dark:bg-red-950/50 dark:text-red-100">
        <p>This activity could not be loaded.</p>
        {downloadUrl ? (
          <a href={downloadUrl} className="mt-2 inline-block font-medium underline">
            Download package
          </a>
        ) : null}
      </div>
    )
  }

  return (
    <div className="relative min-h-[320px] w-full">
      {hasResume ? (
        <p className="mb-2 text-sm text-slate-600 dark:text-neutral-400" role="status">
          Resume where you left off
        </p>
      ) : null}
      <iframe
        title={`SCORM activity: ${title}`}
        src={renderUrl}
        sandbox="allow-scripts allow-same-origin"
        className="h-[min(70vh,640px)] w-full rounded-xl border border-slate-200/80 bg-white dark:border-neutral-600 dark:bg-neutral-900"
        onError={() => setLoadError(true)}
      />
    </div>
  )
}

export type ScormLaunchClientProps = {
  courseCode: string
  scoId: string
  title: string
  downloadUrl?: string
  extractStatus: string
}

/** Loads launch session then renders the SCORM player iframe. */
export function ScormLaunchClient({
  courseCode,
  scoId,
  title,
  downloadUrl,
  extractStatus,
}: ScormLaunchClientProps) {
  const [renderUrl, setRenderUrl] = useState<string | null>(null)
  const [hasResume, setHasResume] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const launch = useCallback(async () => {
    setError(null)
    try {
      const res = await authorizedFetch(
        `/api/v1/courses/${encodeURIComponent(courseCode)}/scorm/${encodeURIComponent(scoId)}/launch`,
        { method: 'POST' },
      )
      const raw = (await res.json()) as Record<string, unknown>
      if (!res.ok) {
        throw new Error(String(raw.message ?? 'Launch failed'))
      }
      const apiBase = import.meta.env.VITE_API_URL ?? ''
      const rel = String(raw.renderUrl ?? '')
      setRenderUrl(rel.startsWith('http') ? rel : `${apiBase}${rel}`)
      const cmi = raw.initialCmi as Record<string, string> | undefined
      setHasResume(
        cmi?.['cmi.core.entry'] === 'resume' ||
          Boolean(cmi?.['cmi.core.suspend_data']) ||
          Boolean(cmi?.['cmi.core.lesson_location']),
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Launch failed')
    }
  }, [courseCode, scoId])

  useEffect(() => {
    if (extractStatus === 'ready') {
      void launch()
    }
  }, [extractStatus, launch])

  if (error) {
    return (
      <p className="text-sm text-red-600 dark:text-red-400" role="alert">
        {error}
      </p>
    )
  }

  if (!renderUrl) {
    return <p className="text-sm text-slate-500 dark:text-neutral-400">Preparing activity…</p>
  }

  return (
    <ScormPlayer
      courseCode={courseCode}
      scoId={scoId}
      title={title}
      renderUrl={renderUrl}
      downloadUrl={downloadUrl}
      ready={extractStatus === 'ready'}
      hasResume={hasResume}
    />
  )
}

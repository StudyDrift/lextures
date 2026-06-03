import { useEffect, useRef, useState } from 'react'
import { Download } from 'lucide-react'
import { authorizedFetch } from '../lib/api'
import { fileContentUrlToPreviewUrl } from '../lib/course-files-api'

type OfficeHtmlPreviewProps = {
  /** Authenticated content URL; preview URL is derived by replacing `/content` with `/preview`. */
  filePath: string
  filename: string
}

export function OfficeHtmlPreview({ filePath, filename }: OfficeHtmlPreviewProps) {
  const [html, setHtml] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [iframeSrc, setIframeSrc] = useState<string | null>(null)
  const blobUrlRef = useRef<string | null>(null)

  const previewUrl = fileContentUrlToPreviewUrl(filePath)

  useEffect(() => {
    if (!previewUrl) {
      setError('Could not load preview.')
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)
    setHtml(null)
    void (async () => {
      try {
        const res = await authorizedFetch(previewUrl)
        if (!res.ok) throw new Error('Failed to load preview.')
        const text = await res.text()
        if (!cancelled) {
          setHtml(text)
          setLoading(false)
        }
      } catch {
        if (!cancelled) {
          setError('Could not render this file.')
          setLoading(false)
        }
      }
    })()
    return () => { cancelled = true }
  }, [previewUrl])

  useEffect(() => {
    if (blobUrlRef.current) {
      URL.revokeObjectURL(blobUrlRef.current)
      blobUrlRef.current = null
    }
    if (!html) {
      setIframeSrc(null)
      return
    }
    const blob = new Blob([html], { type: 'text/html;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    blobUrlRef.current = url
    setIframeSrc(url)
    return () => {
      if (blobUrlRef.current) {
        URL.revokeObjectURL(blobUrlRef.current)
        blobUrlRef.current = null
      }
    }
  }, [html])

  const handleDownload = async () => {
    try {
      const res = await authorizedFetch(filePath)
      if (!res.ok) throw new Error()
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch {
      /* noop */
    }
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center" role="status" aria-label="Loading document preview">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
      </div>
    )
  }

  if (error || !html) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-4 p-8">
        <div className="rounded-2xl bg-slate-100 p-6 dark:bg-neutral-800">
          <svg className="h-12 w-12 text-slate-400 dark:text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
        </div>
        <div className="text-center">
          <p className="text-sm font-medium text-slate-700 dark:text-neutral-300">{filename}</p>
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-500">{error ?? 'Preview unavailable.'}</p>
        </div>
        <button
          type="button"
          onClick={() => void handleDownload()}
          className="flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
        >
          <Download className="h-4 w-4" aria-hidden="true" />
          Download to view
        </button>
      </div>
    )
  }

  return (
    <iframe
      title={`Preview of ${filename}`}
      sandbox="allow-same-origin"
      src={iframeSrc ?? undefined}
      className="h-full w-full border-0 bg-white"
    />
  )
}

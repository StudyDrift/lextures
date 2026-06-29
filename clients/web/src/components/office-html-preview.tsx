import { useEffect, useMemo, useRef, useState } from 'react'
import { Download } from 'lucide-react'
import { authorizedFetch } from '../lib/api'
import { fileContentUrlToPreviewUrl } from '../lib/course-files-api'
import { AnchoredAnnotationLayer, type AnchorSurfaceProps } from './annotation/anchored-annotation-layer'

type OfficeHtmlPreviewProps = {
  /** Authenticated content URL; preview URL is derived by replacing `/content` with `/preview`. */
  filePath: string
  filename: string
  /**
   * When provided, the office HTML is rendered inline (not in an iframe) so text-anchor
   * highlights can be drawn over it. The HTML is already sanitized server-side (bluemonday).
   */
  annotation?: AnchorSurfaceProps
}

const SCOPE_CLASS = 'office-inline-scope'

// Scope the server's preview stylesheet to the inline container so it can't leak into the app.
// The office preview CSS is a flat ruleset (no @media / nesting), so a per-selector prefix is
// sufficient; `body`/`:root` rules are retargeted at the scope element itself.
function scopeOfficeCss(css: string): string {
  return css.replace(/([^{}]+)\{([^}]*)\}/g, (_m, rawSelectors: string, decls: string) => {
    const scoped = rawSelectors
      .split(',')
      .map((sel) => {
        const s = sel.trim()
        if (!s) return ''
        if (s === 'body' || s === 'html' || s === ':root') return `.${SCOPE_CLASS}`
        return `.${SCOPE_CLASS} ${s}`
      })
      .filter(Boolean)
      .join(', ')
    return `${scoped} { ${decls.trim()} }`
  })
}

/** Parse the server preview document into a scoped <style> + body markup for inline rendering. */
function parseInlineOfficeHtml(html: string): { css: string; body: string } {
  const doc = new DOMParser().parseFromString(html, 'text/html')
  const css = Array.from(doc.querySelectorAll('style'))
    .map((s) => s.textContent ?? '')
    .join('\n')
  return { css: scopeOfficeCss(css), body: doc.body?.innerHTML ?? '' }
}

export function OfficeHtmlPreview({ filePath, filename, annotation }: OfficeHtmlPreviewProps) {
  const [html, setHtml] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [iframeSrc, setIframeSrc] = useState<string | null>(null)
  const blobUrlRef = useRef<string | null>(null)
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const contentRef = useRef<HTMLDivElement | null>(null)

  const previewUrl = fileContentUrlToPreviewUrl(filePath)

  const inline = useMemo(
    () => (html && annotation ? parseInlineOfficeHtml(html) : null),
    [html, annotation],
  )

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

  // Inline (annotatable) render: the office body is placed in the light DOM with a scoped
  // stylesheet so the anchor layer can walk its text nodes and paint highlights over it.
  if (annotation && inline) {
    return (
      <div
        ref={scrollRef}
        className={`${SCOPE_CLASS} relative h-full overflow-auto bg-white`}
      >
        {/* eslint-disable-next-line react/no-danger -- server-sanitized office preview HTML */}
        <style dangerouslySetInnerHTML={{ __html: inline.css }} />
        {/* eslint-disable-next-line react/no-danger -- server-sanitized office preview HTML */}
        <div ref={contentRef} dangerouslySetInnerHTML={{ __html: inline.body }} />
        <AnchoredAnnotationLayer
          scrollRef={scrollRef}
          contentRef={contentRef}
          annotations={annotation.annotations}
          readOnly={annotation.readOnly}
          selectedId={annotation.selectedAnnotationId}
          onSelectAnnotation={annotation.onSelectAnnotation}
          onAnchorComplete={annotation.onAnchorComplete}
          recomputeKey={`office:${inline.body.length}`}
        />
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

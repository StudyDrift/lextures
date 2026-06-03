import { useEffect, useId, useMemo, useState, type ReactNode } from 'react'
import { Download } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { createThemedMarkdownComponents } from './markdown/markdown-themed-components'
import { authorizedFetch } from '../lib/api'
import { isMarkdownFilename } from '../lib/file-type'
import { resolveMarkdownTheme } from '../lib/markdown-theme'
import { useLmsDarkMode } from '../hooks/use-lms-dark-mode'

/** Max bytes to load into the preview (avoids huge files in memory). */
const MAX_TEXT_PREVIEW_BYTES = 2 * 1024 * 1024

type TextFileTab = 'source' | 'preview'

type TextFilePreviewProps = {
  filePath: string
  filename: string
  mimeType?: string | null
}

export function TextFilePreview({ filePath, filename, mimeType }: TextFilePreviewProps) {
  const markdownFile = isMarkdownFilename(filename, mimeType)
  const lmsUiDark = useLmsDarkMode()
  const mdTheme = useMemo(
    () => resolveMarkdownTheme('classic', null, { lmsUiDark }),
    [lmsUiDark],
  )
  const mdComponents = useMemo(
    () => createThemedMarkdownComponents(mdTheme),
    [mdTheme],
  )
  const tablistId = useId()
  const [tab, setTab] = useState<TextFileTab>(markdownFile ? 'preview' : 'source')
  const [content, setContent] = useState<string | null>(null)
  const [truncated, setTruncated] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    setContent(null)
    setTruncated(false)

    void (async () => {
      try {
        const res = await authorizedFetch(filePath)
        if (!res.ok) throw new Error('Failed to load file.')
        const lengthHeader = res.headers.get('Content-Length')
        if (lengthHeader) {
          const len = Number.parseInt(lengthHeader, 10)
          if (Number.isFinite(len) && len > MAX_TEXT_PREVIEW_BYTES) {
            if (!cancelled) {
              setError(`This file is too large to preview (${formatBytes(len)}). Download it to view.`)
              setLoading(false)
            }
            return
          }
        }
        const text = await res.text()
        if (cancelled) return
        if (text.length > MAX_TEXT_PREVIEW_BYTES) {
          setContent(text.slice(0, MAX_TEXT_PREVIEW_BYTES))
          setTruncated(true)
        } else {
          setContent(text)
        }
        setLoading(false)
      } catch {
        if (!cancelled) {
          setError('Could not load this file.')
          setLoading(false)
        }
      }
    })()

    return () => { cancelled = true }
  }, [filePath])

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
      <div className="flex h-full items-center justify-center" role="status" aria-label="Loading text preview">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-4 p-8">
        <p className="text-center text-sm text-slate-600 dark:text-neutral-400" role="alert">{error}</p>
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
    <div className="flex h-full flex-col overflow-hidden bg-slate-50 dark:bg-neutral-950">
      {markdownFile && (
        <div
          id={tablistId}
          role="tablist"
          aria-label="Markdown preview mode"
          className="flex shrink-0 gap-1 border-b border-slate-200 bg-white px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
        >
          <TabButton
            active={tab === 'preview'}
            controls="markdown-preview-panel"
            onClick={() => setTab('preview')}
          >
            Preview
          </TabButton>
          <TabButton
            active={tab === 'source'}
            controls="markdown-source-panel"
            onClick={() => setTab('source')}
          >
            Source
          </TabButton>
        </div>
      )}

      {truncated && (
        <p className="shrink-0 border-b border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">
          Preview truncated to the first {formatBytes(MAX_TEXT_PREVIEW_BYTES)}. Download for the full file.
        </p>
      )}

      {tab === 'preview' && markdownFile ? (
        <div
          id="markdown-preview-panel"
          role="tabpanel"
          className={`lms-scope syllabus-md markdown-file-preview min-h-0 flex-1 overflow-auto px-6 py-4 ${mdTheme.classes.article}`}
        >
          <ReactMarkdown remarkPlugins={[remarkGfm]} components={mdComponents}>
            {content ?? ''}
          </ReactMarkdown>
        </div>
      ) : (
        <pre
          id="markdown-source-panel"
          role={markdownFile ? 'tabpanel' : undefined}
          aria-label={`Text preview of ${filename}`}
          className="min-h-0 flex-1 overflow-auto whitespace-pre-wrap break-words p-4 font-mono text-sm leading-relaxed text-slate-800 dark:text-neutral-200"
        >
          {content ?? ''}
        </pre>
      )}
    </div>
  )
}

function TabButton({
  active,
  children,
  controls,
  onClick,
}: {
  active: boolean
  children: ReactNode
  controls: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      aria-controls={controls}
      onClick={onClick}
      className={
        active
          ? 'rounded-lg bg-indigo-50 px-3 py-1.5 text-sm font-semibold text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300'
          : 'rounded-lg px-3 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800'
      }
    >
      {children}
    </button>
  )
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

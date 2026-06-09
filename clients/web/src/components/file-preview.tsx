import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { Download, Music, X, ZoomIn, ZoomOut, RotateCcw } from 'lucide-react'
import { detectPreviewType } from '../lib/file-type'
import { authorizedFetch } from '../lib/api'
import { apiUrl } from '../lib/api'
import { CodeFilePreview } from './code-file-preview'
import { FilePreviewFallback } from './file-preview-fallback'
import { OfficeHtmlPreview } from './office-html-preview'
import { PdfViewer } from './pdf-viewer'
import { TextFilePreview } from './text-file-preview'

export type FilePreviewProps = {
  open: boolean
  filePath: string
  filename: string
  mimeType: string | null
  onClose: () => void
}

// ── Image Lightbox ───────────────────────────────────────────────────────────

type ImageViewerProps = {
  filePath: string
  filename: string
}

function ImageViewer({ filePath, filename }: ImageViewerProps) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [zoom, setZoom] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [dragging, setDragging] = useState(false)
  const dragStart = useRef<{ x: number; y: number; panX: number; panY: number } | null>(null)
  const imgRef = useRef<HTMLImageElement | null>(null)

  useEffect(() => {
    let url: string | null = null
    let cancelled = false

    async function load() {
      try {
        const res = await authorizedFetch(filePath)
        if (!res.ok) throw new Error('Failed to load image.')
        const blob = await res.blob()
        if (cancelled) return
        url = URL.createObjectURL(blob)
        setBlobUrl(url)
      } catch {
        if (!cancelled) setError('Could not load the image.')
      }
    }

    void load()
    return () => {
      cancelled = true
      if (url) URL.revokeObjectURL(url)
    }
  }, [filePath])

  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    let deltaY = e.deltaY
    if (e.deltaMode === 1) deltaY *= 16
    else if (e.deltaMode === 2) deltaY *= 400
    setZoom((z) => {
      const delta = -deltaY * 0.00035
      return Math.max(0.1, Math.min(5, z + delta))
    })
  }, [])

  const handlePointerDown = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    e.currentTarget.setPointerCapture(e.pointerId)
    dragStart.current = { x: e.clientX, y: e.clientY, panX: pan.x, panY: pan.y }
    setDragging(true)
  }, [pan])

  const handlePointerMove = useCallback((e: React.PointerEvent) => {
    if (!dragStart.current) return
    const dx = e.clientX - dragStart.current.x
    const dy = e.clientY - dragStart.current.y
    setPan({ x: dragStart.current.panX + dx, y: dragStart.current.panY + dy })
  }, [])

  const handlePointerUp = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    try { e.currentTarget.releasePointerCapture(e.pointerId) } catch { /* ignore */ }
    dragStart.current = null
    setDragging(false)
  }, [])

  const reset = () => { setZoom(1); setPan({ x: 0, y: 0 }) }

  const handleDownload = () => {
    const a = document.createElement('a')
    a.href = blobUrl ?? apiUrl(filePath)
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }

  if (error) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 p-6">
        <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">{error}</p>
        <button
          type="button"
          onClick={handleDownload}
          className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
        >
          Download instead
        </button>
      </div>
    )
  }

  if (!blobUrl) {
    return (
      <div className="flex h-full items-center justify-center" role="status" aria-label="Loading image">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Image toolbar */}
      <div
        className="flex shrink-0 items-center gap-1 border-b border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-700 dark:bg-neutral-900"
        role="toolbar"
        aria-label="Image viewer controls"
      >
        <button
          type="button"
          onClick={() => setZoom((z) => Math.max(0.1, Math.round((z - 0.15) * 100) / 100))}
          className="rounded-lg p-1.5 text-slate-600 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
          aria-label="Zoom out"
        >
          <ZoomOut className="h-4 w-4" />
        </button>
        <span className="min-w-[3rem] text-center text-xs text-slate-500 dark:text-neutral-500" aria-live="polite">
          {Math.round(zoom * 100)}%
        </span>
        <button
          type="button"
          onClick={() => setZoom((z) => Math.min(5, Math.round((z + 0.15) * 100) / 100))}
          className="rounded-lg p-1.5 text-slate-600 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
          aria-label="Zoom in"
        >
          <ZoomIn className="h-4 w-4" />
        </button>
        <button
          type="button"
          onClick={reset}
          className="rounded-lg p-1.5 text-slate-600 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
          aria-label="Reset zoom and pan"
        >
          <RotateCcw className="h-4 w-4" />
        </button>
        <div className="flex-1" />
        <button
          type="button"
          onClick={handleDownload}
          className="flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-500 dark:hover:bg-neutral-800 dark:hover:text-neutral-300"
          aria-label={`Download ${filename}`}
        >
          <Download className="h-3.5 w-3.5" />
          Download
        </button>
      </div>

      {/* Pan/zoom area */}
      <div
        className="flex-1 overflow-hidden bg-neutral-900 dark:bg-neutral-950"
        onWheel={handleWheel}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
        style={{ cursor: dragging ? 'grabbing' : 'grab' }}
      >
        <div
          className="flex h-full items-center justify-center"
          style={{
            transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
            transformOrigin: 'center center',
            transition: dragging ? 'none' : 'transform 0.1s ease',
          }}
        >
          <img
            ref={imgRef}
            src={blobUrl}
            alt={filename}
            className="max-h-[85vh] max-w-full select-none object-contain shadow-2xl"
            draggable={false}
          />
        </div>
      </div>
    </div>
  )
}

// ── Unsupported file fallback ────────────────────────────────────────────────

function UnsupportedFileView({ filePath, filename }: { filePath: string; filename: string }) {
  return (
    <FilePreviewFallback
      filePath={filePath}
      filename={filename}
      message="This file type cannot be previewed in the browser."
      downloadLabel="Download to view"
    />
  )
}

function VideoFileViewer({ filePath, filename }: { filePath: string; filename: string }) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let url: string | null = null
    let cancelled = false
    void (async () => {
      try {
        const res = await authorizedFetch(filePath)
        if (!res.ok) throw new Error('Failed to load video.')
        const blob = await res.blob()
        if (cancelled) return
        url = URL.createObjectURL(blob)
        setBlobUrl(url)
      } catch {
        if (!cancelled) setError('Could not load the video.')
      }
    })()
    return () => {
      cancelled = true
      if (url) URL.revokeObjectURL(url)
    }
  }, [filePath])

  if (error) {
    return (
      <p className="p-6 text-sm text-rose-700 dark:text-rose-200" role="alert">
        {error}
      </p>
    )
  }
  if (!blobUrl) {
    return (
      <div className="flex h-full items-center justify-center" role="status">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col p-4">
      {/* eslint-disable-next-line jsx-a11y/media-has-caption -- course-file preview; attach captions when storage object is linked */}
      <video
        className="max-h-full w-full rounded-lg bg-black"
        controls
        playsInline
        src={blobUrl}
        aria-label={filename}
      />
    </div>
  )
}

// ── Audio player ─────────────────────────────────────────────────────────────

function AudioViewer({ filePath, filename }: { filePath: string; filename: string }) {
  const [src, setSrc] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const blobUrlRef = useRef<string | null>(null)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await authorizedFetch(filePath)
        if (!res.ok) throw new Error('Failed to load audio.')
        const apiOrigin = new URL(apiUrl(filePath)).origin
        if (new URL(res.url).origin !== apiOrigin) {
          // S3 presigned URL — use directly so the browser can stream
          if (!cancelled) setSrc(res.url)
          return
        }
        const blob = await res.blob()
        if (cancelled) return
        const url = URL.createObjectURL(blob)
        blobUrlRef.current = url
        setSrc(url)
      } catch {
        if (!cancelled) setError('Could not load the audio file.')
      }
    })()
    return () => {
      cancelled = true
      if (blobUrlRef.current) URL.revokeObjectURL(blobUrlRef.current)
    }
  }, [filePath])

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">{error}</p>
      </div>
    )
  }
  if (!src) {
    return (
      <div className="flex h-full items-center justify-center" role="status">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col items-center justify-center gap-6 bg-neutral-50 p-8 dark:bg-neutral-950">
      <div className="rounded-2xl bg-indigo-100 p-8 dark:bg-indigo-950">
        <Music className="h-12 w-12 text-indigo-600 dark:text-indigo-400" aria-hidden />
      </div>
      <p className="text-sm font-medium text-slate-700 dark:text-neutral-300">{filename}</p>
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <audio controls src={src} className="w-full max-w-md" aria-label={filename} />
    </div>
  )
}

// ── Shared preview body (course files + submission attachments) ───────────────

export type FilePreviewBodyProps = {
  filePath: string
  filename: string
  mimeType: string | null | undefined
  /** When `message-only`, load errors omit sidebar download UI (e.g. submission modal). */
  errorVariant?: 'standalone' | 'message-only'
  className?: string
}

export function FilePreviewBody({
  filePath,
  filename,
  mimeType,
  errorVariant = 'standalone',
  className,
}: FilePreviewBodyProps) {
  const previewType = detectPreviewType(mimeType, filename)

  return (
    <div className={className ?? 'h-full min-h-0'}>
      {previewType === 'pdf' && (
        <PdfViewer filePath={filePath} filename={filename} />
      )}
      {previewType === 'image' && (
        <ImageViewer filePath={filePath} filename={filename} />
      )}
      {previewType === 'video' && (
        <VideoFileViewer filePath={filePath} filename={filename} />
      )}
      {previewType === 'audio' && (
        <AudioViewer filePath={filePath} filename={filename} />
      )}
      {previewType === 'office' && (
        <OfficeHtmlPreview filePath={filePath} filename={filename} />
      )}
      {previewType === 'text' && (
        <TextFilePreview
          filePath={filePath}
          filename={filename}
          mimeType={mimeType}
          errorVariant={errorVariant}
        />
      )}
      {previewType === 'code' && (
        <CodeFilePreview filePath={filePath} filename={filename} errorVariant={errorVariant} />
      )}
      {previewType === 'none' && (
        <UnsupportedFileView filePath={filePath} filename={filename} />
      )}
    </div>
  )
}

// ── FilePreview modal ────────────────────────────────────────────────────────

export function FilePreview({ open, filePath, filename, mimeType, onClose }: FilePreviewProps) {
  const titleId = useId()
  const closeRef = useRef<HTMLButtonElement>(null)

  // Focus close button on open
  useEffect(() => {
    if (!open) return
    const t = window.setTimeout(() => closeRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [open])

  // Escape key closes
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const title = filename

  return (
    <div
      className="fixed inset-0 z-[500] flex items-stretch justify-center p-0 md:items-center md:p-2"
      role="presentation"
    >
      {/* Backdrop */}
      <button
        type="button"
        aria-label="Close file preview backdrop"
        className="absolute inset-0 cursor-default border-0 bg-black/60 p-0"
        onClick={onClose}
        tabIndex={-1}
      />

      {/* Dialog */}
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="relative z-10 flex w-full flex-col overflow-hidden rounded-none bg-white shadow-2xl dark:bg-neutral-900 md:rounded-xl"
        style={{ width: 'min(95vw, 1440px)', height: 'min(95vh, 1080px)', maxHeight: '100dvh' }}
      >
        {/* Dialog header */}
        <div className="flex shrink-0 items-center gap-2 border-b border-slate-200 bg-white px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900">
          <h2
            id={titleId}
            className="flex-1 truncate text-sm font-semibold text-slate-800 dark:text-neutral-100"
            title={title}
          >
            {title}
          </h2>
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
            aria-label="Close preview"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="min-h-0 flex-1">
          <FilePreviewBody filePath={filePath} filename={filename} mimeType={mimeType} />
        </div>
      </div>
    </div>
  )
}

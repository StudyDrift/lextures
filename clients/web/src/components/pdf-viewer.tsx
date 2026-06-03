import { useCallback, useEffect, useRef, useState } from 'react'
import { getDocument, GlobalWorkerOptions, TextLayer } from 'pdfjs-dist'
import type { PDFDocumentProxy } from 'pdfjs-dist'
import workerSrc from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import { authorizedFetch } from '../lib/api'
import { ChevronLeft, ChevronRight, ZoomIn, ZoomOut, Download } from 'lucide-react'

GlobalWorkerOptions.workerSrc = workerSrc

let _textLayerStyleInjected = false

function ensureTextLayerStyles() {
  if (_textLayerStyleInjected || typeof document === 'undefined') return
  if (document.getElementById('pdf-text-layer-css')) {
    _textLayerStyleInjected = true
    return
  }
  const el = document.createElement('style')
  el.id = 'pdf-text-layer-css'
  // Minimal styles for the pdfjs TextLayer class — spans are positioned absolutely
  // using the --scale-factor CSS variable set per-container.
  el.textContent = [
    '.pdf-tl{position:absolute;inset:0;overflow:hidden;line-height:1;text-align:initial;}',
    '.pdf-tl span,.pdf-tl br{color:transparent;position:absolute;white-space:pre;cursor:text;transform-origin:0% 0%;}',
    '.pdf-tl ::selection{background:rgba(99,102,241,.35);color:transparent;}',
  ].join('')
  document.head.appendChild(el)
  _textLayerStyleInjected = true
}

const MIN_SCALE = 0.5
const MAX_SCALE = 3.0
const SCALE_STEP = 0.25

export type PdfViewerProps = {
  filePath: string
  filename: string
}

type PageDim = { w: number; h: number }

export function PdfViewer({ filePath, filename }: PdfViewerProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pageDims, setPageDims] = useState<PageDim[]>([])
  const [numPages, setNumPages] = useState(0)
  const [scale, setScale] = useState(1.0)
  const [currentPage, setCurrentPage] = useState(1)
  const [pageInput, setPageInput] = useState('1')

  const pdfRef = useRef<PDFDocumentProxy | null>(null)
  const pageObjsRef = useRef<Awaited<ReturnType<PDFDocumentProxy['getPage']>>[]>([])
  const canvasRefs = useRef<(HTMLCanvasElement | null)[]>([])
  const textRefs = useRef<(HTMLDivElement | null)[]>([])
  const pageContainerRefs = useRef<(HTMLDivElement | null)[]>([])
  const containerRef = useRef<HTMLDivElement | null>(null)
  const renderedPagesRef = useRef<Set<number>>(new Set())
  const renderingPagesRef = useRef<Set<number>>(new Set())
  const observerRef = useRef<IntersectionObserver | null>(null)
  const scaleRef = useRef(scale)

  // Keep scaleRef in sync so async renderPage callbacks always see the latest scale.
  useEffect(() => { scaleRef.current = scale }, [scale])

  useEffect(() => { ensureTextLayerStyles() }, [])

  // Load PDF document and page dimensions
  useEffect(() => {
    let cancelled = false
    let loadedPdf: PDFDocumentProxy | null = null

    async function load() {
      // Reset all state at the start of each new load
      setLoading(true)
      setError(null)
      setPageDims([])
      setNumPages(0)
      setCurrentPage(1)
      setPageInput('1')
      renderedPagesRef.current = new Set()
      renderingPagesRef.current = new Set()
      pdfRef.current = null
      pageObjsRef.current = []

      try {
        const res = await authorizedFetch(filePath)
        if (!res.ok) throw new Error('Failed to load PDF.')
        const data = new Uint8Array(await res.arrayBuffer())
        const doc = await getDocument({ data }).promise

        if (cancelled) {
          await doc.destroy().catch(() => {})
          return
        }

        loadedPdf = doc
        pdfRef.current = doc

        const dims: PageDim[] = []
        const pages: Awaited<ReturnType<PDFDocumentProxy['getPage']>>[] = []
        for (let i = 1; i <= doc.numPages; i++) {
          const page = await doc.getPage(i)
          const vp = page.getViewport({ scale: 1 })
          dims.push({ w: vp.width, h: vp.height })
          pages.push(page)
        }

        if (!cancelled) {
          pageObjsRef.current = pages
          setPageDims(dims)
          setNumPages(doc.numPages)
          setLoading(false)
        }
      } catch {
        if (!cancelled) {
          setError('Could not load the PDF.')
          setLoading(false)
        }
      }
    }

    void load()
    return () => {
      cancelled = true
      if (loadedPdf) void loadedPdf.destroy().catch(() => {})
    }
  }, [filePath])

  const renderPage = useCallback(async (idx: number) => {
    if (renderingPagesRef.current.has(idx)) return
    const page = pageObjsRef.current[idx]
    const canvas = canvasRefs.current[idx]
    if (!page || !canvas) return

    renderingPagesRef.current.add(idx)

    const s = scaleRef.current
    const vp = page.getViewport({ scale: s })
    canvas.width = vp.width
    canvas.height = vp.height

    const ctx = canvas.getContext('2d')
    if (ctx) {
      await page.render({ canvasContext: ctx, viewport: vp }).promise
    }

    // Render text layer for copy/paste and browser find (Ctrl+F)
    const textDiv = textRefs.current[idx]
    if (textDiv) {
      textDiv.innerHTML = ''
      textDiv.className = 'pdf-tl'
      // --scale-factor is used by pdfjs TextLayer for font-size calculations
      textDiv.style.setProperty('--scale-factor', String(s))
      try {
        const textContent = await page.getTextContent()
        const layer = new TextLayer({ textContentSource: textContent, container: textDiv, viewport: vp })
        await layer.render()
      } catch {
        // text layer is best-effort; canvas rendering still works
      }
    }

    renderedPagesRef.current.add(idx)
    renderingPagesRef.current.delete(idx)
  }, [])

  // Set up IntersectionObserver after pages load
  useEffect(() => {
    if (!pageDims.length) return

    observerRef.current?.disconnect()
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue
          const idx = parseInt(entry.target.getAttribute('data-page-idx') ?? '-1', 10)
          if (idx >= 0 && !renderedPagesRef.current.has(idx)) {
            void renderPage(idx)
          }
        }
      },
      { threshold: 0.05, root: containerRef.current },
    )
    observerRef.current = observer

    // Observe after a tick to let React mount the refs
    const t = window.setTimeout(() => {
      pageContainerRefs.current.forEach((div) => {
        if (div) observer.observe(div)
      })
    }, 0)

    return () => {
      window.clearTimeout(t)
      observer.disconnect()
    }
  }, [pageDims, renderPage])

  // Re-render visible pages on scale change
  useEffect(() => {
    if (!pageDims.length) return
    renderedPagesRef.current = new Set()
    renderingPagesRef.current = new Set()
    pageContainerRefs.current.forEach((div, idx) => {
      if (!div) return
      const rect = div.getBoundingClientRect()
      const containerRect = containerRef.current?.getBoundingClientRect()
      const top = containerRect ? containerRect.top : 0
      const bottom = containerRect ? containerRect.bottom : window.innerHeight
      if (rect.top < bottom && rect.bottom > top) {
        void renderPage(idx)
      }
    })
  }, [scale, pageDims, renderPage])

  const fitWidth = useCallback(() => {
    if (!containerRef.current || !pageDims.length) return
    const w = containerRef.current.clientWidth - 40
    const pdfW = pageDims[0]?.w ?? 600
    const newScale = Math.max(MIN_SCALE, Math.min(MAX_SCALE, w / pdfW))
    setScale(Math.round(newScale * 100) / 100)
  }, [pageDims])

  const goToPage = useCallback(
    (n: number) => {
      const p = Math.max(1, Math.min(numPages, n))
      setCurrentPage(p)
      setPageInput(String(p))
      pageContainerRefs.current[p - 1]?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    },
    [numPages],
  )

  const handleDownload = useCallback(async () => {
    try {
      const res = await authorizedFetch(filePath)
      if (!res.ok) return
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch { /* noop */ }
  }, [filePath, filename])

  const commitPageInput = useCallback(() => {
    const n = parseInt(pageInput, 10)
    if (Number.isFinite(n)) goToPage(n)
    else setPageInput(String(currentPage))
  }, [pageInput, currentPage, goToPage])

  if (loading) {
    return (
      <div className="flex h-full min-h-[24rem] items-center justify-center" role="status" aria-label="Loading PDF">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full min-h-[16rem] flex-col items-center justify-center gap-4 p-6">
        <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
        <button
          type="button"
          onClick={() => void handleDownload()}
          className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
        >
          Download instead
        </button>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Toolbar */}
      <div
        className="flex shrink-0 items-center gap-1 border-b border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-700 dark:bg-neutral-900"
        role="toolbar"
        aria-label="PDF viewer controls"
      >
        <button
          type="button"
          disabled={currentPage <= 1}
          onClick={() => goToPage(currentPage - 1)}
          className="rounded-lg p-1.5 text-slate-600 hover:bg-slate-100 disabled:opacity-40 dark:text-neutral-400 dark:hover:bg-neutral-800"
          aria-label="Previous page"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>

        <div className="flex items-center gap-1">
          <input
            type="text"
            inputMode="numeric"
            value={pageInput}
            onChange={(e) => setPageInput(e.target.value)}
            onBlur={commitPageInput}
            onKeyDown={(e) => e.key === 'Enter' && commitPageInput()}
            aria-label="Current page number"
            className="w-10 rounded-lg border border-slate-200 bg-white px-1 py-0.5 text-center text-sm text-slate-700 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
          />
          <span className="text-sm text-slate-500 dark:text-neutral-500">/ {numPages}</span>
        </div>

        <button
          type="button"
          disabled={currentPage >= numPages}
          onClick={() => goToPage(currentPage + 1)}
          className="rounded-lg p-1.5 text-slate-600 hover:bg-slate-100 disabled:opacity-40 dark:text-neutral-400 dark:hover:bg-neutral-800"
          aria-label="Next page"
        >
          <ChevronRight className="h-4 w-4" />
        </button>

        <div className="mx-1 h-4 w-px bg-slate-200 dark:bg-neutral-700" aria-hidden="true" />

        <button
          type="button"
          disabled={scale <= MIN_SCALE}
          onClick={() => setScale((s) => Math.max(MIN_SCALE, Math.round((s - SCALE_STEP) * 100) / 100))}
          className="rounded-lg p-1.5 text-slate-600 hover:bg-slate-100 disabled:opacity-40 dark:text-neutral-400 dark:hover:bg-neutral-800"
          aria-label="Zoom out"
        >
          <ZoomOut className="h-4 w-4" />
        </button>

        <span className="min-w-[3rem] text-center text-xs text-slate-500 dark:text-neutral-500" aria-live="polite" aria-atomic="true">
          {Math.round(scale * 100)}%
        </span>

        <button
          type="button"
          disabled={scale >= MAX_SCALE}
          onClick={() => setScale((s) => Math.min(MAX_SCALE, Math.round((s + SCALE_STEP) * 100) / 100))}
          className="rounded-lg p-1.5 text-slate-600 hover:bg-slate-100 disabled:opacity-40 dark:text-neutral-400 dark:hover:bg-neutral-800"
          aria-label="Zoom in"
        >
          <ZoomIn className="h-4 w-4" />
        </button>

        <button
          type="button"
          onClick={fitWidth}
          className="rounded-lg px-2 py-1 text-xs text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-500 dark:hover:bg-neutral-800 dark:hover:text-neutral-300"
          aria-label="Fit to width"
        >
          Fit
        </button>

        <div className="flex-1" />

        <button
          type="button"
          onClick={() => void handleDownload()}
          className="flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-500 dark:hover:bg-neutral-800 dark:hover:text-neutral-300"
          aria-label={`Download ${filename}`}
        >
          <Download className="h-3.5 w-3.5" />
          Download
        </button>
      </div>

      {/* Scrollable page container */}
      <div
        ref={containerRef}
        className="flex-1 overflow-auto bg-slate-200 p-4 dark:bg-neutral-950"
      >
        <div className="flex flex-col items-center gap-3" role="document" aria-label={`${filename} — ${numPages} pages`}>
          {pageDims.map((dim, idx) => (
            <div
              key={idx}
              ref={(el) => { pageContainerRefs.current[idx] = el }}
              data-page-idx={String(idx)}
              data-testid={`pdf-page-${idx + 1}`}
              className="relative shadow-lg"
              style={{ width: dim.w * scale, height: dim.h * scale }}
            >
              <canvas
                ref={(el) => { canvasRefs.current[idx] = el }}
                className="block bg-white"
                width={dim.w * scale}
                height={dim.h * scale}
                aria-label={`Page ${idx + 1} of ${numPages}`}
              />
              <div
                ref={(el) => { textRefs.current[idx] = el }}
                aria-hidden="true"
                style={{ position: 'absolute', inset: 0, width: dim.w * scale, height: dim.h * scale }}
              />
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

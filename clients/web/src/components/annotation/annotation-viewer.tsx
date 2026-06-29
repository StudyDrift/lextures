import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { GlobalWorkerOptions, getDocument, TextLayer } from 'pdfjs-dist'
import type { PDFDocumentProxy } from 'pdfjs-dist'
import workerSrc from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import { apiUrl, authorizedFetch } from '../../lib/api'
import { getAccessToken } from '../../lib/auth'
import type { SubmissionAnnotationApi } from '../../lib/courses-api'
import { FilePreviewFallback } from '../file-preview-fallback'
import { ensureTextLayerStyles } from '../../lib/pdf-text-layer'
import type { AnnotationTool } from './annotation-toolbar'

GlobalWorkerOptions.workerSrc = workerSrc

type NormPoint = { x: number; y: number }
type NormRect = { x1: number; y1: number; x2: number; y2: number }

export type AnnotationViewerProps = {
  filePath: string | null
  mimeType: string | null
  filename?: string | null
  readOnly: boolean
  fallbackVariant?: 'standalone' | 'message-only'
  tool: AnnotationTool
  colour: string
  annotations: SubmissionAnnotationApi[]
  selectedId?: string | null
  onSelectAnnotation?: (id: string) => void
  /** Highlight is reported as one or more normalized rectangles (text selection quads). */
  onHighlightComplete?: (page: number, rects: NormRect[]) => void
  onDrawComplete?: (page: number, points: NormPoint[]) => void
  onPinComplete?: (page: number, pt: NormPoint) => void
  onTextBoxComplete?: (page: number, rect: NormRect) => void
}

function normFromClient(rect: DOMRect, clientX: number, clientY: number): NormPoint {
  const nx = (clientX - rect.left) / rect.width
  const ny = (clientY - rect.top) / rect.height
  return {
    x: Math.min(1, Math.max(0, nx)),
    y: Math.min(1, Math.max(0, ny)),
  }
}

function finiteRect(o: Record<string, unknown>): NormRect | null {
  const x1 = typeof o.x1 === 'number' ? o.x1 : Number(o.x1)
  const y1 = typeof o.y1 === 'number' ? o.y1 : Number(o.y1)
  const x2 = typeof o.x2 === 'number' ? o.x2 : Number(o.x2)
  const y2 = typeof o.y2 === 'number' ? o.y2 : Number(o.y2)
  if (![x1, y1, x2, y2].every((n) => Number.isFinite(n))) return null
  return { x1, y1, x2, y2 }
}

function parseRect(c: unknown): NormRect | null {
  if (!c || typeof c !== 'object') return null
  return finiteRect(c as Record<string, unknown>)
}

// Highlights may be stored as a single legacy rect ({x1,y1,x2,y2}) or as text-selection
// quads ({ rects: [...] }). Return every rectangle to draw.
function parseHighlightRects(c: unknown): NormRect[] {
  if (!c || typeof c !== 'object') return []
  const o = c as Record<string, unknown>
  if (Array.isArray(o.rects)) {
    const out: NormRect[] = []
    for (const r of o.rects) {
      if (r && typeof r === 'object') {
        const parsed = finiteRect(r as Record<string, unknown>)
        if (parsed) out.push(parsed)
      }
    }
    return out
  }
  const single = finiteRect(o)
  return single ? [single] : []
}

function parsePoints(c: unknown): NormPoint[] {
  if (!c || typeof c !== 'object') return []
  const o = c as { points?: unknown }
  if (!Array.isArray(o.points)) return []
  const out: NormPoint[] = []
  for (const p of o.points) {
    if (!p || typeof p !== 'object') continue
    const r = p as Record<string, unknown>
    const x = typeof r.x === 'number' ? r.x : Number(r.x)
    const y = typeof r.y === 'number' ? r.y : Number(r.y)
    if (Number.isFinite(x) && Number.isFinite(y)) out.push({ x, y })
  }
  return out
}

function parsePin(c: unknown): NormPoint | null {
  if (!c || typeof c !== 'object') return null
  const o = c as Record<string, unknown>
  const x = typeof o.x === 'number' ? o.x : Number(o.x)
  const y = typeof o.y === 'number' ? o.y : Number(o.y)
  if (!Number.isFinite(x) || !Number.isFinite(y)) return null
  return { x, y }
}

// Collect the bounding rectangles of the current text selection that fall on a given page
// box, normalized to that box. Used to turn a text drag-select into highlight quads.
function selectionRectsForBox(box: DOMRect): NormRect[] {
  const sel = window.getSelection()
  if (!sel || sel.isCollapsed || sel.rangeCount === 0) return []
  const out: NormRect[] = []
  for (let i = 0; i < sel.rangeCount; i += 1) {
    const range = sel.getRangeAt(i)
    const rects = range.getClientRects()
    for (let j = 0; j < rects.length; j += 1) {
      const rc = rects[j]
      if (!rc || rc.width < 1 || rc.height < 1) continue
      const cx = rc.left + rc.width / 2
      const cy = rc.top + rc.height / 2
      if (cx < box.left || cx > box.right || cy < box.top || cy > box.bottom) continue
      const x1 = Math.max(0, (rc.left - box.left) / box.width)
      const y1 = Math.max(0, (rc.top - box.top) / box.height)
      const x2 = Math.min(1, (rc.right - box.left) / box.width)
      const y2 = Math.min(1, (rc.bottom - box.top) / box.height)
      if (x2 - x1 > 0.001 && y2 - y1 > 0.001) out.push({ x1, y1, x2, y2 })
    }
  }
  return out
}

function PageOverlay({
  page,
  width,
  height,
  annotations,
  readOnly,
  tool,
  colour,
  selectedId,
  textSelectHighlight,
  onSelectAnnotation,
  onHighlightComplete,
  onDrawComplete,
  onPinComplete,
  onTextBoxComplete,
}: {
  page: number
  width: number
  height: number
  annotations: SubmissionAnnotationApi[]
  readOnly: boolean
  tool: AnnotationTool
  colour: string
  selectedId?: string | null
  /** When true, highlight is created via text selection (PDF), not SVG rectangle drag. */
  textSelectHighlight: boolean
  onSelectAnnotation?: (id: string) => void
  onHighlightComplete?: (page: number, rects: NormRect[]) => void
  onDrawComplete?: (page: number, points: NormPoint[]) => void
  onPinComplete?: (page: number, pt: NormPoint) => void
  onTextBoxComplete?: (page: number, rect: NormRect) => void
}) {
  const rootRef = useRef<HTMLDivElement | null>(null)
  const [drag, setDrag] = useState<
    | { kind: 'highlight' | 'draw' | 'text'; start: NormPoint; cur: NormPoint; pts: NormPoint[] }
    | { kind: 'pin'; start: NormPoint }
    | null
  >(null)

  const annForPage = useMemo(() => annotations.filter((a) => a.page === page), [annotations, page])

  // SVG drag is disabled for highlight when text-selection mode is active so the text
  // layer underneath can receive the selection gesture.
  const svgInteractive =
    !readOnly && tool !== 'select' && !(tool === 'highlight' && textSelectHighlight)

  const onPointerDown = (e: React.PointerEvent) => {
    if (!svgInteractive || !rootRef.current) return
    const rect = rootRef.current.getBoundingClientRect()
    const p = normFromClient(rect, e.clientX, e.clientY)
    if (tool === 'highlight' || tool === 'text') {
      setDrag({ kind: tool, start: p, cur: p, pts: [p] })
    } else if (tool === 'draw') {
      setDrag({ kind: 'draw', start: p, cur: p, pts: [p] })
    } else if (tool === 'pin') {
      setDrag({ kind: 'pin', start: p })
    }
    rootRef.current.setPointerCapture(e.pointerId)
  }

  const onPointerMove = (e: React.PointerEvent) => {
    if (!drag || !svgInteractive || !rootRef.current) return
    const rect = rootRef.current.getBoundingClientRect()
    const p = normFromClient(rect, e.clientX, e.clientY)
    if (drag.kind === 'pin') return
    if (drag.kind === 'draw') {
      setDrag((d) => {
        if (!d || d.kind !== 'draw') return d
        const last = d.pts[d.pts.length - 1]
        if (last && Math.hypot(p.x - last.x, p.y - last.y) < 0.002) return d
        return { ...d, cur: p, pts: [...d.pts, p] }
      })
    } else {
      setDrag((d) => (d && d.kind !== 'pin' ? { ...d, cur: p } : d))
    }
  }

  const finish = (e: React.PointerEvent) => {
    if (!drag || !svgInteractive || !rootRef.current) return
    const rect = rootRef.current.getBoundingClientRect()
    const p = normFromClient(rect, e.clientX, e.clientY)
    try {
      rootRef.current.releasePointerCapture(e.pointerId)
    } catch {
      /* ignore */
    }
    if (drag.kind === 'pin') {
      onPinComplete?.(page, drag.start)
      setDrag(null)
      return
    }
    if (drag.kind === 'draw') {
      const pts = drag.pts.length >= 2 ? drag.pts : [...drag.pts, p]
      if (pts.length >= 2) onDrawComplete?.(page, pts)
      setDrag(null)
      return
    }
    const x1 = Math.min(drag.start.x, p.x)
    const x2 = Math.max(drag.start.x, p.x)
    const y1 = Math.min(drag.start.y, p.y)
    const y2 = Math.max(drag.start.y, p.y)
    if (Math.abs(x2 - x1) < 0.002 && Math.abs(y2 - y1) < 0.002) {
      setDrag(null)
      return
    }
    if (drag.kind === 'highlight') {
      onHighlightComplete?.(page, [{ x1, y1, x2, y2 }])
    } else if (drag.kind === 'text') {
      onTextBoxComplete?.(page, { x1, y1, x2, y2 })
    }
    setDrag(null)
  }

  const draftRect =
    drag && drag.kind !== 'pin' && drag.kind !== 'draw'
      ? {
          x1: Math.min(drag.start.x, drag.cur.x),
          y1: Math.min(drag.start.y, drag.cur.y),
          x2: Math.max(drag.start.x, drag.cur.x),
          y2: Math.max(drag.start.y, drag.cur.y),
        }
      : null

  return (
    <div ref={rootRef} className="absolute inset-0" style={{ width, height }}>
      <svg
        width={width}
        height={height}
        className={svgInteractive ? 'touch-none' : 'pointer-events-none'}
        style={{ pointerEvents: svgInteractive ? 'auto' : 'none' }}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={finish}
        onPointerCancel={() => setDrag(null)}
        role="img"
        aria-label={`Annotations page ${page}`}
      >
        {annForPage.map((a) => {
          const selected = selectedId === a.id
          const selectable = Boolean(onSelectAnnotation)
          const onClick = selectable
            ? (e: React.MouseEvent) => {
                e.stopPropagation()
                onSelectAnnotation?.(a.id)
              }
            : undefined
          const cursor = selectable ? 'pointer' : undefined
          // Selected annotations get pointer events so a click selects them even when the
          // active tool would otherwise route gestures elsewhere.
          const groupPointer = selectable ? ('auto' as const) : undefined
          if (a.toolType === 'highlight') {
            const rects = parseHighlightRects(a.coordsJson)
            if (rects.length === 0) return null
            return (
              <g key={a.id} onClick={onClick} style={{ cursor, pointerEvents: groupPointer }}>
                {rects.map((r, i) => {
                  const sx = Math.min(r.x1, r.x2) * width
                  const sy = Math.min(r.y1, r.y2) * height
                  const sw = Math.abs(r.x2 - r.x1) * width
                  const sh = Math.abs(r.y2 - r.y1) * height
                  return (
                    <rect
                      key={i}
                      x={sx}
                      y={sy}
                      width={sw}
                      height={sh}
                      fill={a.colour}
                      fillOpacity={selected ? 0.5 : 0.35}
                      stroke={selected ? a.colour : 'none'}
                      strokeWidth={selected ? 1.5 : 0}
                    />
                  )
                })}
              </g>
            )
          }
          if (a.toolType === 'draw') {
            const pts = parsePoints(a.coordsJson)
            if (pts.length < 2) return null
            const d = pts
              .map((q, i) => `${i === 0 ? 'M' : 'L'} ${q.x * width} ${q.y * height}`)
              .join(' ')
            return (
              <path
                key={a.id}
                d={d}
                fill="none"
                stroke={a.colour}
                strokeWidth={selected ? 3 : 2}
                strokeLinecap="round"
                strokeLinejoin="round"
                onClick={onClick}
                style={{ cursor, pointerEvents: groupPointer }}
              />
            )
          }
          if (a.toolType === 'pin') {
            const pt = parsePin(a.coordsJson)
            if (!pt) return null
            const cx = pt.x * width
            const cy = pt.y * height
            const s = Math.max(6, Math.min(width, height) * 0.02)
            return (
              <circle
                key={a.id}
                cx={cx}
                cy={cy}
                r={selected ? s * 1.25 : s}
                fill={a.colour}
                fillOpacity={0.85}
                stroke={selected ? '#1e293b' : 'none'}
                strokeWidth={selected ? 1.5 : 0}
                onClick={onClick}
                style={{ cursor, pointerEvents: groupPointer }}
              />
            )
          }
          if (a.toolType === 'text') {
            const r = parseRect(a.coordsJson)
            if (!r) return null
            const sx = Math.min(r.x1, r.x2) * width
            const sy = Math.min(r.y1, r.y2) * height
            const sw = Math.abs(r.x2 - r.x1) * width
            const sh = Math.abs(r.y2 - r.y1) * height
            return (
              <rect
                key={a.id}
                x={sx}
                y={sy}
                width={sw}
                height={sh}
                fill="none"
                stroke={a.colour}
                strokeDasharray="4 3"
                strokeWidth={selected ? 2.5 : 1.5}
                onClick={onClick}
                style={{ cursor, pointerEvents: groupPointer }}
              />
            )
          }
          return null
        })}
        {draftRect ? (
          <rect
            x={draftRect.x1 * width}
            y={draftRect.y1 * height}
            width={(draftRect.x2 - draftRect.x1) * width}
            height={(draftRect.y2 - draftRect.y1) * height}
            fill={colour}
            fillOpacity={0.25}
          />
        ) : null}
        {drag && drag.kind === 'draw' && drag.pts.length > 1 ? (
          <path
            d={drag.pts.map((q, i) => `${i === 0 ? 'M' : 'L'} ${q.x * width} ${q.y * height}`).join(' ')}
            fill="none"
            stroke={colour}
            strokeWidth={2}
          />
        ) : null}
      </svg>
    </div>
  )
}

const PDF_SCALE = 1.2

export function AnnotationViewer({
  filePath,
  mimeType,
  filename,
  readOnly,
  fallbackVariant = 'standalone',
  tool,
  colour,
  annotations,
  selectedId,
  onSelectAnnotation,
  onHighlightComplete,
  onDrawComplete,
  onPinComplete,
  onTextBoxComplete,
}: AnnotationViewerProps) {
  const [error, setError] = useState<string | null>(null)
  const [imageUrl, setImageUrl] = useState<string | null>(null)
  const [imageDisplaySize, setImageDisplaySize] = useState({ w: 400, h: 300 })
  const [pageLayouts, setPageLayouts] = useState<{ n: number; w: number; h: number }[]>([])
  const canvasRefs = useRef<(HTMLCanvasElement | null)[]>([])
  const textRefs = useRef<(HTMLDivElement | null)[]>([])
  const pageBoxRefs = useRef<(HTMLDivElement | null)[]>([])
  const pageObjsRef = useRef<Awaited<ReturnType<PDFDocumentProxy['getPage']>>[]>([])

  useEffect(() => {
    ensureTextLayerStyles()
  }, [])

  // PDF text highlighting works by selecting the transparent text layer; for images we keep
  // the rectangle-drag fallback.
  const textSelectHighlight = mimeType === 'application/pdf'

  useEffect(() => {
    if (!filePath || !mimeType) return undefined
    const fp = filePath
    const mt = mimeType

    let cancelled = false
    let blobUrl: string | null = null
    let loadedPdf: PDFDocumentProxy | null = null

    async function run() {
      setError(null)
      setPageLayouts([])
      pageObjsRef.current = []
      setImageUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev)
        return null
      })

      try {
        if (mt === 'application/pdf') {
          const token = getAccessToken()
          const doc = await getDocument({
            url: apiUrl(fp),
            httpHeaders: token ? { Authorization: `Bearer ${token}` } : undefined,
            withCredentials: false,
          }).promise
          if (cancelled) {
            await doc.destroy().catch(() => {})
            return
          }
          loadedPdf = doc
          const layouts: { n: number; w: number; h: number }[] = []
          const pages: Awaited<ReturnType<PDFDocumentProxy['getPage']>>[] = []
          for (let i = 1; i <= doc.numPages; i += 1) {
            const page = await doc.getPage(i)
            const vp = page.getViewport({ scale: PDF_SCALE })
            layouts.push({ n: i, w: vp.width, h: vp.height })
            pages.push(page)
          }
          if (!cancelled) {
            pageObjsRef.current = pages
            setPageLayouts(layouts)
          }
        } else if (mt.startsWith('image/')) {
          const res = await authorizedFetch(fp)
          if (!res.ok) throw new Error('Could not load image.')
          const blob = await res.blob()
          if (cancelled) return
          blobUrl = URL.createObjectURL(blob)
          setImageUrl(blobUrl)
        } else {
          setError('Preview is only available for PDF and image submissions.')
        }
      } catch {
        if (!cancelled) setError('Could not load submission. Retry?')
      }
    }

    void run()
    return () => {
      cancelled = true
      if (blobUrl) URL.revokeObjectURL(blobUrl)
      if (loadedPdf) void loadedPdf.destroy().catch(() => {})
    }
  }, [filePath, mimeType])

  const renderPdfPages = useCallback(async () => {
    const pages = pageObjsRef.current
    if (!pages.length) return
    await Promise.all(
      pages.map(async (page, idx) => {
        const canvas = canvasRefs.current[idx]
        if (!canvas) return
        const vp = page.getViewport({ scale: PDF_SCALE })
        canvas.width = vp.width
        canvas.height = vp.height
        const ctx = canvas.getContext('2d')
        if (ctx) {
          await page.render({ canvasContext: ctx, viewport: vp }).promise
        }
        // Selectable transparent text layer used to drive highlight selection + copy/find.
        const textDiv = textRefs.current[idx]
        if (textDiv) {
          textDiv.innerHTML = ''
          textDiv.className = 'pdf-tl'
          textDiv.style.setProperty('--scale-factor', String(PDF_SCALE))
          try {
            const textContent = await page.getTextContent()
            const layer = new TextLayer({ textContentSource: textContent, container: textDiv, viewport: vp })
            await layer.render()
          } catch {
            /* text layer is best-effort */
          }
        }
      }),
    )
  }, [])

  useEffect(() => {
    if (!pageLayouts.length) return
    void renderPdfPages()
  }, [pageLayouts, renderPdfPages])

  const handlePageMouseUp = useCallback(
    (idx: number) => {
      if (readOnly || tool !== 'highlight' || !textSelectHighlight) return
      const box = pageBoxRefs.current[idx]
      const layout = pageLayouts[idx]
      if (!box || !layout) return
      const rects = selectionRectsForBox(box.getBoundingClientRect())
      if (rects.length === 0) return
      onHighlightComplete?.(layout.n, rects)
      window.getSelection()?.removeAllRanges()
    },
    [readOnly, tool, textSelectHighlight, pageLayouts, onHighlightComplete],
  )

  if (!filePath || !mimeType) {
    return (
      <p className="rounded-lg border border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-600 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300">
        No file on this submission yet.
      </p>
    )
  }

  if (error) {
    if (filePath && filename) {
      return (
        <FilePreviewFallback
          filePath={filePath}
          filename={filename}
          message={error}
          downloadLabel="Download submission"
          variant={fallbackVariant}
        />
      )
    }
    return (
      <p className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-6 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200">
        {error}
      </p>
    )
  }

  if (imageUrl) {
    return (
      <div className="max-h-[80vh] overflow-auto rounded-xl border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-950">
        <div className="relative inline-block max-w-full">
          <img
            src={imageUrl}
            alt="Submission"
            className="lex-content-img block h-auto max-h-[80vh] max-w-full object-contain"
            onLoad={(e) => {
              const el = e.currentTarget
              setImageDisplaySize({ w: el.offsetWidth || 400, h: el.offsetHeight || 300 })
            }}
          />
          <PageOverlay
            page={1}
            width={imageDisplaySize.w}
            height={imageDisplaySize.h}
            annotations={annotations}
            readOnly={readOnly}
            tool={tool}
            colour={colour}
            selectedId={selectedId}
            textSelectHighlight={false}
            onSelectAnnotation={onSelectAnnotation}
            onHighlightComplete={onHighlightComplete}
            onDrawComplete={onDrawComplete}
            onPinComplete={onPinComplete}
            onTextBoxComplete={onTextBoxComplete}
          />
        </div>
        <p className="border-t border-slate-200 px-3 py-2 text-xs text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
          Image submissions highlight by dragging a box. Use PDF uploads to highlight selected text.
        </p>
      </div>
    )
  }

  if (!pageLayouts.length) {
    return <p className="text-sm text-slate-500 dark:text-neutral-400">Loading PDF…</p>
  }

  const textLayerInteractive = !readOnly && tool === 'highlight' && textSelectHighlight

  return (
    <div className="max-h-[80vh] space-y-4 overflow-auto rounded-xl border border-slate-200 bg-slate-50/50 p-3 dark:border-neutral-700 dark:bg-neutral-950/40">
      {pageLayouts.map((pv, idx) => (
        <div
          key={pv.n}
          ref={(el) => {
            pageBoxRefs.current[idx] = el
          }}
          className="relative mx-auto shadow-sm"
          style={{ width: pv.w, height: pv.h }}
          onMouseUp={() => handlePageMouseUp(idx)}
        >
          <canvas
            ref={(el) => {
              canvasRefs.current[idx] = el
            }}
            className="block bg-white dark:bg-neutral-950"
          />
          <div
            ref={(el) => {
              textRefs.current[idx] = el
            }}
            aria-hidden="true"
            style={{
              position: 'absolute',
              inset: 0,
              width: pv.w,
              height: pv.h,
              pointerEvents: textLayerInteractive ? 'auto' : 'none',
              cursor: textLayerInteractive ? 'text' : 'default',
            }}
          />
          <PageOverlay
            page={pv.n}
            width={pv.w}
            height={pv.h}
            annotations={annotations}
            readOnly={readOnly}
            tool={tool}
            colour={colour}
            selectedId={selectedId}
            textSelectHighlight={textSelectHighlight}
            onSelectAnnotation={onSelectAnnotation}
            onHighlightComplete={onHighlightComplete}
            onDrawComplete={onDrawComplete}
            onPinComplete={onPinComplete}
            onTextBoxComplete={onTextBoxComplete}
          />
        </div>
      ))}
    </div>
  )
}

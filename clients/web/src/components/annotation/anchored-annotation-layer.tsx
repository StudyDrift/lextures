import { useCallback, useEffect, useLayoutEffect, useState, type RefObject } from 'react'
import type { SubmissionAnnotationApi } from '../../lib/courses-api'
import { findAnchorRange, getSelectionAnchor, parseTextAnchor, type TextAnchor } from '../../lib/text-anchor'

/**
 * Annotation wiring passed to a reflowable preview (text/code/office) to make it annotatable.
 * When omitted, the preview renders read-only with no highlight surface.
 */
export type AnchorSurfaceProps = {
  annotations: SubmissionAnnotationApi[]
  readOnly: boolean
  selectedAnnotationId?: string | null
  onSelectAnnotation?: (id: string) => void
  /** Omit to disable highlight creation (e.g. student view). */
  onAnchorComplete?: (anchor: TextAnchor) => void
}

type Box = { left: number; top: number; width: number; height: number }
type OverlayItem = { id: string; colour: string; selected: boolean; boxes: Box[] }

export type AnchoredAnnotationLayerProps = {
  /** The scrollable, position:relative element the overlay is painted into. */
  scrollRef: RefObject<HTMLElement | null>
  /** The element whose text nodes are anchored. Defaults to `scrollRef`. */
  contentRef?: RefObject<HTMLElement | null>
  annotations: SubmissionAnnotationApi[]
  readOnly: boolean
  selectedId?: string | null
  onSelectAnnotation?: (id: string) => void
  /** Called when the grader selects text to create a highlight. Omit to disable creation. */
  onAnchorComplete?: (anchor: TextAnchor) => void
  /** Bump when async content finishes loading so highlights recompute against the new DOM. */
  recomputeKey?: unknown
}

/**
 * Paints text-anchor highlights as absolutely-positioned overlay rectangles over a reflowable
 * preview (Markdown/text/code/office HTML), and turns a text selection into a new anchor. The
 * geometric sibling for PDF/image is `PageOverlay` in `annotation-viewer.tsx`; this one derives
 * its rectangles from live DOM Ranges instead of stored normalized coordinates, so it survives
 * reflow.
 */
export function AnchoredAnnotationLayer({
  scrollRef,
  contentRef,
  annotations,
  readOnly,
  selectedId,
  onSelectAnnotation,
  onAnchorComplete,
  recomputeKey,
}: AnchoredAnnotationLayerProps) {
  const [items, setItems] = useState<OverlayItem[]>([])

  const recompute = useCallback(() => {
    const scroll = scrollRef.current
    const content = (contentRef ?? scrollRef).current
    if (!scroll || !content) {
      setItems([])
      return
    }
    const cbox = scroll.getBoundingClientRect()
    // Convert a viewport rect to coordinates relative to the scroll container's content box,
    // so the overlay scrolls together with the text it marks.
    const toLocal = (r: DOMRect): Box => ({
      left: r.left - cbox.left - scroll.clientLeft + scroll.scrollLeft,
      top: r.top - cbox.top - scroll.clientTop + scroll.scrollTop,
      width: r.width,
      height: r.height,
    })

    const next: OverlayItem[] = []
    for (const a of annotations) {
      if (a.toolType !== 'anchor') continue
      const anchor = parseTextAnchor(a.coordsJson)
      if (!anchor) continue
      const range = findAnchorRange(content, anchor)
      if (!range) continue
      const boxes: Box[] = []
      const rects = range.getClientRects()
      for (let i = 0; i < rects.length; i += 1) {
        const r = rects[i]
        if (r && r.width >= 1 && r.height >= 1) boxes.push(toLocal(r))
      }
      if (boxes.length > 0) {
        next.push({ id: a.id, colour: a.colour, selected: selectedId === a.id, boxes })
      }
    }
    setItems(next)
  }, [annotations, contentRef, scrollRef, selectedId])

  // Recompute synchronously after layout so highlights line up on first paint.
  useLayoutEffect(() => {
    recompute()
  }, [recompute, recomputeKey])

  // Reflow (resize, font/content load) changes the rectangles — recompute on layout changes.
  useEffect(() => {
    const content = (contentRef ?? scrollRef).current
    if (!content) return
    let frame = 0
    const schedule = () => {
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(recompute)
    }
    const ro = new ResizeObserver(schedule)
    ro.observe(content)
    window.addEventListener('resize', schedule)
    return () => {
      cancelAnimationFrame(frame)
      ro.disconnect()
      window.removeEventListener('resize', schedule)
    }
  }, [contentRef, scrollRef, recompute])

  // Turn a text selection into an anchor. The listener lives on the document (not the content
  // element) so a drag that ends outside the content box — in the panel padding, on the
  // scrollbar, or past the edge — is still captured; getSelectionAnchor ignores selections that
  // fall outside the content.
  useEffect(() => {
    if (readOnly || !onAnchorComplete) return
    const content = (contentRef ?? scrollRef).current
    if (!content) return
    const doc = content.ownerDocument
    const onMouseUp = () => {
      const target = (contentRef ?? scrollRef).current
      if (!target) return
      const anchor = getSelectionAnchor(target)
      if (!anchor) return
      onAnchorComplete(anchor)
      doc.defaultView?.getSelection()?.removeAllRanges()
    }
    doc.addEventListener('mouseup', onMouseUp)
    return () => doc.removeEventListener('mouseup', onMouseUp)
  }, [contentRef, scrollRef, readOnly, onAnchorComplete])

  const selectable = Boolean(onSelectAnnotation)

  // Zero-size, top-left anchored wrapper: each box positions itself relative to the scroll
  // container's content origin and scrolls with the content (overflow is intentionally visible).
  return (
    <div
      aria-hidden
      className="pointer-events-none absolute z-10"
      // Physical top/left: boxes are positioned by measured pixel offsets (RTL-consistent),
      // so the wrapper origin must stay physical rather than logical (start/end).
      style={{ top: 0, left: 0, width: 0, height: 0 }}
    >
      {items.map((item) =>
        item.boxes.map((b, i) => (
          <div
            key={`${item.id}:${i}`}
            onClick={
              selectable
                ? (e) => {
                    e.stopPropagation()
                    onSelectAnnotation?.(item.id)
                  }
                : undefined
            }
            style={{
              position: 'absolute',
              left: b.left,
              top: b.top,
              width: b.width,
              height: b.height,
              backgroundColor: item.colour,
              opacity: item.selected ? 0.5 : 0.3,
              mixBlendMode: 'multiply',
              borderRadius: 2,
              outline: item.selected ? `2px solid ${item.colour}` : 'none',
              pointerEvents: selectable ? 'auto' : 'none',
              cursor: selectable ? 'pointer' : undefined,
            }}
          />
        )),
      )}
    </div>
  )
}

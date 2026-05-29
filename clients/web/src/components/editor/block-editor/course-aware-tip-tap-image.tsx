/* eslint-disable react-refresh/only-export-components -- TipTap extension + internal node view */
import Image from '@tiptap/extension-image'
import { NodeViewWrapper, ReactNodeViewRenderer } from '@tiptap/react'
import type { NodeViewProps } from '@tiptap/react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { authorizedFetch } from '../../../lib/api'
import {
  needsAuthenticatedCourseImageSrc,
  resolveAuthorizedFetchPath,
  stripImageDisplayFragment,
} from '../../../lib/course-file-image'
import { DECORATIVE_IMAGE_TITLE } from '../../../lib/image-alt-validation'
import { AltTextPanel } from './alt-text-panel'
import { useAltTextEnforcement } from './alt-text-enforcement-context'

const MIN_DISPLAY = 48

function imageHasValidAlt(alt: string, decorative: boolean): boolean {
  return decorative || alt.trim().length > 0
}

function CourseImageNodeView(props: NodeViewProps) {
  const { node, updateAttributes, selected } = props
  const { enabled: enforcementEnabled } = useAltTextEnforcement()
  const src = (node.attrs.src as string) ?? ''
  const alt = (node.attrs.alt as string) ?? ''
  const decorative = Boolean(node.attrs.decorative)
  const altPending = Boolean(node.attrs.altPending)
  const widthAttr = node.attrs.width as number | null | undefined
  const heightAttr = node.attrs.height as number | null | undefined

  const imgRef = useRef<HTMLImageElement>(null)
  const dragRef = useRef<{
    startDist: number
    baseW: number
    baseH: number
    aspect: number
  } | null>(null)
  const latestDragSize = useRef<{ w: number; h: number } | null>(null)

  const [url, setUrl] = useState<string | null>(() =>
    src && !needsAuthenticatedCourseImageSrc(src) ? src : null,
  )
  const [dragSize, setDragSize] = useState<{ w: number; h: number } | null>(null)
  const [panelDismissed, setPanelDismissed] = useState(false)
  const [panelForcedOpen, setPanelForcedOpen] = useState(
    () => enforcementEnabled && (altPending || !imageHasValidAlt(alt, decorative)),
  )

  const needsAlt = enforcementEnabled && !imageHasValidAlt(alt, decorative)
  const panelOpen =
    (enforcementEnabled && altPending) ||
    panelForcedOpen ||
    (needsAlt && selected && !panelDismissed)

  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect -- sync URL clear / direct src before async blob fetch */
    let cancelled = false
    let blobUrl: string | null = null
    if (!src) {
      setUrl(null)
      return
    }
    if (!needsAuthenticatedCourseImageSrc(src)) {
      setUrl(src)
      return
    }
    /* eslint-enable react-hooks/set-state-in-effect */
    const path = resolveAuthorizedFetchPath(src)
    void authorizedFetch(path)
      .then((r) => {
        if (!r.ok) throw new Error(String(r.status))
        return r.blob()
      })
      .then((blob) => {
        if (cancelled) return
        blobUrl = URL.createObjectURL(blob)
        setUrl(blobUrl)
      })
      .catch(() => {
        if (!cancelled) setUrl(null)
      })
    return () => {
      cancelled = true
      if (blobUrl) URL.revokeObjectURL(blobUrl)
    }
  }, [src])

  const effectiveW = dragSize?.w ?? widthAttr
  const effectiveH = dragSize?.h ?? heightAttr

  const beginCornerScale = useCallback(
    (e: React.PointerEvent) => {
      e.preventDefault()
      e.stopPropagation()
      const img = imgRef.current
      if (!img || !url) return
      const rect = img.getBoundingClientRect()
      const w0 = typeof widthAttr === 'number' && widthAttr > 0 ? widthAttr : rect.width
      const h0 = typeof heightAttr === 'number' && heightAttr > 0 ? heightAttr : rect.height
      if (w0 < 4 || h0 < 4) return
      const aspect = w0 / h0
      const cx = rect.left + rect.width / 2
      const cy = rect.top + rect.height / 2
      const startDist = Math.max(
        Math.hypot(e.clientX - cx, e.clientY - cy),
        Math.max(w0, h0) * 0.08,
      )
      dragRef.current = { startDist, baseW: w0, baseH: h0, aspect }
      latestDragSize.current = { w: w0, h: h0 }
      ;(e.target as HTMLElement).setPointerCapture(e.pointerId)

      const onMove = (ev: PointerEvent) => {
        const d = dragRef.current
        if (!d) return
        const dist = Math.hypot(ev.clientX - cx, ev.clientY - cy)
        let ratio = dist / d.startDist
        ratio = Math.max(ratio, MIN_DISPLAY / Math.max(d.baseW, d.baseH))
        const nw = Math.round(d.baseW * ratio)
        const nh = Math.round(nw / d.aspect)
        const next = { w: Math.max(MIN_DISPLAY, nw), h: Math.max(MIN_DISPLAY, nh) }
        latestDragSize.current = next
        setDragSize(next)
      }

      const onUp = (ev: PointerEvent) => {
        dragRef.current = null
        try {
          ;(e.target as HTMLElement).releasePointerCapture(ev.pointerId)
        } catch {
          /* ignore */
        }
        window.removeEventListener('pointermove', onMove)
        window.removeEventListener('pointerup', onUp)
        window.removeEventListener('pointercancel', onUp)
        const fin = latestDragSize.current
        latestDragSize.current = null
        if (fin) {
          updateAttributes({ width: fin.w, height: fin.h })
        }
        setDragSize(null)
      }

      window.addEventListener('pointermove', onMove)
      window.addEventListener('pointerup', onUp)
      window.addEventListener('pointercancel', onUp)
    },
    [heightAttr, updateAttributes, url, widthAttr],
  )

  const handleClass =
    'absolute z-10 h-2.5 w-2.5 rounded-sm border border-white bg-indigo-500 shadow ring-1 ring-indigo-600/40 pointer-events-auto touch-none'

  const showMissingBadge =
    enforcementEnabled && !imageHasValidAlt(alt, decorative) && !panelOpen

  return (
    <NodeViewWrapper as="figure" className="group/image-node relative my-3 flex justify-center [&_img]:rounded-lg">
      <div className="relative inline-block max-w-full">
        {url ? (
          <img
            ref={imgRef}
            src={url}
            alt={decorative ? '' : alt}
            draggable={false}
            data-drag-handle=""
            className={`box-border max-w-full rounded-lg border object-contain ${
              showMissingBadge
                ? 'border-amber-400 ring-2 ring-amber-300/60 dark:border-amber-600'
                : 'border-slate-200 dark:border-neutral-700'
            }`}
            style={
              typeof effectiveW === 'number' && typeof effectiveH === 'number'
                ? {
                    width: effectiveW,
                    height: effectiveH,
                    objectFit: 'contain',
                    display: 'block',
                  }
                : { maxWidth: '100%', height: 'auto', display: 'block' }
            }
          />
        ) : (
          <span className="text-sm text-slate-500 dark:text-neutral-400">Loading image…</span>
        )}

        {showMissingBadge ? (
          <button
            type="button"
            onClick={() => {
              setPanelDismissed(false)
              setPanelForcedOpen(true)
            }}
            className="absolute -top-2 start-1/2 z-10 -translate-x-1/2 rounded-full bg-amber-500 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-white shadow"
          >
            Alt text required
          </button>
        ) : null}

        {selected && url ? (
          <>
            <div
              className="pointer-events-none absolute inset-0 rounded-lg ring-2 ring-indigo-500 ring-offset-2 ring-offset-white dark:ring-offset-neutral-950"
              aria-hidden
            />
            <button
              type="button"
              aria-label="Edit alt text"
              onClick={() => {
                setPanelDismissed(false)
                setPanelForcedOpen(true)
              }}
              className="absolute -top-2 end-2 z-10 rounded-md bg-white px-2 py-0.5 text-[10px] font-medium text-indigo-700 shadow ring-1 ring-slate-200 dark:bg-neutral-900 dark:text-indigo-300 dark:ring-neutral-600"
            >
              Alt text
            </button>
            <button
              type="button"
              aria-label="Resize image from corner"
              title="Drag to resize"
              className={`${handleClass} -start-1.5 -top-1.5 cursor-nwse-resize`}
              onPointerDown={beginCornerScale}
            />
            <button
              type="button"
              aria-label="Resize image from corner"
              title="Drag to resize"
              className={`${handleClass} -end-1.5 -top-1.5 cursor-nesw-resize`}
              onPointerDown={beginCornerScale}
            />
            <button
              type="button"
              aria-label="Resize image from corner"
              title="Drag to resize"
              className={`${handleClass} -bottom-1.5 -start-1.5 cursor-nesw-resize`}
              onPointerDown={beginCornerScale}
            />
            <button
              type="button"
              aria-label="Resize image from corner"
              title="Drag to resize"
              className={`${handleClass} -bottom-1.5 -end-1.5 cursor-nwse-resize`}
              onPointerDown={beginCornerScale}
            />
          </>
        ) : null}

        {panelOpen ? (
          <AltTextPanel
            alt={alt}
            decorative={decorative}
            imageSrc={src}
            imageUrlForAi={url}
            autoFocus={altPending || !imageHasValidAlt(alt, decorative)}
            onApply={({ alt: nextAlt, decorative: nextDecorative }) => {
              updateAttributes({
                alt: nextDecorative ? '' : nextAlt,
                decorative: nextDecorative,
                altPending: false,
              })
              setPanelForcedOpen(false)
              setPanelDismissed(true)
            }}
            onClose={() => {
              updateAttributes({ altPending: false })
              setPanelForcedOpen(false)
              setPanelDismissed(true)
            }}
          />
        ) : null}
      </div>
    </NodeViewWrapper>
  )
}

export const CourseAwareTipTapImage = Image.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      decorative: {
        default: false,
        parseHTML: (element) => element.getAttribute('data-decorative') === 'true',
        renderHTML: (attributes) =>
          attributes.decorative ? { 'data-decorative': 'true' } : {},
      },
      altPending: {
        default: false,
        rendered: false,
      },
    }
  },

  addNodeView() {
    return ReactNodeViewRenderer(CourseImageNodeView)
  },

  renderMarkdown(node) {
    const a = node.attrs ?? {}
    const alt = (a.alt as string | null | undefined) ?? ''
    const decorative = Boolean(a.decorative)
    const title = (a.title as string | null | undefined) ?? ''
    const width = a.width as number | null | undefined
    const height = a.height as number | null | undefined
    let src = (a.src as string | null | undefined) ?? ''
    const base = stripImageDisplayFragment(src).base
    src = base
    if (
      typeof width === 'number' &&
      typeof height === 'number' &&
      Number.isFinite(width) &&
      Number.isFinite(height) &&
      width > 0 &&
      height > 0
    ) {
      src = `${base}#w=${Math.round(width)}&h=${Math.round(height)}`
    }
    const effectiveTitle = decorative ? DECORATIVE_IMAGE_TITLE : title
    const dest = effectiveTitle ? `${src} "${effectiveTitle.replace(/"/g, '\\"')}"` : src
    return `![${decorative ? '' : alt}](${dest})`
  },

  parseMarkdown(token, helpers) {
    const href = (token as { href?: string }).href ?? ''
    const { base, displayWidth, displayHeight } = stripImageDisplayFragment(href)
    const title = (token as { title?: string | null }).title ?? null
    const decorative = title === DECORATIVE_IMAGE_TITLE
    return helpers.createNode('image', {
      src: base,
      alt: decorative ? '' : ((token as { text?: string }).text ?? ''),
      title: decorative ? null : title,
      decorative,
      width: displayWidth ?? null,
      height: displayHeight ?? null,
    })
  },
})

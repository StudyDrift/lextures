import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent, type WheelEvent } from 'react'
import { useTranslation } from 'react-i18next'
import type { BoardPostPosition } from '../../../lib/boards-api'
import { PostCard } from '../post-card'
import { CardArrangeMenu } from '../card-arrange-menu'
import { BoardLiveCursors } from '../board-live-cursors'
import { postCardEngagementProps, type LayoutRendererProps } from './types'

const DEFAULT_W = 260
const DEFAULT_H = 180
/** Throttle live CRDT position publishes so peers see drag without flooding the relay. */
const LIVE_ARRANGE_MS = 40

function postPosition(post: LayoutRendererProps['posts'][number], index: number): BoardPostPosition {
  if (post.position) return post.position
  const col = index % 4
  const row = Math.floor(index / 4)
  return { x: 40 + col * (DEFAULT_W + 24), y: 40 + row * (DEFAULT_H + 24), w: DEFAULT_W, h: DEFAULT_H }
}

export function CanvasLayout(props: LayoutRendererProps) {
  const { t } = useTranslation('common')
  const containerRef = useRef<HTMLDivElement>(null)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [zoom, setZoom] = useState(1)
  const [panning, setPanning] = useState(false)
  const panOrigin = useRef<{ x: number; y: number; panX: number; panY: number } | null>(null)
  const dragRef = useRef<{
    postId: string
    startX: number
    startY: number
    orig: BoardPostPosition
  } | null>(null)
  const debounceTimers = useRef<Map<string, number>>(new Map())
  const liveThrottleAt = useRef(0)
  /** Local overlay so the dragging user sees smooth motion between CRDT ticks. */
  const [livePositions, setLivePositions] = useState<Map<string, BoardPostPosition>>(() => new Map())
  const livePositionsRef = useRef(livePositions)
  livePositionsRef.current = livePositions

  useEffect(() => {
    const timers = debounceTimers.current
    return () => {
      for (const id of timers.values()) window.clearTimeout(id)
    }
  }, [])

  function publishLive(postId: string, position: BoardPostPosition, force = false) {
    const now = performance.now()
    if (!force && now - liveThrottleAt.current < LIVE_ARRANGE_MS) {
      setLivePositions((prev) => {
        const next = new Map(prev)
        next.set(postId, position)
        return next
      })
      return
    }
    liveThrottleAt.current = now
    setLivePositions((prev) => {
      const next = new Map(prev)
      next.set(postId, position)
      return next
    })
    props.onLiveArrange?.(postId, { position })
  }

  function persistPosition(postId: string, position: BoardPostPosition) {
    const existing = debounceTimers.current.get(postId)
    if (existing) window.clearTimeout(existing)
    // Final CRDT publish immediately so peers converge; REST follows after debounce.
    props.onLiveArrange?.(postId, { position })
    const timer = window.setTimeout(() => {
      void props.onArrange(postId, { position }).finally(() => {
        setLivePositions((prev) => {
          if (!prev.has(postId)) return prev
          const next = new Map(prev)
          next.delete(postId)
          return next
        })
      })
      debounceTimers.current.delete(postId)
    }, 250)
    debounceTimers.current.set(postId, timer)
  }

  function onWheel(e: WheelEvent) {
    if (!e.ctrlKey && !e.metaKey) return
    e.preventDefault()
    const delta = e.deltaY > 0 ? -0.08 : 0.08
    setZoom((z) => Math.min(2.5, Math.max(0.35, z + delta)))
  }

  function onBackgroundPointerDown(e: ReactPointerEvent) {
    if (e.button !== 0 && e.button !== 1) return
    if ((e.target as HTMLElement).closest('[data-board-card]')) return
    setPanning(true)
    panOrigin.current = { x: e.clientX, y: e.clientY, panX: pan.x, panY: pan.y }
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }

  function onBackgroundPointerMove(e: ReactPointerEvent) {
    const rect = containerRef.current?.getBoundingClientRect()
    if (rect && props.onCursorMove) {
      const x = (e.clientX - rect.left - pan.x) / zoom
      const y = (e.clientY - rect.top - pan.y) / zoom
      props.onCursorMove({ x, y })
    }
    if (panning && panOrigin.current) {
      setPan({
        x: panOrigin.current.panX + (e.clientX - panOrigin.current.x),
        y: panOrigin.current.panY + (e.clientY - panOrigin.current.y),
      })
      return
    }
    const drag = dragRef.current
    if (!drag) return
    const dx = (e.clientX - drag.startX) / zoom
    const dy = (e.clientY - drag.startY) / zoom
    const next = { ...drag.orig, x: drag.orig.x + dx, y: drag.orig.y + dy }
    publishLive(drag.postId, next)
  }

  function onBackgroundPointerUp(e: ReactPointerEvent) {
    if (panning) {
      setPanning(false)
      panOrigin.current = null
    }
    const drag = dragRef.current
    if (drag) {
      const pending = livePositionsRef.current.get(drag.postId)
      const position = pending ?? drag.orig
      publishLive(drag.postId, position, true)
      persistPosition(drag.postId, position)
      props.onAnnounce(t('boards.arrange.moved'))
      dragRef.current = null
    }
    try {
      ;(e.currentTarget as HTMLElement).releasePointerCapture(e.pointerId)
    } catch {
      /* ignore */
    }
  }

  if (props.posts.length === 0) {
    return (
      <p className="m-auto max-w-md px-4 text-center text-sm text-slate-500 dark:text-neutral-400">
        {t('boards.detail.emptyPosts')}
      </p>
    )
  }

  return (
    <div className="flex min-h-96 flex-1 flex-col gap-2">
      <div className="flex items-center gap-2 text-xs text-slate-500">
        <button type="button" className="rounded border px-2 py-1 dark:border-neutral-700" onClick={() => setZoom((z) => Math.min(2.5, z + 0.1))}>
          +
        </button>
        <button type="button" className="rounded border px-2 py-1 dark:border-neutral-700" onClick={() => setZoom((z) => Math.max(0.35, z - 0.1))}>
          −
        </button>
        <span>{Math.round(zoom * 100)}%</span>
        <span className="text-slate-400">{t('boards.layout.canvasHint')}</span>
      </div>
      <div
        ref={containerRef}
        className="relative min-h-96 flex-1 overflow-hidden rounded-lg border border-slate-200 bg-[radial-gradient(circle_at_1px_1px,#cbd5e1_1px,transparent_0)] bg-[length:16px_16px] dark:border-neutral-700 dark:bg-[radial-gradient(circle_at_1px_1px,#404040_1px,transparent_0)]"
        role="region"
        aria-label={t('boards.layout.canvas')}
        onWheel={onWheel}
        onPointerDown={onBackgroundPointerDown}
        onPointerMove={onBackgroundPointerMove}
        onPointerUp={onBackgroundPointerUp}
        onPointerCancel={onBackgroundPointerUp}
      >
        <div
          className="absolute origin-top-left"
          style={{ transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})` }}
          onPointerLeave={() => props.onCursorMove?.(null)}
        >
          {props.awareness ? <BoardLiveCursors awareness={props.awareness} enabled /> : null}
          {props.posts.map((post, i) => {
            const pos = livePositions.get(post.id) ?? postPosition(post, i)
            const canArrange = props.canArrangePost(post)
            return (
              <div
                key={post.id}
                data-board-card
                data-post-id={post.id}
                className="absolute"
                style={{ left: pos.x, top: pos.y, width: pos.w, minHeight: pos.h }}
                onPointerDown={(e) => {
                  if (!canArrange || e.button !== 0) return
                  // Don't steal clicks from card actions (delete, arrange, report, forms).
                  const target = e.target as HTMLElement | null
                  if (
                    target?.closest(
                      'button, a, input, textarea, select, label, [role="button"], [role="menuitem"], [role="menu"], [role="dialog"], [data-no-card-drag]',
                    )
                  ) {
                    return
                  }
                  e.stopPropagation()
                  dragRef.current = {
                    postId: post.id,
                    startX: e.clientX,
                    startY: e.clientY,
                    orig: pos,
                  }
                  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
                }}
              >
                <div className={canArrange ? 'cursor-grab active:cursor-grabbing' : undefined}>
                  <PostCard
                    post={post}
                    {...postCardEngagementProps(props, post)}
                    headerActions={
                      <CardArrangeMenu
                        post={post}
                        sections={props.sections}
                        siblings={props.posts}
                        canArrange={canArrange}
                        onMoveToSection={(sectionId) => void props.onArrange(post.id, { sectionId })}
                        onReorder={(sortIndex) => void props.onArrange(post.id, { sortIndex })}
                      />
                    }
                  />
                </div>
                {canArrange ? (
                  <div
                    className="absolute bottom-0 end-0 size-3 cursor-se-resize bg-indigo-400/80"
                    aria-hidden
                    onPointerDown={(e) => {
                      e.stopPropagation()
                      const startX = e.clientX
                      const startY = e.clientY
                      const orig = { ...pos }
                      let pending = { ...orig }
                      const onMove = (ev: PointerEvent) => {
                        const w = Math.max(180, orig.w + (ev.clientX - startX) / zoom)
                        const h = Math.max(120, orig.h + (ev.clientY - startY) / zoom)
                        pending = { ...orig, w, h }
                        publishLive(post.id, pending)
                      }
                      const onUp = () => {
                        window.removeEventListener('pointermove', onMove)
                        window.removeEventListener('pointerup', onUp)
                        publishLive(post.id, pending, true)
                        persistPosition(post.id, pending)
                      }
                      window.addEventListener('pointermove', onMove)
                      window.addEventListener('pointerup', onUp)
                    }}
                  />
                ) : null}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

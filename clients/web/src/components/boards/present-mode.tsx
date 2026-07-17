import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight, Maximize2, X } from 'lucide-react'
import type { BoardPost, BoardSection } from '../../lib/boards-api'

type Props = {
  open: boolean
  onClose: () => void
  boardTitle: string
  posts: BoardPost[]
  sections: BoardSection[]
}

type Mode = 'slideshow' | 'overview'

function postBodyText(post: BoardPost): string {
  const body = post.body
  if (!body) return ''
  if (typeof body === 'string') return body
  if (typeof body.text === 'string') return body.text
  return ''
}

export function BoardPresentMode({ open, onClose, boardTitle, posts, sections }: Props) {
  const { t } = useTranslation('common')
  const [mode, setMode] = useState<Mode>('slideshow')
  const [index, setIndex] = useState(0)
  const [autoAdvance, setAutoAdvance] = useState(false)

  const ordered = useMemo(() => {
    const secOrder = new Map(sections.map((s) => [s.id, s.sortIndex]))
    return [...posts].sort((a, b) => {
      const aSec = a.sectionId ? (secOrder.get(a.sectionId) ?? 0) : Number.POSITIVE_INFINITY
      const bSec = b.sectionId ? (secOrder.get(b.sectionId) ?? 0) : Number.POSITIVE_INFINITY
      if (aSec !== bSec) return aSec - bSec
      return a.sortIndex - b.sortIndex
    })
  }, [posts, sections])

  const sectionTitle = (post: BoardPost) =>
    post.sectionId ? sections.find((s) => s.id === post.sectionId)?.title : undefined

  useEffect(() => {
    if (!open) return
    setIndex(0)
    setMode('slideshow')
    setAutoAdvance(false)
  }, [open])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onClose()
        return
      }
      if (mode !== 'slideshow') return
      if (e.key === 'ArrowRight' || e.key === ' ') {
        e.preventDefault()
        setIndex((i) => Math.min(ordered.length - 1, i + 1))
      } else if (e.key === 'ArrowLeft') {
        e.preventDefault()
        setIndex((i) => Math.max(0, i - 1))
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, mode, onClose, ordered.length])

  useEffect(() => {
    if (!open || !autoAdvance || mode !== 'slideshow' || ordered.length === 0) return
    const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (reduceMotion) return
    const id = window.setInterval(() => {
      setIndex((i) => (i + 1 >= ordered.length ? 0 : i + 1))
    }, 5000)
    return () => window.clearInterval(id)
  }, [open, autoAdvance, mode, ordered.length])

  if (!open) return null

  const current = ordered[index]

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col bg-slate-950 text-white"
      role="dialog"
      aria-modal="true"
      aria-label={t('boards.present.title')}
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-white/10 px-4 py-3">
        <div className="min-w-0">
          <p className="truncate text-sm text-white/70">{boardTitle}</p>
          <h2 className="text-lg font-semibold">{t('boards.present.title')}</h2>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            className={`rounded-md px-3 py-1.5 text-sm ${mode === 'slideshow' ? 'bg-white/20' : 'hover:bg-white/10'}`}
            onClick={() => setMode('slideshow')}
          >
            {t('boards.present.slideshow')}
          </button>
          <button
            type="button"
            className={`rounded-md px-3 py-1.5 text-sm ${mode === 'overview' ? 'bg-white/20' : 'hover:bg-white/10'}`}
            onClick={() => setMode('overview')}
          >
            {t('boards.present.overview')}
          </button>
          {mode === 'slideshow' ? (
            <label className="flex items-center gap-2 text-sm text-white/80">
              <input
                type="checkbox"
                checked={autoAdvance}
                onChange={(e) => setAutoAdvance(e.target.checked)}
              />
              {t('boards.present.autoAdvance')}
            </label>
          ) : null}
          <button
            type="button"
            aria-label={t('boards.present.exit')}
            className="rounded-md p-2 hover:bg-white/10"
            onClick={onClose}
          >
            <X className="size-5" aria-hidden />
          </button>
        </div>
      </div>

      {ordered.length === 0 ? (
        <div className="flex flex-1 items-center justify-center p-8 text-center text-white/70">
          {t('boards.present.empty')}
        </div>
      ) : mode === 'overview' ? (
        <div className="flex-1 overflow-auto p-4 sm:p-8">
          <div className="mx-auto grid max-w-6xl gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {ordered.map((p, i) => (
              <button
                key={p.id}
                type="button"
                className="rounded-lg border border-white/15 bg-white/5 p-4 text-start hover:bg-white/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-400"
                onClick={() => {
                  setIndex(i)
                  setMode('slideshow')
                }}
              >
                {sectionTitle(p) ? (
                  <p className="mb-1 text-xs uppercase tracking-wide text-indigo-300">{sectionTitle(p)}</p>
                ) : null}
                <p className="font-semibold">{p.title || t('boards.present.untitled')}</p>
                <p className="mt-1 line-clamp-3 text-sm text-white/70">{postBodyText(p)}</p>
              </button>
            ))}
          </div>
        </div>
      ) : (
        <div className="flex flex-1 flex-col items-center justify-center gap-6 p-6 sm:p-12">
          <div className="w-full max-w-3xl text-center">
            {current && sectionTitle(current) ? (
              <p className="mb-3 text-sm uppercase tracking-wide text-indigo-300">{sectionTitle(current)}</p>
            ) : null}
            <h3 className="text-3xl font-semibold sm:text-5xl">
              {current?.title || t('boards.present.untitled')}
            </h3>
            {current ? (
              <p className="mt-6 whitespace-pre-wrap text-lg text-white/85 sm:text-2xl">
                {postBodyText(current)}
              </p>
            ) : null}
            {current?.linkUrl ? (
              <p className="mt-4 break-all text-base text-indigo-300">{current.linkUrl}</p>
            ) : null}
          </div>
          <div className="flex items-center gap-4">
            <button
              type="button"
              aria-label={t('boards.present.prev')}
              className="rounded-full border border-white/20 p-3 hover:bg-white/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-400 disabled:opacity-40"
              disabled={index <= 0}
              onClick={() => setIndex((i) => Math.max(0, i - 1))}
            >
              <ChevronLeft className="size-6" aria-hidden />
            </button>
            <span className="min-w-16 text-center text-sm text-white/70" aria-live="polite">
              {t('boards.present.progress', { current: index + 1, total: ordered.length })}
            </span>
            <button
              type="button"
              aria-label={t('boards.present.next')}
              className="rounded-full border border-white/20 p-3 hover:bg-white/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-400 disabled:opacity-40"
              disabled={index >= ordered.length - 1}
              onClick={() => setIndex((i) => Math.min(ordered.length - 1, i + 1))}
            >
              <ChevronRight className="size-6" aria-hidden />
            </button>
            <button
              type="button"
              className="inline-flex items-center gap-1 rounded-md px-3 py-2 text-sm hover:bg-white/10"
              onClick={() => setMode('overview')}
            >
              <Maximize2 className="size-4" aria-hidden />
              {t('boards.present.overview')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

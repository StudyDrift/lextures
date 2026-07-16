import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { MoreVertical } from 'lucide-react'
import type { BoardPost, BoardSection } from '../../lib/boards-api'
import { midpointSortIndex } from '../../lib/boards-api'

type CardArrangeMenuProps = {
  post: BoardPost
  sections: BoardSection[]
  siblings: BoardPost[]
  canArrange: boolean
  onMoveToSection: (sectionId: string) => void
  onReorder: (sortIndex: number) => void
  onSetEventDate?: (iso: string | null) => void
  onSetCoords?: (lat: number, lng: number) => void
  showTimeline?: boolean
  showMap?: boolean
}

/** Keyboard / menu alternative to drag (AC-6). */
export function CardArrangeMenu({
  post,
  sections,
  siblings,
  canArrange,
  onMoveToSection,
  onReorder,
  onSetEventDate,
  onSetCoords,
  showTimeline,
  showMap,
}: CardArrangeMenuProps) {
  const { t } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const [latDraft, setLatDraft] = useState(String(post.lat ?? ''))
  const [lngDraft, setLngDraft] = useState(String(post.lng ?? ''))
  const [coordsOpen, setCoordsOpen] = useState(false)

  if (!canArrange) return null

  const ordered = [...siblings].sort((a, b) => a.sortIndex - b.sortIndex)
  const idx = ordered.findIndex((p) => p.id === post.id)

  function moveUp() {
    if (idx <= 0) return
    const before = ordered[idx - 2]?.sortIndex
    const after = ordered[idx - 1]?.sortIndex
    onReorder(midpointSortIndex(before, after))
    setOpen(false)
  }

  function moveDown() {
    if (idx < 0 || idx >= ordered.length - 1) return
    const before = ordered[idx + 1]?.sortIndex
    const after = ordered[idx + 2]?.sortIndex
    onReorder(midpointSortIndex(before, after))
    setOpen(false)
  }

  function saveCoords() {
    if (!onSetCoords) return
    const la = Number(latDraft)
    const ln = Number(lngDraft)
    if (!Number.isFinite(la) || !Number.isFinite(ln)) return
    onSetCoords(la, ln)
    setCoordsOpen(false)
    setOpen(false)
  }

  return (
    <div className="relative">
      <button
        type="button"
        aria-label={t('boards.arrange.menuAria')}
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className="rounded p-1 text-slate-500 hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:hover:bg-neutral-800"
      >
        <MoreVertical className="size-4" aria-hidden />
      </button>
      {open ? (
        <div
          role="menu"
          className="absolute end-0 z-20 mt-1 w-56 rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
        >
          <button
            type="button"
            role="menuitem"
            className="block w-full px-3 py-1.5 text-start text-sm hover:bg-slate-50 disabled:opacity-40 dark:hover:bg-neutral-800"
            onClick={moveUp}
            disabled={idx <= 0}
          >
            {t('boards.arrange.moveUp')}
          </button>
          <button
            type="button"
            role="menuitem"
            className="block w-full px-3 py-1.5 text-start text-sm hover:bg-slate-50 disabled:opacity-40 dark:hover:bg-neutral-800"
            onClick={moveDown}
            disabled={idx < 0 || idx >= ordered.length - 1}
          >
            {t('boards.arrange.moveDown')}
          </button>
          {sections.length > 0 ? (
            <div className="border-t border-slate-100 px-3 py-1.5 dark:border-neutral-800">
              <p className="mb-1 text-xs font-medium text-slate-500">{t('boards.arrange.moveToSection')}</p>
              {sections.map((s) => (
                <button
                  key={s.id}
                  type="button"
                  role="menuitem"
                  className="block w-full truncate px-1 py-1 text-start text-sm hover:bg-slate-50 disabled:opacity-40 dark:hover:bg-neutral-800"
                  onClick={() => {
                    onMoveToSection(s.id)
                    setOpen(false)
                  }}
                  disabled={post.sectionId === s.id}
                >
                  {s.title}
                </button>
              ))}
            </div>
          ) : null}
          {showTimeline && onSetEventDate ? (
            <div className="border-t border-slate-100 px-3 py-1.5 dark:border-neutral-800">
              <label className="block text-xs font-medium text-slate-500" htmlFor={`event-${post.id}`}>
                {t('boards.arrange.eventDate')}
              </label>
              <input
                id={`event-${post.id}`}
                type="date"
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm dark:border-neutral-700 dark:bg-neutral-800"
                defaultValue={post.eventDate ? post.eventDate.slice(0, 10) : ''}
                onChange={(e) => {
                  onSetEventDate(e.target.value || null)
                  setOpen(false)
                }}
              />
            </div>
          ) : null}
          {showMap && onSetCoords ? (
            <div className="border-t border-slate-100 px-3 py-1.5 dark:border-neutral-800">
              <p className="mb-1 text-xs font-medium text-slate-500">{t('boards.arrange.setCoords')}</p>
              {coordsOpen ? (
                <div className="flex flex-col gap-1.5">
                  <label className="text-xs text-slate-500" htmlFor={`lat-${post.id}`}>
                    {t('boards.arrange.latPrompt')}
                  </label>
                  <input
                    id={`lat-${post.id}`}
                    type="number"
                    step="any"
                    min={-90}
                    max={90}
                    value={latDraft}
                    onChange={(e) => setLatDraft(e.target.value)}
                    className="w-full rounded border border-slate-200 px-2 py-1 text-sm dark:border-neutral-700 dark:bg-neutral-800"
                  />
                  <label className="text-xs text-slate-500" htmlFor={`lng-${post.id}`}>
                    {t('boards.arrange.lngPrompt')}
                  </label>
                  <input
                    id={`lng-${post.id}`}
                    type="number"
                    step="any"
                    min={-180}
                    max={180}
                    value={lngDraft}
                    onChange={(e) => setLngDraft(e.target.value)}
                    className="w-full rounded border border-slate-200 px-2 py-1 text-sm dark:border-neutral-700 dark:bg-neutral-800"
                  />
                  <button
                    type="button"
                    className="rounded bg-indigo-600 px-2 py-1 text-xs font-medium text-white"
                    onClick={saveCoords}
                  >
                    {t('boards.arrange.saveCoords')}
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  role="menuitem"
                  className="block w-full px-1 py-1 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                  onClick={() => {
                    setLatDraft(String(post.lat ?? ''))
                    setLngDraft(String(post.lng ?? ''))
                    setCoordsOpen(true)
                  }}
                >
                  {t('boards.arrange.editCoords')}
                </button>
              )}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

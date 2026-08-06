import { useEffect, useId, useRef, useState } from 'react'
import { ChevronDown, Image, Kanban, LayoutGrid, LayoutList, Table } from 'lucide-react'

import type { CourseCatalogView } from '../../lib/course-catalog-types'

type Props = {
  value: CourseCatalogView
  onChange: (view: CourseCatalogView) => void
}

const VIEW_OPTIONS: { id: CourseCatalogView; label: string; hint: string; icon: typeof LayoutGrid }[] = [
  {
    id: 'cards',
    label: 'Cards',
    hint: 'Visual grid with course cards',
    icon: LayoutGrid,
  },
  {
    id: 'list',
    label: 'List',
    hint: 'Compact rows with thumbnails',
    icon: LayoutList,
  },
  {
    id: 'gallery',
    label: 'Gallery',
    hint: 'Cover-focused tiles with minimal text',
    icon: Image,
  },
  {
    id: 'table',
    label: 'Compact table',
    hint: 'Dense rows with title, status, and term',
    icon: Table,
  },
  {
    id: 'status',
    label: 'By status',
    hint: 'Kanban board with todo, in progress, done, and hidden',
    icon: Kanban,
  },
]

export function CourseCatalogViewMenu({ value, onChange }: Props) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()
  const activeOption = VIEW_OPTIONS.find((option) => option.id === value) ?? VIEW_OPTIONS[0]
  const ActiveIcon = activeOption.icon

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  return (
    <div ref={rootRef} className="relative block w-full text-start sm:inline-block sm:w-auto">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={`View courses as ${activeOption.label}. Open menu to change catalog layout.`}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-border-default bg-surface-raised px-4 py-2.5 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:border-border-strong hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:border-border-default dark:hover:bg-surface-overlay sm:w-auto"
      >
        <ActiveIcon className="h-4 w-4 shrink-0" aria-hidden />
        <span>View</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Course catalog view"
          className="absolute start-0 end-0 z-50 mt-1 min-w-0 overflow-hidden rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg shadow-slate-900/10 sm:left-auto sm:end-0 sm:min-w-[16rem] dark:border-border-default dark:bg-surface-overlay dark:shadow-black/40"
        >
          {VIEW_OPTIONS.map((option) => {
            const Icon = option.icon
            return (
              <button
                key={option.id}
                type="button"
                role="menuitemradio"
                aria-checked={value === option.id}
                onClick={() => {
                  onChange(option.id)
                  setOpen(false)
                }}
                className={[
                  'flex w-full items-start gap-2.5 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:hover:bg-neutral-700',
                  value === option.id ? 'bg-indigo-50 dark:bg-surface-overlay' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
              >
                <Icon className="mt-0.5 h-4 w-4 shrink-0 text-fg-muted" aria-hidden />
                <span className="flex min-w-0 flex-col gap-0.5">
                  <span className="font-semibold text-slate-950 dark:text-fg-default">{option.label}</span>
                  <span className="text-xs text-fg-muted">{option.hint}</span>
                </span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

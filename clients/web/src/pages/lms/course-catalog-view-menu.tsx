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
        className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800 sm:w-auto"
      >
        <ActiveIcon className="h-4 w-4 shrink-0" aria-hidden />
        <span>View</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Course catalog view"
          className="absolute start-0 end-0 z-50 mt-1 min-w-0 overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 sm:left-auto sm:end-0 sm:min-w-[16rem] dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40"
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
                  'flex w-full items-start gap-2.5 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:hover:bg-neutral-700',
                  value === option.id ? 'bg-indigo-50 dark:bg-neutral-800' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
              >
                <Icon className="mt-0.5 h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
                <span className="flex min-w-0 flex-col gap-0.5">
                  <span className="font-semibold text-slate-950 dark:text-neutral-100">{option.label}</span>
                  <span className="text-xs text-slate-500 dark:text-neutral-400">{option.hint}</span>
                </span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

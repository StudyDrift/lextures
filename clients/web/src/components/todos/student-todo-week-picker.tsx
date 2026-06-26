import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { CalendarRange, Check, ChevronDown } from 'lucide-react'
import {
  formatWeekOffsetLabel,
  formatWeekOffsetsButtonLabel,
  formatWeekRangeShort,
  listStudentTodoWeekOptions,
  normalizeWeekOffsets,
  type StudentTodoWeekOffset,
} from '../../lib/student-todo-week'

type StudentTodoWeekPickerProps = {
  value: StudentTodoWeekOffset[]
  onChange: (offsets: StudentTodoWeekOffset[]) => void
  /** Anchor for relative week labels and ranges (defaults to now). */
  now?: Date
}

export function StudentTodoWeekPicker({ value, onChange, now = new Date() }: StudentTodoWeekPickerProps) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  const selectedOffsets = useMemo(() => normalizeWeekOffsets(value), [value])
  const options = useMemo(() => listStudentTodoWeekOptions(now), [now])
  const buttonLabel = useMemo(() => formatWeekOffsetsButtonLabel(selectedOffsets, now), [selectedOffsets, now])

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  function toggleOffset(offset: StudentTodoWeekOffset) {
    const isSelected = selectedOffsets.includes(offset)
    if (isSelected && selectedOffsets.length === 1) return
    const next = isSelected
      ? selectedOffsets.filter((o) => o !== offset)
      : normalizeWeekOffsets([...selectedOffsets, offset])
    onChange(next)
  }

  return (
    <div ref={rootRef} className="relative block w-full text-start sm:inline-block sm:w-auto">
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen((prev) => !prev)}
        className="inline-flex w-full min-w-[12rem] items-center justify-between gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-start shadow-sm transition-[background-color,border-color,box-shadow,transform] duration-150 ease-out hover:border-slate-300 hover:bg-slate-50 active:scale-[0.98] dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800 sm:w-auto"
      >
        <span className="flex min-w-0 items-center gap-2">
          <CalendarRange className="h-4 w-4 shrink-0 text-indigo-600 dark:text-indigo-400" aria-hidden />
          <span className="min-w-0">
            <span className="block truncate text-sm font-semibold text-slate-900 dark:text-neutral-100">
              {buttonLabel.title}
            </span>
            <span className="block truncate text-[11px] text-slate-500 dark:text-neutral-400">
              {buttonLabel.subtitle}
            </span>
          </span>
        </span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-slate-500 transition-transform duration-150 dark:text-neutral-400 ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open ? (
        <ul
          id={menuId}
          role="listbox"
          aria-label="Select weeks"
          aria-multiselectable="true"
          className="absolute end-0 top-[calc(100%+0.375rem)] z-50 max-h-72 w-full min-w-[14rem] overflow-y-auto rounded-xl border border-slate-200 bg-white p-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40 sm:w-auto"
        >
          {options.map((option) => {
            const active = selectedOffsets.includes(option.offset)
            const isLastSelected = active && selectedOffsets.length === 1
            return (
              <li key={option.offset} role="presentation">
                <button
                  type="button"
                  role="option"
                  aria-selected={active}
                  disabled={isLastSelected}
                  onClick={() => toggleOffset(option.offset)}
                  className={[
                    'flex w-full items-center justify-between gap-2 rounded-lg px-2.5 py-2 text-start transition-colors duration-150',
                    active
                      ? 'bg-indigo-50 text-indigo-950 dark:bg-indigo-950/50 dark:text-indigo-100'
                      : 'text-slate-800 hover:bg-slate-50 dark:text-neutral-100 dark:hover:bg-neutral-800',
                    isLastSelected ? 'cursor-default opacity-90' : '',
                  ].join(' ')}
                >
                  <span className="min-w-0">
                    <span className="block text-sm font-medium">{formatWeekOffsetLabel(option.offset)}</span>
                    <span className="block text-[11px] text-slate-500 dark:text-neutral-400">
                      {formatWeekRangeShort(option)}
                    </span>
                  </span>
                  {active ? <Check className="h-4 w-4 shrink-0 text-indigo-600 dark:text-indigo-400" aria-hidden /> : null}
                </button>
              </li>
            )
          })}
        </ul>
      ) : null}
    </div>
  )
}
import { useEffect, useId, useMemo, useState } from 'react'
import { detectBrowserTimezone, formatUtcOffsetLabel } from '../../lib/format'
import { fetchTimezones, type TimezoneEntry } from '../../lib/timezone-api'

type Props = {
  value: string | null
  onChange: (value: string) => void
  disabled?: boolean
  label?: string
  showDetectedHint?: boolean
}

const fieldClass =
  'w-full rounded-lg border border-stone-300 bg-white px-2 py-1.5 text-sm text-stone-900 shadow-sm outline-none transition placeholder:text-stone-400 focus:border-teal-700 focus:ring-2 focus:ring-teal-700/15 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-teal-500 dark:focus:ring-teal-500/20'

function formatTimezoneLabel(id: string): string {
  return id.replace(/_/g, ' ')
}

export function TimezoneSelector({
  value,
  onChange,
  disabled,
  label = 'Time zone',
  showDetectedHint = true,
}: Props) {
  const listId = useId()
  const [entries, setEntries] = useState<TimezoneEntry[]>([])
  const [loadError, setLoadError] = useState<string | null>(null)
  const [query, setQuery] = useState('')
  const [listOpen, setListOpen] = useState(false)
  const detected = useMemo(() => detectBrowserTimezone(), [])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const list = await fetchTimezones()
        if (!cancelled) setEntries(list)
      } catch {
        if (!cancelled) setLoadError('Could not load time zones.')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const selected = value?.trim() || detected

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    const base = q ? entries.filter((e) => e.id.toLowerCase().includes(q)) : entries
    const limited = base.slice(0, q ? 200 : 80)
    if (!q && selected) {
      const idx = limited.findIndex((e) => e.id === selected)
      if (idx > 0) {
        const copy = [...limited]
        const [match] = copy.splice(idx, 1)
        copy.unshift(match)
        return copy
      }
    }
    return limited
  }, [entries, query, selected])

  const showList = listOpen && filtered.length > 0

  function commitQuery() {
    const trimmed = query.trim()
    if (trimmed) {
      onChange(trimmed)
      setQuery('')
    }
    setListOpen(false)
  }

  return (
    <div className="space-y-2">
      <label htmlFor={listId} className="block text-sm font-medium text-stone-800 dark:text-neutral-200">
        {label}
      </label>
      {showDetectedHint && (
        <p className="text-xs text-stone-600 dark:text-neutral-400">
          We detected your time zone as <span className="font-medium text-stone-800 dark:text-neutral-200">{detected}</span>.
          Change it below if that is not correct.
        </p>
      )}
      <input
        id={listId}
        type="search"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={showList}
        aria-controls={`${listId}-listbox`}
        disabled={disabled}
        value={query || selected}
        onChange={(e) => {
          setQuery(e.target.value)
          setListOpen(true)
        }}
        onFocus={() => setListOpen(true)}
        onBlur={() => {
          commitQuery()
        }}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            setQuery('')
            setListOpen(false)
            e.currentTarget.blur()
          }
        }}
        className={fieldClass}
        placeholder="Search time zones…"
      />
      {loadError && (
        <p className="text-xs text-rose-600 dark:text-rose-400" aria-live="polite">
          {loadError}
        </p>
      )}
      {showList && (
        <ul
          id={`${listId}-listbox`}
          role="listbox"
          className="max-h-48 overflow-y-auto rounded-lg border border-stone-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-950"
        >
          {!query.trim() && entries.length > 80 && (
            <li className="border-b border-stone-100 px-2.5 py-1.5 text-xs text-stone-500 dark:border-neutral-800 dark:text-neutral-500">
              Showing the first 80 zones — type to search the full list.
            </li>
          )}
          {filtered.map((e) => {
            const isSelected = e.id === selected
            return (
              <li key={e.id} role="option" aria-selected={isSelected}>
                <button
                  type="button"
                  disabled={disabled}
                  className={`flex w-full items-center justify-between gap-3 px-2.5 py-1.5 text-left text-sm text-stone-900 hover:bg-stone-100 dark:text-neutral-100 dark:hover:bg-neutral-800 ${
                    isSelected ? 'bg-teal-50 font-medium dark:bg-teal-950/50' : ''
                  }`}
                  onMouseDown={(ev) => ev.preventDefault()}
                  onClick={() => {
                    onChange(e.id)
                    setQuery('')
                    setListOpen(false)
                  }}
                >
                  <span className="min-w-0 truncate">{formatTimezoneLabel(e.id)}</span>
                  <span className="shrink-0 font-mono text-xs text-stone-500 dark:text-neutral-400">
                    {formatUtcOffsetLabel(e.offsetMinutes)}
                  </span>
                </button>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

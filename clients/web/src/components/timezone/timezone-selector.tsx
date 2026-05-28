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

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return entries.slice(0, 200)
    return entries.filter((e) => e.id.toLowerCase().includes(q)).slice(0, 200)
  }, [entries, query])

  const selected = value?.trim() || detected

  return (
    <div className="space-y-2">
      <label htmlFor={listId} className="block text-sm font-medium text-stone-800 dark:text-neutral-200">
        {label}
      </label>
      {showDetectedHint && (
        <p className="text-xs text-stone-600 dark:text-neutral-400">
          We detected your time zone as <span className="font-medium">{detected}</span>. Change it below if that is
          not correct.
        </p>
      )}
      <input
        id={listId}
        type="search"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={filtered.length > 0}
        aria-controls={`${listId}-listbox`}
        disabled={disabled}
        value={query || selected}
        onChange={(e) => setQuery(e.target.value)}
        onBlur={() => {
          if (query.trim()) {
            onChange(query.trim())
            setQuery('')
          }
        }}
        className="w-full rounded-lg border border-stone-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
        placeholder="Search time zones…"
      />
      {loadError && (
        <p className="text-xs text-rose-600 dark:text-rose-400" aria-live="polite">
          {loadError}
        </p>
      )}
      <ul
        id={`${listId}-listbox`}
        role="listbox"
        className="max-h-48 overflow-y-auto rounded-lg border border-stone-200 dark:border-neutral-700"
      >
        {filtered.map((e) => (
          <li key={e.id} role="option" aria-selected={e.id === selected}>
            <button
              type="button"
              disabled={disabled}
              className="flex w-full items-center justify-between px-3 py-2 text-left text-sm hover:bg-stone-100 dark:hover:bg-neutral-800"
              onClick={() => {
                onChange(e.id)
                setQuery('')
              }}
            >
              <span>{e.id.replace(/_/g, ' ')}</span>
              <span className="ml-2 font-mono text-xs text-stone-500 dark:text-neutral-400">
                {formatUtcOffsetLabel(e.offsetMinutes)}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}

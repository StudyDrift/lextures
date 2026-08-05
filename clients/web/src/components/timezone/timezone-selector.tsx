import { useEffect, useId, useMemo, useRef, useState } from 'react'
import {
  COURSE_TIMEZONE_LOCAL,
  detectBrowserTimezone,
  formatUtcOffsetLabel,
} from '../../lib/format'
import { fetchTimezones, type TimezoneEntry } from '../../lib/timezone-api'

export { COURSE_TIMEZONE_LOCAL }

export type TimezoneSelectorProps = {
  value: string | null
  onChange: (value: string | null) => void
  disabled?: boolean
  label?: string
  showDetectedHint?: boolean
  /**
   * When true, value may be cleared (null). Course settings typically leave this false
   * but still start from null until the instructor picks a zone.
   */
  allowClear?: boolean
  /**
   * Course mode: pin "Learner local time" (11:59pm for each learner) and treat
   * empty as unset rather than browser-detected.
   */
  courseMode?: boolean
  /** Optional test id prefix. */
  'data-testid'?: string
}

const fieldClass =
  'w-full rounded-lg border border-stone-300 bg-white px-2 py-1.5 text-sm text-stone-900 shadow-sm outline-none transition-[background-color,color,border-color] placeholder:text-stone-400 focus:border-teal-700 focus:ring-2 focus:ring-teal-700/15 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-teal-500 dark:focus:ring-teal-500/20'

function formatTimezoneLabel(id: string): string {
  if (id === COURSE_TIMEZONE_LOCAL) return 'Learner local time'
  return id.replace(/_/g, ' ')
}

function offsetForId(entries: TimezoneEntry[], id: string): number | null {
  if (id === COURSE_TIMEZONE_LOCAL) return null
  const hit = entries.find((e) => e.id === id)
  return hit ? hit.offsetMinutes : null
}

type Preset = {
  id: string
  title: string
  subtitle: string
}

export function TimezoneSelector({
  value,
  onChange,
  disabled,
  label = 'Time zone',
  showDetectedHint = true,
  allowClear = false,
  courseMode = false,
  'data-testid': testId,
}: TimezoneSelectorProps) {
  const listId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const [entries, setEntries] = useState<TimezoneEntry[]>([])
  const [loadError, setLoadError] = useState<string | null>(null)
  const [query, setQuery] = useState('')
  const [listOpen, setListOpen] = useState(false)
  const [highlight, setHighlight] = useState(0)
  const detected = useMemo(() => detectBrowserTimezone(), [])

  const selected = value?.trim() || null

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

  const presets = useMemo((): Preset[] => {
    const out: Preset[] = []
    if (courseMode) {
      out.push({
        id: COURSE_TIMEZONE_LOCAL,
        title: 'Learner local time',
        subtitle: '11:59 PM means 11:59 PM in each learner’s own time zone',
      })
    }
    out.push({
      id: 'UTC',
      title: 'UTC',
      subtitle: 'One global clock (Coordinated Universal Time)',
    })
    if (detected && detected !== 'UTC') {
      out.push({
        id: detected,
        title: formatTimezoneLabel(detected),
        subtitle: 'Detected from this device',
      })
    }
    return out
  }, [courseMode, detected])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    const base = q
      ? entries.filter(
          (e) =>
            e.id.toLowerCase().includes(q) ||
            formatTimezoneLabel(e.id).toLowerCase().includes(q),
        )
      : entries
    // When not searching, keep the list short; search uses the full catalog.
    return base.slice(0, q ? 200 : 80)
  }, [entries, query])

  /** Flattened option ids for keyboard nav: presets (matching query) then filtered. */
  const optionIds = useMemo(() => {
    const q = query.trim().toLowerCase()
    const presetIds = presets
      .filter((p) => {
        if (!q) return true
        return (
          p.id.toLowerCase().includes(q) ||
          p.title.toLowerCase().includes(q) ||
          p.subtitle.toLowerCase().includes(q)
        )
      })
      .map((p) => p.id)
    const rest = filtered.map((e) => e.id).filter((id) => !presetIds.includes(id))
    return [...presetIds, ...rest]
  }, [presets, filtered, query])

  const showList = listOpen && (optionIds.length > 0 || Boolean(query.trim()))

  useEffect(() => {
    setHighlight(0)
  }, [query, listOpen])

  function select(id: string) {
    onChange(id)
    setQuery('')
    setListOpen(false)
    inputRef.current?.blur()
  }

  function clearSelection() {
    onChange(null)
    setQuery('')
    setListOpen(false)
  }

  function onInputKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Escape') {
      setQuery('')
      setListOpen(false)
      e.currentTarget.blur()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setListOpen(true)
      setHighlight((h) => Math.min(h + 1, Math.max(optionIds.length - 1, 0)))
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      setHighlight((h) => Math.max(h - 1, 0))
      return
    }
    if (e.key === 'Enter' && listOpen && optionIds[highlight]) {
      e.preventDefault()
      select(optionIds[highlight])
    }
  }

  const selectedLabel = selected ? formatTimezoneLabel(selected) : null
  const selectedOffset =
    selected && selected !== COURSE_TIMEZONE_LOCAL ? offsetForId(entries, selected) : null
  const isLocal = selected === COURSE_TIMEZONE_LOCAL

  return (
    <div className="space-y-2" data-testid={testId}>
      <label htmlFor={listId} className="block text-sm font-medium text-stone-800 dark:text-neutral-200">
        {label}
      </label>
      {showDetectedHint && !courseMode && (
        <p className="text-xs text-stone-600 dark:text-neutral-400">
          We detected your time zone as{' '}
          <span className="font-medium text-stone-800 dark:text-neutral-200">{detected}</span>.
          Change it below if that is not correct.
        </p>
      )}

      {selected && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg border border-stone-200 bg-stone-50 px-2.5 py-2 dark:border-neutral-700 dark:bg-neutral-900/60">
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium text-stone-900 dark:text-neutral-100">
              {selectedLabel}
            </p>
            <p className="text-xs text-stone-500 dark:text-neutral-400">
              {isLocal
                ? 'Due times apply in each learner’s own time zone'
                : selectedOffset != null
                  ? formatUtcOffsetLabel(selectedOffset)
                  : selected === 'UTC'
                    ? 'UTC+0'
                    : selected}
            </p>
          </div>
          {(allowClear || courseMode) && (
            <button
              type="button"
              disabled={disabled}
              onClick={() => clearSelection()}
              className="shrink-0 rounded-md px-2 py-1 text-xs font-medium text-stone-600 hover:bg-stone-200 hover:text-stone-900 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800 dark:hover:text-neutral-50"
              data-testid={testId ? `${testId}-clear` : undefined}
            >
              Clear
            </button>
          )}
          <button
            type="button"
            disabled={disabled}
            onClick={() => {
              setQuery('')
              setListOpen(true)
              inputRef.current?.focus()
            }}
            className="shrink-0 rounded-md bg-white px-2 py-1 text-xs font-medium text-teal-800 ring-1 ring-stone-200 hover:bg-stone-100 disabled:opacity-50 dark:bg-neutral-950 dark:text-teal-300 dark:ring-neutral-600 dark:hover:bg-neutral-800"
          >
            Change
          </button>
        </div>
      )}

      <input
        ref={inputRef}
        id={listId}
        type="search"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={showList}
        aria-controls={`${listId}-listbox`}
        aria-activedescendant={showList && optionIds[highlight] ? `${listId}-opt-${optionIds[highlight]}` : undefined}
        disabled={disabled}
        value={query}
        onChange={(e) => {
          setQuery(e.target.value)
          setListOpen(true)
        }}
        onFocus={() => {
          setListOpen(true)
        }}
        onBlur={() => {
          // Delay so option mousedown/click can fire first.
          window.setTimeout(() => {
            setListOpen(false)
            setQuery('')
          }, 150)
        }}
        onKeyDown={onInputKeyDown}
        className={fieldClass}
        placeholder={selected ? 'Search for a different time zone…' : 'Search time zones (e.g. UTC, New York)…'}
        autoComplete="off"
        data-testid={testId ? `${testId}-search` : undefined}
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
          className="max-h-56 overflow-y-auto rounded-lg border border-stone-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-950"
        >
          {optionIds.length === 0 && (
            <li className="px-2.5 py-2 text-sm text-stone-500 dark:text-neutral-400">
              No time zones match “{query.trim()}”.
            </li>
          )}
          {optionIds.map((id, index) => {
            const preset = presets.find((p) => p.id === id)
            const entry = entries.find((e) => e.id === id)
            const isSelected = id === selected
            const isActive = index === highlight
            return (
              <li key={id} role="option" aria-selected={isSelected} id={`${listId}-opt-${id}`}>
                <button
                  type="button"
                  disabled={disabled}
                  className={`flex w-full items-start justify-between gap-3 px-2.5 py-2 text-left text-sm text-stone-900 hover:bg-stone-100 dark:text-neutral-100 dark:hover:bg-neutral-800 ${
                    isSelected ? 'bg-teal-50 font-medium dark:bg-teal-950/50' : ''
                  } ${isActive && !isSelected ? 'bg-stone-50 dark:bg-neutral-900' : ''}`}
                  onMouseDown={(ev) => ev.preventDefault()}
                  onMouseEnter={() => setHighlight(index)}
                  onClick={() => select(id)}
                >
                  <span className="min-w-0">
                    <span className="block truncate">
                      {preset ? preset.title : formatTimezoneLabel(id)}
                    </span>
                    {preset && (
                      <span className="mt-0.5 block text-xs font-normal text-stone-500 dark:text-neutral-400">
                        {preset.subtitle}
                      </span>
                    )}
                  </span>
                  {id !== COURSE_TIMEZONE_LOCAL && (
                    <span className="shrink-0 font-mono text-xs text-stone-500 dark:text-neutral-400">
                      {entry
                        ? formatUtcOffsetLabel(entry.offsetMinutes)
                        : id === 'UTC'
                          ? 'UTC+0'
                          : ''}
                    </span>
                  )}
                </button>
              </li>
            )
          })}
          {!query.trim() && entries.length > 80 && (
            <li className="border-t border-stone-100 px-2.5 py-1.5 text-xs text-stone-500 dark:border-neutral-800 dark:text-neutral-500">
              Showing the first 80 zones — type to search the full list.
            </li>
          )}
        </ul>
      )}
    </div>
  )
}

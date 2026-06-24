import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  ArrowDown,
  ArrowUp,
  BookOpen,
  Clock,
  FileText,
  Layers,
  Navigation,
  Search,
  Users,
  Zap,
  Stars,
} from 'lucide-react'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import { usePermissions } from '../../context/use-permissions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { usePlatformScimEnabled } from '../../hooks/use-platform-scim-enabled'
import { PERM_RBAC_MANAGE } from '../../lib/rbac-api'
import {
  capSearchResults,
  filterSearchItems,
  SEARCH_GROUP_LABEL,
  sortSearchItems,
  buildLocalSearchCandidates,
  type SearchGroup,
  type SearchListItem,
} from '../../lib/build-search-items'
import { buildCommandPaletteGoToItems } from '../../lib/command-palette-go-to'
import { buildSearchHubItems } from '../../lib/search-hub'
import {
  applyCoursePickerSelection,
  parseCoursePickerState,
  parseSearchQuery,
} from '../../lib/search-query-parse'
import { listSearchRecents, recordSearchRecent } from '../../lib/search-recents'
import {
  fetchSearchIndex,
  fetchSearchQuery,
  queryResultsToSearchItems,
  type SearchCourseItem,
} from '../../lib/search-api'
import { mergeCoursesWithNavFeatures } from '../../lib/search-course-features'
import { LiveRegion } from '../a11y/live-region'
import { useCommandPalette } from './use-command-palette'

const GROUP_ICONS: Record<SearchGroup, typeof BookOpen> = {
  recent: Clock,
  goto: Navigation,
  action: Zap,
  course: BookOpen,
  person: Users,
  page: FileText,
  content: Layers,
  ai: Stars,
}

const SEARCH_DEBOUNCE_MS = 200
const EMPTY_SERVER_ITEMS: SearchListItem[] = []

function currentCourseCodeFromPath(pathname: string): string | null {
  const m = /^\/courses\/([^/]+)/.exec(pathname)
  if (!m?.[1]) return null
  try {
    return decodeURIComponent(m[1]).trim() || null
  } catch {
    return m[1].trim() || null
  }
}

export function CommandPaletteDialog() {
  const { close } = useCommandPalette()
  const navigate = useNavigate()
  const location = useLocation()
  const { allows } = usePermissions()
  const canManageRbac = allows(PERM_RBAC_MANAGE)
  const { ragNotebookEnabled } = usePlatformFeatures()
  const { scimEnabled } = usePlatformScimEnabled(canManageRbac)
  const globalSearchOptions = useMemo(
    () => ({
      scimEnabled: canManageRbac && scimEnabled,
      ragNotebookEnabled,
    }),
    [canManageRbac, scimEnabled, ragNotebookEnabled],
  )
  const navFeatures = useCourseNavFeatures()
  const inputRef = useRef<HTMLInputElement>(null)
  const dialogRef = useRef<HTMLDivElement>(null)
  const activeRowRef = useRef<HTMLButtonElement | null>(null)

  const [query, setQuery] = useState('')
  const [cursor, setCursor] = useState(0)
  const [courses, setCourses] = useState<SearchCourseItem[]>([])
  const [loadState, setLoadState] = useState<'loading' | 'error' | 'ready'>('loading')
  const [serverItems, setServerItems] = useState<SearchListItem[]>([])
  const [serverLoading, setServerLoading] = useState(false)

  const currentCourseCode = useMemo(
    () => currentCourseCodeFromPath(location.pathname),
    [location.pathname],
  )

  const coursesForSearch = useMemo(
    () => mergeCoursesWithNavFeatures(courses, currentCourseCode, navFeatures),
    [courses, currentCourseCode, navFeatures],
  )

  useEffect(() => {
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prevOverflow
    }
  }, [])

  useEffect(() => {
    void fetchSearchIndex()
      .then((data) => {
        setCourses(Array.isArray(data.courses) ? data.courses : [])
        setLoadState('ready')
      })
      .catch(() => {
        setLoadState('error')
        setCourses([])
      })
  }, [])

  useEffect(() => {
    const t = window.setTimeout(() => inputRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        close()
        return
      }
      if (e.key !== 'Tab') return
      const dialog = dialogRef.current
      if (!dialog) return
      const focusable = Array.from(
        dialog.querySelectorAll<HTMLElement>(
          'button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])',
        ),
      ).filter((el) => !el.closest('[aria-hidden="true"]'))
      if (focusable.length === 0) return
      const first = focusable[0]!
      const last = focusable[focusable.length - 1]!
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault()
          last.focus()
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault()
          first.focus()
        }
      }
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [close])

  const parsed = useMemo(() => parseSearchQuery(query), [query])
  const coursePicker = useMemo(() => parseCoursePickerState(query), [query])
  const isHubMode = parsed.raw === '' && !coursePicker.active
  const serverSearchActive = parsed.text.length >= 2
  const activeServerItems = serverSearchActive ? serverItems : EMPTY_SERVER_ITEMS
  const activeServerLoading = serverSearchActive && serverLoading

  const pickerCourses = useMemo(() => {
    if (!coursePicker.active) return []
    const f = coursePicker.filter
    return coursesForSearch.filter((c) => {
      if (!f) return true
      const code = c.courseCode.toLowerCase()
      const title = c.title.toLowerCase()
      return code.includes(f) || title.includes(f)
    })
  }, [coursePicker, coursesForSearch])

  useEffect(() => {
    if (!serverSearchActive) {
      return
    }
    let cancelled = false
    const timer = window.setTimeout(() => {
      if (cancelled) return
      setServerLoading(true)
      const types = parsed.types
      const typeParam = types
        ? [...types].filter((t) => t === 'course' || t === 'person' || t === 'content').join(',')
        : undefined
      void fetchSearchQuery({
        q: parsed.text,
        scope: parsed.scopeCourseCode,
        types: typeParam,
      })
        .then((data) => {
          if (cancelled) return
          setServerItems(queryResultsToSearchItems(data.groups))
        })
        .catch(() => {
          if (cancelled) return
          setServerItems([])
        })
        .finally(() => {
          if (!cancelled) setServerLoading(false)
        })
    }, SEARCH_DEBOUNCE_MS)
    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [serverSearchActive, parsed.text, parsed.scopeCourseCode, parsed.types])

  const pinnedCourseCode = isHubMode ? currentCourseCode : parsed.scopeCourseCode

  const filtered = useMemo(() => {
    if (isHubMode) {
      return capSearchResults(
        buildSearchHubItems(coursesForSearch, allows, currentCourseCode, globalSearchOptions),
        {
          hubMode: true,
          pinnedCourseCode,
        },
      )
    }

    const localCandidates = buildLocalSearchCandidates(
      coursesForSearch,
      allows,
      parsed,
      globalSearchOptions,
    )
    const localFiltered = filterSearchItems(localCandidates, query, {
      currentCourseCode,
    })
    const go = buildCommandPaletteGoToItems(query, coursesForSearch, allows)

    const seen = new Set<string>()
    const merged: SearchListItem[] = []
    for (const it of [...go, ...activeServerItems, ...localFiltered]) {
      if (seen.has(it.id)) continue
      seen.add(it.id)
      merged.push(it)
    }

    return capSearchResults(sortSearchItems(merged), { pinnedCourseCode })
  }, [
    isHubMode,
    coursesForSearch,
    allows,
    currentCourseCode,
    parsed,
    query,
    activeServerItems,
    pinnedCourseCode,
    globalSearchOptions,
  ])

  const listLength = coursePicker.active ? pickerCourses.length : filtered.length
  const safeIndex = listLength === 0 ? 0 : Math.min(cursor, Math.max(0, listLength - 1))

  useLayoutEffect(() => {
    activeRowRef.current?.scrollIntoView({ block: 'nearest' })
  }, [safeIndex, filtered, pickerCourses, coursePicker.active])

  const selectPickerCourse = (course: SearchCourseItem) => {
    setQuery(applyCoursePickerSelection(query, coursePicker.atIndex, course.courseCode))
    setCursor(0)
    window.setTimeout(() => inputRef.current?.focus(), 0)
  }

  const go = (item: SearchListItem) => {
    if (item.id === 'global:ask-ai') {
      const t = query.trim()
      navigate(t ? `/ai?q=${encodeURIComponent(t)}` : '/ai')
    } else {
      navigate(item.path)
    }
    if (item.group === 'recent') {
      const stored = listSearchRecents().find((r) => `recent:${r.id}` === item.id)
      if (stored) {
        recordSearchRecent({
          id: stored.id,
          group: stored.group,
          title: stored.title,
          subtitle: stored.subtitle,
          path: stored.path,
          haystack: '',
        })
      }
    } else {
      recordSearchRecent(item)
    }
    close()
  }

  const onInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (coursePicker.active) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        if (pickerCourses.length === 0) return
        setCursor((c) => {
          const at = Math.min(c, pickerCourses.length - 1)
          return (at + 1) % pickerCourses.length
        })
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        if (pickerCourses.length === 0) return
        setCursor((c) => {
          const at = Math.min(c, pickerCourses.length - 1)
          return (at - 1 + pickerCourses.length) % pickerCourses.length
        })
        return
      }
      if (e.key === 'Enter') {
        e.preventDefault()
        const course = pickerCourses[safeIndex]
        if (course) selectPickerCourse(course)
        return
      }
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault()
      if (filtered.length === 0) return
      setCursor((c) => {
        const at = Math.min(c, filtered.length - 1)
        return (at + 1) % filtered.length
      })
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      if (filtered.length === 0) return
      setCursor((c) => {
        const at = Math.min(c, filtered.length - 1)
        return (at - 1 + filtered.length) % filtered.length
      })
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      const item = filtered[safeIndex]
      if (item) go(item)
    }
  }

  const resultsAnnouncement =
    loadState === 'ready'
      ? coursePicker.active
        ? pickerCourses.length === 0
          ? 'No matching courses'
          : `${pickerCourses.length} ${pickerCourses.length === 1 ? 'course' : 'courses'}`
        : filtered.length === 0
          ? activeServerLoading
            ? 'Searching'
            : 'No results'
          : `${filtered.length} ${filtered.length === 1 ? 'result' : 'results'}`
      : ''

  const palette = (
    <div
      className="fixed inset-0 z-[100] flex items-start justify-center px-3 pt-12 pb-[env(safe-area-inset-bottom)] sm:px-4 sm:pt-[min(12vh,8rem)]"
      role="dialog"
      aria-modal="true"
      aria-label="Command Palette"
      ref={dialogRef}
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default bg-slate-950/55 backdrop-blur-md dark:bg-neutral-950/75"
        aria-label="Close search"
        tabIndex={-1}
        onClick={() => close()}
      />
      <div className="relative z-10 w-full max-w-xl overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-2xl shadow-slate-900/20 dark:border-neutral-700 dark:bg-neutral-900 dark:shadow-black/50">
        <div className="flex items-center gap-3 border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <Search className="h-5 w-5 shrink-0 text-slate-400 dark:text-neutral-500" aria-hidden />
          <input
            ref={inputRef}
            type="search"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setCursor(0)
            }}
            onKeyDown={onInputKeyDown}
            placeholder="Search courses, content, people… (@course to scope)"
            aria-label="Search"
            className="min-w-0 flex-1 border-0 bg-transparent text-base text-slate-900 outline-none placeholder:text-slate-600 dark:text-neutral-100 dark:placeholder:text-neutral-400"
            autoComplete="off"
            autoCorrect="off"
            spellCheck={false}
            aria-controls={coursePicker.active ? 'command-palette-course-picker' : 'command-palette-results'}
            aria-activedescendant={
              coursePicker.active
                ? pickerCourses[safeIndex]
                  ? `cmd-course-${encodeURIComponent(pickerCourses[safeIndex]!.courseCode)}`
                  : undefined
                : filtered[safeIndex]
                  ? `cmd-result-${filtered[safeIndex].id}`
                  : undefined
            }
          />
          <kbd className="hidden shrink-0 rounded-md border border-slate-200 bg-slate-50 px-2 py-1 font-mono text-[11px] text-slate-600 sm:inline dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
            esc
          </kbd>
        </div>

        <LiveRegion>{resultsAnnouncement}</LiveRegion>

        <div
          id={coursePicker.active ? 'command-palette-course-picker' : 'command-palette-results'}
          role="listbox"
          aria-label={coursePicker.active ? 'Choose a course' : 'Search results'}
          className="max-h-[min(60vh,420px)] overflow-y-auto px-2 py-2"
        >
          {loadState === 'loading' && (
            <p className="px-3 py-8 text-center text-sm text-slate-600 dark:text-neutral-400">Loading…</p>
          )}
          {loadState === 'error' && (
            <p className="px-3 py-8 text-center text-sm text-rose-600 dark:text-rose-400">
              Could not load search.
            </p>
          )}
          {loadState === 'ready' && coursePicker.active && (
            <>
              <div
                className="px-3 pb-1 pt-2 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400"
                aria-hidden="true"
              >
                Scope to course
              </div>
              {pickerCourses.length === 0 && (
                <p className="px-3 py-8 text-center text-sm text-slate-600 dark:text-neutral-400">
                  No matching courses.
                </p>
              )}
              {pickerCourses.map((course, idx) => {
                const selected = idx === safeIndex
                return (
                  <button
                    key={course.courseCode}
                    id={`cmd-course-${encodeURIComponent(course.courseCode)}`}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    ref={selected ? activeRowRef : undefined}
                    className={`flex w-full items-start gap-3 rounded-xl px-3 py-2.5 text-start text-sm transition-[background-color,color,border-color] ${selected
                      ? 'bg-indigo-50 text-slate-900 dark:bg-indigo-950/70 dark:text-neutral-100'
                      : 'text-slate-700 hover:bg-slate-50 dark:text-neutral-300 dark:hover:bg-neutral-800/80'
                      }`}
                    onMouseEnter={() => setCursor(idx)}
                    onClick={() => selectPickerCourse(course)}
                  >
                    <BookOpen
                      className={`mt-0.5 h-4 w-4 shrink-0 ${selected
                        ? 'text-indigo-600 dark:text-indigo-400'
                        : 'text-slate-400 dark:text-neutral-500'
                        }`}
                      aria-hidden
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block font-medium leading-snug">{course.title}</span>
                      <span
                        className={`block text-xs ${selected
                          ? 'text-slate-600 dark:text-neutral-300'
                          : 'text-slate-600 dark:text-neutral-400'
                          }`}
                      >
                        {course.courseCode}
                      </span>
                    </span>
                  </button>
                )
              })}
            </>
          )}
          {loadState === 'ready' && !coursePicker.active && activeServerLoading && filtered.length === 0 && !isHubMode && (
            <p className="px-3 py-8 text-center text-sm text-slate-600 dark:text-neutral-400">Searching…</p>
          )}
          {loadState === 'ready' && !coursePicker.active && !activeServerLoading && filtered.length === 0 && (
            <p className="px-3 py-8 text-center text-sm text-slate-600 dark:text-neutral-400">No results.</p>
          )}
          {loadState === 'ready' && !coursePicker.active &&
            filtered.map((item, idx) => {
              const showHeader = idx === 0 || filtered[idx - 1]!.group !== item.group
              const Icon = GROUP_ICONS[item.group]
              const selected = idx === safeIndex
              return (
                <div key={item.id}>
                  {showHeader && (
                    <div
                      className="px-3 pb-1 pt-2 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400"
                      aria-hidden="true"
                    >
                      {SEARCH_GROUP_LABEL[item.group]}
                    </div>
                  )}
                  <button
                    id={`cmd-result-${item.id}`}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    ref={selected ? activeRowRef : undefined}
                    className={`flex w-full items-start gap-3 rounded-xl px-3 py-2.5 text-start text-sm transition-[background-color,color,border-color] ${selected
                      ? 'bg-indigo-50 text-slate-900 dark:bg-indigo-950/70 dark:text-neutral-100'
                      : 'text-slate-700 hover:bg-slate-50 dark:text-neutral-300 dark:hover:bg-neutral-800/80'
                      }`}
                    onMouseEnter={() => setCursor(idx)}
                    onClick={() => go(item)}
                  >
                    <Icon
                      className={`mt-0.5 h-4 w-4 shrink-0 ${selected
                        ? 'text-indigo-600 dark:text-indigo-400'
                        : 'text-slate-400 dark:text-neutral-500'
                        }`}
                      aria-hidden
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block font-medium leading-snug">{item.title}</span>
                      <span
                        className={`block text-xs ${selected
                          ? 'text-slate-600 dark:text-neutral-300'
                          : 'text-slate-600 dark:text-neutral-400'
                          }`}
                      >
                        {item.subtitle}
                      </span>
                    </span>
                  </button>
                </div>
              )
            })}
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-slate-100 px-4 py-2 text-[11px] text-slate-600 dark:border-neutral-700 dark:text-neutral-400">
          <span className="flex items-center gap-3">
            <span className="inline-flex items-center gap-1">
              <ArrowUp className="h-3.5 w-3.5 opacity-90" aria-hidden />
              <ArrowDown className="h-3.5 w-3.5 opacity-90" aria-hidden />
              Navigate
            </span>
            <span className="inline-flex items-center gap-1">
              <kbd className="rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 font-mono dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                ↵
              </kbd>
              Open
            </span>
          </span>
        </div>
      </div>
    </div>
  )

  return createPortal(palette, document.body)
}

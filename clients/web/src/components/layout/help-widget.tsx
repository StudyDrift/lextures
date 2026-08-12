import { useEffect, useId, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { HelpCircle, X, ExternalLink, MessageCircle, Search, ArrowLeft } from 'lucide-react'
import { authorizedFetch } from '../../lib/api'
import {
  getPublicDocArticleHtml,
  parseDocPath,
  searchPublicContent,
  type PublicContentReaderArticle,
  type PublicContentSearchResult,
} from '../../lib/public-content-api'
import { Button, IconButton } from '../ui'

interface HelpArticle {
  title: string
  url: string
  slug: string
  isFallback?: boolean
}

interface ResultItem {
  id: string
  title: string
  url: string
  isFallback?: boolean
}

const HELP_CENTER_URL = 'https://lextures.com/docs'
const SEARCH_DEBOUNCE_MS = 300
const SEARCH_MAX_RESULTS = 8
const FETCH_TIMEOUT_MS = 5000

function withTimeout(signal?: AbortSignal): AbortSignal {
  const timeoutSignal = AbortSignal.timeout(FETCH_TIMEOUT_MS)
  if (!signal) return timeoutSignal
  return AbortSignal.any([signal, timeoutSignal])
}

function useShowHelpPopover() {
  const [showHelpPopover, setShowHelpPopover] = useState(true)

  useEffect(() => {
    let cancelled = false
    async function loadSetting() {
      try {
        const res = await authorizedFetch('/api/v1/settings/account')
        if (res.ok && !cancelled) {
          const data = (await res.json()) as { showHelpPopover?: boolean }
          if (data.showHelpPopover !== undefined) {
            setShowHelpPopover(data.showHelpPopover)
          }
        }
      } catch {
        // ignore, default to true
      }
    }
    void loadSetting()
    window.addEventListener('studydrift-profile-updated', loadSetting)
    return () => {
      cancelled = true
      window.removeEventListener('studydrift-profile-updated', loadSetting)
    }
  }, [])

  return showHelpPopover
}

function HelpWidgetTrigger({
  open,
  onToggle,
}: {
  open: boolean
  onToggle: () => void
}) {
  return (
    <button
      type="button"
      aria-label="Get help"
      aria-expanded={open}
      aria-haspopup="dialog"
      onClick={onToggle}
      data-testid="help-widget-trigger"
      data-lx-help-entry
      className={`relative inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl transition-[background-color,color,border-color] focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 ${ open ? 'bg-indigo-100 text-accent-fg dark:bg-indigo-900/40 dark:text-indigo-300' : 'text-fg-muted hover:bg-surface-sunken dark:text-fg-muted dark:hover:bg-surface-overlay' }`}
    >
      <HelpCircle className="h-5 w-5" aria-hidden />
    </button>
  )
}

function HelpWidgetPanel({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [contextualArticles, setContextualArticles] = useState<HelpArticle[]>([])
  const [contextualLoading, setContextualLoading] = useState(false)
  const [offline, setOffline] = useState(false)

  const [query, setQuery] = useState('')
  const [searchResults, setSearchResults] = useState<PublicContentSearchResult[] | null>(null)
  const [searching, setSearching] = useState(false)
  const [activeIndex, setActiveIndex] = useState(-1)

  const [reader, setReader] = useState<{
    article: PublicContentReaderArticle | null
    loading: boolean
    error: boolean
    fallbackUrl: string
    fallbackTitle: string
  } | null>(null)

  const { pathname } = useLocation()
  const { i18n, t } = useTranslation('common')
  const panelId = useId()
  const listboxId = useId()
  const panelRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const readerHeadingRef = useRef<HTMLHeadingElement>(null)
  const triggerFocusRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    const load = async () => {
      setContextualLoading(true)
      setOffline(false)
      try {
        const locale = (i18n.language || 'en').split('-')[0]
        const res = await authorizedFetch(
          `/api/v1/help/contextual-articles?route=${encodeURIComponent(pathname)}&locale=${encodeURIComponent(locale)}`,
          { signal: withTimeout() },
        )
        const data = res.ok ? ((await res.json()) as { articles: HelpArticle[] } | null) : null
        if (!cancelled) {
          if (data?.articles) setContextualArticles(data.articles)
          else setOffline(true)
        }
      } catch {
        if (!cancelled) setOffline(true)
      } finally {
        if (!cancelled) setContextualLoading(false)
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [open, pathname, i18n.language])

  // FR-7: debounced live search against the public content search endpoint.
  useEffect(() => {
    if (!open) return
    const trimmed = query.trim()
    if (!trimmed) {
      setSearchResults(null)
      setSearching(false)
      return
    }
    setSearching(true)
    const controller = new AbortController()
    const timer = window.setTimeout(async () => {
      try {
        const results = await searchPublicContent(trimmed, {
          kind: 'doc',
          limit: SEARCH_MAX_RESULTS,
          signal: withTimeout(controller.signal),
        })
        setSearchResults(results)
        setActiveIndex(-1)
      } catch {
        setSearchResults([])
      } finally {
        setSearching(false)
      }
    }, SEARCH_DEBOUNCE_MS)
    return () => {
      window.clearTimeout(timer)
      controller.abort()
    }
  }, [open, query])

  useEffect(() => {
    if (!open) return
    triggerFocusRef.current = document.activeElement as HTMLElement | null
    const panel = panelRef.current
    if (!panel) return
    const firstFocusable = panel.querySelector<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
    )
    firstFocusable?.focus()
  }, [open])

  useEffect(() => {
    if (!open) {
      setQuery('')
      setSearchResults(null)
      setActiveIndex(-1)
      setReader(null)
    }
  }, [open])

  useEffect(() => {
    if (reader) readerHeadingRef.current?.focus()
  }, [reader])

  if (!open) return null

  const inSearchMode = query.trim().length > 0
  const results: ResultItem[] = inSearchMode
    ? (searchResults ?? []).map((r) => ({ id: r.path, title: r.title, url: `https://lextures.com${r.path}` }))
    : contextualArticles.map((a) => ({ id: a.slug, title: a.title, url: a.url, isFallback: a.isFallback }))

  const openReader = async (item: ResultItem) => {
    const parsed = parseDocPath(new URL(item.url, 'https://lextures.com').pathname)
    if (!parsed) {
      window.open(item.url, '_blank', 'noopener,noreferrer')
      return
    }
    setReader({ article: null, loading: true, error: false, fallbackUrl: item.url, fallbackTitle: item.title })
    try {
      const article = await getPublicDocArticleHtml(parsed.category, parsed.slug, withTimeout())
      setReader({ article, loading: false, error: false, fallbackUrl: item.url, fallbackTitle: item.title })
    } catch {
      setReader({ article: null, loading: false, error: true, fallbackUrl: item.url, fallbackTitle: item.title })
    }
  }

  const closeReader = () => setReader(null)

  const handleSearchKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Escape' && inSearchMode) {
      e.stopPropagation()
      setQuery('')
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIndex((i) => Math.min(i + 1, results.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex((i) => Math.max(i - 1, 0))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      const item = results[activeIndex] ?? results[0]
      if (item) void openReader(item)
    }
  }

  const handleOpenHelpCenter = () => {
    window.open(HELP_CENTER_URL, '_blank', 'noopener,noreferrer')
  }

  const loading = inSearchMode ? searching : contextualLoading
  const resultsCountId = `${panelId}-count`

  return (
    <div
      id={panelId}
      ref={panelRef}
      role="dialog"
      aria-modal="true"
      aria-label={t('help.widget.title', { defaultValue: 'Help' })}
      data-testid="help-widget-panel"
      className="absolute end-0 top-full z-50 mt-1 flex w-80 flex-col rounded-xl border border-border-default bg-surface-raised shadow-lg shadow-slate-900/10 sm:w-96 dark:border-border-default dark:bg-surface-overlay dark:shadow-black/40"
    >
      <div className="flex items-center justify-between border-b border-border-subtle px-4 py-3 dark:border-border-default">
        <div className="flex items-center gap-2">
          <HelpCircle className="h-4 w-4 text-accent-fg" />
          <span className="text-sm font-semibold text-fg-default">
            {t('help.widget.title', { defaultValue: 'Help' })}
          </span>
        </div>
        <IconButton
          type="button"
          variant="ghost"
          size="sm"
          onClick={onClose}
          aria-label={t('help.widget.close', { defaultValue: 'Close help panel' })}
          className="text-fg-subtle hover:text-fg-muted"
        >
          <X className="h-4 w-4" />
        </IconButton>
      </div>

      {reader ? (
        <HelpReader
          reader={reader}
          onBack={closeReader}
          headingRef={readerHeadingRef}
          t={t}
        />
      ) : (
        <>
          <div className="border-b border-border-subtle px-4 py-2 dark:border-border-default">
            <div
              role="search"
              className="flex items-center gap-2 rounded-md border border-border-default bg-surface-base px-3 py-1.5 dark:border-border-default dark:bg-neutral-700"
            >
              <Search className="h-3.5 w-3.5 flex-shrink-0 text-fg-subtle" />
              <input
                ref={inputRef}
                type="text"
                role="searchbox"
                placeholder={t('help.widget.search.placeholder', { defaultValue: 'Search help articles…' })}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={handleSearchKeyDown}
                className="w-full bg-transparent text-sm text-fg-muted placeholder-slate-400 focus:outline-none dark:text-fg-default dark:placeholder-neutral-500"
                aria-label={t('help.widget.search.label', { defaultValue: 'Search help articles' })}
                aria-controls={listboxId}
                aria-expanded={results.length > 0}
                aria-activedescendant={activeIndex >= 0 ? `${listboxId}-${activeIndex}` : undefined}
                autoComplete="off"
              />
            </div>
          </div>

          <span id={resultsCountId} className="sr-only" aria-live="polite">
            {!loading &&
              t('help.widget.results.count', { count: results.length, defaultValue: '{{count}} results' })}
          </span>

          <div className="max-h-64 overflow-y-auto px-2 py-2">
            {loading ? (
              <p className="px-2 py-4 text-center text-sm text-fg-subtle">
                {t('help.widget.loading', { defaultValue: 'Loading…' })}
              </p>
            ) : offline && !inSearchMode ? (
              <p className="px-2 py-4 text-center text-sm text-fg-subtle">
                {t('help.widget.offline', { defaultValue: 'Help center unavailable — open lextures.com/docs' })}
              </p>
            ) : results.length > 0 ? (
              <ul
                id={listboxId}
                role="listbox"
                aria-label={t('help.widget.title', { defaultValue: 'Help' })}
                className="space-y-0.5"
              >
                {results.map((item, index) => (
                  <li
                    key={item.id}
                    id={`${listboxId}-${index}`}
                    role="option"
                    aria-selected={index === activeIndex}
                  >
                    <button
                      type="button"
                      onMouseEnter={() => setActiveIndex(index)}
                      onClick={() => void openReader(item)}
                      className={`flex w-full items-center justify-between rounded-md px-3 py-2 text-start text-sm text-fg-muted hover:bg-indigo-50 hover:text-accent-fg focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:text-fg-default dark:hover:bg-indigo-900/30 dark:hover:text-indigo-300 ${index === activeIndex ? 'bg-indigo-50 text-accent-fg dark:bg-indigo-900/30 dark:text-indigo-300' : ''}`}
                    >
                      <span className="flex-1 truncate">{item.title}</span>
                      {item.isFallback ? (
                        <span className="ms-2 shrink-0 text-xs text-fg-muted">
                          {t('marketingContent.translations.fallbackNotice', { defaultValue: 'English' })}
                        </span>
                      ) : null}
                      <ExternalLink className="ms-2 h-3.5 w-3.5 flex-shrink-0 opacity-50" />
                    </button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="px-2 py-4 text-center text-sm text-fg-subtle">
                {inSearchMode
                  ? t('help.widget.empty.search', { query, defaultValue: 'No results for "{{query}}".' })
                  : t('help.widget.empty.contextual', {
                      defaultValue: 'No articles for this screen yet — search or open the help center.',
                    })}
              </p>
            )}
          </div>

          <div className="border-t border-border-subtle px-4 py-3 dark:border-border-default">
            <Button
              type="button"
              onClick={handleOpenHelpCenter}
              className="w-full"
            >
              <MessageCircle className="h-4 w-4" />
              {t('help.widget.visitHelpCenter', { defaultValue: 'Visit Help Center' })}
              <span className="sr-only">{t('help.widget.opensInNewTab', { defaultValue: '(opens in a new tab)' })}</span>
            </Button>
          </div>
        </>
      )}
    </div>
  )
}

function HelpReader({
  reader,
  onBack,
  headingRef,
  t,
}: {
  reader: {
    article: PublicContentReaderArticle | null
    loading: boolean
    error: boolean
    fallbackUrl: string
    fallbackTitle: string
  }
  onBack: () => void
  headingRef: React.RefObject<HTMLHeadingElement | null>
  t: (key: string, opts?: Record<string, unknown>) => string
}) {
  const title = reader.article?.title ?? reader.fallbackTitle
  const url = reader.article ? `https://lextures.com${reader.article.path}` : reader.fallbackUrl

  return (
    <div role="document" aria-label={title} className="flex max-h-96 flex-col">
      <div className="flex items-center gap-2 border-b border-border-subtle px-4 py-2 dark:border-border-default">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onBack}
          className="gap-1 px-1 text-fg-muted dark:text-fg-default"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          {t('help.widget.back', { defaultValue: 'Back' })}
        </Button>
      </div>
      <div className="flex-1 overflow-y-auto px-4 py-3" lang={reader.article?.locale}>
        <h2 ref={headingRef} tabIndex={-1} className="mb-2 text-base font-semibold text-fg-default focus:outline-none">
          {title}
        </h2>
        {reader.article?.categoryTitle ? (
          <p className="mb-3 text-xs uppercase tracking-wide text-fg-subtle">{reader.article.categoryTitle}</p>
        ) : null}
        {reader.loading ? (
          <p className="text-sm text-fg-subtle">{t('help.widget.loading', { defaultValue: 'Loading…' })}</p>
        ) : reader.error || !reader.article ? (
          <p className="text-sm text-fg-subtle">{t('help.widget.readerError', { defaultValue: "This article couldn't be loaded here." })}</p>
        ) : (
          <div
            className="prose prose-sm max-w-none text-fg-default dark:prose-invert"
            dangerouslySetInnerHTML={{ __html: reader.article.bodyHtml }}
          />
        )}
      </div>
      <div className="border-t border-border-subtle px-4 py-3 dark:border-border-default">
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className="flex w-full items-center justify-center gap-2 rounded-lg border border-border-default px-4 py-2 text-sm font-medium text-fg-default hover:bg-surface-sunken focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:hover:bg-neutral-700"
        >
          {t('help.widget.openInHelpCenter', { defaultValue: 'Open in help center' })}
          <ExternalLink className="h-3.5 w-3.5" />
          <span className="sr-only">{t('help.widget.opensInNewTab', { defaultValue: '(opens in a new tab)' })}</span>
        </a>
      </div>
    </div>
  )
}

export function HelpWidgetMenu() {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const showHelpPopover = useShowHelpPopover()

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

  if (!showHelpPopover) return null

  return (
    <div ref={rootRef} className="relative shrink-0">
      <HelpWidgetTrigger open={open} onToggle={() => setOpen((v) => !v)} />
      <HelpWidgetPanel open={open} onClose={() => setOpen(false)} />
    </div>
  )
}

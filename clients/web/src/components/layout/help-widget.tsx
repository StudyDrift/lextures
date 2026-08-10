import { useEffect, useId, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { HelpCircle, X, ExternalLink, MessageCircle, Search } from 'lucide-react'
import { authorizedFetch } from '../../lib/api'

interface HelpArticle {
  title: string
  url: string
  slug: string
}

const HELP_CENTER_URL = 'https://lextures.com/docs'

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
  const [articles, setArticles] = useState<HelpArticle[]>([])
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const { pathname } = useLocation()
  const panelId = useId()
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    const load = async () => {
      setLoading(true)
      try {
        const res = await authorizedFetch(
          `/api/v1/help/contextual-articles?route=${encodeURIComponent(pathname)}`,
        )
        const data = res.ok ? ((await res.json()) as { articles: HelpArticle[] } | null) : null
        if (!cancelled && data?.articles) setArticles(data.articles)
      } catch {
        // silently fail — fallback to help center link
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [open, pathname])

  useEffect(() => {
    if (!open) return
    const panel = panelRef.current
    if (!panel) return
    const firstFocusable = panel.querySelector<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
    )
    firstFocusable?.focus()
  }, [open])

  useEffect(() => {
    if (!open) setQuery('')
  }, [open])

  if (!open) return null

  const filtered = query
    ? articles.filter((a) => a.title.toLowerCase().includes(query.toLowerCase()))
    : articles

  const handleOpenHelpCenter = () => {
    window.open(HELP_CENTER_URL, '_blank', 'noopener,noreferrer')
  }

  return (
    <div
      id={panelId}
      ref={panelRef}
      role="dialog"
      aria-modal="true"
      aria-label="Help"
      data-testid="help-widget-panel"
      className="absolute end-0 top-full z-50 mt-1 flex w-80 flex-col rounded-xl border border-border-default bg-surface-raised shadow-lg shadow-slate-900/10 dark:border-border-default dark:bg-surface-overlay dark:shadow-black/40"
    >
      <div className="flex items-center justify-between border-b border-border-subtle px-4 py-3 dark:border-border-default">
        <div className="flex items-center gap-2">
          <HelpCircle className="h-4 w-4 text-accent-fg" />
          <span className="text-sm font-semibold text-fg-default">Help</span>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close help panel"
          className="rounded p-1 text-fg-subtle hover:bg-surface-sunken hover:text-fg-muted focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:hover:bg-neutral-700 dark:hover:text-fg-default"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="border-b border-border-subtle px-4 py-2 dark:border-border-default">
        <div className="flex items-center gap-2 rounded-md border border-border-default bg-surface-base px-3 py-1.5 dark:border-border-default dark:bg-neutral-700">
          <Search className="h-3.5 w-3.5 flex-shrink-0 text-fg-subtle" />
          <input
            type="text"
            placeholder="Search help articles…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full bg-transparent text-sm text-fg-muted placeholder-slate-400 focus:outline-none dark:text-fg-default dark:placeholder-neutral-500"
            aria-label="Search help articles"
          />
        </div>
      </div>

      <div className="max-h-64 overflow-y-auto px-2 py-2">
        {loading ? (
          <p className="px-2 py-4 text-center text-sm text-fg-subtle">Loading…</p>
        ) : filtered.length > 0 ? (
          <ul role="list" className="space-y-0.5">
            {filtered.map((article) => (
              <li key={article.slug}>
                <a
                  href={article.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center justify-between rounded-md px-3 py-2 text-sm text-fg-muted hover:bg-indigo-50 hover:text-accent-fg focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:text-fg-default dark:hover:bg-indigo-900/30 dark:hover:text-indigo-300"
                >
                  <span className="flex-1 truncate">{article.title}</span>
                  <ExternalLink className="ms-2 h-3.5 w-3.5 flex-shrink-0 opacity-50" />
                </a>
              </li>
            ))}
          </ul>
        ) : (
          <p className="px-2 py-4 text-center text-sm text-fg-subtle">
            {query ? 'No articles matched your search.' : 'No articles available.'}
          </p>
        )}
      </div>

      <div className="border-t border-border-subtle px-4 py-3 dark:border-border-default">
        <button
          type="button"
          onClick={handleOpenHelpCenter}
          className="flex w-full items-center justify-center gap-2 rounded-lg bg-accent-solid px-4 py-2 text-sm font-medium text-white hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2"
        >
          <MessageCircle className="h-4 w-4" />
          Visit Help Center
        </button>
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

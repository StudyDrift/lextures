import { useCallback, useEffect, useId, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import {
  Bell,
  BellRing,
  ClipboardCheck,
  Inbox,
  LayoutGrid,
  Megaphone,
  MessageCircle,
  type LucideIcon,
  X,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import {
  fetchUnifiedNotifications,
  inboxAlertsToUnified,
  notificationActionHref,
  parseFeedNotificationChannel,
  parseInboxNotificationMessageId,
  type UnifiedNotification,
  type UnifiedNotificationKind,
} from '../../lib/unified-notifications'
import { markAllInboxMessagesRead, patchMailbox } from '../../lib/communication-api'
import { formatTimeAgoFromIso } from '../../lib/format-time-ago'
import { useCourseFeedUnread } from '../../context/use-course-feed-unread'
import { useInboxUnreadCount, useMailboxRevision, useRefreshInboxUnread } from '../../context/use-inbox-unread'
import { useInboxNotifications } from '../../context/use-push-notifications'

/** Easing: strong deceleration (not linear) for panel + backdrop. */
const NOTIF_DRAWER_EASE = 'cubic-bezier(0.16, 1, 0.3, 1)'
const NOTIF_DRAWER_MS = 320

function usePrefersReducedMotion(): boolean {
  const [reduced, setReduced] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia('(prefers-reduced-motion: reduce)').matches : false,
  )
  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)')
    const sync = () => setReduced(mq.matches)
    sync()
    mq.addEventListener('change', sync)
    return () => mq.removeEventListener('change', sync)
  }, [])
  return reduced
}

const FILTER_OPTIONS = [
  { id: 'all', label: 'All notifications', icon: LayoutGrid },
  { id: 'alerts', label: 'Alerts', icon: BellRing },
  { id: 'inbox', label: 'Inbox', icon: Inbox },
  { id: 'feed_mention', label: 'Feed', icon: MessageCircle },
  { id: 'graded', label: 'Grades', icon: ClipboardCheck },
  { id: 'announcement', label: 'Announcements', icon: Megaphone },
] as const satisfies ReadonlyArray<{ id: string; label: string; icon: LucideIcon }>

type NotificationFilter = (typeof FILTER_OPTIONS)[number]['id']

function kindIcon(kind: UnifiedNotificationKind) {
  switch (kind) {
    case 'inbox':
      return Inbox
    case 'feed_mention':
      return MessageCircle
    case 'announcement':
      return Megaphone
    case 'graded':
      return ClipboardCheck
    case 'alert':
      return BellRing
  }
}

export function NotificationsDrawerTrigger({
  open,
  onOpen,
}: {
  open: boolean
  onOpen: () => void
}) {
  const inboxUnread = useInboxUnreadCount()
  const { totalFeedUnread } = useCourseFeedUnread()
  const { unreadCount: alertsUnread } = useInboxNotifications()
  const badge = Math.min(99, inboxUnread + totalFeedUnread + alertsUnread)
  return (
    <button
      type="button"
      className="relative inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-slate-600 transition hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 dark:text-neutral-300 dark:hover:bg-neutral-800"
      aria-label={badge > 0 ? `Notifications (${badge} unread)` : 'Notifications'}
      aria-expanded={open}
      aria-haspopup="dialog"
      onClick={onOpen}
    >
      <Bell className="h-5 w-5" aria-hidden />
      {badge > 0 ? (
        <span className="absolute end-1.5 top-1.5 flex h-[1.125rem] min-w-[1.125rem] items-center justify-center rounded-full bg-indigo-600 px-1 text-[10px] font-semibold text-white dark:bg-indigo-500">
          {badge > 99 ? '99+' : badge}
        </span>
      ) : null}
    </button>
  )
}

export function NotificationsDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const enterRafRef = useRef<{ outer: number | null; inner: number | null }>({ outer: null, inner: null })
  const closeBtnRef = useRef<HTMLButtonElement>(null)
  const titleId = useId()
  const descId = useId()
  const reducedMotion = usePrefersReducedMotion()
  const [portalVisible, setPortalVisible] = useState(open)
  const [entered, setEntered] = useState(false)
  const [filter, setFilter] = useState<NotificationFilter>('all')
  const [items, setItems] = useState<UnifiedNotification[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const mailboxRevision = useMailboxRevision()
  const refreshInboxUnread = useRefreshInboxUnread()
  const inboxUnread = useInboxUnreadCount()
  const { totalFeedUnread, clearFeedChannelUnread, clearAllFeedUnread } = useCourseFeedUnread()
  const {
    notifications: alertItems,
    unreadCount: alertsUnread,
    markRead: markAlertRead,
    markAllRead,
    refresh: refreshAlerts,
  } = useInboxNotifications()
  const [markAllBusy, setMarkAllBusy] = useState(false)

  const totalUnread = inboxUnread + totalFeedUnread + alertsUnread

  const filterHasUnread = useCallback(
    (id: NotificationFilter) => {
      switch (id) {
        case 'all':
          return totalUnread > 0
        case 'alerts':
          return alertsUnread > 0
        case 'inbox':
          return inboxUnread > 0
        case 'feed_mention':
          return totalFeedUnread > 0
        case 'graded':
        case 'announcement':
          return false
        default: {
          const _exhaustive: never = id
          return _exhaustive
        }
      }
    },
    [totalUnread, alertsUnread, inboxUnread, totalFeedUnread],
  )

  useEffect(() => {
    if (open) return
    setFilter('all')
  }, [open])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const next = await fetchUnifiedNotifications()
      setItems(next)
    } catch {
      setError('Could not load notifications.')
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!open) return
    void load()
    void refreshAlerts()
  }, [open, load, mailboxRevision, refreshAlerts])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  useLayoutEffect(() => {
    if (open) {
      setPortalVisible(true)
      if (reducedMotion) {
        setEntered(true)
        return
      }
      setEntered(false)
      // Two rAFs so the browser paints translate-x-full before we animate to 0;
      // a single rAF often runs in the same frame as the style flush → no transition.
      enterRafRef.current.outer = requestAnimationFrame(() => {
        enterRafRef.current.outer = null
        enterRafRef.current.inner = requestAnimationFrame(() => {
          enterRafRef.current.inner = null
          setEntered(true)
        })
      })
      return () => {
        if (enterRafRef.current.outer != null) cancelAnimationFrame(enterRafRef.current.outer)
        if (enterRafRef.current.inner != null) cancelAnimationFrame(enterRafRef.current.inner)
        enterRafRef.current = { outer: null, inner: null }
      }
    }
    setEntered(false)
    if (reducedMotion) {
      setPortalVisible(false)
      return
    }
    const t = window.setTimeout(() => setPortalVisible(false), NOTIF_DRAWER_MS)
    return () => window.clearTimeout(t)
  }, [open, reducedMotion])

  useEffect(() => {
    if (!portalVisible) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [portalVisible])

  useEffect(() => {
    if (!entered) return
    const t = window.setTimeout(() => closeBtnRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [entered])

  const mergedAll = useMemo(() => {
    const alerts = inboxAlertsToUnified(alertItems)
    const combined = [...alerts, ...items]
    combined.sort((a, b) => new Date(b.sortAt).getTime() - new Date(a.sortAt).getTime())
    return combined
  }, [alertItems, items])

  const handleNotificationOpen = useCallback(
    (row: UnifiedNotification) => {
      if (row.kind === 'alert' && row.alertId && !row.isRead) {
        void markAlertRead(row.alertId)
        return
      }
      if (row.kind === 'inbox') {
        const messageId = parseInboxNotificationMessageId(row.id)
        if (messageId) {
          void patchMailbox(messageId, { read: true })
            .then(() => refreshInboxUnread())
            .catch(() => {})
          setItems((prev) => prev.filter((item) => item.id !== row.id))
        }
        return
      }
      if (row.kind === 'feed_mention' || row.kind === 'announcement') {
        const feed = parseFeedNotificationChannel(row.id)
        if (feed) clearFeedChannelUnread(feed.courseCode, feed.channelId)
      }
    },
    [markAlertRead, refreshInboxUnread, clearFeedChannelUnread],
  )

  const handleMarkAllAsRead = useCallback(async () => {
    if (markAllBusy || totalUnread === 0) return
    setMarkAllBusy(true)
    try {
      await Promise.all([
        markAllRead(),
        markAllInboxMessagesRead().then(() => refreshInboxUnread()),
      ])
      clearAllFeedUnread()
      setItems((prev) => prev.filter((item) => item.kind !== 'inbox'))
      await load()
      await refreshAlerts()
    } catch {
      /* ignore partial failures; refresh to reconcile */
      void load()
      void refreshAlerts()
      void refreshInboxUnread()
    } finally {
      setMarkAllBusy(false)
    }
  }, [
    markAllBusy,
    totalUnread,
    markAllRead,
    refreshInboxUnread,
    clearAllFeedUnread,
    load,
    refreshAlerts,
  ])

  const filtered = useMemo(() => {
    if (filter === 'all') return mergedAll
    if (filter === 'alerts') return []
    return items.filter((i) => i.kind === filter)
  }, [filter, mergedAll, items])

  const activeFilterLabel = FILTER_OPTIONS.find((option) => option.id === filter)?.label ?? 'All notifications'

  if (!open && !portalVisible) return null

  const transitionStyle = { transitionTimingFunction: NOTIF_DRAWER_EASE } as const

  return createPortal(
    <div className="fixed inset-0 z-[60] flex justify-end">
      <button
        type="button"
        aria-label="Close notifications"
        style={{
          ...transitionStyle,
          transitionProperty: 'opacity',
          transitionDuration: reducedMotion ? '0.01ms' : `${Math.round(NOTIF_DRAWER_MS * 0.85)}ms`,
        }}
        className={`lex-btn-static absolute inset-0 bg-slate-900/45 backdrop-blur-[1px] ${
          entered ? 'opacity-100' : 'opacity-0'
        }`}
        onClick={onClose}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descId}
        style={{
          ...transitionStyle,
          transitionProperty: 'transform',
          transitionDuration: reducedMotion ? '0.01ms' : `${NOTIF_DRAWER_MS}ms`,
        }}
        className={`relative flex h-dvh w-[min(100%,22rem)] flex-col border-s border-slate-200 bg-white shadow-2xl shadow-slate-900/20 will-change-transform dark:border-neutral-700 dark:bg-neutral-900 dark:shadow-black/50 sm:w-[26rem] ${
          entered ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        <div className="flex shrink-0 items-start justify-between gap-3 border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
              <h2 id={titleId} className="text-base font-semibold tracking-tight text-slate-900 dark:text-neutral-100">
                Notifications
              </h2>
              {totalUnread > 0 ? (
                <button
                  type="button"
                  disabled={markAllBusy}
                  onClick={() => void handleMarkAllAsRead()}
                  className="rounded-lg px-2 py-1 text-xs font-medium text-indigo-600 hover:bg-indigo-50 disabled:opacity-60 dark:text-indigo-400 dark:hover:bg-neutral-800"
                >
                  {markAllBusy ? 'Marking…' : 'Mark all as read'}
                </button>
              ) : null}
            </div>
            <p id={descId} className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
              {filter === 'all'
                ? 'Inbox, alerts, feed, grades, and announcements.'
                : `Showing ${activeFilterLabel.toLowerCase()} only.`}
            </p>
          </div>
          <button
            ref={closeBtnRef}
            type="button"
            aria-label="Close"
            onClick={onClose}
            className="lex-icon-hit inline-flex shrink-0 items-center justify-center rounded-lg text-slate-500 motion-safe:transition-colors hover:bg-slate-100 hover:text-slate-800 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
          >
            <X className="h-5 w-5" aria-hidden />
          </button>
        </div>

        <div
          role="tablist"
          aria-label="Filter notifications by type"
          className="grid shrink-0 grid-cols-6 border-b border-slate-100 dark:border-neutral-800"
        >
          {FILTER_OPTIONS.map(({ id, label, icon: Icon }) => {
            const active = filter === id
            const hasUnread = filterHasUnread(id)
            return (
              <button
                key={id}
                type="button"
                role="tab"
                aria-selected={active}
                aria-label={label}
                title={label}
                onClick={() => setFilter(id)}
                className={`relative flex flex-col items-center justify-center gap-1 py-2.5 transition ${
                  active
                    ? 'text-indigo-600 dark:text-indigo-400'
                    : 'text-slate-400 hover:bg-slate-50 hover:text-slate-700 dark:text-neutral-500 dark:hover:bg-neutral-800/60 dark:hover:text-neutral-200'
                }`}
              >
                <span className="relative">
                  <Icon className="h-[18px] w-[18px]" strokeWidth={active ? 2.25 : 1.75} aria-hidden />
                  {hasUnread && !active ? (
                    <span
                      className="absolute -end-0.5 -top-0.5 h-1.5 w-1.5 rounded-full bg-indigo-500 ring-2 ring-white dark:ring-neutral-900"
                      aria-hidden
                    />
                  ) : null}
                </span>
                {active ? (
                  <span
                    className="absolute inset-x-2 bottom-0 h-0.5 rounded-full bg-indigo-600 dark:bg-indigo-400"
                    aria-hidden
                  />
                ) : null}
              </button>
            )
          })}
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
          {/* Alerts tab — system/push notifications from the notifications inbox */}
          {filter === 'alerts' ? (
            <div>
              {alertItems.length === 0 ? (
                <p className="px-2 py-8 text-center text-sm text-slate-500 dark:text-neutral-400">No alerts.</p>
              ) : null}
              <ul className="flex flex-col gap-1 pb-[env(safe-area-inset-bottom)]">
                {alertItems.map((n) => (
                  <li key={n.id}>
                    <Link
                      to={notificationActionHref(n.actionUrl)}
                      onClick={() => {
                        if (!n.isRead) void markAlertRead(n.id)
                        onClose()
                      }}
                      className={`flex gap-3 rounded-xl px-2 py-2.5 text-start transition hover:bg-slate-50 dark:hover:bg-neutral-800 ${n.isRead ? 'opacity-60' : ''}`}
                    >
                      <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
                        <BellRing className="h-5 w-5" aria-hidden />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className={`line-clamp-2 text-sm font-medium ${n.isRead ? 'text-slate-500 dark:text-neutral-400' : 'text-slate-900 dark:text-neutral-100'}`}>
                          {n.title}
                        </span>
                        <span className="mt-0.5 line-clamp-2 text-xs text-slate-500 dark:text-neutral-400">
                          {n.body}
                        </span>
                        <span className="mt-1 text-[11px] text-slate-400 dark:text-neutral-500">
                          {formatTimeAgoFromIso(n.createdAt)}
                        </span>
                      </span>
                      {!n.isRead ? <span className="mt-1 h-2 w-2 shrink-0 rounded-full bg-indigo-500" aria-hidden /> : null}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}

          {filter !== 'alerts' ? (
            <>
          {loading && !items.length ? (
            <p className="px-2 py-6 text-center text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
          ) : null}
          {error ? (
            <p className="px-2 py-4 text-center text-sm text-rose-600 dark:text-rose-400" role="alert">
              {error}
            </p>
          ) : null}
          {!loading && !error && filtered.length === 0 ? (
            <p className="px-2 py-8 text-center text-sm text-slate-500 dark:text-neutral-400">
              Nothing to show for this filter.
            </p>
          ) : null}
          <ul className="flex flex-col gap-1 pb-[env(safe-area-inset-bottom)]">
            {filtered.map((row) => {
              const Icon = kindIcon(row.kind)
              const unreadAlert = row.kind === 'alert' && !row.isRead
              return (
                <li key={row.id}>
                  <Link
                    to={row.href}
                    onClick={() => {
                      handleNotificationOpen(row)
                      onClose()
                    }}
                    className={[
                      'flex gap-3 rounded-xl px-2 py-2.5 text-start transition hover:bg-slate-50 dark:hover:bg-neutral-800',
                      unreadAlert ? '' : row.kind === 'alert' ? 'opacity-60' : '',
                    ].join(' ')}
                  >
                    <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
                      <Icon className="h-5 w-5" aria-hidden />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span
                        className={[
                          'line-clamp-2 text-sm font-medium',
                          unreadAlert
                            ? 'text-slate-900 dark:text-neutral-100'
                            : row.kind === 'alert'
                              ? 'text-slate-500 dark:text-neutral-400'
                              : 'text-slate-900 dark:text-neutral-100',
                        ].join(' ')}
                      >
                        {row.title}
                      </span>
                      <span className="mt-0.5 line-clamp-2 text-xs text-slate-500 dark:text-neutral-400">
                        {row.subtitle}
                      </span>
                      <span className="mt-1 text-[11px] text-slate-400 dark:text-neutral-500">
                        {formatTimeAgoFromIso(row.sortAt)}
                      </span>
                    </span>
                    {unreadAlert ? (
                      <span className="mt-1 h-2 w-2 shrink-0 rounded-full bg-indigo-500" aria-hidden />
                    ) : null}
                  </Link>
                </li>
              )
            })}
          </ul>
            </>
          ) : null}
        </div>

        {loading && items.length > 0 ? (
          <div
            className="pointer-events-none absolute bottom-3 start-1/2 -translate-x-1/2 rounded-full bg-slate-900/80 px-3 py-1 text-xs font-medium text-white dark:bg-neutral-100 dark:text-neutral-900"
            role="status"
          >
            Updating…
          </div>
        ) : null}
      </div>
    </div>,
    document.body,
  )
}

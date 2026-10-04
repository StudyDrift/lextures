import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  CourseChecklistApiError,
  dismissChecklistItem,
  fetchCourseChecklist,
  recheckChecklistItem,
  refreshCourseChecklist,
  restoreChecklistItem,
} from '../../../lib/course-checklist-api'
import type {
  ChecklistItem,
  ChecklistResponse,
  DismissReason,
} from '../../../lib/course-checklist-api-schemas'
import { isOutstandingStatus } from '../../../lib/course-checklist-api-schemas'
import {
  readCategoryCollapseState,
  writeCategoryCollapseState,
  type CategoryCollapseMap,
} from '../../../lib/course-checklist-category-state'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import { emitChecklistTelemetry } from '../../../lib/checklist-telemetry'
import { fetchAiProcessingOptOut } from '../../../lib/study-buddy-api'
import { useCourseChecklistSummary } from '../../../context/course-checklist-summary-context'
import {
  EXTERNAL_LINK_HEALTH_ITEM_ID,
  checklistHasPendingLinkCheck,
  isExternalLinkCheckPending,
  replaceChecklistItem,
} from '../../../lib/checklist-link-health'

type LoadState = 'idle' | 'loading' | 'ready' | 'error' | 'forbidden'

export function useChecklistPage(courseCode: string | undefined) {
  const { refresh: refreshSummary, canManageCourse } = useCourseChecklistSummary()
  const [data, setData] = useState<ChecklistResponse | null>(null)
  const [loadState, setLoadState] = useState<LoadState>('idle')
  const [error, setError] = useState<string | null>(null)
  const [collapsed, setCollapsed] = useState<CategoryCollapseMap>({})
  const [dismissedOpen, setDismissedOpen] = useState(false)
  /** When false, done items are hidden (default). User can toggle to review them. */
  const [showCompleted, setShowCompleted] = useState(false)
  const [busyItemId, setBusyItemId] = useState<string | null>(null)
  const [itemErrors, setItemErrors] = useState<Record<string, string>>({})
  const [liveMessage, setLiveMessage] = useState('')
  const [refreshing, setRefreshing] = useState(false)
  const [dismissTarget, setDismissTarget] = useState<ChecklistItem | null>(null)
  const [dismissBusy, setDismissBusy] = useState(false)
  const [dismissError, setDismissError] = useState<string | null>(null)
  const [aiOptOut, setAiOptOut] = useState(false)
  const [mappingAssistItem, setMappingAssistItem] = useState<ChecklistItem | null>(null)
  const [linkPollNonce, setLinkPollNonce] = useState(0)

  useEffect(() => {
    let cancelled = false
    void fetchAiProcessingOptOut().then((v) => {
      if (!cancelled) setAiOptOut(v)
    })
    return () => {
      cancelled = true
    }
  }, [])

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoadState('loading')
    setError(null)
    try {
      const res = await fetchCourseChecklist(courseCode)
      setData(res)
      const stored = readCategoryCollapseState(courseCode, res.catalogVersion)
      if (stored) {
        setCollapsed(stored)
      } else {
        const initial: CategoryCollapseMap = {}
        for (const cat of res.categories) {
          const outstanding = cat.items.some((i) => isOutstandingStatus(i.status))
          initial[cat.id] = !outstanding
        }
        setCollapsed(initial)
        writeCategoryCollapseState(courseCode, res.catalogVersion, initial)
      }
      setLoadState('ready')
      emitChecklistTelemetry('checklist_viewed')
    } catch (err) {
      if (err instanceof CourseChecklistApiError && err.status === 403) {
        setLoadState('forbidden')
      } else {
        setError(err instanceof Error ? err.message : courseChecklistI18n.loadError)
        setLoadState('error')
      }
      setData(null)
    }
  }, [courseCode])

  useEffect(() => {
    if (!courseCode) return
    if (!canManageCourse) {
      // Still attempt load so direct URL hits get server 403 (FR-27).
    }
    void load()
  }, [courseCode, canManageCourse, load])

  const persistCollapse = useCallback(
    (next: CategoryCollapseMap) => {
      setCollapsed(next)
      if (courseCode && data) {
        writeCategoryCollapseState(courseCode, data.catalogVersion, next)
      }
    },
    [courseCode, data],
  )

  const toggleCategory = useCallback(
    (categoryId: string) => {
      persistCollapse({ ...collapsed, [categoryId]: !collapsed[categoryId] })
    },
    [collapsed, persistCollapse],
  )

  const ensureCategoryExpanded = useCallback(
    (categoryId: string) => {
      if (collapsed[categoryId]) {
        persistCollapse({ ...collapsed, [categoryId]: false })
      }
    },
    [collapsed, persistCollapse],
  )

  const onRefresh = useCallback(async () => {
    if (!courseCode) return
    setRefreshing(true)
    setError(null)
    try {
      const res = await refreshCourseChecklist(courseCode)
      setData(res)
      setLoadState('ready')
      await refreshSummary()
      setLiveMessage(courseChecklistI18n.itemRecheckedLive)
      emitChecklistTelemetry('checklist_refreshed')
    } catch (err) {
      setError(err instanceof Error ? err.message : courseChecklistI18n.loadError)
    } finally {
      setRefreshing(false)
    }
  }, [courseCode, refreshSummary])

  const onDismissConfirm = useCallback(
    async (body: { reason: DismissReason; note?: string }) => {
      if (!courseCode || !dismissTarget || !data) return
      const item = dismissTarget
      setDismissBusy(true)
      setDismissError(null)
      const prev = data
      // Optimistic: move to dismissed
      setData({
        ...data,
        categories: data.categories.map((c) => ({
          ...c,
          items: c.items.filter((i) => i.id !== item.id),
        })),
        dismissed: [
          {
            ...item,
            dismissal: {
              dismissedAt: new Date().toISOString(),
              byUserId: '',
              byDisplayName: 'You',
              reason: body.reason,
              note: body.note ?? '',
            },
          },
          ...data.dismissed,
        ],
        summary: {
          ...data.summary,
          dismissed: data.summary.dismissed + 1,
          outstandingEssential:
            item.tier === 'essential' && isOutstandingStatus(item.status)
              ? Math.max(0, data.summary.outstandingEssential - 1)
              : data.summary.outstandingEssential,
          outstandingTotal: isOutstandingStatus(item.status)
            ? Math.max(0, data.summary.outstandingTotal - 1)
            : data.summary.outstandingTotal,
        },
      })
      try {
        const updated = await dismissChecklistItem(courseCode, item.id, body)
        setData((cur) =>
          cur
            ? {
                ...cur,
                dismissed: cur.dismissed.map((d) => (d.id === item.id ? updated : d)),
              }
            : cur,
        )
        setDismissTarget(null)
        setLiveMessage(courseChecklistI18n.itemDismissedLive)
        emitChecklistTelemetry('checklist_item_dismissed', {
          itemId: item.id,
          reason: body.reason,
        })
        await refreshSummary()
      } catch (err) {
        setData(prev)
        setDismissError(err instanceof Error ? err.message : 'Dismiss failed')
      } finally {
        setDismissBusy(false)
      }
    },
    [courseCode, data, dismissTarget, refreshSummary],
  )

  const onRestore = useCallback(
    async (item: ChecklistItem) => {
      if (!courseCode || !data) return
      setBusyItemId(item.id)
      setItemErrors((e) => {
        const next = { ...e }
        delete next[item.id]
        return next
      })
      const prev = data
      setData({
        ...data,
        dismissed: data.dismissed.filter((d) => d.id !== item.id),
        categories: data.categories.map((c) => {
          // Put restored item back into first category as placeholder; server will correct.
          if (c.id === data.categories[0]?.id) {
            return { ...c, items: [{ ...item, dismissal: null }, ...c.items] }
          }
          return c
        }),
      })
      try {
        await restoreChecklistItem(courseCode, item.id)
        const fresh = await fetchCourseChecklist(courseCode)
        setData(fresh)
        setLiveMessage(courseChecklistI18n.itemRestoredLive)
        emitChecklistTelemetry('checklist_item_restored', { itemId: item.id })
        await refreshSummary()
      } catch (err) {
        setData(prev)
        setItemErrors((e) => ({
          ...e,
          [item.id]: err instanceof Error ? err.message : 'Restore failed',
        }))
      } finally {
        setBusyItemId(null)
      }
    },
    [courseCode, data, refreshSummary],
  )

  const onRecheck = useCallback(
    async (item: ChecklistItem) => {
      if (!courseCode || !data) return
      setBusyItemId(item.id)
      try {
        const updated = await recheckChecklistItem(courseCode, item.id)
        if (item.id === EXTERNAL_LINK_HEALTH_ITEM_ID && isExternalLinkCheckPending(updated)) {
          setData(replaceChecklistItem(data, updated))
          setLinkPollNonce((n) => n + 1)
        } else if (item.id === EXTERNAL_LINK_HEALTH_ITEM_ID) {
          const fresh = await fetchCourseChecklist(courseCode)
          setData(fresh)
        } else {
          setData(replaceChecklistItem(data, updated))
        }
        setLiveMessage(courseChecklistI18n.itemRecheckedLive)
        emitChecklistTelemetry('checklist_item_rechecked', { itemId: item.id })
        await refreshSummary()
      } catch (err) {
        setItemErrors((e) => ({
          ...e,
          [item.id]: err instanceof Error ? err.message : 'Re-check failed',
        }))
      } finally {
        setBusyItemId(null)
      }
    },
    [courseCode, data, refreshSummary],
  )

  const linkCheckPending = useMemo(() => checklistHasPendingLinkCheck(data), [data])

  // A background link check returns "Checking links…" immediately. Keep asking
  // until the worker has written a result, then reload the checklist so the
  // open page and its counts update without a manual refresh.
  useEffect(() => {
    if (!courseCode || !linkCheckPending) return
    let cancelled = false
    let inFlight = false
    let attempts = 0
    const maxAttempts = 20
    const stop = (timer: number) => {
      window.clearInterval(timer)
    }
    const timer = window.setInterval(() => {
      if (cancelled || inFlight) return
      inFlight = true
      attempts += 1
      void (async () => {
        try {
          const updated = await recheckChecklistItem(courseCode, EXTERNAL_LINK_HEALTH_ITEM_ID)
          if (cancelled) return
          if (isExternalLinkCheckPending(updated)) {
            if (attempts >= maxAttempts) stop(timer)
            return
          }
          const fresh = await fetchCourseChecklist(courseCode)
          if (cancelled) return
          setData(fresh)
          setLoadState('ready')
          setLiveMessage(courseChecklistI18n.linkCheckUpdatedLive)
          await refreshSummary()
          stop(timer)
        } catch {
          if (attempts >= maxAttempts) stop(timer)
        } finally {
          inFlight = false
        }
      })()
    }, 2000)
    return () => {
      cancelled = true
      stop(timer)
    }
  }, [courseCode, linkCheckPending, linkPollNonce, refreshSummary])

  const allDone = useMemo(() => {
    if (!data) return false
    return data.summary.outstandingTotal === 0 && data.summary.total > 0
  }, [data])

  const catalogEmpty = useMemo(() => {
    if (!data) return false
    return data.summary.total === 0 && data.dismissed.length === 0
  }, [data])

  return {
    data,
    loadState,
    error,
    load,
    collapsed,
    toggleCategory,
    ensureCategoryExpanded,
    dismissedOpen,
    setDismissedOpen,
    showCompleted,
    setShowCompleted,
    busyItemId,
    itemErrors,
    liveMessage,
    refreshing,
    onRefresh,
    dismissTarget,
    setDismissTarget,
    dismissBusy,
    dismissError,
    onDismissConfirm,
    onRestore,
    onRecheck,
    allDone,
    catalogEmpty,
    aiOptOut,
    hideAiActions: aiOptOut,
    mappingAssistItem,
    setMappingAssistItem,
  }
}

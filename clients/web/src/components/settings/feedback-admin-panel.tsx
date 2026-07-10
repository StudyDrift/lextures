import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
} from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, ChevronLeft, ChevronRight, Loader2, RefreshCw, Search } from 'lucide-react'
import { FeedbackAdminStatusBadge } from './feedback-admin-status-badge'
import {
  dateInputToFromIso,
  dateInputToToIso,
  FEEDBACK_CATEGORIES,
  FEEDBACK_LIST_PAGE_SIZE,
  FEEDBACK_SOURCES,
  FEEDBACK_STATUSES,
  feedbackPersonLabel,
  fetchFeedbackDetail,
  fetchFeedbackList,
  patchFeedback,
  type FeedbackDetail,
  type FeedbackListItem,
  type FeedbackListParams,
  type FeedbackStatus,
} from '../../lib/feedback-admin-api'
import type { FeedbackCategory } from '../../lib/feedback-api'
import { formatAbsolute, formatRelativeCompact } from '../../lib/format-datetime'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

const SEARCH_DEBOUNCE_MS = 300

type ListFilters = {
  status: FeedbackStatus | ''
  category: FeedbackCategory | ''
  source: (typeof FEEDBACK_SOURCES)[number] | ''
  q: string
  fromDate: string
  toDate: string
}

const EMPTY_FILTERS: ListFilters = {
  status: '',
  category: '',
  source: '',
  q: '',
  fromDate: '',
  toDate: '',
}

function filtersToParams(filters: ListFilters, cursor?: string): FeedbackListParams {
  return {
    status: filters.status,
    category: filters.category,
    source: filters.source,
    q: filters.q,
    from: dateInputToFromIso(filters.fromDate) || undefined,
    to: dateInputToToIso(filters.toDate) || undefined,
    cursor,
  }
}

function FeedbackFiltersBar({
  filters,
  onChange,
  onApply,
  onClear,
  disabled,
}: {
  filters: ListFilters
  onChange: (next: ListFilters) => void
  onApply: () => void
  onClear: () => void
  disabled: boolean
}) {
  const { t } = useTranslation('common')
  const searchId = useId()

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-neutral-800 dark:bg-neutral-900/50">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
            {t('settings.feedback.filters.status')}
          </span>
          <select
            value={filters.status}
            onChange={(e) =>
              onChange({ ...filters, status: e.target.value as FeedbackStatus | '' })
            }
            disabled={disabled}
            className="w-full rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
          >
            <option value="">{t('settings.feedback.filters.allStatuses')}</option>
            {FEEDBACK_STATUSES.map((s) => (
              <option key={s} value={s}>
                {t(`settings.feedback.status.${s}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
            {t('settings.feedback.filters.category')}
          </span>
          <select
            value={filters.category}
            onChange={(e) =>
              onChange({ ...filters, category: e.target.value as FeedbackCategory | '' })
            }
            disabled={disabled}
            className="w-full rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
          >
            <option value="">{t('settings.feedback.filters.allCategories')}</option>
            {FEEDBACK_CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {t(`feedback.category.${c}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
            {t('settings.feedback.filters.source')}
          </span>
          <select
            value={filters.source}
            onChange={(e) =>
              onChange({
                ...filters,
                source: e.target.value as (typeof FEEDBACK_SOURCES)[number] | '',
              })
            }
            disabled={disabled}
            className="w-full rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
          >
            <option value="">{t('settings.feedback.filters.allSources')}</option>
            {FEEDBACK_SOURCES.map((s) => (
              <option key={s} value={s}>
                {t(`settings.feedback.source.${s}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm" htmlFor={searchId}>
          <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
            {t('settings.feedback.filters.search')}
          </span>
          <div className="relative">
            <Search
              className="pointer-events-none absolute start-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400"
              aria-hidden
            />
            <input
              id={searchId}
              type="search"
              value={filters.q}
              onChange={(e) => onChange({ ...filters, q: e.target.value })}
              disabled={disabled}
              placeholder={t('settings.feedback.filters.searchPlaceholder')}
              className="w-full rounded-lg border border-slate-200 bg-white py-2 ps-9 pe-3 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            />
          </div>
        </label>
      </div>
      <div className="flex flex-wrap items-end gap-3">
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
            {t('settings.feedback.filters.from')}
          </span>
          <input
            type="date"
            value={filters.fromDate}
            onChange={(e) => onChange({ ...filters, fromDate: e.target.value })}
            disabled={disabled}
            className="rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
          />
        </label>
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
            {t('settings.feedback.filters.to')}
          </span>
          <input
            type="date"
            value={filters.toDate}
            onChange={(e) => onChange({ ...filters, toDate: e.target.value })}
            disabled={disabled}
            className="rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
          />
        </label>
        <button
          type="button"
          onClick={onApply}
          disabled={disabled}
          className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200"
        >
          {t('settings.feedback.filters.apply')}
        </button>
        <button
          type="button"
          onClick={onClear}
          disabled={disabled}
          className="rounded-lg px-3 py-2 text-sm font-medium text-slate-600 hover:text-slate-900 disabled:opacity-50 dark:text-neutral-400 dark:hover:text-neutral-100"
        >
          {t('settings.feedback.filters.clear')}
        </button>
      </div>
    </div>
  )
}

function FeedbackListRow({
  item,
  onOpen,
}: {
  item: FeedbackListItem
  onOpen: (id: string, rowEl: HTMLTableRowElement | null) => void
}) {
  const { t } = useTranslation('common')

  function activate(rowEl: HTMLTableRowElement | null) {
    onOpen(item.id, rowEl)
  }

  function onKeyDown(e: KeyboardEvent<HTMLTableRowElement>) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      activate(e.currentTarget)
    }
  }

  return (
    <tr
      data-feedback-row-id={item.id}
      tabIndex={0}
      onClick={(e) => activate(e.currentTarget)}
      onKeyDown={onKeyDown}
      className="cursor-pointer border-t border-slate-100 hover:bg-slate-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-indigo-500 dark:border-neutral-800 dark:hover:bg-neutral-800/60"
    >
      <td className="px-4 py-3 whitespace-nowrap text-slate-600 dark:text-neutral-400">
        <time dateTime={item.created_at} title={formatAbsolute(item.created_at)}>
          {formatRelativeCompact(item.created_at)}
        </time>
      </td>
      <td className="px-4 py-3 text-slate-900 dark:text-neutral-100">
        {feedbackPersonLabel(item.submitter)}
      </td>
      <td className="px-4 py-3 text-slate-700 dark:text-neutral-300">
        {t(`feedback.category.${item.category}`)}
      </td>
      <td className="px-4 py-3 text-slate-700 dark:text-neutral-300">
        {t(`settings.feedback.source.${item.source}`)}
      </td>
      <td className="px-4 py-3">
        <FeedbackAdminStatusBadge status={item.status} />
      </td>
      <td className="max-w-xs truncate px-4 py-3 text-slate-600 dark:text-neutral-400">
        {item.message_preview}
      </td>
    </tr>
  )
}

function FeedbackListCard({
  item,
  onOpen,
}: {
  item: FeedbackListItem
  onOpen: (id: string) => void
}) {
  const { t } = useTranslation('common')

  return (
    <button
      type="button"
      onClick={() => onOpen(item.id)}
      className="w-full rounded-xl border border-slate-200 bg-white p-4 text-start hover:border-indigo-200 dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-indigo-800/50"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <time
          dateTime={item.created_at}
          title={formatAbsolute(item.created_at)}
          className="text-xs text-slate-500 dark:text-neutral-500"
        >
          {formatRelativeCompact(item.created_at)}
        </time>
        <FeedbackAdminStatusBadge status={item.status} />
      </div>
      <p className="mt-2 text-sm font-medium text-slate-900 dark:text-neutral-100">
        {feedbackPersonLabel(item.submitter)}
      </p>
      <p className="mt-1 text-xs text-slate-500 dark:text-neutral-500">
        {t(`feedback.category.${item.category}`)} · {t(`settings.feedback.source.${item.source}`)}
      </p>
      <p className="mt-2 line-clamp-2 text-sm text-slate-600 dark:text-neutral-400">
        {item.message_preview}
      </p>
    </button>
  )
}

function FeedbackDetailView({
  detail,
  loading,
  error,
  onBack,
  onSaved,
}: {
  detail: FeedbackDetail | null
  loading: boolean
  error: string | null
  onBack: () => void
  onSaved: (updated: FeedbackDetail) => void
}) {
  const { t } = useTranslation('common')
  const backRef = useRef<HTMLButtonElement>(null)
  const [status, setStatus] = useState<FeedbackStatus>('new')
  const [adminNote, setAdminNote] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!detail || loading) return
    backRef.current?.focus()
  }, [detail, loading])

  useEffect(() => {
    if (!detail) return
    setStatus(detail.status)
    setAdminNote(detail.admin_note ?? '')
  }, [detail])

  async function onSave(e: FormEvent) {
    e.preventDefault()
    if (!detail || saving) return
    const prevStatus = detail.status
    const prevNote = detail.admin_note ?? ''
    const optimistic: FeedbackDetail = {
      ...detail,
      status,
      admin_note: adminNote.trim() || undefined,
    }
    setSaving(true)
    onSaved(optimistic)
    try {
      const updated = await patchFeedback(detail.id, {
        status,
        admin_note: adminNote,
      })
      onSaved(updated)
      toastSaveOk(t('settings.feedback.statusUpdated'))
    } catch (err) {
      onSaved({
        ...detail,
        status: prevStatus,
        admin_note: prevNote || undefined,
      })
      toastMutationError(err instanceof Error ? err.message : t('settings.feedback.saveError'))
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="mt-6 text-sm text-slate-500 dark:text-neutral-400">{t('settings.feedback.loading')}</p>
  }
  if (error) {
    return (
      <div className="mt-6 space-y-3">
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-2 text-sm font-medium text-slate-600 dark:text-neutral-400"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden />
          {t('settings.feedback.backToList')}
        </button>
      </div>
    )
  }
  if (!detail) return null

  return (
    <div className="mt-6 space-y-6">
      <button
        ref={backRef}
        type="button"
        onClick={onBack}
        className="inline-flex items-center gap-2 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        {t('settings.feedback.backToList')}
      </button>

      <div className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              {feedbackPersonLabel(detail.submitter)}
            </h3>
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">{detail.submitter.email}</p>
          </div>
          <FeedbackAdminStatusBadge status={detail.status} />
        </div>

        <section className="mt-6" aria-labelledby="feedback-message-heading">
          <h4 id="feedback-message-heading" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            {t('settings.feedback.detail.message')}
          </h4>
          <p className="mt-2 whitespace-pre-wrap break-words text-sm text-slate-800 dark:text-neutral-200">
            {detail.message}
          </p>
        </section>

        <section className="mt-6" aria-labelledby="feedback-context-heading">
          <h4 id="feedback-context-heading" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            {t('settings.feedback.detail.context')}
          </h4>
          <dl className="mt-2 grid gap-2 text-sm sm:grid-cols-2">
            <div>
              <dt className="text-slate-500 dark:text-neutral-500">{t('settings.feedback.detail.route')}</dt>
              <dd className="break-all text-slate-900 dark:text-neutral-100">
                {detail.context.route || '—'}
              </dd>
            </div>
            <div>
              <dt className="text-slate-500 dark:text-neutral-500">{t('settings.feedback.detail.platform')}</dt>
              <dd className="text-slate-900 dark:text-neutral-100">
                {t(`settings.feedback.source.${detail.source}`)}
              </dd>
            </div>
            <div>
              <dt className="text-slate-500 dark:text-neutral-500">{t('settings.feedback.detail.appVersion')}</dt>
              <dd className="text-slate-900 dark:text-neutral-100">{detail.app_version || '—'}</dd>
            </div>
            <div>
              <dt className="text-slate-500 dark:text-neutral-500">{t('settings.feedback.detail.locale')}</dt>
              <dd className="text-slate-900 dark:text-neutral-100">{detail.context.locale || '—'}</dd>
            </div>
            <div>
              <dt className="text-slate-500 dark:text-neutral-500">{t('settings.feedback.detail.submittedAt')}</dt>
              <dd className="text-slate-900 dark:text-neutral-100">{formatAbsolute(detail.created_at)}</dd>
            </div>
            <div>
              <dt className="text-slate-500 dark:text-neutral-500">{t('settings.feedback.detail.category')}</dt>
              <dd className="text-slate-900 dark:text-neutral-100">
                {t(`feedback.category.${detail.category}`)}
              </dd>
            </div>
          </dl>
        </section>

        {detail.resolved_at && detail.resolved_by ? (
          <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">
            {t('settings.feedback.detail.resolved', {
              name: feedbackPersonLabel(detail.resolved_by),
              when: formatAbsolute(detail.resolved_at),
            })}
          </p>
        ) : null}

        <form className="mt-6 space-y-4 border-t border-slate-100 pt-6 dark:border-neutral-800" onSubmit={onSave}>
          <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            {t('settings.feedback.detail.admin')}
          </h4>
          <label className="block text-sm" htmlFor="feedback-admin-status">
            <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
              {t('settings.feedback.detail.status')}
            </span>
            <select
              id="feedback-admin-status"
              value={status}
              onChange={(e) => setStatus(e.target.value as FeedbackStatus)}
              disabled={saving}
              className="w-full max-w-xs rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            >
              {FEEDBACK_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {t(`settings.feedback.status.${s}`)}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-sm">
            <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
              {t('settings.feedback.detail.internalNote')}
            </span>
            <textarea
              value={adminNote}
              onChange={(e) => setAdminNote(e.target.value)}
              rows={4}
              disabled={saving}
              className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            />
          </label>
          <button
            type="submit"
            disabled={saving}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white"
          >
            {saving ? (
              <>
                <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                {t('settings.feedback.saving')}
              </>
            ) : (
              t('settings.feedback.saveNote')
            )}
          </button>
        </form>
      </div>
    </div>
  )
}

export function FeedbackAdminPanel() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const { id: detailId } = useParams<{ id?: string }>()

  const [filters, setFilters] = useState<ListFilters>(EMPTY_FILTERS)
  const [appliedFilters, setAppliedFilters] = useState<ListFilters>(EMPTY_FILTERS)
  const [debouncedQ, setDebouncedQ] = useState('')
  const [items, setItems] = useState<FeedbackListItem[]>([])
  const [total, setTotal] = useState(0)
  const [pageIndex, setPageIndex] = useState(0)
  const [nextCursor, setNextCursor] = useState<string | null>(null)
  const pageCursorsRef = useRef<Record<number, string>>({ 0: '' })
  const [listLoading, setListLoading] = useState(true)
  const [listError, setListError] = useState<string | null>(null)

  const [detail, setDetail] = useState<FeedbackDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState<string | null>(null)

  const lastOpenedRowIdRef = useRef<string | null>(null)
  const listHeadingRef = useRef<HTMLHeadingElement>(null)

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedQ(filters.q)
    }, SEARCH_DEBOUNCE_MS)
    return () => window.clearTimeout(timer)
  }, [filters.q])

  useEffect(() => {
    setAppliedFilters((prev) => ({ ...prev, q: debouncedQ }))
    pageCursorsRef.current = { 0: '' }
    setPageIndex(0)
    setNextCursor(null)
  }, [debouncedQ])

  const resetPagination = useCallback(() => {
    pageCursorsRef.current = { 0: '' }
    setPageIndex(0)
    setNextCursor(null)
  }, [])

  const loadList = useCallback(async () => {
    if (detailId) return
    setListLoading(true)
    setListError(null)
    const cursor = pageCursorsRef.current[pageIndex] ?? ''
    try {
      const data = await fetchFeedbackList(filtersToParams(appliedFilters, cursor || undefined))
      setItems(data.items)
      setTotal(data.total ?? data.items.length)
      setNextCursor(data.next_cursor ?? null)
      if (data.next_cursor) {
        pageCursorsRef.current[pageIndex + 1] = data.next_cursor
      }
    } catch (e) {
      setListError(e instanceof Error ? e.message : t('settings.feedback.loadError'))
      setItems([])
      setNextCursor(null)
    } finally {
      setListLoading(false)
    }
  }, [appliedFilters, detailId, pageIndex, t])

  useEffect(() => {
    void loadList()
  }, [loadList])

  const loadDetail = useCallback(async (id: string) => {
    setDetailLoading(true)
    setDetailError(null)
    try {
      setDetail(await fetchFeedbackDetail(id))
    } catch (e) {
      setDetailError(e instanceof Error ? e.message : t('settings.feedback.loadError'))
      setDetail(null)
    } finally {
      setDetailLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (!detailId) {
      setDetail(null)
      setDetailError(null)
      return
    }
    void loadDetail(detailId)
  }, [detailId, loadDetail])

  function openDetail(id: string, _rowEl?: HTMLTableRowElement | null) {
    lastOpenedRowIdRef.current = id
    navigate(`/settings/feedback/${encodeURIComponent(id)}`)
  }

  function backToList() {
    navigate('/settings/feedback')
  }

  useEffect(() => {
    if (detailId || listLoading) return
    const rowId = lastOpenedRowIdRef.current
    if (!rowId) return

    const frame = window.requestAnimationFrame(() => {
      const row = document.querySelector<HTMLTableRowElement>(
        `[data-feedback-row-id="${CSS.escape(rowId)}"]`,
      )
      if (row) {
        row.focus()
      } else {
        listHeadingRef.current?.focus()
      }
      lastOpenedRowIdRef.current = null
    })
    return () => window.cancelAnimationFrame(frame)
  }, [detailId, listLoading, items])

  function applyFilters() {
    setAppliedFilters(filters)
    resetPagination()
  }

  function onDetailSaved(updated: FeedbackDetail) {
    setDetail(updated)
    setItems((prev) =>
      prev.map((row) =>
        row.id === updated.id
          ? {
              ...row,
              status: updated.status,
            }
          : row,
      ),
    )
  }

  function clearFilters() {
    setFilters(EMPTY_FILTERS)
    setAppliedFilters(EMPTY_FILTERS)
    setDebouncedQ('')
    resetPagination()
  }

  if (detailId) {
    return (
      <FeedbackDetailView
        detail={detail}
        loading={detailLoading}
        error={detailError}
        onBack={backToList}
        onSaved={onDetailSaved}
      />
    )
  }

  const hasNext = Boolean(nextCursor)
  const hasPrev = pageIndex > 0
  const pageStart = pageIndex * FEEDBACK_LIST_PAGE_SIZE + (items.length > 0 ? 1 : 0)
  const pageEnd = pageIndex * FEEDBACK_LIST_PAGE_SIZE + items.length

  return (
    <div>
      <h2
        ref={listHeadingRef}
        tabIndex={-1}
        className="text-base font-semibold text-slate-900 dark:text-neutral-100"
      >
        {t('settings.feedback.title')}
      </h2>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        {t('settings.feedback.description')}
      </p>

      <div className="mt-6">
        <FeedbackFiltersBar
          filters={filters}
          onChange={setFilters}
          onApply={applyFilters}
          onClear={clearFilters}
          disabled={listLoading}
        />
      </div>

      {listLoading && (
        <p className="mt-6 text-sm text-slate-500 dark:text-neutral-400">{t('settings.feedback.loading')}</p>
      )}

      {listError && !listLoading && (
        <div className="mt-6 space-y-3">
          <p role="alert" className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-100">
            {listError}
          </p>
          <button
            type="button"
            onClick={() => void loadList()}
            className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-sm font-medium dark:border-neutral-600"
          >
            <RefreshCw className="h-4 w-4" aria-hidden />
            {t('settings.feedback.retry')}
          </button>
        </div>
      )}

      {!listLoading && !listError && items.length === 0 && (
        <p className="mt-8 text-center text-sm text-slate-500 dark:text-neutral-400">
          {t('settings.feedback.empty')}
        </p>
      )}

      {!listLoading && !listError && items.length > 0 && (
        <>
          <div className="mt-6 hidden overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800 md:block">
            <table className="min-w-full text-start text-sm">
              <thead className="sticky top-0 z-10 bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-neutral-950 dark:text-neutral-400">
                <tr>
                  <th scope="col" className="px-4 py-2.5 font-semibold">
                    {t('settings.feedback.columns.when')}
                  </th>
                  <th scope="col" className="px-4 py-2.5 font-semibold">
                    {t('settings.feedback.columns.submitter')}
                  </th>
                  <th scope="col" className="px-4 py-2.5 font-semibold">
                    {t('settings.feedback.columns.category')}
                  </th>
                  <th scope="col" className="px-4 py-2.5 font-semibold">
                    {t('settings.feedback.columns.source')}
                  </th>
                  <th scope="col" className="px-4 py-2.5 font-semibold">
                    {t('settings.feedback.columns.status')}
                  </th>
                  <th scope="col" className="px-4 py-2.5 font-semibold">
                    {t('settings.feedback.columns.preview')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <FeedbackListRow
                    key={item.id}
                    item={item}
                    onOpen={(id, rowEl) => openDetail(id, rowEl)}
                  />
                ))}
              </tbody>
            </table>
          </div>

          <div className="mt-6 flex flex-col gap-3 md:hidden">
            {items.map((item) => (
              <FeedbackListCard key={item.id} item={item} onOpen={openDetail} />
            ))}
          </div>

          <div className="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm text-slate-600 dark:text-neutral-400">
            <p>
              {t('settings.feedback.pagination.summary', {
                start: pageStart,
                end: pageEnd,
                total,
              })}
            </p>
            <div className="flex gap-2">
              <button
                type="button"
                disabled={!hasPrev || listLoading}
                onClick={() => setPageIndex((p) => Math.max(0, p - 1))}
                className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 disabled:opacity-50 dark:border-neutral-600"
              >
                <ChevronLeft className="h-4 w-4" aria-hidden />
                {t('settings.feedback.pagination.prev')}
              </button>
              <button
                type="button"
                disabled={!hasNext || listLoading}
                onClick={() => setPageIndex((p) => p + 1)}
                className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 disabled:opacity-50 dark:border-neutral-600"
              >
                {t('settings.feedback.pagination.next')}
                <ChevronRight className="h-4 w-4" aria-hidden />
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

import { useCallback, useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FileText, Loader2 } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchAdminTranscriptHolds,
  fetchAdminTranscriptOrders,
  fetchAdminTranscriptsConfig,
  placeAdminTranscriptHold,
  releaseAdminTranscriptHold,
  transitionAdminTranscriptOrder,
  waiveAdminTranscriptOrder,
  type TranscriptHold,
  type TranscriptHoldType,
  type TranscriptOrder,
  type TranscriptOrderTransitionAction,
} from '../../lib/transcripts-api'

const HOLD_TYPES: TranscriptHoldType[] = [
  'financial',
  'disciplinary',
  'registrar',
  'library',
  'other',
]

function statusChipClass(status: string): string {
  switch (status) {
    case 'on_hold':
      return 'bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-100'
    case 'rejected':
    case 'failed':
    case 'canceled':
      return 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-100'
    case 'processing':
    case 'in_review':
      return 'bg-sky-100 text-sky-900 dark:bg-sky-950 dark:text-sky-100'
    case 'completed':
      return 'bg-emerald-100 text-emerald-900 dark:bg-emerald-950 dark:text-emerald-100'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
  }
}

export default function AdminTranscriptsPage() {
  const { t } = useTranslation('common')
  const { ffTranscripts } = usePlatformFeatures()
  const rejectReasonId = useId()
  const holdUserId = useId()
  const [consoleEnabled, setConsoleEnabled] = useState(false)
  const [orders, setOrders] = useState<TranscriptOrder[]>([])
  const [holds, setHolds] = useState<TranscriptHold[]>([])
  const [statusFilter, setStatusFilter] = useState('')
  const [holdFilter, setHoldFilter] = useState<'all' | 'yes' | 'no'>('all')
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<TranscriptOrder | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [holdForm, setHoldForm] = useState({ userId: '', type: 'financial' as TranscriptHoldType, reason: '' })
  const [loading, setLoading] = useState(true)
  const [acting, setActing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!ffTranscripts) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const cfg = await fetchAdminTranscriptsConfig()
      setConsoleEnabled(cfg.registrarConsoleEnabled === true)
      if (!cfg.registrarConsoleEnabled) {
        setOrders([])
        setHolds([])
        return
      }
      const [queue, activeHolds] = await Promise.all([
        fetchAdminTranscriptOrders({
          status: statusFilter || undefined,
          hold: holdFilter === 'all' ? undefined : holdFilter === 'yes',
          q: query || undefined,
        }),
        fetchAdminTranscriptHolds({ active: true }),
      ])
      setOrders(queue)
      setHolds(activeHolds)
      setSelected((prev) => (prev ? (queue.find((o) => o.id === prev.id) ?? prev) : null))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.registrar.loadError'))
    } finally {
      setLoading(false)
    }
  }, [ffTranscripts, statusFilter, holdFilter, query, t])

  useEffect(() => {
    void load()
  }, [load])

  async function runTransition(action: TranscriptOrderTransitionAction) {
    if (!selected) return
    if (action === 'reject' && !rejectReason.trim()) {
      setError(t('transcripts.registrar.rejectReasonRequired'))
      return
    }
    setActing(true)
    setError(null)
    setMessage(null)
    try {
      const updated = await transitionAdminTranscriptOrder(
        selected.id,
        action,
        action === 'reject' ? rejectReason.trim() : undefined,
      )
      setSelected(updated)
      setMessage(t('transcripts.registrar.transitionOk'))
      setRejectReason('')
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.registrar.transitionError'))
    } finally {
      setActing(false)
    }
  }

  async function handlePlaceHold(e: React.FormEvent) {
    e.preventDefault()
    if (!holdForm.userId.trim()) return
    setActing(true)
    setError(null)
    try {
      await placeAdminTranscriptHold({
        userId: holdForm.userId.trim(),
        type: holdForm.type,
        reason: holdForm.reason.trim() || undefined,
      })
      setHoldForm({ userId: '', type: 'financial', reason: '' })
      setMessage(t('transcripts.registrar.holdPlaced'))
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('transcripts.registrar.holdError'))
    } finally {
      setActing(false)
    }
  }

  if (!ffTranscripts) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-10">
        <p className="text-sm text-slate-600 dark:text-neutral-400">{t('transcripts.registrar.featureOff')}</p>
      </div>
    )
  }

  if (!loading && !consoleEnabled) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-10">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-50">
          {t('transcripts.registrar.title')}
        </h1>
        <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
          {t('transcripts.registrar.consoleDisabled')}
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="flex items-start gap-3">
        <FileText className="mt-1 h-6 w-6 text-slate-500" aria-hidden />
        <div>
          <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-50">
            {t('transcripts.registrar.title')}
          </h1>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            {t('transcripts.registrar.subtitle')}
          </p>
        </div>
      </div>

      {error && (
        <p className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-100" role="alert">
          {error}
        </p>
      )}
      {message && (
        <p className="mt-4 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-100" role="status">
          {message}
        </p>
      )}

      <div className="mt-6 flex flex-wrap items-end gap-3">
        <label className="text-sm">
          <span className="block text-slate-600 dark:text-neutral-400">{t('transcripts.registrar.filterStatus')}</span>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="mt-1 rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
          >
            <option value="">{t('transcripts.registrar.filterAll')}</option>
            <option value="in_review">in_review</option>
            <option value="on_hold">on_hold</option>
            <option value="processing">processing</option>
            <option value="rejected">rejected</option>
            <option value="completed">completed</option>
          </select>
        </label>
        <label className="text-sm">
          <span className="block text-slate-600 dark:text-neutral-400">{t('transcripts.registrar.filterHold')}</span>
          <select
            value={holdFilter}
            onChange={(e) => setHoldFilter(e.target.value as 'all' | 'yes' | 'no')}
            className="mt-1 rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
          >
            <option value="all">{t('transcripts.registrar.filterAll')}</option>
            <option value="yes">{t('transcripts.registrar.filterHoldYes')}</option>
            <option value="no">{t('transcripts.registrar.filterHoldNo')}</option>
          </select>
        </label>
        <label className="min-w-[12rem] flex-1 text-sm">
          <span className="block text-slate-600 dark:text-neutral-400">{t('transcripts.registrar.filterSearch')}</span>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') void load()
            }}
            className="mt-1 w-full rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
            placeholder={t('transcripts.registrar.searchPlaceholder')}
          />
        </label>
        <button
          type="button"
          onClick={() => void load()}
          className="rounded-md bg-slate-800 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700 dark:bg-neutral-200 dark:text-neutral-900"
        >
          {t('transcripts.registrar.refresh')}
        </button>
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-[1.2fr_1fr]">
        <section aria-labelledby="queue-heading">
          <h2 id="queue-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
            {t('transcripts.registrar.queueTitle')}
          </h2>
          {loading ? (
            <p className="mt-4 flex items-center gap-2 text-sm text-slate-500">
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
              {t('common.loading')}
            </p>
          ) : orders.length === 0 ? (
            <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">{t('transcripts.registrar.queueEmpty')}</p>
          ) : (
            <ul className="mt-3 divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
              {orders.map((o) => (
                <li key={o.id}>
                  <button
                    type="button"
                    onClick={() => setSelected(o)}
                    className={`flex w-full items-start justify-between gap-3 px-4 py-3 text-left hover:bg-slate-50 dark:hover:bg-neutral-900 ${
                      selected?.id === o.id ? 'bg-slate-50 dark:bg-neutral-900' : 'bg-white dark:bg-neutral-950'
                    }`}
                  >
                    <div>
                      <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                        {o.userEmail ?? o.userId ?? o.id}
                      </p>
                      <p className="text-xs text-slate-500">
                        {o.items.length} {t('transcripts.registrar.recipients')} ·{' '}
                        {o.submittedAt ? new Date(o.submittedAt).toLocaleString() : new Date(o.createdAt).toLocaleString()}
                      </p>
                      {o.studentMessage && (
                        <p className="mt-1 text-xs text-amber-800 dark:text-amber-200">{o.studentMessage}</p>
                      )}
                    </div>
                    <span
                      className={`shrink-0 rounded px-2 py-0.5 text-xs font-medium ${statusChipClass(o.status)}`}
                      aria-label={t('transcripts.status.aria', { status: o.status })}
                    >
                      {t(`transcripts.status.${o.status}`, { defaultValue: o.status })}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section aria-labelledby="detail-heading" className="rounded-lg border border-slate-200 p-4 dark:border-neutral-800">
          <h2 id="detail-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
            {t('transcripts.registrar.detailTitle')}
          </h2>
          {!selected ? (
            <p className="mt-3 text-sm text-slate-500">{t('transcripts.registrar.detailEmpty')}</p>
          ) : (
            <div className="mt-3 space-y-4">
              <div>
                <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">{selected.userEmail}</p>
                <p className="text-xs text-slate-500">{selected.id}</p>
                <p className="mt-2">
                  <span
                    className={`rounded px-2 py-0.5 text-xs font-medium ${statusChipClass(selected.status)}`}
                    aria-label={t('transcripts.status.aria', { status: selected.status })}
                  >
                    {t(`transcripts.status.${selected.status}`, { defaultValue: selected.status })}
                  </span>
                </p>
              </div>
              {selected.paymentStatus ? (
                <p className="text-xs text-slate-500">
                  Payment: {selected.paymentStatus}
                  {selected.totalAmount != null
                    ? ` · ${selected.currency?.toUpperCase() ?? 'USD'} ${(selected.totalAmount / 100).toFixed(2)}`
                    : ''}
                </p>
              ) : null}
              <div className="flex flex-wrap gap-2">
                {(['approve', 'complete', 'cancel', 'hold', 'release'] as const).map((action) => (
                  <button
                    key={action}
                    type="button"
                    disabled={acting}
                    onClick={() => void runTransition(action)}
                    className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:hover:bg-neutral-900"
                  >
                    {t(`transcripts.registrar.action.${action}`)}
                  </button>
                ))}
                {selected.paymentStatus &&
                !['paid', 'waived', 'free', 'refunded'].includes(selected.paymentStatus) ? (
                  <button
                    type="button"
                    disabled={acting}
                    onClick={() => {
                      void (async () => {
                        setActing(true)
                        setError(null)
                        setMessage(null)
                        try {
                          const order = await waiveAdminTranscriptOrder(selected.id, 'Registrar fee waiver')
                          setSelected(order)
                          setMessage('Order fee waived.')
                          await load()
                        } catch (e) {
                          setError(e instanceof Error ? e.message : 'Could not waive order.')
                        } finally {
                          setActing(false)
                        }
                      })()
                    }}
                    className="rounded-md border border-emerald-600 px-3 py-1.5 text-xs font-medium text-emerald-800 hover:bg-emerald-50 disabled:opacity-50 dark:text-emerald-200 dark:hover:bg-emerald-950"
                  >
                    Waive fee
                  </button>
                ) : null}
              </div>
              <div>
                <label htmlFor={rejectReasonId} className="block text-xs font-medium text-slate-600 dark:text-neutral-400">
                  {t('transcripts.registrar.rejectReason')}
                </label>
                <div className="mt-1 flex gap-2">
                  <input
                    id={rejectReasonId}
                    value={rejectReason}
                    onChange={(e) => setRejectReason(e.target.value)}
                    className="flex-1 rounded-md border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  />
                  <button
                    type="button"
                    disabled={acting}
                    onClick={() => void runTransition('reject')}
                    className="rounded-md bg-red-700 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-600 disabled:opacity-50"
                  >
                    {t('transcripts.registrar.action.reject')}
                  </button>
                </div>
              </div>
              {selected.events && selected.events.length > 0 && (
                <div>
                  <h3 className="text-sm font-semibold text-slate-800 dark:text-neutral-200">
                    {t('transcripts.registrar.timeline')}
                  </h3>
                  <ol className="mt-2 space-y-2 border-l border-slate-200 pl-3 dark:border-neutral-700">
                    {selected.events.map((ev) => (
                      <li key={ev.id} className="text-xs text-slate-600 dark:text-neutral-400">
                        <span className="font-medium text-slate-800 dark:text-neutral-200">
                          {ev.fromState ?? '—'} → {ev.toState}
                        </span>
                        {ev.reason ? ` · ${ev.reason}` : ''}
                        <div className="text-[11px] text-slate-400">{new Date(ev.createdAt).toLocaleString()}</div>
                      </li>
                    ))}
                  </ol>
                </div>
              )}
            </div>
          )}
        </section>
      </div>

      <section className="mt-10" aria-labelledby="holds-heading">
        <h2 id="holds-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
          {t('transcripts.registrar.holdsTitle')}
        </h2>
        <form onSubmit={(e) => void handlePlaceHold(e)} className="mt-3 flex flex-wrap items-end gap-2">
          <label className="text-sm">
            <span className="block text-slate-600 dark:text-neutral-400">{t('transcripts.registrar.holdUserId')}</span>
            <input
              id={holdUserId}
              value={holdForm.userId}
              onChange={(e) => setHoldForm((f) => ({ ...f, userId: e.target.value }))}
              className="mt-1 w-72 rounded-md border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
              required
            />
          </label>
          <label className="text-sm">
            <span className="block text-slate-600 dark:text-neutral-400">{t('transcripts.registrar.holdType')}</span>
            <select
              value={holdForm.type}
              onChange={(e) => setHoldForm((f) => ({ ...f, type: e.target.value as TranscriptHoldType }))}
              className="mt-1 rounded-md border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
            >
              {HOLD_TYPES.map((type) => (
                <option key={type} value={type}>
                  {t(`transcripts.holdType.${type}`)}
                </option>
              ))}
            </select>
          </label>
          <label className="min-w-[12rem] flex-1 text-sm">
            <span className="block text-slate-600 dark:text-neutral-400">{t('transcripts.registrar.holdReason')}</span>
            <input
              value={holdForm.reason}
              onChange={(e) => setHoldForm((f) => ({ ...f, reason: e.target.value }))}
              className="mt-1 w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
            />
          </label>
          <button
            type="submit"
            disabled={acting}
            className="rounded-md bg-amber-700 px-3 py-1.5 text-sm font-medium text-white hover:bg-amber-600 disabled:opacity-50"
          >
            {t('transcripts.registrar.placeHold')}
          </button>
        </form>
        {holds.length === 0 ? (
          <p className="mt-4 text-sm text-slate-500">{t('transcripts.registrar.holdsEmpty')}</p>
        ) : (
          <ul className="mt-4 divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
            {holds.map((h) => (
              <li key={h.id} className="flex items-start justify-between gap-3 bg-white px-4 py-3 dark:bg-neutral-950">
                <div>
                  <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                    {t(`transcripts.holdType.${h.type}`)} · {h.userId}
                  </p>
                  <p className="text-xs text-slate-600 dark:text-neutral-400">{h.studentMessage}</p>
                  {h.reason && <p className="mt-1 text-xs text-slate-400">{t('transcripts.registrar.internalReason')}: {h.reason}</p>}
                </div>
                <button
                  type="button"
                  disabled={acting}
                  onClick={() => {
                    void releaseAdminTranscriptHold(h.id)
                      .then(() => load())
                      .catch((err: unknown) =>
                        setError(err instanceof Error ? err.message : t('transcripts.registrar.holdError')),
                      )
                  }}
                  className="shrink-0 rounded-md border border-slate-300 px-2 py-1 text-xs font-medium hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700"
                >
                  {t('transcripts.registrar.releaseHold')}
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}

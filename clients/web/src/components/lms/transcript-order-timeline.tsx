import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  cancelTranscriptOrder,
  fetchTranscriptOrderTimeline,
  resendTranscriptOrderItem,
  type TranscriptOrder,
  type TranscriptOrderTimeline,
  type TranscriptTimelineEntry,
} from '../../lib/transcripts-api'
import { formatDate } from '../../lib/format'

const TERMINAL: TranscriptOrder['status'][] = ['completed', 'canceled', 'rejected', 'failed']

type Props = {
  order: TranscriptOrder
  onChanged?: () => void
}

function entryTone(status: string): string {
  switch (status) {
    case 'opened':
    case 'delivered':
    case 'completed':
      return 'text-emerald-700 dark:text-emerald-300'
    case 'failed':
    case 'rejected':
      return 'text-red-700 dark:text-red-300'
    case 'on_hold':
    case 'canceled':
      return 'text-amber-800 dark:text-amber-200'
    default:
      return 'text-slate-800 dark:text-neutral-200'
  }
}

export function TranscriptOrderTimeline({ order, onChanged }: Props) {
  const { t } = useTranslation('common')
  const [timeline, setTimeline] = useState<TranscriptOrderTimeline | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [liveMsg, setLiveMsg] = useState('')
  const lastFingerprint = useRef('')

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const tl = await fetchTranscriptOrderTimeline(order.id)
      setTimeline(tl)
      const fp = tl.entries.map((e) => `${e.id}:${e.status}`).join('|')
      if (lastFingerprint.current && lastFingerprint.current !== fp) {
        const latest = tl.entries[tl.entries.length - 1]
        setLiveMsg(
          latest
            ? t('transcripts.tracking.liveUpdate', { status: latest.label })
            : t('transcripts.tracking.liveUpdateGeneric'),
        )
      }
      lastFingerprint.current = fp
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('transcripts.tracking.loadError'))
    } finally {
      setLoading(false)
    }
  }, [order.id, t])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    if (TERMINAL.includes(order.status)) return
    const id = window.setInterval(() => {
      void load()
    }, 8000)
    return () => window.clearInterval(id)
  }, [order.status, load])

  async function onCancel() {
    setBusy(true)
    setError(null)
    try {
      await cancelTranscriptOrder(order.id)
      setLiveMsg(t('transcripts.tracking.cancelSuccess'))
      await load()
      onChanged?.()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('transcripts.tracking.cancelError'))
    } finally {
      setBusy(false)
    }
  }

  async function onResend(itemId: string) {
    setBusy(true)
    setError(null)
    try {
      await resendTranscriptOrderItem(order.id, itemId)
      setLiveMsg(t('transcripts.delivery.resendSuccess'))
      await load()
      onChanged?.()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('transcripts.delivery.resend'))
    } finally {
      setBusy(false)
    }
  }

  const canCancel = timeline?.canCancel ?? false
  const resendItems = new Set(timeline?.canResendItems ?? [])
  const entries = timeline?.entries ?? []

  return (
    <div className="mt-3 rounded-md border border-slate-100 bg-slate-50 px-3 py-2 dark:border-neutral-800 dark:bg-neutral-950">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-xs font-semibold text-slate-700 dark:text-neutral-200">
          {t('transcripts.tracking.title')}
        </p>
        <div className="flex flex-wrap gap-2">
          {[...resendItems].map((itemId) => (
            <button
              key={itemId}
              type="button"
              disabled={busy}
              onClick={() => void onResend(itemId)}
              className="text-xs font-medium text-indigo-600 hover:underline disabled:opacity-50 dark:text-indigo-400"
            >
              {t('transcripts.delivery.resend')}
            </button>
          ))}
          {canCancel ? (
            <button
              type="button"
              disabled={busy}
              onClick={() => void onCancel()}
              className="text-xs font-medium text-red-600 hover:underline disabled:opacity-50 dark:text-red-400"
            >
              {t('transcripts.tracking.cancelOrder')}
            </button>
          ) : null}
        </div>
      </div>

      <div className="sr-only" aria-live="polite" aria-atomic="true">
        {liveMsg}
      </div>

      {loading && !timeline ? <p className="mt-1 text-xs text-slate-500">{t('common.loading')}</p> : null}
      {error ? <p className="mt-1 text-xs text-red-600 dark:text-red-400">{error}</p> : null}

      {!loading && entries.length === 0 ? (
        <p className="mt-1 text-xs text-slate-500">{t('transcripts.tracking.empty')}</p>
      ) : (
        <ol className="mt-2 space-y-1.5" aria-label={t('transcripts.tracking.title')}>
          {entries.map((e) => (
            <TimelineStep key={e.id} entry={e} />
          ))}
        </ol>
      )}
    </div>
  )
}

function TimelineStep({ entry }: { entry: TranscriptTimelineEntry }) {
  const { t } = useTranslation('common')
  const kindLabel =
    entry.kind === 'delivery' ? t('transcripts.tracking.kindDelivery') : t('transcripts.tracking.kindOrder')
  return (
    <li className="flex gap-2 text-xs text-slate-600 dark:text-neutral-400">
      <span
        className="mt-1 h-2 w-2 shrink-0 rounded-full bg-slate-400 dark:bg-neutral-500"
        aria-hidden
      />
      <div className="min-w-0">
        <p className={`font-medium ${entryTone(entry.status)}`}>
          <span className="sr-only">{kindLabel}: </span>
          {entry.label}
          <span className="font-normal text-slate-500 dark:text-neutral-500">
            {' · '}
            {formatDate(entry.at, { dateStyle: 'short', timeStyle: 'short' })}
          </span>
        </p>
        {entry.adapter ? (
          <p className="text-[11px] text-slate-500 dark:text-neutral-500">
            {entry.adapter}
            {entry.attemptNo != null ? ` #${entry.attemptNo}` : ''}
          </p>
        ) : null}
        {entry.reason || entry.detail ? (
          <p className="text-[11px] text-slate-500 dark:text-neutral-500">{entry.reason || entry.detail}</p>
        ) : null}
      </div>
    </li>
  )
}

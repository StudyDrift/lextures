import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchTranscriptItemReceipts,
  resendTranscriptOrderItem,
  type TranscriptDeliveryReceipt,
  type TranscriptOrderItem,
} from '../../lib/transcripts-api'
import { formatDate } from '../../lib/format'

type Props = {
  orderId: string
  item: TranscriptOrderItem
  onResent?: () => void
}

function statusLabel(status: TranscriptDeliveryReceipt['status']): string {
  switch (status) {
    case 'queued':
      return 'Queued'
    case 'sent':
      return 'Sent'
    case 'delivered':
      return 'Delivered'
    case 'opened':
      return 'Opened'
    case 'failed':
      return 'Failed'
    default: {
      const _exhaustive: never = status
      return _exhaustive
    }
  }
}

export function TranscriptDeliveryTimeline({ orderId, item, onResent }: Props) {
  const { t } = useTranslation('common')
  const [receipts, setReceipts] = useState<TranscriptDeliveryReceipt[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [resending, setResending] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const rows = await fetchTranscriptItemReceipts(orderId, item.id)
      setReceipts(rows)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('transcripts.delivery.noReceipts'))
    } finally {
      setLoading(false)
    }
  }, [orderId, item.id, t])

  useEffect(() => {
    void load()
  }, [load])

  async function onResend() {
    setResending(true)
    setError(null)
    try {
      await resendTranscriptOrderItem(orderId, item.id)
      await load()
      onResent?.()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('transcripts.delivery.resend'))
    } finally {
      setResending(false)
    }
  }

  const canResend = item.status === 'failed' || item.status === 'delivered'

  return (
    <div className="mt-2 rounded-md border border-slate-100 bg-slate-50 px-3 py-2 dark:border-neutral-800 dark:bg-neutral-950">
      <div className="flex items-center justify-between gap-2">
        <p className="text-xs font-semibold text-slate-700 dark:text-neutral-200">{t('transcripts.delivery.receipts')}</p>
        {canResend ? (
          <button
            type="button"
            onClick={() => void onResend()}
            disabled={resending}
            className="text-xs font-medium text-indigo-600 hover:underline disabled:opacity-50 dark:text-indigo-400"
          >
            {t('transcripts.delivery.resend')}
          </button>
        ) : null}
      </div>
      {loading ? <p className="mt-1 text-xs text-slate-500">{t('common.loading')}</p> : null}
      {error ? <p className="mt-1 text-xs text-red-600 dark:text-red-400">{error}</p> : null}
      {!loading && receipts.length === 0 ? (
        <p className="mt-1 text-xs text-slate-500">{t('transcripts.delivery.noReceipts')}</p>
      ) : (
        <ol className="mt-2 space-y-1">
          {receipts.map((r) => (
            <li key={r.id} className="text-xs text-slate-600 dark:text-neutral-400">
              <span className="font-medium text-slate-800 dark:text-neutral-200">{statusLabel(r.status)}</span>
              {' · '}
              {r.adapter} #{r.attemptNo}
              {' · '}
              {formatDate(r.createdAt, { dateStyle: 'short', timeStyle: 'short' })}
              {r.detail ? ` — ${r.detail}` : ''}
            </li>
          ))}
        </ol>
      )}
    </div>
  )
}

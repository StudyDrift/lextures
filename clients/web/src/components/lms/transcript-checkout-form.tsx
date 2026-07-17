import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { minorUnitsToMajorUnits } from '../../lib/currency-exponent'
import {
  checkoutTranscriptOrder,
  fetchTranscriptOrderQuote,
  type TranscriptOrder,
  type TranscriptQuote,
} from '../../lib/transcripts-api'

function formatMinor(amount: number, currency: string): string {
  const major = minorUnitsToMajorUnits(amount, currency)
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: currency.toUpperCase(),
      maximumFractionDigits: currency.toLowerCase() === 'jpy' ? 0 : 2,
    }).format(major)
  } catch {
    return `${currency.toUpperCase()} ${major}`
  }
}

type TranscriptCheckoutFormProps = {
  orderId: string
  onPaidOrWaived: (order: TranscriptOrder) => void
  onCancel: () => void
}

export function TranscriptCheckoutForm({
  orderId,
  onPaidOrWaived,
  onCancel,
}: TranscriptCheckoutFormProps) {
  const { t } = useTranslation('common')
  const waiverId = useId()
  const [quote, setQuote] = useState<TranscriptQuote | null>(null)
  const [waiverCode, setWaiverCode] = useState('')
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    void fetchTranscriptOrderQuote(orderId)
      .then((res) => {
        if (!cancelled) setQuote(res.quote)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : t('transcripts.checkout.errorQuote'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [orderId, t])

  async function refreshQuote() {
    setBusy(true)
    setError(null)
    try {
      const res = await fetchTranscriptOrderQuote(orderId, waiverCode)
      setQuote(res.quote)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.checkout.errorQuote'))
    } finally {
      setBusy(false)
    }
  }

  async function handlePay() {
    setBusy(true)
    setError(null)
    try {
      const result = await checkoutTranscriptOrder(orderId, {
        waiverCode: waiverCode.trim() || undefined,
        successUrl: `${window.location.origin}/transcripts?checkout=success&orderId=${encodeURIComponent(orderId)}`,
        cancelUrl: `${window.location.origin}/transcripts?checkout=cancel&orderId=${encodeURIComponent(orderId)}`,
      })
      if ('waived' in result && result.waived) {
        onPaidOrWaived(result.order)
        return
      }
      if ('checkoutUrl' in result) {
        window.location.assign(result.checkoutUrl)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.checkout.errorPay'))
    } finally {
      setBusy(false)
    }
  }

  if (loading) {
    return <p className="mt-5 text-sm text-slate-500">{t('transcripts.checkout.loading')}</p>
  }

  return (
    <div className="mt-5 space-y-4">
      <p className="text-sm text-slate-600 dark:text-neutral-400">{t('transcripts.checkout.help')}</p>
      {error ? (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}
      {quote ? (
        <table className="w-full text-start text-sm" aria-label={t('transcripts.checkout.summaryLabel')}>
          <thead>
            <tr className="border-b border-slate-200 text-slate-500 dark:border-neutral-800">
              <th className="py-2 font-medium">{t('transcripts.checkout.line')}</th>
              <th className="py-2 text-end font-medium">{t('transcripts.checkout.amount')}</th>
            </tr>
          </thead>
          <tbody>
            {quote.lines.map((line) => (
              <tr key={`${line.code}-${line.description}`} className="border-b border-slate-100 dark:border-neutral-900">
                <td className="py-2 text-slate-800 dark:text-neutral-100">{line.description}</td>
                <td className="py-2 text-end tabular-nums text-slate-800 dark:text-neutral-100">
                  {formatMinor(line.amount, quote.currency)}
                </td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr>
              <th className="pt-3 font-semibold text-slate-900 dark:text-neutral-50">
                {t('transcripts.checkout.total')}
              </th>
              <td className="pt-3 text-end font-semibold tabular-nums text-slate-900 dark:text-neutral-50">
                {formatMinor(quote.total, quote.currency)}
              </td>
            </tr>
          </tfoot>
        </table>
      ) : null}

      <div>
        <label htmlFor={waiverId} className="block text-sm font-medium text-slate-800 dark:text-neutral-100">
          {t('transcripts.checkout.waiverCode')}
        </label>
        <div className="mt-1 flex gap-2">
          <input
            id={waiverId}
            type="text"
            value={waiverCode}
            onChange={(e) => setWaiverCode(e.target.value)}
            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
            autoComplete="off"
          />
          <button
            type="button"
            onClick={() => void refreshQuote()}
            disabled={busy}
            className="rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700"
          >
            {t('transcripts.checkout.apply')}
          </button>
        </div>
      </div>

      <div className="flex flex-wrap justify-between gap-2">
        <button
          type="button"
          onClick={onCancel}
          disabled={busy}
          className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium dark:border-neutral-700"
        >
          {t('transcripts.order.cancel')}
        </button>
        <button
          type="button"
          onClick={() => void handlePay()}
          disabled={busy || !quote}
          className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
        >
          {busy
            ? t('transcripts.checkout.processing')
            : quote && !quote.requiresPayment
              ? t('transcripts.checkout.continueFree')
              : t('transcripts.checkout.pay')}
        </button>
      </div>
    </div>
  )
}

/* eslint-disable react-refresh/only-export-components -- component file exports shared label helpers */
import { useEffect, useId, useMemo, useState } from 'react'
import { X } from 'lucide-react'
import {
  type MailUrgency,
  type SubmitTranscriptRequestPayload,
  type TranscriptDeliveryType,
  type TranscriptsStudentConfig,
} from '../../lib/transcripts-api'

type PickupUrgencyOption = {
  days: number
  label: string
}

const PICKUP_URGENCY_OPTIONS: PickupUrgencyOption[] = [
  { days: 1, label: '1 business day' },
  { days: 2, label: '2 business days' },
  { days: 3, label: '3 business days' },
]

const MAIL_URGENCY_OPTIONS: { value: MailUrgency; label: string }[] = [
  { value: 'standard', label: '3–5 business days (standard)' },
  { value: 'rush', label: '1–2 business days (rush)' },
]

const DELIVERY_LABELS: Record<TranscriptDeliveryType, string> = {
  email: 'Email',
  mail: 'Mail',
  pickup: 'Pickup',
}

type TranscriptRequestModalProps = {
  open: boolean
  submitting: boolean
  config: TranscriptsStudentConfig | null
  defaultEmail: string
  onClose: () => void
  onSubmit: (payload: SubmitTranscriptRequestPayload) => void
}

export function TranscriptRequestModal({
  open,
  submitting,
  config,
  defaultEmail,
  onClose,
  onSubmit,
}: TranscriptRequestModalProps) {
  const deliveryTypeId = useId()
  const emailId = useId()
  const addressId = useId()
  const urgencyId = useId()

  const [deliveryType, setDeliveryType] = useState<TranscriptDeliveryType>('email')
  const [deliveryEmail, setDeliveryEmail] = useState('')
  const [deliveryAddress, setDeliveryAddress] = useState('')
  const [mailUrgency, setMailUrgency] = useState<MailUrgency>('standard')
  const [pickupUrgencyDays, setPickupUrgencyDays] = useState(PICKUP_URGENCY_OPTIONS[0].days)
  const [formError, setFormError] = useState<string | null>(null)

  const deliveryOptions = useMemo(() => {
    const options: TranscriptDeliveryType[] = ['email', 'mail']
    if (config?.pickupAvailable) {
      options.push('pickup')
    }
    return options
  }, [config?.pickupAvailable])

  const showUrgency = deliveryType === 'mail' || deliveryType === 'pickup'

  useEffect(() => {
    if (!open) return
    setDeliveryType('email')
    setDeliveryEmail(defaultEmail)
    setDeliveryAddress('')
    setMailUrgency('standard')
    setPickupUrgencyDays(PICKUP_URGENCY_OPTIONS[0].days)
    setFormError(null)
  }, [open, defaultEmail])

  useEffect(() => {
    if (!open || submitting) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, submitting, onClose])

  if (!open) return null

  function handleDeliveryTypeChange(next: TranscriptDeliveryType) {
    setDeliveryType(next)
    setFormError(null)
    if (next === 'mail') setMailUrgency('standard')
    if (next === 'pickup') setPickupUrgencyDays(PICKUP_URGENCY_OPTIONS[0].days)
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setFormError(null)

    if (deliveryType === 'email') {
      const email = deliveryEmail.trim()
      if (!email) {
        setFormError('Enter the email address where your transcript should be sent.')
        return
      }
      onSubmit({ deliveryType, deliveryEmail: email })
      return
    }

    if (deliveryType === 'mail') {
      const address = deliveryAddress.trim()
      if (address.length < 10) {
        setFormError('Enter your complete mailing address.')
        return
      }
      onSubmit({ deliveryType, deliveryAddress: address, mailUrgency })
      return
    }

    onSubmit({ deliveryType, urgencyDays: pickupUrgencyDays })
  }

  const urgencyHint =
    deliveryType === 'mail'
      ? 'Mail delivery timelines are measured in business days.'
      : 'Pickup timelines are measured in business days.'

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center dark:bg-neutral-950/80"
      role="dialog"
      aria-modal="true"
      aria-labelledby="transcript-request-title"
      onClick={(e) => {
        if (e.target === e.currentTarget && !submitting) onClose()
      }}
    >
      <div className="w-full max-w-lg overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-950">
        <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3 dark:border-neutral-800">
          <h3 id="transcript-request-title" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            Request transcript
          </h3>
          <button
            type="button"
            onClick={() => !submitting && onClose()}
            disabled={submitting}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
            aria-label="Close"
          >
            <X className="h-5 w-5" aria-hidden />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4 p-4">
          {formError && (
            <p role="alert" className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">
              {formError}
            </p>
          )}

          <div>
            <label htmlFor={deliveryTypeId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Delivery method
            </label>
            <select
              id={deliveryTypeId}
              value={deliveryType}
              onChange={(e) => handleDeliveryTypeChange(e.target.value as TranscriptDeliveryType)}
              disabled={submitting}
              className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
            >
              {deliveryOptions.map((option) => (
                <option key={option} value={option}>
                  {DELIVERY_LABELS[option]}
                </option>
              ))}
            </select>
          </div>

          {deliveryType === 'email' && (
            <div>
              <label htmlFor={emailId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Email address
              </label>
              <input
                id={emailId}
                type="email"
                required
                autoComplete="email"
                value={deliveryEmail}
                onChange={(e) => setDeliveryEmail(e.target.value)}
                disabled={submitting}
                placeholder="you@example.com"
                className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
              />
            </div>
          )}

          {deliveryType === 'mail' && (
            <div>
              <label htmlFor={addressId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Mailing address
              </label>
              <textarea
                id={addressId}
                required
                rows={4}
                value={deliveryAddress}
                onChange={(e) => setDeliveryAddress(e.target.value)}
                disabled={submitting}
                placeholder={'Full name\nStreet address\nCity, State ZIP\nCountry'}
                className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
              />
            </div>
          )}

          {deliveryType === 'pickup' && config?.pickupInstructions && (
            <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm text-slate-700 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300">
              <p className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                Pickup instructions
              </p>
              <p className="mt-2 whitespace-pre-wrap">{config.pickupInstructions}</p>
            </div>
          )}

          {showUrgency && (
            <div>
              <label htmlFor={urgencyId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Urgency
              </label>
              {deliveryType === 'mail' ? (
                <select
                  id={urgencyId}
                  value={mailUrgency}
                  onChange={(e) => setMailUrgency(e.target.value as MailUrgency)}
                  disabled={submitting}
                  className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
                >
                  {MAIL_URGENCY_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              ) : (
                <select
                  id={urgencyId}
                  value={pickupUrgencyDays}
                  onChange={(e) => setPickupUrgencyDays(Number(e.target.value))}
                  disabled={submitting}
                  className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
                >
                  {PICKUP_URGENCY_OPTIONS.map((option) => (
                    <option key={option.days} value={option.days}>
                      {option.label}
                    </option>
                  ))}
                </select>
              )}
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">{urgencyHint}</p>
            </div>
          )}

          <div className="flex items-center justify-end gap-2 border-t border-slate-100 pt-4 dark:border-neutral-800">
            <button
              type="button"
              onClick={() => !submitting && onClose()}
              disabled={submitting}
              className="rounded-xl px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-50"
            >
              {submitting ? 'Submitting…' : 'Submit request'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export function deliveryTypeLabel(type: TranscriptDeliveryType): string {
  return DELIVERY_LABELS[type]
}

export function urgencyLabel(
  deliveryType: TranscriptDeliveryType,
  days?: number,
  unit?: 'days' | 'business_days',
  daysMin?: number,
): string | null {
  if (deliveryType === 'email') return null

  if (daysMin != null && days != null) {
    if (daysMin === days) {
      return days === 1 ? '1 business day' : `${days} business days`
    }
    return `${daysMin}–${days} business days`
  }

  if (days == null || unit == null) return null
  const unitLabel = unit === 'business_days' ? 'business day' : 'day'
  return days === 1 ? `1 ${unitLabel}` : `${days} ${unitLabel}s`
}
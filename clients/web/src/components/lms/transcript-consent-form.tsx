import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchTranscriptConsentPreview,
  signTranscriptConsent,
  type TranscriptConsentPreview,
  type TranscriptOrder,
} from '../../lib/transcripts-api'

type TranscriptConsentFormProps = {
  orderId: string
  onSigned: (order: TranscriptOrder) => void
  onCancel?: () => void
}

export function TranscriptConsentForm({ orderId, onSigned, onCancel }: TranscriptConsentFormProps) {
  const { t, i18n } = useTranslation('common')
  const textId = useId()
  const typedId = useId()
  const agreeId = useId()
  const [preview, setPreview] = useState<TranscriptConsentPreview | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [typedName, setTypedName] = useState('')
  const [agree, setAgree] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    void fetchTranscriptConsentPreview(orderId, i18n.language)
      .then((p) => {
        if (!cancelled) setPreview(p)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : t('transcripts.consent.errorLoad'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [orderId, i18n.language, t])

  async function handleSign() {
    if (!preview) return
    const name = typedName.trim()
    if (name.length < 2) {
      setError(t('transcripts.consent.errorTypedName'))
      return
    }
    if (!agree) {
      setError(t('transcripts.consent.errorAgree'))
      return
    }
    setBusy(true)
    setError(null)
    try {
      const { order } = await signTranscriptConsent(orderId, {
        method: 'typed',
        signatureData: name,
        agree: true,
        locale: i18n.language,
      })
      onSigned(order)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.consent.errorSign'))
    } finally {
      setBusy(false)
    }
  }

  if (loading) {
    return <p className="mt-4 text-sm text-slate-500">{t('transcripts.consent.loading')}</p>
  }
  if (!preview) {
    return (
      <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
        {error ?? t('transcripts.consent.errorLoad')}
      </p>
    )
  }

  if (preview.requiresGuardian) {
    return (
      <div className="mt-5 space-y-3" role="status">
        <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-100">
          {t('transcripts.consent.guardianPending')}
        </p>
        {onCancel ? (
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium dark:border-neutral-700"
          >
            {t('transcripts.order.close')}
          </button>
        ) : null}
      </div>
    )
  }

  if (!preview.requiresConsent) {
    return (
      <p className="mt-5 text-sm text-slate-600 dark:text-neutral-400" role="status">
        {t('transcripts.consent.notRequired')}
      </p>
    )
  }

  return (
    <div className="mt-5 space-y-4">
      <p className="text-sm text-slate-600 dark:text-neutral-400">{t('transcripts.consent.help')}</p>

      <div>
        <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
          {t('transcripts.consent.recipientsHeading')}
        </h3>
        <ul className="mt-2 list-inside list-disc text-sm text-slate-700 dark:text-neutral-300">
          {preview.recipients.map((r) => (
            <li key={r.id}>
              {r.name} <span className="text-xs text-slate-500">({r.type})</span>
            </li>
          ))}
        </ul>
        <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
          {t('transcripts.consent.scopeLabel')}: {preview.scope}
        </p>
        <p className="text-xs text-slate-500 dark:text-neutral-400">
          {t('transcripts.consent.purposeLabel')}: {preview.purpose}
        </p>
      </div>

      <div>
        <h3 id={textId} className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
          {t('transcripts.consent.authorizationHeading')}
        </h3>
        <p className="mt-1 text-xs text-slate-500">
          {t('transcripts.consent.versionLabel', { version: preview.textVersion })}
        </p>
        <pre
          aria-labelledby={textId}
          className="mt-2 max-h-56 overflow-y-auto whitespace-pre-wrap rounded-md border border-slate-200 bg-slate-50 p-3 text-xs text-slate-800 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-200"
        >
          {preview.authorizationText}
        </pre>
      </div>

      <label htmlFor={typedId} className="block text-sm font-medium text-slate-800 dark:text-neutral-100">
        {t('transcripts.consent.typedSignature')}
        <input
          id={typedId}
          type="text"
          autoComplete="name"
          value={typedName}
          onChange={(e) => setTypedName(e.target.value)}
          className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
          aria-describedby={`${typedId}-hint`}
        />
        <span id={`${typedId}-hint`} className="mt-1 block text-xs text-slate-500">
          {t('transcripts.consent.typedHint')}
        </span>
      </label>

      <label htmlFor={agreeId} className="flex items-start gap-2 text-sm text-slate-800 dark:text-neutral-100">
        <input
          id={agreeId}
          type="checkbox"
          checked={agree}
          onChange={(e) => setAgree(e.target.checked)}
          className="mt-1 h-4 w-4 rounded border-slate-300 text-indigo-600"
        />
        <span>{t('transcripts.consent.agreeCheckbox')}</span>
      </label>

      <p className="text-xs text-slate-500 dark:text-neutral-400">
        {t('transcripts.consent.dateStamp', { date: new Date().toLocaleString() })}
      </p>

      {error ? (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      <div className="flex flex-wrap gap-2">
        {onCancel ? (
          <button
            type="button"
            onClick={onCancel}
            disabled={busy}
            className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium dark:border-neutral-700"
          >
            {t('transcripts.order.cancel')}
          </button>
        ) : null}
        <button
          type="button"
          onClick={() => void handleSign()}
          disabled={busy}
          className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
        >
          {busy ? t('transcripts.consent.signing') : t('transcripts.consent.sign')}
        </button>
      </div>
    </div>
  )
}

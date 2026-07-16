import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { TranscriptConsentForm } from './transcript-consent-form'
import {
  createTranscriptOrder,
  searchTranscriptRecipients,
  submitTranscriptOrder,
  type CreateOrderItemPayload,
  type TranscriptDeliveryMethod,
  type TranscriptDocument,
  type TranscriptOrderUrgency,
  type TranscriptRecipient,
  type TranscriptRecipientType,
} from '../../lib/transcripts-api'

const DELIVERY_METHODS: TranscriptDeliveryMethod[] = [
  'electronic_pesc',
  'electronic_pdf',
  'secure_link_email',
  'postal_mail',
  'api_peer',
]

const RECIPIENT_TYPES: Array<TranscriptRecipientType | ''> = [
  '',
  'institution',
  'application_service',
  'employer',
  'self',
  'other',
]

type DraftItem = {
  key: string
  recipient: TranscriptRecipient | null
  adHocName: string
  adHocEmail: string
  adHocAddress: string
  deliveryMethod: TranscriptDeliveryMethod
  urgency: TranscriptOrderUrgency
  documentId: string
}

type TranscriptOrderBuilderProps = {
  open: boolean
  submitting: boolean
  documents: TranscriptDocument[]
  onClose: () => void
  onSubmitted: (orderId: string) => void
}

function methodLabelKey(m: TranscriptDeliveryMethod): string {
  switch (m) {
    case 'electronic_pesc':
      return 'transcripts.delivery.electronicPesc'
    case 'electronic_pdf':
      return 'transcripts.delivery.electronicPdf'
    case 'secure_link_email':
      return 'transcripts.delivery.secureLinkEmail'
    case 'postal_mail':
      return 'transcripts.delivery.postalMail'
    case 'api_peer':
      return 'transcripts.delivery.apiPeer'
    default: {
      const _exhaustive: never = m
      return _exhaustive
    }
  }
}

function typeLabelKey(t: TranscriptRecipientType): string {
  switch (t) {
    case 'institution':
      return 'transcripts.recipientType.institution'
    case 'application_service':
      return 'transcripts.recipientType.applicationService'
    case 'employer':
      return 'transcripts.recipientType.employer'
    case 'self':
      return 'transcripts.recipientType.self'
    case 'other':
      return 'transcripts.recipientType.other'
    default: {
      const _exhaustive: never = t
      return _exhaustive
    }
  }
}

function newDraftItem(documentId = ''): DraftItem {
  return {
    key: crypto.randomUUID(),
    recipient: null,
    adHocName: '',
    adHocEmail: '',
    adHocAddress: '',
    deliveryMethod: 'secure_link_email',
    urgency: 'standard',
    documentId,
  }
}

export function TranscriptOrderBuilder({
  open,
  submitting,
  documents,
  onClose,
  onSubmitted,
}: TranscriptOrderBuilderProps) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const searchId = useId()
  const listboxId = useId()
  const [step, setStep] = useState<1 | 2 | 3 | 4>(1)
  const [items, setItems] = useState<DraftItem[]>([newDraftItem()])
  const [activeIdx, setActiveIdx] = useState(0)
  const [query, setQuery] = useState('')
  const [typeFilter, setTypeFilter] = useState<TranscriptRecipientType | ''>('')
  const [results, setResults] = useState<TranscriptRecipient[]>([])
  const [searchOpen, setSearchOpen] = useState(false)
  const [cursor, setCursor] = useState(0)
  const [searching, setSearching] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [pendingConsentOrderId, setPendingConsentOrderId] = useState<string | null>(null)
  const searchTimer = useRef<number | null>(null)

  useEffect(() => {
    if (!open) return
    const defaultDoc = documents.find((d) => d.variant === 'official')?.id ?? documents[0]?.id ?? ''
    setStep(1)
    setItems([newDraftItem(defaultDoc)])
    setActiveIdx(0)
    setQuery('')
    setTypeFilter('')
    setResults([])
    setFormError(null)
    setBusy(false)
    setPendingConsentOrderId(null)
  }, [open, documents])

  useEffect(() => {
    if (!open || submitting || busy) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, submitting, busy, onClose])

  useEffect(() => {
    if (!open || step !== 2) return
    if (searchTimer.current) window.clearTimeout(searchTimer.current)
    searchTimer.current = window.setTimeout(() => {
      void (async () => {
        setSearching(true)
        try {
          const list = await searchTranscriptRecipients({ q: query, type: typeFilter })
          setResults(list)
          setCursor(0)
          setSearchOpen(true)
        } catch {
          setResults([])
        } finally {
          setSearching(false)
        }
      })()
    }, 200)
    return () => {
      if (searchTimer.current) window.clearTimeout(searchTimer.current)
    }
  }, [open, step, query, typeFilter])

  if (!open) return null

  const active = items[activeIdx] ?? items[0]

  function updateActive(patch: Partial<DraftItem>) {
    setItems((prev) => prev.map((it, i) => (i === activeIdx ? { ...it, ...patch } : it)))
  }

  function pickRecipient(rec: TranscriptRecipient) {
    const preferred =
      rec.capabilities.find((c) => c === 'secure_link_email') ??
      rec.capabilities.find((c) => c === 'electronic_pdf') ??
      rec.capabilities[0] ??
      'secure_link_email'
    updateActive({
      recipient: rec,
      adHocName: '',
      adHocEmail: '',
      adHocAddress: '',
      deliveryMethod: preferred,
    })
    setSearchOpen(false)
    setQuery(rec.name)
    setFormError(null)
  }

  function sendToMyself() {
    const self = results.find((r) => r.type === 'self')
    if (self) {
      pickRecipient(self)
      return
    }
    void searchTranscriptRecipients({ type: 'self' }).then((list) => {
      const found = list.find((r) => r.type === 'self')
      if (found) pickRecipient(found)
    })
  }

  function allowedMethods(item: DraftItem): TranscriptDeliveryMethod[] {
    if (item.recipient) {
      return DELIVERY_METHODS.filter((m) => item.recipient!.capabilities.includes(m))
    }
    return DELIVERY_METHODS.filter((m) => m === 'secure_link_email' || m === 'postal_mail' || m === 'electronic_pdf')
  }

  function buildPayload(): CreateOrderItemPayload[] | string {
    if (items.length === 0) return t('transcripts.order.errorEmpty')
    const out: CreateOrderItemPayload[] = []
    for (const it of items) {
      const methods = allowedMethods(it)
      if (!methods.includes(it.deliveryMethod)) {
        return t('transcripts.order.errorInvalidMethod', { name: it.recipient?.name ?? it.adHocName })
      }
      const base: CreateOrderItemPayload = {
        deliveryMethod: it.deliveryMethod,
        urgency: it.urgency,
        documentId: it.documentId || undefined,
      }
      if (it.recipient) {
        out.push({ ...base, recipientId: it.recipient.id })
        continue
      }
      const name = it.adHocName.trim()
      if (!name) return t('transcripts.order.errorAdHocName')
      const adHoc: CreateOrderItemPayload['adHocRecipient'] = {
        type: 'other',
        name,
        capabilities: methods,
      }
      if (it.adHocEmail.trim()) adHoc.email = it.adHocEmail.trim()
      if (it.adHocAddress.trim()) adHoc.address = { raw: it.adHocAddress.trim() }
      if (it.deliveryMethod === 'postal_mail' && it.adHocAddress.trim().length < 10) {
        return t('transcripts.order.errorAddress')
      }
      out.push({ ...base, adHocRecipient: adHoc })
    }
    return out
  }

  async function handleSubmit() {
    const payload = buildPayload()
    if (typeof payload === 'string') {
      setFormError(payload)
      return
    }
    setBusy(true)
    setFormError(null)
    try {
      const draft = await createTranscriptOrder(payload)
      const submitted = await submitTranscriptOrder(draft.id)
      if (submitted.status === 'pending_consent') {
        setPendingConsentOrderId(submitted.id)
        setStep(4)
        return
      }
      onSubmitted(submitted.id)
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t('transcripts.order.errorSubmit'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center" role="presentation">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-5 shadow-xl dark:bg-neutral-900"
      >
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
              {t('transcripts.order.title')}
            </h2>
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
              {t(step === 4 ? 'transcripts.order.stepLabelConsent' : 'transcripts.order.stepLabel', {
                step: step === 4 ? 4 : step,
              })}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={busy || submitting}
            className="rounded p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
            aria-label={t('transcripts.order.close')}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {formError ? (
          <p role="alert" className="mt-3 text-sm text-red-600 dark:text-red-400">
            {formError}
          </p>
        ) : null}

        {step === 1 ? (
          <div className="mt-5 space-y-3">
            <p className="text-sm text-slate-600 dark:text-neutral-400">{t('transcripts.order.step1Help')}</p>
            <label className="block text-sm font-medium text-slate-800 dark:text-neutral-100">
              {t('transcripts.order.document')}
              <select
                className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                value={active.documentId}
                onChange={(e) => updateActive({ documentId: e.target.value })}
              >
                <option value="">{t('transcripts.order.documentNone')}</option>
                {documents.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.variant} · v{d.version}
                  </option>
                ))}
              </select>
            </label>
          </div>
        ) : null}

        {step === 2 ? (
          <div className="mt-5 space-y-4">
            <div className="flex flex-wrap gap-2">
              {items.map((it, i) => (
                <button
                  key={it.key}
                  type="button"
                  onClick={() => setActiveIdx(i)}
                  className={`rounded-full px-3 py-1 text-xs font-medium ${
                    i === activeIdx
                      ? 'bg-indigo-600 text-white'
                      : 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
                  }`}
                >
                  {it.recipient?.name || it.adHocName || t('transcripts.order.recipientN', { n: i + 1 })}
                </button>
              ))}
              <button
                type="button"
                onClick={() => {
                  setItems((prev) => [...prev, newDraftItem(active.documentId)])
                  setActiveIdx(items.length)
                  setQuery('')
                }}
                className="rounded-full border border-dashed border-slate-300 px-3 py-1 text-xs font-medium text-slate-600 dark:border-neutral-600 dark:text-neutral-300"
              >
                {t('transcripts.order.addRecipient')}
              </button>
            </div>

            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => sendToMyself()}
                className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-700"
              >
                {t('transcripts.order.sendToMyself')}
              </button>
              <label className="text-sm">
                <span className="sr-only">{t('transcripts.order.typeFilter')}</span>
                <select
                  value={typeFilter}
                  onChange={(e) => setTypeFilter(e.target.value as TranscriptRecipientType | '')}
                  className="rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                >
                  {RECIPIENT_TYPES.map((ty) => (
                    <option key={ty || 'all'} value={ty}>
                      {ty ? t(typeLabelKey(ty)) : t('transcripts.order.typeAll')}
                    </option>
                  ))}
                </select>
              </label>
            </div>

            <div className="relative">
              <label htmlFor={searchId} className="block text-sm font-medium text-slate-800 dark:text-neutral-100">
                {t('transcripts.order.searchLabel')}
              </label>
              <input
                id={searchId}
                type="search"
                role="combobox"
                aria-expanded={searchOpen}
                aria-controls={listboxId}
                aria-autocomplete="list"
                aria-activedescendant={results[cursor] ? `${listboxId}-opt-${cursor}` : undefined}
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value)
                  updateActive({ recipient: null })
                }}
                onFocus={() => setSearchOpen(true)}
                onKeyDown={(e) => {
                  if (!searchOpen || results.length === 0) return
                  if (e.key === 'ArrowDown') {
                    e.preventDefault()
                    setCursor((c) => Math.min(c + 1, results.length - 1))
                  } else if (e.key === 'ArrowUp') {
                    e.preventDefault()
                    setCursor((c) => Math.max(c - 1, 0))
                  } else if (e.key === 'Enter') {
                    e.preventDefault()
                    pickRecipient(results[cursor])
                  }
                }}
                placeholder={t('transcripts.order.searchPlaceholder')}
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
              />
              {searchOpen ? (
                <ul
                  id={listboxId}
                  role="listbox"
                  className="absolute z-10 mt-1 max-h-56 w-full overflow-auto rounded-md border border-slate-200 bg-white shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
                >
                  {searching ? (
                    <li className="px-3 py-2 text-sm text-slate-500">{t('common.loading')}</li>
                  ) : results.length === 0 ? (
                    <li className="px-3 py-2 text-sm text-slate-500">{t('transcripts.order.noResults')}</li>
                  ) : (
                    results.map((rec, i) => (
                      <li key={rec.id} role="option" id={`${listboxId}-opt-${i}`} aria-selected={i === cursor}>
                        <button
                          type="button"
                          className={`flex w-full flex-col items-start px-3 py-2 text-start text-sm ${
                            i === cursor ? 'bg-indigo-50 dark:bg-indigo-950' : ''
                          }`}
                          onMouseEnter={() => setCursor(i)}
                          onClick={() => pickRecipient(rec)}
                        >
                          <span className="font-medium text-slate-900 dark:text-neutral-50">{rec.name}</span>
                          <span className="text-xs text-slate-500">
                            {t(typeLabelKey(rec.type))}
                            {rec.verified ? ` · ${t('transcripts.order.verified')}` : ''}
                            {' · '}
                            {rec.capabilities.map((c) => t(methodLabelKey(c))).join(', ')}
                          </span>
                        </button>
                      </li>
                    ))
                  )}
                </ul>
              ) : null}
            </div>

            {!active.recipient ? (
              <fieldset className="space-y-2 rounded-md border border-slate-200 p-3 dark:border-neutral-700">
                <legend className="px-1 text-sm font-medium">{t('transcripts.order.adHocLegend')}</legend>
                <label className="block text-sm">
                  {t('transcripts.order.adHocName')}
                  <input
                    value={active.adHocName}
                    onChange={(e) => updateActive({ adHocName: e.target.value })}
                    className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                  />
                </label>
                <label className="block text-sm">
                  {t('transcripts.order.adHocEmail')}
                  <input
                    type="email"
                    value={active.adHocEmail}
                    onChange={(e) => updateActive({ adHocEmail: e.target.value })}
                    className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                  />
                </label>
                <label className="block text-sm">
                  {t('transcripts.order.adHocAddress')}
                  <textarea
                    value={active.adHocAddress}
                    onChange={(e) => updateActive({ adHocAddress: e.target.value })}
                    rows={3}
                    className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                  />
                </label>
              </fieldset>
            ) : null}

            <div className="grid gap-3 sm:grid-cols-2">
              <label className="block text-sm font-medium">
                {t('transcripts.order.deliveryMethod')}
                <select
                  value={active.deliveryMethod}
                  onChange={(e) => updateActive({ deliveryMethod: e.target.value as TranscriptDeliveryMethod })}
                  className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                >
                  {allowedMethods(active).map((m) => (
                    <option key={m} value={m}>
                      {t(methodLabelKey(m))}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block text-sm font-medium">
                {t('transcripts.order.urgency')}
                <select
                  value={active.urgency}
                  onChange={(e) => updateActive({ urgency: e.target.value as TranscriptOrderUrgency })}
                  className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                >
                  <option value="standard">{t('transcripts.order.urgencyStandard')}</option>
                  <option value="rush">{t('transcripts.order.urgencyRush')}</option>
                </select>
              </label>
            </div>
          </div>
        ) : null}

        {step === 3 ? (
          <div className="mt-5 space-y-3">
            <p className="text-sm text-slate-600 dark:text-neutral-400">{t('transcripts.order.reviewHelp')}</p>
            <ul className="divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
              {items.map((it) => (
                <li key={it.key} className="px-3 py-2 text-sm">
                  <p className="font-medium text-slate-900 dark:text-neutral-50">
                    {it.recipient?.name || it.adHocName || t('transcripts.order.unnamed')}
                  </p>
                  <p className="text-xs text-slate-500">
                    {t(methodLabelKey(it.deliveryMethod))} · {t(`transcripts.order.urgency${it.urgency === 'rush' ? 'Rush' : 'Standard'}`)}
                  </p>
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        {step === 4 && pendingConsentOrderId ? (
          <TranscriptConsentForm
            orderId={pendingConsentOrderId}
            onSigned={(order) => onSubmitted(order.id)}
            onCancel={onClose}
          />
        ) : null}

        {step < 4 ? (
          <div className="mt-6 flex flex-wrap justify-between gap-2">
            <button
              type="button"
              onClick={() => (step === 1 ? onClose() : setStep((s) => (s === 3 ? 2 : 1)))}
              disabled={busy || submitting}
              className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium dark:border-neutral-700"
            >
              {step === 1 ? t('transcripts.order.cancel') : t('transcripts.order.back')}
            </button>
            {step < 3 ? (
              <button
                type="button"
                onClick={() => {
                  if (step === 2) {
                    const err = buildPayload()
                    if (typeof err === 'string') {
                      setFormError(err)
                      return
                    }
                  }
                  setFormError(null)
                  setStep((s) => (s === 1 ? 2 : 3))
                }}
                className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500"
              >
                {t('transcripts.order.next')}
              </button>
            ) : (
              <button
                type="button"
                onClick={() => void handleSubmit()}
                disabled={busy || submitting}
                className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
              >
                {busy || submitting ? t('transcripts.order.submitting') : t('transcripts.order.submit')}
              </button>
            )}
          </div>
        ) : null}
      </div>
    </div>
  )
}

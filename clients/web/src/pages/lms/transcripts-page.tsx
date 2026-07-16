import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { TranscriptConsentForm } from '../../components/lms/transcript-consent-form'
import { TranscriptOrderBuilder } from '../../components/lms/transcript-order-builder'
import { deliveryTypeLabel, TranscriptRequestModal, urgencyLabel } from '../../components/lms/transcript-request-modal'
import { useConfirm } from '../../components/use-confirm'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { authorizedFetch } from '../../lib/api'
import { formatDate } from '../../lib/format'
import {
  exportTranscriptConsent,
  fetchTranscriptDocuments,
  fetchTranscriptOrders,
  fetchTranscriptPreview,
  fetchTranscriptRequests,
  fetchTranscriptsConfig,
  generateTranscriptDocument,
  revokeTranscriptConsent,
  saveTranscriptDocumentDownload,
  saveTranscriptPreviewPDF,
  submitTranscriptRequest,
  type AcademicRecord,
  type SubmitTranscriptRequestPayload,
  type TranscriptDocument,
  type TranscriptOrder,
  type TranscriptRequest,
  type TranscriptsStudentConfig,
} from '../../lib/transcripts-api'
import { LmsPage } from './lms-page'

function statusLabel(status: TranscriptRequest['status']): string {
  switch (status) {
    case 'queued':
      return 'Queued'
    case 'submitted':
      return 'Submitted to institution'
    case 'failed':
      return 'Failed'
    default: {
      const _exhaustive: never = status
      return _exhaustive
    }
  }
}

function statusClass(status: TranscriptRequest['status']): string {
  switch (status) {
    case 'submitted':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
    case 'failed':
      return 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300'
    case 'queued':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
    default: {
      const _exhaustive: never = status
      return _exhaustive
    }
  }
}

function variantLabel(variant: TranscriptDocument['variant']): string {
  switch (variant) {
    case 'official':
      return 'Official'
    case 'unofficial':
      return 'Unofficial'
    case 'partial':
      return 'Partial'
    case 'in_progress':
      return 'In progress'
    default: {
      const _exhaustive: never = variant
      return _exhaustive
    }
  }
}

export default function TranscriptsPage() {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const { ffTranscripts, loading: featuresLoading } = usePlatformFeatures()
  const [requests, setRequests] = useState<TranscriptRequest[]>([])
  const [orders, setOrders] = useState<TranscriptOrder[]>([])
  const [documents, setDocuments] = useState<TranscriptDocument[]>([])
  const [preview, setPreview] = useState<AcademicRecord | null>(null)
  const [transcriptConfig, setTranscriptConfig] = useState<TranscriptsStudentConfig | null>(null)
  const [defaultEmail, setDefaultEmail] = useState('')
  const [loading, setLoading] = useState(true)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [issuing, setIssuing] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [orderBuilderOpen, setOrderBuilderOpen] = useState(false)
  const [consentOrderId, setConsentOrderId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [list, docs, cfg, meRes, orderList] = await Promise.all([
        fetchTranscriptRequests(),
        fetchTranscriptDocuments(),
        fetchTranscriptsConfig(),
        authorizedFetch('/api/v1/me'),
        fetchTranscriptOrders().catch(() => [] as TranscriptOrder[]),
      ])
      setRequests(list)
      setDocuments(docs)
      setTranscriptConfig(cfg)
      setOrders(orderList)
      if (meRes.ok) {
        const me = (await meRes.json()) as { email?: string }
        if (me.email) setDefaultEmail(me.email)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load transcript requests.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffTranscripts) return
    void load()
  }, [featuresLoading, ffTranscripts, load])

  async function handlePreview() {
    setPreviewLoading(true)
    setError(null)
    try {
      const data = await fetchTranscriptPreview()
      setPreview(data.record)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load preview.')
    } finally {
      setPreviewLoading(false)
    }
  }

  async function handleIssueOfficial() {
    setIssuing(true)
    setMessage(null)
    setError(null)
    try {
      const { document } = await generateTranscriptDocument({ variant: 'official', format: ['pdf', 'xml'] })
      setMessage(`Official transcript version ${document.version} issued.`)
      setDocuments((prev) => [document, ...prev])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not issue official transcript.')
    } finally {
      setIssuing(false)
    }
  }

  async function handleSubmit(payload: SubmitTranscriptRequestPayload) {
    setSubmitting(true)
    setMessage(null)
    setError(null)
    try {
      const req = await submitTranscriptRequest(payload)
      setMessage('Your transcript request has been queued. We will notify your institution shortly.')
      setRequests((prev) => [req, ...prev])
      setModalOpen(false)
      window.setTimeout(() => void load(), 3000)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not submit request.')
    } finally {
      setSubmitting(false)
    }
  }

  if (featuresLoading) {
    return (
      <LmsPage title="Transcripts" description="Preview and request academic transcripts.">
        <p className="mt-8 text-sm text-slate-500">Loading…</p>
      </LmsPage>
    )
  }

  if (!ffTranscripts) {
    return (
      <LmsPage title="Transcripts" description="Preview and request academic transcripts.">
        <p className="mt-8 text-sm text-slate-600 dark:text-neutral-400">
          Transcripts is not enabled on this platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Transcripts" description="Preview unofficial records and manage issued transcripts.">
      <div className="mt-8 max-w-3xl space-y-10">
        {error && (
          <div role="alert" className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">
            {error}
          </div>
        )}
        {message && (
          <div role="status" className="rounded-md bg-blue-50 px-4 py-3 text-sm text-blue-700 dark:bg-blue-950 dark:text-blue-300">
            {message}
          </div>
        )}

        <section aria-labelledby="preview-heading">
          <h2 id="preview-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100">
            Unofficial transcript
          </h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Preview your academic record before ordering an official copy. Previews are watermarked and not stored as
            official artifacts.
          </p>
          <div className="mt-4 flex flex-wrap gap-3">
            <button
              type="button"
              onClick={() => void handlePreview()}
              disabled={previewLoading}
              className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
            >
              {previewLoading ? 'Loading…' : 'Preview unofficial transcript'}
            </button>
            <button
              type="button"
              onClick={() => {
                void saveTranscriptPreviewPDF().catch((e: unknown) => {
                  setError(e instanceof Error ? e.message : 'Could not download PDF.')
                })
              }}
              className="rounded-md border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-900"
            >
              Download unofficial PDF
            </button>
            {transcriptConfig?.officialEnabled ? (
              <button
                type="button"
                onClick={() => void handleIssueOfficial()}
                disabled={issuing}
                className="rounded-md border border-indigo-600 px-4 py-2 text-sm font-semibold text-indigo-700 hover:bg-indigo-50 disabled:opacity-50 dark:text-indigo-300 dark:hover:bg-indigo-950"
              >
                {issuing ? 'Issuing…' : 'Issue official transcript'}
              </button>
            ) : null}
          </div>

          {preview ? (
            <div className="mt-6 space-y-4 rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
              {preview.hasInProgress ? (
                <p role="status" className="text-sm text-amber-700 dark:text-amber-300">
                  This record includes in-progress courses that are not yet finalized.
                </p>
              ) : null}
              <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                {preview.student.name}
                {preview.student.studentId ? ` · ${preview.student.studentId}` : null}
              </p>
              <p className="text-xs text-slate-500 dark:text-neutral-400">{preview.institution.name}</p>
              {preview.terms.length === 0 ? (
                <p className="text-sm text-slate-500 dark:text-neutral-400">
                  No graded enrollments yet. Final grades must be submitted before courses appear on a transcript.
                </p>
              ) : (
                preview.terms.map((term) => (
                  <div key={`${term.label}-${term.startedOn ?? ''}`}>
                    <h3 className="text-sm font-semibold text-slate-800 dark:text-neutral-100">{term.label}</h3>
                    <table className="mt-2 w-full text-start text-sm">
                      <caption className="sr-only">Courses for {term.label}</caption>
                      <thead>
                        <tr className="border-b border-slate-200 text-xs text-slate-500 dark:border-neutral-700">
                          <th scope="col" className="py-1 pe-2 font-medium">Code</th>
                          <th scope="col" className="py-1 pe-2 font-medium">Title</th>
                          <th scope="col" className="py-1 pe-2 font-medium">Credits</th>
                          <th scope="col" className="py-1 font-medium">Grade</th>
                        </tr>
                      </thead>
                      <tbody>
                        {term.courses.map((c) => (
                          <tr key={`${c.code}-${c.title}`} className="border-b border-slate-100 dark:border-neutral-800">
                            <td className="py-1.5 pe-2 font-mono text-xs">{c.code}</td>
                            <td className="py-1.5 pe-2">{c.title}</td>
                            <td className="py-1.5 pe-2">{c.creditsEarned}</td>
                            <td className="py-1.5">{c.grade}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                      Term credits: {term.termCredits}
                      {term.termGpa != null ? ` · Term GPA: ${term.termGpa.toFixed(3)}` : null}
                    </p>
                  </div>
                ))
              )}
              <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
                Cumulative credits earned: {preview.cumulative.creditsEarned}
                {preview.cumulative.gpa != null ? ` · GPA: ${preview.cumulative.gpa.toFixed(3)}` : null}
              </p>
            </div>
          ) : null}
        </section>

        <section aria-labelledby="issued-heading">
          <h2 id="issued-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100">
            My issued documents
          </h2>
          {loading ? (
            <p className="mt-4 text-sm text-slate-500">Loading…</p>
          ) : documents.length === 0 ? (
            <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">No issued transcripts yet.</p>
          ) : (
            <ul className="mt-4 divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
              {documents.map((doc) => (
                <li key={doc.id} className="flex flex-wrap items-center justify-between gap-3 bg-white px-4 py-3 dark:bg-neutral-900">
                  <div>
                    <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                      {variantLabel(doc.variant)} · v{doc.version}
                    </p>
                    <p className="text-xs text-slate-500 dark:text-neutral-400">
                      {formatDate(doc.generatedAt, { dateStyle: 'medium', timeStyle: 'short' })}
                      {doc.gpaCumulative != null ? ` · GPA ${doc.gpaCumulative.toFixed(3)}` : null}
                      {doc.creditsEarned != null ? ` · ${doc.creditsEarned} credits` : null}
                    </p>
                  </div>
                  <div className="flex gap-2">
                    {doc.hasPdf ? (
                      <button
                        type="button"
                        onClick={() => {
                          void saveTranscriptDocumentDownload(doc.id, 'pdf').catch((e: unknown) => {
                            setError(e instanceof Error ? e.message : 'Could not download PDF.')
                          })
                        }}
                        className="text-sm font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                      >
                        PDF
                      </button>
                    ) : null}
                    {doc.hasXml ? (
                      <button
                        type="button"
                        onClick={() => {
                          void saveTranscriptDocumentDownload(doc.id, 'xml').catch((e: unknown) => {
                            setError(e instanceof Error ? e.message : 'Could not download XML.')
                          })
                        }}
                        className="text-sm font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                      >
                        XML
                      </button>
                    ) : null}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section aria-labelledby="request-heading">
          <h2 id="request-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100">
            {transcriptConfig?.ordersUiEnabled ? t('transcripts.order.sectionTitle') : 'Request delivery'}
          </h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            {transcriptConfig?.ordersUiEnabled
              ? t('transcripts.order.sectionHelp')
              : 'Submit a delivery request to your institution (email, mail, or pickup).'}
          </p>
          <div className="mt-4">
            <button
              type="button"
              onClick={() => {
                if (transcriptConfig?.ordersUiEnabled) setOrderBuilderOpen(true)
                else setModalOpen(true)
              }}
              disabled={loading}
              className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
            >
              {transcriptConfig?.ordersUiEnabled
                ? t('transcripts.order.newOrder')
                : 'Request transcript delivery'}
            </button>
          </div>
        </section>

        {transcriptConfig?.ordersUiEnabled ? (
          <section aria-labelledby="orders-heading">
            <h2 id="orders-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100">
              {t('transcripts.order.historyTitle')}
            </h2>
            {loading ? (
              <p className="mt-4 text-sm text-slate-500">{t('common.loading')}</p>
            ) : orders.length === 0 ? (
              <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">{t('transcripts.order.historyEmpty')}</p>
            ) : (
              <ul className="mt-4 divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
                {orders.map((o) => {
                  const statusKey = `transcripts.status.${o.status}`
                  const onHold = o.onHold || o.status === 'on_hold'
                  const rejected = o.status === 'rejected'
                  return (
                    <li key={o.id} className="bg-white px-4 py-3 dark:bg-neutral-900">
                      <div className="flex items-start justify-between gap-4">
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                            {t('transcripts.order.itemCount', { count: o.items.length })}
                          </p>
                          <p className="text-xs text-slate-500 dark:text-neutral-400">
                            {formatDate(o.createdAt, { dateStyle: 'medium', timeStyle: 'short' })}
                          </p>
                          <ul className="mt-2 space-y-1">
                            {o.items.map((it) => (
                              <li key={it.id} className="text-xs text-slate-600 dark:text-neutral-400">
                                {it.recipient?.name ?? t('transcripts.order.unnamed')} · {it.deliveryMethod} · {it.urgency}
                              </li>
                            ))}
                          </ul>
                          {onHold && o.studentMessage && (
                            <div
                              className="mt-3 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-950 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-100"
                              role="status"
                            >
                              <p className="font-semibold">{t('transcripts.order.onHoldBanner')}</p>
                              <p className="mt-1">{o.studentMessage}</p>
                              <p className="mt-1 text-amber-800 dark:text-amber-200">{t('transcripts.order.onHoldCta')}</p>
                            </div>
                          )}
                          {rejected && o.rejectionReason && (
                            <div
                              className="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-900 dark:border-red-900 dark:bg-red-950 dark:text-red-100"
                              role="status"
                            >
                              <p className="font-semibold">{t('transcripts.order.rejectedBanner')}</p>
                              <p className="mt-1">{o.rejectionReason}</p>
                            </div>
                          )}
                          {o.status === 'pending_consent' ? (
                            <div
                              className="mt-3 rounded-md border border-indigo-200 bg-indigo-50 px-3 py-2 text-xs text-indigo-950 dark:border-indigo-900 dark:bg-indigo-950 dark:text-indigo-100"
                              role="status"
                            >
                              <p className="font-semibold">{t('transcripts.consent.pendingBanner')}</p>
                              <p className="mt-1">{t('transcripts.consent.pendingHelp')}</p>
                              <button
                                type="button"
                                className="mt-2 font-semibold text-indigo-700 underline dark:text-indigo-300"
                                onClick={() => setConsentOrderId(o.id)}
                              >
                                {t('transcripts.consent.reviewAndSign')}
                              </button>
                            </div>
                          ) : null}
                          {o.consentId &&
                          o.status !== 'completed' &&
                          o.status !== 'canceled' &&
                          o.status !== 'rejected' &&
                          o.status !== 'pending_consent' ? (
                            <div className="mt-3 flex flex-wrap gap-3 text-xs">
                              <button
                                type="button"
                                className="font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                                onClick={() => {
                                  void exportTranscriptConsent(o.id, 'json')
                                    .then(() => setMessage(t('transcripts.consent.exportReady')))
                                    .catch((e: unknown) => {
                                      setError(e instanceof Error ? e.message : t('transcripts.consent.errorExport'))
                                    })
                                }}
                              >
                                {t('transcripts.consent.export')}
                              </button>
                              <button
                                type="button"
                                className="font-medium text-red-600 hover:underline dark:text-red-400"
                                onClick={() => {
                                  void (async () => {
                                    const ok = await confirm({
                                      title: t('transcripts.consent.revoke'),
                                      description: t('transcripts.consent.revokeConfirm'),
                                      variant: 'danger',
                                      confirmLabel: t('transcripts.consent.revoke'),
                                    })
                                    if (!ok) return
                                    try {
                                      await revokeTranscriptConsent(o.id)
                                      setMessage(t('transcripts.consent.revokeSuccess'))
                                      void load()
                                    } catch (e: unknown) {
                                      setError(e instanceof Error ? e.message : t('transcripts.consent.errorRevoke'))
                                    }
                                  })()
                                }}
                              >
                                {t('transcripts.consent.revoke')}
                              </button>
                            </div>
                          ) : null}
                        </div>
                        <span
                          className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium ${
                            onHold
                              ? 'bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-100'
                              : rejected
                                ? 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-100'
                                : o.status === 'pending_consent'
                                  ? 'bg-indigo-100 text-indigo-900 dark:bg-indigo-950 dark:text-indigo-100'
                                  : 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
                          }`}
                          aria-label={t('transcripts.status.aria', { status: o.status })}
                        >
                          {t(statusKey, { defaultValue: o.status })}
                        </span>
                      </div>
                    </li>
                  )
                })}
              </ul>
            )}
          </section>
        ) : (
          <section aria-labelledby="history-heading">
            <h2 id="history-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100">
              Delivery requests
            </h2>
            {loading ? (
              <p className="mt-4 text-sm text-slate-500">Loading…</p>
            ) : requests.length === 0 ? (
              <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">No transcript requests yet.</p>
            ) : (
              <ul className="mt-4 divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
                {requests.map((r) => {
                  const urgency = urgencyLabel(r.deliveryType, r.urgencyDays, r.urgencyUnit, r.urgencyDaysMin)
                  return (
                    <li key={r.id} className="flex items-start justify-between gap-4 bg-white px-4 py-3 dark:bg-neutral-900">
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                          {deliveryTypeLabel(r.deliveryType)}
                          {urgency ? ` · ${urgency}` : null}
                        </p>
                        <p className="text-xs text-slate-500 dark:text-neutral-400">
                          Submitted {formatDate(r.requestedAt, { dateStyle: 'medium', timeStyle: 'short' })}
                          {r.submittedAt
                            ? ` · Delivered ${formatDate(r.submittedAt, { dateStyle: 'medium', timeStyle: 'short' })}`
                            : null}
                        </p>
                        {r.deliveryEmail && (
                          <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">To: {r.deliveryEmail}</p>
                        )}
                        {r.deliveryAddress && (
                          <p className="mt-1 whitespace-pre-wrap text-xs text-slate-600 dark:text-neutral-400">
                            {r.deliveryAddress}
                          </p>
                        )}
                        {r.errorMessage && (
                          <p className="mt-1 text-xs text-red-600 dark:text-red-400">{r.errorMessage}</p>
                        )}
                      </div>
                      <span className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium ${statusClass(r.status)}`}>
                        {statusLabel(r.status)}
                      </span>
                    </li>
                  )
                })}
              </ul>
            )}
          </section>
        )}
      </div>

      <TranscriptRequestModal
        open={modalOpen}
        submitting={submitting}
        config={transcriptConfig}
        defaultEmail={defaultEmail}
        onClose={() => !submitting && setModalOpen(false)}
        onSubmit={(payload) => void handleSubmit(payload)}
      />
      <TranscriptOrderBuilder
        open={orderBuilderOpen}
        submitting={submitting}
        documents={documents}
        onClose={() => !submitting && setOrderBuilderOpen(false)}
        onSubmitted={() => {
          setMessage(t('transcripts.order.submitSuccess'))
          setOrderBuilderOpen(false)
          void load()
        }}
      />
      {consentOrderId ? (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center" role="presentation">
          <div
            role="dialog"
            aria-modal="true"
            aria-label={t('transcripts.consent.dialogTitle')}
            className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-5 shadow-xl dark:bg-neutral-900"
          >
            <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
              {t('transcripts.consent.dialogTitle')}
            </h2>
            <TranscriptConsentForm
              orderId={consentOrderId}
              onSigned={() => {
                setMessage(t('transcripts.consent.signSuccess'))
                setConsentOrderId(null)
                void load()
              }}
              onCancel={() => setConsentOrderId(null)}
            />
          </div>
        </div>
      ) : null}
      {ConfirmDialogHost}
    </LmsPage>
  )
}

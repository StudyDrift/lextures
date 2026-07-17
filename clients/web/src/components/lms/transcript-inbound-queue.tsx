import { useCallback, useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Inbox, Loader2 } from 'lucide-react'
import { authorizedFetch } from '../../lib/api'
import {
  acceptAdminTranscriptInbound,
  fetchAdminTranscriptInbound,
  fetchAdminTranscriptInboundCourses,
  fetchAdminTranscriptInboundDetail,
  matchAdminTranscriptInbound,
  rejectAdminTranscriptInbound,
  type TranscriptInboundCourse,
  type TranscriptInboundDocument,
  type TranscriptInboundEvent,
} from '../../lib/transcripts-api'

function inboundStatusClass(status: string): string {
  switch (status) {
    case 'quarantined':
    case 'rejected':
      return 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-100'
    case 'unmatched':
    case 'received':
      return 'bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-100'
    case 'matched':
    case 'parsed':
      return 'bg-sky-100 text-sky-900 dark:bg-sky-950 dark:text-sky-100'
    case 'accepted':
      return 'bg-emerald-100 text-emerald-900 dark:bg-emerald-950 dark:text-emerald-100'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
  }
}

type Props = {
  enabled: boolean
}

export function TranscriptInboundQueue({ enabled }: Props) {
  const { t } = useTranslation('common')
  const matchUserId = useId()
  const rejectReasonId = useId()
  const [statusFilter, setStatusFilter] = useState('')
  const [query, setQuery] = useState('')
  const [docs, setDocs] = useState<TranscriptInboundDocument[]>([])
  const [selected, setSelected] = useState<TranscriptInboundDocument | null>(null)
  const [events, setEvents] = useState<TranscriptInboundEvent[]>([])
  const [courses, setCourses] = useState<TranscriptInboundCourse[]>([])
  const [matchUser, setMatchUser] = useState('')
  const [rejectReason, setRejectReason] = useState('')
  const [loading, setLoading] = useState(false)
  const [acting, setActing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!enabled) return
    setLoading(true)
    setError(null)
    try {
      const list = await fetchAdminTranscriptInbound({
        status: statusFilter || undefined,
        q: query || undefined,
      })
      setDocs(list)
      setSelected((prev) => (prev ? (list.find((d) => d.id === prev.id) ?? prev) : null))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.inbound.loadError'))
    } finally {
      setLoading(false)
    }
  }, [enabled, statusFilter, query, t])

  useEffect(() => {
    void load()
  }, [load])

  async function selectDoc(doc: TranscriptInboundDocument) {
    setSelected(doc)
    setError(null)
    setMessage(null)
    try {
      const [detail, courseData] = await Promise.all([
        fetchAdminTranscriptInboundDetail(doc.id),
        fetchAdminTranscriptInboundCourses(doc.id).catch(() => ({ courses: [] as TranscriptInboundCourse[] })),
      ])
      setSelected(detail.document)
      setEvents(detail.events)
      setCourses(courseData.courses)
      setMatchUser(detail.document.matchedUserId ?? '')
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.inbound.loadError'))
    }
  }

  async function runMatch() {
    if (!selected || !matchUser.trim()) return
    setActing(true)
    setError(null)
    try {
      const updated = await matchAdminTranscriptInbound(selected.id, matchUser.trim())
      setSelected(updated)
      setMessage(t('transcripts.inbound.matchOk'))
      await load()
      await selectDoc(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.inbound.actionError'))
    } finally {
      setActing(false)
    }
  }

  async function runAccept() {
    if (!selected) return
    setActing(true)
    setError(null)
    try {
      const updated = await acceptAdminTranscriptInbound(selected.id)
      setSelected(updated)
      setMessage(t('transcripts.inbound.acceptOk'))
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.inbound.actionError'))
    } finally {
      setActing(false)
    }
  }

  async function runReject() {
    if (!selected || !rejectReason.trim()) {
      setError(t('transcripts.inbound.rejectReasonRequired'))
      return
    }
    setActing(true)
    setError(null)
    try {
      const updated = await rejectAdminTranscriptInbound(selected.id, rejectReason.trim())
      setSelected(updated)
      setRejectReason('')
      setMessage(t('transcripts.inbound.rejectOk'))
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.inbound.actionError'))
    } finally {
      setActing(false)
    }
  }

  async function openOriginal() {
    if (!selected) return
    try {
      const res = await authorizedFetch(
        `/api/v1/admin/transcripts/inbound/${encodeURIComponent(selected.id)}/original`,
      )
      if (!res.ok) throw new Error(t('transcripts.inbound.actionError'))
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      window.open(url, '_blank', 'noopener,noreferrer')
      setTimeout(() => URL.revokeObjectURL(url), 60_000)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('transcripts.inbound.actionError'))
    }
  }

  if (!enabled) return null

  return (
    <section className="mt-10 border-t border-slate-200 pt-8 dark:border-neutral-800" aria-labelledby="inbound-queue-heading">
      <div className="flex items-start gap-3">
        <Inbox className="mt-1 h-5 w-5 text-slate-500" aria-hidden />
        <div>
          <h2 id="inbound-queue-heading" className="text-xl font-semibold text-slate-900 dark:text-neutral-50">
            {t('transcripts.inbound.title')}
          </h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">{t('transcripts.inbound.subtitle')}</p>
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

      <div className="mt-4 flex flex-wrap items-end gap-3">
        <label className="text-sm">
          <span className="block text-slate-600 dark:text-neutral-400">{t('transcripts.inbound.filterStatus')}</span>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="mt-1 rounded-md border border-slate-300 bg-white px-2 py-1.5 dark:border-neutral-700 dark:bg-neutral-900"
          >
            <option value="">{t('transcripts.inbound.filterAll')}</option>
            <option value="unmatched">{t('transcripts.inbound.status.unmatched')}</option>
            <option value="matched">{t('transcripts.inbound.status.matched')}</option>
            <option value="accepted">{t('transcripts.inbound.status.accepted')}</option>
            <option value="rejected">{t('transcripts.inbound.status.rejected')}</option>
            <option value="quarantined">{t('transcripts.inbound.status.quarantined')}</option>
          </select>
        </label>
        <label className="text-sm">
          <span className="block text-slate-600 dark:text-neutral-400">{t('transcripts.inbound.filterSearch')}</span>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('transcripts.inbound.searchPlaceholder')}
            className="mt-1 w-56 rounded-md border border-slate-300 bg-white px-2 py-1.5 dark:border-neutral-700 dark:bg-neutral-900"
          />
        </label>
        <button
          type="button"
          onClick={() => void load()}
          className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-700"
        >
          {t('transcripts.inbound.refresh')}
        </button>
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        <div>
          <h3 className="text-sm font-medium text-slate-800 dark:text-neutral-200">{t('transcripts.inbound.queueTitle')}</h3>
          {loading ? (
            <p className="mt-3 flex items-center gap-2 text-sm text-slate-500">
              <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
              {t('transcripts.inbound.loading')}
            </p>
          ) : docs.length === 0 ? (
            <p className="mt-3 text-sm text-slate-500">{t('transcripts.inbound.queueEmpty')}</p>
          ) : (
            <ul className="mt-3 divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
              {docs.map((doc) => (
                <li key={doc.id}>
                  <button
                    type="button"
                    onClick={() => void selectDoc(doc)}
                    className={`w-full px-3 py-3 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-900 ${
                      selected?.id === doc.id ? 'bg-slate-50 dark:bg-neutral-900' : ''
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium text-slate-900 dark:text-neutral-50">
                        {doc.sourceName || t('transcripts.inbound.unknownSource')}
                      </span>
                      <span className={`rounded px-1.5 py-0.5 text-xs ${inboundStatusClass(doc.status)}`}>
                        {t(`transcripts.inbound.status.${doc.status}`, { defaultValue: doc.status })}
                      </span>
                    </div>
                    <p className="mt-1 text-xs text-slate-500">
                      {doc.studentName || t('transcripts.inbound.unknownStudent')} · {new Date(doc.receivedAt).toLocaleString()}
                    </p>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div>
          <h3 className="text-sm font-medium text-slate-800 dark:text-neutral-200">{t('transcripts.inbound.detailTitle')}</h3>
          {!selected ? (
            <p className="mt-3 text-sm text-slate-500">{t('transcripts.inbound.detailEmpty')}</p>
          ) : (
            <div className="mt-3 space-y-4 rounded-md border border-slate-200 p-4 dark:border-neutral-800">
              <dl className="grid grid-cols-2 gap-2 text-sm">
                <dt className="text-slate-500">{t('transcripts.inbound.fieldFormat')}</dt>
                <dd>{docFormatLabel(selected.format)}</dd>
                <dt className="text-slate-500">{t('transcripts.inbound.fieldConfidence')}</dt>
                <dd>{selected.matchConfidence != null ? selected.matchConfidence.toFixed(2) : '—'}</dd>
                <dt className="text-slate-500">{t('transcripts.inbound.fieldMatchedUser')}</dt>
                <dd className="break-all">{selected.matchedUserId || '—'}</dd>
              </dl>

              {selected.quarantineReason && (
                <p className="text-sm text-red-700 dark:text-red-300">
                  {t('transcripts.inbound.quarantineReason')}: {selected.quarantineReason}
                </p>
              )}

              <div>
                <label htmlFor={matchUserId} className="block text-sm text-slate-600 dark:text-neutral-400">
                  {t('transcripts.inbound.matchUser')}
                </label>
                <div className="mt-1 flex flex-wrap gap-2">
                  <input
                    id={matchUserId}
                    value={matchUser}
                    onChange={(e) => setMatchUser(e.target.value)}
                    placeholder={t('transcripts.inbound.matchUserPlaceholder')}
                    className="min-w-[14rem] flex-1 rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  />
                  <button
                    type="button"
                    disabled={acting}
                    onClick={() => void runMatch()}
                    className="rounded-md bg-slate-900 px-3 py-1.5 text-sm text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
                  >
                    {t('transcripts.inbound.match')}
                  </button>
                </div>
              </div>

              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  disabled={acting || !selected.matchedUserId || selected.status === 'accepted'}
                  onClick={() => void runAccept()}
                  className="rounded-md bg-emerald-700 px-3 py-1.5 text-sm text-white disabled:opacity-50"
                >
                  {t('transcripts.inbound.accept')}
                </button>
                <button
                  type="button"
                  onClick={() => void openOriginal()}
                  className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-700"
                >
                  {t('transcripts.inbound.viewOriginal')}
                </button>
              </div>

              <div>
                <label htmlFor={rejectReasonId} className="block text-sm text-slate-600 dark:text-neutral-400">
                  {t('transcripts.inbound.rejectReason')}
                </label>
                <div className="mt-1 flex flex-wrap gap-2">
                  <input
                    id={rejectReasonId}
                    value={rejectReason}
                    onChange={(e) => setRejectReason(e.target.value)}
                    className="min-w-[14rem] flex-1 rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  />
                  <button
                    type="button"
                    disabled={acting}
                    onClick={() => void runReject()}
                    className="rounded-md border border-red-300 px-3 py-1.5 text-sm text-red-800 dark:border-red-900 dark:text-red-200"
                  >
                    {t('transcripts.inbound.reject')}
                  </button>
                </div>
              </div>

              <div>
                <h4 className="text-sm font-medium">{t('transcripts.inbound.coursesTitle')}</h4>
                {courses.length === 0 ? (
                  <p className="mt-1 text-sm text-slate-500">{t('transcripts.inbound.coursesEmpty')}</p>
                ) : (
                  <ul className="mt-2 max-h-48 overflow-auto text-sm">
                    {courses.map((c, i) => (
                      <li key={`${c.code}-${i}`} className="border-t border-slate-100 py-1 dark:border-neutral-800">
                        <span className="font-medium">{c.code}</span> {c.title} — {c.grade} ({c.creditsEarned})
                      </li>
                    ))}
                  </ul>
                )}
              </div>

              <div>
                <h4 className="text-sm font-medium">{t('transcripts.inbound.eventsTitle')}</h4>
                <ul className="mt-2 max-h-40 overflow-auto text-xs text-slate-600 dark:text-neutral-400">
                  {events.map((e) => (
                    <li key={e.id} className="py-0.5">
                      {new Date(e.createdAt).toLocaleString()} · {e.eventType}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  )
}

function docFormatLabel(format: string): string {
  switch (format) {
    case 'pesc_xml':
      return 'PESC XML'
    case 'pdf':
      return 'PDF'
    case 'edi':
      return 'EDI'
    default:
      return format
  }
}

import { useCallback, useEffect, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createConsentStudy,
  exportConsentingParticipants,
  fetchConsentRecords,
  fetchConsentStudies,
  updateConsentStudy,
  type ConsentParticipant,
  type ConsentRecord,
  type ConsentStudyWithRate,
  type StudyStatus,
} from '../../lib/research-consent-api'
import { formatDateTime } from '../../lib/format'

function consentPercent(rate: { granted: number; declined: number; withdrawn: number }): number {
  const total = rate.granted + rate.declined + rate.withdrawn
  if (total === 0) return 0
  return Math.round((rate.granted / total) * 100)
}

function CreateStudyForm({ onCreated }: { onCreated: () => void }) {
  const [title, setTitle] = useState('')
  const [irbProtocol, setIrbProtocol] = useState('')
  const [consentText, setConsentText] = useState('')
  const [dataUseDescription, setDataUseDescription] = useState('')
  const [courseIds, setCourseIds] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      const ids = courseIds
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      await createConsentStudy({
        title,
        irbProtocol,
        consentText,
        dataUseDescription,
        targetCriteria: ids.length > 0 ? { courseIds: ids } : undefined,
      })
      setTitle('')
      setIrbProtocol('')
      setConsentText('')
      setDataUseDescription('')
      setCourseIds('')
      onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not create study.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={submit} className="space-y-3 rounded-xl border border-slate-200 p-4 dark:border-neutral-800">
      <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Create a consent study</h2>
      {error && (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
      <div>
        <label htmlFor="study-title" className="block text-sm font-medium">
          Title
        </label>
        <input
          id="study-title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          required
          className="mt-1 w-full rounded-lg border px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
        />
      </div>
      <div>
        <label htmlFor="study-irb" className="block text-sm font-medium">
          IRB protocol number
        </label>
        <input
          id="study-irb"
          value={irbProtocol}
          onChange={(e) => setIrbProtocol(e.target.value)}
          required
          className="mt-1 w-full rounded-lg border px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
        />
      </div>
      <div>
        <label htmlFor="study-consent" className="block text-sm font-medium">
          Consent form text (Markdown)
        </label>
        <textarea
          id="study-consent"
          value={consentText}
          onChange={(e) => setConsentText(e.target.value)}
          required
          rows={5}
          className="mt-1 w-full rounded-lg border px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
        />
      </div>
      <div>
        <label htmlFor="study-datause" className="block text-sm font-medium">
          Data-use description
        </label>
        <textarea
          id="study-datause"
          value={dataUseDescription}
          onChange={(e) => setDataUseDescription(e.target.value)}
          required
          rows={2}
          className="mt-1 w-full rounded-lg border px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
        />
      </div>
      <div>
        <label htmlFor="study-courses" className="block text-sm font-medium">
          Target course IDs <span className="text-slate-400">(comma-separated; blank = whole institution)</span>
        </label>
        <input
          id="study-courses"
          value={courseIds}
          onChange={(e) => setCourseIds(e.target.value)}
          className="mt-1 w-full rounded-lg border px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
        />
      </div>
      <button
        type="submit"
        disabled={submitting}
        className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
      >
        Create study (draft)
      </button>
    </form>
  )
}

function StudyRow({ item, onChanged }: { item: ConsentStudyWithRate; onChanged: () => void }) {
  const { study, consentRate } = item
  const [busy, setBusy] = useState(false)
  const [records, setRecords] = useState<ConsentRecord[] | null>(null)
  const [participants, setParticipants] = useState<ConsentParticipant[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function setStatus(status: StudyStatus) {
    setBusy(true)
    setError(null)
    try {
      await updateConsentStudy(study.id, { status })
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not update study.')
    } finally {
      setBusy(false)
    }
  }

  async function loadRecords() {
    setError(null)
    try {
      setRecords(await fetchConsentRecords(study.id))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load records.')
    }
  }

  async function loadExport() {
    setError(null)
    try {
      const res = await exportConsentingParticipants(study.id)
      setParticipants(res.participants)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not export participants.')
    }
  }

  return (
    <li className="rounded-xl border border-slate-200 p-4 dark:border-neutral-800">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">{study.title}</p>
          <p className="text-xs text-slate-500">
            IRB {study.irbProtocol} · <span className="uppercase">{study.status}</span> ·{' '}
            {consentPercent(consentRate)}% consent ({consentRate.granted} granted, {consentRate.declined} declined,{' '}
            {consentRate.withdrawn} withdrawn)
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {study.status !== 'active' && (
            <button
              type="button"
              disabled={busy}
              onClick={() => void setStatus('active')}
              className="rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
            >
              Activate
            </button>
          )}
          {study.status === 'active' && (
            <button
              type="button"
              disabled={busy}
              onClick={() => void setStatus('closed')}
              className="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium dark:border-neutral-700"
            >
              Close
            </button>
          )}
          <button
            type="button"
            onClick={() => void loadRecords()}
            className="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium dark:border-neutral-700"
          >
            Audit log
          </button>
          <button
            type="button"
            onClick={() => void loadExport()}
            className="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium dark:border-neutral-700"
          >
            Export consenting
          </button>
        </div>
      </div>

      {error && (
        <p role="alert" className="mt-2 text-xs text-red-600 dark:text-red-400">
          {error}
        </p>
      )}

      {records && (
        <div className="mt-3 overflow-x-auto">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Consent audit log</p>
          <table className="mt-1 w-full text-start text-xs">
            <thead>
              <tr className="text-slate-400">
                <th className="py-1 pe-3">Decision</th>
                <th className="py-1 pe-3">User</th>
                <th className="py-1 pe-3">IP address</th>
                <th className="py-1 pe-3">When</th>
              </tr>
            </thead>
            <tbody>
              {records.map((r) => (
                <tr key={r.id} className="border-t border-slate-100 dark:border-neutral-800">
                  <td className="py-1 pe-3">{r.decision}</td>
                  <td className="py-1 pe-3 font-mono">{r.userId.slice(0, 8)}</td>
                  <td className="py-1 pe-3">{r.ipAddress ?? '—'}</td>
                  <td className="py-1 pe-3">{formatDateTime(r.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {participants && (
        <div className="mt-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Consenting participants ({participants.length})
          </p>
          <ul className="mt-1 space-y-0.5 text-xs">
            {participants.map((p) => (
              <li key={p.userId}>
                {p.displayName ? `${p.displayName} · ` : ''}
                {p.email}
              </li>
            ))}
          </ul>
        </div>
      )}
    </li>
  )
}

export default function ConsentStudiesAdminPage() {
  const { ffResearchConsent, loading: featuresLoading } = usePlatformFeatures()
  const [studies, setStudies] = useState<ConsentStudyWithRate[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      setStudies(await fetchConsentStudies())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load studies.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffResearchConsent) {
      setLoading(false)
      return
    }
    void load()
  }, [ffResearchConsent, featuresLoading, load])

  if (!ffResearchConsent && !featuresLoading) {
    return (
      <div className="mx-auto max-w-4xl p-6">
        <h1 className="mb-2 text-xl font-semibold">Research consent studies</h1>
        <p className="text-sm text-slate-600">Research consent is not enabled for this platform.</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-6">
      <div>
        <h1 className="text-xl font-semibold">Research consent studies</h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Create IRB consent studies, monitor consent rates, and export data for consenting participants only.
        </p>
      </div>

      <CreateStudyForm onCreated={() => void load()} />

      {error && (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-slate-500">Loading…</p>
      ) : studies.length === 0 ? (
        <p className="text-sm text-slate-500">No studies yet.</p>
      ) : (
        <ul className="space-y-3">
          {studies.map((item) => (
            <StudyRow key={item.study.id} item={item} onChanged={() => void load()} />
          ))}
        </ul>
      )}
    </div>
  )
}

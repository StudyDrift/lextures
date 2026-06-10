import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { BookCopy } from 'lucide-react'
import { usePermissions } from '../../context/use-permissions'
import {
  fetchModuleTextbookResource,
  patchModuleTextbookResource,
  recordTextbookResourceAccess,
  fetchInclusiveAccess,
  configureInclusiveAccess,
  type TextbookResourcePayload,
  type TextbookResourceMeta,
  type InclusiveAccessStatus,
  type BookstoreProvider,
} from '../../lib/courses-api'
import { permCourseItemCreate } from '../../lib/rbac-api'
import { LmsPage } from './lms-page'
import { InclusiveAccessBanner } from './inclusive-access-banner'

const PROVIDER_LABELS: Record<BookstoreProvider, string> = {
  vitalsource: 'VitalSource',
  redshelf: 'RedShelf',
}

/** Best-effort reader deep link by ISBN. Chapter/page deep linking is handled by the LTI launch. */
function readerUrl(provider: BookstoreProvider, isbn: string): string {
  const clean = isbn.replace(/[^0-9Xx]/g, '')
  if (provider === 'redshelf') return `https://platform.virdocs.com/read/${encodeURIComponent(clean)}`
  return `https://bookshelf.vitalsource.com/reader/books/${encodeURIComponent(clean)}`
}

export default function CourseModuleTextbookResourcePage() {
  const { courseCode, itemId } = useParams<{ courseCode: string; itemId: string }>()
  const { allows, loading: permLoading } = usePermissions()

  const [data, setData] = useState<TextbookResourcePayload | null>(null)
  const [ia, setIa] = useState<InclusiveAccessStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [meta, setMeta] = useState<TextbookResourceMeta>({})
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  // Inclusive Access editor (instructor).
  const [iaIsbn, setIaIsbn] = useState('')
  const [iaTitle, setIaTitle] = useState('')
  const [iaOptOutUrl, setIaOptOutUrl] = useState('')
  const [iaProvider, setIaProvider] = useState<BookstoreProvider>('vitalsource')
  const [iaEnabled, setIaEnabled] = useState(false)
  const [iaSaving, setIaSaving] = useState(false)
  const [iaError, setIaError] = useState<string | null>(null)

  const canEdit = Boolean(
    courseCode && itemId && !permLoading && allows(permCourseItemCreate(courseCode)),
  )

  const load = useCallback(async () => {
    if (!courseCode || !itemId) return
    setLoading(true)
    setLoadError(null)
    try {
      const [row, iaStatus] = await Promise.all([
        fetchModuleTextbookResource(courseCode, itemId),
        fetchInclusiveAccess(courseCode).catch(() => ({ enabled: false }) as InclusiveAccessStatus),
      ])
      if (!row) {
        setLoadError('This textbook resource could not be found.')
        setData(null)
        return
      }
      setData(row)
      setMeta(row.metadata ?? {})
      setIa(iaStatus)
      setIaEnabled(iaStatus.enabled)
      setIaIsbn(iaStatus.isbn ?? row.metadata?.isbn ?? '')
      setIaTitle(iaStatus.title ?? row.metadata?.title ?? '')
      setIaOptOutUrl(iaStatus.optOutUrl ?? '')
      setIaProvider(iaStatus.provider ?? row.provider ?? 'vitalsource')
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : 'Could not load this textbook.')
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    void load()
  }, [load])

  async function onOpen() {
    if (!courseCode || !itemId || !data) return
    // Record the anonymized COUNTER launch event (best-effort).
    void recordTextbookResourceAccess(courseCode, itemId)
    const isbn = (data.metadata?.isbn ?? '').trim()
    if (isbn) {
      try {
        window.open(readerUrl(data.provider, isbn), '_blank', 'noopener,noreferrer')
      } catch {
        /* ignore */
      }
    }
  }

  async function onSaveMeta(e: React.FormEvent) {
    e.preventDefault()
    if (!courseCode || !itemId) return
    setSaving(true)
    setSaveError(null)
    setSaved(false)
    try {
      const row = await patchModuleTextbookResource(courseCode, itemId, { metadata: meta })
      setData(row)
      setMeta(row.metadata ?? {})
      setSaved(true)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Could not save textbook details.')
    } finally {
      setSaving(false)
    }
  }

  async function onSaveIa(e: React.FormEvent) {
    e.preventDefault()
    if (!courseCode) return
    setIaSaving(true)
    setIaError(null)
    try {
      const status = await configureInclusiveAccess(courseCode, {
        isbn: iaIsbn.trim(),
        title: iaTitle.trim(),
        optOutUrl: iaOptOutUrl.trim(),
        provider: iaProvider,
        enabled: iaEnabled,
      })
      setIa(status)
    } catch (err) {
      setIaError(err instanceof Error ? err.message : 'Could not save Inclusive Access settings.')
    } finally {
      setIaSaving(false)
    }
  }

  const modulesHref =
    courseCode != null && courseCode !== ''
      ? `/courses/${encodeURIComponent(courseCode)}/modules`
      : '/courses'

  const fieldClasses =
    'mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100'

  return (
    <LmsPage title={data?.metadata?.title || 'Textbook'}>
      <div className="mx-auto max-w-2xl">
        <p className="mb-4 text-sm text-slate-600 dark:text-neutral-400">
          <Link to={modulesHref} className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400">
            ← Modules
          </Link>
        </p>

        {ia ? <InclusiveAccessBanner status={ia} storageKey={courseCode} /> : null}

        {loading && <p className="text-sm text-slate-600 dark:text-neutral-400">Loading…</p>}
        {loadError && (
          <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
            {loadError}
          </p>
        )}

        {!loading && !loadError && data && (
          <div className="rounded-2xl border border-slate-200/80 bg-white/90 p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900/85">
            <div className="flex items-start gap-3">
              <span className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-orange-200/90 bg-orange-50 text-orange-700 dark:border-orange-500/40 dark:bg-orange-950/55 dark:text-orange-200">
                <BookCopy className="h-5 w-5" strokeWidth={2} aria-hidden />
              </span>
              <div className="min-w-0 flex-1">
                <span className="inline-flex items-center rounded-full border border-orange-200/80 bg-orange-50 px-2.5 py-0.5 text-xs font-medium text-orange-700 dark:border-orange-500/40 dark:bg-orange-950/55 dark:text-orange-200">
                  {PROVIDER_LABELS[data.provider]}
                </span>
                {data.metadata?.title ? (
                  <h1 className="mt-2 text-lg font-semibold text-slate-900 dark:text-neutral-100">
                    {data.metadata.title}
                  </h1>
                ) : null}
                <dl className="mt-2 grid grid-cols-1 gap-x-6 gap-y-1 text-sm text-slate-600 sm:grid-cols-2 dark:text-neutral-400">
                  {data.metadata?.isbn ? (
                    <div>
                      <dt className="inline font-medium text-slate-700 dark:text-neutral-300">ISBN: </dt>
                      <dd className="inline font-mono">{data.metadata.isbn}</dd>
                    </div>
                  ) : null}
                  {data.metadata?.edition ? (
                    <div>
                      <dt className="inline font-medium text-slate-700 dark:text-neutral-300">Edition: </dt>
                      <dd className="inline">{data.metadata.edition}</dd>
                    </div>
                  ) : null}
                  {data.metadata?.publisher ? (
                    <div>
                      <dt className="inline font-medium text-slate-700 dark:text-neutral-300">Publisher: </dt>
                      <dd className="inline">{data.metadata.publisher}</dd>
                    </div>
                  ) : null}
                  {data.metadata?.chapter ? (
                    <div>
                      <dt className="inline font-medium text-slate-700 dark:text-neutral-300">Chapter: </dt>
                      <dd className="inline">{data.metadata.chapter}</dd>
                    </div>
                  ) : null}
                  {data.metadata?.pageRange ? (
                    <div>
                      <dt className="inline font-medium text-slate-700 dark:text-neutral-300">Pages: </dt>
                      <dd className="inline">{data.metadata.pageRange}</dd>
                    </div>
                  ) : null}
                </dl>

                <div className="mt-4">
                  <button
                    type="button"
                    onClick={() => void onOpen()}
                    aria-label={`Open ${data.metadata?.title || 'textbook'}${data.metadata?.chapter ? `, ${data.metadata.chapter}` : ''} in ${PROVIDER_LABELS[data.provider]}`}
                    className="inline-flex items-center justify-center rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-indigo-500"
                  >
                    Open in {PROVIDER_LABELS[data.provider]}
                  </button>
                  {!data.metadata?.isbn ? (
                    <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
                      An ISBN has not been configured yet.
                    </p>
                  ) : null}
                </div>

                {canEdit && (
                  <form
                    className="mt-6 space-y-4 border-t border-slate-200 pt-6 dark:border-neutral-700"
                    onSubmit={onSaveMeta}
                  >
                    <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                      Textbook details
                    </h2>
                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        Title
                        <input
                          type="text"
                          value={meta.title ?? ''}
                          onChange={(e) => setMeta((m) => ({ ...m, title: e.target.value }))}
                          disabled={saving}
                          className={fieldClasses}
                        />
                      </label>
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        ISBN
                        <input
                          type="text"
                          value={meta.isbn ?? ''}
                          onChange={(e) => setMeta((m) => ({ ...m, isbn: e.target.value }))}
                          disabled={saving}
                          className={fieldClasses}
                        />
                      </label>
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        Edition
                        <input
                          type="text"
                          value={meta.edition ?? ''}
                          onChange={(e) => setMeta((m) => ({ ...m, edition: e.target.value }))}
                          disabled={saving}
                          className={fieldClasses}
                        />
                      </label>
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        Publisher
                        <input
                          type="text"
                          value={meta.publisher ?? ''}
                          onChange={(e) => setMeta((m) => ({ ...m, publisher: e.target.value }))}
                          disabled={saving}
                          className={fieldClasses}
                        />
                      </label>
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        Chapter
                        <input
                          type="text"
                          value={meta.chapter ?? ''}
                          onChange={(e) => setMeta((m) => ({ ...m, chapter: e.target.value }))}
                          disabled={saving}
                          className={fieldClasses}
                        />
                      </label>
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        Page range
                        <input
                          type="text"
                          value={meta.pageRange ?? ''}
                          onChange={(e) => setMeta((m) => ({ ...m, pageRange: e.target.value }))}
                          disabled={saving}
                          className={fieldClasses}
                        />
                      </label>
                    </div>
                    {saveError ? (
                      <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
                        {saveError}
                      </p>
                    ) : null}
                    {saved ? (
                      <p className="text-sm text-emerald-700 dark:text-emerald-300" role="status">
                        Saved.
                      </p>
                    ) : null}
                    <div className="flex justify-end">
                      <button
                        type="submit"
                        disabled={saving}
                        className="rounded-xl bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-white"
                      >
                        {saving ? 'Saving…' : 'Save details'}
                      </button>
                    </div>
                  </form>
                )}

                {canEdit && (
                  <form
                    className="mt-6 space-y-4 border-t border-slate-200 pt-6 dark:border-neutral-700"
                    onSubmit={onSaveIa}
                  >
                    <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                      Inclusive Access
                    </h2>
                    <p className="text-xs text-slate-500 dark:text-neutral-400">
                      When enabled, students see an opt-out banner linking to the bookstore opt-out form.
                    </p>
                    <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
                      <input
                        type="checkbox"
                        checked={iaEnabled}
                        onChange={(e) => setIaEnabled(e.target.checked)}
                        disabled={iaSaving}
                      />
                      Inclusive Access enabled for this course
                    </label>
                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        Title
                        <input
                          type="text"
                          value={iaTitle}
                          onChange={(e) => setIaTitle(e.target.value)}
                          disabled={iaSaving}
                          className={fieldClasses}
                        />
                      </label>
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        ISBN
                        <input
                          type="text"
                          value={iaIsbn}
                          onChange={(e) => setIaIsbn(e.target.value)}
                          disabled={iaSaving}
                          className={fieldClasses}
                        />
                      </label>
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        Opt-out URL
                        <input
                          type="url"
                          value={iaOptOutUrl}
                          onChange={(e) => setIaOptOutUrl(e.target.value)}
                          disabled={iaSaving}
                          placeholder="https://bookstore.example.edu/opt-out"
                          className={fieldClasses}
                        />
                      </label>
                      <label className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        Provider
                        <select
                          value={iaProvider}
                          onChange={(e) => setIaProvider(e.target.value as BookstoreProvider)}
                          disabled={iaSaving}
                          className={fieldClasses}
                        >
                          <option value="vitalsource">VitalSource</option>
                          <option value="redshelf">RedShelf</option>
                        </select>
                      </label>
                    </div>
                    {iaError ? (
                      <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
                        {iaError}
                      </p>
                    ) : null}
                    <div className="flex justify-end">
                      <button
                        type="submit"
                        disabled={iaSaving}
                        className="rounded-xl bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-white"
                      >
                        {iaSaving ? 'Saving…' : 'Save Inclusive Access'}
                      </button>
                    </div>
                  </form>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </LmsPage>
  )
}

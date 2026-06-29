import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Download, FileUp, Loader2 } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  CSV_TEMPLATE,
  fetchImportJobStatus,
  importResultUrl,
  uploadUserImport,
  type ImportJobStatus,
  type ImportRowError,
} from '../../lib/admin-console-api'

const POLL_MS = 3000

type MergeStrategy = 'create_only' | 'upsert' | 'sync'
type ImportProfile = 'lextures_native' | 'oneroster_v1.2'

export default function AdminImport() {
  const fileInputId = useId()
  const liveRegionId = useId()
  const fileRef = useRef<HTMLInputElement>(null)
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const { bulkCsvImportEnabled } = usePlatformFeatures()

  const [file, setFile] = useState<File | null>(null)
  const [mergeStrategy, setMergeStrategy] = useState<MergeStrategy>('upsert')
  const [profile, setProfile] = useState<ImportProfile>('lextures_native')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [parseErrors, setParseErrors] = useState<ImportRowError[]>([])
  const [job, setJob] = useState<ImportJobStatus | null>(null)
  const [liveMessage, setLiveMessage] = useState('')

  const pollJob = useCallback(
    async (jobId: string) => {
      const status = await fetchImportJobStatus(jobId, orgId)
      setJob(status)
      const total = status.totalRows ?? 0
      const pct = total > 0 ? Math.round((status.processedRows / total) * 100) : 0
      setLiveMessage(
        status.status === 'complete'
          ? `Import complete. Created ${status.createdCount}, updated ${status.updatedCount}, deactivated ${status.deactivatedCount}, errors ${status.errorRows}.`
          : `Import ${status.status}: ${pct}% (${status.processedRows} of ${total} rows).`,
      )
      return status
    },
    [orgId],
  )

  useEffect(() => {
    if (!job || job.status === 'complete' || job.status === 'failed') return
    const id = window.setInterval(() => {
      void pollJob(job.jobId).catch(() => undefined)
    }, POLL_MS)
    return () => window.clearInterval(id)
  }, [job, pollJob])

  async function runImport(dryRun: boolean) {
    if (!file) {
      setError('Choose a CSV file to upload.')
      return
    }
    setBusy(true)
    setError(null)
    setParseErrors([])
    setJob(null)
    try {
      const res = await uploadUserImport(file, { mergeStrategy, profile, dryRun, orgId })
      setParseErrors(res.parseErrors ?? [])
      const status = await pollJob(res.jobId)
      setJob(status)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed.')
    } finally {
      setBusy(false)
    }
  }

  function downloadTemplate() {
    const blob = new Blob([CSV_TEMPLATE], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'users-template.csv'
    a.click()
    URL.revokeObjectURL(url)
  }

  if (!bulkCsvImportEnabled) {
    return (
      <div className="p-6 text-sm text-slate-600 dark:text-slate-400">
        Bulk user CSV import is not enabled for this platform.
      </div>
    )
  }

  const allErrors = [...parseErrors, ...(job?.errors ?? [])]

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-4 md:p-6">
      <header>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">Import users</h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
          Upload a CSV to create, update, or deactivate users in your organization.
        </p>
      </header>

      <div id={liveRegionId} className="sr-only" aria-live="polite" aria-atomic="true">
        {liveMessage}
      </div>

      <section className="rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
        <div className="grid gap-4 md:grid-cols-2">
          <label className="block text-sm">
            <span className="mb-1 block font-medium text-slate-700 dark:text-slate-300">Import profile</span>
            <select
              className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
              value={profile}
              onChange={(e) => setProfile(e.target.value as ImportProfile)}
            >
              <option value="lextures_native">Lextures native</option>
              <option value="oneroster_v1.2">OneRoster 1.2</option>
            </select>
          </label>
          <label className="block text-sm">
            <span className="mb-1 block font-medium text-slate-700 dark:text-slate-300">Merge strategy</span>
            <select
              className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
              value={mergeStrategy}
              onChange={(e) => setMergeStrategy(e.target.value as MergeStrategy)}
            >
              <option value="create_only">Create only</option>
              <option value="upsert">Upsert (create + update)</option>
              <option value="sync">Sync (upsert + deactivate missing)</option>
            </select>
          </label>
        </div>

        <div className="mt-4">
          <label htmlFor={fileInputId} className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
            CSV file
          </label>
          <input
            id={fileInputId}
            ref={fileRef}
            type="file"
            accept=".csv,text/csv"
            className="block w-full text-sm"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
          <p className="mt-2 text-xs text-slate-500">
            No imports yet?{' '}
            <button type="button" className="text-blue-600 underline" onClick={downloadTemplate}>
              Download CSV template
            </button>
          </p>
        </div>

        {error ? (
          <p className="mt-3 text-sm text-red-600" role="alert">
            {error}
          </p>
        ) : null}

        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            disabled={busy}
            className="inline-flex items-center gap-2 rounded-md bg-slate-100 px-3 py-2 text-sm font-medium text-slate-900 hover:bg-slate-200 disabled:opacity-50 dark:bg-neutral-800 dark:text-slate-100"
            onClick={() => void runImport(true)}
          >
            {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <FileUp className="h-4 w-4" aria-hidden />}
            Dry run
          </button>
          <button
            type="button"
            disabled={busy}
            className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            onClick={() => void runImport(false)}
          >
            {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <FileUp className="h-4 w-4" aria-hidden />}
            Import
          </button>
        </div>
      </section>

      {job ? (
        <section className="rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Job status</h2>
          <p className="mt-1 text-sm capitalize text-slate-600 dark:text-slate-400">
            {job.status}
            {job.totalRows != null ? ` — ${job.processedRows} / ${job.totalRows} rows` : ''}
          </p>
          {job.status !== 'complete' && job.status !== 'failed' ? (
            <div className="mt-3 h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-neutral-800">
              <div
                className="h-full bg-blue-600 transition-all"
                style={{
                  width: `${job.totalRows ? Math.min(100, Math.round((job.processedRows / job.totalRows) * 100)) : 0}%`,
                }}
                role="progressbar"
                aria-valuenow={job.processedRows}
                aria-valuemin={0}
                aria-valuemax={job.totalRows ?? 0}
                aria-label="Import progress"
              />
            </div>
          ) : null}
          {job.status === 'complete' ? (
            <dl className="mt-3 grid grid-cols-2 gap-2 text-sm md:grid-cols-4">
              <div>
                <dt className="text-slate-500">Created</dt>
                <dd className="font-medium">{job.createdCount}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Updated</dt>
                <dd className="font-medium">{job.updatedCount}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Deactivated</dt>
                <dd className="font-medium">{job.deactivatedCount}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Errors</dt>
                <dd className="font-medium">{job.errorRows}</dd>
              </div>
            </dl>
          ) : null}
          {job.hasResult && job.status === 'complete' && !job.dryRun ? (
            <a
              href={importResultUrl(job.jobId, orgId)}
              className="mt-3 inline-flex items-center gap-1 text-sm text-blue-600 underline"
            >
              <Download className="h-4 w-4" aria-hidden />
              Download result CSV
            </a>
          ) : null}
        </section>
      ) : null}

      {allErrors.length > 0 ? (
        <section className="overflow-x-auto rounded-lg border border-slate-200 dark:border-neutral-800">
          <table className="min-w-full text-left text-sm">
            <caption className="sr-only">Import validation errors</caption>
            <thead className="sticky top-0 bg-slate-50 dark:bg-neutral-900">
              <tr>
                <th scope="col" className="px-3 py-2 font-medium">
                  Row
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  Column
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  Error
                </th>
              </tr>
            </thead>
            <tbody>
              {allErrors.map((e, i) => (
                <tr key={`${e.row}-${e.column}-${i}`} className="border-t border-slate-100 dark:border-neutral-800">
                  <td className="px-3 py-2">{e.row}</td>
                  <td className="px-3 py-2">{e.column}</td>
                  <td className="px-3 py-2">{e.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}
    </div>
  )
}

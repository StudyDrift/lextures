import { Download } from 'lucide-react'
import { formatDateTime } from '../../lib/format'
import { splitFilename } from '../../lib/file-type'
import { downloadAuthorizedFile } from '../../lib/download-file'

type SubmissionFileDetailsPanelProps = {
  filename: string
  filePath: string | null
  submittedAt?: string | null
  blindLabel?: string | null
  mimeType?: string | null
}

function extensionFromMime(mimeType: string | null | undefined): string | null {
  const mt = (mimeType ?? '').toLowerCase().trim()
  if (!mt) return null
  const map: Record<string, string> = {
    'image/png': 'png',
    'image/jpeg': 'jpg',
    'image/gif': 'gif',
    'image/webp': 'webp',
    'application/pdf': 'pdf',
    'text/plain': 'txt',
  }
  return map[mt] ?? null
}

export function SubmissionFileDetailsPanel({
  filename,
  filePath,
  submittedAt,
  blindLabel,
  mimeType,
}: SubmissionFileDetailsPanelProps) {
  const { name, extension: extFromName } = splitFilename(filename)
  const extension = extFromName ?? extensionFromMime(mimeType)
  const hasFile = Boolean(filePath)

  async function handleDownload() {
    if (!filePath) return
    try {
      await downloadAuthorizedFile(filePath, filename)
    } catch {
      /* noop */
    }
  }

  return (
    <div className="flex flex-col gap-5 p-5" aria-label="Submission file details">
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          Submission file
        </h3>
        {hasFile ? (
          <dl className="mt-3 space-y-3 text-sm">
            <div>
              <dt className="text-xs font-medium text-slate-500 dark:text-neutral-400">File name</dt>
              <dd className="mt-1 break-words font-medium leading-snug text-slate-900 dark:text-neutral-100">
                {name}
                {extension ? `.${extension}` : ''}
              </dd>
            </div>
            {extension ? (
              <div>
                <dt className="text-xs font-medium text-slate-500 dark:text-neutral-400">Extension</dt>
                <dd className="mt-1 font-mono text-slate-700 dark:text-neutral-300">.{extension}</dd>
              </div>
            ) : null}
            {mimeType ? (
              <div>
                <dt className="text-xs font-medium text-slate-500 dark:text-neutral-400">Type</dt>
                <dd className="mt-1 break-all font-mono text-xs text-slate-600 dark:text-neutral-400">
                  {mimeType}
                </dd>
              </div>
            ) : null}
          </dl>
        ) : (
          <p className="mt-3 text-sm text-slate-600 dark:text-neutral-400">No file attached to this submission.</p>
        )}
      </div>

      {blindLabel ? (
        <p className="rounded-lg border border-indigo-200 bg-indigo-50/90 px-3 py-2 text-sm text-indigo-950 dark:border-indigo-900/60 dark:bg-indigo-950/40 dark:text-indigo-100">
          {blindLabel}
        </p>
      ) : null}

      {submittedAt ? (
        <div className="text-sm">
          <p className="text-xs font-medium text-slate-500 dark:text-neutral-400">Submitted</p>
          <p className="mt-1 text-slate-800 dark:text-neutral-200">
            {formatDateTime(submittedAt, { dateStyle: 'medium', timeStyle: 'short' })}
          </p>
        </div>
      ) : null}

      {hasFile ? (
        <button
          type="button"
          onClick={() => void handleDownload()}
          className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:hover:bg-neutral-900"
        >
          <Download className="h-4 w-4" aria-hidden="true" />
          Download file
        </button>
      ) : null}
    </div>
  )
}

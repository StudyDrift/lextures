import { Download } from 'lucide-react'
import { formatDateTime } from '../../lib/format'
import { splitFilename } from '../../lib/file-type'
import { downloadAuthorizedFile } from '../../lib/download-file'
import type { SubmissionAttachmentApi } from '../../lib/courses-api'

type SubmissionFileDetailsPanelProps = {
  files: SubmissionAttachmentApi[]
  selectedFileId: string | null
  onSelectFile: (fileId: string) => void
  submittedAt?: string | null
  blindLabel?: string | null
  onDownloadAll?: () => void
  downloadAllBusy?: boolean
}

function displayLabel(filename: string): string {
  const trimmed = filename.trim()
  if (trimmed) return trimmed
  return 'submission'
}

export function SubmissionFileDetailsPanel({
  files,
  selectedFileId,
  onSelectFile,
  submittedAt,
  blindLabel,
  onDownloadAll,
  downloadAllBusy = false,
}: SubmissionFileDetailsPanelProps) {
  const hasFiles = files.length > 0
  const activeFileId = selectedFileId ?? files[0]?.fileId ?? null

  async function handleDownload(file: SubmissionAttachmentApi) {
    try {
      await downloadAuthorizedFile(file.contentPath, file.filename)
    } catch {
      /* noop */
    }
  }

  return (
    <div className="flex flex-col gap-5 p-5" aria-label="Submission files">
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          Submission files
        </h3>
        {hasFiles ? (
          <ul className="mt-3 space-y-1" role="list">
            {files.map((file) => {
              const label = displayLabel(file.filename)
              const { extension } = splitFilename(label)
              const selected = file.fileId === activeFileId
              return (
                <li key={file.fileId}>
                  <div
                    className={`flex items-center gap-2 rounded-lg border px-3 py-2 ${
                      selected
                        ? 'border-indigo-300 bg-indigo-50/80 dark:border-indigo-800 dark:bg-indigo-950/40'
                        : 'border-slate-200 bg-white dark:border-neutral-600 dark:bg-neutral-950'
                    }`}
                  >
                    <button
                      type="button"
                      onClick={() => onSelectFile(file.fileId)}
                      className={`min-w-0 flex-1 text-left text-sm font-medium leading-snug ${
                        selected
                          ? 'text-indigo-900 dark:text-indigo-100'
                          : 'text-slate-800 hover:text-indigo-700 dark:text-neutral-100 dark:hover:text-indigo-300'
                      }`}
                      aria-current={selected ? 'true' : undefined}
                    >
                      <span className="break-words">{label}</span>
                      {extension ? (
                        <span className="sr-only">{` (${extension})`}</span>
                      ) : null}
                    </button>
                    <button
                      type="button"
                      onClick={() => void handleDownload(file)}
                      className="inline-flex shrink-0 items-center justify-center rounded-lg border border-slate-300 p-2 text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-900"
                      aria-label={`Download ${label}`}
                    >
                      <Download className="h-4 w-4" aria-hidden="true" />
                    </button>
                  </div>
                </li>
              )
            })}
          </ul>
        ) : (
          <p className="mt-3 text-sm text-slate-600 dark:text-neutral-400">No files attached to this submission.</p>
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

      {hasFiles && files.length > 1 && onDownloadAll ? (
        <button
          type="button"
          onClick={() => void onDownloadAll()}
          disabled={downloadAllBusy}
          className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:hover:bg-neutral-900"
        >
          <Download className="h-4 w-4" aria-hidden="true" />
          Download all files
        </button>
      ) : null}
      {hasFiles && files.length === 1 ? (
        <button
          type="button"
          onClick={() => void handleDownload(files[0]!)}
          className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:hover:bg-neutral-900"
        >
          <Download className="h-4 w-4" aria-hidden="true" />
          Download file
        </button>
      ) : null}
    </div>
  )
}
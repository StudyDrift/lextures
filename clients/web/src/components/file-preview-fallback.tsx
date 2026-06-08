import { Download } from 'lucide-react'
import { downloadAuthorizedFile } from '../lib/download-file'
import { splitFilename } from '../lib/file-type'

type FilePreviewFallbackProps = {
  filePath: string
  filename: string
  message: string
  downloadLabel?: string
  /** When `message-only`, shows just the alert (file details live elsewhere, e.g. modal sidebar). */
  variant?: 'standalone' | 'message-only'
}

export function FilePreviewFallback({
  filePath,
  filename,
  message,
  downloadLabel = 'Download',
  variant = 'standalone',
}: FilePreviewFallbackProps) {
  const { name, extension } = splitFilename(filename)

  async function handleDownload() {
    try {
      await downloadAuthorizedFile(filePath, filename)
    } catch {
      /* noop */
    }
  }

  if (variant === 'message-only') {
    return (
      <div className="flex h-full min-h-48 items-center justify-center p-8">
        <p className="max-w-md text-center text-sm text-slate-600 dark:text-neutral-400" role="alert">
          {message}
        </p>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col items-center justify-center gap-4 rounded-lg border border-slate-200 bg-slate-50 p-8 dark:border-neutral-700 dark:bg-neutral-900/60">
      <p className="text-center text-sm text-slate-600 dark:text-neutral-400" role="alert">
        {message}
      </p>
      <dl className="space-y-1 text-center text-sm">
        <div>
          <dt className="sr-only">File name</dt>
          <dd className="font-medium text-slate-900 dark:text-neutral-100">{name}</dd>
        </div>
        <div>
          <dt className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-500">
            Extension
          </dt>
          <dd className="font-mono text-slate-700 dark:text-neutral-300">
            {extension ? `.${extension}` : 'None'}
          </dd>
        </div>
      </dl>
      <button
        type="button"
        onClick={() => void handleDownload()}
        className="flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
      >
        <Download className="h-4 w-4" aria-hidden="true" />
        {downloadLabel}
      </button>
    </div>
  )
}

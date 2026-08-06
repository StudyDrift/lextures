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
        <p className="max-w-md text-center text-sm text-fg-muted" role="alert">
          {message}
        </p>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col items-center justify-center gap-4 rounded-lg border border-border-default bg-surface-base p-8 dark:border-border-default/60">
      <p className="text-center text-sm text-fg-muted" role="alert">
        {message}
      </p>
      <dl className="space-y-1 text-center text-sm">
        <div>
          <dt className="sr-only">File name</dt>
          <dd className="font-medium text-fg-default">{name}</dd>
        </div>
        <div>
          <dt className="text-xs font-semibold uppercase tracking-wide text-fg-subtle">
            Extension
          </dt>
          <dd className="font-mono text-fg-muted">
            {extension ? `.${extension}` : 'None'}
          </dd>
        </div>
      </dl>
      <button
        type="button"
        onClick={() => void handleDownload()}
        className="flex items-center gap-2 rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
      >
        <Download className="h-4 w-4" aria-hidden="true" />
        {downloadLabel}
      </button>
    </div>
  )
}

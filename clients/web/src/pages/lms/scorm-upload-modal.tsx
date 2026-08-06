import { useId, useState } from 'react'
import { scormI18n } from '../../lib/scorm-i18n'

type ScormUploadModalProps = {
  open: boolean
  onClose: () => void
  onSave: (title: string, file: File) => Promise<void>
  saving?: boolean
  errorMessage?: string | null
}

export function ScormUploadModal({
  open,
  onClose,
  onSave,
  saving,
  errorMessage,
}: ScormUploadModalProps) {
  const titleId = useId()
  const fileId = useId()
  const [title, setTitle] = useState('')
  const [file, setFile] = useState<File | null>(null)

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && !saving) onClose()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-md rounded-xl border border-border-default bg-surface-raised p-5 shadow-xl dark:border-border-default dark:bg-surface-overlay"
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-fg-default">
          {scormI18n.uploadLabel}
        </h2>
        <p className="mt-1 text-sm text-fg-muted">{scormI18n.uploadHint}</p>
        <form
          className="mt-4 space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            if (!file || saving) return
            void onSave(title.trim() || file.name.replace(/\.zip$/i, ''), file)
          }}
        >
          <label className="block text-sm font-medium text-fg-muted">
            Title
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="mt-1 w-full rounded-lg border border-border-default px-3 py-2 text-sm dark:border-border-default dark:bg-surface-raised"
              disabled={saving}
            />
          </label>
          <label className="block text-sm font-medium text-fg-muted">
            SCORM package (.zip)
            <input
              id={fileId}
              type="file"
              accept=".zip,application/zip"
              className="mt-1 block w-full text-sm"
              disabled={saving}
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
          </label>
          {errorMessage ? (
            <p className="text-sm text-danger-fg" role="alert">
              {errorMessage}
            </p>
          ) : null}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              className="rounded-lg px-3 py-2 text-sm font-medium text-fg-muted hover:bg-surface-sunken dark:text-fg-muted dark:hover:bg-neutral-700"
              disabled={saving}
              onClick={onClose}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving || !file}
              className="rounded-lg bg-accent-solid px-3 py-2 text-sm font-medium text-white hover:bg-accent disabled:opacity-60"
            >
              {saving ? 'Uploading…' : 'Upload'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

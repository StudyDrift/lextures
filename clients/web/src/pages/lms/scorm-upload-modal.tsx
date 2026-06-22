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
        className="w-full max-w-md rounded-xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-600 dark:bg-neutral-800"
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-neutral-100">
          {scormI18n.uploadLabel}
        </h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{scormI18n.uploadHint}</p>
        <form
          className="mt-4 space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            if (!file || saving) return
            void onSave(title.trim() || file.name.replace(/\.zip$/i, ''), file)
          }}
        >
          <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
            Title
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
              disabled={saving}
            />
          </label>
          <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
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
            <p className="text-sm text-red-600 dark:text-red-400" role="alert">
              {errorMessage}
            </p>
          ) : null}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              className="rounded-lg px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-700"
              disabled={saving}
              onClick={onClose}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving || !file}
              className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-60"
            >
              {saving ? 'Uploading…' : 'Upload'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

import { Image as ImageIcon, Upload, X } from 'lucide-react'
import { useCallback, useEffect, useId, useRef, useState } from 'react'

const ACCEPT = 'image/png,image/jpeg,image/jpg,image/gif,image/webp'

export type MarkdownImageUploadModalProps = {
  open: boolean
  onClose: () => void
  onUpload: (files: File[]) => Promise<void>
}

export function MarkdownImageUploadModal({ open, onClose, onUpload }: MarkdownImageUploadModalProps) {
  const titleId = useId()
  const inputId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragOver, setDragOver] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const reset = useCallback(() => {
    setDragOver(false)
    setBusy(false)
    setError(null)
  }, [])

  const handleClose = useCallback(() => {
    if (busy) return
    reset()
    onClose()
  }, [busy, onClose, reset])

  useEffect(() => {
    if (!open) return
    reset()
    const t = window.setTimeout(() => inputRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [open, reset])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !busy) {
        e.preventDefault()
        handleClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, busy, handleClose])

  const uploadFiles = useCallback(
    async (files: File[]) => {
      const images = files.filter((f) => f.type.startsWith('image/'))
      if (!images.length) {
        setError('Choose a PNG, JPEG, GIF, or WebP image.')
        return
      }
      setError(null)
      setBusy(true)
      try {
        await onUpload(images)
        reset()
        onClose()
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Could not upload image.')
        setBusy(false)
      }
    },
    [onClose, onUpload, reset],
  )

  const onInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files ? [...e.target.files] : []
      e.target.value = ''
      if (files.length) void uploadFiles(files)
    },
    [uploadFiles],
  )

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[95] flex items-end justify-center bg-black/40 p-4 sm:items-center"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) handleClose()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-md rounded-xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-600 dark:bg-neutral-900"
      >
        <div className="mb-4 flex items-start justify-between gap-3">
          <div>
            <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-neutral-100">
              Insert image
            </h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Upload an image to embed it in your notebook.
            </p>
          </div>
          <button
            type="button"
            onClick={handleClose}
            disabled={busy}
            className="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 disabled:opacity-50 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <input
          ref={inputRef}
          id={inputId}
          type="file"
          accept={ACCEPT}
          multiple
          className="sr-only"
          disabled={busy}
          onChange={onInputChange}
        />

        <label
          htmlFor={inputId}
          onDragEnter={(e) => {
            e.preventDefault()
            if (!busy) setDragOver(true)
          }}
          onDragOver={(e) => {
            e.preventDefault()
            if (!busy) {
              e.dataTransfer.dropEffect = 'copy'
              setDragOver(true)
            }
          }}
          onDragLeave={(e) => {
            e.preventDefault()
            if (e.currentTarget.contains(e.relatedTarget as Node | null)) return
            setDragOver(false)
          }}
          onDrop={(e) => {
            e.preventDefault()
            setDragOver(false)
            if (busy) return
            void uploadFiles([...e.dataTransfer.files])
          }}
          className={`flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-6 py-10 text-center transition-[background-color,color,border-color] ${
            dragOver
              ? 'border-indigo-400 bg-indigo-50/80 dark:border-indigo-500 dark:bg-indigo-950/30'
              : 'border-slate-200 bg-slate-50/70 hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-950/50 dark:hover:border-neutral-600 dark:hover:bg-neutral-950'
          } ${busy ? 'pointer-events-none opacity-60' : ''}`}
        >
          <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-white shadow-sm dark:bg-neutral-800">
            {busy ? (
              <Upload className="h-5 w-5 animate-pulse text-indigo-500" aria-hidden />
            ) : (
              <ImageIcon className="h-5 w-5 text-indigo-500" aria-hidden />
            )}
          </div>
          <span className="text-sm font-medium text-slate-800 dark:text-neutral-100">
            {busy ? 'Uploading…' : 'Drop an image here'}
          </span>
          <span className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            or click to browse (PNG, JPEG, GIF, WebP)
          </span>
        </label>

        {error ? (
          <p className="mt-3 text-sm text-rose-600 dark:text-rose-400" role="alert">
            {error}
          </p>
        ) : null}
      </div>
    </div>
  )
}

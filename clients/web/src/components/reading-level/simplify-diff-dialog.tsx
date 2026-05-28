export type SimplifyDiffDialogProps = {
  open: boolean
  original: string
  simplified: string
  targetFkgl: number
  computedFkgl?: number
  loading?: boolean
  error?: string | null
  onClose: () => void
  onAccept: () => void
}

export function SimplifyDiffDialog({
  open,
  original,
  simplified,
  targetFkgl,
  computedFkgl,
  loading,
  error,
  onClose,
  onAccept,
}: SimplifyDiffDialogProps) {
  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="simplify-diff-title"
    >
      <div className="max-h-[90vh] w-full max-w-3xl overflow-auto rounded-xl bg-white p-6 shadow-xl dark:bg-neutral-900">
        <h2 id="simplify-diff-title" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
          Simplify to Grade {targetFkgl}
        </h2>
        {computedFkgl != null && (
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Computed reading level: Grade {computedFkgl.toFixed(1)}
          </p>
        )}
        {error && (
          <p className="mt-2 text-sm text-rose-700 dark:text-rose-300" role="alert">
            {error}
          </p>
        )}
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <div>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Original</h3>
            <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap rounded border border-slate-200 bg-slate-50 p-3 text-sm dark:border-neutral-700 dark:bg-neutral-800">
              {original}
            </pre>
          </div>
          <div>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Simplified</h3>
            <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap rounded border border-slate-200 bg-slate-50 p-3 text-sm dark:border-neutral-700 dark:bg-neutral-800">
              {loading ? 'Generating…' : simplified}
            </pre>
          </div>
        </div>
        <div className="mt-6 flex justify-end gap-2">
          <button
            type="button"
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm dark:border-neutral-600"
            onClick={onClose}
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={loading || !simplified.trim()}
            className="rounded-lg bg-slate-900 px-4 py-2 text-sm text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
            onClick={onAccept}
          >
            Accept simplified text
          </button>
        </div>
      </div>
    </div>
  )
}

import { useEffect, useRef, useState } from 'react'
import { X, Sparkles, Play, Save } from 'lucide-react'

type VibeActivityCreateModalProps = {
  open: boolean
  onClose: () => void
  onSave: (title: string, html: string) => void | Promise<void>
  saving?: boolean
  error?: string | null
  initialTitle?: string
  initialHtml?: string
}

export function VibeActivityCreateModal({
  open,
  onClose,
  onSave,
  saving,
  error,
  initialTitle = '',
  initialHtml = '',
}: VibeActivityCreateModalProps) {
  const [title, setTitle] = useState(initialTitle)
  const [html, setHtml] = useState(initialHtml)
  const [showPreview, setShowPreview] = useState(true)
  const dialogRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (open) {
      setTitle(initialTitle)
      setHtml(initialHtml || defaultTemplate())
    }
  }, [open, initialTitle, initialHtml])

  // Close on escape
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const canSave = title.trim().length > 0 && html.trim().length > 0

  async function handleSave() {
    if (!canSave || saving) return
    await onSave(title.trim(), html)
  }

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/50 p-4" role="dialog" aria-modal>
      <div
        ref={dialogRef}
        className="flex h-[min(92vh,820px)] w-full max-w-6xl flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl dark:border-neutral-700 dark:bg-neutral-900"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <div className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-rose-600" />
            <div>
              <div className="font-semibold text-slate-950 dark:text-neutral-100">Create Vibe Activity</div>
              <div className="text-xs text-slate-500 dark:text-neutral-400">Self-contained interactive HTML for students</div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="rounded p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Body */}
        <div className="flex min-h-0 flex-1 flex-col gap-3 p-4 md:flex-row">
          {/* Left: prompt + source editor */}
          <div className="flex w-full flex-col gap-3 md:w-5/12">
            <div>
              <label className="text-xs font-medium text-slate-600 dark:text-neutral-400">Activity title</label>
              <input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Interactive Cell Explorer"
                className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
              />
            </div>

            <div className="flex-1">
              <div className="mb-1 flex items-center justify-between">
                <label className="text-xs font-medium text-slate-600 dark:text-neutral-400">HTML source (self-contained)</label>
                <button
                  type="button"
                  onClick={() => setHtml(defaultTemplate())}
                  className="text-xs text-rose-600 hover:underline"
                >
                  Reset template
                </button>
              </div>
              <textarea
                value={html}
                onChange={(e) => setHtml(e.target.value)}
                spellCheck={false}
                className="h-full min-h-[280px] w-full resize-y rounded-lg border border-slate-300 bg-slate-950 p-3 font-mono text-xs text-slate-100 dark:border-neutral-700"
                placeholder="<!doctype html>..."
              />
            </div>

            <div className="rounded-lg bg-amber-50 p-2 text-[11px] text-amber-800 dark:bg-amber-950/40 dark:text-amber-200">
              Tip: Use Tailwind via CDN (<code>https://cdn.tailwindcss.com</code>) for rapid styling. Keep everything in one file.
            </div>
          </div>

          {/* Right: Preview */}
          <div className="flex w-full flex-col md:w-7/12">
            <div className="mb-1 flex items-center justify-between">
              <div className="flex items-center gap-2 text-xs font-medium text-slate-600 dark:text-neutral-400">
                <Play className="h-3.5 w-3.5" /> Live preview
              </div>
              <button
                type="button"
                onClick={() => setShowPreview((v) => !v)}
                className="text-xs text-slate-500 hover:text-slate-700 dark:hover:text-neutral-300"
              >
                {showPreview ? 'Hide' : 'Show'} preview
              </button>
            </div>

            {showPreview ? (
              <div className="flex-1 overflow-hidden rounded-xl border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-950">
                <iframe
                  title="vibe-preview"
                  sandbox="allow-scripts allow-forms allow-same-origin"
                  srcDoc={html || '<!doctype html><html><body style="padding:2rem;font-family:sans-serif;color:#888">Start typing HTML on the left…</body></html>'}
                  className="block h-full w-full"
                />
              </div>
            ) : (
              <div className="flex flex-1 items-center justify-center rounded-xl border border-dashed border-slate-300 text-xs text-slate-400 dark:border-neutral-700">
                Preview hidden
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-slate-200 bg-slate-50 px-4 py-3 dark:border-neutral-700 dark:bg-neutral-800/60">
          <div className="text-xs text-slate-500 dark:text-neutral-400">
            AI generation coming soon (requires platform OpenRouter key). Paste or write HTML directly for now.
          </div>

          <div className="flex items-center gap-2">
            {error && <span className="text-xs text-red-600">{error}</span>}
            <button
              type="button"
              onClick={onClose}
              disabled={saving}
              className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm hover:bg-white dark:border-neutral-600 dark:hover:bg-neutral-700"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSave}
              disabled={!canSave || saving}
              className="inline-flex items-center gap-2 rounded-lg bg-rose-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-rose-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              <Save className="h-4 w-4" />
              {saving ? 'Saving…' : 'Save & Add to Module'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function defaultTemplate(): string {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <script src="https://cdn.tailwindcss.com"></script>
  <title>Vibe Activity</title>
  <style>body { font-family: system-ui, sans-serif; }</style>
</head>
<body class="bg-slate-50 p-8">
  <div class="max-w-2xl mx-auto">
    <div class="rounded-2xl bg-white shadow p-6">
      <h1 class="text-2xl font-semibold text-slate-900">Hello, students!</h1>
      <p class="mt-2 text-slate-600">Replace this with your interactive experience. Add buttons, canvases, or Tailwind components.</p>

      <button onclick="celebrate()" class="mt-4 rounded-xl bg-indigo-600 px-4 py-2 text-white">Try me</button>
      <div id="result" class="mt-3 text-sm text-emerald-600"></div>
    </div>
  </div>

  <script>
    function celebrate() {
      const el = document.getElementById('result');
      if (el) el.textContent = 'Great job! You interacted with the activity.';
      // Add your own JS here – this is a fully self-contained page.
    }
  </script>
</body>
</html>`
}

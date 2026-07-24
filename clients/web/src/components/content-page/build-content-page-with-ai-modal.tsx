import { useEffect, useId, useState } from 'react'
import { Loader2, Sparkles, X } from 'lucide-react'
import type { DraftContentPageSection } from '../../lib/courses-api'

type BuildContentPageWithAiModalProps = {
  open: boolean
  existingMarkdown: string
  /** Optional copy tweaks for quiz intro vs content page. */
  description?: string
  placeholder?: string
  onClose: () => void
  onBuild: (args: {
    prompt: string
    existingMarkdown: string
  }) => Promise<DraftContentPageSection[]>
  onBuilt: (sections: DraftContentPageSection[]) => void
}

/**
 * Prompt modal: describe the page topic; AI returns draft sections for the editor
 * (not persisted until the user saves).
 */
export function BuildContentPageWithAiModal({
  open,
  existingMarkdown,
  description = 'Describe what this page should cover. The draft replaces the current editor content; nothing is saved until you click Save.',
  placeholder = 'e.g. An introduction to photosynthesis for high school biology, with key vocabulary and a short practice check…',
  onClose,
  onBuild,
  onBuilt,
}: BuildContentPageWithAiModalProps) {
  const titleId = useId()
  const promptId = useId()
  const [prompt, setPrompt] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setPrompt('')
    setError(null)
    setBusy(false)
  }, [open])

  if (!open) return null

  async function submit() {
    const text = prompt.trim()
    if (!text || busy) return
    setBusy(true)
    setError(null)
    try {
      const sections = await onBuild({
        prompt: text,
        existingMarkdown: existingMarkdown.trim(),
      })
      if (sections.length === 0) {
        setError('No content sections were generated. Try a more specific description.')
        return
      }
      onBuilt(sections)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not generate content.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      onClick={(e) => {
        if (e.target === e.currentTarget && !busy) onClose()
      }}
    >
      <div className="w-full max-w-lg overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <h3
            id={titleId}
            className="inline-flex items-center gap-1.5 text-sm font-semibold text-slate-900 dark:text-neutral-100"
          >
            <Sparkles className="h-4 w-4 text-indigo-500" aria-hidden />
            Build with AI
          </h3>
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="space-y-4 p-4">
          <p className="text-sm text-slate-600 dark:text-neutral-300">{description}</p>
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-600 dark:text-neutral-400" htmlFor={promptId}>
              Topic description
            </label>
            <textarea
              id={promptId}
              rows={5}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              disabled={busy}
              autoFocus
              placeholder={placeholder}
              className="w-full resize-y rounded-xl border border-slate-200 px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-2 focus:ring-indigo-400/30 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault()
                  void submit()
                }
              }}
            />
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              ⌘/Ctrl + Enter to generate
            </p>
          </div>
          {error ? (
            <p className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
              {error}
            </p>
          ) : null}
        </div>
        <div className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/80 px-4 py-3 dark:border-neutral-700 dark:bg-neutral-950/50">
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="rounded-xl px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void submit()}
            disabled={busy || prompt.trim() === ''}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-60"
          >
            {busy ? <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden /> : null}
            {busy ? 'Generating…' : 'Generate'}
          </button>
        </div>
      </div>
    </div>
  )
}

import { useEffect, useId, useState } from 'react'
import { Loader2, Sparkles, X } from 'lucide-react'
import type { DraftContentPageSection } from '../../lib/courses-api'

type BuildContentPageWithAiModalProps = {
  open: boolean
  existingMarkdown: string
  /** Optional copy tweaks for quiz intro vs content page. */
  description?: string
  placeholder?: string
  /**
   * When true, show the “Include interactive tools” toggle (Content Tools enabled for the course).
   * Defaults to false (prose-only path).
   */
  contentToolsAvailable?: boolean
  /** Initial value for the tools toggle when the modal opens. */
  defaultIncludeTools?: boolean
  onClose: () => void
  onBuild: (args: {
    prompt: string
    existingMarkdown: string
    includeTools: boolean
  }) => Promise<DraftContentPageSection[]>
  onBuilt: (sections: DraftContentPageSection[]) => void | Promise<void>
}

/**
 * Prompt modal: describe the page topic; AI returns draft sections for the editor
 * (not persisted until the user saves). Optionally includes interactive content tools.
 */
export function BuildContentPageWithAiModal({
  open,
  existingMarkdown,
  description = 'Describe what this page should cover. The draft replaces the current editor content; nothing is saved until you click Save.',
  placeholder = 'e.g. An introduction to photosynthesis for high school biology, with key vocabulary and a short practice check…',
  contentToolsAvailable = false,
  defaultIncludeTools = true,
  onClose,
  onBuild,
  onBuilt,
}: BuildContentPageWithAiModalProps) {
  const titleId = useId()
  const promptId = useId()
  const toolsToggleId = useId()
  const [prompt, setPrompt] = useState('')
  const [includeTools, setIncludeTools] = useState(defaultIncludeTools)
  const [busy, setBusy] = useState(false)
  const [phase, setPhase] = useState<'prompt' | 'placing'>('prompt')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setPrompt('')
    setIncludeTools(contentToolsAvailable ? defaultIncludeTools : false)
    setError(null)
    setBusy(false)
    setPhase('prompt')
  }, [open, contentToolsAvailable, defaultIncludeTools])

  if (!open) return null

  async function submit() {
    const text = prompt.trim()
    if (!text || busy) return
    setBusy(true)
    setError(null)
    setPhase('prompt')
    try {
      const wantTools = Boolean(contentToolsAvailable && includeTools)
      const sections = await onBuild({
        prompt: text,
        existingMarkdown: existingMarkdown.trim(),
        includeTools: wantTools,
      })
      if (sections.length === 0) {
        setError('No content sections were generated. Try a more specific description.')
        return
      }
      const hasTools = sections.some((s) => (s.tools?.length ?? 0) > 0)
      if (hasTools) setPhase('placing')
      await onBuilt(sections)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not generate content.')
      setPhase('prompt')
    } finally {
      setBusy(false)
      setPhase('prompt')
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
      <div className="w-full max-w-lg overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-xl dark:border-border-default dark:bg-surface-raised">
        <div className="flex items-center justify-between border-b border-border-default px-4 py-3 dark:border-border-default">
          <h3
            id={titleId}
            className="inline-flex items-center gap-1.5 text-sm font-semibold text-fg-default"
          >
            <Sparkles className="h-4 w-4 text-indigo-500" aria-hidden />
            Build with AI
          </h3>
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="rounded-lg p-1.5 text-fg-muted hover:bg-surface-sunken hover:text-fg-default disabled:opacity-50 dark:text-fg-muted dark:hover:bg-surface-overlay dark:hover:text-fg-default"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="space-y-4 p-4">
          <p className="text-sm text-fg-muted">{description}</p>
          <div>
            <label className="mb-1 block text-xs font-medium text-fg-muted" htmlFor={promptId}>
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
              className="w-full resize-y rounded-xl border border-border-default px-3 py-2 text-sm text-fg-default placeholder:text-fg-subtle focus:border-indigo-400 focus:outline-none focus:ring-2 focus:ring-indigo-400/30 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:placeholder:text-neutral-500"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault()
                  void submit()
                }
              }}
            />
            <p className="mt-1 text-xs text-fg-muted">
              ⌘/Ctrl + Enter to generate
            </p>
          </div>
          {contentToolsAvailable ? (
            <div className="rounded-xl border border-border-default bg-slate-50/80 px-3 py-3 dark:border-border-default/40">
              <label className="flex cursor-pointer items-start gap-3" htmlFor={toolsToggleId}>
                <input
                  id={toolsToggleId}
                  type="checkbox"
                  checked={includeTools}
                  disabled={busy}
                  onChange={(e) => setIncludeTools(e.target.checked)}
                  className="mt-0.5 h-4 w-4 rounded border-border-strong text-accent-fg focus:ring-indigo-500 disabled:opacity-60"
                />
                <span className="min-w-0">
                  <span className="block text-sm font-medium text-fg-default">
                    Include interactive tools
                  </span>
                  <span className="mt-0.5 block text-xs leading-relaxed text-fg-muted">
                    Let AI add checks for understanding, flashcards, polls, and other content tools
                    alongside the prose. You can edit or remove them before saving.
                  </span>
                </span>
              </label>
            </div>
          ) : null}
          {error ? (
            <p className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
              {error}
            </p>
          ) : null}
        </div>
        <div className="flex items-center justify-end gap-2 border-t border-border-default bg-slate-50/80 px-4 py-3 dark:border-border-default/50">
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="rounded-xl px-3 py-2 text-sm font-medium text-fg-muted hover:bg-surface-sunken disabled:opacity-50 dark:text-fg-muted dark:hover:bg-surface-overlay"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void submit()}
            disabled={busy || prompt.trim() === ''}
            className="inline-flex items-center gap-2 rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-60"
          >
            {busy ? <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden /> : null}
            {busy
              ? phase === 'placing'
                ? 'Placing tools…'
                : 'Generating…'
              : 'Generate'}
          </button>
        </div>
      </div>
    </div>
  )
}

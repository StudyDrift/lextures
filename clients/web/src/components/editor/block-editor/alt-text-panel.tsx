import { Loader2, Sparkles } from 'lucide-react'
import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { announce } from '../../../lib/a11y/announcer'
import { suggestAltText } from '../../../lib/alt-text-api'
import { useAltTextEnforcement } from './alt-text-enforcement-context'

export type AltTextPanelProps = {
  alt: string
  decorative: boolean
  imageSrc: string
  imageUrlForAi?: string | null
  onApply: (next: { alt: string; decorative: boolean }) => void
  onClose: () => void
  /** When true, focus the input and announce on mount (new image insertion). */
  autoFocus?: boolean
}

export function AltTextPanel({
  alt,
  decorative,
  imageSrc,
  imageUrlForAi,
  onApply,
  onClose,
  autoFocus = false,
}: AltTextPanelProps) {
  const { courseCode, onAiUnavailable } = useAltTextEnforcement()
  const inputId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const [draftAlt, setDraftAlt] = useState(alt)
  const [draftDecorative, setDraftDecorative] = useState(decorative)
  const [aiLoading, setAiLoading] = useState(false)
  const [aiSuggestion, setAiSuggestion] = useState<string | null>(null)

  useEffect(() => {
    setDraftAlt(alt)
    setDraftDecorative(decorative)
  }, [alt, decorative])

  useEffect(() => {
    if (!autoFocus) return
    announce(
      'Alt text required for image. Enter description or mark as decorative.',
      'assertive',
    )
    inputRef.current?.focus()
  }, [autoFocus])

  const requestSuggestion = useCallback(async () => {
    if (!courseCode) return
    const url = (imageUrlForAi ?? imageSrc).trim()
    if (!url) return
    setAiLoading(true)
    setAiSuggestion(null)
    try {
      const res = await suggestAltText(courseCode, url)
      if (res.suggestion) {
        setAiSuggestion(res.suggestion)
        if (!draftDecorative) {
          setDraftAlt(res.suggestion)
        }
      }
    } catch {
      onAiUnavailable?.()
    } finally {
      setAiLoading(false)
    }
  }, [courseCode, draftDecorative, imageSrc, imageUrlForAi, onAiUnavailable])

  useEffect(() => {
    if (!autoFocus || draftDecorative || !courseCode) return
    const timer = window.setTimeout(() => {
      void requestSuggestion()
    }, 300)
    return () => window.clearTimeout(timer)
  }, [autoFocus, courseCode, draftDecorative, requestSuggestion])

  const canSave = draftDecorative || draftAlt.trim().length > 0

  return (
    <div
      role="dialog"
      aria-label="Image alternative text"
      className="absolute start-0 top-full z-20 mt-2 w-[min(20rem,calc(100vw-2rem))] rounded-xl border border-slate-200 bg-white p-3 shadow-lg shadow-slate-900/15 dark:border-neutral-600 dark:bg-neutral-900"
      onMouseDown={(e) => e.stopPropagation()}
    >
      <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">
        Add alt text (required for accessibility)
      </p>
      <div className="space-y-2">
        <div>
          <label htmlFor={inputId} className="mb-1 block text-xs text-slate-600 dark:text-neutral-300">
            Alternative text for image
          </label>
          <div className="relative">
            <input
              ref={inputRef}
              id={inputId}
              type="text"
              value={draftDecorative ? '' : draftAlt}
              disabled={draftDecorative || aiLoading}
              placeholder={aiLoading ? 'Generating suggestion…' : 'Describe this image'}
              onChange={(e) => setDraftAlt(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && canSave) {
                  e.preventDefault()
                  onApply({ alt: draftDecorative ? '' : draftAlt.trim(), decorative: draftDecorative })
                }
                if (e.key === 'Escape') {
                  e.preventDefault()
                  onClose()
                }
              }}
              className="w-full rounded-md border border-slate-200 bg-white px-2.5 py-1.5 pe-8 text-sm text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
            />
            {aiLoading ? (
              <Loader2
                className="pointer-events-none absolute end-2 top-1/2 h-4 w-4 -translate-y-1/2 animate-spin text-indigo-500"
                aria-hidden
              />
            ) : null}
          </div>
          {aiSuggestion && !draftDecorative ? (
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              AI suggestion applied — edit before saving if needed.
            </p>
          ) : null}
        </div>
        <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
          <input
            type="checkbox"
            checked={draftDecorative}
            onChange={(e) => {
              const checked = e.target.checked
              setDraftDecorative(checked)
              if (checked) setDraftAlt('')
            }}
            className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
          />
          Mark as decorative (screen readers skip)
        </label>
        <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
          <button
            type="button"
            aria-label="Generate alt text with AI"
            disabled={aiLoading || draftDecorative || !courseCode}
            onClick={() => void requestSuggestion()}
            className="inline-flex items-center gap-1 rounded-md px-2 py-1.5 text-xs font-medium text-indigo-700 hover:bg-indigo-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-indigo-300 dark:hover:bg-indigo-950/40"
          >
            <Sparkles className="h-3.5 w-3.5" aria-hidden />
            Suggest with AI
          </button>
          <div className="flex items-center gap-1.5">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md px-2.5 py-1.5 text-xs text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={!canSave}
              onClick={() =>
                onApply({ alt: draftDecorative ? '' : draftAlt.trim(), decorative: draftDecorative })
              }
              className="rounded-md bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-indigo-500"
            >
              Save alt text
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

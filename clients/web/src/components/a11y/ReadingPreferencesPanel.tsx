import { useEffect, useId, useRef, useState } from 'react'
import { X } from 'lucide-react'
import { useReadingPreferences } from '../../context/reading-preferences-context'
import { LiveRegion } from './live-region'
import type {
  FontFace,
  LetterSpacing,
  LineHeight,
  RulerColor,
  WordSpacing,
} from '../../lib/reading-preferences'

interface Props {
  open: boolean
  onClose: () => void
}

const fontOptions: { value: FontFace; label: string; description: string }[] = [
  { value: 'default',       label: 'Default',               description: 'Plus Jakarta Sans' },
  { value: 'open-dyslexic', label: 'OpenDyslexic',          description: 'Optimised for dyslexic readers' },
  { value: 'atkinson',      label: 'Atkinson Hyperlegible',  description: 'High legibility sans-serif' },
  { value: 'system',        label: 'System font',            description: "Your device's default font" },
]

const spacingSteps: { value: LetterSpacing; label: string }[] = [
  { value: 'normal', label: 'Normal' },
  { value: 'wide',   label: 'Wide' },
  { value: 'wider',  label: 'Wider' },
]

const wordSpacingSteps: { value: WordSpacing; label: string }[] = [
  { value: 'normal', label: 'Normal' },
  { value: 'wide',   label: 'Wide' },
  { value: 'wider',  label: 'Wider' },
]

const lineHeightSteps: { value: LineHeight; label: string }[] = [
  { value: 'normal', label: 'Normal (1.5×)' },
  { value: 'tall',   label: 'Tall (1.8×)' },
  { value: 'taller', label: 'Taller (2.0×)' },
]

const rulerColorOptions: { value: RulerColor; label: string; bg: string }[] = [
  { value: 'yellow', label: 'Yellow tint', bg: 'rgba(255, 248, 0, 0.25)' },
  { value: 'grey',   label: 'Grey tint',   bg: 'rgba(128, 128, 128, 0.2)' },
]

export function ReadingPreferencesPanel({ open, onClose }: Props) {
  const { prefs, loading, update } = useReadingPreferences()
  const panelRef = useRef<HTMLDivElement>(null)
  const closeBtnRef = useRef<HTMLButtonElement>(null)
  const titleId = useId()
  const [liveAnnouncement, setLiveAnnouncement] = useState('')

  /* Trap focus + close on Escape */
  useEffect(() => {
    if (!open) return
    closeBtnRef.current?.focus()
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onClose()
        return
      }
      if (e.key !== 'Tab') return
      const panel = panelRef.current
      if (!panel) return
      const focusable = panel.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
      )
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault()
          last?.focus()
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault()
          first?.focus()
        }
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <>
      {/* Backdrop (click-outside closes) */}
      <div
        className="fixed inset-0 z-40"
        aria-hidden="true"
        onClick={onClose}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-label="Reading Preferences"
        className="fixed end-4 top-16 z-50 w-80 max-h-[calc(100dvh-5rem)] overflow-y-auto rounded-2xl border border-slate-200 bg-white shadow-xl shadow-slate-900/10 dark:border-neutral-700 dark:bg-neutral-900 dark:shadow-black/40 sm:w-96"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3 dark:border-neutral-800">
          <h2 id={titleId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            Reading Preferences
          </h2>
          <button
            ref={closeBtnRef}
            type="button"
            aria-label="Close Reading Preferences panel"
            onClick={onClose}
            className="rounded-lg p-1 text-slate-500 hover:bg-slate-100 hover:text-slate-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
          >
            <X className="h-4 w-4" aria-hidden />
          </button>
        </div>

        {loading ? (
          <div className="space-y-4 p-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-8 motion-safe:animate-pulse rounded-lg bg-slate-100 dark:bg-neutral-800" />
            ))}
          </div>
        ) : (
          <div className="space-y-5 p-4">
            {/* Font face */}
            <fieldset>
              <legend className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                Font
              </legend>
              <div className="space-y-1.5">
                {fontOptions.map((opt) => (
                  <label
                    key={opt.value}
                    className="flex cursor-pointer items-center gap-3 rounded-lg border border-transparent px-3 py-2 hover:bg-slate-50 has-[:checked]:border-indigo-200 has-[:checked]:bg-indigo-50 dark:hover:bg-neutral-800 dark:has-[:checked]:border-indigo-800 dark:has-[:checked]:bg-indigo-950/30"
                  >
                    <input
                      type="radio"
                      name="reading-font-face"
                      value={opt.value}
                      checked={prefs.fontFace === opt.value}
                      onChange={() => update({ fontFace: opt.value })}
                      className="h-4 w-4 border-slate-300 text-indigo-600 focus:ring-indigo-500/30"
                      aria-label={`Font: ${opt.label} — ${opt.description}`}
                    />
                    <span className="min-w-0">
                      <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                        {opt.label}
                      </span>
                      <span className="block text-xs text-slate-500 dark:text-neutral-400">
                        {opt.description}
                      </span>
                    </span>
                  </label>
                ))}
              </div>
            </fieldset>

            {/* Letter spacing */}
            <SpacingControl
              legend="Letter Spacing"
              name="reading-letter-spacing"
              options={spacingSteps}
              value={prefs.letterSpacing}
              onChange={(v) => update({ letterSpacing: v as LetterSpacing })}
            />

            {/* Word spacing */}
            <SpacingControl
              legend="Word Spacing"
              name="reading-word-spacing"
              options={wordSpacingSteps}
              value={prefs.wordSpacing}
              onChange={(v) => update({ wordSpacing: v as WordSpacing })}
            />

            {/* Line height */}
            <SpacingControl
              legend="Line Height"
              name="reading-line-height"
              options={lineHeightSteps}
              value={prefs.lineHeight}
              onChange={(v) => update({ lineHeight: v as LineHeight })}
            />

            {/* Reading ruler */}
            <div>
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                  Reading Ruler
                </span>
                <button
                  type="button"
                  role="switch"
                  aria-checked={prefs.rulerEnabled}
                  aria-label={`Reading ruler: ${prefs.rulerEnabled ? 'on' : 'off'}`}
                  onClick={() => update({ rulerEnabled: !prefs.rulerEnabled })}
                  className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40 ${
                    prefs.rulerEnabled
                      ? 'bg-indigo-600 dark:bg-indigo-500'
                      : 'bg-slate-200 dark:bg-neutral-700'
                  }`}
                >
                  <span
                    className={`inline-block h-3.5 w-3.5 rounded-full bg-white shadow-sm ${
                      prefs.rulerEnabled ? 'translate-x-4' : 'translate-x-0.5'
                    }`}
                  />
                </button>
              </div>
              {prefs.rulerEnabled && (
                <div className="mt-2.5">
                  <p className="mb-1.5 text-xs text-slate-500 dark:text-neutral-400">Ruler colour</p>
                  <div className="flex gap-2">
                    {rulerColorOptions.map((opt) => (
                      <label key={opt.value} className="flex cursor-pointer items-center gap-1.5">
                        <input
                          type="radio"
                          name="reading-ruler-color"
                          value={opt.value}
                          checked={prefs.rulerColor === opt.value}
                          onChange={() => update({ rulerColor: opt.value })}
                          className="sr-only"
                          aria-label={`Ruler colour: ${opt.label}`}
                        />
                        <span
                          aria-hidden="true"
                          style={{ background: opt.bg }}
                          className={`h-5 w-8 rounded border-2 ${
                            prefs.rulerColor === opt.value
                              ? 'border-indigo-500'
                              : 'border-slate-200 dark:border-neutral-600'
                          }`}
                        />
                        <span className="text-xs text-slate-600 dark:text-neutral-300">{opt.label}</span>
                      </label>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Accessibility display — plan 12.7 */}
            <div className="border-t border-slate-100 pt-4 dark:border-neutral-800">
              <p className="mb-3 text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                Display
              </p>
              <LiveRegion politeness="polite">{liveAnnouncement}</LiveRegion>
              <div className="space-y-3">
                <AccessibilityToggle
                  id="pref-high-contrast"
                  label="High contrast"
                  description="Increases contrast to at least 7:1 for text and interactive elements."
                  checked={prefs.highContrastEnabled}
                  onChange={(v) => {
                    update({ highContrastEnabled: v })
                    setLiveAnnouncement(v ? 'High contrast enabled' : 'High contrast disabled')
                  }}
                />
                <AccessibilityToggle
                  id="pref-reduce-motion"
                  label="Reduce motion"
                  description="Stops animations and transitions to reduce motion-triggered discomfort."
                  checked={prefs.reducedMotionEnabled}
                  onChange={(v) => {
                    update({ reducedMotionEnabled: v })
                    setLiveAnnouncement(v ? 'Reduce motion enabled' : 'Reduce motion disabled')
                  }}
                />
              </div>
            </div>
          </div>
        )}
      </div>
    </>
  )
}

interface AccessibilityToggleProps {
  id: string
  label: string
  description: string
  checked: boolean
  onChange: (value: boolean) => void
}

function AccessibilityToggle({ id, label, description, checked, onChange }: AccessibilityToggleProps) {
  return (
    <div className="flex items-start gap-3">
      <button
        id={id}
        role="switch"
        aria-checked={checked}
        aria-describedby={`${id}-desc`}
        onClick={() => onChange(!checked)}
        style={{ backgroundColor: checked ? 'rgb(79 70 229)' : 'rgb(209 213 219)' }}
        className="mt-0.5 relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40"
      >
        <span className="sr-only">{label}</span>
        <span
          aria-hidden="true"
          className={`inline-block h-3.5 w-3.5 rounded-full bg-white shadow-sm motion-safe:transition-transform ${checked ? 'translate-x-4' : 'translate-x-0.5'}`}
        />
      </button>
      <div className="min-w-0 flex-1">
        <label htmlFor={id} className="cursor-pointer select-none text-sm font-medium text-slate-900 dark:text-neutral-100">
          {label}
        </label>
        <p id={`${id}-desc`} className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
          {description}
        </p>
      </div>
    </div>
  )
}

interface SpacingControlProps {
  legend: string
  name: string
  options: { value: string; label: string }[]
  value: string
  onChange: (v: string) => void
}

function SpacingControl({ legend, name, options, value, onChange }: SpacingControlProps) {
  return (
    <fieldset>
      <legend className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        {legend}
      </legend>
      <div className="flex gap-1.5">
        {options.map((opt) => (
          <label
            key={opt.value}
            className={`flex flex-1 cursor-pointer items-center justify-center rounded-lg border px-2 py-1.5 text-xs font-medium ${
              value === opt.value
                ? 'border-indigo-300 bg-indigo-50 text-indigo-700 dark:border-indigo-700 dark:bg-indigo-950/40 dark:text-indigo-300'
                : 'border-slate-200 bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:border-neutral-600 dark:hover:bg-neutral-700'
            }`}
          >
            <input
              type="radio"
              name={name}
              value={opt.value}
              checked={value === opt.value}
              onChange={() => onChange(opt.value)}
              className="sr-only"
              aria-label={`${legend}: ${opt.label}`}
            />
            {opt.label}
          </label>
        ))}
      </div>
    </fieldset>
  )
}

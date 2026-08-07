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
import { TEXT_SCALE_OPTIONS, normalizeTextScale } from '../../lib/reading-preferences'
import { createFocusTrap } from '../../lib/a11y/focus-trap'
import { useInertBackground } from '../ui/use-inert-background'

interface Props {
  open: boolean
  onClose: () => void
}

const fontOptions: { value: FontFace; label: string; description: string }[] = [
  { value: 'default',       label: 'Default',               description: 'Lextures' },
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

  useInertBackground(open)

  /* Trap focus + close on Escape (UX.4 FR-4 via shared focus-trap). */
  useEffect(() => {
    if (!open || !panelRef.current) return
    const trap = createFocusTrap(panelRef.current, {
      initialFocus: closeBtnRef.current,
    })
    trap.activate()
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('keydown', onKey)
      trap.deactivate()
    }
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
        className="fixed end-4 top-16 z-50 w-80 max-h-[calc(100dvh-5rem)] overflow-y-auto rounded-2xl border border-border-default bg-surface-raised shadow-xl shadow-slate-900/10 dark:border-border-default dark:bg-surface-raised dark:shadow-black/40 sm:w-96"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border-subtle px-4 py-3 dark:border-border-subtle">
          <h2 id={titleId} className="text-body-sm font-semibold text-fg-default">
            Reading Preferences
          </h2>
          <button
            ref={closeBtnRef}
            type="button"
            aria-label="Close Reading Preferences panel"
            onClick={onClose}
            className="rounded-lg p-1 text-fg-muted hover:bg-surface-sunken hover:text-fg-muted focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40 dark:text-fg-muted dark:hover:bg-surface-overlay dark:hover:text-fg-default"
          >
            <X className="h-4 w-4" aria-hidden />
          </button>
        </div>

        {loading ? (
          <div className="space-y-4 p-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-8 motion-safe:animate-pulse rounded-lg bg-surface-sunken" />
            ))}
          </div>
        ) : (
          <div className="space-y-5 p-4">
            <p className="text-caption text-fg-default">
              These settings apply across modules, assignments, discussions, and the notebook.
            </p>

            {/* Text scale — UX.3 */}
            <fieldset>
              <legend className="mb-2 text-overline text-fg-default">
                Text size
              </legend>
              <div className="flex gap-1.5">
                {TEXT_SCALE_OPTIONS.map((opt) => {
                  const current = normalizeTextScale(prefs.textScale)
                  return (
                    <label
                      key={opt.value}
                      className={`flex flex-1 cursor-pointer items-center justify-center rounded-lg border px-2 py-1.5 text-caption font-medium ${
                        current === opt.value
                          ? 'border-accent-border bg-accent-surface text-accent-fg'
                          : 'border-border-default bg-surface-raised text-fg-muted hover:border-border-strong hover:bg-surface-base'
                      }`}
                    >
                      <input
                        type="radio"
                        name="reading-text-scale"
                        value={opt.value}
                        checked={current === opt.value}
                        onChange={() => {
                          update({ textScale: opt.value })
                          setLiveAnnouncement(`Text size ${opt.label}`)
                        }}
                        className="sr-only"
                        aria-label={`Text size: ${opt.label}`}
                      />
                      {opt.label}
                    </label>
                  )
                })}
              </div>
              <p className="mt-1.5 text-caption text-fg-default">
                Preview: <span className="text-body">The quick brown fox jumps.</span>
              </p>
            </fieldset>

            {/* Font face */}
            <fieldset>
              <legend className="mb-2 text-overline text-fg-default">
                Font
              </legend>
              <div className="space-y-1.5">
                {fontOptions.map((opt) => (
                  <label
                    key={opt.value}
                    className="flex cursor-pointer items-center gap-3 rounded-lg border border-transparent px-3 py-2 hover:bg-surface-base has-[:checked]:border-indigo-200 has-[:checked]:bg-indigo-50 dark:hover:bg-surface-overlay dark:has-[:checked]:border-indigo-800 dark:has-[:checked]:bg-indigo-950/30"
                  >
                    <input
                      type="radio"
                      name="reading-font-face"
                      value={opt.value}
                      checked={prefs.fontFace === opt.value}
                      onChange={() => update({ fontFace: opt.value })}
                      className="h-4 w-4 border-border-strong text-accent-fg focus:ring-indigo-500/30"
                      aria-label={`Font: ${opt.label} — ${opt.description}`}
                    />
                    <span className="min-w-0">
                      <span className="block text-body-sm font-medium text-fg-default">
                        {opt.label}
                      </span>
                      <span className="block text-caption text-fg-default">
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
              onChange={(v) => {
                update({ letterSpacing: v as LetterSpacing })
                setLiveAnnouncement(`Letter spacing ${v}`)
              }}
            />

            {/* Word spacing */}
            <SpacingControl
              legend="Word Spacing"
              name="reading-word-spacing"
              options={wordSpacingSteps}
              value={prefs.wordSpacing}
              onChange={(v) => {
                update({ wordSpacing: v as WordSpacing })
                setLiveAnnouncement(`Word spacing ${v}`)
              }}
            />

            {/* Line height */}
            <SpacingControl
              legend="Line Height"
              name="reading-line-height"
              options={lineHeightSteps}
              value={prefs.lineHeight}
              onChange={(v) => {
                update({ lineHeight: v as LineHeight })
                setLiveAnnouncement(`Line height ${v}`)
              }}
            />

            {/* Reading ruler */}
            <div>
              <div className="flex items-center justify-between">
                <span className="text-overline text-fg-default">
                  Reading Ruler
                </span>
                <button
                  type="button"
                  role="switch"
                  aria-checked={prefs.rulerEnabled}
                  aria-label={`Reading ruler: ${prefs.rulerEnabled ? 'on' : 'off'}`}
                  onClick={() => update({ rulerEnabled: !prefs.rulerEnabled })}
                  className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40 ${ prefs.rulerEnabled ? 'bg-accent-solid dark:bg-indigo-500' : 'bg-slate-200 dark:bg-neutral-700' }`}
                >
                  <span
                    className={`inline-block h-3.5 w-3.5 rounded-full bg-surface-raised shadow-sm ${ prefs.rulerEnabled ? 'translate-x-4' : 'translate-x-0.5' }`}
                  />
                </button>
              </div>
              {prefs.rulerEnabled && (
                <div className="mt-2.5">
                  <p className="mb-1.5 text-caption text-fg-default">Ruler colour</p>
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
                          className={`h-5 w-8 rounded border-2 ${ prefs.rulerColor === opt.value ? 'border-indigo-500' : 'border-border-default' }`}
                        />
                        <span className="text-caption text-fg-default">{opt.label}</span>
                      </label>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Accessibility display — plan 12.7 */}
            <div className="border-t border-border-subtle pt-4 dark:border-border-subtle">
              <p className="mb-3 text-overline text-fg-default">
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
          className={`inline-block h-3.5 w-3.5 rounded-full bg-surface-raised shadow-sm motion-safe:transition-transform ${checked ? 'translate-x-4' : 'translate-x-0.5'}`}
        />
      </button>
      <div className="min-w-0 flex-1">
        <label htmlFor={id} className="cursor-pointer select-none text-body-sm font-medium text-fg-default">
          {label}
        </label>
        <p id={`${id}-desc`} className="mt-0.5 text-caption text-fg-default">
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
      <legend className="mb-2 text-overline text-fg-default">
        {legend}
      </legend>
      <div className="flex gap-1.5">
        {options.map((opt) => (
          <label
            key={opt.value}
            className={`flex flex-1 cursor-pointer items-center justify-center rounded-lg border px-2 py-1.5 text-caption font-medium ${ value === opt.value ? 'border-accent-border bg-accent-surface text-accent-fg' : 'border-border-default bg-surface-raised text-fg-muted hover:border-border-strong hover:bg-surface-base dark:bg-surface-overlay' }`}
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

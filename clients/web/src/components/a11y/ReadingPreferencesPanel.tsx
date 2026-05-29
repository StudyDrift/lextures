import { useEffect, useRef, useState } from 'react'
import { authorizedFetch } from '../../lib/api'
import { applyReadingPrefs, parseReadingPrefs } from '../../lib/reading-prefs'
import { LiveRegion } from './live-region'

type Prefs = {
  highContrast: boolean
  reduceMotion: boolean
}

async function fetchPrefs(): Promise<Prefs> {
  const res = await authorizedFetch('/api/v1/me/reading-preferences')
  if (!res.ok) throw new Error('Failed to load preferences')
  const raw: unknown = await res.json()
  return parseReadingPrefs(raw)
}

async function savePrefs(patch: Partial<Prefs>): Promise<Prefs> {
  const res = await authorizedFetch('/api/v1/me/reading-preferences', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  if (!res.ok) throw new Error('Failed to save preferences')
  const raw: unknown = await res.json()
  return parseReadingPrefs(raw)
}

type ToggleProps = {
  id: string
  label: string
  description: string
  checked: boolean
  disabled: boolean
  onChange: (value: boolean) => void
}

function PrefToggle({ id, label, description, checked, disabled, onChange }: ToggleProps) {
  return (
    <div className="flex items-start gap-3 py-3">
      <button
        id={id}
        role="switch"
        aria-checked={checked}
        aria-describedby={`${id}-desc`}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className="mt-0.5 relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent motion-safe:transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:opacity-50"
        style={{ backgroundColor: checked ? 'rgb(79 70 229)' : 'rgb(209 213 219)' }}
      >
        <span className="sr-only">{label}</span>
        <span
          aria-hidden="true"
          className="pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 motion-safe:transition-transform"
          style={{ transform: checked ? 'translateX(20px)' : 'translateX(0)' }}
        />
      </button>
      <div className="min-w-0 flex-1">
        <label htmlFor={id} className="text-sm font-medium text-slate-900 dark:text-neutral-100 cursor-pointer select-none">
          {label}
        </label>
        <p id={`${id}-desc`} className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
          {description}
        </p>
      </div>
    </div>
  )
}

type Props = {
  /** When provided, shows a read-only message instead of toggles. */
  accommodationLabel?: string
}

export function ReadingPreferencesPanel({ accommodationLabel }: Props) {
  const [prefs, setPrefs] = useState<Prefs>({ highContrast: false, reduceMotion: false })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [announcement, setAnnouncement] = useState('')
  const isMounted = useRef(true)

  useEffect(() => {
    isMounted.current = true
    fetchPrefs()
      .then((p) => {
        if (isMounted.current) setPrefs(p)
      })
      .catch(() => { /* use defaults */ })
      .finally(() => {
        if (isMounted.current) setLoading(false)
      })
    return () => {
      isMounted.current = false
    }
  }, [])

  async function toggle(field: keyof Prefs, value: boolean) {
    if (saving) return
    const next = { ...prefs, [field]: value }
    setPrefs(next)
    applyReadingPrefs(next)
    setSaving(true)
    try {
      const saved = await savePrefs({ [field]: value })
      if (isMounted.current) {
        setPrefs(saved)
        applyReadingPrefs(saved)
        window.dispatchEvent(new CustomEvent('lextures-reading-prefs-updated'))
        const label =
          field === 'highContrast'
            ? saved.highContrast
              ? 'High contrast enabled'
              : 'High contrast disabled'
            : saved.reduceMotion
            ? 'Reduce motion enabled'
            : 'Reduce motion disabled'
        setAnnouncement(label)
      }
    } catch {
      if (isMounted.current) {
        setPrefs((prev) => ({ ...prev, [field]: !value }))
        applyReadingPrefs({ ...prefs, [field]: !value })
      }
    } finally {
      if (isMounted.current) setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="py-4 text-sm text-slate-500 dark:text-neutral-400" aria-busy="true">
        Loading preferences…
      </div>
    )
  }

  return (
    <section aria-labelledby="reading-prefs-heading" className="space-y-1">
      <LiveRegion politeness="polite">{announcement}</LiveRegion>
      <h2 id="reading-prefs-heading" className="text-base font-semibold text-slate-900 dark:text-neutral-100">
        Accessibility display
      </h2>
      {accommodationLabel && (
        <p className="text-xs text-slate-500 dark:text-neutral-400 mb-2" role="note">
          {accommodationLabel}
        </p>
      )}
      <div className="divide-y divide-slate-100 dark:divide-neutral-800">
        <PrefToggle
          id="pref-high-contrast"
          label="High contrast"
          description="Increases contrast to at least 7:1 for all text and interactive elements."
          checked={prefs.highContrast}
          disabled={saving || !!accommodationLabel}
          onChange={(v) => toggle('highContrast', v)}
        />
        <PrefToggle
          id="pref-reduce-motion"
          label="Reduce motion"
          description="Stops all animations and transitions to reduce motion-triggered discomfort."
          checked={prefs.reduceMotion}
          disabled={saving || !!accommodationLabel}
          onChange={(v) => toggle('reduceMotion', v)}
        />
      </div>
    </section>
  )
}

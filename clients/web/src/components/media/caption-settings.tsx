import { useId } from 'react'
import type { CaptionPreferences, CaptionColorPreset, CaptionFontSize, CaptionPosition } from '../../lib/caption-preferences'

type Props = {
  open: boolean
  prefs: CaptionPreferences
  onChange: (next: CaptionPreferences) => void
  onClose: () => void
}

export function CaptionSettings({ open, prefs, onChange, onClose }: Props) {
  const titleId = useId()
  if (!open) return null

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="absolute bottom-14 right-2 z-20 w-72 rounded-lg border border-slate-600 bg-slate-900 p-4 text-white shadow-xl"
    >
      <div className="mb-3 flex items-center justify-between gap-2">
        <h2 id={titleId} className="text-sm font-semibold">
          Caption settings
        </h2>
        <button
          type="button"
          className="rounded px-2 py-1 text-xs hover:bg-slate-700"
          onClick={onClose}
          aria-label="Close caption settings"
        >
          Close
        </button>
      </div>

      <label className="mb-3 flex items-center justify-between gap-2 text-sm">
        <span>Font size</span>
        <select
          className="rounded bg-slate-800 px-2 py-1 text-sm"
          value={prefs.fontSize}
          aria-label="Caption font size"
          onChange={(e) =>
            onChange({ ...prefs, fontSize: e.target.value as CaptionFontSize })
          }
        >
          <option value="small">Small</option>
          <option value="medium">Medium</option>
          <option value="large">Large</option>
        </select>
      </label>

      <label className="mb-3 flex items-center justify-between gap-2 text-sm">
        <span>Colors</span>
        <select
          className="rounded bg-slate-800 px-2 py-1 text-sm"
          value={prefs.colorPreset}
          aria-label="Caption color preset"
          onChange={(e) =>
            onChange({ ...prefs, colorPreset: e.target.value as CaptionColorPreset })
          }
        >
          <option value="default">White on dark</option>
          <option value="high-contrast">High contrast</option>
          <option value="yellow-on-black">Yellow on black</option>
        </select>
      </label>

      <label className="mb-2 flex items-center justify-between gap-2 text-sm">
        <span>Position</span>
        <select
          className="rounded bg-slate-800 px-2 py-1 text-sm"
          value={prefs.position}
          aria-label="Caption position"
          onChange={(e) =>
            onChange({ ...prefs, position: e.target.value as CaptionPosition })
          }
        >
          <option value="bottom">Bottom</option>
          <option value="top">Top</option>
        </select>
      </label>

      <p className="mt-2 text-xs text-slate-400" role="status" aria-live="polite">
        Settings apply to videos this session.
      </p>
    </div>
  )
}

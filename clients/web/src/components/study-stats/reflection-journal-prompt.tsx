import { useState } from 'react'
import { createReflectionJournalEntry } from '../../lib/study-reflection-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

type Props = {
  courseId?: string
}

export function ReflectionJournalPrompt({ courseId }: Props) {
  const { selfReflectionEnabled } = usePlatformFeatures()
  const [open, setOpen] = useState(false)
  const [text, setText] = useState('')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  if (!selfReflectionEnabled) return null

  async function save() {
    const entry = text.trim()
    if (!entry) return
    setSaving(true)
    try {
      await createReflectionJournalEntry({ entryText: entry, courseId })
      setText('')
      setSaved(true)
      setOpen(false)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="mt-8 border-t border-slate-200 pt-6 dark:border-neutral-700">
      {saved ? (
        <p className="text-sm text-emerald-700 dark:text-emerald-300">Reflection saved to your private journal.</p>
      ) : null}
      {!open ? (
        <button
          type="button"
          className="text-sm font-medium text-slate-600 underline hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100"
          onClick={() => setOpen(true)}
        >
          How did this feel? (optional)
        </button>
      ) : (
        <div className="space-y-2">
          <label htmlFor="reflection-journal" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
            Quick reflection (private, max 280 characters)
          </label>
          <textarea
            id="reflection-journal"
            rows={3}
            maxLength={280}
            value={text}
            onChange={(e) => setText(e.target.value)}
            className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
            aria-label="Study session reflection"
            placeholder="e.g. Felt confused about recursion today"
          />
          <div className="flex gap-2">
            <button
              type="button"
              disabled={saving || !text.trim()}
              onClick={() => void save()}
              className="rounded-lg bg-slate-800 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
            <button
              type="button"
              className="rounded-lg px-3 py-1.5 text-sm text-slate-600 dark:text-neutral-400"
              onClick={() => setOpen(false)}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

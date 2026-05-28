import { useEffect, useState } from 'react'
import { TimezoneSelector } from '../timezone/timezone-selector'
import { useUserTimezone } from '../../hooks/use-user-timezone'
import { detectBrowserTimezone } from '../../lib/format'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

export function AccountTimezonePanel() {
  const { timezone, setTimezone, loading } = useUserTimezone()
  const [draft, setDraft] = useState(detectBrowserTimezone())
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (timezone) setDraft(timezone)
    else if (!loading) setDraft(detectBrowserTimezone())
  }, [timezone, loading])

  async function onSave() {
    setSaving(true)
    setError(null)
    try {
      await setTimezone(draft.trim() || null)
      toastSaveOk('Time zone saved')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Could not save time zone.'
      setError(msg)
      toastMutationError(msg)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="mt-8">
      <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">Time zone</p>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        Assignment due dates and availability windows are shown in this time zone.
      </p>
      <div className="mt-3 max-w-md">
        <TimezoneSelector
          value={draft}
          onChange={setDraft}
          disabled={loading || saving}
          showDetectedHint={!timezone}
        />
        <button
          type="button"
          disabled={loading || saving}
          onClick={() => void onSave()}
          className="mt-3 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Save time zone'}
        </button>
        {error && (
          <p className="mt-2 text-sm text-rose-600 dark:text-rose-400" role="status">
            {error}
          </p>
        )}
      </div>
    </div>
  )
}

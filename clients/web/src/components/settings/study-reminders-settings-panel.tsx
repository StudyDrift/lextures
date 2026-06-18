import { useCallback, useEffect, useId, useState } from 'react'
import { Link } from 'react-router-dom'
import { Bell, PauseCircle, Save } from 'lucide-react'
import {
  fetchReminderConfig,
  formatReminderTimeLabel,
  patchReminderConfig,
  pauseReminders,
  type ReminderConfig,
} from '../../lib/study-reminders-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { usePlatformFeatures } from '../../context/platform-features-context'

export function StudyRemindersSettingsPanel() {
  const { ffStudyReminders, loading: featuresLoading } = usePlatformFeatures()
  const enableId = useId()
  const timeId = useId()
  const goalId = useId()
  const goalHelpId = useId()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [pausing, setPausing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [config, setConfig] = useState<ReminderConfig | null>(null)
  const [emailChannel, setEmailChannel] = useState(true)
  const [pushChannel, setPushChannel] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const c = await fetchReminderConfig()
      setConfig(c)
      setEmailChannel(c.reminderChannels.includes('email'))
      setPushChannel(c.reminderChannels.includes('push'))
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load study reminders.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  if (featuresLoading || !ffStudyReminders) return null

  if (loading) {
    return <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">Loading study reminders…</p>
  }
  if (error) {
    return <p className="mt-4 text-sm text-rose-700 dark:text-rose-300">{error}</p>
  }
  if (!config) return null

  const channels = [
    ...(emailChannel ? ['email'] : []),
    ...(pushChannel ? ['push'] : []),
  ]

  return (
    <section aria-labelledby="study-reminders-heading" className="mt-6 rounded-2xl border border-slate-200 bg-white p-5 dark:border-neutral-700 dark:bg-neutral-900">
      <div className="flex items-start gap-3">
        <Bell className="mt-0.5 h-5 w-5 shrink-0 text-indigo-600 dark:text-indigo-300" aria-hidden />
        <div className="min-w-0 flex-1">
          <h3 id="study-reminders-heading" className="text-base font-semibold text-slate-900 dark:text-neutral-100">
            Study reminders
          </h3>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Set a daily study goal and get reminded when you have not studied yet today.
          </p>

          {config.pausedUntil ? (
            <p className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100">
              Reminders paused until {config.pausedUntil}.
            </p>
          ) : null}

          <div className="mt-5 space-y-5">
            <label className="flex items-center gap-3 text-sm font-medium text-slate-800 dark:text-neutral-200">
              <input
                id={enableId}
                type="checkbox"
                className="h-4 w-4 rounded border-slate-300"
                checked={config.enabled}
                onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
              />
              Enable daily study reminders
            </label>

            <div>
              <label htmlFor={goalId} className="block text-sm font-medium text-slate-800 dark:text-neutral-200">
                Daily goal (minutes)
              </label>
              <p id={goalHelpId} className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                Aim for a realistic daily target — most learners start with 15–30 minutes.
              </p>
              <input
                id={goalId}
                type="range"
                min={5}
                max={120}
                step={5}
                aria-describedby={goalHelpId}
                className="mt-2 w-full"
                value={config.dailyGoalMinutes}
                onChange={(e) => setConfig({ ...config, dailyGoalMinutes: Number(e.target.value) })}
              />
              <p className="mt-1 text-sm text-slate-700 dark:text-neutral-300">{config.dailyGoalMinutes} minutes</p>
            </div>

            <div>
              <label htmlFor={timeId} className="block text-sm font-medium text-slate-800 dark:text-neutral-200">
                Reminder time
              </label>
              <input
                id={timeId}
                type="time"
                className="mt-2 rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                value={config.reminderTime}
                onChange={(e) => setConfig({ ...config, reminderTime: e.target.value })}
              />
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                Uses your account time zone ({formatReminderTimeLabel(config.reminderTime)} local).
              </p>
            </div>

            <fieldset className="space-y-2">
              <legend className="text-sm font-medium text-slate-800 dark:text-neutral-200">Reminder channels</legend>
              <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
                <input type="checkbox" checked={emailChannel} onChange={(e) => setEmailChannel(e.target.checked)} />
                Email
              </label>
              <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
                <input type="checkbox" checked={pushChannel} onChange={(e) => setPushChannel(e.target.checked)} />
                Push notifications
              </label>
            </fieldset>

            <label className="flex items-center gap-3 text-sm text-slate-700 dark:text-neutral-300">
              <input
                type="checkbox"
                checked={config.weeklySummary}
                onChange={(e) => setConfig({ ...config, weeklySummary: e.target.checked })}
              />
              Weekly progress summary email (Sundays)
            </label>
          </div>

          <div className="mt-6 flex flex-wrap gap-3">
            <button
              type="button"
              disabled={saving || channels.length === 0}
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-60"
              onClick={() => {
                if (channels.length === 0) return
                setSaving(true)
                void patchReminderConfig({
                  enabled: config.enabled,
                  dailyGoalMinutes: config.dailyGoalMinutes,
                  reminderTime: config.reminderTime,
                  reminderChannels: channels,
                  weeklySummary: config.weeklySummary,
                })
                  .then((c) => {
                    setConfig(c)
                    toastSaveOk()
                  })
                  .catch((e) => toastMutationError(e instanceof Error ? e.message : 'Could not save.'))
                  .finally(() => setSaving(false))
              }}
            >
              <Save className="h-4 w-4" aria-hidden />
              Save reminders
            </button>
            <button
              type="button"
              disabled={pausing}
              className="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-50 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
              onClick={() => {
                setPausing(true)
                void pauseReminders(7)
                  .then(setConfig)
                  .catch((e) => toastMutationError(e instanceof Error ? e.message : 'Could not pause.'))
                  .finally(() => setPausing(false))
              }}
            >
              <PauseCircle className="h-4 w-4" aria-hidden />
              Pause 7 days
            </button>
            <Link
              to="/settings/notifications"
              className="inline-flex items-center self-center text-sm font-medium text-indigo-700 underline dark:text-indigo-300"
            >
              Manage email preferences
            </Link>
          </div>
        </div>
      </div>
    </section>
  )
}

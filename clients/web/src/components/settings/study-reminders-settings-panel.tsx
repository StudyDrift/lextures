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

type Props = {
  embedded?: boolean
}

export function StudyRemindersSettingsPanel({ embedded = false }: Props) {
  const { ffStudyReminders, loading: featuresLoading } = usePlatformFeatures()
  const enableId = useId()
  const timeId = useId()
  const goalId = useId()
  const goalHelpId = useId()
  const [loading, setLoading] = useState(false)
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
    if (featuresLoading || !ffStudyReminders) return
    void load()
  }, [load, featuresLoading, ffStudyReminders])

  if (featuresLoading || !ffStudyReminders) return null

  if (loading) {
    return <p className="mt-4 text-sm text-fg-muted">Loading study reminders…</p>
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
    <section
      aria-labelledby="study-reminders-heading"
      className={`${embedded ? '' : 'mt-6 '}rounded-2xl border border-border-default bg-surface-raised p-5 dark:border-border-default dark:bg-surface-raised`}
    >
      <div className="flex items-start gap-3">
        <Bell className="mt-0.5 h-5 w-5 shrink-0 text-accent-fg dark:text-indigo-300" aria-hidden />
        <div className="min-w-0 flex-1">
          <h3 id="study-reminders-heading" className="text-base font-semibold text-fg-default">
            Study reminders
          </h3>
          <p className="mt-1 text-sm text-fg-muted">
            Set a daily study goal and get reminded when you have not studied yet today.
          </p>

          {config.pausedUntil ? (
            <p className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100">
              Reminders paused until {config.pausedUntil}.
            </p>
          ) : null}

          <div className="mt-5 space-y-5">
            <label className="flex items-center gap-3 text-sm font-medium text-fg-default">
              <input
                id={enableId}
                type="checkbox"
                className="h-4 w-4 rounded border-border-strong"
                checked={config.enabled}
                onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
              />
              Enable daily study reminders
            </label>

            <div>
              <label htmlFor={goalId} className="block text-sm font-medium text-fg-default">
                Daily goal (minutes)
              </label>
              <p id={goalHelpId} className="mt-1 text-xs text-fg-muted">
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
              <p className="mt-1 text-sm text-fg-muted">{config.dailyGoalMinutes} minutes</p>
            </div>

            <div>
              <label htmlFor={timeId} className="block text-sm font-medium text-fg-default">
                Reminder time
              </label>
              <input
                id={timeId}
                type="time"
                className="mt-2 rounded-lg border border-border-default px-3 py-2 text-sm dark:border-border-default dark:bg-surface-base"
                value={config.reminderTime}
                onChange={(e) => setConfig({ ...config, reminderTime: e.target.value })}
              />
              <p className="mt-1 text-xs text-fg-muted">
                Uses your account time zone ({formatReminderTimeLabel(config.reminderTime)} local).
              </p>
            </div>

            <fieldset className="space-y-2">
              <legend className="text-sm font-medium text-fg-default">Reminder channels</legend>
              <label className="flex items-center gap-2 text-sm text-fg-muted">
                <input type="checkbox" checked={emailChannel} onChange={(e) => setEmailChannel(e.target.checked)} />
                Email
              </label>
              <label className="flex items-center gap-2 text-sm text-fg-muted">
                <input type="checkbox" checked={pushChannel} onChange={(e) => setPushChannel(e.target.checked)} />
                Push notifications
              </label>
            </fieldset>

            <label className="flex items-center gap-3 text-sm text-fg-muted">
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
              className="inline-flex items-center gap-2 rounded-lg bg-accent-solid px-4 py-2 text-sm font-semibold text-white hover:bg-accent disabled:opacity-60"
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
              className="inline-flex items-center gap-2 rounded-lg border border-border-default bg-surface-raised px-4 py-2 text-sm font-semibold text-fg-default hover:bg-surface-base disabled:opacity-60 dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
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
              className="inline-flex items-center self-center text-sm font-medium text-accent-fg underline dark:text-indigo-300"
            >
              Manage email preferences
            </Link>
          </div>
        </div>
      </div>
    </section>
  )
}

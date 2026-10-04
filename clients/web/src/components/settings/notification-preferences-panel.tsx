import { useCallback, useEffect, useId, useState } from 'react'
import { Save } from 'lucide-react'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { subscribeToPush, getExistingPushSubscription } from '../../lib/push-notifications'

type PreferenceRow = {
  eventType: string
  emailEnabled: boolean
  pushEnabled: boolean
  smsEnabled: boolean
  digestMode: 'instant' | 'daily' | 'off'
}

type EventCopy = {
  label: string
  description?: string
}

// Labels cover notificationevents.All. Unknown keys are title-cased so the
// Event column never shows a raw snake_case key.
const EVENT_COPY: Record<string, EventCopy> = {
  grade_posted: { label: 'Grade posted' },
  assignment_created: { label: 'New assignment' },
  discussion_reply: { label: 'Discussion reply' },
  course_announcement: { label: 'Course announcement' },
  submission_received: { label: 'Submission received' },
  assignment_due_reminder: { label: 'Assignment due reminder' },
  password_reset: { label: 'Password reset' },
  welcome_invite: { label: 'Welcome / invite' },
  meeting_reminder: { label: 'Meeting reminder' },
  conference_confirmed: { label: 'Conference confirmed' },
  conference_reminder: { label: 'Conference reminder' },
  coaching_tip_weekly: { label: 'Weekly coaching tip' },
  canvas_course_imported: { label: 'Canvas course imported' },
  course_copy_imported: { label: 'Course copied from another course' },
  course_copy_import_failed: { label: 'Course copy failed' },
  inbox_message: { label: 'Inbox message' },
  incomplete_granted: { label: 'Incomplete granted' },
  incomplete_reminder: { label: 'Incomplete reminder' },
  ceu_awarded: { label: 'Continuing education credit awarded' },
  certificate_issued: { label: 'Certificate issued' },
  payment_failed: { label: 'Payment failed' },
  study_reminder_daily: { label: 'Daily study reminder' },
  study_reminder_streak_at_risk: { label: 'Study streak at risk' },
  study_reminder_weekly_summary: { label: 'Weekly study summary' },
  seat_utilization_alert: { label: 'Seat use alert' },
  intro_course_completed: { label: 'Intro course completed' },
  transcript_order_submitted: { label: 'Transcript order submitted' },
  transcript_order_on_hold: { label: 'Transcript order on hold' },
  transcript_order_consent_needed: { label: 'Transcript consent needed' },
  transcript_order_payment_needed: { label: 'Transcript payment needed' },
  transcript_order_approved: { label: 'Transcript order approved' },
  transcript_order_rejected: { label: 'Transcript order rejected' },
  transcript_order_sent: { label: 'Transcript sent' },
  transcript_order_delivered: { label: 'Transcript delivered' },
  transcript_order_opened: { label: 'Transcript opened' },
  transcript_order_failed: { label: 'Transcript delivery failed' },
  transcript_order_canceled: { label: 'Transcript order canceled' },
  transcript_order_exception: { label: 'Transcript order needs attention' },
  adaptive_content_regressing: {
    label: 'Adapted content may be hurting learning',
    description: 'An adapted section is doing worse than the unchanged version.',
  },
  adaptive_content_fairness: {
    label: 'Uneven results across learner groups',
    description: 'Adapted content may be working better for some learners than others.',
  },
  adaptive_content_contest: {
    label: 'Reported problem with adapted content',
    description: 'A learner says an adapted section looks wrong.',
  },
  content_tool_state_reset: { label: 'Activity progress reset' },
}

function readableEventLabel(eventType: string): string {
  const known = EVENT_COPY[eventType]?.label
  if (known) return known
  const words = eventType.split('_').filter(Boolean)
  if (words.length === 0) return eventType
  const [first, ...rest] = words
  return [first.charAt(0).toUpperCase() + first.slice(1), ...rest].join(' ')
}

const DIGEST_OPTIONS: { value: PreferenceRow['digestMode']; label: string }[] = [
  { value: 'instant', label: 'Instant email' },
  { value: 'daily', label: 'Daily digest' },
  { value: 'off', label: 'Off' },
]

export function NotificationPreferencesPanel() {
  const baseId = useId()
  const [rows, setRows] = useState<PreferenceRow[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [pushSubscribed, setPushSubscribed] = useState(false)
  const [pushLoading, setPushLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await authorizedFetch('/api/v1/me/notification-preferences')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error(readApiErrorMessage(raw))
      }
      const list = (raw as { preferences?: PreferenceRow[] }).preferences ?? []
      setRows(list)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not load preferences.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
    void getExistingPushSubscription().then((sub) => setPushSubscribed(!!sub))
  }, [load])

  const updateRow = (eventType: string, patch: Partial<PreferenceRow>) => {
    setRows((prev) =>
      prev.map((r) => (r.eventType === eventType ? { ...r, ...patch } : r)),
    )
  }

  const save = async () => {
    setSaving(true)
    try {
      const res = await authorizedFetch('/api/v1/me/notification-preferences', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ preferences: rows }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error(readApiErrorMessage(raw))
      }
      const list = (raw as { preferences?: PreferenceRow[] }).preferences ?? []
      setRows(list)
      toastSaveOk('Notification preferences saved.')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not save preferences.')
    } finally {
      setSaving(false)
    }
  }

  const enablePush = async () => {
    setPushLoading(true)
    try {
      const sub = await subscribeToPush()
      if (sub) {
        setPushSubscribed(true)
        toastSaveOk('Push notifications enabled.')
      } else {
        toastMutationError('Could not enable push notifications. Check browser permissions.')
      }
    } finally {
      setPushLoading(false)
    }
  }

  if (loading) {
    return <p className="mt-4 text-sm text-fg-muted">Loading notification preferences…</p>
  }

  return (
    <div>
      <p className="mt-2 text-sm text-fg-muted">
        Choose which events send you notifications.
      </p>

      {/* Push enable banner */}
      {!pushSubscribed && 'Notification' in window && (
        <div className="mt-4 flex items-center justify-between rounded-xl border border-indigo-200 bg-indigo-50 px-4 py-3 dark:border-indigo-800 dark:bg-indigo-950/30">
          <div>
            <p className="text-sm font-medium text-indigo-900 dark:text-indigo-200">Enable browser push notifications</p>
            <p className="text-xs text-accent-fg dark:text-indigo-400">Get real-time alerts even when the tab is in the background.</p>
          </div>
          <button
            type="button"
            onClick={() => void enablePush()}
            disabled={pushLoading}
            className="shrink-0 rounded-lg bg-accent-solid px-3 py-1.5 text-sm font-medium text-white hover:bg-accent disabled:opacity-60"
          >
            {pushLoading ? 'Enabling…' : 'Enable push'}
          </button>
        </div>
      )}

      <div className="mt-4 overflow-x-auto rounded-xl border border-border-default">
        <table className="min-w-full text-sm" data-testid="notification-preferences-table">
          <thead className="bg-surface-base text-start text-xs font-semibold uppercase tracking-wide text-fg-muted dark:bg-surface-overlay dark:text-fg-muted">
            <tr>
              <th className="px-4 py-3" scope="col">
                Event
              </th>
              <th className="px-4 py-3" scope="col">
                Email
              </th>
              <th className="px-4 py-3" scope="col">
                Push
              </th>
              <th className="px-4 py-3" scope="col">
                SMS
              </th>
              <th className="px-4 py-3" scope="col">
                Delivery
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
            {rows.map((row) => {
              const emailId = `${baseId}-${row.eventType}-email`
              const pushId = `${baseId}-${row.eventType}-push`
              const smsId = `${baseId}-${row.eventType}-sms`
              const digestId = `${baseId}-${row.eventType}-digest`
              const label = readableEventLabel(row.eventType)
              const description = EVENT_COPY[row.eventType]?.description
              return (
                <tr key={row.eventType} className="bg-surface-raised">
                  <td className="px-4 py-3 font-medium text-fg-default">
                    <span className="block">{label}</span>
                    {description ? (
                      <span className="mt-0.5 block text-xs font-normal text-fg-muted">
                        {description}
                      </span>
                    ) : null}
                  </td>
                  <td className="px-4 py-3">
                    <label htmlFor={emailId} className="sr-only">
                      Email for {label}
                    </label>
                    <button
                      id={emailId}
                      type="button"
                      role="switch"
                      aria-checked={row.emailEnabled}
                      onClick={() =>
                        updateRow(row.eventType, { emailEnabled: !row.emailEnabled })
                      }
                      className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${ row.emailEnabled ? 'bg-accent-solid' : 'bg-slate-200 dark:bg-neutral-600' }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-surface-raised shadow transition-transform ${ row.emailEnabled ? 'translate-x-5' : 'translate-x-0' }`}
                      />
                    </button>
                  </td>
                  <td className="px-4 py-3">
                    <label htmlFor={pushId} className="sr-only">
                      Push for {label}
                    </label>
                    <button
                      id={pushId}
                      type="button"
                      role="switch"
                      aria-checked={row.pushEnabled}
                      onClick={() =>
                        updateRow(row.eventType, { pushEnabled: !row.pushEnabled })
                      }
                      className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${ row.pushEnabled ? 'bg-accent-solid' : 'bg-slate-200 dark:bg-neutral-600' }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-surface-raised shadow transition-transform ${ row.pushEnabled ? 'translate-x-5' : 'translate-x-0' }`}
                      />
                    </button>
                  </td>
                  <td className="px-4 py-3">
                    <label htmlFor={smsId} className="sr-only">
                      SMS for {label}
                    </label>
                    <button
                      id={smsId}
                      type="button"
                      role="switch"
                      aria-checked={row.smsEnabled}
                      onClick={() =>
                        updateRow(row.eventType, { smsEnabled: !row.smsEnabled })
                      }
                      className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${ row.smsEnabled ? 'bg-accent-solid' : 'bg-slate-200 dark:bg-neutral-600' }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-surface-raised shadow transition-transform ${ row.smsEnabled ? 'translate-x-5' : 'translate-x-0' }`}
                      />
                    </button>
                  </td>
                  <td className="px-4 py-3">
                    <label htmlFor={digestId} className="sr-only">
                      Delivery for {label}
                    </label>
                    <select
                      id={digestId}
                      value={row.digestMode}
                      disabled={!row.emailEnabled}
                      onChange={(e) =>
                        updateRow(row.eventType, {
                          digestMode: e.target.value as PreferenceRow['digestMode'],
                        })
                      }
                      className="rounded-lg border border-border-default bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-raised"
                    >
                      {DIGEST_OPTIONS.map((o) => (
                        <option key={o.value} value={o.value}>
                          {o.label}
                        </option>
                      ))}
                    </select>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <div className="mt-4 flex justify-end">
        <button
          type="button"
          onClick={() => void save()}
          disabled={saving}
          className="inline-flex items-center gap-2 rounded-lg bg-accent-solid px-4 py-2 text-sm font-medium text-white hover:bg-accent disabled:opacity-60"
        >
          <Save className="h-4 w-4" aria-hidden />
          {saving ? 'Saving…' : 'Save preferences'}
        </button>
      </div>
    </div>
  )
}

import { useId, useState } from 'react'
import {
  deleteBotMapping,
  upsertBotMapping,
  type BotChannelMapping,
  type BotConnection,
} from '../lib/bots-api'

const BOT_EVENT_TYPES = [
  'assignment.created',
  'assignment.due_soon',
  'grade.released',
  'announcement.created',
] as const

const EVENT_LABEL: Record<(typeof BOT_EVENT_TYPES)[number], string> = {
  'assignment.created': 'New assignments',
  'assignment.due_soon': 'Due-date reminders',
  'grade.released': 'Grade releases',
  'announcement.created': 'Announcements',
}

type Props = {
  connection: BotConnection
  onUpdated: () => void
}

/** Admin panel for mapping course events to platform channels (plan 16.6). */
export function BotChannelMappingsPanel({ connection, onUpdated }: Props) {
  const channelId = useId()
  const courseId = useId()
  const [channel, setChannel] = useState('')
  const [course, setCourse] = useState('')
  const [events, setEvents] = useState<string[]>(['assignment.created', 'announcement.created'])
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const mappings = connection.mappings ?? []

  function toggleEvent(eventType: string) {
    setEvents((prev) =>
      prev.includes(eventType) ? prev.filter((e) => e !== eventType) : [...prev, eventType],
    )
  }

  async function saveMapping() {
    if (!channel.trim()) {
      setError('Channel ID is required.')
      return
    }
    if (events.length === 0) {
      setError('Select at least one event type.')
      return
    }
    setBusy('save')
    setError(null)
    try {
      await upsertBotMapping(connection.id, {
        channelId: channel.trim(),
        courseId: course.trim() || undefined,
        eventTypes: events,
      })
      setChannel('')
      setCourse('')
      onUpdated()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save mapping.')
    } finally {
      setBusy(null)
    }
  }

  async function removeMapping(mapping: BotChannelMapping) {
    setBusy(`delete-${mapping.id}`)
    setError(null)
    try {
      await deleteBotMapping(connection.id, mapping.id)
      onUpdated()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete mapping.')
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="mt-4 border-t border-slate-200 pt-4 dark:border-neutral-600" data-testid="bot-mappings">
      <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        Channel mappings
      </h4>
      {mappings.length > 0 ? (
        <ul className="mt-2 space-y-1 text-xs text-slate-600 dark:text-neutral-400">
          {mappings.map((m) => (
            <li key={m.id} className="flex items-center justify-between gap-2">
              <span>
                <code className="rounded bg-slate-100 px-1 dark:bg-neutral-800">{m.channelId}</code>
                {m.courseId ? ` · course ${m.courseId.slice(0, 8)}…` : ' · all courses'}
                {' · '}
                {m.eventTypes.join(', ')}
              </span>
              <button
                type="button"
                className="text-rose-600 hover:text-rose-500"
                disabled={busy === `delete-${m.id}`}
                onClick={() => void removeMapping(m)}
              >
                Remove
              </button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-2 text-xs text-slate-500 dark:text-neutral-500">
          No channel mappings yet. Add one below to start receiving notifications.
        </p>
      )}

      <div className="mt-3 space-y-2">
        <label htmlFor={channelId} className="block text-xs font-medium text-slate-700 dark:text-neutral-300">
          Channel ID
        </label>
        <input
          id={channelId}
          type="text"
          value={channel}
          onChange={(e) => setChannel(e.target.value)}
          placeholder="C01234567 or #general channel id"
          className="w-full rounded border border-slate-300 px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-900"
        />
        <label htmlFor={courseId} className="block text-xs font-medium text-slate-700 dark:text-neutral-300">
          Course ID (optional)
        </label>
        <input
          id={courseId}
          type="text"
          value={course}
          onChange={(e) => setCourse(e.target.value)}
          placeholder="Leave blank for org-wide notifications"
          className="w-full rounded border border-slate-300 px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-900"
        />
        <fieldset>
          <legend className="text-xs font-medium text-slate-700 dark:text-neutral-300">Events</legend>
          <ul className="mt-1 space-y-1">
            {BOT_EVENT_TYPES.map((eventType) => (
              <li key={eventType}>
                <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-neutral-400">
                  <input
                    type="checkbox"
                    checked={events.includes(eventType)}
                    onChange={() => toggleEvent(eventType)}
                  />
                  {EVENT_LABEL[eventType]}
                </label>
              </li>
            ))}
          </ul>
        </fieldset>
        {error ? (
          <p className="text-xs text-rose-600" role="alert">
            {error}
          </p>
        ) : null}
        <button
          type="button"
          disabled={busy === 'save'}
          onClick={() => void saveMapping()}
          className="rounded bg-slate-800 px-3 py-1 text-xs font-medium text-white hover:bg-slate-900 disabled:opacity-50 dark:bg-neutral-200 dark:text-neutral-900"
        >
          {busy === 'save' ? 'Saving…' : 'Add mapping'}
        </button>
      </div>
    </div>
  )
}
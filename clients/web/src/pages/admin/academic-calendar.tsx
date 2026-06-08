import { useCallback, useEffect, useRef, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { fetchOrgTerms, type OrgTerm } from '../../lib/courses-api'
import {
  fetchCalendarEvents,
  createCalendarEvent,
  patchCalendarEvent,
  deleteCalendarEvent,
  type AcademicCalendarEvent,
  type CalendarEventType,
} from '../../lib/courses-api'
import { getJwtSubject } from '../../lib/auth'

const EVENT_TYPES: { value: CalendarEventType; label: string }[] = [
  { value: 'term_start', label: 'Term Start' },
  { value: 'term_end', label: 'Term End' },
  { value: 'add_drop_deadline', label: 'Add/Drop Deadline' },
  { value: 'withdrawal_deadline', label: 'Withdrawal Deadline' },
  { value: 'finals_start', label: 'Finals Start' },
  { value: 'finals_end', label: 'Finals End' },
  { value: 'no_class_day', label: 'No Class Day' },
  { value: 'holiday', label: 'Holiday' },
  { value: 'custom', label: 'Custom' },
]

function eventTypeLabel(t: string) {
  return EVENT_TYPES.find((e) => e.value === t)?.label ?? t
}

type ModalState =
  | { mode: 'closed' }
  | { mode: 'create' }
  | { mode: 'edit'; event: AcademicCalendarEvent }

function EventModal({
  state,
  orgId,
  termId,
  onClose,
  onSaved,
}: {
  state: ModalState
  orgId: string
  termId: string
  onClose: () => void
  onSaved: (ev: AcademicCalendarEvent) => void
}) {
  const isEdit = state.mode === 'edit'
  const existing = state.mode === 'edit' ? state.event : null

  const [eventType, setEventType] = useState<CalendarEventType>(existing?.eventType ?? 'holiday')
  const [eventName, setEventName] = useState(existing?.eventName ?? '')
  const [startDate, setStartDate] = useState(existing?.startDate ?? '')
  const [endDate, setEndDate] = useState(existing?.endDate ?? '')
  const [notes, setNotes] = useState(existing?.notes ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const firstInputRef = useRef<HTMLInputElement>(null)
  useEffect(() => {
    firstInputRef.current?.focus()
  }, [])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!eventName.trim() || !startDate) return
    setSaving(true)
    setError(null)
    try {
      let saved: AcademicCalendarEvent
      if (isEdit && existing) {
        saved = await patchCalendarEvent(orgId, existing.id, {
          eventType,
          eventName: eventName.trim(),
          startDate,
          endDate: endDate || '',
          notes: notes || '',
        })
      } else {
        saved = await createCalendarEvent(orgId, {
          termId: termId || undefined,
          eventType,
          eventName: eventName.trim(),
          startDate,
          endDate: endDate || undefined,
          notes: notes || undefined,
        })
      }
      onSaved(saved)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save event.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={isEdit ? 'Edit calendar event' : 'Add calendar event'}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onClick={(e) => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="bg-white dark:bg-neutral-900 rounded-xl shadow-xl w-full max-w-md p-6">
        <h2 className="text-lg font-semibold mb-4">{isEdit ? 'Edit event' : 'Add calendar event'}</h2>
        <form onSubmit={(e) => { void handleSubmit(e) }} className="space-y-4">
          <div>
            <label htmlFor="cal-event-type" className="block text-sm font-medium mb-1">Event type</label>
            <select
              id="cal-event-type"
              value={eventType}
              onChange={(e) => setEventType(e.target.value as CalendarEventType)}
              className="w-full border rounded-lg px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
              required
            >
              {EVENT_TYPES.map((t) => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="cal-event-name" className="block text-sm font-medium mb-1">Event name</label>
            <input
              id="cal-event-name"
              ref={firstInputRef}
              type="text"
              value={eventName}
              onChange={(e) => setEventName(e.target.value)}
              placeholder="e.g. Spring Break"
              className="w-full border rounded-lg px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
              required
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label htmlFor="cal-start-date" className="block text-sm font-medium mb-1">Start date</label>
              <input
                id="cal-start-date"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="w-full border rounded-lg px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                required
              />
            </div>
            <div>
              <label htmlFor="cal-end-date" className="block text-sm font-medium mb-1">End date <span className="text-slate-400">(opt.)</span></label>
              <input
                id="cal-end-date"
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                className="w-full border rounded-lg px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
              />
            </div>
          </div>
          <div>
            <label htmlFor="cal-notes" className="block text-sm font-medium mb-1">Notes <span className="text-slate-400">(opt.)</span></label>
            <textarea
              id="cal-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
              className="w-full border rounded-lg px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            />
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <div className="flex justify-end gap-2 pt-1">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm rounded-lg border dark:border-neutral-600 hover:bg-slate-50 dark:hover:bg-neutral-800"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="px-4 py-2 text-sm rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add event'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default function AcademicCalendarAdminPage() {
  const { ffAcademicCalendar } = usePlatformFeatures()
  const [orgId, setOrgId] = useState('')
  const [terms, setTerms] = useState<OrgTerm[]>([])
  const [selectedTermId, setSelectedTermId] = useState('')
  const [events, setEvents] = useState<AcademicCalendarEvent[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [modal, setModal] = useState<ModalState>({ mode: 'closed' })
  const [deleting, setDeleting] = useState<string | null>(null)

  useEffect(() => {
    const sub = getJwtSubject()
    if (!sub) return
    // Derive orgId from the current user's org via platform settings context — for this page we
    // allow the admin to type in the org UUID directly or select from a text field.
  }, [])

  const loadTerms = useCallback((oid: string) => {
    if (!oid.trim()) return
    void fetchOrgTerms(oid)
      .then(setTerms)
      .catch(() => setTerms([]))
  }, [])

  const loadEvents = useCallback(() => {
    if (!orgId.trim()) return
    setLoading(true)
    setError(null)
    void fetchCalendarEvents(orgId.trim(), selectedTermId || undefined)
      .then(setEvents)
      .catch((e: unknown) => {
        setError(e instanceof Error ? e.message : 'Failed to load events.')
        setEvents([])
      })
      .finally(() => setLoading(false))
  }, [orgId, selectedTermId])

  useEffect(() => {
    if (ffAcademicCalendar && orgId.trim()) loadEvents()
  }, [ffAcademicCalendar, loadEvents, orgId, selectedTermId])

  async function handleDelete(eventId: string) {
    if (!window.confirm('Delete this calendar event?')) return
    setDeleting(eventId)
    try {
      await deleteCalendarEvent(orgId, eventId)
      setEvents((prev) => prev.filter((e) => e.id !== eventId))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete event.')
    } finally {
      setDeleting(null)
    }
  }

  function handleSaved(ev: AcademicCalendarEvent) {
    setModal({ mode: 'closed' })
    setEvents((prev) => {
      const idx = prev.findIndex((e) => e.id === ev.id)
      if (idx >= 0) {
        const next = [...prev]
        next[idx] = ev
        return next
      }
      return [...prev, ev].sort((a, b) => a.startDate.localeCompare(b.startDate))
    })
  }

  if (!ffAcademicCalendar) {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <h1 className="text-xl font-semibold mb-2">Academic Calendar</h1>
        <p className="text-sm text-slate-600">Academic calendar is not enabled for this platform.</p>
      </div>
    )
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <h1 className="text-xl font-semibold mb-1">Academic Calendar</h1>
      <p className="text-sm text-slate-600 dark:text-neutral-400 mb-6">
        Manage institutional calendar events — add/drop deadlines, finals, holidays, and no-class days.
      </p>

      <div className="flex flex-wrap gap-3 mb-6 items-end">
        <div>
          <label htmlFor="org-id-input" className="block text-sm font-medium mb-1">Org ID</label>
          <input
            id="org-id-input"
            type="text"
            value={orgId}
            onChange={(e) => {
              setOrgId(e.target.value)
              loadTerms(e.target.value.trim())
            }}
            placeholder="Organization UUID"
            className="border rounded-lg px-3 py-2 text-sm w-72 dark:border-neutral-600 dark:bg-neutral-800"
          />
        </div>
        {terms.length > 0 && (
          <div>
            <label htmlFor="term-select" className="block text-sm font-medium mb-1">Term</label>
            <select
              id="term-select"
              value={selectedTermId}
              onChange={(e) => setSelectedTermId(e.target.value)}
              className="border rounded-lg px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            >
              <option value="">All terms</option>
              {terms.map((t) => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
          </div>
        )}
        <button
          onClick={() => setModal({ mode: 'create' })}
          disabled={!orgId.trim()}
          className="px-4 py-2 text-sm rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-40"
        >
          Add event
        </button>
      </div>

      {loading && <p className="text-sm text-slate-500">Loading…</p>}
      {error && <p className="text-sm text-red-600 mb-3">{error}</p>}

      {!loading && events.length === 0 && orgId.trim() && (
        <div className="border border-dashed rounded-xl p-8 text-center text-slate-500 dark:border-neutral-700">
          <p className="mb-2">No calendar events configured.</p>
          <button
            onClick={() => setModal({ mode: 'create' })}
            className="text-indigo-600 hover:underline text-sm"
          >
            Add the first event
          </button>
        </div>
      )}

      {events.length > 0 && (
        <div
          role="grid"
          aria-label="Calendar events"
          className="border rounded-xl overflow-hidden dark:border-neutral-700"
        >
          <div
            role="row"
            className="grid grid-cols-[1fr_1fr_1fr_auto] gap-4 px-4 py-2 bg-slate-50 dark:bg-neutral-800 text-xs font-medium text-slate-500 uppercase tracking-wide"
          >
            <span role="columnheader">Event</span>
            <span role="columnheader">Type</span>
            <span role="columnheader">Date</span>
            <span role="columnheader" className="sr-only">Actions</span>
          </div>
          {events.map((ev) => (
            <div
              key={ev.id}
              role="row"
              className="grid grid-cols-[1fr_1fr_1fr_auto] gap-4 px-4 py-3 border-t dark:border-neutral-700 items-center text-sm"
            >
              <span role="gridcell" className="font-medium">{ev.eventName}</span>
              <span role="gridcell" className="text-slate-600 dark:text-neutral-400">{eventTypeLabel(ev.eventType)}</span>
              <span role="gridcell" className="text-slate-600 dark:text-neutral-400">
                <time dateTime={ev.startDate}>{ev.startDate}</time>
                {ev.endDate && (
                  <> – <time dateTime={ev.endDate}>{ev.endDate}</time></>
                )}
              </span>
              <span role="gridcell" className="flex gap-2">
                <button
                  onClick={() => setModal({ mode: 'edit', event: ev })}
                  className="text-xs text-indigo-600 hover:underline"
                  aria-label={`Edit ${ev.eventName}`}
                >
                  Edit
                </button>
                <button
                  onClick={() => { void handleDelete(ev.id) }}
                  disabled={deleting === ev.id}
                  className="text-xs text-red-600 hover:underline disabled:opacity-40"
                  aria-label={`Delete ${ev.eventName}`}
                >
                  {deleting === ev.id ? '…' : 'Delete'}
                </button>
              </span>
            </div>
          ))}
        </div>
      )}

      {modal.mode !== 'closed' && (
        <EventModal
          state={modal}
          orgId={orgId.trim()}
          termId={selectedTermId}
          onClose={() => setModal({ mode: 'closed' })}
          onSaved={handleSaved}
        />
      )}
    </div>
  )
}

import { useCallback, useEffect, useState } from 'react'
import { CalendarDays } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createConferenceAvailability,
  fetchMyUserId,
  type CreateAvailabilityInput,
} from '../../lib/conferences-api'

const SLOT_DURATIONS = [5, 10, 15, 20, 30] as const

export default function ConferenceAvailabilitySetup() {
  const { ffConferenceScheduling } = usePlatformFeatures()
  const [teacherId, setTeacherId] = useState('')
  const [schoolId, setSchoolId] = useState('')
  const [date, setDate] = useState('')
  const [windowStart, setWindowStart] = useState('16:00')
  const [windowEnd, setWindowEnd] = useState('18:00')
  const [slotDuration, setSlotDuration] = useState<number>(15)
  const [gapDuration, setGapDuration] = useState(5)
  const [location, setLocation] = useState('')
  const [videoLink, setVideoLink] = useState('')
  const [mode, setMode] = useState<'in_person' | 'virtual'>('in_person')
  const [status, setStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    void fetchMyUserId().then(setTeacherId).catch(() => setError('Could not load your profile.'))
  }, [])

  const handleSubmit = useCallback(async () => {
    if (!teacherId || !schoolId.trim() || !date) {
      setError('School ID and date are required.')
      return
    }
    setSaving(true)
    setError(null)
    setStatus(null)
    try {
      const input: CreateAvailabilityInput = {
        schoolId: schoolId.trim(),
        date,
        windowStart,
        windowEnd,
        slotDuration,
        gapDuration,
        location: mode === 'in_person' && location.trim() ? location.trim() : null,
        videoLink: mode === 'virtual' && videoLink.trim() ? videoLink.trim() : null,
      }
      const result = await createConferenceAvailability(teacherId, input)
      setStatus(`Saved ${result.slots.length} bookable slots for ${date}.`)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save availability.')
    } finally {
      setSaving(false)
    }
  }, [teacherId, schoolId, date, windowStart, windowEnd, slotDuration, gapDuration, location, videoLink, mode])

  if (!ffConferenceScheduling) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-10">
        <p className="text-sm text-neutral-600 dark:text-neutral-400">
          Conference scheduling is not enabled on this platform.
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6 px-4 py-8 md:px-8">
      <header className="flex flex-col gap-2 border-b border-slate-200 pb-6 dark:border-neutral-800">
        <div className="flex items-center gap-2 text-sm font-medium text-indigo-700 dark:text-indigo-300">
          <CalendarDays className="h-4 w-4" aria-hidden />
          Conference availability
        </div>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
          Set parent-teacher conference slots
        </h1>
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Define your available window; the system generates bookable slots for parents in the portal.
        </p>
      </header>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-100">
          {error}
        </div>
      )}
      {status && (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-100">
          {status}
        </div>
      )}

      <form
        className="flex flex-col gap-4"
        onSubmit={(e) => {
          e.preventDefault()
          void handleSubmit()
        }}
      >
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">School org unit ID</span>
          <input
            type="text"
            value={schoolId}
            onChange={(e) => setSchoolId(e.target.value)}
            className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
            placeholder="UUID of your school org unit"
            required
          />
        </label>

        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">Conference date</span>
          <input
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
            required
          />
        </label>

        <div className="grid gap-4 sm:grid-cols-2">
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">Start time</span>
            <input
              type="time"
              value={windowStart}
              onChange={(e) => setWindowStart(e.target.value)}
              className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
              required
            />
          </label>
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">End time</span>
            <input
              type="time"
              value={windowEnd}
              onChange={(e) => setWindowEnd(e.target.value)}
              className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
              required
            />
          </label>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">Slot duration (minutes)</span>
            <select
              value={slotDuration}
              onChange={(e) => setSlotDuration(Number(e.target.value))}
              className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
            >
              {SLOT_DURATIONS.map((d) => (
                <option key={d} value={d}>
                  {d} minutes
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">Gap between slots (minutes)</span>
            <input
              type="number"
              min={0}
              max={30}
              value={gapDuration}
              onChange={(e) => setGapDuration(Number(e.target.value))}
              className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
            />
          </label>
        </div>

        <fieldset className="flex flex-col gap-2">
          <legend className="text-sm font-medium">Conference mode</legend>
          <label className="flex items-center gap-2 text-sm">
            <input type="radio" name="mode" checked={mode === 'in_person'} onChange={() => setMode('in_person')} />
            In-person (room)
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input type="radio" name="mode" checked={mode === 'virtual'} onChange={() => setMode('virtual')} />
            Virtual (video link)
          </label>
        </fieldset>

        {mode === 'in_person' ? (
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">Room / location</span>
            <input
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
              placeholder="Room 204"
            />
          </label>
        ) : (
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">Video conference link</span>
            <input
              type="url"
              value={videoLink}
              onChange={(e) => setVideoLink(e.target.value)}
              className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
              placeholder="https://meet.example.com/abc-defg-hij"
            />
          </label>
        )}

        <button
          type="submit"
          disabled={saving}
          className="mt-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 disabled:opacity-60"
        >
          {saving ? 'Saving…' : 'Generate slots'}
        </button>
      </form>
    </div>
  )
}

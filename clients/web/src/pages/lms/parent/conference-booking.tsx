import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { CalendarHeart } from 'lucide-react'
import { SlotPicker } from '../../../components/conference/slot-picker'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import { fetchParentChildren, type ParentChildSummary } from '../../../lib/parent-api'
import {
  bookConferenceSlot,
  cancelConferenceBooking,
  listConferenceSlots,
  listParentConferenceTeachers,
  type ConferenceSlot,
  type ConferenceTeacher,
} from '../../../lib/conferences-api'

function childLabel(c: ParentChildSummary): string {
  const n = c.displayName?.trim()
  if (n) return n
  return c.email
}

function teacherLabel(t: ConferenceTeacher): string {
  return t.displayName?.trim() || 'Teacher'
}

export default function ConferenceBooking() {
  const { ffConferenceScheduling } = usePlatformFeatures()
  const [params, setParams] = useSearchParams()
  const [children, setChildren] = useState<ParentChildSummary[] | null>(null)
  const [teachers, setTeachers] = useState<ConferenceTeacher[]>([])
  const [selectedTeacherId, setSelectedTeacherId] = useState('')
  const [slots, setSlots] = useState<ConferenceSlot[]>([])
  const [conferenceDate, setConferenceDate] = useState('')
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [booking, setBooking] = useState(false)

  const selectedStudentId = params.get('student') ?? ''

  useEffect(() => {
    void fetchParentChildren()
      .then((data) => setChildren(data.children))
      .catch((e) => setLoadError(e instanceof Error ? e.message : 'Could not load children.'))
  }, [])

  useEffect(() => {
    if (!children || children.length === 0) return
    if (!selectedStudentId || !children.some((c) => c.studentUserId === selectedStudentId)) {
      setParams({ student: children[0].studentUserId }, { replace: true })
    }
  }, [children, selectedStudentId, setParams])

  useEffect(() => {
    if (!selectedStudentId || !ffConferenceScheduling) return
    void listParentConferenceTeachers(selectedStudentId)
      .then((t) => {
        setTeachers(t)
        if (t.length > 0 && !selectedTeacherId) setSelectedTeacherId(t[0].teacherId)
      })
      .catch((e) => setLoadError(e instanceof Error ? e.message : 'Could not load teachers.'))
  }, [selectedStudentId, ffConferenceScheduling, selectedTeacherId])

  const loadSlots = useCallback(async () => {
    if (!selectedTeacherId || !conferenceDate) return
    try {
      const data = await listConferenceSlots(selectedTeacherId, conferenceDate)
      setSlots(data.slots)
      setLoadError(null)
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : 'Could not load slots.')
    }
  }, [selectedTeacherId, conferenceDate])

  useEffect(() => {
    void loadSlots()
  }, [loadSlots])

  const selectedTeacher = useMemo(
    () => teachers.find((t) => t.teacherId === selectedTeacherId),
    [teachers, selectedTeacherId],
  )

  const myBookedSlotId = useMemo(() => {
    const booked = slots.find((s) => s.status === 'booked' && s.bookedForChild === selectedStudentId)
    return booked?.id ?? null
  }, [slots, selectedStudentId])

  const handleBook = useCallback(
    async (slot: ConferenceSlot) => {
      if (!selectedStudentId) return
      setBooking(true)
      setActionError(null)
      setSuccess(null)
      try {
        await bookConferenceSlot(slot.id, selectedStudentId)
        setSuccess('Conference booked! Check your email for a calendar invite.')
        await loadSlots()
      } catch (e) {
        setActionError(e instanceof Error ? e.message : 'Could not book slot.')
      } finally {
        setBooking(false)
      }
    },
    [selectedStudentId, loadSlots],
  )

  const handleCancel = useCallback(
    async (slot: ConferenceSlot) => {
      setBooking(true)
      setActionError(null)
      setSuccess(null)
      try {
        await cancelConferenceBooking(slot.id)
        setSuccess('Booking cancelled.')
        await loadSlots()
      } catch (e) {
        setActionError(e instanceof Error ? e.message : 'Could not cancel booking.')
      } finally {
        setBooking(false)
      }
    },
    [loadSlots],
  )

  if (!ffConferenceScheduling) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-10">
        <p className="text-sm text-neutral-600 dark:text-neutral-400">
          Conference scheduling is not enabled on this platform.
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-6 px-4 py-8 md:px-8">
      <header className="flex flex-col gap-2 border-b border-slate-200 pb-6 dark:border-neutral-800">
        <div className="flex items-center gap-2 text-sm font-medium text-indigo-700 dark:text-indigo-300">
          <CalendarHeart className="h-4 w-4" aria-hidden />
          Parent portal
        </div>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
          Book parent-teacher conferences
        </h1>
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Choose a time with each of your child&apos;s teachers.{' '}
          <Link to="/parent" className="text-indigo-600 underline dark:text-indigo-400">
            Back to family dashboard
          </Link>
        </p>
      </header>

      {loadError && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-100">
          {loadError}
        </div>
      )}
      {actionError && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-100">
          {actionError}
        </div>
      )}
      {success && (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-100">
          {success}
        </div>
      )}

      {children && children.length > 0 && (
        <>
          <div role="listbox" aria-label="Select student" className="flex flex-wrap gap-2">
            {children.map((c) => {
              const active = c.studentUserId === selectedStudentId
              return (
                <button
                  key={c.studentUserId}
                  type="button"
                  role="option"
                  aria-selected={active}
                  onClick={() => setParams({ student: c.studentUserId })}
                  className={`rounded-full px-4 py-2 text-sm font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 ${
                    active
                      ? 'bg-indigo-600 text-white'
                      : 'bg-neutral-100 text-neutral-800 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-100'
                  }`}
                >
                  {childLabel(c)}
                </button>
              )
            })}
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="flex flex-col gap-1 text-sm">
              <span className="font-medium">Teacher</span>
              <select
                value={selectedTeacherId}
                onChange={(e) => setSelectedTeacherId(e.target.value)}
                className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
              >
                {teachers.map((t) => (
                  <option key={t.teacherId} value={t.teacherId}>
                    {teacherLabel(t)}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1 text-sm">
              <span className="font-medium">Conference date</span>
              <input
                type="date"
                value={conferenceDate}
                onChange={(e) => setConferenceDate(e.target.value)}
                className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
              />
            </label>
          </div>

          {selectedTeacher && conferenceDate && (
            <section aria-labelledby="slot-picker-heading">
              <h2 id="slot-picker-heading" className="mb-3 text-lg font-semibold text-neutral-900 dark:text-neutral-50">
                Slots with {teacherLabel(selectedTeacher)}
              </h2>
              <SlotPicker
                slots={slots}
                teacherName={teacherLabel(selectedTeacher)}
                myBookedSlotId={myBookedSlotId}
                onBook={(slot) => void handleBook(slot)}
                onCancel={(slot) => void handleCancel(slot)}
                booking={booking}
              />
            </section>
          )}
        </>
      )}
    </div>
  )
}

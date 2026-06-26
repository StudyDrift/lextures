import { authorizedFetch } from './api'
import { formatDateTime } from './format'
import { readApiErrorMessage } from './errors'

export type ConferenceSlotStatus = 'open' | 'booked' | 'cancelled'

export type ConferenceAvailability = {
  id: string
  teacherId: string
  schoolId: string
  date: string
  slotDuration: number
  gapDuration: number
  windowStart: string
  windowEnd: string
  location?: string | null
  videoLink?: string | null
  createdAt: string
}

export type ConferenceSlot = {
  id: string
  availabilityId: string
  startAt: string
  endAt: string
  status: ConferenceSlotStatus
  bookedByParent?: string | null
  bookedForChild?: string | null
  bookedAt?: string | null
}

export type ConferenceTeacher = {
  teacherId: string
  displayName?: string | null
}

export type ScheduleEntry = ConferenceSlot & {
  teacherId: string
  teacherDisplayName?: string | null
  location?: string | null
  videoLink?: string | null
  childDisplayName?: string | null
}

export type CreateAvailabilityInput = {
  schoolId: string
  date: string
  windowStart: string
  windowEnd: string
  slotDuration: number
  gapDuration?: number
  location?: string | null
  videoLink?: string | null
}

export function formatConferenceSlotTime(slot: ConferenceSlot): string {
  return formatDateTime(slot.startAt)
}

export async function createConferenceAvailability(
  teacherId: string,
  input: CreateAvailabilityInput,
): Promise<{ availability: ConferenceAvailability; slots: ConferenceSlot[] }> {
  const res = await authorizedFetch(`/api/v1/teachers/${teacherId}/conference-availability`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  const raw = (await res.json()) as {
    availability?: ConferenceAvailability
    slots?: ConferenceSlot[]
  }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return { availability: raw.availability!, slots: raw.slots ?? [] }
}

export async function listConferenceSlots(
  teacherId: string,
  date: string,
): Promise<{ availability: ConferenceAvailability | null; slots: ConferenceSlot[] }> {
  const res = await authorizedFetch(
    `/api/v1/teachers/${teacherId}/conference-slots?date=${encodeURIComponent(date)}`,
  )
  const raw = (await res.json()) as {
    availability?: ConferenceAvailability | null
    slots?: ConferenceSlot[]
  }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return { availability: raw.availability ?? null, slots: raw.slots ?? [] }
}

export async function listParentConferenceTeachers(studentId: string): Promise<ConferenceTeacher[]> {
  const res = await authorizedFetch(
    `/api/v1/parent/conference-teachers?studentId=${encodeURIComponent(studentId)}`,
  )
  const raw = (await res.json()) as { teachers?: ConferenceTeacher[] }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw.teachers ?? []
}

export async function bookConferenceSlot(slotId: string, studentId: string): Promise<ConferenceSlot> {
  const res = await authorizedFetch(`/api/v1/conference-slots/${slotId}/book`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ studentId }),
  })
  const raw = (await res.json()) as { slot?: ConferenceSlot }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw.slot!
}

export async function cancelConferenceBooking(slotId: string): Promise<ConferenceSlot> {
  const res = await authorizedFetch(`/api/v1/conference-slots/${slotId}/book`, { method: 'DELETE' })
  const raw = (await res.json()) as { slot?: ConferenceSlot }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw.slot!
}

export async function listSchoolConferenceSchedule(
  orgUnitId: string,
  date: string,
): Promise<ScheduleEntry[]> {
  const res = await authorizedFetch(
    `/api/v1/admin/org-units/${orgUnitId}/conference-schedule?date=${encodeURIComponent(date)}`,
  )
  const raw = (await res.json()) as { schedule?: ScheduleEntry[] }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw.schedule ?? []
}

export async function fetchMyUserId(): Promise<string> {
  const res = await authorizedFetch('/api/v1/me')
  const raw = (await res.json()) as { id?: string; userId?: string }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw.id ?? raw.userId ?? ''
}

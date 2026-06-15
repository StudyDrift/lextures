import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type AccommodationType =
  | 'extended_time_1_5x'
  | 'extended_time_2x'
  | 'separate_testing'
  | 'alternate_format'
  | 'screen_reader'
  | 'speech_to_text'
  | 'reduced_distraction'
  | 'other'

export const ACCOMMODATION_TYPES: { value: AccommodationType; label: string; description: string }[] = [
  { value: 'extended_time_1_5x', label: '1.5x Extended Time', description: 'Quiz time limits multiplied by 1.5.' },
  { value: 'extended_time_2x', label: '2.0x Extended Time', description: 'Quiz time limits doubled.' },
  { value: 'separate_testing', label: 'Separate Testing', description: 'Flags a separate testing environment.' },
  { value: 'alternate_format', label: 'Alternate Format', description: 'Alternate-format materials (e.g. braille, large print).' },
  { value: 'screen_reader', label: 'Screen Reader', description: 'Enables text-to-speech / screen reader support.' },
  { value: 'speech_to_text', label: 'Speech-to-Text', description: 'Enables speech-to-text input.' },
  { value: 'reduced_distraction', label: 'Reduced Distraction', description: 'Reduced-distraction quiz interface.' },
  { value: 'other', label: 'Other', description: 'Additional accommodation (no automatic enforcement).' },
]

export function accommodationLabel(type: string): string {
  return ACCOMMODATION_TYPES.find((t) => t.value === type)?.label ?? type
}

export type AccommodationProfile = {
  id: string
  studentId: string
  accommodations: AccommodationType[]
  customParams: Record<string, unknown>
  effectiveFrom: string
  effectiveUntil?: string
  labels: string[]
  isActive: boolean
  notifiedAt?: string
  createdAt: string
}

export type AffectedCourse = {
  courseId: string
  courseCode: string
  title: string
}

export type CreateProfileInput = {
  studentId: string
  accommodations: AccommodationType[]
  customParams?: Record<string, unknown>
  effectiveFrom?: string
  effectiveUntil?: string
}

// --- coordinator / admin ---

export async function fetchAccommodationProfiles(): Promise<AccommodationProfile[]> {
  const res = await authorizedFetch('/api/v1/accessibility/profiles')
  if (!res.ok) {
    throw new Error('Could not load accommodation profiles.')
  }
  const data = (await res.json()) as { profiles?: AccommodationProfile[] }
  return data.profiles ?? []
}

export async function createAccommodationProfile(input: CreateProfileInput): Promise<AccommodationProfile> {
  const res = await authorizedFetch('/api/v1/accessibility/profiles', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not create accommodation profile.')
  }
  const data = (await res.json()) as { profile?: AccommodationProfile }
  if (!data.profile) {
    throw new Error('Unexpected response from server.')
  }
  return data.profile
}

export async function updateAccommodationProfile(
  id: string,
  patch: Partial<CreateProfileInput> & { isActive?: boolean },
): Promise<AccommodationProfile> {
  const res = await authorizedFetch(`/api/v1/accessibility/profiles/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not update accommodation profile.')
  }
  const data = (await res.json()) as { profile?: AccommodationProfile }
  if (!data.profile) {
    throw new Error('Unexpected response from server.')
  }
  return data.profile
}

export async function notifyInstructors(
  id: string,
): Promise<{ notifiedInstructorCount: number; letter: string }> {
  const res = await authorizedFetch(`/api/v1/accessibility/profiles/${id}/notify-instructors`, {
    method: 'POST',
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not notify instructors.')
  }
  return (await res.json()) as { notifiedInstructorCount: number; letter: string }
}

// --- student ---

export async function fetchMyAccommodations(): Promise<{
  profiles: AccommodationProfile[]
  affectedCourses: AffectedCourse[]
}> {
  const res = await authorizedFetch('/api/v1/me/accommodation-profiles')
  if (!res.ok) {
    throw new Error('Could not load your accommodations.')
  }
  const data = (await res.json()) as {
    profiles?: AccommodationProfile[]
    affectedCourses?: AffectedCourse[]
  }
  return { profiles: data.profiles ?? [], affectedCourses: data.affectedCourses ?? [] }
}

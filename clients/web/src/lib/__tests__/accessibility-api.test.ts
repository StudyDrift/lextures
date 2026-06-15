import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  accommodationLabel,
  createAccommodationProfile,
  fetchAccommodationProfiles,
  fetchMyAccommodations,
  notifyInstructors,
  updateAccommodationProfile,
} from '../accessibility-api'

vi.mock('../api', () => ({
  authorizedFetch: vi.fn(),
}))

vi.mock('../errors', () => ({
  readApiErrorMessage: (raw: Record<string, unknown>) => (raw?.message as string) ?? '',
}))

import { authorizedFetch } from '../api'

const mockFetch = authorizedFetch as unknown as ReturnType<typeof vi.fn>

describe('accommodationLabel', () => {
  it('maps known types to friendly labels', () => {
    expect(accommodationLabel('extended_time_1_5x')).toBe('1.5x Extended Time')
    expect(accommodationLabel('bogus')).toBe('bogus')
  })
})

describe('fetchAccommodationProfiles', () => {
  beforeEach(() => mockFetch.mockReset())

  it('returns the profiles array', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ profiles: [{ id: 'p1', labels: ['1.5x Extended Time'] }] }),
    })
    const res = await fetchAccommodationProfiles()
    expect(res).toHaveLength(1)
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/accessibility/profiles')
  })

  it('throws on error response', async () => {
    mockFetch.mockResolvedValue({ ok: false, json: async () => ({}) })
    await expect(fetchAccommodationProfiles()).rejects.toThrow()
  })
})

describe('createAccommodationProfile', () => {
  beforeEach(() => mockFetch.mockReset())

  it('posts the profile payload', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ profile: { id: 'p1', studentId: 's1' } }),
    })
    const prof = await createAccommodationProfile({
      studentId: 's1',
      accommodations: ['extended_time_1_5x'],
    })
    expect(prof.id).toBe('p1')
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/accessibility/profiles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ studentId: 's1', accommodations: ['extended_time_1_5x'] }),
    })
  })

  it('surfaces server error messages', async () => {
    mockFetch.mockResolvedValue({ ok: false, json: async () => ({ message: 'bad input' }) })
    await expect(
      createAccommodationProfile({ studentId: 's1', accommodations: [] }),
    ).rejects.toThrow('bad input')
  })
})

describe('updateAccommodationProfile', () => {
  beforeEach(() => mockFetch.mockReset())

  it('patches isActive', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ profile: { id: 'p1', isActive: false } }),
    })
    const prof = await updateAccommodationProfile('p1', { isActive: false })
    expect(prof.isActive).toBe(false)
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/accessibility/profiles/p1', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isActive: false }),
    })
  })
})

describe('notifyInstructors', () => {
  beforeEach(() => mockFetch.mockReset())

  it('returns the notify result', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ notifiedInstructorCount: 2, letter: 'Dear Instructor' }),
    })
    const res = await notifyInstructors('p1')
    expect(res.notifiedInstructorCount).toBe(2)
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/accessibility/profiles/p1/notify-instructors', {
      method: 'POST',
    })
  })
})

describe('fetchMyAccommodations', () => {
  beforeEach(() => mockFetch.mockReset())

  it('returns profiles and affected courses', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        profiles: [{ id: 'p1' }],
        affectedCourses: [{ courseId: 'c1', courseCode: 'C-ABC123', title: 'Bio' }],
      }),
    })
    const res = await fetchMyAccommodations()
    expect(res.profiles).toHaveLength(1)
    expect(res.affectedCourses[0].courseCode).toBe('C-ABC123')
  })
})

import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CoursePublic } from '../courses-api'
import { useViewerEnrollmentRolesState } from '../use-viewer-enrollment-roles'

const fetchCourse = vi.hoisted(() => vi.fn())

vi.mock('../courses-api', () => ({ fetchCourse }))

describe('useViewerEnrollmentRolesState', () => {
  beforeEach(() => {
    fetchCourse.mockReset()
  })

  it('reports an error instead of staying blank when the course request fails, then recovers on retry', async () => {
    fetchCourse.mockRejectedValueOnce(new Error('The request timed out.'))
    fetchCourse.mockResolvedValueOnce({ courseCode: 'C-1', viewerEnrollmentRoles: ['student'] } as CoursePublic)

    const { result } = renderHook(() => useViewerEnrollmentRolesState('C-1'))
    await waitFor(() => expect(result.current.error).toBe('The request timed out.'))
    expect(result.current.roles).toBeNull()

    act(() => result.current.retry())
    await waitFor(() => expect(result.current.roles).toEqual(['student']))
    expect(result.current.error).toBeNull()
    expect(fetchCourse).toHaveBeenCalledTimes(2)
  })
})

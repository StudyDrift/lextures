import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  courseFileContentPathname,
  fetchCourseFileImageBlob,
  needsAuthenticatedCourseImageSrc,
  resolveAuthorizedFetchPath,
} from '../course-file-image'

vi.mock('../api', () => ({
  apiUrl: (path: string) => `http://api.test${path}`,
  authorizedFetch: vi.fn(),
}))

vi.mock('../auth', () => ({
  getBearerToken: vi.fn(() => null),
}))

import { authorizedFetch } from '../api'
import { getBearerToken } from '../auth'

describe('courseFileContentPathname', () => {
  const courseFile =
    '/api/v1/courses/C-WLCOME/course-files/6528692e-282a-4a23-94f7-d12c52082e59/content'

  it('strips catalog thumbnail query params', () => {
    expect(
      courseFileContentPathname(`${courseFile}?w=640&h=320&q=85`),
    ).toBe(courseFile)
  })

  it('strips display-size hash fragments', () => {
    expect(courseFileContentPathname(`${courseFile}#w=1200&h=600`)).toBe(courseFile)
  })
})

describe('needsAuthenticatedCourseImageSrc', () => {
  const courseFile =
    '/api/v1/courses/C-WLCOME/course-files/6528692e-282a-4a23-94f7-d12c52082e59/content'

  it('requires auth for thumbnail URLs with resize query params', () => {
    expect(needsAuthenticatedCourseImageSrc(`${courseFile}?w=640&h=320&q=85`)).toBe(true)
  })

  it('does not require auth for static assets', () => {
    expect(needsAuthenticatedCourseImageSrc('/course-card-hero.png')).toBe(false)
  })
})

describe('resolveAuthorizedFetchPath', () => {
  it('preserves resize query params for authorized fetch', () => {
    const src =
      '/api/v1/courses/C-WLCOME/course-files/6528692e-282a-4a23-94f7-d12c52082e59/content?w=640&h=320&q=85'
    expect(resolveAuthorizedFetchPath(src)).toBe(src)
  })
})

describe('fetchCourseFileImageBlob', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.mocked(authorizedFetch).mockReset()
    vi.mocked(getBearerToken).mockReturnValue(null)
  })

  it('returns the public blob without auth when the storefront hero is readable', async () => {
    const blob = new Blob(['hero'], { type: 'image/jpeg' })
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, blob: () => Promise.resolve(blob) }),
    )

    const out = await fetchCourseFileImageBlob(
      '/api/v1/courses/C-AIESS1/course-files/75782c7e-8410-4ac5-8f88-61a3290b938e/content',
    )

    expect(out).toBe(blob)
    expect(authorizedFetch).not.toHaveBeenCalled()
    expect(fetch).toHaveBeenCalledWith(
      'http://api.test/api/v1/courses/C-AIESS1/course-files/75782c7e-8410-4ac5-8f88-61a3290b938e/content',
      { headers: { Prefer: 'return=representation' } },
    )
  })

  it('falls back to authorizedFetch after a 401', async () => {
    const blob = new Blob(['private'], { type: 'image/png' })
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 401, blob: () => Promise.resolve(new Blob()) }),
    )
    vi.mocked(authorizedFetch).mockResolvedValue({
      ok: true,
      blob: () => Promise.resolve(blob),
    } as Response)

    const out = await fetchCourseFileImageBlob(
      '/api/v1/courses/C-TEST/course-files/00000000-0000-4000-8000-000000000099/content',
    )

    expect(out).toBe(blob)
    expect(authorizedFetch).toHaveBeenCalledWith(
      '/api/v1/courses/C-TEST/course-files/00000000-0000-4000-8000-000000000099/content',
      { headers: { Prefer: 'return=representation' } },
    )
  })

  it('skips the public probe when a bearer token is already present', async () => {
    const blob = new Blob(['private'], { type: 'image/png' })
    vi.mocked(getBearerToken).mockReturnValue('tok')
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    vi.mocked(authorizedFetch).mockResolvedValue({
      ok: true,
      blob: () => Promise.resolve(blob),
    } as Response)

    const out = await fetchCourseFileImageBlob(
      '/api/v1/courses/C-TEST/course-files/00000000-0000-4000-8000-000000000099/content',
    )

    expect(out).toBe(blob)
    expect(fetchMock).not.toHaveBeenCalled()
    expect(authorizedFetch).toHaveBeenCalledWith(
      '/api/v1/courses/C-TEST/course-files/00000000-0000-4000-8000-000000000099/content',
      { headers: { Prefer: 'return=representation' } },
    )
  })
})
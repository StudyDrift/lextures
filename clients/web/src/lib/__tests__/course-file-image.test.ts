import { describe, expect, it } from 'vitest'
import {
  courseFileContentPathname,
  needsAuthenticatedCourseImageSrc,
  resolveAuthorizedFetchPath,
} from '../course-file-image'

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
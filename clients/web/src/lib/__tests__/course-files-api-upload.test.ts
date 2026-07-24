import { beforeEach, describe, expect, it, vi } from 'vitest'

const authorizedFetch = vi.fn()

vi.mock('../api', () => ({
  authorizedFetch: (...args: unknown[]) => authorizedFetch(...args),
}))

vi.mock('../errors', () => ({
  readApiErrorMessage: () => 'api error',
}))

import { uploadManagedCourseFile } from '../course-files-api'

describe('uploadManagedCourseFile', () => {
  beforeEach(() => {
    authorizedFetch.mockReset()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, status: 200 }),
    )
  })

  it('returns the FileItem when the server accepts the body directly', async () => {
    const item = {
      id: 'f1',
      courseId: 'c1',
      folderId: null,
      storageKey: 'k',
      originalFilename: 'a.png',
      displayName: 'a.png',
      mimeType: 'image/png',
      byteSize: 3,
      uploadedBy: null,
      createdAt: '',
      updatedAt: '',
    }
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => item,
    })

    const file = new File(['abc'], 'a.png', { type: 'image/png' })
    const result = await uploadManagedCourseFile('C-1', file, null)
    expect(result).toEqual(item)
    expect(authorizedFetch).toHaveBeenCalledTimes(1)
  })

  it('presigns, PUTs, and confirms when a presigned URL is returned', async () => {
    authorizedFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          objectKey: 'managed-files/C-1/x.png',
          presignedPutUrl: 'https://storage.example/put',
          expiresAt: '2099-01-01T00:00:00Z',
          courseId: 'c1',
          folderId: null,
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          id: 'f2',
          courseId: 'c1',
          folderId: null,
          storageKey: 'managed-files/C-1/x.png',
          originalFilename: 'a.png',
          displayName: 'a.png',
          mimeType: 'image/png',
          byteSize: 3,
          uploadedBy: null,
          createdAt: '',
          updatedAt: '',
        }),
      })

    const file = new File(['abc'], 'a.png', { type: 'image/png' })
    const result = await uploadManagedCourseFile('C-1', file)
    expect(result.id).toBe('f2')
    expect(globalThis.fetch).toHaveBeenCalledWith(
      'https://storage.example/put',
      expect.objectContaining({ method: 'PUT' }),
    )
    expect(authorizedFetch).toHaveBeenCalledTimes(2)
  })
})

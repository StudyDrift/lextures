import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MarkdownImageUploadModal } from '../markdown-image-upload-modal'

const listCourseFiles = vi.fn()
const uploadManagedCourseFile = vi.fn()
const getFileContentUrl = vi.fn(
  (courseCode: string, itemId: string) =>
    `/api/v1/courses/${courseCode}/files/items/${itemId}/content`,
)

vi.mock('../../../../lib/course-files-api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../../lib/course-files-api')>()
  return {
    ...actual,
    listCourseFiles: (...args: unknown[]) => listCourseFiles(...args),
    uploadManagedCourseFile: (...args: unknown[]) => uploadManagedCourseFile(...args),
    getFileContentUrl: (...args: unknown[]) =>
      getFileContentUrl(...(args as [string, string])),
  }
})

describe('MarkdownImageUploadModal', () => {
  beforeEach(() => {
    listCourseFiles.mockReset()
    uploadManagedCourseFile.mockReset()
    listCourseFiles.mockResolvedValue({
      folderId: null,
      breadcrumbs: [],
      folders: [{ id: 'folder-1', courseId: 'c', parentId: null, name: 'Handouts', createdBy: null, createdAt: '', updatedAt: '' }],
      files: [
        {
          id: 'file-1',
          courseId: 'c',
          folderId: null,
          storageKey: 'k1',
          originalFilename: 'diagram.png',
          displayName: 'diagram.png',
          mimeType: 'image/png',
          byteSize: 2048,
          uploadedBy: null,
          createdAt: '',
          updatedAt: '',
        },
        {
          id: 'file-2',
          courseId: 'c',
          folderId: null,
          storageKey: 'k2',
          originalFilename: 'syllabus.pdf',
          displayName: 'syllabus.pdf',
          mimeType: 'application/pdf',
          byteSize: 4096,
          uploadedBy: null,
          createdAt: '',
          updatedAt: '',
        },
      ],
    })
  })

  it('does not render when closed', () => {
    render(
      <MarkdownImageUploadModal open={false} onClose={vi.fn()} onInsert={vi.fn()} courseCode="C-1" />,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('lists course files and inserts the selected image on Insert', async () => {
    const user = userEvent.setup()
    const onInsert = vi.fn().mockResolvedValue(undefined)
    const onClose = vi.fn()

    render(
      <MarkdownImageUploadModal
        open
        courseCode="C-1"
        onClose={onClose}
        onInsert={onInsert}
      />,
    )

    await waitFor(() => expect(screen.getByText('diagram.png')).toBeInTheDocument())
    expect(screen.getByRole('heading', { name: /insert file or image/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Insert' })).toBeDisabled()

    await user.click(screen.getByRole('option', { name: /diagram\.png/i }))
    const insertBtn = screen.getByRole('button', { name: 'Insert' })
    expect(insertBtn).toBeEnabled()
    await user.click(insertBtn)

    await waitFor(() =>
      expect(onInsert).toHaveBeenCalledWith([
        {
          contentPath: '/api/v1/courses/C-1/files/items/file-1/content',
          displayName: 'diagram.png',
          mimeType: 'image/png',
        },
      ]),
    )
    expect(onClose).toHaveBeenCalled()
  })

  it('stages a dropped upload and uploads it on Insert', async () => {
    const user = userEvent.setup()
    const onInsert = vi.fn().mockResolvedValue(undefined)
    uploadManagedCourseFile.mockResolvedValue({
      id: 'new-1',
      courseId: 'c',
      folderId: null,
      storageKey: 'k-new',
      originalFilename: 'photo.jpg',
      displayName: 'photo.jpg',
      mimeType: 'image/jpeg',
      byteSize: 100,
      uploadedBy: null,
      createdAt: '',
      updatedAt: '',
    })

    render(
      <MarkdownImageUploadModal
        open
        courseCode="C-1"
        onClose={vi.fn()}
        onInsert={onInsert}
      />,
    )

    await waitFor(() => expect(listCourseFiles).toHaveBeenCalled())

    const file = new File(['img'], 'photo.jpg', { type: 'image/jpeg' })
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    expect(input).toBeTruthy()
    await user.upload(input, file)

    expect(await screen.findByText('photo.jpg')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Insert' }))

    await waitFor(() => expect(uploadManagedCourseFile).toHaveBeenCalledWith('C-1', file, null))
    expect(onInsert).toHaveBeenCalledWith([
      {
        contentPath: '/api/v1/courses/C-1/files/items/new-1/content',
        displayName: 'photo.jpg',
        mimeType: 'image/jpeg',
      },
    ])
  })

  it('uses uploadFile fallback when courseCode is absent', async () => {
    const user = userEvent.setup()
    const uploadFile = vi.fn().mockResolvedValue('data:image/png;base64,abc')
    const onInsert = vi.fn().mockResolvedValue(undefined)
    const file = new File(['x'], 'local.png', { type: 'image/png' })

    render(
      <MarkdownImageUploadModal
        open
        onClose={vi.fn()}
        onInsert={onInsert}
        uploadFile={uploadFile}
        initialFiles={[file]}
      />,
    )

    expect(screen.queryByRole('listbox', { name: /course files/i })).not.toBeInTheDocument()
    const staged = screen.getByLabelText('Files to upload')
    expect(within(staged).getByText('local.png')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Insert' }))
    await waitFor(() => expect(uploadFile).toHaveBeenCalledWith(file))
    expect(onInsert).toHaveBeenCalledWith([
      {
        contentPath: 'data:image/png;base64,abc',
        displayName: 'local.png',
        mimeType: 'image/png',
      },
    ])
  })

  it('closes on Escape when not busy', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    listCourseFiles.mockResolvedValue({ folderId: null, breadcrumbs: [], folders: [], files: [] })

    render(
      <MarkdownImageUploadModal open courseCode="C-1" onClose={onClose} onInsert={vi.fn()} />,
    )
    await waitFor(() => expect(listCourseFiles).toHaveBeenCalled())
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
})

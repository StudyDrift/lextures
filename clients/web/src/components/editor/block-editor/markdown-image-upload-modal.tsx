import {
  ChevronRight,
  File as FileIcon,
  Folder,
  Image as ImageIcon,
  Upload,
  X,
} from 'lucide-react'
import { useCallback, useEffect, useId, useRef, useState } from 'react'
import {
  formatBytes,
  getFileContentUrl,
  listCourseFiles,
  uploadManagedCourseFile,
  type FileFolder,
  type FileItem,
  type FolderBreadcrumb,
} from '../../../lib/course-files-api'

const ACCEPT =
  'image/png,image/jpeg,image/jpg,image/gif,image/webp,application/pdf,text/plain,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.zip'

export type CourseFileInsertItem = {
  contentPath: string
  displayName: string
  mimeType: string
}

export type MarkdownImageUploadModalProps = {
  open: boolean
  onClose: () => void
  /** When set, browse/upload via the course files manager. */
  courseCode?: string
  /**
   * Fallback uploader when `courseCode` is unset (e.g. global notebook data URLs).
   * Returns a content path or data URL for each file.
   */
  uploadFile?: (file: File) => Promise<string>
  /** Files staged before the dialog opens (e.g. dropped on the toolbar button). */
  initialFiles?: File[]
  onInsert: (items: CourseFileInsertItem[]) => void | Promise<void>
}

type StagedLocal = {
  key: string
  file: File
}

function isImageMime(mime: string): boolean {
  return mime.toLowerCase().startsWith('image/')
}

function fileIconForMime(mime: string) {
  if (isImageMime(mime)) return ImageIcon
  return FileIcon
}

export function MarkdownImageUploadModal({
  open,
  onClose,
  courseCode,
  uploadFile,
  initialFiles,
  onInsert,
}: MarkdownImageUploadModalProps) {
  const titleId = useId()
  const inputId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragOver, setDragOver] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [folderId, setFolderId] = useState<string | undefined>(undefined)
  const [breadcrumbs, setBreadcrumbs] = useState<FolderBreadcrumb[]>([])
  const [folders, setFolders] = useState<FileFolder[]>([])
  const [files, setFiles] = useState<FileItem[]>([])
  const [listLoading, setListLoading] = useState(false)
  const [listError, setListError] = useState<string | null>(null)

  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set())
  const [staged, setStaged] = useState<StagedLocal[]>([])

  const canBrowse = Boolean(courseCode)
  const canUpload = Boolean(courseCode) || Boolean(uploadFile)

  const reset = useCallback(() => {
    setDragOver(false)
    setBusy(false)
    setError(null)
    setFolderId(undefined)
    setBreadcrumbs([])
    setFolders([])
    setFiles([])
    setListLoading(false)
    setListError(null)
    setSelectedIds(new Set())
    setStaged([])
  }, [])

  const handleClose = useCallback(() => {
    if (busy) return
    reset()
    onClose()
  }, [busy, onClose, reset])

  const loadFolder = useCallback(
    async (nextFolderId?: string) => {
      if (!courseCode) return
      setListLoading(true)
      setListError(null)
      try {
        const contents = await listCourseFiles(courseCode, nextFolderId)
        setFolderId(contents.folderId ?? undefined)
        setBreadcrumbs(contents.breadcrumbs ?? [])
        setFolders(contents.folders)
        setFiles(contents.files)
      } catch (e) {
        setListError(e instanceof Error ? e.message : 'Could not load course files.')
        setFolders([])
        setFiles([])
      } finally {
        setListLoading(false)
      }
    },
    [courseCode],
  )

  useEffect(() => {
    if (!open) return
    reset()
    if (initialFiles?.length) {
      setStaged(
        initialFiles.map((file, i) => ({
          key: `staged-${file.name}-${file.size}-${file.lastModified}-${i}`,
          file,
        })),
      )
    }
    if (courseCode) void loadFolder(undefined)
    const t = window.setTimeout(() => inputRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [open, reset, courseCode, loadFolder, initialFiles])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !busy) {
        e.preventDefault()
        handleClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, busy, handleClose])

  const stageFiles = useCallback((incoming: File[]) => {
    if (!incoming.length) return
    setError(null)
    setStaged((prev) => {
      const next = [...prev]
      for (const file of incoming) {
        const key = `staged-${file.name}-${file.size}-${file.lastModified}-${next.length}`
        next.push({ key, file })
      }
      return next
    })
  }, [])

  const toggleSelected = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const removeStaged = useCallback((key: string) => {
    setStaged((prev) => prev.filter((s) => s.key !== key))
  }, [])

  const onInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const list = e.target.files ? [...e.target.files] : []
      e.target.value = ''
      if (list.length) stageFiles(list)
    },
    [stageFiles],
  )

  const selectionCount = selectedIds.size + staged.length

  const handleInsert = useCallback(async () => {
    if (!selectionCount || busy) return
    setBusy(true)
    setError(null)
    try {
      const items: CourseFileInsertItem[] = []

      if (courseCode) {
        for (const file of files) {
          if (!selectedIds.has(file.id)) continue
          items.push({
            contentPath: getFileContentUrl(courseCode, file.id),
            displayName: file.displayName || file.originalFilename,
            mimeType: file.mimeType || 'application/octet-stream',
          })
        }
        for (const { file } of staged) {
          const uploaded = await uploadManagedCourseFile(courseCode, file, folderId ?? null)
          items.push({
            contentPath: getFileContentUrl(courseCode, uploaded.id),
            displayName: uploaded.displayName || uploaded.originalFilename,
            mimeType: uploaded.mimeType || file.type || 'application/octet-stream',
          })
        }
      } else if (uploadFile) {
        for (const { file } of staged) {
          const contentPath = await uploadFile(file)
          items.push({
            contentPath,
            displayName: file.name,
            mimeType: file.type || 'application/octet-stream',
          })
        }
      }

      if (!items.length) {
        setError('Select a course file or add an upload to insert.')
        setBusy(false)
        return
      }

      await onInsert(items)
      reset()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not insert file.')
      setBusy(false)
    }
  }, [
    busy,
    courseCode,
    files,
    folderId,
    onClose,
    onInsert,
    reset,
    selectedIds,
    selectionCount,
    staged,
    uploadFile,
  ])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[95] flex items-end justify-center bg-black/40 p-4 sm:items-center"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) handleClose()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="flex max-h-[min(40rem,calc(100vh-2rem))] w-full max-w-lg flex-col rounded-xl border border-slate-200 bg-white shadow-xl dark:border-neutral-600 dark:bg-neutral-900"
      >
        <div className="flex items-start justify-between gap-3 border-b border-slate-100 px-5 py-4 dark:border-neutral-800">
          <div>
            <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-neutral-100">
              Insert file or image
            </h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              {canBrowse
                ? 'Choose from course files or upload new ones, then click Insert.'
                : 'Upload a file or image, then click Insert.'}
            </p>
          </div>
          <button
            type="button"
            onClick={handleClose}
            disabled={busy}
            className="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 disabled:opacity-50 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
          {canBrowse ? (
            <div>
              <nav aria-label="Folder path" className="mb-2 flex flex-wrap items-center gap-1 text-xs">
                <button
                  type="button"
                  disabled={busy || listLoading}
                  onClick={() => void loadFolder(undefined)}
                  className="rounded px-1.5 py-0.5 font-medium text-indigo-600 hover:bg-indigo-50 disabled:opacity-50 dark:text-indigo-300 dark:hover:bg-indigo-950/40"
                >
                  Course files
                </button>
                {breadcrumbs.map((crumb) => (
                  <span key={crumb.id} className="flex items-center gap-1">
                    <ChevronRight className="h-3 w-3 text-slate-400" aria-hidden />
                    <button
                      type="button"
                      disabled={busy || listLoading}
                      onClick={() => void loadFolder(crumb.id)}
                      className="rounded px-1.5 py-0.5 font-medium text-indigo-600 hover:bg-indigo-50 disabled:opacity-50 dark:text-indigo-300 dark:hover:bg-indigo-950/40"
                    >
                      {crumb.name}
                    </button>
                  </span>
                ))}
              </nav>

              <div
                role="listbox"
                aria-label="Course files"
                aria-multiselectable="true"
                className="max-h-48 overflow-auto rounded-lg border border-slate-200 dark:border-neutral-700"
              >
                {listLoading ? (
                  <p className="px-3 py-4 text-sm text-slate-500 dark:text-neutral-400">Loading files…</p>
                ) : listError ? (
                  <p className="px-3 py-4 text-sm text-rose-600 dark:text-rose-400" role="alert">
                    {listError}
                  </p>
                ) : folders.length === 0 && files.length === 0 ? (
                  <p className="px-3 py-4 text-sm text-slate-500 dark:text-neutral-400">
                    This folder is empty. Upload a file below.
                  </p>
                ) : (
                  <ul className="divide-y divide-slate-100 dark:divide-neutral-800">
                    {folders.map((folder) => (
                      <li key={folder.id}>
                        <button
                          type="button"
                          disabled={busy}
                          onClick={() => void loadFolder(folder.id)}
                          className="flex w-full items-center gap-2 px-3 py-2 text-start text-sm text-slate-800 hover:bg-slate-50 disabled:opacity-50 dark:text-neutral-100 dark:hover:bg-neutral-800"
                        >
                          <Folder className="h-4 w-4 shrink-0 text-amber-500" aria-hidden />
                          <span className="truncate font-medium">{folder.name}</span>
                        </button>
                      </li>
                    ))}
                    {files.map((file) => {
                      const Icon = fileIconForMime(file.mimeType)
                      const selected = selectedIds.has(file.id)
                      return (
                        <li key={file.id}>
                          <button
                            type="button"
                            role="option"
                            aria-selected={selected}
                            disabled={busy}
                            onClick={() => toggleSelected(file.id)}
                            className={`flex w-full items-center gap-2 px-3 py-2 text-start text-sm disabled:opacity-50 ${
                              selected
                                ? 'bg-indigo-50 text-indigo-950 dark:bg-indigo-950/40 dark:text-indigo-50'
                                : 'text-slate-800 hover:bg-slate-50 dark:text-neutral-100 dark:hover:bg-neutral-800'
                            }`}
                          >
                            <Icon className="h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
                            <span className="min-w-0 flex-1 truncate font-medium">
                              {file.displayName || file.originalFilename}
                            </span>
                            <span className="shrink-0 text-xs text-slate-400 dark:text-neutral-500">
                              {formatBytes(file.byteSize)}
                            </span>
                          </button>
                        </li>
                      )
                    })}
                  </ul>
                )}
              </div>
            </div>
          ) : null}

          {canUpload ? (
            <div>
              <input
                ref={inputRef}
                id={inputId}
                type="file"
                accept={canBrowse ? ACCEPT : 'image/png,image/jpeg,image/jpg,image/gif,image/webp'}
                multiple
                className="sr-only"
                disabled={busy}
                onChange={onInputChange}
              />
              <label
                htmlFor={inputId}
                onDragEnter={(e) => {
                  e.preventDefault()
                  if (!busy) setDragOver(true)
                }}
                onDragOver={(e) => {
                  e.preventDefault()
                  if (!busy) {
                    e.dataTransfer.dropEffect = 'copy'
                    setDragOver(true)
                  }
                }}
                onDragLeave={(e) => {
                  e.preventDefault()
                  if (e.currentTarget.contains(e.relatedTarget as Node | null)) return
                  setDragOver(false)
                }}
                onDrop={(e) => {
                  e.preventDefault()
                  setDragOver(false)
                  if (busy) return
                  stageFiles([...e.dataTransfer.files])
                }}
                className={`flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-6 py-8 text-center transition-[background-color,color,border-color] ${
                  dragOver
                    ? 'border-indigo-400 bg-indigo-50/80 dark:border-indigo-500 dark:bg-indigo-950/30'
                    : 'border-slate-200 bg-slate-50/70 hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-950/50 dark:hover:border-neutral-600 dark:hover:bg-neutral-950'
                } ${busy ? 'pointer-events-none opacity-60' : ''}`}
              >
                <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-white shadow-sm dark:bg-neutral-800">
                  {busy ? (
                    <Upload className="h-5 w-5 motion-safe:animate-pulse text-indigo-500" aria-hidden />
                  ) : (
                    <ImageIcon className="h-5 w-5 text-indigo-500" aria-hidden />
                  )}
                </div>
                <span className="text-sm font-medium text-slate-800 dark:text-neutral-100">
                  {busy ? 'Inserting…' : 'Drop files here'}
                </span>
                <span className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                  {canBrowse
                    ? 'or click to browse (images, PDFs, and common documents)'
                    : 'or click to browse (PNG, JPEG, GIF, WebP)'}
                </span>
              </label>

              {staged.length > 0 ? (
                <ul className="mt-3 space-y-1" aria-label="Files to upload">
                  {staged.map(({ key, file }) => {
                    const Icon = fileIconForMime(file.type)
                    return (
                      <li
                        key={key}
                        className="flex items-center gap-2 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                      >
                        <Icon className="h-4 w-4 shrink-0 text-slate-500" aria-hidden />
                        <span className="min-w-0 flex-1 truncate text-slate-800 dark:text-neutral-100">
                          {file.name}
                        </span>
                        <span className="shrink-0 text-xs text-slate-400">{formatBytes(file.size)}</span>
                        <button
                          type="button"
                          disabled={busy}
                          onClick={() => removeStaged(key)}
                          className="rounded p-0.5 text-slate-400 hover:bg-slate-100 hover:text-slate-600 disabled:opacity-50 dark:hover:bg-neutral-800"
                          aria-label={`Remove ${file.name}`}
                        >
                          <X className="h-3.5 w-3.5" />
                        </button>
                      </li>
                    )
                  })}
                </ul>
              ) : null}
            </div>
          ) : null}

          {error ? (
            <p className="text-sm text-rose-600 dark:text-rose-400" role="alert">
              {error}
            </p>
          ) : null}
        </div>

        <div className="flex items-center justify-between gap-3 border-t border-slate-100 px-5 py-3 dark:border-neutral-800">
          <p className="text-xs text-slate-500 dark:text-neutral-400">
            {selectionCount === 0
              ? 'Nothing selected'
              : `${selectionCount} selected`}
          </p>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handleClose}
              disabled={busy}
              className="rounded-md px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => void handleInsert()}
              disabled={busy || selectionCount === 0}
              className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-indigo-500 dark:hover:bg-indigo-400"
            >
              {busy ? 'Inserting…' : 'Insert'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

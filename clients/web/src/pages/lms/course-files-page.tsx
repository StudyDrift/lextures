import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { formatAbsoluteShort } from '../../lib/format-datetime'
import { courseItemCreatePermission } from '../../lib/courses-api'
import { usePermissions } from '../../context/use-permissions'
import {
  listCourseFiles,
  createFolder,
  renameFolder,
  deleteFolder,
  initiateFileUpload,
  uploadToPresignedUrl,
  confirmFileUpload,
  renameFile,
  moveFile,
  deleteFile,
  getFileContentUrl,
  formatBytes,
  fileIconForMime,
  type FileFolder,
  type FileItem,
  type FolderContents,
} from '../../lib/course-files-api'
import { LmsPage } from './lms-page'

type ContextMenu =
  | { kind: 'folder'; item: FileFolder; x: number; y: number }
  | { kind: 'file'; item: FileItem; x: number; y: number }

export default function CourseFilesPage() {
  const { courseCode: rawCode } = useParams<{ courseCode: string }>()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''
  const [searchParams, setSearchParams] = useSearchParams()
  const folderId = searchParams.get('folder') ?? undefined

  const { allows, loading: permLoading } = usePermissions()
  const canManage = !permLoading && !!courseCode && allows(courseItemCreatePermission(courseCode))

  const [contents, setContents] = useState<FolderContents | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // breadcrumbs: list of {id, name} from root to current
  const [breadcrumbs, setBreadcrumbs] = useState<{ id: string; name: string }[]>([])

  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<string | null>(null)

  const [renamingFolder, setRenamingFolder] = useState<FileFolder | null>(null)
  const [renamingFile, setRenamingFile] = useState<FileItem | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [newFolderName, setNewFolderName] = useState('')
  const [showNewFolder, setShowNewFolder] = useState(false)

  const [contextMenu, setContextMenu] = useState<ContextMenu | null>(null)
  const [movingFile, setMovingFile] = useState<FileItem | null>(null)

  const fileInputRef = useRef<HTMLInputElement>(null)

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    setError(null)
    try {
      const result = await listCourseFiles(courseCode, folderId)
      setContents(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load files.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, folderId])

  useEffect(() => { void load() }, [load])

  // Keep breadcrumb trail in sync when folderId changes
  useEffect(() => {
    if (!folderId) {
      setBreadcrumbs([])
      return
    }
    // If we navigated into a subfolder, append; otherwise rebuild from contents
    if (contents?.folderId === folderId) return
    // On direct navigation (e.g. back/forward) we may not have parent info — just clear
    setBreadcrumbs([])
  }, [folderId, contents?.folderId])

  function navigateToFolder(id: string | null, name?: string) {
    if (!id) {
      setSearchParams({})
      setBreadcrumbs([])
    } else {
      setSearchParams({ folder: id })
      setBreadcrumbs(prev => {
        const existing = prev.findIndex(b => b.id === id)
        if (existing >= 0) return prev.slice(0, existing + 1)
        return [...prev, { id, name: name ?? 'Folder' }]
      })
    }
  }

  async function handleCreateFolder() {
    const name = newFolderName.trim()
    if (!name) return
    try {
      await createFolder(courseCode, name, folderId ?? null)
      setNewFolderName('')
      setShowNewFolder(false)
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Could not create folder.')
    }
  }

  async function handleRenameFolder() {
    if (!renamingFolder) return
    const name = renameValue.trim()
    if (!name) return
    try {
      await renameFolder(courseCode, renamingFolder.id, name)
      setRenamingFolder(null)
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Could not rename folder.')
    }
  }

  async function handleDeleteFolder(folder: FileFolder) {
    if (!confirm(`Delete folder "${folder.name}" and all its contents? This cannot be undone.`)) return
    try {
      await deleteFolder(courseCode, folder.id)
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Could not delete folder.')
    }
  }

  async function handleRenameFile() {
    if (!renamingFile) return
    const name = renameValue.trim()
    if (!name) return
    try {
      await renameFile(courseCode, renamingFile.id, name)
      setRenamingFile(null)
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Could not rename file.')
    }
  }

  async function handleDeleteFile(file: FileItem) {
    if (!confirm(`Delete "${file.displayName}"? This cannot be undone.`)) return
    try {
      await deleteFile(courseCode, file.id)
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Could not delete file.')
    }
  }

  async function handleMoveFile(targetFolderId: string | null) {
    if (!movingFile) return
    try {
      await moveFile(courseCode, movingFile.id, targetFolderId)
      setMovingFile(null)
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Could not move file.')
    }
  }

  async function handleFilesSelected(files: FileList | null) {
    if (!files || files.length === 0) return
    setUploading(true)
    for (let i = 0; i < files.length; i++) {
      const file = files[i]
      setUploadProgress(`Uploading ${file.name}…`)
      try {
        const result = await initiateFileUpload(courseCode, file, folderId ?? null)
        if ('presigned' in result) {
          await uploadToPresignedUrl(result.presigned.presignedPutUrl!, file)
          await confirmFileUpload(
            courseCode,
            result.presigned.objectKey,
            file.name,
            file.type || 'application/octet-stream',
            file.size,
            folderId ?? null,
          )
        }
      } catch (err) {
        alert(`Failed to upload ${file.name}: ${err instanceof Error ? err.message : 'Unknown error'}`)
      }
    }
    setUploading(false)
    setUploadProgress(null)
    void load()
  }

  function openContextMenu(e: React.MouseEvent, item: FileFolder | FileItem, kind: 'folder' | 'file') {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ kind, item: item as FileFolder & FileItem, x: e.clientX, y: e.clientY })
  }

  return (
    <LmsPage title="Files" fillHeight>
      {/* dismiss context menu on outside click */}
      {contextMenu && (
        <div
          className="fixed inset-0 z-10"
          onClick={() => setContextMenu(null)}
        />
      )}

      {/* Toolbar */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        {/* Breadcrumbs */}
        <nav className="flex min-w-0 flex-1 items-center gap-1 text-sm">
          <button
            onClick={() => navigateToFolder(null)}
            className="text-indigo-600 hover:underline dark:text-indigo-400"
          >
            Files
          </button>
          {breadcrumbs.map((b, i) => (
            <span key={b.id} className="flex items-center gap-1">
              <span className="text-slate-400">/</span>
              {i < breadcrumbs.length - 1 ? (
                <button
                  onClick={() => navigateToFolder(b.id, b.name)}
                  className="text-indigo-600 hover:underline dark:text-indigo-400"
                >
                  {b.name}
                </button>
              ) : (
                <span className="truncate font-medium text-slate-800 dark:text-neutral-200">{b.name}</span>
              )}
            </span>
          ))}
        </nav>

        {canManage && (
          <div className="flex shrink-0 items-center gap-2">
            <button
              type="button"
              onClick={() => setShowNewFolder(v => !v)}
              className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              📁 New folder
            </button>
            <button
              type="button"
              disabled={uploading}
              onClick={() => fileInputRef.current?.click()}
              className="inline-flex items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 disabled:opacity-60"
            >
              {uploading ? (uploadProgress ?? 'Uploading…') : '⬆ Upload'}
            </button>
            <input
              ref={fileInputRef}
              type="file"
              multiple
              className="hidden"
              onChange={e => void handleFilesSelected(e.target.files)}
            />
          </div>
        )}
      </div>

      {/* New folder inline form */}
      {showNewFolder && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900">
          <input
            autoFocus
            type="text"
            value={newFolderName}
            onChange={e => setNewFolderName(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') void handleCreateFolder(); if (e.key === 'Escape') { setShowNewFolder(false); setNewFolderName('') } }}
            placeholder="Folder name"
            className="min-w-0 flex-1 rounded border border-slate-300 bg-white px-2 py-1 text-sm text-slate-900 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
          />
          <button
            type="button"
            onClick={() => void handleCreateFolder()}
            className="rounded bg-indigo-600 px-3 py-1 text-sm font-medium text-white hover:bg-indigo-700"
          >
            Create
          </button>
          <button
            type="button"
            onClick={() => { setShowNewFolder(false); setNewFolderName('') }}
            className="rounded px-2 py-1 text-sm text-slate-500 hover:text-slate-700 dark:text-neutral-400"
          >
            Cancel
          </button>
        </div>
      )}

      {/* Content */}
      {loading ? (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-slate-500 dark:text-neutral-400">Loading…</span>
        </div>
      ) : error ? (
        <div className="rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-400">
          {error}
        </div>
      ) : !contents || (contents.folders.length === 0 && contents.files.length === 0) ? (
        <EmptyState canManage={canManage} onUpload={() => fileInputRef.current?.click()} />
      ) : (
        <div className="overflow-hidden rounded-xl border border-slate-200 bg-white dark:border-neutral-800 dark:bg-neutral-950">
          <table className="min-w-full divide-y divide-slate-100 text-sm dark:divide-neutral-800">
            <thead>
              <tr className="bg-slate-50 dark:bg-neutral-900">
                <th className="py-2.5 pl-4 pr-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">Name</th>
                <th className="px-3 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">Size</th>
                <th className="px-3 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">Modified</th>
                {canManage && <th className="py-2.5 pl-3 pr-4 text-right text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">Actions</th>}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50 dark:divide-neutral-800/60">
              {contents.folders.map(folder => (
                <FolderRow
                  key={folder.id}
                  folder={folder}
                  canManage={canManage}
                  renamingFolder={renamingFolder}
                  renameValue={renameValue}
                  setRenameValue={setRenameValue}
                  onNavigate={() => navigateToFolder(folder.id, folder.name)}
                  onRenameStart={() => { setRenamingFolder(folder); setRenameValue(folder.name) }}
                  onRenameSubmit={() => void handleRenameFolder()}
                  onRenameCancel={() => setRenamingFolder(null)}
                  onDelete={() => void handleDeleteFolder(folder)}
                  onContextMenu={e => openContextMenu(e, folder, 'folder')}
                />
              ))}
              {contents.files.map(file => (
                <FileRow
                  key={file.id}
                  file={file}
                  courseCode={courseCode}
                  canManage={canManage}
                  renamingFile={renamingFile}
                  renameValue={renameValue}
                  setRenameValue={setRenameValue}
                  onRenameStart={() => { setRenamingFile(file); setRenameValue(file.displayName) }}
                  onRenameSubmit={() => void handleRenameFile()}
                  onRenameCancel={() => setRenamingFile(null)}
                  onDelete={() => void handleDeleteFile(file)}
                  onMove={() => setMovingFile(file)}
                  onContextMenu={e => openContextMenu(e, file, 'file')}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Context menu */}
      {contextMenu && (
        <div
          className="fixed z-20 min-w-[160px] rounded-lg border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
          style={{ top: contextMenu.y, left: contextMenu.x }}
          onClick={e => e.stopPropagation()}
        >
          {contextMenu.kind === 'folder' && (
            <>
              <button
                className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                onClick={() => { navigateToFolder(contextMenu.item.id, (contextMenu.item as FileFolder).name); setContextMenu(null) }}
              >
                Open
              </button>
              {canManage && (
                <>
                  <button
                    className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                    onClick={() => { setRenamingFolder(contextMenu.item as FileFolder); setRenameValue((contextMenu.item as FileFolder).name); setContextMenu(null) }}
                  >
                    Rename
                  </button>
                  <hr className="my-1 border-slate-100 dark:border-neutral-700" />
                  <button
                    className="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30"
                    onClick={() => { void handleDeleteFolder(contextMenu.item as FileFolder); setContextMenu(null) }}
                  >
                    Delete
                  </button>
                </>
              )}
            </>
          )}
          {contextMenu.kind === 'file' && (
            <>
              <a
                href={getFileContentUrl(courseCode, contextMenu.item.id)}
                target="_blank"
                rel="noreferrer"
                className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                onClick={() => setContextMenu(null)}
              >
                Download
              </a>
              {canManage && (
                <>
                  <button
                    className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                    onClick={() => { setRenamingFile(contextMenu.item as FileItem); setRenameValue((contextMenu.item as FileItem).displayName); setContextMenu(null) }}
                  >
                    Rename
                  </button>
                  <button
                    className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                    onClick={() => { setMovingFile(contextMenu.item as FileItem); setContextMenu(null) }}
                  >
                    Move to…
                  </button>
                  <hr className="my-1 border-slate-100 dark:border-neutral-700" />
                  <button
                    className="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30"
                    onClick={() => { void handleDeleteFile(contextMenu.item as FileItem); setContextMenu(null) }}
                  >
                    Delete
                  </button>
                </>
              )}
            </>
          )}
        </div>
      )}

      {/* Move to folder picker */}
      {movingFile && contents && (
        <MovePicker
          file={movingFile}
          folders={contents.folders}
          currentFolderId={folderId ?? null}
          onMove={handleMoveFile}
          onCancel={() => setMovingFile(null)}
        />
      )}
    </LmsPage>
  )
}

function EmptyState({ canManage, onUpload }: { canManage: boolean; onUpload: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-slate-200 py-16 text-center dark:border-neutral-700">
      <span className="mb-3 text-4xl">📁</span>
      <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">No files yet</p>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        {canManage ? 'Upload files or create folders to organize course materials.' : 'No files have been uploaded to this course yet.'}
      </p>
      {canManage && (
        <button
          type="button"
          onClick={onUpload}
          className="mt-4 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
        >
          Upload files
        </button>
      )}
    </div>
  )
}

type FolderRowProps = {
  folder: FileFolder
  canManage: boolean
  renamingFolder: FileFolder | null
  renameValue: string
  setRenameValue: (v: string) => void
  onNavigate: () => void
  onRenameStart: () => void
  onRenameSubmit: () => void
  onRenameCancel: () => void
  onDelete: () => void
  onContextMenu: (e: React.MouseEvent) => void
}

function FolderRow({
  folder, canManage, renamingFolder, renameValue, setRenameValue,
  onNavigate, onRenameStart, onRenameSubmit, onRenameCancel, onDelete, onContextMenu,
}: FolderRowProps) {
  const isRenaming = renamingFolder?.id === folder.id
  return (
    <tr
      className="group cursor-pointer hover:bg-slate-50 dark:hover:bg-neutral-900/50"
      onContextMenu={onContextMenu}
    >
      <td className="py-2.5 pl-4 pr-3">
        {isRenaming ? (
          <form
            className="flex items-center gap-2"
            onSubmit={e => { e.preventDefault(); onRenameSubmit() }}
          >
            <span className="text-base">📁</span>
            <input
              autoFocus
              value={renameValue}
              onChange={e => setRenameValue(e.target.value)}
              onKeyDown={e => e.key === 'Escape' && onRenameCancel()}
              className="min-w-0 flex-1 rounded border border-indigo-400 px-1.5 py-0.5 text-sm dark:bg-neutral-800"
            />
            <button type="submit" className="text-xs text-indigo-600 hover:underline">Save</button>
            <button type="button" onClick={onRenameCancel} className="text-xs text-slate-400 hover:underline">Cancel</button>
          </form>
        ) : (
          <button
            className="flex items-center gap-2 text-left text-sm font-medium text-slate-800 hover:text-indigo-600 dark:text-neutral-100 dark:hover:text-indigo-400"
            onClick={onNavigate}
          >
            <span className="text-base">📁</span>
            {folder.name}
          </button>
        )}
      </td>
      <td className="px-3 py-2.5 text-slate-400 dark:text-neutral-500">—</td>
      <td className="px-3 py-2.5 text-slate-500 dark:text-neutral-400">
        {formatAbsoluteShort(folder.updatedAt)}
      </td>
      {canManage && (
        <td className="py-2.5 pl-3 pr-4 text-right">
          <div className="invisible flex items-center justify-end gap-2 group-hover:visible">
            <button onClick={onRenameStart} className="text-xs text-slate-500 hover:text-indigo-600 dark:text-neutral-400">Rename</button>
            <button onClick={onDelete} className="text-xs text-red-500 hover:text-red-700">Delete</button>
          </div>
        </td>
      )}
    </tr>
  )
}

type FileRowProps = {
  file: FileItem
  courseCode: string
  canManage: boolean
  renamingFile: FileItem | null
  renameValue: string
  setRenameValue: (v: string) => void
  onRenameStart: () => void
  onRenameSubmit: () => void
  onRenameCancel: () => void
  onDelete: () => void
  onMove: () => void
  onContextMenu: (e: React.MouseEvent) => void
}

function FileRow({
  file, courseCode, canManage, renamingFile, renameValue, setRenameValue,
  onRenameStart, onRenameSubmit, onRenameCancel, onDelete, onMove, onContextMenu,
}: FileRowProps) {
  const isRenaming = renamingFile?.id === file.id
  return (
    <tr
      className="group cursor-default hover:bg-slate-50 dark:hover:bg-neutral-900/50"
      onContextMenu={onContextMenu}
    >
      <td className="py-2.5 pl-4 pr-3">
        {isRenaming ? (
          <form
            className="flex items-center gap-2"
            onSubmit={e => { e.preventDefault(); onRenameSubmit() }}
          >
            <span className="text-base">{fileIconForMime(file.mimeType)}</span>
            <input
              autoFocus
              value={renameValue}
              onChange={e => setRenameValue(e.target.value)}
              onKeyDown={e => e.key === 'Escape' && onRenameCancel()}
              className="min-w-0 flex-1 rounded border border-indigo-400 px-1.5 py-0.5 text-sm dark:bg-neutral-800"
            />
            <button type="submit" className="text-xs text-indigo-600 hover:underline">Save</button>
            <button type="button" onClick={onRenameCancel} className="text-xs text-slate-400 hover:underline">Cancel</button>
          </form>
        ) : (
          <a
            href={getFileContentUrl(courseCode, file.id)}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-2 text-sm font-medium text-slate-800 hover:text-indigo-600 dark:text-neutral-100 dark:hover:text-indigo-400"
          >
            <span className="text-base">{fileIconForMime(file.mimeType)}</span>
            {file.displayName}
          </a>
        )}
      </td>
      <td className="px-3 py-2.5 text-slate-500 dark:text-neutral-400">{formatBytes(file.byteSize)}</td>
      <td className="px-3 py-2.5 text-slate-500 dark:text-neutral-400">
        {formatAbsoluteShort(file.updatedAt)}
      </td>
      {canManage && (
        <td className="py-2.5 pl-3 pr-4 text-right">
          <div className="invisible flex items-center justify-end gap-2 group-hover:visible">
            <button onClick={onRenameStart} className="text-xs text-slate-500 hover:text-indigo-600 dark:text-neutral-400">Rename</button>
            <button onClick={onMove} className="text-xs text-slate-500 hover:text-indigo-600 dark:text-neutral-400">Move</button>
            <button onClick={onDelete} className="text-xs text-red-500 hover:text-red-700">Delete</button>
          </div>
        </td>
      )}
    </tr>
  )
}

function MovePicker({
  file,
  folders,
  currentFolderId,
  onMove,
  onCancel,
}: {
  file: FileItem
  folders: FileFolder[]
  currentFolderId: string | null
  onMove: (folderId: string | null) => void
  onCancel: () => void
}) {
  return (
    <div className="fixed inset-0 z-30 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-neutral-900">
        <h2 className="mb-4 text-base font-semibold text-slate-900 dark:text-neutral-100">
          Move "{file.displayName}"
        </h2>
        <ul className="mb-4 max-h-56 overflow-y-auto rounded-lg border border-slate-200 dark:border-neutral-700">
          <li>
            <button
              className={`flex w-full items-center gap-2 px-4 py-2.5 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800 ${currentFolderId === null ? 'font-semibold text-indigo-600' : ''}`}
              onClick={() => onMove(null)}
            >
              📁 Root (top level)
            </button>
          </li>
          {folders.map(f => (
            <li key={f.id}>
              <button
                className={`flex w-full items-center gap-2 px-4 py-2.5 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800 ${currentFolderId === f.id ? 'font-semibold text-indigo-600' : ''}`}
                onClick={() => onMove(f.id)}
              >
                📁 {f.name}
              </button>
            </li>
          ))}
        </ul>
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={onCancel}
            className="rounded px-3 py-1.5 text-sm text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  )
}

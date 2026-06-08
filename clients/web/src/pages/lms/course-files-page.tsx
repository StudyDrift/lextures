import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'
import {
  Archive,
  ChevronDown,
  FileText,
  Film,
  FolderOpen,
  FolderPlus,
  Image,
  Music,
  Paperclip,
  Presentation,
  Search,
  Sheet,
  Trash2,
  Upload,
} from 'lucide-react'
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
  moveFolder,
  deleteFile,
  getFileContentUrl,
  formatBytes,
  type FileFolder,
  type FileItem,
  type FolderContents,
} from '../../lib/course-files-api'
import { authorizedFetch, wsUrl } from '../../lib/api'
import { getAccessToken } from '../../lib/auth'
import { FilePreview } from '../../components/file-preview'
import { ConfirmDialog } from '../../components/confirm-dialog'
import { CloudImportMenu } from '../../components/cloud-import-menu'
import { LmsPage } from './lms-page'

type ContextMenu =
  | { kind: 'folder'; item: FileFolder; x: number; y: number }
  | { kind: 'file'; item: FileItem; x: number; y: number }

type SelectedItemKey = `folder:${string}` | `file:${string}`

function selectedItemKey(kind: 'folder' | 'file', id: string): SelectedItemKey {
  return `${kind}:${id}`
}

function matchesSearchQuery(name: string, query: string): boolean {
  return name.toLowerCase().includes(query.trim().toLowerCase())
}

const CHECKBOX_COLUMN_CLASS = 'w-11 px-4 py-2.5 align-middle text-left'
const CHECKBOX_INPUT_CLASS =
  'block h-4 w-4 shrink-0 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-900'

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

  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<string | null>(null)
  const [cloudImportError, setCloudImportError] = useState<string | null>(null)

  const [renamingFolder, setRenamingFolder] = useState<FileFolder | null>(null)
  const [renamingFile, setRenamingFile] = useState<FileItem | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [newFolderName, setNewFolderName] = useState('')
  const [showNewFolder, setShowNewFolder] = useState(false)

  const [contextMenu, setContextMenu] = useState<ContextMenu | null>(null)
  const [draggingItem, setDraggingItem] = useState<{ kind: 'folder' | 'file'; id: string } | null>(null)
  const [dragOverFolderId, setDragOverFolderId] = useState<string | null>(null)
  const [dragOverBreadcrumbId, setDragOverBreadcrumbId] = useState<string | 'root' | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedItems, setSelectedItems] = useState<Set<SelectedItemKey>>(new Set())
  const [previewFile, setPreviewFile] = useState<FileItem | null>(null)
  const [pendingDelete, setPendingDelete] = useState<{
    title: string
    description: string
    action: () => Promise<void>
  } | null>(null)
  const [deleting, setDeleting] = useState(false)

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

  const loadRef = useRef(load)
  useEffect(() => { loadRef.current = load }, [load])

  useEffect(() => {
    if (!courseCode) return
    const token = getAccessToken()
    if (!token) return
    const ws = new WebSocket(wsUrl(`/api/v1/courses/${encodeURIComponent(courseCode)}/files/ws`))
    ws.onopen = () => { ws.send(JSON.stringify({ authToken: token })) }
    ws.onmessage = (ev) => {
      try {
        const data = JSON.parse(String(ev.data)) as { type?: string }
        if (data.type === 'files_changed') void loadRef.current()
      } catch { /* ignore */ }
    }
    return () => { ws.close() }
  }, [courseCode])

  useEffect(() => {
    setSelectedItems(new Set())
    setSearchQuery('')
  }, [folderId])

  const breadcrumbs = useMemo(() => {
    if (!folderId || contents?.folderId !== folderId) return []
    return contents.breadcrumbs ?? []
  }, [folderId, contents?.folderId, contents?.breadcrumbs])

  const filteredFolders = useMemo(() => {
    if (!contents) return []
    const q = searchQuery.trim()
    if (!q) return contents.folders
    return contents.folders.filter(folder => matchesSearchQuery(folder.name, q))
  }, [contents, searchQuery])

  const filteredFiles = useMemo(() => {
    if (!contents) return []
    const q = searchQuery.trim()
    if (!q) return contents.files
    return contents.files.filter(file => matchesSearchQuery(file.displayName, q))
  }, [contents, searchQuery])

  const visibleItemKeys = useMemo(() => {
    const keys: SelectedItemKey[] = []
    for (const folder of filteredFolders) keys.push(selectedItemKey('folder', folder.id))
    for (const file of filteredFiles) keys.push(selectedItemKey('file', file.id))
    return keys
  }, [filteredFolders, filteredFiles])

  const allVisibleSelected =
    visibleItemKeys.length > 0 && visibleItemKeys.every(key => selectedItems.has(key))

  const someVisibleSelected =
    visibleItemKeys.some(key => selectedItems.has(key)) && !allVisibleSelected

  function toggleItemSelection(key: SelectedItemKey) {
    setSelectedItems(prev => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  function toggleSelectAllVisible() {
    setSelectedItems(prev => {
      if (allVisibleSelected) {
        const next = new Set(prev)
        for (const key of visibleItemKeys) next.delete(key)
        return next
      }
      const next = new Set(prev)
      for (const key of visibleItemKeys) next.add(key)
      return next
    })
  }

  function clearSelection() {
    setSelectedItems(new Set())
  }

  function getSelectedFolderAndFile() {
    if (!contents || selectedItems.size !== 1) return null
    const key = [...selectedItems][0]
    const [kind, id] = key.split(':') as ['folder' | 'file', string]
    if (kind === 'folder') {
      const folder = contents.folders.find(f => f.id === id)
      return folder ? { kind: 'folder' as const, folder } : null
    }
    const file = contents.files.find(f => f.id === id)
    return file ? { kind: 'file' as const, file } : null
  }

  async function handleMoveSelectedUp() {
    if (!folderId || selectedItems.size === 0) return
    const parentFolderId = breadcrumbs.length >= 2 ? breadcrumbs[breadcrumbs.length - 2].id : null
    const foldersToMove = contents?.folders.filter(f => selectedItems.has(selectedItemKey('folder', f.id))) ?? []
    const filesToMove = contents?.files.filter(f => selectedItems.has(selectedItemKey('file', f.id))) ?? []
    try {
      for (const folder of foldersToMove) {
        await moveFolder(courseCode, folder.id, parentFolderId)
      }
      for (const file of filesToMove) {
        await moveFile(courseCode, file.id, parentFolderId)
      }
      clearSelection()
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Could not move items.')
    }
  }

  function handleDeleteSelected() {
    if (!contents || selectedItems.size === 0) return
    const foldersToDelete = contents.folders.filter(f =>
      selectedItems.has(selectedItemKey('folder', f.id)),
    )
    const filesToDelete = contents.files.filter(f =>
      selectedItems.has(selectedItemKey('file', f.id)),
    )
    const total = foldersToDelete.length + filesToDelete.length
    const description =
      total === 1
        ? foldersToDelete.length === 1
          ? `Delete folder "${foldersToDelete[0].name}" and all its contents? This cannot be undone.`
          : `Delete "${filesToDelete[0].displayName}"? This cannot be undone.`
        : `Delete ${total} selected items? Folders and their contents will be removed. This cannot be undone.`
    setPendingDelete({
      title: total === 1 ? 'Delete item' : `Delete ${total} items`,
      description,
      action: async () => {
        for (const folder of foldersToDelete) {
          await deleteFolder(courseCode, folder.id)
        }
        for (const file of filesToDelete) {
          await deleteFile(courseCode, file.id)
        }
        clearSelection()
        void load()
      },
    })
  }

  function handleRenameSelected() {
    const selected = getSelectedFolderAndFile()
    if (!selected) return
    if (selected.kind === 'folder') {
      setRenamingFolder(selected.folder)
      setRenameValue(selected.folder.name)
      return
    }
    setRenamingFile(selected.file)
    setRenameValue(selected.file.displayName)
  }

  function navigateToFolder(id: string | null) {
    if (!id) {
      setSearchParams({})
    } else {
      setSearchParams({ folder: id })
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

  function handleDeleteFolder(folder: FileFolder) {
    setPendingDelete({
      title: 'Delete folder',
      description: `Delete folder "${folder.name}" and all its contents? This cannot be undone.`,
      action: async () => {
        await deleteFolder(courseCode, folder.id)
        void load()
      },
    })
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

  function handleDeleteFile(file: FileItem) {
    setPendingDelete({
      title: 'Delete file',
      description: `Delete "${file.displayName}"? This cannot be undone.`,
      action: async () => {
        await deleteFile(courseCode, file.id)
        void load()
      },
    })
  }

  async function handleMoveItem(kind: 'folder' | 'file', id: string, targetFolderId: string | null) {
    try {
      if (kind === 'file') {
        await moveFile(courseCode, id, targetFolderId)
      } else {
        await moveFolder(courseCode, id, targetFolderId)
      }
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : `Could not move ${kind}.`)
    }
  }

  async function handleDownloadFile(file: FileItem) {
    try {
      const res = await authorizedFetch(getFileContentUrl(courseCode, file.id))
      if (!res.ok) throw new Error()
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = file.displayName
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch {
      /* noop */
    }
  }

  async function uploadSingleFile(file: File) {
    setUploadProgress(`Uploading ${file.name}…`)
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
  }

  async function handleFilesSelected(files: FileList | null) {
    if (!files || files.length === 0) return
    setUploading(true)
    setCloudImportError(null)
    for (let i = 0; i < files.length; i++) {
      const file = files[i]
      try {
        await uploadSingleFile(file)
      } catch (err) {
        alert(`Failed to upload ${file.name}: ${err instanceof Error ? err.message : 'Unknown error'}`)
      }
    }
    setUploading(false)
    setUploadProgress(null)
    void load()
  }

  async function handleCloudImport(file: File) {
    setUploading(true)
    setCloudImportError(null)
    try {
      await uploadSingleFile(file)
      void load()
    } catch (err) {
      setCloudImportError(err instanceof Error ? err.message : 'Could not import file from cloud storage.')
    } finally {
      setUploading(false)
      setUploadProgress(null)
    }
  }

  function openContextMenu(e: React.MouseEvent, item: FileFolder | FileItem, kind: 'folder' | 'file') {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ kind, item: item as FileFolder & FileItem, x: e.clientX, y: e.clientY })
  }

  return (
    <LmsPage title="Files">
      {/* dismiss context menu on outside click */}
      {contextMenu && (
        <div
          className="fixed inset-0 z-10"
          onClick={() => setContextMenu(null)}
        />
      )}

      {/* Breadcrumb trail */}
      <nav
        aria-label="Folder path"
        className="mb-3 flex min-w-0 flex-wrap items-center gap-1 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900/60"
      >
        <span
          onDragOver={(e) => {
            if (draggingItem && folderId !== undefined) {
              e.preventDefault()
              e.dataTransfer.dropEffect = 'move'
            }
          }}
          onDragEnter={() => {
            if (draggingItem && folderId !== undefined) {
              setDragOverBreadcrumbId('root')
            }
          }}
          onDragLeave={(e) => {
            if (!e.currentTarget.contains(e.relatedTarget as Node)) {
              setDragOverBreadcrumbId(null)
            }
          }}
          onDrop={() => {
            if (draggingItem && folderId !== undefined) {
              void handleMoveItem(draggingItem.kind, draggingItem.id, null)
            }
            setDragOverBreadcrumbId(null)
          }}
          className={`rounded px-1.5 py-0.5 transition-all duration-150 ${
            draggingItem && folderId !== undefined
              ? dragOverBreadcrumbId === 'root'
                ? 'bg-indigo-100 font-semibold text-indigo-700 ring-2 ring-indigo-500 dark:bg-indigo-900/60 dark:text-indigo-300'
                : 'border border-dashed border-indigo-300 bg-indigo-50/50 dark:border-indigo-700 dark:bg-indigo-950/20'
              : ''
          }`}
        >
          <button
            type="button"
            onClick={() => navigateToFolder(null)}
            className="text-indigo-600 hover:underline dark:text-indigo-400"
            aria-current={!folderId ? 'page' : undefined}
          >
            Files
          </button>
        </span>
        {breadcrumbs.map((b, i) => {
          const isLast = i === breadcrumbs.length - 1
          const canDropOnCrumb = draggingItem && folderId !== b.id && draggingItem.id !== b.id
          return (
            <span key={b.id} className="flex min-w-0 items-center gap-1">
              <span className="text-slate-400 dark:text-neutral-500" aria-hidden>/</span>
              <span
                onDragOver={(e) => {
                  if (canDropOnCrumb) {
                    e.preventDefault()
                    e.dataTransfer.dropEffect = 'move'
                  }
                }}
                onDragEnter={() => {
                  if (canDropOnCrumb) {
                    setDragOverBreadcrumbId(b.id)
                  }
                }}
                onDragLeave={(e) => {
                  if (!e.currentTarget.contains(e.relatedTarget as Node)) {
                    setDragOverBreadcrumbId(null)
                  }
                }}
                onDrop={() => {
                  if (canDropOnCrumb) {
                    void handleMoveItem(draggingItem!.kind, draggingItem!.id, b.id)
                  }
                  setDragOverBreadcrumbId(null)
                }}
                className={`min-w-0 rounded px-1.5 py-0.5 transition-all duration-150 ${
                  canDropOnCrumb
                    ? dragOverBreadcrumbId === b.id
                      ? 'bg-indigo-100 font-semibold text-indigo-700 ring-2 ring-indigo-500 dark:bg-indigo-900/60 dark:text-indigo-300'
                      : 'border border-dashed border-indigo-300 bg-indigo-50/50 dark:border-indigo-700 dark:bg-indigo-950/20'
                    : ''
                }`}
              >
                {!isLast ? (
                  <button
                    type="button"
                    onClick={() => navigateToFolder(b.id)}
                    className="max-w-[12rem] truncate text-indigo-600 hover:underline dark:text-indigo-400"
                  >
                    {b.name}
                  </button>
                ) : (
                  <span className="block max-w-[12rem] truncate font-medium text-slate-800 dark:text-neutral-200" aria-current="page">{b.name}</span>
                )}
              </span>
            </span>
          )
        })}
      </nav>

      {/* Toolbar */}
      <div className="mb-4 flex flex-wrap items-center justify-end gap-2">
        {canManage && (
          <div className="flex shrink-0 items-center gap-2">
            <button
              type="button"
              onClick={() => setShowNewFolder(v => !v)}
              className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              <FolderPlus className="h-4 w-4" aria-hidden />
              New folder
            </button>
            <button
              type="button"
              disabled={uploading}
              onClick={() => fileInputRef.current?.click()}
              className="inline-flex items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 disabled:opacity-60"
            >
              {uploading ? (
                uploadProgress ?? 'Uploading…'
              ) : (
                <>
                  <Upload className="h-4 w-4" aria-hidden />
                  Upload
                </>
              )}
            </button>
            <CloudImportMenu
              disabled={uploading}
              onImportFile={handleCloudImport}
              onError={setCloudImportError}
            />
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

      {cloudImportError && (
        <p className="mb-4 text-sm text-rose-700 dark:text-rose-300" role="alert">
          {cloudImportError}
        </p>
      )}

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

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-[12rem] flex-1 sm:max-w-md">
          <Search
            className="pointer-events-none absolute start-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 dark:text-neutral-500"
            aria-hidden
          />
          <input
            type="search"
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            placeholder="Search files and folders…"
            aria-label="Search files and folders"
            className="w-full rounded-lg border border-slate-200 bg-white py-2 ps-9 pe-3 text-sm text-slate-900 outline-none placeholder:text-slate-500 focus:border-indigo-300 focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-indigo-500"
          />
        </div>
        {canManage && selectedItems.size > 0 && (
          <div className="flex shrink-0 items-center gap-2">
            <span className="text-sm text-slate-600 dark:text-neutral-300">
              {selectedItems.size} selected
            </span>
            <SelectionActionsMenu
              canRename={selectedItems.size === 1}
              onRename={handleRenameSelected}
              canMoveUp={!!folderId}
              onMoveUp={() => void handleMoveSelectedUp()}
              onDelete={() => void handleDeleteSelected()}
            />
            <button
              type="button"
              onClick={clearSelection}
              className="text-sm text-slate-500 hover:text-slate-700 dark:text-neutral-400 dark:hover:text-neutral-200"
            >
              Clear
            </button>
          </div>
        )}
      </div>

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
      ) : filteredFolders.length === 0 && filteredFiles.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-slate-200 bg-white py-16 text-center dark:border-neutral-800 dark:bg-neutral-950">
          <Search className="mb-3 h-10 w-10 text-slate-300 dark:text-neutral-600" aria-hidden />
          <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">No matching files or folders</p>
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
            Try a different search term.
          </p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-slate-200 bg-white dark:border-neutral-800 dark:bg-neutral-950">
          <table className="min-w-full table-fixed divide-y divide-slate-100 text-sm dark:divide-neutral-800">
            {canManage && (
              <colgroup>
                <col className="w-11" />
                <col />
                <col className="w-28" />
                <col className="w-36" />
                <col className="w-36" />
              </colgroup>
            )}
            <thead>
              <tr className="bg-slate-50 dark:bg-neutral-900">
                {canManage && (
                  <th className={CHECKBOX_COLUMN_CLASS}>
                    <SelectAllCheckbox
                      checked={allVisibleSelected}
                      indeterminate={someVisibleSelected}
                      onChange={toggleSelectAllVisible}
                    />
                  </th>
                )}
                <th className={`py-2.5 pr-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400 ${canManage ? 'pl-2' : 'pl-4'}`}>Name</th>
                <th className="px-3 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">Size</th>
                <th className="px-3 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">Modified</th>
                {canManage && <th className="py-2.5 pl-3 pr-4 text-right text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">Actions</th>}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50 dark:divide-neutral-800/60">
              {filteredFolders.map(folder => (
                <FolderRow
                  key={folder.id}
                  folder={folder}
                  canManage={canManage}
                  selected={selectedItems.has(selectedItemKey('folder', folder.id))}
                  onToggleSelect={() => toggleItemSelection(selectedItemKey('folder', folder.id))}
                  renamingFolder={renamingFolder}
                  renameValue={renameValue}
                  setRenameValue={setRenameValue}
                  onNavigate={() => navigateToFolder(folder.id)}
                  onRenameStart={() => { setRenamingFolder(folder); setRenameValue(folder.name) }}
                  onRenameSubmit={() => void handleRenameFolder()}
                  onRenameCancel={() => setRenamingFolder(null)}
                  onDelete={() => void handleDeleteFolder(folder)}
                  onContextMenu={e => openContextMenu(e, folder, 'folder')}
                  draggingItem={draggingItem}
                  dragOverFolderId={dragOverFolderId}
                  onDragStart={(kind, id) => setDraggingItem({ kind, id })}
                  onDragEnd={() => { setDraggingItem(null); setDragOverFolderId(null); setDragOverBreadcrumbId(null); }}
                  onDragEnter={(id) => setDragOverFolderId(id)}
                  onDragLeave={() => setDragOverFolderId(null)}
                  onDrop={(kind, dragId, targetId) => void handleMoveItem(kind, dragId, targetId)}
                />
              ))}
              {filteredFiles.map(file => (
                <FileRow
                  key={file.id}
                  file={file}
                  canManage={canManage}
                  selected={selectedItems.has(selectedItemKey('file', file.id))}
                  onToggleSelect={() => toggleItemSelection(selectedItemKey('file', file.id))}
                  renamingFile={renamingFile}
                  renameValue={renameValue}
                  setRenameValue={setRenameValue}
                  onRenameStart={() => { setRenamingFile(file); setRenameValue(file.displayName) }}
                  onRenameSubmit={() => void handleRenameFile()}
                  onRenameCancel={() => setRenamingFile(null)}
                  onDelete={() => void handleDeleteFile(file)}
                  onPreview={() => setPreviewFile(file)}
                  onContextMenu={e => openContextMenu(e, file, 'file')}
                  draggingItem={draggingItem}
                  onDragStart={(kind, id) => setDraggingItem({ kind, id })}
                  onDragEnd={() => { setDraggingItem(null); setDragOverFolderId(null); setDragOverBreadcrumbId(null); }}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Context menu */}
      {contextMenu && (
        <div
          className="fixed inset-0 z-10"
          onClick={() => setContextMenu(null)}
        />
      )}
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
                onClick={() => { navigateToFolder(contextMenu.item.id); setContextMenu(null) }}
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
              <button
                className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                onClick={() => { setPreviewFile(contextMenu.item as FileItem); setContextMenu(null) }}
              >
                Preview
              </button>
              <button
                className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                onClick={() => { void handleDownloadFile(contextMenu.item as FileItem); setContextMenu(null) }}
              >
                Download
              </button>
              {canManage && (
                <>
                  <button
                    className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                    onClick={() => { setRenamingFile(contextMenu.item as FileItem); setRenameValue((contextMenu.item as FileItem).displayName); setContextMenu(null) }}
                  >
                    Rename
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

      <ConfirmDialog
        open={pendingDelete !== null}
        title={pendingDelete?.title ?? ''}
        description={pendingDelete?.description}
        confirmLabel="Delete"
        variant="danger"
        busy={deleting}
        onConfirm={async () => {
          if (!pendingDelete) return
          setDeleting(true)
          try {
            await pendingDelete.action()
          } catch (err) {
            alert(err instanceof Error ? err.message : 'Could not delete.')
          } finally {
            setDeleting(false)
            setPendingDelete(null)
          }
        }}
        onClose={() => { if (!deleting) setPendingDelete(null) }}
      />

      <FilePreview
        open={previewFile !== null}
        filePath={previewFile ? getFileContentUrl(courseCode, previewFile.id) : ''}
        filename={previewFile?.displayName ?? ''}
        mimeType={previewFile?.mimeType ?? null}
        onClose={() => setPreviewFile(null)}
      />
    </LmsPage>
  )
}

function FileMimeIcon({ mimeType }: { mimeType: string }) {
  const className = 'h-4 w-4 shrink-0 text-slate-400 dark:text-neutral-500'
  if (mimeType.startsWith('image/')) return <Image className={className} aria-hidden />
  if (mimeType === 'application/pdf') return <FileText className={className} aria-hidden />
  if (mimeType.includes('spreadsheet') || mimeType.includes('excel') || mimeType.includes('csv')) {
    return <Sheet className={className} aria-hidden />
  }
  if (mimeType.includes('presentation') || mimeType.includes('powerpoint')) {
    return <Presentation className={className} aria-hidden />
  }
  if (mimeType.includes('word') || mimeType.includes('document')) {
    return <FileText className={className} aria-hidden />
  }
  if (mimeType.startsWith('video/')) return <Film className={className} aria-hidden />
  if (mimeType.startsWith('audio/')) return <Music className={className} aria-hidden />
  if (mimeType.includes('zip') || mimeType.includes('archive') || mimeType.includes('compressed')) {
    return <Archive className={className} aria-hidden />
  }
  return <Paperclip className={className} aria-hidden />
}

function SelectAllCheckbox({
  checked,
  indeterminate,
  onChange,
}: {
  checked: boolean
  indeterminate: boolean
  onChange: () => void
}) {
  const ref = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (ref.current) ref.current.indeterminate = indeterminate
  }, [indeterminate])

  return (
    <input
      ref={ref}
      type="checkbox"
      checked={checked}
      onChange={onChange}
      aria-label="Select all visible items"
      className={CHECKBOX_INPUT_CLASS}
    />
  )
}

function SelectionActionsMenu({
  canRename,
  onRename,
  canMoveUp,
  onMoveUp,
  onDelete,
}: {
  canRename: boolean
  onRename: () => void
  canMoveUp: boolean
  onMoveUp: () => void
  onDelete: () => void
}) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  return (
    <div ref={rootRef} className="relative inline-block">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen(o => !o)}
        className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
      >
        Actions
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>
      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Selected file actions"
          className="absolute start-0 z-50 mt-1 min-w-[10rem] overflow-hidden rounded-lg border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
        >
          <button
            type="button"
            role="menuitem"
            disabled={!canRename}
            onClick={() => {
              onRename()
              setOpen(false)
            }}
            className="flex w-full items-center px-2.5 py-2 text-start text-sm hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-neutral-800"
          >
            Rename
          </button>
          {canMoveUp && (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onMoveUp()
                setOpen(false)
              }}
              className="flex w-full items-center px-2.5 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
            >
              Move up a folder
            </button>
          )}
          <hr className="my-1 border-slate-100 dark:border-neutral-700" />
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onDelete()
              setOpen(false)
            }}
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30"
          >
            <Trash2 className="h-4 w-4 shrink-0" aria-hidden />
            Delete
          </button>
        </div>
      )}
    </div>
  )
}

function EmptyState({ canManage, onUpload }: { canManage: boolean; onUpload: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-slate-200 py-16 text-center dark:border-neutral-700">
      <FolderOpen className="mb-3 h-12 w-12 text-slate-300 dark:text-neutral-600" aria-hidden />
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
  selected: boolean
  onToggleSelect: () => void
  renamingFolder: FileFolder | null
  renameValue: string
  setRenameValue: (v: string) => void
  onNavigate: () => void
  onRenameStart: () => void
  onRenameSubmit: () => void
  onRenameCancel: () => void
  onDelete: () => void
  onContextMenu: (e: React.MouseEvent) => void
  draggingItem: { kind: 'folder' | 'file'; id: string } | null
  dragOverFolderId: string | null
  onDragStart: (kind: 'folder' | 'file', id: string) => void
  onDragEnd: () => void
  onDragEnter: (id: string) => void
  onDragLeave: () => void
  onDrop: (kind: 'folder' | 'file', dragId: string, targetId: string) => void
}

function FolderRow({
  folder, canManage, selected, onToggleSelect, renamingFolder, renameValue, setRenameValue,
  onNavigate, onRenameStart, onRenameSubmit, onRenameCancel, onDelete, onContextMenu,
  draggingItem, dragOverFolderId, onDragStart, onDragEnd, onDragEnter, onDragLeave, onDrop,
}: FolderRowProps) {
  const isRenaming = renamingFolder?.id === folder.id
  const isDraggingActive = draggingItem !== null
  const isSelf = draggingItem?.id === folder.id && draggingItem?.kind === 'folder'
  const isDragOver = dragOverFolderId === folder.id

  let dragClass = 'group cursor-pointer hover:bg-slate-50 dark:hover:bg-neutral-900/50 transition-colors'
  if (isDraggingActive) {
    if (isSelf) {
      dragClass = 'opacity-40 select-none pointer-events-none'
    } else {
      if (isDragOver) {
        dragClass = 'bg-indigo-100/90 dark:bg-indigo-900/60 text-indigo-700 dark:text-indigo-300 ring-2 ring-indigo-500 ring-inset font-semibold shadow-sm'
      } else {
        dragClass = 'bg-indigo-50/60 dark:bg-indigo-950/30 text-indigo-600 dark:text-indigo-400 border border-dashed border-indigo-300 dark:border-indigo-700'
      }
    }
  }

  return (
    <tr
      className={dragClass}
      onContextMenu={onContextMenu}
      draggable={canManage && !isRenaming}
      onDragStart={(e) => {
        if (canManage && !isRenaming) {
          e.dataTransfer.effectAllowed = 'move'
          // Defer state update so the browser captures the drag ghost before re-render
          setTimeout(() => onDragStart('folder', folder.id), 0)
        }
      }}
      onDragEnd={onDragEnd}
      onDragOver={(e) => {
        if (isDraggingActive && !isSelf) {
          e.preventDefault()
          e.dataTransfer.dropEffect = 'move'
        }
      }}
      onDragEnter={() => {
        if (isDraggingActive && !isSelf) {
          onDragEnter(folder.id)
        }
      }}
      onDragLeave={(e) => {
        // Only clear highlight when truly leaving the row, not when crossing into a child cell
        if (!e.currentTarget.contains(e.relatedTarget as Node)) {
          onDragLeave()
        }
      }}
      onDrop={(e) => {
        if (isDraggingActive && !isSelf) {
          e.preventDefault()
          onDrop(draggingItem.kind, draggingItem.id, folder.id)
        }
      }}
    >
      {canManage && (
        <td className={CHECKBOX_COLUMN_CLASS}>
          <input
            type="checkbox"
            checked={selected}
            onChange={onToggleSelect}
            onClick={e => e.stopPropagation()}
            aria-label={`Select folder ${folder.name}`}
            className={CHECKBOX_INPUT_CLASS}
          />
        </td>
      )}
      <td className={`py-2.5 pr-3 ${canManage ? 'pl-2' : 'pl-4'}`}>
        {isRenaming ? (
          <form
            className="flex items-center gap-2"
            onSubmit={e => { e.preventDefault(); onRenameSubmit() }}
          >
            <FolderOpen className="h-4 w-4 shrink-0 text-slate-400 dark:text-neutral-500" aria-hidden />
            <input
              autoFocus
              value={renameValue}
              onChange={e => setRenameValue(e.target.value)}
              onKeyDown={e => e.key === 'Escape' && onRenameCancel()}
              className="min-w-0 flex-1 rounded border border-indigo-400 px-1.5 py-0.5 text-sm dark:bg-neutral-800"
            />
            <button type="submit" className="text-xs text-indigo-600 hover:underline font-medium">Save</button>
            <button type="button" onClick={onRenameCancel} className="text-xs text-slate-400 hover:underline">Cancel</button>
          </form>
        ) : (
          <button
            className="flex items-center gap-2 text-left text-sm font-medium text-slate-800 hover:text-indigo-600 dark:text-neutral-100 dark:hover:text-indigo-400"
            onClick={onNavigate}
          >
            <FolderOpen className="h-4 w-4 shrink-0 text-slate-400 dark:text-neutral-500" aria-hidden />
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
            <button onClick={onRenameStart} className="text-xs text-slate-500 hover:text-indigo-600 dark:text-neutral-400 font-medium">Rename</button>
            <button onClick={onDelete} className="text-xs text-red-500 hover:text-red-700 font-medium">Delete</button>
          </div>
        </td>
      )}
    </tr>
  )
}

type FileRowProps = {
  file: FileItem
  canManage: boolean
  selected: boolean
  onToggleSelect: () => void
  renamingFile: FileItem | null
  renameValue: string
  setRenameValue: (v: string) => void
  onRenameStart: () => void
  onRenameSubmit: () => void
  onRenameCancel: () => void
  onDelete: () => void
  onPreview: () => void
  onContextMenu: (e: React.MouseEvent) => void
  draggingItem: { kind: 'folder' | 'file'; id: string } | null
  onDragStart: (kind: 'folder' | 'file', id: string) => void
  onDragEnd: () => void
}

function FileRow({
  file, canManage, selected, onToggleSelect, renamingFile, renameValue, setRenameValue,
  onRenameStart, onRenameSubmit, onRenameCancel, onDelete, onPreview, onContextMenu,
  draggingItem, onDragStart, onDragEnd,
}: FileRowProps) {
  const isRenaming = renamingFile?.id === file.id
  const isDraggingActive = draggingItem !== null
  const isSelf = draggingItem?.id === file.id && draggingItem?.kind === 'file'

  let dragClass = 'group cursor-default hover:bg-slate-50 dark:hover:bg-neutral-900/50 transition-colors'
  if (isDraggingActive) {
    if (isSelf) {
      dragClass = 'opacity-40 select-none pointer-events-none'
    } else {
      dragClass = 'opacity-30 pointer-events-none select-none grayscale'
    }
  }

  return (
    <tr
      className={dragClass}
      onContextMenu={onContextMenu}
      draggable={canManage && !isRenaming}
      onDragStart={(e) => {
        if (canManage && !isRenaming) {
          e.dataTransfer.effectAllowed = 'move'
          setTimeout(() => onDragStart('file', file.id), 0)
        }
      }}
      onDragEnd={onDragEnd}
    >
      {canManage && (
        <td className={CHECKBOX_COLUMN_CLASS}>
          <input
            type="checkbox"
            checked={selected}
            onChange={onToggleSelect}
            onClick={e => e.stopPropagation()}
            aria-label={`Select file ${file.displayName}`}
            className={CHECKBOX_INPUT_CLASS}
          />
        </td>
      )}
      <td className={`py-2.5 pr-3 ${canManage ? 'pl-2' : 'pl-4'}`}>
        {isRenaming ? (
          <form
            className="flex items-center gap-2"
            onSubmit={e => { e.preventDefault(); onRenameSubmit() }}
          >
            <FileMimeIcon mimeType={file.mimeType} />
            <input
              autoFocus
              value={renameValue}
              onChange={e => setRenameValue(e.target.value)}
              onKeyDown={e => e.key === 'Escape' && onRenameCancel()}
              className="min-w-0 flex-1 rounded border border-indigo-400 px-1.5 py-0.5 text-sm dark:bg-neutral-800"
            />
            <button type="submit" className="text-xs text-indigo-600 hover:underline font-medium">Save</button>
            <button type="button" onClick={onRenameCancel} className="text-xs text-slate-400 hover:underline">Cancel</button>
          </form>
        ) : (
          <button
            type="button"
            onClick={onPreview}
            className="flex items-center gap-2 text-left text-sm font-medium text-slate-800 hover:text-indigo-600 dark:text-neutral-100 dark:hover:text-indigo-400"
          >
            <FileMimeIcon mimeType={file.mimeType} />
            {file.displayName}
          </button>
        )}
      </td>
      <td className="px-3 py-2.5 text-slate-500 dark:text-neutral-400">{formatBytes(file.byteSize)}</td>
      <td className="px-3 py-2.5 text-slate-500 dark:text-neutral-400">
        {formatAbsoluteShort(file.updatedAt)}
      </td>
      {canManage && (
        <td className="py-2.5 pl-3 pr-4 text-right">
          <div className="invisible flex items-center justify-end gap-2 group-hover:visible">
            <button onClick={onRenameStart} className="text-xs text-slate-500 hover:text-indigo-600 dark:text-neutral-400 font-medium">Rename</button>
            <button onClick={onDelete} className="text-xs text-red-500 hover:text-red-700 font-medium">Delete</button>
          </div>
        </td>
      )}
    </tr>
  )
}

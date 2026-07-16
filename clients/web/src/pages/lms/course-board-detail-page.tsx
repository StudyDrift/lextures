import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, MoreHorizontal } from 'lucide-react'
import {
  arrangeBoardPost,
  createBoardSection,
  deleteBoard,
  deleteBoardPost,
  deleteBoardSection,
  fetchBoard,
  fetchBoardModerationQueue,
  listBoardPosts,
  listBoardSections,
  patchBoard,
  type ArrangeBoardPostInput,
  type Board,
  type BoardFilterAction,
  type BoardLayout,
  type BoardModerationMode,
  type BoardPost,
  type BoardReactionMode,
  type BoardSection,
  type BoardSortMode,
} from '../../lib/boards-api'
import { courseItemCreatePermission, fetchCourse } from '../../lib/courses-api'
import { toastMutationError } from '../../lib/lms-toast'
import { useConfirm } from '../../components/use-confirm'
import { BoardSurface } from '../../components/boards/board-surface'
import { BoardPresenceBar } from '../../components/boards/board-presence-bar'
import { BoardSyncStatus } from '../../components/boards/board-sync-status'
import { LayoutSwitcher } from '../../components/boards/layout-switcher'
import { SortControls } from '../../components/boards/sort-controls'
import { PostComposer } from '../../components/boards/post-composer'
import { BoardShareDialog } from '../../components/boards/share-dialog'
import { BoardModerationQueue } from '../../components/boards/moderation-queue'
import { SaveAsTemplateDialog } from '../../components/boards/save-as-template-dialog'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { usePermissions } from '../../context/use-permissions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { getJwtSubject } from '../../lib/auth'
import { useBoardRealtime } from '../../lib/boards-realtime'
import { LmsPage } from './lms-page'

const LAYOUTS_HIDING_SORT: BoardLayout[] = ['canvas', 'timeline', 'map', 'columns']

export default function CourseBoardDetailPage() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const { confirm, ConfirmDialogHost } = useConfirm()
  const { courseCode: rawCode, boardId: rawBoardId } = useParams<{
    courseCode: string
    boardId: string
  }>()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''
  const boardId = rawBoardId ? decodeURIComponent(rawBoardId) : ''
  const { allows, loading: permLoading } = usePermissions()
  const { ffVisualBoards, ffBoardsRealtime } = usePlatformFeatures()
  const canManageBoard = !permLoading && !!courseCode && allows(courseItemCreatePermission(courseCode))
  const viewerId = getJwtSubject()

  const [board, setBoard] = useState<Board | null>(null)
  const [posts, setPosts] = useState<BoardPost[]>([])
  const [sections, setSections] = useState<BoardSection[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [renaming, setRenaming] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const [menuOpen, setMenuOpen] = useState(false)
  const [sortMode, setSortMode] = useState<BoardSortMode>('newest')
  const [announce, setAnnounce] = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [shareOpen, setShareOpen] = useState(false)
  const [moderationOpen, setModerationOpen] = useState(false)
  const [saveTemplateOpen, setSaveTemplateOpen] = useState(false)
  const [minorsFloor, setMinorsFloor] = useState(false)

  const realtimeEnabled =
    ffVisualBoards && ffBoardsRealtime && !!board && !board.archived && !loading && !error
  const refetchPostsTimer = useRef<number | null>(null)
  const realtime = useBoardRealtime({
    courseCode,
    boardId,
    enabled: realtimeEnabled,
    displayName: viewerId ? viewerId.slice(0, 8) : t('boards.presence.anonymous'),
    posts,
    onRemoteCardAdded: () => setAnnounce(t('boards.sync.cardAdded')),
    onUnknownPostIds: () => {
      if (refetchPostsTimer.current) window.clearTimeout(refetchPostsTimer.current)
      refetchPostsTimer.current = window.setTimeout(() => {
        void listBoardPosts(courseCode, boardId)
          .then(setPosts)
          .catch(() => {
            /* keep local state; next reconnect/refetch will heal */
          })
      }, 100)
    },
  })
  const displayPosts = realtimeEnabled ? realtime.mergedPosts : posts

  const listBase = `/courses/${encodeURIComponent(courseCode)}/boards`

  const load = useCallback(async () => {
    if (!courseCode || !boardId) return
    setLoading(true)
    setError(null)
    try {
      if (!ffVisualBoards) {
        setError(t('boards.error.disabled'))
        return
      }
      const course = await fetchCourse(courseCode)
      if (!course.visualBoardsEnabled) {
        setError(t('boards.error.disabled'))
        return
      }
      const [row, postRows, sectionRows] = await Promise.all([
        fetchBoard(courseCode, boardId),
        listBoardPosts(courseCode, boardId),
        listBoardSections(courseCode, boardId),
      ])
      setBoard(row)
      setTitleDraft(row.title)
      setPosts(postRows)
      setSections(sectionRows)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('boards.error.loadBoard'))
    } finally {
      setLoading(false)
    }
  }, [boardId, courseCode, ffVisualBoards, t])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    if (!settingsOpen || !canManageBoard || !courseCode || !boardId) return
    void fetchBoardModerationQueue(courseCode, boardId)
      .then((q) => setMinorsFloor(q.minorsFloor))
      .catch(() => {
        /* ignore — settings still usable */
      })
  }, [settingsOpen, canManageBoard, courseCode, boardId])

  useCoursePageTitle(board?.title ?? null)

  async function saveRename() {
    if (!board || !titleDraft.trim()) return
    try {
      const updated = await patchBoard(courseCode, board.id, { title: titleDraft.trim() })
      setBoard(updated)
      setRenaming(false)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function archiveBoard() {
    if (!board) return
    if (
      !(await confirm({
        title: t('boards.archive.confirm'),
        confirmLabel: t('boards.archive.confirmLabel'),
        variant: 'danger',
      }))
    ) {
      return
    }
    try {
      await deleteBoard(courseCode, board.id)
      navigate(listBase)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function removePost(postId: string) {
    if (
      !(await confirm({
        title: t('boards.post.deleteConfirm'),
        confirmLabel: t('boards.post.delete'),
        variant: 'danger',
      }))
    ) {
      return
    }
    try {
      await deleteBoardPost(courseCode, boardId, postId)
      setPosts((prev) => prev.filter((p) => p.id !== postId))
      realtime.publishPostDeleted(postId)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  function canManagePost(post: BoardPost): boolean {
    if (canManageBoard) return true
    if (viewerId && post.authorId && viewerId.toLowerCase() === post.authorId.toLowerCase()) {
      return true
    }
    return false
  }

  function canArrangePost(post: BoardPost): boolean {
    if (canManageBoard) return true
    if (board?.layoutLocked) return false
    if (viewerId && post.authorId && viewerId.toLowerCase() === post.authorId.toLowerCase()) {
      return true
    }
    return false
  }

  async function handleArrange(postId: string, input: ArrangeBoardPostInput) {
    realtime.publishArrangement(postId, input)
    const updated = await arrangeBoardPost(courseCode, boardId, postId, input)
    setPosts((prev) => prev.map((p) => (p.id === postId ? updated : p)))
  }

  async function handleChangeLayout(next: BoardLayout) {
    if (!board || next === board.layout) return
    const leavingCanvas = board.layout === 'canvas' && next !== 'canvas'
    if (leavingCanvas) {
      const ok = await confirm({
        title: t('boards.layout.switchConfirm'),
        confirmLabel: t('boards.layout.switchConfirmLabel'),
      })
      if (!ok) return
    }
    try {
      const updated = await patchBoard(courseCode, board.id, { layout: next })
      setBoard(updated)
      setAnnounce(t('boards.layout.changed', { layout: t(`boards.layout.${next}`) }))
      if (next === 'columns') {
        const sectionRows = await listBoardSections(courseCode, boardId)
        setSections(sectionRows)
        const postRows = await listBoardPosts(courseCode, boardId)
        setPosts(postRows)
      }
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleToggleLock() {
    if (!board) return
    try {
      const updated = await patchBoard(courseCode, board.id, { layoutLocked: !board.layoutLocked })
      setBoard(updated)
      setAnnounce(
        updated.layoutLocked ? t('boards.layout.lockedAnnounce') : t('boards.layout.unlockedAnnounce'),
      )
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleReactionMode(mode: BoardReactionMode) {
    if (!board || mode === board.reactionMode) return
    try {
      const updated = await patchBoard(courseCode, board.id, { reactionMode: mode })
      setBoard(updated)
      setAnnounce(t('boards.settings.reactionModeChanged', { mode: t(`boards.settings.mode.${mode}`) }))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  const boardFrozen =
    !!board?.frozenUntil && new Date(board.frozenUntil).getTime() > Date.now()
  const postingBlocked = !!board && !canManageBoard && (board.locked || boardFrozen)

  async function patchModeration(patch: {
    moderationMode?: BoardModerationMode
    filterAction?: BoardFilterAction
    locked?: boolean
    freezeMinutes?: number
    frozenUntil?: string | null
  }) {
    if (!board) return
    try {
      const updated = await patchBoard(courseCode, board.id, patch)
      setBoard(updated)
      if (patch.locked !== undefined) {
        setAnnounce(patch.locked ? t('boards.moderation.lockedAnnounce') : t('boards.moderation.unlockedAnnounce'))
      }
      if (patch.freezeMinutes !== undefined) {
        setAnnounce(t('boards.moderation.frozenAnnounce', { minutes: patch.freezeMinutes }))
      }
      if (patch.frozenUntil === null || patch.frozenUntil === '') {
        setAnnounce(t('boards.moderation.unfrozenAnnounce'))
      }
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  function updatePost(post: BoardPost) {
    setPosts((prev) => prev.map((p) => (p.id === post.id ? post : p)))
  }

  return (
    <>
      {ConfirmDialogHost}
      <div className="sr-only" role="status" aria-live="polite">
        {announce}
      </div>
      <LmsPage title={board?.title ?? t('boards.detail.title')} fillHeight omitHeader>
        {loading ? (
          <div className="flex min-h-48 flex-1 items-center justify-center">
            <span className="text-sm text-slate-500 dark:text-neutral-400">{t('common.loading')}</span>
          </div>
        ) : error ? (
          <div className="rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-400">
            {error}
          </div>
        ) : board ? (
          <div className="flex min-h-48 flex-1 flex-col gap-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0 space-y-2">
                <Link
                  to={listBase}
                  className="inline-flex items-center gap-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
                >
                  <ArrowLeft className="size-4" aria-hidden />
                  {t('boards.detail.back')}
                </Link>
                {renaming ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <input
                      value={titleDraft}
                      onChange={(e) => setTitleDraft(e.target.value)}
                      maxLength={200}
                      className="rounded-md border border-slate-300 px-3 py-1.5 text-lg font-semibold dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                      aria-label={t('boards.create.titleLabel')}
                    />
                    <button
                      type="button"
                      onClick={() => void saveRename()}
                      className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white"
                    >
                      {t('boards.rename.save')}
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setRenaming(false)
                        setTitleDraft(board.title)
                      }}
                      className="rounded-md px-3 py-1.5 text-sm text-slate-600 dark:text-neutral-300"
                    >
                      {t('dialogs.cancel')}
                    </button>
                  </div>
                ) : (
                  <h1 className="truncate text-2xl font-semibold text-slate-900 dark:text-neutral-100">
                    {board.title}
                  </h1>
                )}
                {board.description ? (
                  <p className="text-sm text-slate-600 dark:text-neutral-300">{board.description}</p>
                ) : null}
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {realtimeEnabled ? (
                  <>
                    <BoardSyncStatus connState={realtime.connState} />
                    {realtime.awareness ? <BoardPresenceBar awareness={realtime.awareness} /> : null}
                  </>
                ) : null}
                {postingBlocked ? (
                  <p className="text-sm text-amber-700 dark:text-amber-400" role="status">
                    {board.locked ? t('boards.moderation.lockedBanner') : t('boards.moderation.frozenBanner')}
                  </p>
                ) : (board.capabilities?.canPost ?? board.canPost !== false) ? (
                  <PostComposer
                    courseCode={courseCode}
                    boardId={board.id}
                    onCreated={(post) => {
                      setPosts((prev) => [post, ...prev])
                      realtime.publishPostCreated(post)
                    }}
                  />
                ) : null}
                {canManageBoard ? (
                  <div className="relative">
                    <button
                      type="button"
                      aria-label={t('boards.detail.menuAria')}
                      aria-expanded={menuOpen}
                      onClick={() => setMenuOpen((o) => !o)}
                      className="rounded-md border border-slate-200 p-2 text-slate-600 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800"
                    >
                      <MoreHorizontal className="h-5 w-5" aria-hidden />
                    </button>
                    {menuOpen ? (
                      <div className="absolute end-0 z-10 mt-1 w-48 rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900">
                        <button
                          type="button"
                          className="block w-full px-3 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                          onClick={() => {
                            setMenuOpen(false)
                            setRenaming(true)
                          }}
                        >
                          {t('boards.rename.action')}
                        </button>
                        <button
                          type="button"
                          className="block w-full px-3 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                          onClick={() => {
                            setMenuOpen(false)
                            setShareOpen(true)
                          }}
                        >
                          {t('boards.share.action')}
                        </button>
                        <button
                          type="button"
                          className="block w-full px-3 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                          onClick={() => {
                            setMenuOpen(false)
                            setModerationOpen(true)
                          }}
                        >
                          {t('boards.moderation.queueAction')}
                        </button>
                        <button
                          type="button"
                          className="block w-full px-3 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                          onClick={() => {
                            setMenuOpen(false)
                            setSettingsOpen((o) => !o)
                          }}
                        >
                          {t('boards.settings.action')}
                        </button>
                        <button
                          type="button"
                          className="block w-full px-3 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                          onClick={() => {
                            setMenuOpen(false)
                            setSaveTemplateOpen(true)
                          }}
                        >
                          {t('boards.template.saveAction')}
                        </button>
                        <button
                          type="button"
                          className="block w-full px-3 py-2 text-start text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30"
                          onClick={() => {
                            setMenuOpen(false)
                            void archiveBoard()
                          }}
                        >
                          {t('boards.archive.action')}
                        </button>
                      </div>
                    ) : null}
                  </div>
                ) : null}
              </div>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-3">
              <LayoutSwitcher
                layout={board.layout}
                layoutLocked={board.layoutLocked}
                canManage={canManageBoard}
                onChangeLayout={(layout) => void handleChangeLayout(layout)}
                onToggleLock={() => void handleToggleLock()}
              />
              <SortControls
                value={sortMode}
                onChange={setSortMode}
                hidden={LAYOUTS_HIDING_SORT.includes(board.layout)}
              />
            </div>

            {canManageBoard && settingsOpen ? (
              <div className="space-y-4 rounded-md border border-slate-200 bg-white p-3 dark:border-neutral-700 dark:bg-neutral-900">
                <label className="flex flex-col gap-1 text-sm">
                  <span className="font-medium text-slate-800 dark:text-neutral-100">
                    {t('boards.settings.reactionMode')}
                  </span>
                  <select
                    value={board.reactionMode}
                    onChange={(e) => void handleReactionMode(e.target.value as BoardReactionMode)}
                    className="max-w-xs rounded-md border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                    aria-label={t('boards.settings.reactionMode')}
                  >
                    {(['none', 'like', 'vote', 'star', 'grade'] as BoardReactionMode[]).map((mode) => (
                      <option key={mode} value={mode}>
                        {t(`boards.settings.mode.${mode}`)}
                      </option>
                    ))}
                  </select>
                </label>
                <p className="text-xs text-slate-500 dark:text-neutral-400">
                  {t('boards.settings.assignmentHint')}
                </p>
                {minorsFloor ? (
                  <p className="text-xs text-amber-700 dark:text-amber-400" role="status">
                    {t('boards.moderation.minorsFloor')}
                  </p>
                ) : null}
                <label className="flex flex-col gap-1 text-sm">
                  <span className="font-medium text-slate-800 dark:text-neutral-100">
                    {t('boards.moderation.modeLabel')}
                  </span>
                  <select
                    value={board.moderationMode}
                    disabled={minorsFloor}
                    onChange={(e) =>
                      void patchModeration({ moderationMode: e.target.value as BoardModerationMode })
                    }
                    className="max-w-xs rounded-md border border-slate-300 px-2 py-1.5 text-sm disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800"
                  >
                    <option value="open">{t('boards.moderation.mode.open')}</option>
                    <option value="approval">{t('boards.moderation.mode.approval')}</option>
                  </select>
                </label>
                <label className="flex flex-col gap-1 text-sm">
                  <span className="font-medium text-slate-800 dark:text-neutral-100">
                    {t('boards.moderation.filterLabel')}
                  </span>
                  <select
                    value={board.filterAction}
                    disabled={minorsFloor}
                    onChange={(e) =>
                      void patchModeration({ filterAction: e.target.value as BoardFilterAction })
                    }
                    className="max-w-xs rounded-md border border-slate-300 px-2 py-1.5 text-sm disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800"
                  >
                    <option value="flag">{t('boards.moderation.filter.flag')}</option>
                    <option value="block">{t('boards.moderation.filter.block')}</option>
                  </select>
                </label>
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => void patchModeration({ locked: !board.locked })}
                    className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
                  >
                    {board.locked ? t('boards.moderation.unlock') : t('boards.moderation.lock')}
                  </button>
                  <button
                    type="button"
                    onClick={() => void patchModeration({ freezeMinutes: 5 })}
                    className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
                  >
                    {t('boards.moderation.freeze5')}
                  </button>
                  {boardFrozen ? (
                    <button
                      type="button"
                      onClick={() => void patchModeration({ frozenUntil: '' })}
                      className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
                    >
                      {t('boards.moderation.unfreeze')}
                    </button>
                  ) : null}
                </div>
              </div>
            ) : null}

            <div
              className="flex min-h-64 flex-1 flex-col gap-3 rounded-lg border border-dashed border-slate-300 bg-slate-50 p-3 dark:border-neutral-700 dark:bg-neutral-900/40 sm:p-4"
              role="region"
              aria-label={t('boards.detail.canvasAria')}
            >
              <BoardSurface
                courseCode={courseCode}
                board={board}
                posts={displayPosts}
                sections={sections}
                sortMode={sortMode}
                canManageBoard={canManageBoard}
                canArrangePost={canArrangePost}
                canManagePost={canManagePost}
                onDeletePost={(id) => void removePost(id)}
                onPostUpdate={updatePost}
                onArrange={handleArrange}
                onSectionsChange={setSections}
                onCreateSection={(title) => createBoardSection(courseCode, boardId, title)}
                onDeleteSection={async (sectionId) => {
                  await deleteBoardSection(courseCode, boardId, sectionId)
                  const [sectionRows, postRows] = await Promise.all([
                    listBoardSections(courseCode, boardId),
                    listBoardPosts(courseCode, boardId),
                  ])
                  setSections(sectionRows)
                  setPosts(postRows)
                  setAnnounce(t('boards.section.deleted'))
                }}
                onAnnounce={setAnnounce}
                awareness={realtimeEnabled ? realtime.awareness : null}
                onCursorMove={realtimeEnabled ? realtime.setCursor : undefined}
              />
            </div>
          </div>
        ) : null}
      </LmsPage>
      {board && canManageBoard ? (
        <BoardShareDialog
          open={shareOpen}
          onClose={() => setShareOpen(false)}
          courseCode={courseCode}
          board={board}
          onBoardUpdated={setBoard}
        />
      ) : null}
      {board && canManageBoard ? (
        <SaveAsTemplateDialog
          open={saveTemplateOpen}
          onClose={() => setSaveTemplateOpen(false)}
          courseCode={courseCode}
          board={board}
        />
      ) : null}
      {board && canManageBoard ? (
        <BoardModerationQueue
          open={moderationOpen}
          onClose={() => setModerationOpen(false)}
          courseCode={courseCode}
          boardId={board.id}
          onChanged={() => {
            void listBoardPosts(courseCode, boardId).then(setPosts)
            void fetchBoard(courseCode, boardId).then((row) => {
              setBoard(row)
            })
          }}
        />
      ) : null}
    </>
  )
}

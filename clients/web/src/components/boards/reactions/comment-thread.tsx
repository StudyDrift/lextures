import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  createBoardPostComment,
  deleteBoardPostComment,
  listBoardPostComments,
  patchBoardPostComment,
  type BoardComment,
} from '../../../lib/boards-api'
import { toastMutationError } from '../../../lib/lms-toast'
import { getJwtSubject } from '../../../lib/auth'

type CommentThreadProps = {
  courseCode: string
  boardId: string
  postId: string
  canManageBoard: boolean
  canInteract: boolean
  onCountChange?: (delta: number) => void
}

function bodyText(c: BoardComment): string {
  if (!c.body) return ''
  if (typeof c.body === 'string') return c.body
  return c.body.text || c.body.html?.replace(/<[^>]+>/g, ' ') || ''
}

function nestComments(comments: BoardComment[]): { comment: BoardComment; children: BoardComment[] }[] {
  const byParent = new Map<string | null, BoardComment[]>()
  for (const c of comments) {
    const key = c.parentId ?? null
    const list = byParent.get(key) ?? []
    list.push(c)
    byParent.set(key, list)
  }
  const roots = byParent.get(null) ?? []
  return roots.map((comment) => ({
    comment,
    children: byParent.get(comment.id) ?? [],
  }))
}

export function CommentThread({
  courseCode,
  boardId,
  postId,
  canManageBoard,
  canInteract,
  onCountChange,
}: CommentThreadProps) {
  const { t } = useTranslation('common')
  const viewerId = getJwtSubject()
  const [comments, setComments] = useState<BoardComment[]>([])
  const [loading, setLoading] = useState(true)
  const [draft, setDraft] = useState('')
  const [replyTo, setReplyTo] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    void listBoardPostComments(courseCode, boardId, postId)
      .then((rows) => {
        if (!cancelled) setComments(rows)
      })
      .catch((err) => {
        if (!cancelled) toastMutationError(err instanceof Error ? err.message : String(err))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [boardId, courseCode, postId])

  async function submit() {
    const text = draft.trim()
    if (!text || !canInteract || busy) return
    setBusy(true)
    try {
      const created = await createBoardPostComment(courseCode, boardId, postId, {
        body: { text, html: `<p>${escapeHtml(text)}</p>` },
        parentId: replyTo ?? undefined,
      })
      setComments((prev) => [...prev, created])
      setDraft('')
      setReplyTo(null)
      onCountChange?.(1)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  async function hideComment(id: string) {
    if (!canManageBoard || busy) return
    setBusy(true)
    try {
      const updated = await patchBoardPostComment(courseCode, boardId, postId, id, { hidden: true })
      setComments((prev) => prev.map((c) => (c.id === id ? updated : c)))
      onCountChange?.(-1)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  async function removeComment(id: string) {
    if (busy) return
    setBusy(true)
    try {
      await deleteBoardPostComment(courseCode, boardId, postId, id)
      setComments((prev) =>
        prev.map((c) => (c.id === id ? { ...c, hidden: true } : c)),
      )
      onCountChange?.(-1)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const visible = canManageBoard ? comments : comments.filter((c) => !c.hidden)
  const nested = nestComments(visible)

  return (
    <div className="mt-2 space-y-2 border-t border-slate-100 pt-2 dark:border-neutral-800">
      <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        {t('boards.comment.threadHeading')}
      </h4>
      {loading ? (
        <p className="text-xs text-slate-500">{t('common.loading')}</p>
      ) : nested.length === 0 ? (
        <p className="text-xs text-slate-500 dark:text-neutral-400">{t('boards.comment.empty')}</p>
      ) : (
        <ul className="space-y-2" aria-label={t('boards.comment.threadAria')}>
          {nested.map(({ comment, children }) => (
            <li key={comment.id}>
              <CommentRow
                comment={comment}
                viewerId={viewerId}
                canManageBoard={canManageBoard}
                onReply={() => setReplyTo(comment.id)}
                onHide={() => void hideComment(comment.id)}
                onDelete={() => void removeComment(comment.id)}
              />
              {children.length > 0 ? (
                <ul className="ms-4 mt-2 space-y-2 border-s border-slate-200 ps-3 dark:border-neutral-700">
                  {children.map((child) => (
                    <li key={child.id}>
                      <CommentRow
                        comment={child}
                        viewerId={viewerId}
                        canManageBoard={canManageBoard}
                        onReply={() => setReplyTo(child.id)}
                        onHide={() => void hideComment(child.id)}
                        onDelete={() => void removeComment(child.id)}
                      />
                    </li>
                  ))}
                </ul>
              ) : null}
            </li>
          ))}
        </ul>
      )}

      {canInteract ? (
        <div className="space-y-1">
          {replyTo ? (
            <p className="text-xs text-slate-500">
              {t('boards.comment.replying')}{' '}
              <button
                type="button"
                className="text-indigo-600 underline dark:text-indigo-400"
                onClick={() => setReplyTo(null)}
              >
                {t('boards.comment.cancelReply')}
              </button>
            </p>
          ) : null}
          <textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            rows={2}
            maxLength={4000}
            placeholder={t('boards.comment.placeholder')}
            className="w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
            aria-label={t('boards.comment.add')}
          />
          <button
            type="button"
            disabled={busy || !draft.trim()}
            onClick={() => void submit()}
            className="rounded-md bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
          >
            {t('boards.comment.submit')}
          </button>
        </div>
      ) : null}
    </div>
  )
}

function CommentRow({
  comment,
  viewerId,
  canManageBoard,
  onReply,
  onHide,
  onDelete,
}: {
  comment: BoardComment
  viewerId: string | null
  canManageBoard: boolean
  onReply: () => void
  onHide: () => void
  onDelete: () => void
}) {
  const { t } = useTranslation('common')
  const isAuthor =
    !!viewerId && !!comment.authorId && viewerId.toLowerCase() === comment.authorId.toLowerCase()

  if (comment.hidden && canManageBoard) {
    return (
      <article className="rounded bg-slate-50 px-2 py-1.5 text-xs italic text-slate-500 dark:bg-neutral-800/60 dark:text-neutral-400">
        {t('boards.comment.hiddenPlaceholder')}
      </article>
    )
  }

  return (
    <article className="rounded-md bg-slate-50 px-2 py-1.5 text-sm dark:bg-neutral-800/50">
      <p className="whitespace-pre-wrap text-slate-700 dark:text-neutral-200">{bodyText(comment)}</p>
      <div className="mt-1 flex flex-wrap gap-2 text-xs">
        <button type="button" className="text-indigo-600 dark:text-indigo-400" onClick={onReply}>
          {t('boards.comment.reply')}
        </button>
        {isAuthor ? (
          <button type="button" className="text-red-600 dark:text-red-400" onClick={onDelete}>
            {t('boards.comment.delete')}
          </button>
        ) : null}
        {canManageBoard ? (
          <button type="button" className="text-amber-700 dark:text-amber-400" onClick={onHide}>
            {t('boards.comment.hide')}
          </button>
        ) : null}
      </div>
    </article>
  )
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

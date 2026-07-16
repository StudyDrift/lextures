import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Heart, Star, ArrowBigUp } from 'lucide-react'
import {
  applyReactionResult,
  putBoardPostReaction,
  syncBoardPostGrade,
  type BoardPost,
  type BoardReactionMode,
} from '../../../lib/boards-api'
import { toastMutationError } from '../../../lib/lms-toast'

type ReactionControlProps = {
  courseCode: string
  boardId: string
  post: BoardPost
  reactionMode: BoardReactionMode
  canInteract: boolean
  canGrade: boolean
  assignmentLinked: boolean
  onPostUpdate: (post: BoardPost) => void
  onAnnounce?: (message: string) => void
}

export function ReactionControl({
  courseCode,
  boardId,
  post,
  reactionMode,
  canInteract,
  canGrade,
  assignmentLinked,
  onPostUpdate,
  onAnnounce,
}: ReactionControlProps) {
  const { t } = useTranslation('common')
  const [busy, setBusy] = useState(false)

  if (reactionMode === 'none') return null

  async function toggleLikeOrVote(kind: 'like' | 'vote') {
    if (!canInteract || busy) return
    setBusy(true)
    try {
      const result = await putBoardPostReaction(courseCode, boardId, post.id, { kind })
      onPostUpdate(applyReactionResult(post, result))
      onAnnounce?.(result.active ? t(`boards.react.${kind}On`) : t(`boards.react.${kind}Off`))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  async function setStars(value: number) {
    if (!canInteract || busy) return
    setBusy(true)
    try {
      const result = await putBoardPostReaction(courseCode, boardId, post.id, {
        kind: 'star',
        value,
      })
      onPostUpdate(applyReactionResult(post, result))
      onAnnounce?.(t('boards.react.starSet', { value }))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  async function setGrade(raw: string) {
    if (!canGrade || busy) return
    const value = Number(raw)
    if (!Number.isFinite(value)) return
    setBusy(true)
    try {
      const result = await putBoardPostReaction(courseCode, boardId, post.id, {
        kind: 'grade',
        value,
      })
      onPostUpdate(applyReactionResult(post, result))
      onAnnounce?.(t('boards.react.gradeSet', { value }))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  async function sendToGradebook() {
    if (!canGrade || !assignmentLinked || busy) return
    setBusy(true)
    try {
      const result = await syncBoardPostGrade(courseCode, boardId, post.id)
      onAnnounce?.(t('boards.react.gradeSynced', { value: result.pointsEarned }))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const pressed = !!post.myReaction
  const count = post.reactionCount ?? 0

  switch (reactionMode) {
    case 'like':
      return (
        <button
          type="button"
          disabled={!canInteract || busy}
          aria-pressed={pressed}
          aria-label={pressed ? t('boards.react.unlike') : t('boards.react.like')}
          onClick={() => void toggleLikeOrVote('like')}
          className="inline-flex min-h-9 items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
        >
          <Heart className={`size-4 ${pressed ? 'fill-rose-500 text-rose-500' : ''}`} aria-hidden />
          {count > 0 ? <span className="tabular-nums">{count}</span> : null}
        </button>
      )
    case 'vote':
      return (
        <button
          type="button"
          disabled={!canInteract || busy}
          aria-pressed={pressed}
          aria-label={pressed ? t('boards.react.unvote') : t('boards.react.vote')}
          onClick={() => void toggleLikeOrVote('vote')}
          className="inline-flex min-h-9 items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
        >
          <ArrowBigUp
            className={`size-4 ${pressed ? 'fill-indigo-500 text-indigo-500' : ''}`}
            aria-hidden
          />
          <span className="tabular-nums">{count}</span>
        </button>
      )
    case 'star': {
      const mine = post.myReaction?.value ?? 0
      return (
        <div className="flex flex-wrap items-center gap-1" role="group" aria-label={t('boards.react.starLabel')}>
          {[1, 2, 3, 4, 5].map((n) => (
            <button
              key={n}
              type="button"
              disabled={!canInteract || busy}
              aria-label={t('boards.react.starN', { value: n })}
              aria-pressed={mine === n}
              onClick={() => void setStars(n)}
              className="rounded p-1 text-amber-500 hover:bg-amber-50 disabled:opacity-50 dark:hover:bg-amber-950/30"
            >
              <Star className={`size-4 ${mine >= n ? 'fill-current' : ''}`} aria-hidden />
            </button>
          ))}
          {post.avgStars != null ? (
            <span className="ms-1 text-xs tabular-nums text-slate-500 dark:text-neutral-400">
              {t('boards.react.avgStars', { avg: post.avgStars.toFixed(1), count })}
            </span>
          ) : null}
        </div>
      )
    }
    case 'grade':
      return (
        <div className="flex flex-wrap items-center gap-2">
          {canGrade ? (
            <>
              <label className="flex items-center gap-1 text-xs text-slate-600 dark:text-neutral-300">
                <span>{t('boards.react.grade')}</span>
                <input
                  type="number"
                  inputMode="decimal"
                  defaultValue={post.grade ?? post.myReaction?.value ?? ''}
                  disabled={busy}
                  className="w-16 rounded border border-slate-300 px-1.5 py-1 text-xs dark:border-neutral-600 dark:bg-neutral-800"
                  aria-label={t('boards.react.gradeInput')}
                  onBlur={(e) => {
                    if (e.target.value !== '') void setGrade(e.target.value)
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      void setGrade((e.target as HTMLInputElement).value)
                    }
                  }}
                />
              </label>
              {assignmentLinked ? (
                <button
                  type="button"
                  disabled={busy || post.grade == null}
                  onClick={() => void sendToGradebook()}
                  className="rounded-md border border-slate-200 px-2 py-1 text-xs font-medium text-indigo-600 hover:bg-indigo-50 disabled:opacity-50 dark:border-neutral-700 dark:text-indigo-400 dark:hover:bg-indigo-950/30"
                >
                  {t('boards.react.sendGradebook')}
                </button>
              ) : null}
            </>
          ) : post.grade != null ? (
            <span className="text-xs font-medium text-slate-700 dark:text-neutral-200">
              {t('boards.react.yourGrade', { value: post.grade })}
            </span>
          ) : null}
        </div>
      )
    default: {
      const _exhaustive: never = reactionMode
      return _exhaustive
    }
  }
}

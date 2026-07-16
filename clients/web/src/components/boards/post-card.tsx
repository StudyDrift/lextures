import { useTranslation } from 'react-i18next'
import { FileText, Flag, Link2, AlertTriangle, Loader2, MessageSquare, Trash2 } from 'lucide-react'
import {
  type BoardPost,
  type BoardReactionMode,
  videoEmbedFromUrl,
} from '../../lib/boards-api'
import { parseWhiteboardElements } from '../../lib/whiteboard/serialize'
import type { DrawEl } from '../../lib/whiteboard/types'
import { useEffect, useRef, useState } from 'react'
import { ReactionControl } from './reactions/reaction-control'
import { CommentThread } from './reactions/comment-thread'
import { BoardReportDialog } from './report-dialog'

type PostCardProps = {
  post: BoardPost
  courseCode?: string
  boardId?: string
  reactionMode?: BoardReactionMode
  canManage: boolean
  canManageBoard?: boolean
  canInteract?: boolean
  assignmentLinked?: boolean
  onDelete?: (postId: string) => void
  onPostUpdate?: (post: BoardPost) => void
  onAnnounce?: (message: string) => void
}

function bodyPlain(post: BoardPost): string {
  if (!post.body) return ''
  if (typeof post.body === 'string') return post.body
  return post.body.text || post.body.html?.replace(/<[^>]+>/g, ' ') || ''
}

function DrawingThumb({ data }: { data: unknown }) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const els = parseWhiteboardElements(data)
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    ctx.fillStyle = '#f8fafc'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    for (const el of els) drawEl(ctx, el)
  }, [data])
  return (
    <canvas
      ref={canvasRef}
      width={280}
      height={160}
      className="w-full rounded border border-slate-200 dark:border-neutral-700"
      aria-hidden
    />
  )
}

function drawEl(ctx: CanvasRenderingContext2D, el: DrawEl) {
  ctx.strokeStyle = el.color
  ctx.lineWidth = el.width
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  switch (el.type) {
    case 'stroke': {
      if (el.pts.length < 2) return
      ctx.beginPath()
      ctx.moveTo(el.pts[0][0], el.pts[0][1])
      for (let i = 1; i < el.pts.length; i++) ctx.lineTo(el.pts[i][0], el.pts[i][1])
      ctx.stroke()
      return
    }
    case 'rect':
      ctx.strokeRect(el.x, el.y, el.w, el.h)
      return
    case 'circle':
      ctx.beginPath()
      ctx.ellipse(el.cx, el.cy, el.rx, el.ry, 0, 0, Math.PI * 2)
      ctx.stroke()
      return
    case 'triangle':
      ctx.beginPath()
      ctx.moveTo(el.x1, el.y1)
      ctx.lineTo(el.x2, el.y2)
      ctx.lineTo(el.x3, el.y3)
      ctx.closePath()
      ctx.stroke()
      return
    case 'line':
      ctx.beginPath()
      ctx.moveTo(el.x1, el.y1)
      ctx.lineTo(el.x2, el.y2)
      ctx.stroke()
      return
    default: {
      const _exhaustive: never = el
      return _exhaustive
    }
  }
}

export function PostCard({
  post,
  courseCode,
  boardId,
  reactionMode = 'none',
  canManage,
  canManageBoard = false,
  canInteract = true,
  assignmentLinked = false,
  onDelete,
  onPostUpdate,
  onAnnounce,
}: PostCardProps) {
  const { t } = useTranslation('common')
  const [commentsOpen, setCommentsOpen] = useState(false)
  const [reportOpen, setReportOpen] = useState(false)
  const att = post.attachment
  const scan = att?.scanStatus
  const embed = post.linkUrl ? videoEmbedFromUrl(post.linkUrl) : null
  const showEngagement = !!courseCode && !!boardId && !!onPostUpdate
  const isPending = post.status === 'pending'
  const isRemoved = !!post.removed || (!!post.hidden && !canManageBoard)

  return (
    <article
      id={`board-post-${post.id}`}
      className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-white p-3 shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
      tabIndex={-1}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          {post.title ? (
            <h3 className="truncate text-sm font-semibold text-slate-900 dark:text-neutral-100">
              {post.title}
            </h3>
          ) : null}
          <p className="text-xs text-slate-500 dark:text-neutral-400">
            {t(`boards.post.type.${post.contentType}`)}
          </p>
          {isPending ? (
            <p className="mt-1 text-xs font-medium text-amber-700 dark:text-amber-400" role="status">
              {t('boards.moderation.pendingBadge')}
            </p>
          ) : null}
        </div>
        <div className="flex shrink-0 items-center gap-1">
          {courseCode && boardId && !canManageBoard ? (
            <button
              type="button"
              onClick={() => setReportOpen(true)}
              className="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-neutral-800"
              aria-label={t('boards.report.action')}
            >
              <Flag className="size-4" aria-hidden />
            </button>
          ) : null}
          {canManage && onDelete ? (
            <button
              type="button"
              onClick={() => onDelete(post.id)}
              className="rounded p-1 text-slate-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/30"
              aria-label={t('boards.post.delete')}
            >
              <Trash2 className="size-4" aria-hidden />
            </button>
          ) : null}
        </div>
      </div>

      {isRemoved ? (
        <p className="text-sm italic text-slate-500 dark:text-neutral-400">
          {t('boards.moderation.removedPlaceholder')}
        </p>
      ) : null}

      {!isRemoved && post.contentType === 'text' ? (
        post.body?.html ? (
          <div
            className="prose prose-sm max-w-none dark:prose-invert"
            dangerouslySetInnerHTML={{ __html: post.body.html }}
          />
        ) : (
          <p className="whitespace-pre-wrap text-sm text-slate-700 dark:text-neutral-200">
            {bodyPlain(post)}
          </p>
        )
      ) : null}

      {!isRemoved &&
      (post.contentType === 'image' || post.contentType === 'file' || post.contentType === 'video' || post.contentType === 'audio') &&
      att ? (
        scan === 'pending' ? (
          <div className="flex items-center gap-2 text-sm text-slate-500">
            <Loader2 className="size-4 motion-safe:animate-spin" aria-hidden />
            {t('boards.post.scanning')}
          </div>
        ) : scan === 'blocked' ? (
          <div className="flex items-center gap-2 text-sm text-amber-700 dark:text-amber-400">
            <AlertTriangle className="size-4" aria-hidden />
            {t('boards.post.blocked')}
          </div>
        ) : post.contentType === 'image' && att.url ? (
          <img
            src={att.url}
            alt={att.altText || post.title || t('boards.post.imageAltFallback')}
            className="max-h-64 w-full rounded object-contain"
          />
        ) : post.contentType === 'audio' && att.url ? (
          <audio controls src={att.url} className="w-full" preload="metadata">
            <track kind="captions" />
          </audio>
        ) : post.contentType === 'video' && att.url ? (
          <video controls src={att.url} className="max-h-64 w-full rounded" preload="metadata">
            <track kind="captions" />
          </video>
        ) : att.url ? (
          <a
            href={att.url}
            className="inline-flex items-center gap-2 text-sm font-medium text-indigo-600 dark:text-indigo-400"
            download={att.fileName}
          >
            <FileText className="size-4" aria-hidden />
            {att.fileName}
          </a>
        ) : null
      ) : null}

      {!isRemoved && (post.contentType === 'link' || post.contentType === 'video') && post.linkUrl ? (
        embed ? (
          <div className="aspect-video overflow-hidden rounded bg-black">
            <iframe
              title={post.linkPreview?.title || post.title || t('boards.post.videoEmbed')}
              src={
                embed.provider === 'youtube'
                  ? `https://www.youtube.com/embed/${embed.id}`
                  : `https://player.vimeo.com/video/${embed.id}`
              }
              className="h-full w-full border-0"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
              allowFullScreen
            />
          </div>
        ) : post.linkPreview?.title || post.linkPreview?.description ? (
          <a
            href={post.linkUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="flex gap-3 rounded border border-slate-200 p-2 hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
          >
            {post.linkPreview.image ? (
              <img
                src={post.linkPreview.image}
                alt=""
                className="h-16 w-16 shrink-0 rounded object-cover"
              />
            ) : (
              <Link2 className="size-8 shrink-0 text-slate-400" aria-hidden />
            )}
            <span className="min-w-0 text-sm">
              <span className="block font-medium text-slate-900 dark:text-neutral-100">
                {post.linkPreview.title || post.linkUrl}
              </span>
              {post.linkPreview.description ? (
                <span className="mt-0.5 line-clamp-2 text-slate-500 dark:text-neutral-400">
                  {post.linkPreview.description}
                </span>
              ) : null}
            </span>
          </a>
        ) : (
          <a
            href={post.linkUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="break-all text-sm text-indigo-600 dark:text-indigo-400"
          >
            {post.linkUrl}
          </a>
        )
      ) : null}

      {!isRemoved && post.contentType === 'drawing' ? <DrawingThumb data={post.drawingData} /> : null}

      {!isRemoved && showEngagement && courseCode && boardId && onPostUpdate ? (
        <div className="mt-1 flex flex-col gap-1 border-t border-slate-100 pt-2 dark:border-neutral-800">
          <div className="flex flex-wrap items-center gap-2">
            <ReactionControl
              courseCode={courseCode}
              boardId={boardId}
              post={post}
              reactionMode={reactionMode}
              canInteract={canInteract}
              canGrade={canManageBoard}
              assignmentLinked={assignmentLinked}
              onPostUpdate={onPostUpdate}
              onAnnounce={onAnnounce}
            />
            <button
              type="button"
              aria-expanded={commentsOpen}
              aria-controls={`board-post-comments-${post.id}`}
              onClick={() => setCommentsOpen((o) => !o)}
              className="inline-flex min-h-9 items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
            >
              <MessageSquare className="size-4" aria-hidden />
              <span className="tabular-nums">{post.commentCount ?? 0}</span>
              <span className="sr-only">{t('boards.comment.toggle')}</span>
            </button>
          </div>
          {commentsOpen ? (
            <div id={`board-post-comments-${post.id}`}>
              <CommentThread
                courseCode={courseCode}
                boardId={boardId}
                postId={post.id}
                canManageBoard={canManageBoard}
                canInteract={canInteract}
                onCountChange={(delta) =>
                  onPostUpdate({
                    ...post,
                    commentCount: Math.max(0, (post.commentCount ?? 0) + delta),
                  })
                }
              />
            </div>
          ) : null}
        </div>
      ) : null}
      {courseCode && boardId ? (
        <BoardReportDialog
          open={reportOpen}
          onClose={() => setReportOpen(false)}
          courseCode={courseCode}
          boardId={boardId}
          postId={post.id}
        />
      ) : null}
    </article>
  )
}

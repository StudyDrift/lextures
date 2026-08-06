import { EnrollmentAvatar } from '../enrollment/enrollment-avatar'
import { formatAbsolute, formatRelativeCompact } from '../../lib/format-datetime'
import type { GradeCommentApi } from '../../lib/courses-api'

export type CommentRosterPerson = {
  userId: string
  displayName: string | null
  avatarUrl?: string | null
}

function authorLabel(c: GradeCommentApi, roster?: Map<string, CommentRosterPerson>): string {
  const fromComment = c.displayName?.trim()
  if (fromComment) return fromComment
  const uid = c.userId?.trim().toLowerCase()
  if (uid && roster?.has(uid)) {
    const p = roster.get(uid)!
    return p.displayName?.trim() || '—'
  }
  return '—'
}

function resolveAvatar(
  c: GradeCommentApi,
  roster?: Map<string, CommentRosterPerson>,
): string | null | undefined {
  if (c.avatarUrl?.trim()) return c.avatarUrl
  const uid = c.userId?.trim().toLowerCase()
  if (uid && roster?.has(uid)) return roster.get(uid)?.avatarUrl
  return null
}

function resolveUserId(c: GradeCommentApi): string {
  const uid = c.userId?.trim()
  if (uid) return uid
  return `comment-${(c.id ?? c.displayName ?? c.body).toLowerCase()}`
}

type SubmissionCommentThreadProps = {
  comments: GradeCommentApi[]
  roster?: Map<string, CommentRosterPerson>
  emptyLabel?: string
}

export function SubmissionCommentThread({
  comments,
  roster,
  emptyLabel = 'No feedback yet. Add a comment when you save the grade.',
}: SubmissionCommentThreadProps) {
  if (comments.length === 0) {
    return (
      <p className="rounded-xl border border-dashed border-border-default bg-slate-50/80 px-4 py-6 text-center text-sm text-fg-muted dark:border-border-default/30 dark:text-fg-muted">
        {emptyLabel}
      </p>
    )
  }

  return (
    <div
      className="space-y-1 rounded-xl border border-border-default bg-surface-raised p-3 dark:border-border-default/40"
      role="log"
      aria-label="Grade feedback conversation"
    >
      {comments.map((c, index) => {
        const name = authorLabel(c, roster)
        const userId = resolveUserId(c)
        const avatarUrl = resolveAvatar(c, roster)
        const createdAt = c.createdAt?.trim() ?? ''
        const showTimestamp = Boolean(createdAt)
        const isCompact = index > 0

        return (
          <article
            key={c.id ?? `${userId}-${index}`}
            className={`group/commentrow flex gap-2.5 rounded-lg px-1 py-1.5 hover:bg-slate-50/90 dark:hover:bg-neutral-900/50 ${ isCompact ? 'mt-0.5' : '' }`}
          >
            <EnrollmentAvatar
              userId={userId}
              name={name}
              avatarUrl={avatarUrl}
              size="sm"
            />
            <div className="min-w-0 flex-1 pt-0.5">
              <div className="flex items-baseline gap-x-1.5 leading-none">
                <span className="truncate text-[13px] font-semibold text-fg-default">
                  {name}
                </span>
                {showTimestamp ? (
                  <time
                    className="shrink-0 text-[10px] font-medium text-neutral-500 dark:text-fg-muted"
                    dateTime={createdAt}
                    title={formatAbsolute(createdAt)}
                  >
                    {formatRelativeCompact(createdAt)}
                  </time>
                ) : null}
                {c.source === 'rubric' ? (
                  <span className="rounded bg-violet-100/90 px-1 py-px text-[9px] font-medium text-violet-800 dark:bg-violet-950/50 dark:text-violet-200">
                    Rubric
                  </span>
                ) : null}
              </div>
              <p className="mt-1 whitespace-pre-wrap break-words text-[0.8125rem] leading-relaxed text-fg-default">
                {c.body}
              </p>
            </div>
          </article>
        )
      })}
    </div>
  )
}
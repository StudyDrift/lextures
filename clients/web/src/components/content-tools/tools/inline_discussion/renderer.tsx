import { useEffect, useId, useRef, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'

type ThreadPost = {
  id: string
  parentPostId?: string | null
  text: string
  authorDisplay?: string
  authorId?: string
  anonymous?: boolean
  upvoteCount?: number
  viewerUpvoted?: boolean
  createdAt?: string
  endorsed?: boolean
  removed?: boolean
  tombstone?: boolean
  editedAt?: string
  isOwn?: boolean
  canEdit?: boolean
  canDelete?: boolean
}

type ThreadResult = {
  prompt?: string
  posts?: ThreadPost[]
  canSeePeers?: boolean
  locked?: boolean
  lockReason?: string
  total?: number
  page?: number
  pageSize?: number
  anonymity?: string
  requirements?: {
    requiredPosts?: number
    requiredReplies?: number
    myPosts?: number
    myReplies?: number
    completed?: boolean
  }
  error?: string
  message?: string
  preserveInput?: boolean
  crisis?: boolean
  post?: ThreadPost
  state?: Record<string, unknown>
}

function asPosts(raw: unknown): ThreadPost[] {
  if (!Array.isArray(raw)) return []
  return raw.filter((p): p is ThreadPost => {
    if (!p || typeof p !== 'object') return false
    return typeof (p as ThreadPost).id === 'string'
  })
}

function nestPosts(posts: ThreadPost[]): Array<ThreadPost & { replies: ThreadPost[] }> {
  const roots: Array<ThreadPost & { replies: ThreadPost[] }> = []
  const byId = new Map<string, ThreadPost & { replies: ThreadPost[] }>()
  for (const p of posts) {
    byId.set(p.id, { ...p, replies: [] })
  }
  for (const p of posts) {
    const node = byId.get(p.id)!
    const parent = p.parentPostId ? byId.get(p.parentPostId) : undefined
    if (parent) {
      parent.replies.push(node)
    } else if (!p.parentPostId) {
      roots.push(node)
    } else {
      roots.push(node)
    }
  }
  return roots
}

export default function InlineDiscussionRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const promptId = useId()
  const composerId = useId()
  const liveId = useId()
  const composerRef = useRef<HTMLTextAreaElement>(null)
  const [draft, setDraft] = useState(() => (typeof state.draft === 'string' ? state.draft : ''))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [thread, setThread] = useState<ThreadResult | null>(null)
  const [replyTo, setReplyTo] = useState<string | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const draftTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const prompt = typeof config.prompt === 'string' ? config.prompt : ''
  const postBeforeYouSee = config.postBeforeYouSee !== false
  const anonymity =
    typeof config.anonymity === 'string' ? config.anonymity : 'named'
  const requiredPosts =
    typeof config.requiredPosts === 'number' ? config.requiredPosts : 1
  const requiredReplies =
    typeof config.requiredReplies === 'number' ? config.requiredReplies : 0
  const allowReplies = config.allowReplies !== false

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = (await runAction('thread', { page: 1 })) as ThreadResult
        if (!cancelled) setThread(res)
      } catch {
        if (!cancelled) setError(t('contentTools.tools.inline_discussion.loadError'))
      }
    })()
    return () => {
      cancelled = true
    }
  }, [runAction, t])

  useEffect(() => {
    return () => {
      if (draftTimer.current) clearTimeout(draftTimer.current)
    }
  }, [])

  function onDraftChange(value: string) {
    setDraft(value)
    if (draftTimer.current) clearTimeout(draftTimer.current)
    draftTimer.current = setTimeout(() => {
      void save({ draft: value })
    }, 800)
  }

  async function submitPost() {
    if (readOnly || busy) return
    const text = draft.trim()
    if (!text) return
    setBusy(true)
    setError(null)
    try {
      const input: Record<string, unknown> = {
        text,
        idempotencyKey: crypto.randomUUID(),
      }
      if (replyTo) input.parentPostId = replyTo
      const res = (await runAction(editingId ? 'edit' : 'post', editingId
        ? { postId: editingId, text }
        : input)) as ThreadResult
      if (res.error) {
        setError(res.message || res.error)
        if (!res.preserveInput) setDraft('')
        return
      }
      setDraft('')
      setReplyTo(null)
      setEditingId(null)
      void save({ draft: '' })
      announce(t('contentTools.tools.inline_discussion.postAnnounced'))
      composerRef.current?.focus()
      const refreshed = (await runAction('thread', { page: 1 })) as ThreadResult
      setThread(refreshed)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.inline_discussion.postError'))
    } finally {
      setBusy(false)
    }
  }

  async function onReport(postId: string) {
    try {
      await runAction('report', { postId, category: 'inappropriate' })
      announce(t('contentTools.tools.inline_discussion.reportThanks'))
    } catch {
      setError(t('contentTools.tools.inline_discussion.reportError'))
    }
  }

  async function onDelete(postId: string) {
    try {
      await runAction('delete', { postId })
      const refreshed = (await runAction('thread', { page: 1 })) as ThreadResult
      setThread(refreshed)
    } catch {
      setError(t('contentTools.tools.inline_discussion.deleteError'))
    }
  }

  async function onUpvote(postId: string) {
    try {
      await runAction('upvote', { postId })
      const refreshed = (await runAction('thread', { page: 1 })) as ThreadResult
      setThread(refreshed)
    } catch {
      /* ignore */
    }
  }

  const posts = nestPosts(asPosts(thread?.posts))
  const locked = Boolean(thread?.locked)
  const req = thread?.requirements
  const myPosts = req?.myPosts ?? (Array.isArray(state.myPostIds) ? state.myPostIds.length : 0)
  const myReplies = req?.myReplies ?? (Array.isArray(state.myReplyIds) ? state.myReplyIds.length : 0)

  return (
    <section
      className="space-y-4 rounded-lg border border-border-default bg-surface-raised p-4 dark:border-border-default dark:bg-surface-raised"
      data-testid="inline-discussion"
      data-content-tool="inline_discussion"
      aria-labelledby={promptId}
    >
      <header className="space-y-1">
        <h2
          id={promptId}
          className="text-base font-semibold text-fg-default"
        >
          {t('contentTools.tools.inline_discussion.label')}
        </h2>
        <p className="text-sm text-fg-default whitespace-pre-wrap">{prompt}</p>
        <p className="text-xs text-fg-muted">
          {t('contentTools.tools.inline_discussion.requirements', {
            posts: String(requiredPosts),
            replies: String(requiredReplies),
          })}
        </p>
        <p className="text-xs text-fg-muted" data-testid="inline-discussion-progress">
          {t('contentTools.tools.inline_discussion.progress', {
            posts: String(myPosts),
            replies: String(myReplies),
          })}
        </p>
      </header>

      {!readOnly ? (
        <div className="space-y-2">
          <label htmlFor={composerId} className="block text-sm font-medium text-fg-default">
            {replyTo
              ? t('contentTools.tools.inline_discussion.replyLabel')
              : editingId
                ? t('contentTools.tools.inline_discussion.editLabel')
                : t('contentTools.tools.inline_discussion.composerLabel')}
          </label>
          {anonymity === 'anonymous_to_peers' ? (
            <p className="text-xs text-amber-800 dark:text-amber-200" data-testid="inline-discussion-anonymity-note">
              {t('contentTools.tools.inline_discussion.anonymityNote')}
            </p>
          ) : null}
          <p className="text-xs text-fg-muted">
            {t('contentTools.tools.inline_discussion.moderationNote')}
          </p>
          <textarea
            id={composerId}
            ref={composerRef}
            data-testid="inline-discussion-composer"
            className="w-full rounded-md border border-border-default bg-surface-raised px-3 py-2 text-sm dark:border-border-default dark:bg-surface-base"
            rows={3}
            value={draft}
            disabled={busy}
            onChange={(e) => onDraftChange(e.target.value)}
          />
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              data-testid="inline-discussion-submit"
              className="rounded-md bg-slate-900 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
              disabled={busy || !draft.trim()}
              onClick={() => void submitPost()}
            >
              {busy
                ? t('contentTools.tools.inline_discussion.submitting')
                : editingId
                  ? t('contentTools.tools.inline_discussion.saveEdit')
                  : replyTo
                    ? t('contentTools.tools.inline_discussion.submitReply')
                    : t('contentTools.tools.inline_discussion.submitPost')}
            </button>
            {replyTo || editingId ? (
              <button
                type="button"
                className="text-sm text-fg-muted underline dark:text-fg-muted"
                onClick={() => {
                  setReplyTo(null)
                  setEditingId(null)
                }}
              >
                {t('contentTools.tools.inline_discussion.cancel')}
              </button>
            ) : null}
          </div>
        </div>
      ) : (
        <p className="text-sm text-fg-muted">
          {t('contentTools.tools.inline_discussion.readOnly')}
        </p>
      )}

      <div id={liveId} className="sr-only" aria-live="polite" />

      {error ? (
        <p role="alert" className="text-sm text-rose-700 dark:text-rose-300" data-testid="inline-discussion-error">
          {error}
        </p>
      ) : null}

      {locked && postBeforeYouSee ? (
        <p
          className="rounded-md bg-surface-base px-3 py-2 text-sm text-fg-muted dark:bg-surface-overlay dark:text-fg-default"
          data-testid="inline-discussion-locked"
        >
          {t('contentTools.tools.inline_discussion.lockedHint')}
        </p>
      ) : null}

      {!locked && posts.length === 0 ? (
        <p className="text-sm text-fg-muted" data-testid="inline-discussion-empty">
          {t('contentTools.tools.inline_discussion.empty')}
        </p>
      ) : null}

      <ol className="space-y-3" data-testid="inline-discussion-thread">
        {posts.map((post) => (
          <PostArticle
            key={post.id}
            post={post}
            depth={0}
            allowReplies={allowReplies && !readOnly}
            readOnly={readOnly}
            t={t}
            onReply={(id) => {
              setReplyTo(id)
              setEditingId(null)
              composerRef.current?.focus()
            }}
            onEdit={(p) => {
              setEditingId(p.id)
              setReplyTo(null)
              setDraft(p.text)
              composerRef.current?.focus()
            }}
            onDelete={(id) => void onDelete(id)}
            onReport={(id) => void onReport(id)}
            onUpvote={(id) => void onUpvote(id)}
          />
        ))}
      </ol>
    </section>
  )
}

function PostArticle({
  post,
  depth,
  allowReplies,
  readOnly,
  t,
  onReply,
  onEdit,
  onDelete,
  onReport,
  onUpvote,
}: {
  post: ThreadPost & { replies?: ThreadPost[] }
  depth: number
  allowReplies: boolean
  readOnly: boolean
  t: ContentToolRendererProps['t']
  onReply: (id: string) => void
  onEdit: (p: ThreadPost) => void
  onDelete: (id: string) => void
  onReport: (id: string) => void
  onUpvote: (id: string) => void
}) {
  const replies = (post.replies ?? []).slice(0, depth === 0 ? 3 : 0)
  const extra = (post.replies ?? []).length - replies.length
  const [showAll, setShowAll] = useState(false)
  const visibleReplies = showAll ? post.replies ?? [] : replies

  const name =
    post.authorDisplay ||
    (post.anonymous
      ? t('contentTools.tools.inline_discussion.anonymousClassmate')
      : t('contentTools.tools.inline_discussion.classmate'))

  return (
    <li className={depth > 0 ? 'ms-4 border-s border-border-default ps-3 dark:border-border-default' : ''}>
      <article
        className="space-y-1 rounded-md bg-surface-base p-3/60"
        data-testid={`inline-discussion-post-${post.id}`}
        aria-label={t('contentTools.tools.inline_discussion.postAria', {
          author: name,
          when: post.createdAt || '',
        })}
      >
        <header className="flex flex-wrap items-baseline gap-2 text-xs text-fg-muted">
          <span className="font-medium text-fg-default">{name}</span>
          {post.endorsed ? (
            <span
              className="rounded bg-emerald-100 px-1.5 py-0.5 text-emerald-900 dark:bg-emerald-900/40 dark:text-emerald-100"
              data-testid="inline-discussion-endorsed"
            >
              {t('contentTools.tools.inline_discussion.endorsedBadge')}
            </span>
          ) : null}
          {post.editedAt ? <span>{t('contentTools.tools.inline_discussion.edited')}</span> : null}
        </header>
        {post.tombstone || post.removed ? (
          <p className="text-sm italic text-fg-muted">
            {t('contentTools.tools.inline_discussion.tombstone')}
          </p>
        ) : (
          <p className="text-sm text-fg-default whitespace-pre-wrap">{post.text}</p>
        )}
        {!readOnly && !post.removed ? (
          <div className="flex flex-wrap gap-2 pt-1 text-xs">
            <button type="button" className="underline" onClick={() => onUpvote(post.id)}>
              {t('contentTools.tools.inline_discussion.upvote', {
                count: String(post.upvoteCount ?? 0),
              })}
            </button>
            {allowReplies && depth === 0 ? (
              <button type="button" className="underline" onClick={() => onReply(post.id)}>
                {t('contentTools.tools.inline_discussion.reply')}
              </button>
            ) : null}
            {post.canEdit ? (
              <button type="button" className="underline" onClick={() => onEdit(post)}>
                {t('contentTools.tools.inline_discussion.edit')}
              </button>
            ) : null}
            {post.canDelete ? (
              <button type="button" className="underline" onClick={() => onDelete(post.id)}>
                {t('contentTools.tools.inline_discussion.delete')}
              </button>
            ) : null}
            {!post.isOwn ? (
              <button type="button" className="underline" onClick={() => onReport(post.id)}>
                {t('contentTools.tools.inline_discussion.report')}
              </button>
            ) : null}
          </div>
        ) : null}
      </article>
      {visibleReplies.length > 0 ? (
        <ol className="mt-2 space-y-2">
          {visibleReplies.map((r) => (
            <PostArticle
              key={r.id}
              post={{ ...r, replies: [] }}
              depth={depth + 1}
              allowReplies={false}
              readOnly={readOnly}
              t={t}
              onReply={onReply}
              onEdit={onEdit}
              onDelete={onDelete}
              onReport={onReport}
              onUpvote={onUpvote}
            />
          ))}
        </ol>
      ) : null}
      {extra > 0 && !showAll ? (
        <button
          type="button"
          className="mt-1 text-xs underline text-fg-muted"
          onClick={() => setShowAll(true)}
        >
          {t('contentTools.tools.inline_discussion.showMore', { count: String(extra) })}
        </button>
      ) : null}
    </li>
  )
}

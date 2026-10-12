import type { DiscussionPost, DiscussionThreadDetail } from '../../lib/discussions-api'

/** Block nodes that carry no visible content on their own (only through children). */
const STRUCTURAL_NODE_TYPES = new Set([
  'doc',
  'paragraph',
  'heading',
  'hardBreak',
  'blockquote',
  'bulletList',
  'orderedList',
  'listItem',
  'codeBlock',
])

function nodeHasVisibleContent(node: unknown): boolean {
  if (node == null) return false
  if (typeof node === 'string') return node.trim() !== ''
  if (Array.isArray(node)) return node.some(nodeHasVisibleContent)
  if (typeof node !== 'object') return false
  const n = node as { type?: unknown; text?: unknown; content?: unknown }
  if (n.type === 'text') return typeof n.text === 'string' && n.text.trim() !== ''
  if (Array.isArray(n.content) && n.content.length > 0) {
    return n.content.some(nodeHasVisibleContent)
  }
  // Leaf node: structural leaves (empty paragraph, line break) are blank; anything else
  // (image, horizontal rule, embed…) counts as content.
  return typeof n.type === 'string' && !STRUCTURAL_NODE_TYPES.has(n.type)
}

/** True when a TipTap doc (or legacy string body) has no visible content. */
export function isEmptyDiscussionDoc(doc: unknown): boolean {
  return !nodeHasVisibleContent(doc)
}

export type DiscussionOpeningPostResolution = {
  /** Body rendered in the opening post card, or null when there is nothing to show. */
  openingBody: unknown
  /** Reply promoted into the opening card (thread body was blank), if any. */
  promotedPost: DiscussionPost | null
  /** Remaining replies; children of a promoted post are re-parented to the top level. */
  replies: DiscussionPost[]
}

/**
 * Threads created with only a title have a blank body, and authors often type the prompt as
 * their first reply. Show that first top-level reply by the thread author as the opening post
 * instead of an empty box followed by the prompt as a separate reply.
 */
export function resolveDiscussionOpeningPost(
  thread: Pick<DiscussionThreadDetail, 'authorId' | 'body'>,
  posts: DiscussionPost[],
): DiscussionOpeningPostResolution {
  if (!isEmptyDiscussionDoc(thread.body)) {
    return { openingBody: thread.body, promotedPost: null, replies: posts }
  }
  const firstTopLevel = posts
    .filter((p) => !p.parentPostId)
    .reduce<DiscussionPost | null>(
      (earliest, p) => (earliest == null || p.createdAt < earliest.createdAt ? p : earliest),
      null,
    )
  if (
    !firstTopLevel ||
    firstTopLevel.authorId !== thread.authorId ||
    isEmptyDiscussionDoc(firstTopLevel.body)
  ) {
    return { openingBody: null, promotedPost: null, replies: posts }
  }
  const replies = posts
    .filter((p) => p.id !== firstTopLevel.id)
    .map((p) => (p.parentPostId === firstTopLevel.id ? { ...p, parentPostId: null } : p))
  return { openingBody: firstTopLevel.body, promotedPost: firstTopLevel, replies }
}

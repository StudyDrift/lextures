import { Avatar } from '../ui'
import { formatAbsolute, formatRelativeCompact } from '../../lib/format-datetime'
import { discussionAuthorLabel } from './discussion-author-label'

function initials(name: string): string {
  const parts = name.split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase()
}

export function DiscussionAuthorByline({
  name,
  avatarUrl,
  createdAt,
}: {
  name?: string | null
  avatarUrl?: string | null
  createdAt: string
}) {
  const label = discussionAuthorLabel(name)
  const src = avatarUrl?.trim() || undefined
  return (
    <div className="flex items-center gap-2">
      <Avatar src={src} alt={label} initials={initials(label)} size="sm" />
      <div className="min-w-0">
        <p className="truncate text-sm font-semibold text-fg-default">{label}</p>
        <time
          className="text-xs text-fg-muted"
          dateTime={createdAt}
          title={formatAbsolute(createdAt)}
        >
          {formatRelativeCompact(createdAt)}
        </time>
      </div>
    </div>
  )
}

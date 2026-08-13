/**
 * Visible author byline (SEO.3 FR-21 / UX).
 * Mirrors Person schema; retired authors render as plain text with no link.
 */
import {
  authorDisplayName,
  authorPath,
  authorSlugFrom,
  getAuthor,
  isAuthorLinkable,
  type AuthorRef,
} from '../lib/authors'
import { formatDate } from '../utils/blog'

export type BylineProps = {
  authorSlug: AuthorRef
  datePublished?: string
  dateModified?: string
  reviewedBySlug?: AuthorRef
  reviewedOn?: string
  /** Compact single-line layout under 640px is CSS-driven. */
  className?: string
}

function InitialsAvatar({ name }: { name: string }) {
  const initials = String(name || '')
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map(w => w[0]?.toUpperCase() ?? '')
    .join('')
  return (
    <span
      className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-sm font-semibold"
      style={{ backgroundColor: 'rgba(106,197,176,0.2)', color: '#2f6f63' }}
      aria-hidden
    >
      {initials || '?'}
    </span>
  )
}

export function Byline({
  authorSlug,
  datePublished,
  dateModified,
  reviewedBySlug,
  reviewedOn,
  className = '',
}: BylineProps) {
  const slug = authorSlugFrom(authorSlug)
  const author = getAuthor(slug)
  const name = authorDisplayName(authorSlug)
  const href = authorPath(slug)
  const jobTitle = author?.jobTitle
  const bio = author?.bio
  const image = author?.image

  return (
    <footer
      className={`flex flex-col gap-3 border-t border-slate-200/80 pt-6 sm:flex-row sm:items-start sm:gap-4 ${className}`}
    >
      <div className="flex min-w-0 items-center gap-3 sm:items-start">
        {image ? (
          <img
            src={image}
            alt=""
            className="h-10 w-10 shrink-0 rounded-full object-cover"
            width={40}
            height={40}
          />
        ) : (
          <InitialsAvatar name={name} />
        )}
        <div className="min-w-0">
          <p className="text-sm text-slate-500">
            <span className="sr-only">Written by </span>
            {href && isAuthorLinkable(slug) ? (
              <a
                href={href}
                className="font-medium text-slate-900 no-underline hover:underline"
              >
                {name}
              </a>
            ) : (
              <span className="font-medium text-slate-900">{name}</span>
            )}
            {jobTitle ? (
              <span className="text-slate-500"> · {jobTitle}</span>
            ) : null}
          </p>
          {bio ? (
            <p className="mt-1 hidden text-sm leading-relaxed text-slate-600 sm:line-clamp-2 sm:block">
              {bio}
            </p>
          ) : null}
          <p className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-slate-400">
            {datePublished ? (
              <time dateTime={datePublished}>
                {formatDate(datePublished)}
              </time>
            ) : null}
            {dateModified && dateModified !== datePublished ? (
              <span>
                Updated{' '}
                <time dateTime={dateModified}>{formatDate(dateModified)}</time>
              </span>
            ) : null}
            {reviewedBySlug ? (
              <span>
                Reviewed by {authorDisplayName(reviewedBySlug)}
                {reviewedOn ? (
                  <>
                    {' '}
                    on <time dateTime={reviewedOn}>{formatDate(reviewedOn)}</time>
                  </>
                ) : null}
              </span>
            ) : null}
          </p>
        </div>
      </div>
    </footer>
  )
}

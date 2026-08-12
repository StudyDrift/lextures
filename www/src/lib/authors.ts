/**
 * Author registry (SEO.3 FR-18–FR-21, AC-5, AC-10).
 *
 * Front-matter `author: <slug>` is validated against this registry at build time.
 * Retired authors: page returns 404, Person node dropped, byline stays plain text.
 */
export type AuthorStatus = 'active' | 'retired'

export type Author = {
  slug: string
  name: string
  jobTitle: string
  bio: string
  image?: string
  knowsAbout: string[]
  sameAs: string[]
  alumniOf?: string[]
  credentials?: string[]
  /** ISO date consent was recorded (S04). */
  consentRecordedAt: string
  status: AuthorStatus
  /** When true, this person is the Organization founder (about#founder). */
  isFounder?: boolean
}

import { contentSource } from './content-source'

/**
 * Static registry. Prefer adding markdown under content/authors/ later if volume grows;
 * TypeScript keeps consent + status type-safe for the small team at launch.
 */
const FILE_AUTHORS: readonly Author[] = [
  {
    slug: 'chase-willden',
    name: 'Chase Willden',
    jobTitle: 'Founder',
    bio: 'Founder of Lextures. Builds adaptive learning systems using Item Response Theory, spaced repetition, and open-source LMS infrastructure for schools and homeschool families.',
    knowsAbout: [
      'adaptive learning',
      'Item Response Theory',
      'learning management systems',
      'open source education software',
      'assessment design',
    ],
    sameAs: ['https://github.com/StudyDrift'],
    consentRecordedAt: '2026-08-11',
    status: 'active',
    isFounder: true,
  },
] as const

const apiAuthors: Author[] = contentSource.listAuthors().map(author => {
  const links = Array.isArray(author.links) ? author.links : author.links?.sameAs || []
  const website = !Array.isArray(author.links) ? author.links?.website : undefined
  return {
    slug: author.slug, name: author.name, jobTitle: author.jobTitle || '', bio: author.bio || '',
    knowsAbout: author.knowsAbout || [], sameAs: [...links, ...(website ? [website] : [])],
    consentRecordedAt: '', status: author.status || 'active',
  }
})

/** The API registry is authoritative for API builds; files remain the rollback source. */
export const AUTHORS: readonly Author[] = apiAuthors.length ? apiAuthors : FILE_AUTHORS

const bySlug = new Map(AUTHORS.map(a => [a.slug, a]))

export function getAuthor(slug: string): Author | undefined {
  return bySlug.get(slug)
}

export function getActiveAuthors(): Author[] {
  return AUTHORS.filter(a => a.status === 'active')
}

export function requireAuthor(slug: string, context?: string): Author {
  const a = getAuthor(slug)
  if (!a) {
    throw new Error(
      `Unknown author slug "${slug}"${context ? ` (${context})` : ''}. ` +
        `Register the author in www/src/lib/authors.ts or fix the front-matter.`,
    )
  }
  return a
}

/** Display name for bylines; works for retired authors (plain text, no link). */
export function authorDisplayName(slug: string): string {
  return getAuthor(slug)?.name ?? slug
}

export function isAuthorLinkable(slug: string): boolean {
  const a = getAuthor(slug)
  return Boolean(a && a.status === 'active')
}

export function authorPath(slug: string): string | null {
  return isAuthorLinkable(slug) ? `/authors/${slug}` : null
}

/** Validate all content author slugs; throws listing offenders. */
export function assertKnownAuthors(
  entries: Array<{ path: string; authorSlug: string }>,
): void {
  const bad: string[] = []
  for (const e of entries) {
    if (!getAuthor(e.authorSlug)) {
      bad.push(`${e.path}: unknown author slug "${e.authorSlug}"`)
    }
  }
  if (bad.length) {
    throw new Error(`Author registry validation failed:\n${bad.join('\n')}`)
  }
}

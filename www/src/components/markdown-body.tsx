/**
 * Renders pre-built HTML from markdown (SEO.4 FR-5).
 * Prefer `html` from build-time rendering; falls back to lite runtime for
 * interactive API content only.
 */
import { renderMarkdownLite } from '../lib/markdown-lite'

type MarkdownBodyProps = {
  /** Pre-rendered HTML (preferred — zero markdown runtime). */
  html?: string
  /** Raw markdown for interactive API payloads only. */
  markdown?: string
  className?: string
}

export function MarkdownBody({ html, markdown, className }: MarkdownBodyProps) {
  const content = html ?? (markdown ? renderMarkdownLite(markdown) : '')
  if (!content) return null
  return (
    <div
      className={className}
      dangerouslySetInnerHTML={{ __html: content }}
    />
  )
}

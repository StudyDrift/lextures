import type { DocumentHeadOptions } from './document-head'
import { applyDocumentHead, clearJsonLd } from './document-head'
import { useEffect } from 'react'
import { canonicalUrl } from './site-origin'

const DEFAULT_TITLE = 'Lextures — The learning environment that adapts'
const DEFAULT_DESCRIPTION =
  'The learning environment that adapts. One platform for adaptive quizzing, interactive content, grading, and enrollment — instead of a patchwork of vendors.'

/**
 * Updates document title, meta description, canonical, OG/Twitter tags, robots,
 * and optional JSON-LD on mount; restores homepage defaults on unmount (SEO.1 FR-9).
 */
export function useDocumentHead(opts: DocumentHeadOptions): void {
  const { title, description, canonical, image, jsonLd, robots, markdownAlternate } = opts
  useEffect(() => {
    applyDocumentHead({ title, description, canonical, image, jsonLd, robots, markdownAlternate })
    return () => {
      applyDocumentHead({
        title: DEFAULT_TITLE,
        description: DEFAULT_DESCRIPTION,
        canonical: canonicalUrl('/'),
        robots: 'index,follow',
        markdownAlternate: null,
      })
      clearJsonLd()
    }
  }, [title, description, canonical, image, jsonLd, robots, markdownAlternate])
}

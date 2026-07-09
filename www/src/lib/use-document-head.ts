import type { DocumentHeadOptions } from './document-head'
import { applyDocumentHead, clearJsonLd } from './document-head'
import { useEffect } from 'react'

const DEFAULT_TITLE = 'Lextures — The learning environment that adapts'
const DEFAULT_DESCRIPTION =
  'The learning environment that adapts. One platform for adaptive quizzing, interactive content, grading, and enrollment — instead of a patchwork of vendors.'

/**
 * Updates document title, meta description, canonical, OG/Twitter tags, and
 * optional JSON-LD on mount; restores homepage defaults on unmount (plan MKT10).
 */
export function useDocumentHead(opts: DocumentHeadOptions): void {
  const { title, description, canonical, image, jsonLd } = opts
  useEffect(() => {
    applyDocumentHead({ title, description, canonical, image, jsonLd })
    return () => {
      applyDocumentHead({
        title: DEFAULT_TITLE,
        description: DEFAULT_DESCRIPTION,
        canonical: 'https://lextures.com/',
      })
      clearJsonLd()
    }
  }, [title, description, canonical, image, jsonLd])
}

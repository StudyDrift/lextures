import type { MarketingArticle, MarketingArticleWrite } from '../../../lib/marketing-content-api'

export type Directive = {
  id: string
  label: string
  markdown: string
}

/** Extractability score is 0–10 with publish floor 8.0 (MC.4). */
export const MARKETING_SCORE_MAX = 10
export const MARKETING_PUBLISH_SCORE_FLOOR = 8

export type LintMetadataInput = {
  title: string
  description: string
  authorSlug: string
  cluster: string
  primaryQuestion: string
  keywords?: string[] | null
  locale?: string
  contentUpdatedAt?: string | null
}

/** Shape expected by POST /admin/marketing/lint metadata. */
export function lintMetadata(article: LintMetadataInput) {
  return {
    title: article.title,
    description: article.description,
    author: article.authorSlug,
    authorSlug: article.authorSlug,
    cluster: article.cluster,
    primaryQuestion: article.primaryQuestion,
    keywords: article.keywords ?? [],
    locale: article.locale || 'en',
    updated: article.contentUpdatedAt?.slice(0, 10) || new Date().toISOString().slice(0, 10),
  }
}

export function scoreMeterPercent(score: number | null | undefined): number {
  if (score == null || Number.isNaN(score)) return 0
  return Math.max(0, Math.min(100, (score / MARKETING_SCORE_MAX) * 100))
}

export function scoreToneClass(score: number | null | undefined): string {
  if (score == null) return 'text-fg-muted'
  if (score >= MARKETING_PUBLISH_SCORE_FLOOR) return 'text-success-fg'
  if (score >= 6) return 'text-warning-fg'
  return 'text-danger-fg'
}

export function scoreBarClass(score: number | null | undefined): string {
  if (score == null) return 'bg-border-strong'
  if (score >= MARKETING_PUBLISH_SCORE_FLOOR) return 'bg-success-fg'
  if (score >= 6) return 'bg-warning-fg'
  return 'bg-danger-fg'
}

export function isBlockingFinding(severity: string | undefined): boolean {
  return severity === 'error'
}

export function formatQualityScore(score: number | null | undefined): string {
  if (score == null || Number.isNaN(score)) return '—'
  return Number.isInteger(score) ? String(score) : score.toFixed(1)
}

export const directives: Directive[] = [
  // Guidance must stay markdown-only: HTML comments trip safety.raw-html and block publish.
  ['key-takeaways', 'Key takeaways', ':::key-takeaways\n- First takeaway\n- Second takeaway\n- Third takeaway\n:::\n'],
  ['answer', 'Direct answer', ':::answer\nWrite a 40–60 word answer to the primary question here.\n:::\n'],
  ['definition', 'Definition', ':::definition term="Term"\nWrite a concise, self-contained definition.\n:::\n'],
  ['comparison-table', 'Comparison table', ':::comparison-table\n| Option | Best for | Considerations |\n| --- | --- | --- |\n| A | … | … |\n| B | … | … |\n:::\n'],
  ['steps', 'Steps', ':::steps\n1. First step\n2. Second step\n3. Third step\n:::\n'],
  ['faq', 'FAQ', ':::faq\n### Question one?\nAnswer one.\n\n### Question two?\nAnswer two.\n\n### Question three?\nAnswer three.\n:::\n'],
  ['callout', 'Callout', ':::callout note\nImportant context belongs here.\n:::\n'],
  ['stat', 'Statistic', ':::stat\n**00%** — Explain the statistic and cite its source.\n:::\n'],
  ['sources', 'Sources', ':::sources\n- [Source title](https://example.com)\n:::\n'],
].map(([id, label, markdown]) => ({ id, label, markdown }))

export function slugify(value: string): string {
  return value.toLowerCase().trim().normalize('NFKD').replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 100)
}

export function writePayload(article: MarketingArticle): MarketingArticleWrite {
  return {
    kind: article.kind, slug: article.slug, locale: article.locale, categoryId: article.categoryId, title: article.title,
    description: article.description, bodyMd: article.bodyMd, authorSlug: article.authorSlug, reviewerSlug: article.reviewerSlug,
    reviewDueOn: article.reviewDueOn,
    primaryQuestion: article.primaryQuestion, cluster: article.cluster, pillar: article.pillar, verifiedAgainst: article.verifiedAgainst,
    keywords: article.keywords, relatedTo: article.relatedTo, roles: article.roles, segments: article.segments, citations: article.citations,
    heroMediaId: article.heroMediaId, socialTitle: article.socialTitle ?? '', socialDescription: article.socialDescription ?? '',
    noindex: article.noindex, canonicalOverride: article.canonicalOverride,
  }
}

export function commaList(value: string): string[] {
  return value.split(',').map((item) => item.trim()).filter(Boolean)
}

export function simpleLineDiff(before: string, after: string) {
  const oldLines = before.split('\n')
  const newLines = after.split('\n')
  const max = Math.max(oldLines.length, newLines.length)
  const lines: Array<{ type: 'same' | 'removed' | 'added'; text: string }> = []
  for (let index = 0; index < max; index += 1) {
    const oldLine = oldLines[index]
    const newLine = newLines[index]
    if (oldLine === newLine) lines.push({ type: 'same', text: oldLine ?? '' })
    else {
      if (oldLine !== undefined) lines.push({ type: 'removed', text: oldLine })
      if (newLine !== undefined) lines.push({ type: 'added', text: newLine })
    }
  }
  return lines
}

/** Decorative images use this title marker in markdown wire format (plan 12.5). */
export const DECORATIVE_IMAGE_TITLE = 'lex-decorative'

export type MarkdownImageAltStatus = {
  alt: string
  src: string
  title?: string
  decorative: boolean
  hasValidAlt: boolean
  line: number
}

const markdownImageRe = /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)/g

/** Scan markdown for image alt-text coverage. */
export function scanMarkdownImages(markdown: string): MarkdownImageAltStatus[] {
  const lines = markdown.split('\n')
  const out: MarkdownImageAltStatus[] = []
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    markdownImageRe.lastIndex = 0
    let m: RegExpExecArray | null
    while ((m = markdownImageRe.exec(line)) !== null) {
      const alt = m[1] ?? ''
      const src = m[2] ?? ''
      const title = m[3]
      const decorative = title === DECORATIVE_IMAGE_TITLE
      const hasValidAlt = decorative || alt.trim().length > 0
      out.push({ alt, src, title, decorative, hasValidAlt, line: i + 1 })
    }
  }
  return out
}

export type AltTextCoverage = {
  withAlt: number
  total: number
  missing: MarkdownImageAltStatus[]
}

export function summarizeAltTextCoverage(markdown: string): AltTextCoverage {
  const images = scanMarkdownImages(markdown)
  const missing = images.filter((img) => !img.hasValidAlt)
  return {
    withAlt: images.length - missing.length,
    total: images.length,
    missing,
  }
}

/** True when any image lacks alt text and is not marked decorative. */
export function markdownHasMissingAltText(markdown: string): boolean {
  return summarizeAltTextCoverage(markdown).missing.length > 0
}

/** Summarize alt coverage across multiple markdown sections. */
export function summarizeSectionsAltText(sections: { markdown: string }[]): AltTextCoverage {
  let withAlt = 0
  let total = 0
  const missing: MarkdownImageAltStatus[] = []
  for (const s of sections) {
    const cov = summarizeAltTextCoverage(s.markdown)
    withAlt += cov.withAlt
    total += cov.total
    missing.push(...cov.missing)
  }
  return { withAlt, total, missing }
}

export function coveragePercent(withAlt: number, total: number): number {
  if (total <= 0) return 100
  return Math.round((withAlt / total) * 100)
}

export function formatCoverageLabel(withAlt: number, total: number): string {
  const pct = coveragePercent(withAlt, total)
  return `${pct}% (${withAlt}/${total} images)`
}

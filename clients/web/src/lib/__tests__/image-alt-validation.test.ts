import { describe, expect, it } from 'vitest'
import {
  coveragePercent,
  formatCoverageLabel,
  markdownHasMissingAltText,
  scanMarkdownImages,
  summarizeAltTextCoverage,
  DECORATIVE_IMAGE_TITLE,
} from '../image-alt-validation'

describe('image-alt-validation', () => {
  it('detects missing, valid, and decorative images', () => {
    const md = `![Diagram](https://x/a.png)
![](https://x/b.png)
![](https://x/c.png "${DECORATIVE_IMAGE_TITLE}")`
    const images = scanMarkdownImages(md)
    expect(images).toHaveLength(3)
    expect(images[0].hasValidAlt).toBe(true)
    expect(images[1].hasValidAlt).toBe(false)
    expect(images[2].decorative).toBe(true)
    expect(images[2].hasValidAlt).toBe(true)
  })

  it('summarizes coverage', () => {
    const md = '![a](x.png)\n![](y.png)'
    const cov = summarizeAltTextCoverage(md)
    expect(cov.withAlt).toBe(1)
    expect(cov.total).toBe(2)
    expect(cov.missing).toHaveLength(1)
    expect(markdownHasMissingAltText(md)).toBe(true)
  })

  it('formats coverage label', () => {
    expect(formatCoverageLabel(8, 10)).toBe('80% (8/10 images)')
    expect(coveragePercent(0, 0)).toBe(100)
  })
})

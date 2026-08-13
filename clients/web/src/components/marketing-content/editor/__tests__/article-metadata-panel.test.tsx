import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { MarketingArticle } from '../../../../lib/marketing-content-api'
import { ArticleMetadataPanel } from '../article-metadata-panel'

describe('ArticleMetadataPanel', () => {
  it('renders legacy articles with null collection metadata', () => {
    const article = {
      id: 'article-1', kind: 'blog', slug: 'legacy', path: '/blog/legacy', title: 'Legacy',
      status: 'draft', authorSlug: 'author', updatedAt: '', revisionNo: 1, locale: 'en',
      bodyMd: '', description: '', primaryQuestion: '', cluster: '', pillar: '',
      verifiedAgainst: '', keywords: null, relatedTo: null, roles: null, segments: null,
      citations: null, noindex: false,
    } as unknown as MarketingArticle

    render(<ArticleMetadataPanel article={article} onChange={vi.fn()} categories={null} authors={null} knownPaths={null} isNew={false} />)

    // Field marks required controls with a trailing asterisk in the label text.
    expect(screen.getByLabelText(/^Keywords/)).toHaveValue('')
    expect(screen.getByLabelText('Citations')).toHaveValue('')
    expect(screen.getByLabelText(/^Locale/)).toHaveValue('en')
    expect(screen.getByLabelText(/^Locale/)).toBeDisabled()
    expect(screen.getByLabelText(/^Author/)).toBeInTheDocument()
    expect(screen.getByLabelText(/^Primary question/)).toBeInTheDocument()
    expect(screen.getByLabelText(/^Cluster/)).toBeInTheDocument()
  })
})

import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { MarketingArticle } from '../../../../lib/marketing-content-api'
import { ArticleMetadataPanel } from '../article-metadata-panel'

vi.mock('../../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffMotionControls: false }),
}))

const generateMarketingArticleMetadata = vi.fn()
vi.mock('../../../../lib/marketing-content-ai-api', () => ({
  generateMarketingArticleMetadata: (...args: unknown[]) => generateMarketingArticleMetadata(...args),
}))

describe('ArticleMetadataPanel', () => {
  beforeEach(() => {
    generateMarketingArticleMetadata.mockReset()
  })

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

  it('shows a required-field finding on the matching control', () => {
    const article = {
      id: 'article-1', kind: 'blog', slug: 'advice', path: '/blog/advice', title: 'Advice',
      status: 'draft', authorSlug: 'author', updatedAt: '', revisionNo: 1, locale: 'en',
      bodyMd: '', description: '', primaryQuestion: '', cluster: '', pillar: '',
      verifiedAgainst: '', keywords: [], relatedTo: [], roles: [], segments: [],
      citations: [], noindex: false,
    } as unknown as MarketingArticle

    render(
      <ArticleMetadataPanel
        article={article}
        onChange={vi.fn()}
        categories={null}
        authors={null}
        knownPaths={null}
        isNew={false}
        findings={[{ rule: 'fm.cluster', severity: 'error', message: 'Required metadata field is missing.', path: 'cluster' }]}
        highlightField="cluster"
      />,
    )

    expect(screen.getByText('Required metadata field is missing.')).toBeInTheDocument()
    expect(screen.getByLabelText(/^Cluster/)).toHaveAttribute('aria-invalid', 'true')
  })

  it('hides Fill out with AI unless the author can use it', () => {
    const article = {
      id: 'article-1', kind: 'blog', slug: 'legacy', path: '/blog/legacy', title: 'Legacy',
      status: 'draft', authorSlug: 'author', updatedAt: '', revisionNo: 1, locale: 'en',
      bodyMd: '', description: '', primaryQuestion: '', cluster: '', pillar: '',
      verifiedAgainst: '', keywords: null, relatedTo: null, roles: null, segments: null,
      citations: null, noindex: false,
    } as unknown as MarketingArticle

    render(<ArticleMetadataPanel article={article} onChange={vi.fn()} categories={null} authors={null} knownPaths={null} isNew={false} />)
    expect(screen.queryByRole('button', { name: 'Fill out with AI' })).not.toBeInTheDocument()
  })

  it('fills description and auto slug from AI', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    generateMarketingArticleMetadata.mockResolvedValue({
      title: '',
      slug: 'Homeschooling Student Advice',
      description: 'Practical advice for homeschool families.',
      bodyMd: '',
      primaryQuestion: '',
      cluster: '',
      pillar: '',
      keywords: [],
    })
    const article = {
      id: '', kind: 'blog', slug: 'homeschooling-student-advice', path: '', title: 'Homeschooling student advice',
      status: 'draft', authorSlug: '', updatedAt: '', revisionNo: 0, locale: 'en',
      bodyMd: '', description: '', primaryQuestion: '', cluster: '', pillar: '',
      verifiedAgainst: '', keywords: [], relatedTo: [], roles: [], segments: [],
      citations: [], noindex: false,
    } as unknown as MarketingArticle

    render(
      <ArticleMetadataPanel
        article={article}
        onChange={onChange}
        categories={null}
        authors={null}
        knownPaths={null}
        isNew
        canFillWithAI
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Fill out with AI' }))

    expect(generateMarketingArticleMetadata).toHaveBeenCalledWith({
      kind: 'blog',
      existingTitle: 'Homeschooling student advice',
      existingBodyMd: '',
    })
    expect(onChange).toHaveBeenCalledWith({
      description: 'Practical advice for homeschool families.',
      slug: 'homeschooling-student-advice',
    })
  })

  it('asks for a title or body before generating', async () => {
    const user = userEvent.setup()
    const article = {
      id: '', kind: 'blog', slug: '', path: '', title: '',
      status: 'draft', authorSlug: '', updatedAt: '', revisionNo: 0, locale: 'en',
      bodyMd: '', description: '', primaryQuestion: '', cluster: '', pillar: '',
      verifiedAgainst: '', keywords: [], relatedTo: [], roles: [], segments: [],
      citations: [], noindex: false,
    } as unknown as MarketingArticle

    render(
      <ArticleMetadataPanel
        article={article}
        onChange={vi.fn()}
        categories={null}
        authors={null}
        knownPaths={null}
        isNew
        canFillWithAI
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Fill out with AI' }))

    expect(generateMarketingArticleMetadata).not.toHaveBeenCalled()
    expect(screen.getByText('Add a title or article body first.')).toBeInTheDocument()
  })
})

import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ArticlePreview } from '../article-preview'

describe('ArticlePreview', () => {
  it('renders markdown headings, emphasis, and lists instead of source markers', () => {
    render(
      <ArticlePreview
        title="Homeschooling Student Advice"
        body={'## Why keep records?\n\nFamilies need **one place** for grades.\n\n- Transcripts\n- Lesson plans\n'}
      />,
    )

    expect(screen.getByRole('heading', { level: 1, name: 'Homeschooling Student Advice' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 2, name: 'Why keep records?' })).toBeInTheDocument()
    expect(screen.getByText('one place').tagName).toBe('STRONG')
    expect(screen.getByText('Transcripts').tagName).toBe('LI')
    expect(screen.queryByText(/## Why keep records/)).not.toBeInTheDocument()
    expect(screen.queryByText(/\*\*one place\*\*/)).not.toBeInTheDocument()
  })

  it('renders marketing directive blocks instead of fence syntax', () => {
    render(
      <ArticlePreview
        title="Advice"
        body={[
          ':::key-takeaways',
          '- Keep grades in one place',
          ':::',
          '',
          ':::answer',
          'Use a single record so transcripts stay current.',
          ':::',
        ].join('\n')}
      />,
    )

    expect(screen.getByRole('heading', { name: 'Key takeaways' })).toBeInTheDocument()
    expect(screen.getByText('Keep grades in one place')).toBeInTheDocument()
    expect(screen.getByText('Use a single record so transcripts stay current.')).toBeInTheDocument()
    expect(screen.queryByText(/:::key-takeaways/)).not.toBeInTheDocument()
  })
})

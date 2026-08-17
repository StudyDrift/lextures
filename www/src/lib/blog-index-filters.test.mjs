import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { blogPostMatches, blogPostSearchText, paginateBlogPosts } from './blog-index-filters.ts'

describe('blog listing filters', () => {
  const critical = blogPostSearchText([
    'Critical Thinking Through Memorization',
    'An exploration of the irony in education systems',
    'critical-thinking-through-memorization',
    'chase-willden',
    'Chase Willden',
    'author:chase-willden',
    'pillar:p2',
  ])

  const blooms = blogPostSearchText([
    "Flipping the Pyramid: Bloom's Taxonomy in the Age of Generative AI",
    'Use Bloom\'s taxonomy to design visible evidence of reasoning',
    'blooms-taxonomy-in-the-age-of-ai',
    'chase-willden',
    'Chase Willden',
    'author:chase-willden',
    'pillar:p2',
  ])

  it('matches title terms such as critical', () => {
    assert.equal(blogPostMatches(critical, { query: 'critical', pillar: '', author: '' }), true)
    assert.equal(blogPostMatches(blooms, { query: 'critical', pillar: '', author: '' }), false)
  })

  it('matches hyphenated slugs as words', () => {
    assert.equal(blogPostMatches(critical, { query: 'memorization', pillar: '', author: '' }), true)
  })

  it('filters by pillar and author tokens without using them as search text', () => {
    assert.equal(blogPostMatches(critical, { query: '', pillar: 'p2', author: '' }), true)
    assert.equal(blogPostMatches(critical, { query: '', pillar: 'p6', author: '' }), false)
    assert.equal(blogPostMatches(critical, { query: 'author:chase-willden', pillar: '', author: '' }), false)
    assert.equal(blogPostMatches(critical, { query: '', pillar: '', author: 'chase-willden' }), true)
  })

  it('paginates filtered results', () => {
    assert.deepEqual(paginateBlogPosts(['a', 'b', 'c'], 2, 1), ['b'])
  })
})

import { describe, expect, it } from 'vitest'
import {
  bodyHasDirective,
  directiveTemplateForFinding,
  findingLocationLabel,
  markdownLineRange,
  mergeRepairDraft,
  solveAllFindings,
  solveFindingsSequentially,
  visibleLineSnippet,
} from '../article-finding-nav'

describe('article finding navigation', () => {
  it('maps structure rules to insertable directive templates', () => {
    expect(directiveTemplateForFinding('struct.key-takeaways')).toContain(':::key-takeaways')
    expect(directiveTemplateForFinding('struct.answer-block')).toContain(':::answer')
    expect(directiveTemplateForFinding('fm.cluster')).toBeNull()
  })

  it('detects whether a required directive already exists', () => {
    expect(bodyHasDirective('# Title\n\n:::answer\nYes.\n:::\n', 'struct.answer-block')).toBe(true)
    expect(bodyHasDirective('# Title\n\nNo answer yet.\n', 'struct.answer-block')).toBe(false)
  })

  it('resolves markdown line ranges and visible snippets', () => {
    const body = 'alpha\n## How does loop scheduling work?\nomega'
    expect(markdownLineRange(body, 2)).toEqual({
      start: 6,
      end: 39,
      text: '## How does loop scheduling work?',
    })
    expect(visibleLineSnippet('## How does loop scheduling work?')).toBe('How does loop scheduling work?')
  })

  it('labels metadata paths and body lines', () => {
    expect(findingLocationLabel({ rule: 'fm.cluster', severity: 'error', message: 'missing', path: 'cluster' })).toBe('cluster')
    expect(findingLocationLabel({ rule: 'struct.heading-questions', severity: 'warning', message: 'headings', line: 7 })).toBe('line 7')
    expect(findingLocationLabel({ rule: 'extractability.score', severity: 'error', message: 'low', line: 1 })).toBeNull()
  })

  it('merges a repair draft without wiping fields the model left blank', () => {
    const next = mergeRepairDraft(
      {
        title: 'Old title',
        slug: 'old-title',
        description: 'Old description',
        bodyMd: 'Old body',
        primaryQuestion: 'Old?',
        cluster: 'Old cluster',
        pillar: 'Old pillar',
        keywords: ['old'],
      },
      {
        title: 'New title',
        slug: '',
        description: '',
        socialTitle: '',
        socialDescription: '',
        bodyMd: ':::answer\nFixed.\n:::',
        primaryQuestion: 'New?',
        cluster: 'Focus',
        pillar: '',
        keywords: ['attention', 'classroom'],
      },
    )
    expect(next).toMatchObject({
      title: 'New title',
      slug: 'new-title',
      description: 'Old description',
      bodyMd: ':::answer\nFixed.\n:::',
      primaryQuestion: 'New?',
      cluster: 'Focus',
      pillar: 'Old pillar',
      keywords: ['attention', 'classroom'],
    })
  })

  it('repairs every finding in a single pass', async () => {
    const result = await solveAllFindings({
      article: {
        kind: 'blog',
        title: 'Old title',
        slug: 'old-title',
        description: 'Old description',
        bodyMd: 'Old body',
        primaryQuestion: '',
        cluster: '',
        pillar: '',
        keywords: [],
      },
      findings: [
        { rule: 'passage.length', severity: 'warning', message: 'Direct answer is 69 words; target is 40-60.', line: 10 },
        { rule: 'extractability.score', severity: 'warning', message: 'Extractability score 3.5 is below 8.0.', line: 1 },
      ],
      repair: async (article, findings) => ({
        title: article.title,
        slug: article.slug,
        description: article.description,
        socialTitle: '',
        socialDescription: '',
        bodyMd: `fixed:${findings.map((finding) => finding.rule).join(',')}`,
        primaryQuestion: article.primaryQuestion,
        cluster: article.cluster,
        pillar: article.pillar,
        keywords: article.keywords,
      }),
    })
    expect(result.applied).toBe(2)
    expect(result.error).toBeUndefined()
    expect(result.article.bodyMd).toBe('fixed:passage.length,extractability.score')
  })

  it('walks every finding, including warnings, and applies each repair', async () => {
    const seen: string[] = []
    const result = await solveFindingsSequentially({
      article: {
        kind: 'blog',
        title: 'Old title',
        slug: 'old-title',
        description: 'Old description',
        bodyMd: 'Old body',
        primaryQuestion: '',
        cluster: '',
        pillar: '',
        keywords: [],
      },
      findings: [
        { rule: 'passage.length', severity: 'warning', message: 'Direct answer is 69 words; target is 40-60.', line: 10 },
        { rule: 'cite.source-resolvable', severity: 'warning', message: 'Citation has no resolvable source definition.', line: 16 },
        { rule: 'fm.cluster', severity: 'error', message: 'Required metadata field is missing.', path: 'cluster' },
      ],
      onProgress: (_index, _total, finding) => { seen.push(finding.rule) },
      repair: async (article, finding) => ({
        title: article.title,
        slug: article.slug,
        description: article.description,
        socialTitle: '',
        socialDescription: '',
        bodyMd: `${article.bodyMd}\nfixed:${finding.rule}`,
        primaryQuestion: article.primaryQuestion,
        cluster: finding.path === 'cluster' ? 'Focus' : article.cluster,
        pillar: article.pillar,
        keywords: article.keywords,
      }),
    })
    expect(seen).toEqual(['passage.length', 'cite.source-resolvable', 'fm.cluster'])
    expect(result.applied).toBe(3)
    expect(result.error).toBeUndefined()
    expect(result.article.bodyMd).toContain('fixed:passage.length')
    expect(result.article.bodyMd).toContain('fixed:cite.source-resolvable')
    expect(result.article.cluster).toBe('Focus')
  })
})

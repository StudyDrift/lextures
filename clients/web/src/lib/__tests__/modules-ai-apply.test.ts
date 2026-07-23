import { describe, expect, it } from 'vitest'
import { describeModulesAiProposal, orderModulesAiProposals } from '../modules-ai-apply'

describe('describeModulesAiProposal', () => {
  it('describes create and rename ops', () => {
    expect(describeModulesAiProposal({ op: 'create_module', title: 'Week 2' })).toContain('Week 2')
    expect(
      describeModulesAiProposal({
        op: 'rename',
        itemId: 'x',
        title: 'Intro',
      }),
    ).toContain('Intro')
    expect(
      describeModulesAiProposal({
        op: 'set_published',
        itemId: 'x',
        published: true,
      }),
    ).toMatch(/Publish/i)
    expect(
      describeModulesAiProposal({
        op: 'create_quiz',
        title: 'Quiz A',
        moduleTitle: 'Week 2',
      }),
    ).toContain('Week 2')
  })
})

describe('orderModulesAiProposals', () => {
  it('puts create_module before child creates', () => {
    const ordered = orderModulesAiProposals([
      { op: 'create_quiz', title: 'Q1', moduleTitle: 'Week 3' },
      { op: 'create_module', title: 'Week 3' },
      { op: 'create_assignment', title: 'A1', moduleTitle: 'Week 3' },
    ])
    expect(ordered.map((p) => p.op)).toEqual([
      'create_module',
      'create_quiz',
      'create_assignment',
    ])
  })
})

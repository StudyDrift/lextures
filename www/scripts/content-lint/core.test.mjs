import test from 'node:test'
import assert from 'node:assert/strict'
import { analyzeContent } from './core.mjs'

const front = `---\ntitle: Test\ndescription: ${'A'.repeat(125)}\npublished: 2026-01-01\nupdated: 2026-01-01\nauthor: chase-willden\ncluster: test\nprimaryQuestion: What is this?\nkeywords: [test, content]\n---\n`
test('missing answer names the requirement', () => assert(analyzeContent(front + 'Text', 'new.mdx', { enforce: true }).issues.some(i => i.rule === 'answer-box')))
test('95-word answer reports actual count', () => assert.match(analyzeContent(front + `:::answer\n${'word '.repeat(95)}\n:::`, 'new.mdx').issues.find(i => i.rule === 'answer-length').message, /95 words/))
test('uncited statistic reports its line', () => { const i = analyzeContent(front + '73% of teachers agree.', 'new.mdx').issues.find(i => i.rule === 'uncited-statistic'); assert.equal(i.line, 1) })
test('rejects imports, HTML, and unknown directives', () => { const rules = analyzeContent(front + 'import X from "x"\n<div>x</div>\n:::evil\nx\n:::', 'new.mdx').issues.filter(i => i.rule === 'security-policy'); assert.equal(rules.length, 3) })

import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import {
  agentAllowedByRobots,
  extractCanonical,
  extractH1,
  isEdgeBotBlock,
  normalizePath,
} from './lib.mjs'

describe('normalizePath', () => {
  it('strips trailing slashes except root', () => {
    assert.equal(normalizePath('https://lextures.com/parents/'), '/parents')
    assert.equal(normalizePath('https://lextures.com/parents'), '/parents')
    assert.equal(normalizePath('https://lextures.com/'), '/')
    assert.equal(normalizePath('/docs/courses/'), '/docs/courses')
  })
})

describe('extractCanonical', () => {
  it('reads rel-then-href and href-then-rel', () => {
    assert.equal(
      extractCanonical('<link rel="canonical" href="https://lextures.com/parents" />'),
      'https://lextures.com/parents',
    )
    assert.equal(
      extractCanonical('<link href="https://lextures.com/about" rel="canonical">'),
      'https://lextures.com/about',
    )
  })
})

describe('extractH1', () => {
  it('strips nested tags', () => {
    assert.equal(extractH1('<h1 class="x"><span>Hello</span> world</h1>'), 'Hello world')
  })
})

describe('isEdgeBotBlock', () => {
  it('detects Cloudflare plain-text blocks', () => {
    assert.equal(isEdgeBotBlock(403, 'Your request was blocked.', 'text/plain'), true)
    assert.equal(isEdgeBotBlock(200, '<html><h1>Ok</h1></html>', 'text/html'), false)
    assert.equal(isEdgeBotBlock(404, 'not found', 'text/plain'), false)
  })
})

describe('agentAllowedByRobots', () => {
  const robots = `
User-agent: ClaudeBot
Allow: /

User-agent: PerplexityBot
Disallow: /

User-agent: *
Disallow: /404
`
  it('honors agent-specific allow/disallow', () => {
    assert.equal(agentAllowedByRobots(robots, 'ClaudeBot'), true)
    assert.equal(agentAllowedByRobots(robots, 'PerplexityBot'), false)
    assert.equal(agentAllowedByRobots(robots, 'GPTBot'), true)
  })

  it('ignores Cloudflare managed preamble when origin policy allows', () => {
    const live = `
# BEGIN Cloudflare Managed content
User-agent: ClaudeBot
Disallow: /
User-agent: GPTBot
Disallow: /
# END Cloudflare Managed Content

# Lextures crawler policy
User-agent: ClaudeBot
Allow: /
User-agent: GPTBot
Allow: /
User-agent: *
Allow: /
`
    assert.equal(agentAllowedByRobots(live, 'ClaudeBot'), true)
    assert.equal(agentAllowedByRobots(live, 'GPTBot'), true)
  })
})

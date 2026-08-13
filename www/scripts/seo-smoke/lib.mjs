/** Shared helpers for post-deploy SEO smoke checks. */

export const AI_AGENTS = ['GPTBot', 'OAI-SearchBot', 'ClaudeBot', 'PerplexityBot']

/** Pathname with trailing slash stripped (except `/`), matching www URL policy. */
export function normalizePath(value) {
  try {
    return new URL(value, 'https://lextures.com').pathname.replace(/\/$/, '') || '/'
  } catch {
    return value
  }
}

/** Extract canonical href whether `rel` or `href` comes first. */
export function extractCanonical(html) {
  const re =
    /<link\b(?=[^>]*\brel=["']canonical["'])(?=[^>]*\bhref=["']([^"']+)["'])[^>]*>/i
  return html.match(re)?.[1] || null
}

export function extractH1(html) {
  const inner = html.match(/<h1\b[^>]*>([\s\S]*?)<\/h1>/i)?.[1]
  return inner ? inner.replace(/<[^>]+>/g, '').trim() : ''
}

/** Cloudflare (and similar) edge blocks of spoofed bot UAs from CI IPs. */
export function isEdgeBotBlock(status, body, contentType) {
  if (status !== 403 && status !== 401) return false
  const text = String(body || '')
  const type = String(contentType || '')
  if (/your request was blocked/i.test(text)) return true
  if (/cf-ray|attention required|cloudflare/i.test(text)) return true
  if (type.includes('text/plain') && /blocked|denied|forbidden/i.test(text)) return true
  return false
}

/**
 * Drop Cloudflare-injected managed robots preamble so we evaluate the origin
 * policy generated from `crawler-policy.ts` (SEO.2 source of truth).
 */
export function stripManagedRobotsPreamble(robotsTxt) {
  return String(robotsTxt || '').replace(
    /#\s*BEGIN Cloudflare Managed content[\s\S]*?#\s*END Cloudflare Managed Content\s*/i,
    '',
  )
}

/**
 * Parse robots.txt for whether `agent` may fetch `/`.
 * Fails closed only on an explicit `Disallow: /` in the matching agent block
 * (or `User-agent: *` when no agent-specific block exists).
 */
export function agentAllowedByRobots(robotsTxt, agent) {
  const blocks = []
  let current = null
  for (const raw of stripManagedRobotsPreamble(robotsTxt).split(/\r?\n/)) {
    const line = raw.trim()
    if (!line || line.startsWith('#')) continue
    const ua = line.match(/^user-agent:\s*(.+)$/i)?.[1]?.trim()
    if (ua) {
      current = { agents: [ua], disallows: [] }
      blocks.push(current)
      continue
    }
    if (!current) continue
    const disallow = line.match(/^disallow:\s*(.*)$/i)?.[1]
    if (disallow !== undefined) current.disallows.push(disallow.trim())
  }

  const match =
    blocks.find(b => b.agents.some(a => a.toLowerCase() === agent.toLowerCase())) ||
    blocks.find(b => b.agents.includes('*'))
  if (!match) return true
  return !match.disallows.some(d => d === '/')
}

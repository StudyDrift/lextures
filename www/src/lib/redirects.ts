/**
 * Declarative redirect map (SEO.1 FR-13 / FR-14).
 * GitHub Pages cannot issue real 301s; until Cloudflare Pages migration (FR-12),
 * legacy paths get meta-refresh + canonical stubs. `_redirects` is emitted for
 * hosts that support it.
 */

export type RedirectRule = {
  from: string
  to: string
  /** Preferred status when the host supports it (Cloudflare Pages, Netlify). */
  status: 301 | 308
  addedAt: string
  reason: string
}

/**
 * Permanent path moves. `/self-learner` → `/homeschool` (HS.2).
 * Add entries here when URLs change so SEO.5 can replay on host migration.
 */
export const REDIRECTS: RedirectRule[] = [
  { from: '/docs/creating-a-new-course', to: '/docs/getting-started/creating-a-new-course', status: 301, addedAt: '2026-08-11', reason: 'Move legacy help article into the categorized help center.' },
  { from: '/docs/finding-your-course', to: '/docs/getting-started/finding-your-course', status: 301, addedAt: '2026-08-11', reason: 'Move legacy help article into the categorized help center.' },
  { from: '/docs/navigating-the-course-interface', to: '/docs/getting-started/navigating-the-course-interface', status: 301, addedAt: '2026-08-11', reason: 'Move legacy help article into the categorized help center.' },
  { from: '/docs/self-hosting', to: '/docs/self-hosting/self-hosting-requirements-install', status: 301, addedAt: '2026-08-11', reason: 'Replace the legacy self-hosting article with the categorized installation guide.' },
  { from: '/docs/connecting-lextures-to-zapier', to: '/docs/integrations/connecting-lextures-to-zapier', status: 301, addedAt: '2026-08-11', reason: 'Move legacy help article into the categorized help center.' },
  { from: '/docs/using-lextures-with-make', to: '/docs/integrations/using-lextures-with-make', status: 301, addedAt: '2026-08-11', reason: 'Move legacy help article into the categorized help center.' },
  { from: '/self-learner', to: '/homeschool', status: 301, addedAt: '2026-08-11', reason: 'Consolidate the legacy independent-learner audience page.' },
  { from: '/self-learner/', to: '/homeschool', status: 301, addedAt: '2026-08-11', reason: 'Remove the legacy trailing slash.' },
  { from: '/k-12', to: '/k12', status: 301, addedAt: '2026-08-11', reason: 'Adopt the stable K-12 segment URL.' },
  { from: '/k-12/', to: '/k12', status: 301, addedAt: '2026-08-11', reason: 'Adopt the stable K-12 segment URL and remove the trailing slash.' },
]

export function flattenAndValidateRedirects(
  validTargets: Iterable<string>,
  rules: RedirectRule[] = REDIRECTS,
): RedirectRule[] {
  const valid = new Set(validTargets)
  const byFrom = new Map(rules.map(rule => [rule.from, rule]))
  const flattened = rules.map(rule => {
    const seen = new Set([rule.from])
    let target = rule.to
    while (byFrom.has(target)) {
      if (seen.has(target)) throw new Error(`Redirect cycle detected at ${target}`)
      seen.add(target)
      target = byFrom.get(target)!.to
    }
    if (!target.startsWith('/')) throw new Error(`External redirect target is prohibited: ${target}`)
    if (!valid.has(target)) throw new Error(`Redirect target does not exist: ${rule.from} -> ${target}`)
    return { ...rule, to: target }
  })
  if (new Set(flattened.map(rule => rule.from)).size !== flattened.length) {
    throw new Error('Duplicate redirect source')
  }
  return flattened
}

/** Cloudflare Pages / Netlify `_redirects` body. */
export function buildRedirectsFile(rules: RedirectRule[] = REDIRECTS): string {
  const lines = rules.map(r => `${r.from} ${r.to} ${r.status}`)
  // SPA fallback: only for paths with no static file (real 404s still use 404.html on GH Pages)
  lines.push('/*    /404.html   404')
  return `${lines.join('\n')}\n`
}

/**
 * Static HTML stub for hosts that cannot 301 (GitHub Pages).
 * Shape matches the existing /self-learner redirect.
 */
export function buildRedirectStubHtml(toPath: string, siteOrigin: string): string {
  const origin = siteOrigin.replace(/\/$/, '')
  const target = toPath.startsWith('/') ? toPath : `/${toPath}`
  const canonical = `${origin}${target === '/' ? '/' : target.replace(/\/+$/, '')}`
  const esc = (s: string) =>
    s
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;')
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta http-equiv="refresh" content="0; url=${esc(target)}" />
  <link rel="canonical" href="${esc(canonical)}" />
  <meta name="robots" content="noindex,follow" />
  <title>Moved — Lextures</title>
</head>
<body>
  <p>This page has moved to <a href="${esc(target)}">${esc(target)}</a>.</p>
</body>
</html>
`
}

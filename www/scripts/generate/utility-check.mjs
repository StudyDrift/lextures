const words = value => String(value || '').toLowerCase().match(/[a-z0-9]+(?:'[a-z]+)?/g) || []
export function ngrams(value, size = 5) { const tokens = words(value); return new Set(tokens.slice(0, Math.max(0, tokens.length - size + 1)).map((_, i) => tokens.slice(i, i + size).join(' '))) }
export function similarity(left, right) { const a = ngrams(left); const b = ngrams(right); if (!a.size || !b.size) return 0; let shared = 0; for (const gram of a) if (b.has(gram)) shared++; return shared / Math.min(a.size, b.size) }
export function checkUtilityPage(page, siblings = []) {
  const maxSiblingSimilarity = Math.max(0, ...siblings.filter(item => item.path !== page.path).map(item => similarity(page.prose, item.prose)))
  const uniqueWords = words(page.prose).length
  const tests = { action: Boolean(page.actions?.length), unique: uniqueWords >= 150 && maxSiblingSimilarity < 0.6, sourced: Boolean(page.sources?.length), connected: (page.inboundLinks ?? 0) >= 3 && (page.outboundLinks?.length ?? 0) >= 3 }
  const indexed = Object.values(tests).every(Boolean) && (!page.reviewRequired || Boolean(page.reviewedBy))
  return { family: page.family, path: page.path, tests, uniqueWords, maxSiblingSimilarity, inboundLinks: page.inboundLinks ?? 0, indexed, robots: indexed ? 'index,follow' : 'noindex,follow' }
}
export function checkFamily(pages) { const report = pages.map(page => checkUtilityPage(page, pages.filter(item => item.family === page.family))); const passing = report.filter(item => item.indexed).length; return { report, launchable: report.length > 0 && passing / report.length >= 0.8 } }

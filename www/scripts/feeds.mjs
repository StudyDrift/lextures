const xmlEscape = value => String(value ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&apos;')
const plainText = value => [...String(value ?? '').replace(/<script[\s\S]*?<\/script>/gi, ' ').replace(/<[^>]+>/g, ' ').replace(/\[([^\]]+)\]\([^)]+\)/g, '$1').replace(/[*_`#>~-]/g, ' ')].filter(char => { const code = char.codePointAt(0); return code === 9 || code === 10 || code === 13 || code >= 32 }).join('').replace(/\s+/g, ' ').trim()

export function buildFeeds(posts, origin = 'https://lextures.com') {
  const base = origin.replace(/\/$/, '')
  const items = posts.filter(post => !post.noindex).sort((a, b) => String(b.date).localeCompare(String(a.date))).slice(0, 20)
  const rssItems = items.map(post => {
    const link = post.canonicalOverride || `${base}${post.path}`
    return `<item><title>${xmlEscape(post.title)}</title><link>${xmlEscape(link)}</link><guid isPermaLink="true">${xmlEscape(link)}</guid><description>${xmlEscape(plainText(post.description))}</description><author>${xmlEscape(post.authorName || post.author)}</author><pubDate>${new Date(post.publishedAt || `${post.date}T00:00:00Z`).toUTCString()}</pubDate></item>`
  }).join('')
  const rss = `<?xml version="1.0" encoding="UTF-8"?>\n<rss version="2.0"><channel><title>Lextures Blog</title><link>${base}/blog</link><description>Essays on learning, assessment, and education technology.</description>${rssItems}</channel></rss>\n`
  const json = JSON.stringify({ version: 'https://jsonfeed.org/version/1.1', title: 'Lextures Blog', home_page_url: `${base}/blog`, feed_url: `${base}/blog/feed.json`, items: items.map(post => { const url = post.canonicalOverride || `${base}${post.path}`; return { id: url, url, title: post.title, content_text: plainText(post.content || post.description), summary: plainText(post.description), date_published: post.publishedAt || `${post.date}T00:00:00Z`, authors: [{ name: post.authorName || post.author }] } }) }, null, 2) + '\n'
  return { rss, json, itemCount: items.length }
}

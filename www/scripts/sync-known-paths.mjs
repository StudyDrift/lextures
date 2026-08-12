import { readFile } from 'node:fs/promises'
const apiBase = (process.env.API_BASE || 'https://self.lextures.com').replace(/\/$/, '')
const token = process.env.CONTENT_KNOWN_PATHS_TOKEN
if (!token) { console.warn('[known-paths] WARN: token unset; skipping sync'); process.exit(0) }
try {
  const manifest = JSON.parse(await readFile(process.env.SEO_MANIFEST_PATH || 'dist/.seo-manifest.json', 'utf8'))
  const response = await fetch(`${apiBase}/api/v1/admin/marketing/known-paths`, { method: 'POST', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ paths: manifest.urls.map(url => url.path) }) })
  if (!response.ok) throw new Error(`HTTP ${response.status}`)
  console.log(`[known-paths] synced ${manifest.urls.length} generated paths`)
} catch (error) { console.warn(`[known-paths] WARN: sync failed: ${error.message || error}`) }

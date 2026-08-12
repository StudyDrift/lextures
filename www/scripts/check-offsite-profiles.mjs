import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '../..')

export function claimedSameAsUrls(markdown) {
  return markdown
    .split('\n')
    .filter(line => line.startsWith('|') && /\|\s*claimed\s*\|/.test(line) && /\|\s*yes\s*\|/.test(line))
    .map(line => line.split('|').map(value => value.trim())[3])
    .filter(value => /^https:\/\//.test(value))
}

export function configuredSameAsUrls(entitySource, siteLinksSource) {
  const block = entitySource.match(/VERIFIED_SAME_AS[^=]*=\s*\[([\s\S]*?)\]/)?.[1] ?? ''
  const uncommented = block.split('\n').filter(line => !line.trim().startsWith('//')).join('\n')
  const urls = [...uncommented.matchAll(/['"](https:\/\/[^'"]+)['"]/g)].map(match => match[1])
  if (/SITE_LINKS\.github/.test(uncommented)) {
    const github = siteLinksSource.match(/github:\s*['"](https:\/\/[^'"]+)['"]/)?.[1]
    if (github) urls.push(github)
  }
  return [...new Set(urls)].sort()
}

export function validateSameAs({ profilesMarkdown, entitySource, siteLinksSource }) {
  const claimed = claimedSameAsUrls(profilesMarkdown).sort()
  const configured = configuredSameAsUrls(entitySource, siteLinksSource)
  const missingFromRegister = configured.filter(url => !claimed.includes(url))
  const missingFromSchema = claimed.filter(url => !configured.includes(url))
  return { claimed, configured, missingFromRegister, missingFromSchema }
}

export function runCheck(root = repoRoot) {
  const result = validateSameAs({
    profilesMarkdown: readFileSync(path.join(root, 'docs/plan/seo/offsite/profiles.md'), 'utf8'),
    entitySource: readFileSync(path.join(root, 'www/src/lib/schema/entity.ts'), 'utf8'),
    siteLinksSource: readFileSync(path.join(root, 'www/src/lib/site-links.ts'), 'utf8'),
  })
  if (result.missingFromRegister.length || result.missingFromSchema.length) {
    throw new Error([
      `sameAs/profile register mismatch.`,
      `Missing from claimed register: ${result.missingFromRegister.join(', ') || 'none'}`,
      `Missing from schema: ${result.missingFromSchema.join(', ') || 'none'}`,
    ].join('\n'))
  }
  return result
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const result = runCheck()
  console.log(`Off-site profile check passed (${result.configured.length} verified sameAs URL).`)
}

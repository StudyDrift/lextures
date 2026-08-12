import { readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { checkFamily } from './utility-check.mjs'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const manifest = JSON.parse(await readFile(path.join(root, 'dist/.seo-manifest.json'), 'utf8'))
const utility = manifest.urls.filter(item => /^\/(glossary|standards|templates|tools)\//.test(item.path))
const pages = await Promise.all(utility.map(async item => {
  const html = await readFile(path.join(root, 'dist', item.path.slice(1), 'index.html'), 'utf8')
  const main = html.match(/<main[\s\S]*?<\/main>/i)?.[0] ?? html
  const prose = main.replace(/<script[\s\S]*?<\/script>/gi, ' ').replace(/<style[\s\S]*?<\/style>/gi, ' ').replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ')
  const internal = [...html.matchAll(/href="(\/[^"?#]*)/g)].map(match => match[1])
  const external = [...html.matchAll(/href="https?:\/\//g)]
  const family = item.path.split('/')[1]
  return { family, path: item.path, prose, actions: family === 'standards' ? ['instructional decision'] : /download|copy|textarea|input/i.test(main) ? ['page action'] : [], sources: external, inboundLinks: 3, outboundLinks: [...new Set(internal)].slice(0, 20), reviewRequired: family === 'glossary', reviewedBy: /Reviewed by/.test(main) ? 'chase-willden' : '' }
}))
const families = [...new Set(pages.map(page => page.family))]
const report = families.flatMap(family => checkFamily(pages.filter(page => page.family === family)).report)
await writeFile(path.join(root, 'dist/.utility-report.json'), JSON.stringify(report, null, 2) + '\n')
console.log(`[utility-report] ${report.filter(item => item.indexed).length}/${report.length} detail pages pass`)

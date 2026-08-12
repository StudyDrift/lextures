#!/usr/bin/env node
import { createHash } from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const serverRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = path.resolve(serverRoot, '..')
const archiveSha = '079aacf8fef3b171a54b2063dce723471714b824'
const outputPath = path.join(serverRoot, 'migrations/485_marketing_content_seed.sql')

function git(...args) {
  return execFileSync('git', ['-C', repoRoot, ...args], { encoding: 'utf8' }).trimEnd()
}

function archived(file) {
  return git('show', `${archiveSha}:${file}`)
}

function splitList(value, separator = ',') {
  const trimmed = value.trim().replace(/^\[|\]$/g, '')
  if (!trimmed) return []
  return trimmed.split(separator).map(item => item.trim().replace(/^["']|["']$/g, '')).filter(Boolean)
}

function frontmatter(file, raw) {
  const normalized = raw.replace(/\r\n/g, '\n')
  const match = normalized.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/)
  if (!match) throw new Error(`${file}: missing front matter`)
  const values = {}
  const lists = {}
  for (const line of match[1].split('\n')) {
    if (!line.trim() || line.trimStart().startsWith('#')) continue
    const colon = line.indexOf(':')
    if (colon < 1) throw new Error(`${file}: invalid front matter line: ${line}`)
    const key = line.slice(0, colon).trim()
    const value = line.slice(colon + 1).trim().replace(/^["']|["']$/g, '')
    values[key] = value
    if (/^\[.*\]$/.test(value)) lists[key] = splitList(value)
    else if (key === 'citations') lists[key] = splitList(value, '|')
  }
  return { values, lists, body: match[2].trim() }
}

function lastmod(file, metadata) {
  if (metadata.updated) return { value: metadata.updated, source: 'frontmatter' }
  const logged = git('log', '-1', '--format=%cI', archiveSha, '--', file).trim()
  if (logged) return { value: logged, source: 'git' }
  return { value: metadata.date, source: 'published' }
}

const categorySource = archived('www/src/docs/_categories.ts')
const categories = [...categorySource.matchAll(/\['([^']+)',\s*'([^']*)',\s*'([^']*)',\s*([0-9]+),\s*'([^']*)'\]/g)].map(match => ({
  slug: match[1], title: match[2], description: match[3], sortOrder: Number(match[4]), platformPath: match[5],
}))
if (categories.length !== 16) throw new Error(`expected 16 categories, found ${categories.length}`)

const authors = [{
  slug: 'chase-willden',
  name: 'Chase Willden',
  jobTitle: 'Founder',
  bio: 'Founder of Lextures. Builds adaptive learning systems using Item Response Theory, spaced repetition, and open-source LMS infrastructure for schools and homeschool families.',
  knowsAbout: ['adaptive learning', 'Item Response Theory', 'learning management systems', 'open source education software', 'assessment design'],
  links: { sameAs: ['https://github.com/StudyDrift'], consentRecordedAt: '2026-08-11' },
  status: 'active',
}]

const files = git('ls-tree', '-r', '--name-only', archiveSha, '--', 'www/src/blog', 'www/src/docs')
  .split('\n')
  .filter(file => /^www\/src\/blog\/[^/]+\.md$/.test(file) || /^www\/src\/docs\/[^/]+\/[^/]+\.md$/.test(file))

const articles = files.map(file => {
  const kind = file.includes('/blog/') ? 'blog' : 'doc'
  const slug = path.basename(file, '.md')
  const parsed = frontmatter(file, archived(file))
  const meta = parsed.values
  const category = kind === 'doc' ? (meta.category || path.basename(path.dirname(file))) : null
  const updated = lastmod(file, meta)
  const metadata = {}
  for (const key of ['contentContract', 'supportTicketThemes']) {
    if (meta[key]) metadata[key] = parsed.lists[key] || meta[key]
  }
  const sourceHash = createHash('sha256').update(JSON.stringify({ kind, slug, category, meta, body: parsed.body })).digest('hex')
  return {
    kind, slug, category,
    path: kind === 'blog' ? `/blog/${slug}` : `/docs/${category}/${slug}`,
    title: meta.title || '', description: meta.description || '', bodyMd: parsed.body,
    author: meta.author || 'chase-willden', reviewer: meta.reviewedBy || null,
    publishedAt: meta.date || null, contentUpdatedAt: updated.value || meta.date || null,
    reviewedAt: meta.reviewedAt || null, reviewDueOn: meta.reviewDue || null,
    primaryQuestion: meta.primaryQuestion || '', cluster: meta.cluster || '', pillar: meta.pillar || '',
    briefRef: meta.briefRef || '', verifiedAgainst: meta.verifiedAgainst || '',
    keywords: parsed.lists.keywords || [], relatedTo: parsed.lists.relatedTo || [],
    roles: parsed.lists.roles || [], segments: parsed.lists.segments || [], citations: parsed.lists.citations || [],
    extra: { import: { sourcePath: file, gitSha: archiveSha, importedAt: '2026-08-12T00:00:00Z', lastmodSource: updated.source, sourceHash, importedRevision: 1, metadata }, seedMigration: '485' },
  }
})
if (articles.length !== 70) throw new Error(`expected 70 articles, found ${articles.length}`)
if (articles.filter(article => article.kind === 'blog').length !== 5) throw new Error('expected 5 blog articles')
if (articles.some(article => !article.title || !article.description || !article.publishedAt)) throw new Error('seed article missing required metadata')
if (articles.some(article => !authors.some(author => author.slug === article.author))) throw new Error('seed article has unknown author')
if (articles.some(article => article.category && !categories.some(category => category.slug === article.category))) throw new Error('seed article has unknown category')

const payload = JSON.stringify({ authors, categories, articles })
const sql = `-- MC.15 — seed the archived file-based marketing corpus during normal deployment.
-- Generated by server/scripts/generate-marketing-content-seed.mjs from ${archiveSha}.
-- Do not edit the JSON payload by hand; regenerate it and review the diff.

CREATE TEMP TABLE mc_seed_payload ON COMMIT DROP AS
SELECT $mc_seed$${payload}$mc_seed$::jsonb AS value;

INSERT INTO marketing.content_authors (slug,name,job_title,bio,knows_about,links,status)
SELECT a->>'slug',a->>'name',a->>'jobTitle',a->>'bio',
       ARRAY(SELECT jsonb_array_elements_text(a->'knowsAbout')),a->'links',a->>'status'
FROM mc_seed_payload p CROSS JOIN LATERAL jsonb_array_elements(p.value->'authors') a
ON CONFLICT (slug) DO NOTHING;

INSERT INTO marketing.content_categories (id,slug,locale,title,description,sort_order,platform_path)
SELECT md5('mc-category:en:'||(c->>'slug'))::uuid,c->>'slug','en',c->>'title',c->>'description',
       (c->>'sortOrder')::integer,c->>'platformPath'
FROM mc_seed_payload p CROSS JOIN LATERAL jsonb_array_elements(p.value->'categories') c
ON CONFLICT DO NOTHING;

INSERT INTO marketing.content_articles (
  id,kind,slug,locale,translation_group_id,category_id,path,title,description,body_md,status,
  author_slug,reviewer_slug,published_at,first_published_at,content_updated_at,reviewed_at,
  review_due_on,primary_question,cluster,pillar,brief_ref,verified_against,keywords,related_to,
  roles,segments,citations,quality_report,extra,revision_no
)
SELECT
  md5('mc-article:'||(a->>'kind')||':en:'||(a->>'slug'))::uuid,
  a->>'kind',a->>'slug','en',md5('mc-translation:'||(a->>'kind')||':en:'||(a->>'slug'))::uuid,
  CASE WHEN a->>'category' IS NULL THEN NULL ELSE md5('mc-category:en:'||(a->>'category'))::uuid END,
  a->>'path',a->>'title',a->>'description',a->>'bodyMd','published',a->>'author',
  NULLIF(a->>'reviewer',''),(a->>'publishedAt')::timestamptz,(a->>'publishedAt')::timestamptz,
  (a->>'contentUpdatedAt')::timestamptz,NULLIF(a->>'reviewedAt','')::date,
  NULLIF(a->>'reviewDueOn','')::date,a->>'primaryQuestion',a->>'cluster',a->>'pillar',
  a->>'briefRef',a->>'verifiedAgainst',ARRAY(SELECT jsonb_array_elements_text(a->'keywords')),
  ARRAY(SELECT jsonb_array_elements_text(a->'relatedTo')),ARRAY(SELECT jsonb_array_elements_text(a->'roles')),
  ARRAY(SELECT jsonb_array_elements_text(a->'segments')),ARRAY(SELECT jsonb_array_elements_text(a->'citations')),
  '{"grandfathered":true,"source":"migration-485"}'::jsonb,a->'extra',1
FROM mc_seed_payload p CROSS JOIN LATERAL jsonb_array_elements(p.value->'articles') a
ON CONFLICT DO NOTHING;

INSERT INTO marketing.content_revisions (id,article_id,revision_no,body_md,metadata,change_note,status_after)
SELECT md5('mc-revision:'||a.id::text||':1')::uuid,a.id,1,a.body_md,to_jsonb(a)-'search_tsv',
       'seeded from archived marketing content at ${archiveSha}','published'
FROM marketing.content_articles a
WHERE a.extra->>'seedMigration'='485'
ON CONFLICT (article_id,revision_no) DO NOTHING;

INSERT INTO marketing.content_route_hints (route_prefix,article_id,position)
SELECT h.prefix,a.id,h.position
FROM (VALUES
  ('/courses','finding-your-course',1),('/courses','navigating-the-course-interface',2),
  ('/courses','creating-a-new-course',3),('/quiz','navigating-the-course-interface',1),
  ('/gradebook','navigating-the-course-interface',1),('/settings','finding-your-course',1),
  ('/inbox','navigating-the-course-interface',1)
) h(prefix,slug,position)
JOIN marketing.content_articles a ON a.kind='doc' AND a.locale='en' AND a.slug=h.slug AND a.deleted_at IS NULL
ON CONFLICT (route_prefix,article_id) DO NOTHING;
`

if (process.argv.includes('--check')) {
  const current = await readFile(outputPath, 'utf8')
  if (current !== sql) {
    console.error('485_marketing_content_seed.sql is stale; regenerate it')
    process.exit(1)
  }
} else {
  await writeFile(outputPath, sql)
  console.log(`Wrote ${path.relative(repoRoot, outputPath)} with ${articles.length} articles and ${categories.length} categories.`)
}

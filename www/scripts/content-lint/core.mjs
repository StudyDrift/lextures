import path from 'node:path'

export const ALLOWED_FRONTMATTER = new Set([
  'title', 'description', 'published', 'date', 'updated', 'author', 'cluster',
  'primaryQuestion', 'keywords', 'reviewedBy', 'reviewedAt', 'relatedTo', 'noindex',
  'citations', 'locale', 'contentContract', 'pillar', 'briefRef', 'reviewDue', 'contributor',
  'category', 'roles', 'segments', 'verifiedAgainst', 'supportTicketThemes',
])

export const COMPONENTS = new Set([
  'key-takeaways', 'answer', 'definition', 'comparison-table', 'steps', 'faq',
  'callout', 'stat', 'sources',
])

const words = value => String(value || '').replace(/<[^>]*>|[#>*_`[\](){}|:-]/g, ' ').trim().split(/\s+/).filter(Boolean)
const lineOf = (source, offset) => source.slice(0, offset).split(/\r?\n/).length

export function parseContent(source, file = '') {
  const match = source.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/)
  const errors = []
  if (!match) return { meta: {}, body: source, errors: [{ file, line: 1, rule: 'frontmatter', message: 'Front matter is required.', fix: 'Add a YAML front-matter block.' }] }
  const meta = {}
  for (const [index, raw] of match[1].split(/\r?\n/).entries()) {
    if (!raw.trim() || /^\s/.test(raw)) continue
    const colon = raw.indexOf(':')
    if (colon < 1) continue
    const key = raw.slice(0, colon).trim()
    let value = raw.slice(colon + 1).trim().replace(/^['"]|['"]$/g, '')
    if (!ALLOWED_FRONTMATTER.has(key)) errors.push({ file, line: index + 2, rule: 'frontmatter-unknown', message: `Unknown front-matter key “${key}”.`, fix: 'Remove it or add it to the documented schema.' })
    if (/^\[.*\]$/.test(value)) value = value.slice(1, -1).split(',').map(v => v.trim().replace(/^['"]|['"]$/g, '')).filter(Boolean)
    meta[key] = value
  }
  return { meta, body: match[2].trim(), errors }
}

export function blocks(body, name) {
  const found = []
  const re = new RegExp(`^:::[ \\t]*${name}(?:[ \\t]+([^\\n]*))?\\r?\\n([\\s\\S]*?)^:::[ \\t]*$`, 'gim')
  for (const m of body.matchAll(re)) found.push({ args: (m[1] || '').trim(), content: m[2].trim(), line: lineOf(body, m.index) })
  return found
}

function issue(file, line, rule, message, fix, severity = 'error') { return { file, line, rule, message, fix, severity } }
function questionHeading(text) { return /\?$/.test(text) || /^(how|what|when|where|why|who|which|can|should|does|do|is|are)\b/i.test(text) }
function readingGrade(text) {
  const sentences = Math.max(1, (text.match(/[.!?]+(?:\s|$)/g) || []).length)
  const all = words(text); const syllables = all.reduce((sum, word) => sum + Math.max(1, (word.toLowerCase().match(/[aeiouy]+/g) || []).length - (/e$/.test(word) ? 1 : 0)), 0)
  return Math.max(0, 0.39 * (all.length / sentences) + 11.8 * (syllables / Math.max(1, all.length)) - 15.59)
}

export function analyzeContent(source, file = '', { enforce = false } = {}) {
  const parsed = parseContent(source, file); const { meta, body } = parsed
  const issues = [...parsed.errors]
  const required = ['title', 'description', 'updated', 'author', 'cluster', 'primaryQuestion', 'keywords']
  if (!meta.published && !meta.date) issues.push(issue(file, 2, 'frontmatter-required', 'Missing “published”.', 'Add published: YYYY-MM-DD.'))
  for (const key of required) if (!meta[key]) issues.push(issue(file, 2, 'frontmatter-required', `Missing “${key}”.`, `Add ${key}: … to front matter.`))
  if (meta.description && (meta.description.length < 120 || meta.description.length > 160)) issues.push(issue(file, 2, 'description-length', `Description is ${meta.description.length} characters; target is 120–160.`, 'Rewrite it as a distinct 120–160 character summary.', 'warning'))
  if (meta.description === meta.title) issues.push(issue(file, 2, 'description-duplicate', 'Description duplicates the title.', 'Write a standalone summary.'))
  const imports = body.match(/^\s*(?:import|export)\s.+$/m)
  if (imports) issues.push(issue(file, lineOf(body, imports.index), 'security-policy', 'Imports and exports are forbidden in content.', 'Use an allowlisted content directive.'))
  const rawHtml = body.match(/<\/?[A-Za-z][^>]*>/)
  if (rawHtml) issues.push(issue(file, lineOf(body, rawHtml.index), 'security-policy', 'Raw HTML and JSX are forbidden in content.', 'Use Markdown or an allowlisted content directive.'))
  for (const match of body.matchAll(/^:::[ \\t]*([\w-]+)/gim)) if (!COMPONENTS.has(match[1].toLowerCase())) issues.push(issue(file, lineOf(body, match.index), 'security-policy', `Directive “${match[1]}” is not allowlisted.`, `Use one of: ${[...COMPONENTS].join(', ')}.`))

  const takeaways = blocks(body, 'key-takeaways')[0]
  const takeawayCount = takeaways ? (takeaways.content.match(/^\s*[-*]\s+/gm) || []).length : 0
  if (!takeaways || takeawayCount < 3 || takeawayCount > 5) issues.push(issue(file, takeaways?.line || 1, 'key-takeaways', `Key takeaways must contain 3–5 bullets; found ${takeawayCount}.`, 'Add a :::key-takeaways block with 3–5 complete conclusions.'))
  const answer = blocks(body, 'answer')[0]; const answerWords = words(answer?.content).length
  if (!answer) issues.push(issue(file, 1, 'answer-box', 'A direct answer is required.', 'Add a :::answer block that answers primaryQuestion in 40–60 words.'))
  else if (answerWords < 40 || answerWords > 60) issues.push(issue(file, answer.line, 'answer-length', `Answer is ${answerWords} words; target is 40–60.`, 'Tighten or expand the answer to 40–60 words.'))
  const headings = [...body.matchAll(/^##\s+(.+)$/gm)].filter(m => !/^Frequently asked questions/i.test(m[1]))
  const questionHeadings = headings.filter(m => questionHeading(m[1])).length
  for (const h of headings) if (/^The\s+/i.test(h[1]) && !questionHeading(h[1])) issues.push(issue(file, lineOf(body, h.index), 'narrative-heading', `Narrative heading “${h[1]}”.`, 'Phrase it as a reader question or a noun phrase naming the answer.', 'warning'))
  const faq = blocks(body, 'faq')[0]; const faqCount = faq ? (faq.content.match(/^###\s+.+\?\s*$/gm) || []).length : 0
  if (!faq || faqCount < 3 || faqCount > 6) issues.push(issue(file, faq?.line || 1, 'faq', `FAQ must contain 3–6 questions; found ${faqCount}.`, 'Add a :::faq block with 3–6 “### Question?” entries and 40–80 word answers.'))
  const statMatches = [...body.matchAll(/\b\d+(?:\.\d+)?%|\b\d{2,}\s+(?:teachers|students|schools|learners|percent)\b/gi)]
  const citedLines = new Set([...body.matchAll(/\[\^\d+\]/g)].map(m => lineOf(body, m.index)))
  for (const stat of statMatches) if (!citedLines.has(lineOf(body, stat.index))) issues.push(issue(file, lineOf(body, stat.index), 'uncited-statistic', `Uncited numeric claim “${stat[0]}”.`, 'Add an inline [^n] citation to a primary source.'))
  const citations = [...body.matchAll(/^\[\^\d+\]:\s+(https?:\/\/\S+)/gm)]
  for (const c of citations) if (/^javascript:/i.test(c[1])) issues.push(issue(file, lineOf(body, c.index), 'security-policy', 'Unsafe source URL.', 'Use an HTTPS primary-source URL.'))
  const internalLinks = [...body.matchAll(/\[([^\]]+)\]\((\/(?!\/)[^)]+)\)/g)].filter(m => !/^(here|read more|click here)$/i.test(m[1].trim()))
  const prose = body.replace(/^:::[\s\S]*?^:::\s*$/gm, '').replace(/^#{1,6}.+$/gm, '').split(/\n\s*\n/).map(p => words(p).length).filter(n => n >= 20)
  const meanPassage = prose.length ? prose.reduce((a, b) => a + b, 0) / prose.length : 0
  const hasStructure = /^(?:[-*]|\d+\.)\s+/m.test(body) || blocks(body, 'comparison-table').length || blocks(body, 'steps').length
  const signals = {
    keyTakeaways: { weight: 1, earned: takeawayCount >= 3 && takeawayCount <= 5 ? 1 : 0 },
    answerBox: { weight: 1.5, earned: answerWords >= 40 && answerWords <= 60 ? 1.5 : 0 },
    questionHeadings: { weight: 1.5, earned: headings.length && questionHeadings / headings.length >= .6 ? 1.5 : 0 },
    passageLength: { weight: 1.5, earned: meanPassage >= 120 && meanPassage <= 180 ? 1.5 : 0 },
    citations: { weight: 2, earned: citations.length >= Math.max(1, Math.ceil(words(body).length / 400)) ? 2 : 0 },
    structuredContent: { weight: 1, earned: hasStructure ? 1 : 0 },
    faq: { weight: 1, earned: faqCount >= 3 && faqCount <= 6 ? 1 : 0 },
    internalLinks: { weight: .5, earned: internalLinks.length >= 3 ? .5 : 0 },
  }
  const score = Object.values(signals).reduce((sum, s) => sum + s.earned, 0)
  if (enforce && score < 6) issues.push(issue(file, 1, 'extractability-score', `Extractability score ${score.toFixed(1)} is below 6.0.`, 'Address the failed signal breakdown.'))
  else if (score < 8) issues.push(issue(file, 1, 'extractability-score', `Extractability score ${score.toFixed(1)} is below 8.0.`, 'Address the failed signal breakdown.', 'warning'))
  const grade = readingGrade(body)
  const target = file.includes(`${path.sep}docs${path.sep}`) ? [8, 10] : [9, 11]
  if (grade < target[0] || grade > target[1]) issues.push(issue(file, 1, 'reading-level', `Flesch–Kincaid grade is ${grade.toFixed(1)}; target is ${target[0]}–${target[1]}.`, 'Shorten sentences or clarify vocabulary where editorially appropriate.', 'warning'))
  // Existing Markdown remains visible in the report but is grandfathered until its refresh.
  // New .mdx files (and files opting into contentContract) enforce errors at build time.
  if (!enforce && !meta.contentContract) for (const item of issues) item.severity = 'warning'
  return { file, grandfathered: !enforce && !meta.contentContract, score, signals, wordCount: words(body).length, passageLengths: prose, citationCount: citations.length, readingLevel: Number(grade.toFixed(1)), definitions: blocks(body, 'definition').map(d => ({ term: d.args.replace(/^term=['"]?|['"]?$/g, ''), definition: d.content })), issues, meta }
}

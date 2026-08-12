import fs from 'node:fs'
import path from 'node:path'

export const ENGINES = ['chatgpt', 'google_ai', 'gemini', 'perplexity', 'claude', 'copilot']
export const SOURCES = ['gsc', 'bing', 'ga4', 'crux', 'crawl', 'crm', 'mentions', 'ai_visibility']
export function readJson(file) { return JSON.parse(fs.readFileSync(file, 'utf8')) }
export function brandRegex(text) { return new RegExp(text.split('\n').map(v => v.trim()).filter(v => v && !v.startsWith('#')).map(v => `(?:${v})`).join('|'), 'i') }
export function pageFamily(value) {
  const pathname = value.startsWith('http') ? new URL(value).pathname : value
  for (const family of ['/docs', '/blog', '/resources', '/compare', '/courses', '/glossary', '/platform']) if (pathname === family || pathname.startsWith(`${family}/`)) return family
  return 'other'
}
export function validatePrompts(config) {
  const errors = [], ids = new Set(), categories = new Map()
  if (!config.version) errors.push('prompt set has no version')
  if (config.prompts?.length !== 60) errors.push(`expected 60 prompts, found ${config.prompts?.length ?? 0}`)
  for (const prompt of config.prompts ?? []) {
    if (!prompt.id || ids.has(prompt.id)) errors.push(`missing or duplicate prompt id: ${prompt.id}`)
    ids.add(prompt.id); categories.set(prompt.category, (categories.get(prompt.category) ?? 0) + 1)
    if (!prompt.text?.trim()) errors.push(`prompt ${prompt.id} has no text`)
  }
  for (const category of ['category','capability','comparison','brand','problem','segment']) if (categories.get(category) !== 10) errors.push(`${category} must contain 10 prompts`)
  return errors
}
export function visibilitySummary(rows, promptConfig) {
  const ids = new Set(promptConfig.prompts.map(p => p.id)), byEngine = Object.fromEntries(ENGINES.map(engine => [engine, { answers:0, mentions:0 }])), competitors = new Map(), citedUrls = new Set(), entityErrors = []
  for (const row of rows) {
    if (!ids.has(row.prompt_id) || !byEngine[row.engine]) continue
    byEngine[row.engine].answers++; if (row.mentioned) byEngine[row.engine].mentions++
    for (const name of row.competitors ?? []) competitors.set(name, (competitors.get(name) ?? 0) + 1)
    for (const url of row.cited_urls ?? []) citedUrls.add(url)
    if (row.entity_accuracy === false) entityErrors.push({prompt_id:row.prompt_id,engine:row.engine,date:row.date,errors:row.entity_errors ?? []})
  }
  const answers = Object.values(byEngine).reduce((sum,v) => sum+v.answers,0), mentions = Object.values(byEngine).reduce((sum,v) => sum+v.mentions,0)
  return {byEngine,answers,mentions,shareOfVoice:answers ? mentions/answers : null,competitors:[...competitors.entries()].sort((a,b)=>b[1]-a[1]),citedUrls:[...citedUrls],entityErrors}
}
export function crawlAlerts(current, previous=[]) {
  const alerts=[], prev = new Map(previous.map(row => [row.bot,row.requests]))
  for (const row of current) {
    const prior=prev.get(row.bot), non200=(row.status_3xx??0)+(row.status_4xx??0)+(row.status_5xx??0)
    if (prior>0 && row.requests<prior*.5) alerts.push(`${row.bot}: crawl rate dropped ${Math.round((1-row.requests/prior)*100)}%`)
    if (row.requests && non200/row.requests>.05) alerts.push(`${row.bot}: non-200 rate ${Math.round(non200/row.requests*100)}%`)
    if (row.page_family==='sitemap' && non200) alerts.push(`${row.bot}: sitemap returned non-200`)
  }
  return alerts
}
export function freshness(statuses, now=new Date()) {
  return SOURCES.map(source => { const status=statuses.find(v=>v.source===source); if (!status?.last_success) return {source,state:'missing',lastSuccess:null}; const age=(now-new Date(status.last_success))/36e5, max=source==='ai_visibility'||source==='crux'?192:48; return {source,state:status.ok===false?'failed':age>max?'stale':'fresh',lastSuccess:status.last_success} })
}
export function readJsonlIfPresent(file) {
  if (!fs.existsSync(file)) return []
  return fs.readFileSync(file,'utf8').split('\n').filter(Boolean).map((line,index)=>{try{return JSON.parse(line)}catch{throw new Error(`${path.basename(file)}:${index+1}: invalid JSON`)}})
}

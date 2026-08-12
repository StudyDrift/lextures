#!/usr/bin/env node
import fs from 'node:fs'; import path from 'node:path'
import {brandRegex,crawlAlerts,freshness,pageFamily,readJson,readJsonlIfPresent,validatePrompts,visibilitySummary} from './core.mjs'
const root=path.resolve(import.meta.dirname,'../../..'), defs=path.join(root,'docs/plan/seo/measurement'), data=process.env.SEO_MEASUREMENT_DATA_DIR||path.join(defs,'data'), prompts=readJson(path.join(defs,'prompts.yaml')), command=process.argv[2]||'validate'
const errors=validatePrompts(prompts), terms=brandRegex(fs.readFileSync(path.join(defs,'brand-terms.txt'),'utf8')), channels=readJson(path.join(defs,'channels.yaml'))
if (!channels.ai_assistant?.referrer_hosts?.includes('perplexity.ai')) errors.push('AI channels must include perplexity.ai')
if (!terms.test('Lextures LMS')||pageFamily('/docs/start')!=='/docs') errors.push('classification definitions invalid')
if(errors.length){errors.forEach(v=>console.error(`ERROR ${v}`));process.exitCode=1}
else if(command==='validate') console.log(`OK ${prompts.prompts.length} prompts (${prompts.version}), ${channels.ai_assistant.referrer_hosts.length} AI referrers`)
else if(command==='report'){
 const ai=visibilitySummary(readJsonlIfPresent(path.join(data,'ai_visibility.jsonl')),prompts), states=freshness(readJsonlIfPresent(path.join(data,'freshness.jsonl'))), alerts=crawlAlerts(readJsonlIfPresent(path.join(data,'crawl-current.jsonl')),readJsonlIfPresent(path.join(data,'crawl-previous.jsonl')))
 console.log(`# SEO measurement dashboard\n\nGenerated: ${new Date().toISOString()}\n\n## Data freshness\n\n| Source | State | Last success |\n|---|---|---|`); states.forEach(v=>console.log(`| ${v.source} | ${v.state} | ${v.lastSuccess??'not connected'} |`))
 console.log('\n## AI share of voice\n\n| Engine | Mentions | Answers | Share |\n|---|---:|---:|---:|'); Object.entries(ai.byEngine).forEach(([e,v])=>console.log(`| ${e} | ${v.mentions} | ${v.answers} | ${v.answers?(v.mentions/v.answers*100).toFixed(1)+'%':'not connected'} |`))
 console.log(`\nOverall: ${ai.shareOfVoice===null?'not connected':(ai.shareOfVoice*100).toFixed(1)+'%'} (${ai.mentions}/${ai.answers})\n\nCited URLs: ${ai.citedUrls.length}; entity errors: ${ai.entityErrors.length}.\n\n## Crawl alerts\n\n${alerts.length?alerts.map(v=>`- ${v}`).join('\n'):'No alerts in supplied observations (or crawl source not connected).'}\n\n## Required weekly rollup\n\nIndexed pages, non-brand clicks, top movers, referring domains, CWV pass rate, conversions, and per-plan page-family results come from source exports. Missing exports remain “not connected”; they are never rendered as zero.`)
} else {console.error('usage: cli.mjs validate|report');process.exitCode=2}

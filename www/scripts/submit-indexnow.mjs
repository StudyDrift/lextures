#!/usr/bin/env node
/**
 * Post-deploy IndexNow + Google sitemap ping (SEO.2 FR-16…FR-19).
 *
 * Compares the current dist/.seo-manifest.json to a previous copy and POSTs
 * changed/new URLs to IndexNow. Failures warn but do not exit non-zero
 * (deploy must not fail on index submission).
 *
 * Env:
 *   SITE_ORIGIN — default https://lextures.com
 *   SEO_MANIFEST_PATH — current manifest (default dist/.seo-manifest.json)
 *   SEO_MANIFEST_PREV_PATH — previous deploy manifest (optional)
 *   INDEXNOW_KEY — override key (default from manifest / constant)
 *   DRY_RUN=1 — log only, no network
 */

import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  batchIndexNowUrls,
  buildIndexNowBody,
  diffManifestUrls,
  INDEXNOW_BATCH_SIZE,
} from './seo-artifacts.mjs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')

const SITE_ORIGIN = (process.env.SITE_ORIGIN || 'https://lextures.com').replace(/\/$/, '')
const MANIFEST_PATH =
  process.env.SEO_MANIFEST_PATH || path.join(ROOT, 'dist/.seo-manifest.json')
const PREV_PATH = process.env.SEO_MANIFEST_PREV_PATH || ''
const DRY_RUN = process.env.DRY_RUN === '1' || process.env.DRY_RUN === 'true'

async function loadJson(p) {
  if (!p) return null
  try {
    const raw = await readFile(p, 'utf8')
    return JSON.parse(raw)
  } catch (err) {
    console.warn(`[submit-indexnow] WARN: could not read ${p}: ${err.message || err}`)
    return null
  }
}

async function main() {
  const current = await loadJson(MANIFEST_PATH)
  if (!current?.urls?.length) {
    console.warn('[submit-indexnow] WARN: no current seo-manifest; skipping')
    return
  }

  const prev = PREV_PATH ? await loadJson(PREV_PATH) : null
  let urlList = diffManifestUrls(prev, current)

  // First deploy / no previous: submit all indexable URLs (capped by batching)
  if (!prev) {
    urlList = current.urls
      .filter(u => !(u.robots || '').includes('noindex') && u.path !== '/404')
      .map(u => u.canonical || `${SITE_ORIGIN}${u.path === '/' ? '/' : u.path}`)
    console.log(
      `[submit-indexnow] no previous manifest — submitting all ${urlList.length} indexable URL(s)`,
    )
  } else {
    console.log(`[submit-indexnow] ${urlList.length} changed/new URL(s) vs previous deploy`)
  }

  if (urlList.length === 0) {
    console.log('[submit-indexnow] nothing to submit')
  }

  const host = new URL(SITE_ORIGIN).host
  const key = process.env.INDEXNOW_KEY || current.indexNowKey
  if (!key) {
    console.warn('[submit-indexnow] WARN: no IndexNow key; skipping submission')
  } else {
    const keyLocation = `${SITE_ORIGIN}/${key}.txt`
    const batches = batchIndexNowUrls(urlList, INDEXNOW_BATCH_SIZE)
    let submitted = 0

    for (let i = 0; i < batches.length; i++) {
      const batch = batches[i]
      const body = buildIndexNowBody({ host, key, keyLocation, urlList: batch })
      console.log(
        `[submit-indexnow] batch ${i + 1}/${batches.length}: ${batch.length} URL(s) → api.indexnow.org`,
      )
      if (DRY_RUN) {
        console.log('[submit-indexnow] DRY_RUN sample:', batch.slice(0, 5))
        submitted += batch.length
        continue
      }
      try {
        const res = await fetch('https://api.indexnow.org/indexnow', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json; charset=utf-8',
            'User-Agent': 'lextures-www-indexnow/1.0',
          },
          body: JSON.stringify(body),
        })
        const status = res.status
        // 200/202 success; 422 may mean invalid URL list — warn only
        if (status === 200 || status === 202) {
          console.log(`[submit-indexnow] IndexNow HTTP ${status} (ok)`)
          submitted += batch.length
        } else {
          const text = await res.text().catch(() => '')
          console.warn(
            `[submit-indexnow] WARN: IndexNow HTTP ${status} ${text.slice(0, 200)}`,
          )
        }
      } catch (err) {
        console.warn(`[submit-indexnow] WARN: IndexNow request failed: ${err.message || err}`)
      }
    }
    console.log(`[submit-indexnow] submitted count (ok batches): ${submitted}`)
  }

  // Google sitemap ping (legacy, best-effort)
  const sitemapUrl = encodeURIComponent(`${SITE_ORIGIN}/sitemap.xml`)
  const pingUrl = `https://www.google.com/ping?sitemap=${sitemapUrl}`
  console.log(`[submit-indexnow] Google sitemap ping: ${SITE_ORIGIN}/sitemap.xml`)
  if (!DRY_RUN) {
    try {
      const res = await fetch(pingUrl, {
        method: 'GET',
        headers: { 'User-Agent': 'lextures-www-indexnow/1.0' },
      })
      console.log(`[submit-indexnow] Google ping HTTP ${res.status}`)
    } catch (err) {
      console.warn(`[submit-indexnow] WARN: Google ping failed: ${err.message || err}`)
    }
  } else {
    console.log('[submit-indexnow] DRY_RUN skip Google ping')
  }
}

const isMain =
  process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isMain) {
  main().catch(err => {
    // Never fail deploy
    console.warn('[submit-indexnow] WARN: unexpected error:', err)
    process.exit(0)
  })
}

export { main as submitIndexNow }

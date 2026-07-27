#!/usr/bin/env node
/**
 * CT.5 FR-4 / AC-2 — fail CI when a manifest schema diff is inconsistent with its version bump.
 *
 * Compares each tool's current manifest against the previous git revision (if present).
 * When PREV_MANIFEST_DIR is set, reads sibling manifests from that directory instead.
 */
import { readFileSync, existsSync, readdirSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execSync } from 'node:child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const toolsRoot = join(__dirname, '../../../server/internal/service/contenttools/tools')

function parseSemVer(v) {
  const [maj, min, pat] = String(v).trim().split(/[-+]/)[0].split('.').map(Number)
  return { major: maj, minor: min, patch: pat }
}

function bumpKind(fromVer, toVer) {
  const a = parseSemVer(fromVer)
  const b = parseSemVer(toVer)
  if (a.major === b.major && a.minor === b.minor && a.patch === b.patch) return 'none'
  if (b.major > a.major) return 'major'
  if (b.major < a.major) throw new Error(`version decreased ${fromVer} → ${toVer}`)
  if (b.minor > a.minor) return 'minor'
  if (b.minor < a.minor) throw new Error(`version decreased ${fromVer} → ${toVer}`)
  if (b.patch > a.patch) return 'patch'
  throw new Error(`version decreased ${fromVer} → ${toVer}`)
}

const RANK = { none: 0, patch: 1, minor: 2, major: 3 }

function classifySchemaDiff(oldSchema, newSchema) {
  const findings = []
  const oldProps = oldSchema?.properties ?? {}
  const newProps = newSchema?.properties ?? {}
  const oldReq = new Set(oldSchema?.required ?? [])
  const newReq = new Set(newSchema?.required ?? [])
  for (const name of Object.keys(oldProps)) {
    const path = `$.properties.${name}`
    if (!(name in newProps)) {
      findings.push({ path, kind: 'major', note: 'field removed' })
      continue
    }
    if (!oldReq.has(name) && newReq.has(name)) {
      findings.push({ path, kind: 'major', note: 'became required' })
    }
  }
  for (const name of Object.keys(newProps)) {
    if (name in oldProps) continue
    const path = `$.properties.${name}`
    findings.push({
      path,
      kind: newReq.has(name) ? 'major' : 'minor',
      note: newReq.has(name) ? 'new required field' : 'additive optional field',
    })
  }
  let kind = 'none'
  for (const f of findings) {
    if (RANK[f.kind] > RANK[kind]) kind = f.kind
  }
  return { kind, findings }
}

function loadPrevManifest(toolId) {
  const prevDir = process.env.PREV_MANIFEST_DIR
  if (prevDir) {
    const p = join(prevDir, toolId, 'manifest.json')
    if (existsSync(p)) return JSON.parse(readFileSync(p, 'utf8'))
    return null
  }
  try {
    const raw = execSync(
      `git show HEAD~1:server/internal/service/contenttools/tools/${toolId}/manifest.json`,
      { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] },
    )
    return JSON.parse(raw)
  } catch {
    return null
  }
}

const toolDirs = readdirSync(toolsRoot, { withFileTypes: true })
  .filter((d) => d.isDirectory())
  .map((d) => d.name)

let failed = 0
for (const toolId of toolDirs) {
  const curPath = join(toolsRoot, toolId, 'manifest.json')
  if (!existsSync(curPath)) continue
  const cur = JSON.parse(readFileSync(curPath, 'utf8'))
  const prev = loadPrevManifest(toolId)
  if (!prev) {
    console.log(`OK: ${toolId} (no previous manifest to diff)`)
    continue
  }
  if (prev.version === cur.version) {
    const cfg = classifySchemaDiff(prev.configSchema, cur.configSchema)
    const st = classifySchemaDiff(prev.stateSchema, cur.stateSchema)
    if (RANK[cfg.kind] > 0 || RANK[st.kind] > 0) {
      console.error(
        `FAIL: ${toolId} schema changed without a version bump (config=${cfg.kind}, state=${st.kind})`,
      )
      failed++
    } else {
      console.log(`OK: ${toolId}@${cur.version} unchanged schemas`)
    }
    continue
  }
  const declared = bumpKind(prev.version, cur.version)
  for (const [label, oldS, newS] of [
    ['configSchema', prev.configSchema, cur.configSchema],
    ['stateSchema', prev.stateSchema, cur.stateSchema],
  ]) {
    const { kind, findings } = classifySchemaDiff(oldS, newS)
    if (RANK[declared] < RANK[kind]) {
      const paths = findings
        .filter((f) => RANK[f.kind] >= RANK[kind])
        .map((f) => f.path)
        .join(', ')
      console.error(
        `FAIL: ${toolId} ${label} requires ${kind} but version ${prev.version} → ${cur.version} is ${declared}; offending: ${paths}`,
      )
      failed++
    }
  }
  if (failed === 0) console.log(`OK: ${toolId} ${prev.version} → ${cur.version} (${declared})`)
}

if (failed > 0) process.exit(1)
console.log('Tool schema-diff check passed.')

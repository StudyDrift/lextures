/**
 * E2E.4 — completed-feature traceability and coverage gate.
 *
 * Scans `docs/completed/**` (eligible Markdown stories), validates a reviewed
 * manifest disposition per story, and generates summary reports. Validation is
 * deterministic and does not launch browsers or call models.
 */

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const LIB_DIR = path.dirname(fileURLToPath(import.meta.url))
export const E2E_ROOT = path.resolve(LIB_DIR, '..')
export const REPO_ROOT = path.resolve(E2E_ROOT, '..')

export const DEFAULT_MANIFEST_REL = 'e2e/coverage/completed-feature-manifest.json'
export const DEFAULT_REPORT_REL = 'e2e/coverage/REPORT.md'

/** Coverage classifications (FR-2). */
export const COVERAGE_LEVELS = [
  'journey',
  'smoke',
  'api-contract',
  'covered-by-parent',
  'manual',
  'not-applicable',
  'missing',
] as const

export type CoverageLevel = (typeof COVERAGE_LEVELS)[number]

export const CLIENTS = ['web', 'mobile', 'cli', 'ops', 'docs', 'multi'] as const
export type ClientKind = (typeof CLIENTS)[number]

export const MARKETS = ['K12', 'HE', 'HS', 'ALL'] as const
export type Market = (typeof MARKETS)[number]

export const RISK_LEVELS = ['critical', 'major', 'minor', 'none'] as const
export type RiskLevel = (typeof RISK_LEVELS)[number]

/** Six flag-lifecycle dimensions (FR-4 / AC-3). */
export type FlagCoverage = {
  settingsToggle: boolean | 'n/a'
  disabledState: boolean | 'n/a'
  enabledJourney: boolean | 'n/a'
  authorization: boolean | 'n/a'
  dependency: boolean | 'n/a'
  rollback: boolean | 'n/a'
  /** Optional pointer into E2E.1 / E2E.2 / E2E.3 artifacts. */
  lifecycleFamilyId?: string
  notes?: string
}

export type SpecLink = {
  /** Repo-relative path, e.g. e2e/tests/auth.spec.ts */
  path: string
  /** Optional Playwright test title filter. */
  titles?: string[]
}

export type ManualEvidence = {
  owner: string
  cadence: string
  /** Internal evidence pointer (no external secrets). */
  location: string
}

export type CoverageEntry = {
  /** Stable story ID (Feature ID when known; otherwise filename stem). */
  id: string
  /** Repo-relative path under docs/completed. */
  path: string
  /** Optional aliases for renamed stories (Maintainability NFR). */
  aliases?: string[]
  markets: Market[]
  risk: RiskLevel
  client: ClientKind
  coverage: CoverageLevel
  rationale: string
  owner: string
  specs?: SpecLink[]
  parentStoryId?: string
  flags?: FlagCoverage
  manualEvidence?: ManualEvidence
  /** Required when coverage === 'missing' (AC-5). */
  severity?: RiskLevel
  targetMilestone?: string
  notes?: string
  /** True when a human reviewed the disposition (bootstrap sets true). */
  reviewed: boolean
}

export type CoverageManifest = {
  version: 1
  generatedNote: string
  exclusions: string[]
  entries: CoverageEntry[]
}

/** Explicit exclusion rules for FR-1 (indexes / assets / non-stories). */
export const EXCLUSION_RULES: ReadonlyArray<{
  id: string
  description: string
  match: (repoRelativePath: string) => boolean
}> = [
  {
    id: 'readme-indexes',
    description: 'Section README.md index files',
    match: (p) => path.basename(p).toLowerCase() === 'readme.md',
  },
  {
    id: 'assets-dir',
    description: 'Static assets under docs/completed/assets',
    match: (p) => p.startsWith('docs/completed/assets/') || p === 'docs/completed/assets',
  },
  {
    id: 'non-markdown',
    description: 'Non-Markdown files (images, etc.)',
    match: (p) => !p.endsWith('.md'),
  },
]

export function isExcludedCompletedPath(repoRelativePath: string): boolean {
  const normalized = repoRelativePath.replace(/\\/g, '/')
  return EXCLUSION_RULES.some((rule) => rule.match(normalized))
}

function walkMarkdownFiles(absDir: string, acc: string[] = []): string[] {
  if (!fs.existsSync(absDir)) return acc
  for (const ent of fs.readdirSync(absDir, { withFileTypes: true })) {
    const abs = path.join(absDir, ent.name)
    if (ent.isDirectory()) {
      walkMarkdownFiles(abs, acc)
      continue
    }
    if (ent.isFile() && ent.name.endsWith('.md')) acc.push(abs)
  }
  return acc
}

/** List eligible completed story paths (repo-relative, sorted). */
export function listEligibleCompletedStories(repoRoot: string = REPO_ROOT): string[] {
  const completedRoot = path.join(repoRoot, 'docs/completed')
  const files = walkMarkdownFiles(completedRoot)
  return files
    .map((abs) => path.relative(repoRoot, abs).replace(/\\/g, '/'))
    .filter((rel) => !isExcludedCompletedPath(rel))
    .sort((a, b) => a.localeCompare(b))
}

export function defaultStoryIdFromPath(repoRelativePath: string): string {
  const base = path.basename(repoRelativePath, '.md')
  // BUG-… ids include extra hyphens before the slug.
  const bug = base.match(/^(BUG-[A-Za-z0-9]+-\d+)/)
  if (bug) return bug[1]

  const dash = base.indexOf('-')
  if (dash > 0) {
    const maybeId = base.slice(0, dash)
    // E2E.1, VC.1, VC.M1, M9.1, 16.9, IC01, AP.4, MKT1, T01, W07, AN.1, LH.1, 09, …
    if (
      /^[A-Za-z][A-Za-z0-9]{0,7}(?:\.[A-Za-z]?\d+)+$/i.test(maybeId) ||
      /^[A-Za-z]\d+(?:\.\d+)+$/i.test(maybeId) ||
      /^[A-Za-z]{1,6}\d+$/i.test(maybeId) ||
      /^\d+(?:\.\d+)+$/.test(maybeId) ||
      /^\d{2,}$/.test(maybeId)
    ) {
      return maybeId
    }
  }
  return base
}

export function sectionOfStory(repoRelativePath: string): string {
  const parts = repoRelativePath.replace(/\\/g, '/').split('/')
  // docs/completed/<section>/… or docs/completed/<file>.md
  if (parts.length <= 3) return '_root'
  return parts[2] ?? '_root'
}

export function loadManifest(manifestPath: string): CoverageManifest {
  const raw = fs.readFileSync(manifestPath, 'utf8')
  const parsed = JSON.parse(raw) as CoverageManifest
  return parsed
}

export function resolveManifestPath(
  repoRoot: string = REPO_ROOT,
  rel: string = DEFAULT_MANIFEST_REL,
): string {
  return path.isAbsolute(rel) ? rel : path.join(repoRoot, rel)
}

/** Values that look like embedded credentials (not vocabulary in paths/keys). */
const SECRET_VALUE =
  /(?:bearer\s+[a-z0-9._\-]{12,})|(?:(?:api[_-]?key|password|secret|token|private[_-]?key)\s*[:=]\s*['"]?[^'"\s,]{8,})/i
const EXTERNAL_EVIDENCE = /^https?:\/\//i

/** Known schema keys that may contain security vocabulary without being secrets. */
const ALLOWED_SCHEMA_KEYS = new Set([
  'authorization',
  'settingsToggle',
  'disabledState',
  'enabledJourney',
  'dependency',
  'rollback',
])

function isCoverageLevel(v: unknown): v is CoverageLevel {
  return typeof v === 'string' && (COVERAGE_LEVELS as readonly string[]).includes(v)
}

function push(errors: string[], msg: string): void {
  errors.push(msg)
}

/**
 * Validate the reviewed manifest against the completed-doc tree and spec files.
 * Returns human-readable errors (empty = ok).
 */
export function validateCompletedFeatureCoverage(options?: {
  repoRoot?: string
  manifest?: CoverageManifest
  manifestPath?: string
  /** When true, require reviewed:true on every entry. */
  requireReviewed?: boolean
}): string[] {
  const repoRoot = options?.repoRoot ?? REPO_ROOT
  const manifest =
    options?.manifest ??
    loadManifest(options?.manifestPath ?? resolveManifestPath(repoRoot))
  const requireReviewed = options?.requireReviewed ?? true
  const errors: string[] = []

  if (manifest.version !== 1) {
    push(errors, `manifest.version must be 1 (got ${String(manifest.version)})`)
  }

  const eligible = listEligibleCompletedStories(repoRoot)
  const eligibleSet = new Set(eligible)
  const byPath = new Map<string, CoverageEntry>()
  const byId = new Map<string, CoverageEntry>()

  for (const [i, entry] of manifest.entries.entries()) {
    const prefix = `entries[${i}] (${entry?.id ?? '?'})`

    if (!entry || typeof entry !== 'object') {
      push(errors, `${prefix}: entry must be an object`)
      continue
    }
    if (typeof entry.id !== 'string' || !entry.id.trim()) {
      push(errors, `${prefix}: id is required`)
    }
    if (typeof entry.path !== 'string' || !entry.path.startsWith('docs/completed/')) {
      push(errors, `${prefix}: path must be under docs/completed/`)
    } else if (!eligibleSet.has(entry.path)) {
      if (!fs.existsSync(path.join(repoRoot, entry.path))) {
        push(errors, `${prefix}: broken document link ${entry.path}`)
      } else if (isExcludedCompletedPath(entry.path)) {
        push(errors, `${prefix}: path is excluded by FR-1 rules: ${entry.path}`)
      }
    }
    if (byPath.has(entry.path)) {
      push(errors, `${prefix}: duplicate path ${entry.path} (also ${byPath.get(entry.path)!.id})`)
    } else if (entry.path) {
      byPath.set(entry.path, entry)
    }
    if (entry.id) {
      const prior = byId.get(entry.id)
      if (prior && prior.path !== entry.path) {
        push(errors, `${prefix}: duplicate id ${entry.id} (also ${prior.path})`)
      } else {
        byId.set(entry.id, entry)
      }
    }
    if (!isCoverageLevel(entry.coverage)) {
      push(errors, `${prefix}: unknown classification ${String(entry.coverage)}`)
    }
    if (!Array.isArray(entry.markets) || entry.markets.length === 0) {
      push(errors, `${prefix}: markets must be a non-empty array`)
    }
    if (!(RISK_LEVELS as readonly string[]).includes(entry.risk)) {
      push(errors, `${prefix}: unknown risk ${String(entry.risk)}`)
    }
    if (!(CLIENTS as readonly string[]).includes(entry.client)) {
      push(errors, `${prefix}: unknown client ${String(entry.client)}`)
    }
    if (typeof entry.rationale !== 'string' || !entry.rationale.trim()) {
      push(errors, `${prefix}: rationale is required`)
    }
    if (typeof entry.owner !== 'string' || !entry.owner.trim()) {
      push(errors, `${prefix}: owner is required`)
    }
    if (requireReviewed && entry.reviewed !== true) {
      push(errors, `${prefix}: reviewed must be true`)
    }

    // Secret-like values (Security NFR) — ignore known schema keys / path vocabulary.
    const valueBlob = JSON.stringify(entry, (key, value) => {
      if (ALLOWED_SCHEMA_KEYS.has(key)) return undefined
      if (key === 'path' || key === 'id' || key === 'aliases') return undefined
      return value
    })
    if (SECRET_VALUE.test(valueBlob)) {
      push(errors, `${prefix}: secret-like field content is not allowed in the manifest`)
    }

    const automated =
      entry.coverage === 'journey' ||
      entry.coverage === 'smoke' ||
      entry.coverage === 'api-contract'
    if (automated) {
      if (!entry.specs || entry.specs.length === 0) {
        push(errors, `${prefix}: ${entry.coverage} requires at least one specs[].path`)
      } else {
        for (const spec of entry.specs) {
          if (!spec?.path || typeof spec.path !== 'string') {
            push(errors, `${prefix}: spec link missing path`)
            continue
          }
          const abs = path.join(repoRoot, spec.path)
          if (!fs.existsSync(abs)) {
            push(errors, `${prefix}: broken spec link ${spec.path}`)
          }
        }
      }
    }

    if (entry.coverage === 'covered-by-parent') {
      if (!entry.parentStoryId?.trim()) {
        push(errors, `${prefix}: covered-by-parent requires parentStoryId`)
      }
    }

    if (entry.coverage === 'manual') {
      if (!entry.manualEvidence?.owner?.trim() || !entry.manualEvidence.cadence?.trim()) {
        push(errors, `${prefix}: manual requires manualEvidence.owner and cadence`)
      }
      if (entry.manualEvidence?.location && EXTERNAL_EVIDENCE.test(entry.manualEvidence.location)) {
        push(
          errors,
          `${prefix}: manualEvidence.location must be an internal pointer (no external http URLs)`,
        )
      }
    }

    if (entry.coverage === 'missing') {
      if (!entry.severity || !(RISK_LEVELS as readonly string[]).includes(entry.severity)) {
        push(errors, `${prefix}: missing requires severity`)
      }
      if (!entry.owner?.trim()) {
        push(errors, `${prefix}: missing requires owner`)
      }
      if (!entry.targetMilestone?.trim()) {
        push(errors, `${prefix}: missing requires targetMilestone`)
      }
    }

    if (entry.flags) {
      for (const dim of [
        'settingsToggle',
        'disabledState',
        'enabledJourney',
        'authorization',
        'dependency',
        'rollback',
      ] as const) {
        const v = entry.flags[dim]
        if (v !== true && v !== false && v !== 'n/a') {
          push(errors, `${prefix}: flags.${dim} must be true|false|"n/a"`)
        }
      }
    }
  }

  // FR-1 / AC-1 / AC-4: every eligible story has exactly one entry.
  for (const storyPath of eligible) {
    if (!byPath.has(storyPath)) {
      push(errors, `missing manifest entry for ${storyPath}`)
    }
  }

  // Resolve covered-by-parent references after the full index exists.
  for (const entry of manifest.entries) {
    if (entry.coverage !== 'covered-by-parent' || !entry.parentStoryId) continue
    const parent =
      byId.get(entry.parentStoryId) ||
      manifest.entries.find(
        (e) => e.id === entry.parentStoryId || e.aliases?.includes(entry.parentStoryId!),
      )
    if (!parent) {
      push(errors, `${entry.id}: parentStoryId ${entry.parentStoryId} not found in manifest`)
    }
  }

  return errors.sort((a, b) => a.localeCompare(b))
}

export type CoverageReportSummary = {
  totalStories: number
  byCoverage: Record<CoverageLevel, number>
  bySection: Record<string, Record<CoverageLevel, number>>
  byMarket: Record<string, number>
  byClient: Record<string, number>
  missing: Array<{
    id: string
    path: string
    severity: string
    owner: string
    targetMilestone: string
  }>
  flagGaps: Array<{ id: string; path: string; gaps: string[] }>
}

function emptyCoverageCounts(): Record<CoverageLevel, number> {
  return {
    journey: 0,
    smoke: 0,
    'api-contract': 0,
    'covered-by-parent': 0,
    manual: 0,
    'not-applicable': 0,
    missing: 0,
  }
}

/** Build summary counts without rewriting reviewer rationale (FR-6). */
export function summarizeCoverage(manifest: CoverageManifest): CoverageReportSummary {
  const byCoverage = emptyCoverageCounts()
  const bySection: Record<string, Record<CoverageLevel, number>> = {}
  const byMarket: Record<string, number> = {}
  const byClient: Record<string, number> = {}
  const missing: CoverageReportSummary['missing'] = []
  const flagGaps: CoverageReportSummary['flagGaps'] = []

  const sorted = [...manifest.entries].sort((a, b) => a.path.localeCompare(b.path))
  for (const entry of sorted) {
    byCoverage[entry.coverage] = (byCoverage[entry.coverage] ?? 0) + 1
    const section = sectionOfStory(entry.path)
    bySection[section] ??= emptyCoverageCounts()
    bySection[section][entry.coverage] += 1
    for (const m of entry.markets) {
      byMarket[m] = (byMarket[m] ?? 0) + 1
    }
    byClient[entry.client] = (byClient[entry.client] ?? 0) + 1

    if (entry.coverage === 'missing') {
      missing.push({
        id: entry.id,
        path: entry.path,
        severity: entry.severity ?? entry.risk,
        owner: entry.owner,
        targetMilestone: entry.targetMilestone ?? '',
      })
    }

    if (entry.flags) {
      const gaps: string[] = []
      for (const dim of [
        'settingsToggle',
        'disabledState',
        'enabledJourney',
        'authorization',
        'dependency',
        'rollback',
      ] as const) {
        if (entry.flags[dim] === false) gaps.push(dim)
      }
      if (gaps.length > 0) {
        flagGaps.push({ id: entry.id, path: entry.path, gaps })
      }
    }
  }

  return {
    totalStories: manifest.entries.length,
    byCoverage,
    bySection,
    byMarket,
    byClient,
    missing,
    flagGaps,
  }
}

/** Render a semantic Markdown report (Accessibility NFR). */
export function renderCoverageReportMarkdown(
  summary: CoverageReportSummary,
  options?: { title?: string },
): string {
  const title = options?.title ?? 'Completed feature E2E coverage report'
  const lines: string[] = [
    `# ${title}`,
    '',
    `Total stories: **${summary.totalStories}**`,
    '',
    '## Coverage levels',
    '',
    '| Level | Count |',
    '|---|---:|',
  ]
  for (const level of COVERAGE_LEVELS) {
    lines.push(`| ${level} | ${summary.byCoverage[level] ?? 0} |`)
  }

  lines.push('', '## By client', '', '| Client | Count |', '|---|---:|')
  for (const [client, count] of Object.entries(summary.byClient).sort()) {
    lines.push(`| ${client} | ${count} |`)
  }

  lines.push('', '## By market tag', '', '| Market | Count |', '|---|---:|')
  for (const [market, count] of Object.entries(summary.byMarket).sort()) {
    lines.push(`| ${market} | ${count} |`)
  }

  lines.push('', '## Missing journeys (severity / owner / milestone)', '')
  if (summary.missing.length === 0) {
    lines.push('_None._')
  } else {
    lines.push('| ID | Severity | Owner | Milestone | Path |')
    lines.push('|---|---|---|---|---|')
    for (const row of summary.missing) {
      lines.push(
        `| ${row.id} | ${row.severity} | ${row.owner} | ${row.targetMilestone} | [${row.path}](../../${row.path}) |`,
      )
    }
  }

  lines.push('', '## Flag lifecycle gaps', '')
  if (summary.flagGaps.length === 0) {
    lines.push('_No flagged entries with false dimensions._')
  } else {
    lines.push('| ID | Gaps | Path |')
    lines.push('|---|---|---|')
    for (const row of summary.flagGaps) {
      lines.push(`| ${row.id} | ${row.gaps.join(', ')} | [${row.path}](../../${row.path}) |`)
    }
  }

  lines.push('', '## Section totals', '')
  lines.push(
    '| Section | journey | smoke | api-contract | covered-by-parent | manual | not-applicable | missing |',
  )
  lines.push('|---|---:|---:|---:|---:|---:|---:|---:|')
  for (const section of Object.keys(summary.bySection).sort()) {
    const c = summary.bySection[section]
    lines.push(
      `| ${section} | ${c.journey} | ${c.smoke} | ${c['api-contract']} | ${c['covered-by-parent']} | ${c.manual} | ${c['not-applicable']} | ${c.missing} |`,
    )
  }

  lines.push('')
  return lines.join('\n')
}

/** Observability helper: diff two manifests for CI artifacts. */
export function diffCoverageManifests(
  before: CoverageManifest,
  after: CoverageManifest,
): { added: string[]; removed: string[]; levelChanges: string[] } {
  const beforeByPath = new Map(before.entries.map((e) => [e.path, e]))
  const afterByPath = new Map(after.entries.map((e) => [e.path, e]))
  const added: string[] = []
  const removed: string[] = []
  const levelChanges: string[] = []

  for (const pathKey of afterByPath.keys()) {
    if (!beforeByPath.has(pathKey)) added.push(pathKey)
  }
  for (const pathKey of beforeByPath.keys()) {
    if (!afterByPath.has(pathKey)) removed.push(pathKey)
  }
  for (const [pathKey, afterEntry] of afterByPath) {
    const prev = beforeByPath.get(pathKey)
    if (prev && prev.coverage !== afterEntry.coverage) {
      levelChanges.push(`${pathKey}: ${prev.coverage} → ${afterEntry.coverage}`)
    }
  }
  added.sort()
  removed.sort()
  levelChanges.sort()
  return { added, removed, levelChanges }
}

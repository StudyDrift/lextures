/**
 * One-shot bootstrap for E2E.4: scan docs/completed + e2e/tests and write a
 * reviewed baseline manifest. Re-run only when intentionally regenerating;
 * day-to-day edits go through the JSON manifest / check command.
 *
 * Usage: npx tsx scripts/bootstrap-completed-coverage-manifest.ts
 */
import fs from 'node:fs'
import path from 'node:path'
import {
  COVERAGE_LEVELS,
  DEFAULT_MANIFEST_REL,
  EXCLUSION_RULES,
  REPO_ROOT,
  type ClientKind,
  type CoverageEntry,
  type CoverageLevel,
  type CoverageManifest,
  type FlagCoverage,
  type Market,
  type RiskLevel,
  defaultStoryIdFromPath,
  listEligibleCompletedStories,
  sectionOfStory,
} from '../lib/completed-feature-coverage.js'

type SpecIndex = { file: string; stem: string; tokens: Set<string> }

function listSpecs(repoRoot: string): SpecIndex[] {
  const dir = path.join(repoRoot, 'e2e/tests')
  return fs
    .readdirSync(dir)
    .filter((f) => f.endsWith('.spec.ts'))
    .sort()
    .map((file) => {
      const stem = file.replace(/\.spec\.ts$/, '')
      return { file, stem, tokens: tokenSet(stem) }
    })
}

function tokenSet(s: string): Set<string> {
  return new Set(
    s
      .toLowerCase()
      .split(/[^a-z0-9]+/)
      .filter((t) => t.length > 2 && !STOP.has(t)),
  )
}

const STOP = new Set([
  'the',
  'and',
  'for',
  'with',
  'from',
  'not',
  'non',
  'md',
  'spec',
  'test',
  'plan',
  'feature',
  'features',
  'settings',
  'platform',
  'course',
  'specific',
  'implementation',
  'implemented',
  'functional',
  'enforced',
  'stubbed',
  'matrix',
  'meta',
  'api',
  'ui',
  'contract',
])

/** High-confidence curated story → specs (and optional classification overrides). */
const CURATED: Record<
  string,
  Partial<CoverageEntry> & { specs: NonNullable<CoverageEntry['specs']> }
> = {
  'docs/completed/e2e/E2E.1-course-feature-flag-matrix.md': {
    id: 'E2E.1',
    coverage: 'journey',
    risk: 'major',
    client: 'web',
    markets: ['ALL'],
    owner: 'Web Platform / QA',
    rationale: 'Course feature matrix + authz + nav + API-only group spaces.',
    specs: [
      { path: 'e2e/tests/course-features-matrix-meta.spec.ts' },
      { path: 'e2e/tests/course-features-authz.spec.ts' },
      { path: 'e2e/tests/course-features-ui-matrix-a.spec.ts' },
      { path: 'e2e/tests/course-features-ui-matrix-b.spec.ts' },
      { path: 'e2e/tests/course-features-ui-matrix-c.spec.ts' },
      { path: 'e2e/tests/course-features-nav-matrix.spec.ts' },
      { path: 'e2e/tests/course-features-api-only.spec.ts' },
    ],
    flags: flagCoverage({
      settingsToggle: true,
      disabledState: true,
      enabledJourney: true,
      authorization: true,
      dependency: 'n/a',
      rollback: true,
      notes: 'Full course matrix; dependency/rollback for families in E2E.3',
    }),
  },
  'docs/completed/e2e/E2E.2-platform-feature-flag-contract.md': {
    id: 'E2E.2',
    coverage: 'journey',
    risk: 'major',
    client: 'web',
    markets: ['ALL'],
    owner: 'Web Platform / QA',
    rationale: 'Registry-wide platform feature API/UI/authz contract.',
    specs: [
      { path: 'e2e/tests/platform-features-matrix-meta.spec.ts' },
      { path: 'e2e/tests/platform-features-authz.spec.ts' },
      { path: 'e2e/tests/platform-features-api-contract-a.spec.ts' },
      { path: 'e2e/tests/platform-features-api-contract-b.spec.ts' },
      { path: 'e2e/tests/platform-features-api-contract-c.spec.ts' },
      { path: 'e2e/tests/platform-features-ui-sample.spec.ts' },
    ],
    flags: flagCoverage({
      settingsToggle: true,
      disabledState: true,
      enabledJourney: true,
      authorization: true,
      dependency: 'n/a',
      rollback: 'n/a',
      notes: 'Contract coverage; family rollback in E2E.3',
    }),
  },
  'docs/completed/e2e/E2E.3-flagged-feature-rollback-and-dependencies.md': {
    id: 'E2E.3',
    coverage: 'journey',
    risk: 'major',
    client: 'web',
    markets: ['ALL'],
    owner: 'Web Platform / QA',
    rationale: 'Representative lifecycle + dependency truth tables for flagged families.',
    specs: [
      { path: 'e2e/tests/feature-lifecycle-meta.spec.ts' },
      { path: 'e2e/tests/feature-lifecycle-collaboration.spec.ts' },
      { path: 'e2e/tests/feature-lifecycle-credentials.spec.ts' },
      { path: 'e2e/tests/feature-lifecycle-commerce-api.spec.ts' },
      { path: 'e2e/tests/feature-lifecycle-ai.spec.ts' },
      { path: 'e2e/tests/feature-lifecycle-priority2.spec.ts' },
    ],
    flags: flagCoverage({
      settingsToggle: true,
      disabledState: true,
      enabledJourney: true,
      authorization: true,
      dependency: true,
      rollback: true,
    }),
  },
  'docs/completed/e2e/E2E.4-completed-feature-traceability.md': {
    id: 'E2E.4',
    coverage: 'api-contract',
    risk: 'major',
    client: 'ops',
    markets: ['ALL'],
    owner: 'QA / Developer Experience',
    rationale: 'Coverage gate validated by meta + npm coverage check/self-test (no product UI journey).',
    specs: [{ path: 'e2e/tests/completed-feature-coverage-meta.spec.ts' }],
  },
}

function flagCoverage(partial: FlagCoverage): FlagCoverage {
  return partial
}

type SectionPolicy = {
  client: ClientKind
  owner: string
  defaultCoverage?: CoverageLevel
  defaultRationale?: string
  markets?: Market[]
  risk?: RiskLevel
}

const SECTION_POLICY: Record<string, SectionPolicy> = {
  mobile: {
    client: 'mobile',
    owner: 'Mobile',
    defaultCoverage: 'not-applicable',
    defaultRationale: 'Mobile client story; web Playwright suite is not the coverage surface.',
    markets: ['ALL'],
    risk: 'minor',
  },
  '07-mobile-offline-cross-platform': {
    client: 'mobile',
    owner: 'Mobile',
    defaultCoverage: 'not-applicable',
    defaultRationale: 'Mobile/offline platform story; covered outside web E2E.',
    markets: ['ALL'],
    risk: 'minor',
  },
  '21-cli': {
    client: 'cli',
    owner: 'CLI',
    defaultCoverage: 'not-applicable',
    defaultRationale: 'CLI surface; not a web Playwright journey.',
    markets: ['ALL'],
    risk: 'minor',
  },
  cli: {
    client: 'cli',
    owner: 'CLI',
    defaultCoverage: 'not-applicable',
    defaultRationale: 'CLI surface; not a web Playwright journey.',
    markets: ['ALL'],
    risk: 'minor',
  },
  lighthouse: {
    client: 'web',
    owner: 'Web Platform / Perf',
    defaultCoverage: 'smoke',
    defaultRationale: 'Lighthouse harness coverage (no full product journey).',
    markets: ['ALL'],
    risk: 'minor',
  },
  '20-docs-trust': {
    client: 'docs',
    owner: 'Trust / Legal',
    defaultCoverage: 'manual',
    defaultRationale: 'Trust/legal documentation; controlled manual evidence.',
    markets: ['ALL'],
    risk: 'major',
  },
  '10-compliance-privacy-security': {
    client: 'web',
    owner: 'Compliance / Security',
    markets: ['ALL'],
    risk: 'critical',
  },
  '17-platform-performance-operability': {
    client: 'ops',
    owner: 'Platform Ops',
    defaultCoverage: 'not-applicable',
    defaultRationale: 'Operational/performance story; not a user-journey E2E responsibility.',
    markets: ['ALL'],
    risk: 'minor',
  },
  emails: {
    client: 'ops',
    owner: 'Platform',
    defaultCoverage: 'not-applicable',
    defaultRationale: 'Email template/content story; validated outside browser E2E.',
    markets: ['ALL'],
    risk: 'minor',
  },
  animations: {
    client: 'web',
    owner: 'Web Platform',
    markets: ['ALL'],
    risk: 'minor',
  },
  assets: {
    client: 'docs',
    owner: 'Docs',
    defaultCoverage: 'not-applicable',
    defaultRationale: 'Asset bundle; excluded from journey expectations.',
    markets: ['ALL'],
    risk: 'none',
  },
  // Homeschool rebrand series (HS.1–HS.6): copy/docs/metadata; not web Playwright journeys.
  homeschool: {
    client: 'docs',
    owner: 'Platform + Compliance',
    defaultCoverage: 'not-applicable',
    defaultRationale:
      'Segment rebrand (terminology, marketing, client labels, server copy, docs/ISMS/e2e metadata); not a user-journey E2E target.',
    markets: ['HS'],
    risk: 'minor',
  },
  e2e: {
    client: 'web',
    owner: 'QA / Developer Experience',
    markets: ['ALL'],
    risk: 'major',
  },
  '13-k12-specific': {
    client: 'web',
    owner: 'K12 Product',
    markets: ['K12'],
    risk: 'major',
  },
  '14-higher-ed-specific': {
    client: 'web',
    owner: 'HE Product',
    markets: ['HE'],
    risk: 'major',
  },
  // Map key matches the frozen on-disk folder docs/completed/15-self-learner-specific/ (HS.6).
  '15-self-learner-specific': {
    client: 'web',
    owner: 'Homeschool Product',
    markets: ['HS'],
    risk: 'major',
  },
}

const DEFAULT_OWNER_BY_PREFIX: Array<{ prefix: string; owner: string }> = [
  { prefix: 'docs/completed/01-', owner: 'Learning Platform' },
  { prefix: 'docs/completed/02-', owner: 'Assessment' },
  { prefix: 'docs/completed/03-', owner: 'Grading' },
  { prefix: 'docs/completed/04-', owner: 'Identity' },
  { prefix: 'docs/completed/05-', owner: 'Tenancy / RBAC' },
  { prefix: 'docs/completed/06-', owner: 'Collaboration' },
  { prefix: 'docs/completed/08-', owner: 'Content / Media' },
  { prefix: 'docs/completed/09-', owner: 'Analytics' },
  { prefix: 'docs/completed/11-', owner: 'i18n' },
  { prefix: 'docs/completed/12-', owner: 'Accessibility' },
  { prefix: 'docs/completed/16-', owner: 'Integrations' },
  { prefix: 'docs/completed/18-', owner: 'Admin Experience' },
  { prefix: 'docs/completed/19-', owner: 'AI Platform' },
  { prefix: 'docs/completed/visual-collaboration/', owner: 'Visual Collaboration' },
  { prefix: 'docs/completed/interactive-quizzes/', owner: 'Interactive Quizzes' },
  { prefix: 'docs/completed/transcripts/', owner: 'Transcripts' },
  { prefix: 'docs/completed/marketplace', owner: 'Marketplace' },
  { prefix: 'docs/completed/ai-providers/', owner: 'AI Providers' },
  { prefix: 'docs/completed/learner-profile/', owner: 'Learner Profile' },
  { prefix: 'docs/completed/intro-course/', owner: 'Intro Course' },
  { prefix: 'docs/completed/web/', owner: 'Web Platform' },
  { prefix: 'docs/completed/badges/', owner: 'Gamification' },
  { prefix: 'docs/completed/feedback/', owner: 'Feedback' },
  { prefix: 'docs/completed/agent-grader/', owner: 'Grading Agent' },
  { prefix: 'docs/completed/grading-agent/', owner: 'Grading Agent' },
]

/** Filename keyword → preferred spec stem(s). */
const KEYWORD_SPEC: Array<{ keywords: string[]; specs: string[]; level?: CoverageLevel }> = [
  { keywords: ['auth', 'login', 'password', 'signup'], specs: ['auth'] },
  { keywords: ['ferpa'], specs: ['ferpa'], level: 'journey' },
  { keywords: ['coppa'], specs: ['coppa'], level: 'journey' },
  { keywords: ['ccpa'], specs: ['ccpa'], level: 'journey' },
  { keywords: ['wcag'], specs: ['wcag'], level: 'journey' },
  { keywords: ['vpat'], specs: ['vpat'], level: 'manual' },
  { keywords: ['parent', 'portal', 'guardian'], specs: ['parent-portal'] },
  { keywords: ['discussion', 'forum'], specs: ['discussions'] },
  { keywords: ['attendance'], specs: ['attendance', 'course-attendance'] },
  { keywords: ['gradebook'], specs: ['gradebook'] },
  { keywords: ['calendar'], specs: ['calendar', 'academic-calendar', 'calendar-feeds'] },
  { keywords: ['billing', 'payment', 'stripe'], specs: ['billing'] },
  { keywords: ['scorm'], specs: ['scorm'] },
  { keywords: ['xapi', 'cmi5'], specs: ['xapi-emission'] },
  { keywords: ['webhook'], specs: ['webhooks'] },
  { keywords: ['zapier'], specs: ['zapier-connector'] },
  { keywords: ['sis'], specs: ['sis-integration'] },
  { keywords: ['lti'], specs: ['integrations'] },
  { keywords: ['rubric'], specs: ['gradebook'] },
  { keywords: ['plagiarism', 'originality'], specs: ['plagiarism-workflow'] },
  { keywords: ['peer', 'review'], specs: ['peer-review'] },
  { keywords: ['office', 'hours'], specs: ['office-hours'] },
  { keywords: ['inbox', 'messaging'], specs: ['inbox', 'multilingual-messaging'] },
  { keywords: ['broadcast'], specs: ['broadcasts'] },
  { keywords: ['notification', 'push'], specs: ['push-notifications'] },
  { keywords: ['i18n', 'locale', 'translation', 'rtl'], specs: ['i18n', 'rtl-locale', 'locale-format', 'translation-memory'] },
  { keywords: ['caption', 'transcript'], specs: ['captions', 'video-captions-accessibility', 'transcript-fees'] },
  { keywords: ['ai', 'tutor'], specs: ['ai-tutor', 'study-buddy', 'lesson-generator'] },
  { keywords: ['marketplace'], specs: ['course-marketplace-listing', 'course-marketplace-purchase', 'course-marketplace-storefront'] },
  { keywords: ['public', 'api'], specs: ['public-api'] },
  { keywords: ['rbac', 'permission', 'role'], specs: ['rbac-permission-first'] },
  { keywords: ['organization', 'tenant'], specs: ['organizations'] },
  { keywords: ['enrollment'], specs: ['enrollments', 'enrollment-lifecycle', 'self-paced-enrollment'] },
  { keywords: ['module'], specs: ['modules'] },
  { keywords: ['quiz', 'assessment'], specs: ['misc'] },
  { keywords: ['conditional', 'release'], specs: ['conditional-release'] },
  { keywords: ['differentiated'], specs: ['differentiated-assignments'] },
  { keywords: ['collab', 'collaborative'], specs: ['collab-docs'] },
  { keywords: ['virtual', 'classroom', 'live'], specs: ['virtual-classroom'] },
  { keywords: ['group', 'space'], specs: ['group-spaces'] },
  { keywords: ['report', 'card'], specs: ['report-cards'] },
  { keywords: ['board', 'visual', 'collaboration'], specs: ['feature-lifecycle-collaboration'] },
  { keywords: ['interactive', 'quiz'], specs: ['feature-lifecycle-collaboration'] },
  { keywords: ['credential', 'diploma', 'certificate'], specs: ['credentials', 'feature-lifecycle-credentials'] },
  { keywords: ['advising'], specs: ['advising'] },
  { keywords: ['library', 'oer', 'ereserve'], specs: ['library', 'oer-library', 'mobile-library-ereserves'] },
  { keywords: ['accessibility', 'a11y', 'screen', 'reader'], specs: ['screen-reader-a11y', 'accessibility-intake', 'keyboard-navigation', 'alt-text-enforcement'] },
  { keywords: ['dyslexia', 'reading'], specs: ['dyslexia-reading-preferences', 'read-aloud', 'reading-level'] },
  { keywords: ['high', 'contrast', 'motion'], specs: ['high-contrast-reduced-motion', 'contrast'] },
  { keywords: ['lighthouse'], specs: ['lighthouse-harness', 'lighthouse-dashboard-a11y'] },
  { keywords: ['feature', 'help', 'onboarding'], specs: ['feature-help'] },
  { keywords: ['admin', 'console'], specs: ['admin-console', 'admin-search', 'admin-audit-log'] },
  { keywords: ['backup', 'restore'], specs: ['backup-restore'] },
  { keywords: ['rate', 'limit'], specs: ['rate-limiting'] },
  { keywords: ['cache', 'redis'], specs: ['caching-layer'] },
  { keywords: ['observability', 'health'], specs: ['observability', 'health'] },
  { keywords: ['status', 'page'], specs: ['status-page'] },
  { keywords: ['trust', 'center'], specs: ['trust-center'] },
  { keywords: ['legal'], specs: ['legal-pages'] },
  { keywords: ['impersonation'], specs: ['impersonation'] },
  { keywords: ['seat'], specs: ['seat-management'] },
  { keywords: ['revenue'], specs: ['revenue-share'] },
  { keywords: ['ceu'], specs: ['ceu-tracking'] },
  { keywords: ['ccr'], specs: ['ccr'] },
  { keywords: ['sbg', 'standards'], specs: ['sbg'] },
  { keywords: ['mastery'], specs: ['mastery-heatmap'] },
  { keywords: ['behavior'], specs: ['behavior'] },
  { keywords: ['at-risk', 'atrisk'], specs: ['at-risk'] },
  { keywords: ['accommodation'], specs: ['accommodations-engine'] },
  { keywords: ['grader', 'agent'], specs: ['grader-agent'] },
  { keywords: ['h5p'], specs: ['h5p'] },
  { keywords: ['equation'], specs: ['equation-editor'] },
  { keywords: ['notebook', 'flashcard'], specs: ['notebook-flashcards'] },
  { keywords: ['feed'], specs: ['feed'] },
  { keywords: ['syllabus'], specs: ['syllabus'] },
  { keywords: ['blueprint'], specs: ['blueprint-settings'] },
  { keywords: ['section'], specs: ['sections-settings'] },
  { keywords: ['outcome'], specs: ['outcomes-settings', 'outcomes-report'] },
  { keywords: ['what-if', 'whatif'], specs: ['what-if-grades'] },
  { keywords: ['final', 'grade'], specs: ['final-grade-submission'] },
  { keywords: ['incomplete'], specs: ['incomplete-grade-workflow'] },
  { keywords: ['custom', 'field'], specs: ['custom-fields'] },
  { keywords: ['bulk', 'import'], specs: ['bulk-user-import'] },
  { keywords: ['access', 'key'], specs: ['access-keys'] },
  { keywords: ['bot'], specs: ['bots'] },
  { keywords: ['bookstore', 'textbook'], specs: ['bookstore-textbook'] },
  { keywords: ['consortium'], specs: ['consortium-sharing'] },
  { keywords: ['age', 'appropriate'], specs: ['age-appropriate-ui-mode'] },
  { keywords: ['iso'], specs: ['iso-compliance'] },
  { keywords: ['state', 'compliance'], specs: ['state-compliance'] },
  { keywords: ['research', 'consent'], specs: ['research-consent'] },
  { keywords: ['security', 'disclosure'], specs: ['security-disclosure'] },
  { keywords: ['av', 'scan', 'malware'], specs: ['av-scan'] },
  { keywords: ['tus', 'upload'], specs: ['tus-uploads'] },
  { keywords: ['transcode'], specs: ['transcode'] },
  { keywords: ['file', 'preview'], specs: ['file-preview'] },
  { keywords: ['storage', 'quota'], specs: ['storage-quotas'] },
  { keywords: ['scheduler'], specs: ['scheduler'] },
  { keywords: ['maintenance'], specs: ['maintenance-banner'] },
  { keywords: ['archived'], specs: ['archived-settings'] },
  { keywords: ['import', 'export'], specs: ['import-export-settings'] },
  { keywords: ['ai', 'provider'], specs: ['ai-providers-settings'] },
  { keywords: ['ai', 'disclosure'], specs: ['ai-disclosure'] },
  { keywords: ['help', 'widget'], specs: ['help-widget'] },
  { keywords: ['dashboard'], specs: ['dashboard'] },
  { keywords: ['navigation'], specs: ['navigation'] },
  { keywords: ['timezone'], specs: ['timezone'] },
  { keywords: ['speech'], specs: ['speech-to-text'] },
  { keywords: ['item', 'analysis'], specs: ['item-analysis'] },
  { keywords: ['learning', 'path'], specs: ['learning-paths'] },
  { keywords: ['self', 'reflection'], specs: ['self-reflection'] },
  { keywords: ['study', 'reminder'], specs: ['study-reminders'] },
  { keywords: ['student', 'progress'], specs: ['student-progress'] },
  { keywords: ['instructor', 'insight'], specs: ['instructor-insights', 'mobile-instructor-insights'] },
  { keywords: ['engagement'], specs: ['engagement'] },
  { keywords: ['classroom', 'signal'], specs: ['classroom-signals'] },
  { keywords: ['course', 'review'], specs: ['course-reviews'] },
  { keywords: ['course', 'catalog'], specs: ['course-catalog', 'public-course-catalog'] },
  { keywords: ['report', 'export'], specs: ['report-export'] },
  { keywords: ['grade', 'level'], specs: ['grade-level'] },
  { keywords: ['external', 'link'], specs: ['external-links'] },
]

function ownerFor(storyPath: string, section: string): string {
  const policy = SECTION_POLICY[section]
  if (policy?.owner) return policy.owner
  for (const row of DEFAULT_OWNER_BY_PREFIX) {
    if (storyPath.startsWith(row.prefix)) return row.owner
  }
  return 'Product / QA'
}

function isGapNote(storyPath: string): boolean {
  const base = path.basename(storyPath).toLowerCase()
  return (
    base.includes('not-implemented') ||
    base.includes('non-functional') ||
    base.includes('not-enforced') ||
    base.includes('stubbed') ||
    base.includes('gap')
  )
}

function isMobileStory(storyPath: string, section: string): boolean {
  if (section === 'mobile' || section === '07-mobile-offline-cross-platform') return true
  if (/(^|\/)VC\.M\d/i.test(path.basename(storyPath))) return true
  return false
}

function scoreStoryToSpec(storyPath: string, spec: SpecIndex): number {
  const base = path.basename(storyPath, '.md').toLowerCase()
  const storyTokens = tokenSet(base)
  let hits = 0
  for (const t of storyTokens) {
    if (spec.tokens.has(t)) hits++
  }
  let score = hits * 12
  // Substring bonuses on significant chunks.
  const compactStory = base.replace(/^[a-z]*\d+(\.\d+)*-?/i, '').replace(/[^a-z0-9]+/g, '')
  const compactSpec = spec.stem.replace(/[^a-z0-9]+/g, '')
  if (compactStory.length >= 6 && compactSpec.includes(compactStory)) score += 40
  if (compactSpec.length >= 6 && compactStory.includes(compactSpec)) score += 35
  if (spec.stem === compactStory || normalize(base) === normalize(spec.stem)) score += 50
  return score
}

function normalize(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, '-')
}

function keywordMatches(
  storyPath: string,
): { specs: string[]; level?: CoverageLevel; hits: number } | null {
  const base = path.basename(storyPath, '.md').toLowerCase()
  const tokens = [...tokenSet(base)]
  let best: { specs: string[]; level?: CoverageLevel; hits: number } | null = null
  for (const row of KEYWORD_SPEC) {
    const hits = row.keywords.filter((k) => tokens.includes(k) || base.includes(k)).length
    if (hits === 0) continue
    if (!best || hits > best.hits || (hits === best.hits && row.keywords.length < best.specs.length)) {
      best = { specs: row.specs, level: row.level, hits }
    }
  }
  return best
}

function existingSpecPaths(stems: string[], index: SpecIndex[]): CoverageEntry['specs'] {
  const byStem = new Map(index.map((s) => [s.stem, s.file]))
  const out: NonNullable<CoverageEntry['specs']> = []
  for (const stem of stems) {
    const file = byStem.get(stem)
    if (file) out.push({ path: `e2e/tests/${file}` })
  }
  return out
}

function buildEntry(storyPath: string, specs: SpecIndex[]): CoverageEntry {
  const curated = CURATED[storyPath]
  if (curated) {
    return {
      id: curated.id ?? defaultStoryIdFromPath(storyPath),
      path: storyPath,
      markets: curated.markets ?? ['ALL'],
      risk: curated.risk ?? 'major',
      client: curated.client ?? 'web',
      coverage: curated.coverage ?? 'journey',
      rationale: curated.rationale ?? 'Curated mapping.',
      owner: curated.owner ?? 'QA',
      specs: curated.specs,
      flags: curated.flags,
      reviewed: true,
    }
  }

  const section = sectionOfStory(storyPath)
  const policy = SECTION_POLICY[section]
  const id = defaultStoryIdFromPath(storyPath)
  const owner = ownerFor(storyPath, section)
  const markets = policy?.markets ?? ['ALL']
  const risk = policy?.risk ?? 'major'
  const client = isMobileStory(storyPath, section)
    ? 'mobile'
    : (policy?.client ?? 'web')

  if (isGapNote(storyPath)) {
    return {
      id,
      path: storyPath,
      markets,
      risk: 'none',
      client: 'docs',
      coverage: 'not-applicable',
      rationale: 'Gap/limitation note documenting non-implemented behavior; not an E2E journey target.',
      owner,
      reviewed: true,
    }
  }

  if (policy?.defaultCoverage === 'not-applicable' || client === 'mobile' || client === 'cli') {
    return {
      id,
      path: storyPath,
      markets,
      risk: risk === 'critical' ? 'major' : risk,
      client,
      coverage: 'not-applicable',
      rationale:
        policy?.defaultRationale ??
        (client === 'mobile'
          ? 'Mobile client story; web Playwright is not the coverage surface.'
          : 'Out of scope for web Playwright journeys.'),
      owner,
      reviewed: true,
    }
  }

  if (policy?.defaultCoverage === 'manual') {
    return {
      id,
      path: storyPath,
      markets,
      risk,
      client: policy.client,
      coverage: 'manual',
      rationale: policy.defaultRationale ?? 'Manual / controlled evidence.',
      owner,
      manualEvidence: {
        owner,
        cadence: 'quarterly',
        location: `internal://docs/completed/${id}`,
      },
      reviewed: true,
    }
  }

  // Lighthouse section default smoke.
  if (policy?.defaultCoverage === 'smoke' && section === 'lighthouse') {
    const lhSpecs = existingSpecPaths(['lighthouse-harness', 'lighthouse-dashboard-a11y'], specs)
    return {
      id,
      path: storyPath,
      markets,
      risk,
      client: 'web',
      coverage: 'smoke',
      rationale: policy.defaultRationale ?? 'Lighthouse harness smoke.',
      owner,
      specs: lhSpecs.length
        ? lhSpecs
        : [{ path: 'e2e/tests/lighthouse-harness.spec.ts' }],
      reviewed: true,
    }
  }

  // Keyword + fuzzy match to specs.
  const kw = keywordMatches(storyPath)
  let linked = kw ? existingSpecPaths(kw.specs, specs) : []
  if (!linked || linked.length === 0) {
    const ranked = specs
      .map((s) => ({ s, score: scoreStoryToSpec(storyPath, s) }))
      .filter((r) => r.score >= 36)
      .sort((a, b) => b.score - a.score)
    linked = ranked.slice(0, 3).map((r) => ({ path: `e2e/tests/${r.s.file}` }))
  }

  if (linked && linked.length > 0) {
    const firstSpec = specs.find((s) => `e2e/tests/${s.file}` === linked[0]!.path)
    const strong =
      (kw?.level === 'journey') ||
      (kw?.hits ?? 0) >= 2 ||
      (firstSpec != null && scoreStoryToSpec(storyPath, firstSpec) >= 50)
    const coverage: CoverageLevel = strong ? 'journey' : 'smoke'

    const entry: CoverageEntry = {
      id,
      path: storyPath,
      markets,
      risk,
      client: 'web',
      coverage,
      rationale: strong
        ? 'Automated Playwright coverage linked by reviewed filename/keyword mapping.'
        : 'Partial/smoke Playwright coverage linked by reviewed mapping; deepen to journey when risk warrants.',
      owner,
      specs: linked,
      reviewed: true,
    }

    // Mark likely flagged platform/course stories with explicit flag dimensions.
    if (/feature.?flag|platform.?feature|kill.?switch|ff[A-Z]/i.test(path.basename(storyPath))) {
      entry.flags = flagCoverage({
        settingsToggle: true,
        disabledState: false,
        enabledJourney: true,
        authorization: false,
        dependency: false,
        rollback: false,
        notes: 'Flag lifecycle dimensions incomplete; see E2E.2/E2E.3 for registry/family coverage.',
      })
    }
    return entry
  }

  // VC / IQ / transcripts parents: leaf stories often covered by parent family.
  if (
    section === 'visual-collaboration' ||
    section === 'interactive-quizzes' ||
    section === 'transcripts' ||
    section === 'marketplace' ||
    section === 'marketplace-courses'
  ) {
    const parentBySection: Record<string, string> = {
      'visual-collaboration': 'VC.1',
      'interactive-quizzes': 'IQ.1',
      transcripts: 'T01',
      marketplace: 'MKT1',
      'marketplace-courses': 'MC0',
    }
    const parent = parentBySection[section]
    // Foundation docs are journeys when we have lifecycle specs.
    if (
      /foundation|1-foundation|MKT1|MC0|IQ\.1|VC\.1|T01/i.test(path.basename(storyPath)) ||
      id === parent
    ) {
      const familySpecs =
        section === 'visual-collaboration' || section === 'interactive-quizzes'
          ? existingSpecPaths(['feature-lifecycle-collaboration'], specs)
          : section === 'transcripts'
            ? existingSpecPaths(['feature-lifecycle-credentials', 'credentials', 'transcript-fees'], specs)
            : existingSpecPaths(
                [
                  'course-marketplace-listing',
                  'course-marketplace-purchase',
                  'course-marketplace-storefront',
                  'feature-lifecycle-commerce-api',
                ],
                specs,
              )
      if (familySpecs?.length) {
        return {
          id,
          path: storyPath,
          markets,
          risk,
          client: 'web',
          coverage: 'journey',
          rationale: 'Foundation story covered by family lifecycle / marketplace journeys.',
          owner,
          specs: familySpecs,
          flags: flagCoverage({
            settingsToggle: true,
            disabledState: true,
            enabledJourney: true,
            authorization: true,
            dependency: true,
            rollback: true,
            lifecycleFamilyId:
              section === 'transcripts'
                ? 'transcripts'
                : section === 'interactive-quizzes'
                  ? 'interactive-quizzes'
                  : section === 'visual-collaboration'
                    ? 'visual-boards'
                    : 'payments',
          }),
          reviewed: true,
        }
      }
    }

    if (parent && id !== parent) {
      return {
        id,
        path: storyPath,
        markets,
        risk,
        client: client === 'mobile' ? 'mobile' : 'web',
        coverage: client === 'mobile' ? 'not-applicable' : 'covered-by-parent',
        rationale:
          client === 'mobile'
            ? 'Mobile variant; web Playwright is not the coverage surface.'
            : `Covered by parent family story ${parent} and its linked lifecycle/happy-path specs.`,
        owner,
        parentStoryId: client === 'mobile' ? undefined : parent,
        reviewed: true,
      }
    }
  }

  // Compliance without a linked spec → manual.
  if (section === '10-compliance-privacy-security' || section === '20-docs-trust') {
    return {
      id,
      path: storyPath,
      markets,
      risk,
      client: section === '20-docs-trust' ? 'docs' : 'web',
      coverage: 'manual',
      rationale: 'Compliance/trust story; controlled manual evidence with owner and cadence.',
      owner,
      manualEvidence: {
        owner,
        cadence: 'quarterly',
        location: `internal://compliance/${id}`,
      },
      reviewed: true,
    }
  }

  // Default: owned missing disposition (AC-5) so CI integrity can pass.
  return {
    id,
    path: storyPath,
    markets,
    risk,
    client: 'web',
    coverage: 'missing',
    rationale: 'No durable Playwright journey linked yet; tracked as an owned coverage gap.',
    owner,
    severity: risk === 'none' ? 'minor' : risk,
    targetMilestone: 'E2E-coverage-backlog',
    reviewed: true,
  }
}

function main(): void {
  const stories = listEligibleCompletedStories(REPO_ROOT)
  const specs = listSpecs(REPO_ROOT)
  const entries = stories.map((p) => buildEntry(p, specs))

  // Ensure parent IDs referenced by covered-by-parent exist; if parent missing, demote.
  const byId = new Map(entries.map((e) => [e.id, e]))
  for (const entry of entries) {
    if (entry.coverage === 'covered-by-parent' && entry.parentStoryId && !byId.has(entry.parentStoryId)) {
      const wanted = entry.parentStoryId
      const alt = entries.find(
        (e) =>
          e.id === wanted ||
          e.path.includes(`/${wanted}-`) ||
          e.path.endsWith(`/${wanted}.md`),
      )
      if (alt) {
        entry.parentStoryId = alt.id
        byId.set(alt.id, alt)
      } else {
        entry.coverage = 'missing'
        entry.parentStoryId = undefined
        entry.severity = entry.risk === 'none' ? 'minor' : entry.risk
        entry.targetMilestone = 'E2E-coverage-backlog'
        entry.rationale = `Parent ${wanted} not found; tracked as owned gap.`
      }
    }
  }

  // Dedupe IDs by appending path hash suffix when collisions occur across paths.
  const seen = new Map<string, string>()
  for (const entry of entries) {
    const prior = seen.get(entry.id)
    if (prior && prior !== entry.path) {
      const suffix = sectionOfStory(entry.path).replace(/[^a-z0-9]+/gi, '-').slice(0, 24)
      entry.aliases = [entry.id]
      entry.id = `${entry.id}@${suffix || 'dup'}`
    }
    seen.set(entry.id, entry.path)
  }

  const manifest: CoverageManifest = {
    version: 1,
    generatedNote:
      'Reviewed baseline bootstrap for E2E.4. Classifications and rationales are human-maintainable; regenerate only with intent.',
    exclusions: EXCLUSION_RULES.map((r) => `${r.id}: ${r.description}`),
    entries: entries.sort((a, b) => a.path.localeCompare(b.path)),
  }

  // Sanity: only known coverage levels.
  for (const e of manifest.entries) {
    if (!(COVERAGE_LEVELS as readonly string[]).includes(e.coverage)) {
      throw new Error(`invalid coverage for ${e.path}`)
    }
  }

  const outPath = path.join(REPO_ROOT, DEFAULT_MANIFEST_REL)
  fs.mkdirSync(path.dirname(outPath), { recursive: true })
  fs.writeFileSync(outPath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8')

  const counts = Object.fromEntries(COVERAGE_LEVELS.map((l) => [l, 0])) as Record<CoverageLevel, number>
  for (const e of manifest.entries) counts[e.coverage]++
  console.log(`Wrote ${manifest.entries.length} entries → ${DEFAULT_MANIFEST_REL}`)
  console.log(counts)
}

main()

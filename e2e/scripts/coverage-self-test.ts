/**
 * E2E.4 unit/integration self-test (no Playwright, no API stack).
 * Usage: npm run e2e:coverage:test
 */
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import {
  COVERAGE_LEVELS,
  REPO_ROOT,
  type CoverageEntry,
  type CoverageManifest,
  defaultStoryIdFromPath,
  diffCoverageManifests,
  isExcludedCompletedPath,
  listEligibleCompletedStories,
  loadManifest,
  renderCoverageReportMarkdown,
  resolveManifestPath,
  summarizeCoverage,
  validateCompletedFeatureCoverage,
} from '../lib/completed-feature-coverage.js'

function withFixtureTree(files: Record<string, string>, fn: (root: string) => void): void {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'e2e4-self-'))
  try {
    for (const [rel, body] of Object.entries(files)) {
      const abs = path.join(tmp, rel)
      fs.mkdirSync(path.dirname(abs), { recursive: true })
      fs.writeFileSync(abs, body)
    }
    fn(tmp)
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true })
  }
}

function test(name: string, fn: () => void): void {
  fn()
  console.log(`  ok  ${name}`)
}

console.log('e2e:coverage:test')

test('exclusions skip README and assets', () => {
  assert.equal(isExcludedCompletedPath('docs/completed/transcripts/README.md'), true)
  assert.equal(isExcludedCompletedPath('docs/completed/assets/x.png'), true)
  assert.equal(isExcludedCompletedPath('docs/completed/e2e/E2E.1-x.md'), false)
})

test('stable ids from filenames', () => {
  assert.equal(defaultStoryIdFromPath('docs/completed/e2e/E2E.1-course.md'), 'E2E.1')
  assert.equal(defaultStoryIdFromPath('docs/completed/mobile/M9.1-catalog.md'), 'M9.1')
  assert.equal(defaultStoryIdFromPath('docs/completed/visual-collaboration/VC.M1-x.md'), 'VC.M1')
})

test('AC-2 broken spec link names story id', () => {
  withFixtureTree(
    {
      'docs/completed/sample/S1-demo.md': '# S1\n',
    },
    (root) => {
      const entry: CoverageEntry = {
        id: 'S1',
        path: 'docs/completed/sample/S1-demo.md',
        markets: ['ALL'],
        risk: 'minor',
        client: 'web',
        coverage: 'journey',
        rationale: 'fixture',
        owner: 'QA',
        reviewed: true,
        specs: [{ path: 'e2e/tests/missing-file.spec.ts' }],
      }
      const errors = validateCompletedFeatureCoverage({
        repoRoot: root,
        manifest: { version: 1, generatedNote: 'f', exclusions: [], entries: [entry] },
      })
      assert.ok(errors.some((e) => e.includes('S1') && e.includes('missing-file.spec.ts')))
    },
  )
})

test('AC-4 new story without disposition fails', () => {
  withFixtureTree({ 'docs/completed/sample/S2-new.md': '# S2\n' }, (root) => {
    const errors = validateCompletedFeatureCoverage({
      repoRoot: root,
      manifest: { version: 1, generatedNote: 'f', exclusions: [], entries: [] },
    })
    assert.ok(errors.some((e) => e.includes('missing manifest entry') && e.includes('S2-new.md')))
  })
})

test('AC-5 missing requires severity owner milestone', () => {
  withFixtureTree({ 'docs/completed/sample/S3-gap.md': '# S3\n' }, (root) => {
    const entry: CoverageEntry = {
      id: 'S3',
      path: 'docs/completed/sample/S3-gap.md',
      markets: ['ALL'],
      risk: 'major',
      client: 'web',
      coverage: 'missing',
      rationale: 'gap',
      owner: '',
      reviewed: true,
    }
    const errors = validateCompletedFeatureCoverage({
      repoRoot: root,
      manifest: { version: 1, generatedNote: 'f', exclusions: [], entries: [entry] },
    })
    assert.ok(errors.some((e) => e.includes('severity')))
    assert.ok(errors.some((e) => e.includes('owner')))
    assert.ok(errors.some((e) => e.includes('targetMilestone')))
  })
})

test('unknown classification rejected', () => {
  withFixtureTree({ 'docs/completed/sample/S4.md': '# S4\n' }, (root) => {
    const entry = {
      id: 'S4',
      path: 'docs/completed/sample/S4.md',
      markets: ['ALL'],
      risk: 'minor',
      client: 'web',
      coverage: 'full-send',
      rationale: 'bad',
      owner: 'QA',
      reviewed: true,
    } as unknown as CoverageEntry
    const errors = validateCompletedFeatureCoverage({
      repoRoot: root,
      manifest: { version: 1, generatedNote: 'f', exclusions: [], entries: [entry] },
    })
    assert.ok(errors.some((e) => e.includes('unknown classification')))
  })
})

test('manual rejects external evidence URLs', () => {
  withFixtureTree({ 'docs/completed/sample/S5.md': '# S5\n' }, (root) => {
    const entry: CoverageEntry = {
      id: 'S5',
      path: 'docs/completed/sample/S5.md',
      markets: ['ALL'],
      risk: 'major',
      client: 'docs',
      coverage: 'manual',
      rationale: 'manual',
      owner: 'Compliance',
      reviewed: true,
      manualEvidence: {
        owner: 'Compliance',
        cadence: 'quarterly',
        location: 'https://example.com/secret-evidence',
      },
    }
    const errors = validateCompletedFeatureCoverage({
      repoRoot: root,
      manifest: { version: 1, generatedNote: 'f', exclusions: [], entries: [entry] },
    })
    assert.ok(errors.some((e) => e.includes('internal pointer')))
  })
})

test('fixture tree add/move/delete integrity', () => {
  withFixtureTree(
    {
      'docs/completed/a/A1.md': '# A1\n',
      'docs/completed/a/A2.md': '# A2\n',
      'e2e/tests/a1.spec.ts': 'test()',
    },
    (root) => {
      const mk = (partial: Partial<CoverageEntry> & Pick<CoverageEntry, 'id' | 'path' | 'coverage'>): CoverageEntry => ({
        markets: ['ALL'],
        risk: 'minor',
        client: 'web',
        rationale: 'f',
        owner: 'QA',
        reviewed: true,
        ...partial,
      })
      const good: CoverageManifest = {
        version: 1,
        generatedNote: 'f',
        exclusions: [],
        entries: [
          mk({
            id: 'A1',
            path: 'docs/completed/a/A1.md',
            coverage: 'journey',
            specs: [{ path: 'e2e/tests/a1.spec.ts' }],
          }),
          mk({
            id: 'A2',
            path: 'docs/completed/a/A2.md',
            coverage: 'missing',
            severity: 'minor',
            targetMilestone: 'm1',
          }),
        ],
      }
      assert.deepEqual(validateCompletedFeatureCoverage({ repoRoot: root, manifest: good }), [])

      // delete story file but keep entry → broken link
      fs.unlinkSync(path.join(root, 'docs/completed/a/A2.md'))
      const afterDelete = validateCompletedFeatureCoverage({ repoRoot: root, manifest: good })
      assert.ok(afterDelete.some((e) => e.includes('broken document link') || e.includes('missing manifest')))

      // move: rename file without updating manifest path
      fs.writeFileSync(path.join(root, 'docs/completed/a/A2-moved.md'), '# A2\n')
      const movedErrors = validateCompletedFeatureCoverage({ repoRoot: root, manifest: good })
      assert.ok(movedErrors.some((e) => e.includes('A2-moved.md') || e.includes('broken document link')))
    },
  )
})

test('report headings/tables are semantic', () => {
  const manifest = loadManifest(resolveManifestPath(REPO_ROOT))
  const md = renderCoverageReportMarkdown(summarizeCoverage(manifest))
  assert.ok(md.startsWith('# '))
  assert.ok(md.includes('## Coverage levels'))
  assert.ok(md.includes('| Level | Count |'))
  for (const level of COVERAGE_LEVELS) assert.ok(md.includes(`| ${level} |`))
})

test('diffCoverageManifests is sorted/deterministic', () => {
  const base: CoverageManifest = {
    version: 1,
    generatedNote: 'a',
    exclusions: [],
    entries: [
      {
        id: 'A',
        path: 'docs/completed/a.md',
        markets: ['ALL'],
        risk: 'minor',
        client: 'web',
        coverage: 'smoke',
        rationale: 'x',
        owner: 'QA',
        reviewed: true,
        specs: [{ path: 'e2e/tests/auth.spec.ts' }],
      },
      {
        id: 'B',
        path: 'docs/completed/b.md',
        markets: ['ALL'],
        risk: 'minor',
        client: 'web',
        coverage: 'missing',
        rationale: 'x',
        owner: 'QA',
        severity: 'minor',
        targetMilestone: 'm',
        reviewed: true,
      },
    ],
  }
  const next: CoverageManifest = {
    version: 1,
    generatedNote: 'b',
    exclusions: [],
    entries: [
      { ...base.entries[0]!, coverage: 'journey' },
      {
        id: 'C',
        path: 'docs/completed/c.md',
        markets: ['ALL'],
        risk: 'minor',
        client: 'web',
        coverage: 'not-applicable',
        rationale: 'x',
        owner: 'QA',
        reviewed: true,
      },
    ],
  }
  const diff = diffCoverageManifests(base, next)
  assert.deepEqual(diff.added, ['docs/completed/c.md'])
  assert.deepEqual(diff.removed, ['docs/completed/b.md'])
  assert.deepEqual(diff.levelChanges, ['docs/completed/a.md: smoke → journey'])
})

test('repo manifest validates and scan is under 10s', () => {
  const started = Date.now()
  const errors = validateCompletedFeatureCoverage({ repoRoot: REPO_ROOT })
  const elapsed = Date.now() - started
  assert.deepEqual(errors, [])
  assert.ok(elapsed < 10_000, `scan took ${elapsed}ms`)
  const eligible = listEligibleCompletedStories(REPO_ROOT)
  const manifest = loadManifest(resolveManifestPath(REPO_ROOT))
  assert.equal(eligible.length, manifest.entries.length)
})

console.log('e2e:coverage:test — all passed')

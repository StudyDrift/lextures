/**
 * VPAT (Voluntary Product Accessibility Template) e2e tests (plan 10.8).
 *
 * Verifies that /accessibility/vpat is publicly accessible, contains the
 * required VPAT 2.5 INT content, is axe-clean, and has correct download and
 * accommodation contact links.
 *
 * Covers FR-1 through FR-6, AC-1 through AC-5, and the §16 test plan.
 */
import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

async function axeScan(page: import('@playwright/test').Page) {
  return new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .disableRules(['color-contrast']) // audited manually; headless Tailwind CSS produces false positives
    .analyze()
}

function assertNoViolations(results: Awaited<ReturnType<AxeBuilder['analyze']>>) {
  const critical = results.violations.filter(
    (v) => v.impact === 'critical' || v.impact === 'serious',
  )
  if (critical.length > 0) {
    const summary = critical.map((v) => `  [${v.impact}] ${v.id}: ${v.description}`).join('\n')
    throw new Error(`Axe found ${critical.length} critical/serious violation(s):\n${summary}`)
  }
}

// ── AC-1: publicly accessible without authentication ──────────────────────────

test.describe('VPAT page — public access (AC-1)', () => {
  test('loads without authentication and does not redirect to login', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await expect(page).toHaveURL('/accessibility/vpat')
    await expect(page.getByRole('heading', { level: 1, name: /accessibility conformance report/i })).toBeVisible()
  })

  test('page title includes VPAT and Lextures (FR-4)', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await expect(page).toHaveTitle(/VPAT.*Lextures|Lextures.*VPAT/i)
  })
})

// ── WCAG content completeness ─────────────────────────────────────────────────

test.describe('VPAT content — WCAG 2.1 criteria (FR-1, FR-2)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await page.waitForLoadState('networkidle')
  })

  test('contains WCAG 2.1 Level A heading', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /level a success criteria/i })).toBeVisible()
  })

  test('contains WCAG 2.1 Level AA heading', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /level aa success criteria/i })).toBeVisible()
  })

  test('WCAG tables have SC, Title, Conformance, and Remarks columns', async ({ page }) => {
    const tables = page.getByRole('table', { name: /wcag success criteria/i })
    const first = tables.first()
    await expect(first).toBeVisible()
    await expect(first.getByRole('columnheader', { name: /^sc$/i })).toBeVisible()
    await expect(first.getByRole('columnheader', { name: /title/i })).toBeVisible()
    await expect(first.getByRole('columnheader', { name: /conformance/i })).toBeVisible()
    await expect(first.getByRole('columnheader', { name: /remarks/i })).toBeVisible()
  })

  test('SC 1.1.1 Non-text Content is present (FR-2)', async ({ page }) => {
    await expect(page.getByText('1.1.1')).toBeVisible()
    await expect(page.getByText('Non-text Content')).toBeVisible()
  })

  // AC-2: 1.2.2 Captions shows Partially Supports with a specific remark
  test('SC 1.2.2 Captions (Prerecorded) shows Partially Supports with remark (AC-2)', async ({ page }) => {
    await expect(page.getByText('1.2.2')).toBeVisible()
    // "Partially Supports" badge should appear at least once in the table
    const badges = page.getByText('Partially Supports')
    await expect(badges.first()).toBeVisible()
    // The remarks for 1.2.2 mention captions being in progress
    await expect(page.getByText(/auto-captions.*in progress/i)).toBeVisible()
  })

  test('Level AA criterion 4.1.3 Status Messages is present', async ({ page }) => {
    await expect(page.getByText('4.1.3')).toBeVisible()
    await expect(page.getByText('Status Messages')).toBeVisible()
  })
})

// ── Section 508 content ───────────────────────────────────────────────────────

test.describe('VPAT content — Section 508 (FR-1)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await page.waitForLoadState('networkidle')
  })

  test('has Section 508 Functional Performance Criteria section', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /chapter 3.*functional performance/i })).toBeVisible()
  })

  test('FPC table contains 302.1 Without Vision', async ({ page }) => {
    await expect(page.getByText('302.1')).toBeVisible()
    await expect(page.getByText('Without Vision')).toBeVisible()
  })

  test('has Section 508 Chapter 5 Software section', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /chapter 5.*software/i })).toBeVisible()
  })

  test('Chapter 5 table contains 502.2.1', async ({ page }) => {
    await expect(page.getByText('502.2.1')).toBeVisible()
  })

  test('has Section 508 Chapter 6 Support Documentation section', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /chapter 6.*support documentation/i })).toBeVisible()
  })

  test('Chapter 6 table contains 602.2', async ({ page }) => {
    await expect(page.getByText('602.2')).toBeVisible()
  })
})

// ── EN 301 549 content ────────────────────────────────────────────────────────

test.describe('VPAT content — EN 301 549 (FR-1)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await page.waitForLoadState('networkidle')
  })

  test('has EN 301 549 Chapter 9 Web section', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /chapter 9.*web/i })).toBeVisible()
  })

  test('Chapter 9 references WCAG tables', async ({ page }) => {
    await expect(page.getByText(/clauses 9\.1\.1\.1.*wcag 2\.1/i)).toBeVisible()
  })

  test('has EN 301 549 Chapter 11 Software section', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /chapter 11.*software/i })).toBeVisible()
  })

  test('Chapter 11 table contains 11.5.2', async ({ page }) => {
    await expect(page.getByText('11.5.2')).toBeVisible()
  })

  test('has EN 301 549 Chapter 12 Documentation section', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /chapter 12.*documentation/i })).toBeVisible()
  })
})

// ── FR-3: Download link ───────────────────────────────────────────────────────

test.describe('VPAT download (FR-3)', () => {
  test('download link is present and has download attribute', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    const downloadLink = page.getByRole('link', { name: /download vpat/i })
    await expect(downloadLink).toBeVisible()
    await expect(downloadLink).toHaveAttribute('download')
  })

  test('download link points to a VPAT file', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    const downloadLink = page.getByRole('link', { name: /download vpat/i })
    const href = await downloadLink.getAttribute('href')
    expect(href).toMatch(/VPAT_2\.5_INT_Lextures/)
  })
})

// ── FR-4: Product version and evaluation date ─────────────────────────────────

test.describe('Product version and date (FR-4)', () => {
  test('product information section shows evaluation date', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await expect(page.getByText(/May 27, 2026/)).toBeVisible()
  })

  test('product information section shows product name', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await expect(page.getByRole('heading', { name: /product information/i })).toBeVisible()
  })
})

// ── FR-6: Accommodation contact ───────────────────────────────────────────────

test.describe('Accessibility support contact (FR-6)', () => {
  test('Contact Accessibility Support heading is present', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await expect(page.getByRole('heading', { name: /contact accessibility support/i })).toBeVisible()
  })

  // AC-5: contact link pre-populates subject
  test('contact link routes to mailto with pre-populated accommodation subject (AC-5)', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    const contactLink = page.getByRole('link', { name: /contact accessibility support/i })
    await expect(contactLink).toBeVisible()
    const href = await contactLink.getAttribute('href')
    expect(href).toContain('accessibility@lextures.com')
    expect(href).toContain('subject=Accessibility%20accommodation%20request')
  })
})

// ── AC-4: Accessible table navigation ────────────────────────────────────────

test.describe('Accessible table markup (AC-4)', () => {
  test('WCAG table headers are correctly scoped', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await page.waitForLoadState('networkidle')

    const firstTable = page.getByRole('table', { name: /wcag success criteria/i }).first()
    await expect(firstTable).toBeVisible()

    // All column headers must have scope="col"
    const headers = firstTable.locator('th[scope="col"]')
    await expect(headers).toHaveCount(4)
  })

  test('FPC table has accessible label', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    const fpcTable = page.getByRole('table', { name: /functional performance criteria/i })
    await expect(fpcTable).toBeVisible()
  })

  test('table of contents navigation is present and usable', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    const toc = page.getByRole('navigation', { name: /report sections/i })
    await expect(toc).toBeVisible()
    const links = toc.getByRole('link')
    await expect(links).toHaveCount(10)
  })
})

// ── axe-core accessibility gate ───────────────────────────────────────────────

test.describe('VPAT page — axe-core accessibility (§16 test plan)', () => {
  test('VPAT page has no critical/serious WCAG violations', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    await page.waitForLoadState('networkidle')
    const results = await axeScan(page)
    assertNoViolations(results)
  })
})

// ── Footer and header navigation ─────────────────────────────────────────────

test.describe('Navigation links', () => {
  test('header nav contains Conformance Statement link back to /accessibility', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    const nav = page.getByRole('navigation', { name: 'Legal' })
    await expect(nav.getByRole('link', { name: /conformance statement/i })).toBeVisible()
  })

  test('header nav contains Privacy and Terms links', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    const nav = page.getByRole('navigation', { name: 'Legal' })
    await expect(nav.getByRole('link', { name: 'Privacy' })).toBeVisible()
    await expect(nav.getByRole('link', { name: 'Terms' })).toBeVisible()
  })

  test('footer contains Accessibility Statement link', async ({ page }) => {
    await page.goto('/accessibility/vpat')
    const footer = page.locator('footer')
    await expect(footer.getByRole('link', { name: /accessibility statement/i })).toBeVisible()
  })
})

// ── Conformance statement links to VPAT ──────────────────────────────────────

test.describe('Conformance statement links to VPAT', () => {
  test('/accessibility page header nav contains VPAT link', async ({ page }) => {
    await page.goto('/accessibility')
    const nav = page.getByRole('navigation', { name: 'Legal' })
    await expect(nav.getByRole('link', { name: 'VPAT' })).toBeVisible()
  })

  test('/accessibility footer links to VPAT', async ({ page }) => {
    await page.goto('/accessibility')
    const footer = page.locator('footer')
    await expect(footer.getByRole('link', { name: 'VPAT' })).toBeVisible()
  })

  test('VPAT link in conformance statement navigates to /accessibility/vpat', async ({ page }) => {
    await page.goto('/accessibility')
    const nav = page.getByRole('navigation', { name: 'Legal' })
    await nav.getByRole('link', { name: 'VPAT' }).click()
    await expect(page).toHaveURL('/accessibility/vpat')
    await expect(page.getByRole('heading', { level: 1, name: /accessibility conformance report/i })).toBeVisible()
  })
})

// ── Privacy policy links to VPAT ─────────────────────────────────────────────

test.describe('Privacy policy accessibility section', () => {
  test('privacy policy contains link to /accessibility/vpat', async ({ page }) => {
    await page.goto('/privacy')
    await page.waitForLoadState('networkidle')
    const vpatLink = page.getByRole('link', { name: /accessibility conformance report.*vpat/i })
    await expect(vpatLink).toBeVisible()
    await expect(vpatLink).toHaveAttribute('href', '/accessibility/vpat')
  })
})

// ── Sidebar footer Accessibility link ────────────────────────────────────────

test.describe('Sidebar footer Accessibility link (authenticated)', () => {
  test('authenticated sidebar footer contains Accessibility link', async ({ page }) => {
    // Inject a valid token to reach the authenticated shell
    const signupRes = await page.request.post(
      `${process.env.E2E_API_URL ?? 'http://localhost:8080'}/api/v1/auth/signup`,
      {
        data: {
          email: `vpat-e2e-${Date.now()}@test.invalid`,
          password: 'E2eTestPass1!',
          displayName: 'VPAT E2E',
        },
      },
    )
    const { access_token } = (await signupRes.json()) as { access_token: string }

    await page.goto('/')
    await page.evaluate((token: string) => {
      localStorage.setItem('studydrift_access_token', token)
    }, access_token)
    await page.goto('/')
    await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible({ timeout: 15000 })

    const sideNavFooter = page.locator('footer').filter({ hasText: /accessibility/i })
    await expect(sideNavFooter.getByRole('link', { name: /accessibility/i })).toBeVisible()
  })
})

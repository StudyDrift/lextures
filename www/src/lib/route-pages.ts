/**
 * Lazy page loaders for per-route code splitting (SEO.4 FR-3).
 * route-manifest stays free of eager page imports so content routes do not
 * pull marketplace / calculator / markdown into every visit.
 */
import type { ComponentType } from 'react'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyPage = ComponentType<any>

type Loader = () => Promise<AnyPage>

function pick(mod: Record<string, unknown>, name: string): AnyPage {
  const C = mod[name] || mod.default
  if (!C || (typeof C !== 'function' && typeof C !== 'object')) {
    throw new Error(`Page export "${name}" not found`)
  }
  return C as AnyPage
}

/** Map of path pattern → async page component loader. */
export const PAGE_LOADERS: Record<string, Loader> = {
  '/': () => import('../pages/home-page').then(m => pick(m, 'HomePage')),
  '/about': () => import('../pages/about-page').then(m => pick(m, 'AboutPage')),
  '/press': () => import('../pages/press-page').then(m => pick(m, 'PressPage')),
  '/authors': () => import('../pages/authors-index-page').then(m => pick(m, 'AuthorsIndexPage')),
  '/authors/:slug': () => import('../pages/author-page').then(m => pick(m, 'AuthorPage')),
  '/get-started': () => import('../pages/get-started-page').then(m => pick(m, 'GetStartedPage')),
  '/parents': () => import('../pages/parents-page').then(m => pick(m, 'ParentsPage')),
  '/higher-ed': () => import('../pages/higher-ed-page').then(m => pick(m, 'HigherEdPage')),
  '/k12': () => import('../pages/k12-page').then(m => pick(m, 'K12Page')),
  '/homeschool': () => import('../pages/homeschool-page').then(m => pick(m, 'HomeschoolPage')),
  '/pricing': () => import('../pages/pricing-page').then(m => pick(m, 'PricingPage')),
  '/pricing/calculator': () =>
    import('../pages/pricing-calculator-page').then(m => pick(m, 'PricingCalculatorPage')),
  '/courses': () => import('../pages/courses-page').then(m => pick(m, 'CoursesPage')),
  '/courses/subject/:subject': () => import('../pages/catalog-hub-page').then(m => pick(m, 'CatalogHubPage')),
  '/courses/level/:level': () => import('../pages/catalog-hub-page').then(m => pick(m, 'CatalogHubPage')),
  '/courses/:slug': () => import('../pages/course-detail-page').then(m => pick(m, 'CourseDetailPage')),
  '/request-information': () =>
    import('../pages/request-information-page').then(m => pick(m, 'RequestInformationPage')),
  ...Object.fromEntries(
    [
      '/platform', '/platform/adaptive-learning', '/platform/assessment', '/platform/grading',
      '/platform/analytics', '/platform/accessibility', '/platform/ai', '/resources', '/guides',
      '/research', '/trust', '/compare', '/integrations',
      '/alternatives',
    ].map(path => [path, () => import('../pages/ia-page').then(m => pick(m, 'IaPage'))]),
  ),
  '/glossary': () => import('../pages/utility-pages').then(m => pick(m, 'GlossaryIndexPage')),
  '/glossary/:term': () => import('../pages/utility-pages').then(m => pick(m, 'GlossaryTermPage')),
  '/standards': () => import('../pages/utility-pages').then(m => pick(m, 'StandardsIndexPage')),
  '/standards/:framework': () => import('../pages/utility-pages').then(m => pick(m, 'StandardsHubPage')),
  '/standards/:framework/:grade': () => import('../pages/utility-pages').then(m => pick(m, 'StandardsHubPage')),
  '/standards/:framework/:grade/:code': () => import('../pages/utility-pages').then(m => pick(m, 'StandardPage')),
  '/templates': () => import('../pages/utility-pages').then(m => pick(m, 'TemplatesIndexPage')),
  '/templates/:slug': () => import('../pages/utility-pages').then(m => pick(m, 'TemplatePage')),
  '/tools': () => import('../pages/utility-pages').then(m => pick(m, 'ToolsIndexPage')),
  '/tools/:tool': () => import('../pages/utility-pages').then(m => pick(m, 'ToolPage')),
  '/compare/lextures-vs-:slug': () => import('../pages/comparison-page').then(m => pick(m, 'ComparisonPage')),
  '/alternatives/:slug': () => import('../pages/comparison-page').then(m => pick(m, 'AlternativesPage')),
  '/integrations/:slug': () => import('../pages/comparison-page').then(m => pick(m, 'IntegrationPage')),
  '/blog': () => import('../pages/blog-index').then(m => pick(m, 'BlogIndex')),
  '/resources/guides': () => import('../pages/guides-index-page').then(m => pick(m, 'GuidesIndexPage')),
  '/resources/research': () => import('../pages/research-pages').then(m => pick(m, 'ResearchIndexPage')),
  '/resources/research/methodology': () => import('../pages/research-pages').then(m => pick(m, 'ResearchMethodologyPage')),
  '/blog/:slug': () => import('../pages/blog-post').then(m => pick(m, 'BlogPost')),
  '/docs': () => import('../pages/docs-index').then(m => pick(m, 'DocsIndex')),
  '/docs/:category': () => import('../pages/docs-category').then(m => pick(m, 'DocsCategory')),
  '/docs/:category/:slug': () => import('../pages/docs-post').then(m => pick(m, 'DocsPost')),
  '/internal/content-kit': () => import('../pages/content-kit-page').then(m => pick(m, 'ContentKitPage')),
  '/privacy': () => import('../pages/legal-pages').then(m => pick(m, 'PrivacyPolicyPage')),
  '/privacy/history': () =>
    import('../pages/legal-pages').then(m => pick(m, 'PrivacyPolicyHistoryPage')),
  '/terms': () => import('../pages/legal-pages').then(m => pick(m, 'TermsOfServicePage')),
  '/terms/history': () =>
    import('../pages/legal-pages').then(m => pick(m, 'TermsOfServiceHistoryPage')),
  '/security': () => import('../pages/security-page').then(m => pick(m, 'SecurityPage')),
  '/accessibility': () =>
    import('../pages/accessibility-conformance-page').then(m => pick(m, 'AccessibilityConformancePage')),
  '/accessibility/vpat': () => import('../pages/vpat-page').then(m => pick(m, 'VpatPage')),
  '/privacy-rights/california': () =>
    import('../pages/california-privacy-rights-page').then(m =>
      pick(m, 'CaliforniaPrivacyRightsPage'),
    ),
  '/404': () => import('../pages/not-found-page').then(m => pick(m, 'NotFoundPage')),
}

export function getPageLoader(pattern: string): Loader | null {
  return PAGE_LOADERS[pattern] ?? null
}

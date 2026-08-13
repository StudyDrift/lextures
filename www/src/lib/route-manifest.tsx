/**
 * Single source of truth for marketing-site routes + SEO metadata (SEO.1 FR-1).
 * Consumed by the client router, static generator, sitemap builder, and CI.
 *
 * SEO.4: page components load via `route-pages.ts` dynamic import() (FR-3).
 * `interactive: false` skips client React hydration (FR-4).
 */
import type { ComponentType, ReactElement } from 'react'
import { blogPostMeta, docArticleMeta } from '../utils/content-meta'
import { HELP_CATEGORIES, getHelpCategory } from '../docs/_categories'
import { getActiveAuthors, getAuthor, isAuthorLinkable } from './authors'
import type { JsonLdNode } from './document-head'
import { truncateMetaDescription, truncateTitle } from './document-head'
import { COURSES_COPY } from './courses-copy'
import { COMPETITORS, getCompetitor, VERIFIED_AT } from './competitors'
import { INTEGRATIONS, getIntegration } from './integrations'
import { GLOSSARY_TERMS, STANDARDS, TEMPLATES, TOOL_SLUGS, getGlossaryTerm, getStandard, getTemplate } from '../data/utility-content'
import { getPageLoader } from './route-pages'
import { canonicalUrl, SITE_ORIGIN } from './site-origin'
import {
  accessibilityGraph,
  aboutPageGraph,
  authorPageGraph,
  authorsIndexGraph,
  blogPostGraph,
  coursesIndexGraph,
  defaultPageGraph,
  docsPostGraph,
  homePageGraph,
  legalDocumentGraph,
  pricingPageGraph,
  securityGraph,
  comparisonPageGraph,
  integrationPageGraph,
} from './schema/page-graphs'
import { DEFAULT_LOCALE, getLocale, isPublishedLocale, isValidLocaleCode, isWellFormedBcp47, localizedPath, type LocaleCode, type TranslationStatus } from './locales'

export type { JsonLdNode }

export type RenderContext = {
  path: string
  origin: string
  params: Record<string, string>
}

export type RouteDescriptor = {
  /** Path pattern: `/pricing` or `/blog/:slug`. No trailing slash except `/`. */
  path: string
  title: string | ((ctx: RenderContext) => string)
  description: string | ((ctx: RenderContext) => string)
  changefreq?: 'daily' | 'weekly' | 'monthly' | 'yearly'
  /** Include in sitemap.xml (false for legal history, thank-you, 404). */
  sitemap: boolean
  robots?: 'index,follow' | 'noindex,follow'
  ogImage?: string
  jsonLd?: (ctx: RenderContext) => JsonLdNode[]
  lastmodSource?: 'git' | 'content' | 'build'
  /** BCP 47 locale. English remains unprefixed. */
  locale: LocaleCode
  /** Unprefixed English source path for a localized route. */
  translationOf?: string
  translationStatus?: TranslationStatus
  /** Source version reviewed by the translator, used for staleness reporting. */
  sourceUpdatedAt?: string
  /**
   * When false, the page is prerendered HTML only — no React hydration
   * (SEO.4 FR-4). Header/footer work without JS via progressive enhancement.
   * Default true for interactive marketplace / forms / calculator.
   */
  interactive?: boolean
  /**
   * For dynamic families (`/blog/:slug`, `/docs/:slug`, `/courses/:slug`),
   * list every concrete path known at build time.
   */
  enumerate?: () => Array<{
    path: string
    title?: string
    description?: string
    lastmod?: string
    robots?: 'index,follow' | 'noindex,follow'
    params?: Record<string, string>
  }>
  /** Sitemap priority hint (0.0–1.0). */
  priority?: string
  /** IA declarations used for breadcrumbs, hubs, related content, and graph checks. */
  parent?: string
  hub?: boolean
  cluster?: string
  relatedTo?: string[]
  navGroup?: 'platform' | 'solutions' | 'resources' | 'trust' | 'company'
}

function ctx(path: string, params: Record<string, string> = {}): RenderContext {
  return { path, origin: SITE_ORIGIN, params }
}

function matchPattern(pattern: string, pathname: string): Record<string, string> | null {
  if (!pattern.includes(':')) {
    return pattern === pathname ? {} : null
  }
  const patternParts = pattern.split('/').filter(Boolean)
  const pathParts = pathname.split('/').filter(Boolean)
  if (patternParts.length !== pathParts.length) return null
  const params: Record<string, string> = {}
  for (let i = 0; i < patternParts.length; i++) {
    const pp = patternParts[i]
    const vp = pathParts[i]
    if (pp.startsWith(':')) {
      params[pp.slice(1)] = decodeURIComponent(vp)
    } else if (pp !== vp) {
      return null
    }
  }
  return params
}

/** Props passed into the page component for a given pattern. */
export function pagePropsFor(
  pattern: string,
  renderCtx: RenderContext,
): Record<string, unknown> {
  if (pattern.includes(':')) return { ...renderCtx.params }
  return {}
}

export type HreflangAlternate = { hreflang: string; href: string }

function isPublishable(descriptor: RouteDescriptor): boolean {
  return isValidLocaleCode(descriptor.locale) && isPublishedLocale(descriptor.locale) &&
    (descriptor.locale === DEFAULT_LOCALE || descriptor.translationStatus === 'complete')
}

/** Static marketing routes (ordered for predictable manifests). */
const ENGLISH_ROUTE_MANIFEST: Array<Omit<RouteDescriptor, 'locale'>> = [
  {
    path: '/',
    title: 'Lextures — The learning environment that adapts',
    description:
      'The learning environment that adapts. One platform for adaptive quizzing, interactive content, grading, and enrollment — instead of a patchwork of vendors.',
    changefreq: 'weekly',
    sitemap: true,
    priority: '1.0',
    lastmodSource: 'build',
    // CSS-only hero motion — skip React hydration so lab TBT stays in the content budget.
    interactive: false,
    jsonLd: homePageGraph,
    hub: true,
  },
  {
    path: '/about',
    title: 'About Lextures — Who we are',
    description:
      'Lextures is an adaptive LMS for K–12, higher education, and homeschool. Founded by Chase Willden; open source under AGPL-3.0.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.8',
    interactive: false,
    jsonLd: aboutPageGraph,
    parent: '/', navGroup: 'company',
  },
  {
    path: '/press',
    title: 'Press & media resources — Lextures',
    description:
      'Lextures company facts, approved boilerplate, brand assets, founder information, research resources, and media contact.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.6',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Press'),
    parent: '/about',
    navGroup: 'company',
  },
  {
    path: '/authors',
    title: 'Authors — Lextures',
    description:
      'Named authors behind Lextures blog posts and documentation — credentials, bios, and published work.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.5',
    interactive: false,
    jsonLd: authorsIndexGraph,
    parent: '/about', hub: true, navGroup: 'company',
  },
  {
    path: '/authors/:slug',
    title: c => {
      const a = getAuthor(c.params.slug ?? '')
      return truncateTitle(a ? `${a.name} — Lextures` : 'Author — Lextures')
    },
    description: c => {
      const a = getAuthor(c.params.slug ?? '')
      return truncateMetaDescription(a?.bio || 'Lextures author profile')
    },
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.4',
    interactive: false,
    jsonLd: authorPageGraph,
    parent: '/authors', navGroup: 'company',
    enumerate: () =>
      getActiveAuthors()
        .filter(a => isAuthorLinkable(a.slug))
        .map(a => ({
          path: `/authors/${a.slug}`,
          title: truncateTitle(`${a.name} — Lextures`),
          description: truncateMetaDescription(a.bio),
          params: { slug: a.slug },
        })),
  },
  {
    path: '/get-started',
    title: 'Get started — Lextures',
    description:
      'Create a free Lextures account for your school or start as a homeschool learner. Self-host or use our hosted apps.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.8',
    interactive: true,
    jsonLd: c => defaultPageGraph(c, 'Get started'),
    parent: '/',
  },
  {
    path: '/parents',
    title: 'Parents & guardians — Lextures',
    description:
      "See your child's grades and due dates in one place when your district enables the Lextures parent portal.",
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.6',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Parents'),
    parent: '/', navGroup: 'solutions', cluster: 'parents',
  },
  {
    path: '/higher-ed',
    title: 'Higher education — Lextures',
    description:
      'Adaptive LMS for colleges and universities: SSO, SCIM, LTI 1.3, grade audit trails, and self-hosting under AGPL-3.0.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.6',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Higher education'),
    parent: '/', navGroup: 'solutions', cluster: 'higher-ed',
  },
  {
    path: '/k12',
    title: 'K–12 schools — Lextures',
    description:
      'Standards-aligned gradebook, roster sync, spaced repetition, and misconception flags for K–12 districts and schools.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.6',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'K–12'),
    parent: '/', navGroup: 'solutions', cluster: 'k12',
  },
  {
    path: '/homeschool',
    title: 'Homeschool — Lextures',
    description:
      'Create or enroll in courses, practice with IRT-routed quizzes, and clear spaced-repetition reviews. Self-host free or use self.lextures.com.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.6',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Homeschool'),
    parent: '/', navGroup: 'solutions', cluster: 'homeschool',
  },
  {
    path: '/pricing',
    title: 'Pricing — Lextures',
    description:
      'Self-host free under AGPL-3.0, homeschool plans from $20/month, and per-student institution pricing with bulk discounts.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.8',
    interactive: false,
    jsonLd: pricingPageGraph,
    parent: '/',
  },
  {
    path: '/pricing/calculator',
    title: 'Pricing calculator — Lextures',
    description:
      'Estimate hosted Lextures cost by enrollment size with automatic bulk discounts for schools and districts.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.7',
    interactive: true,
    jsonLd: c => defaultPageGraph(c, 'Calculator'),
    parent: '/pricing',
  },
  {
    path: '/courses',
    title: COURSES_COPY.pageTitle,
    description: COURSES_COPY.pageDescription,
    changefreq: 'daily',
    sitemap: true,
    priority: '0.9',
    interactive: true,
    jsonLd: coursesIndexGraph,
    parent: '/', hub: true,
  },
  {
    path: '/courses/subject/:subject',
    title: c => `${c.params.subject} courses — Lextures`,
    description: c => `Explore ${c.params.subject} courses on Lextures, compare levels, formats, course content, and enrollment options.`,
    changefreq: 'daily', sitemap: true, priority: '0.7', interactive: true,
    parent: '/courses', hub: true,
  },
  {
    path: '/courses/level/:level',
    title: c => `${c.params.level} courses — Lextures`,
    description: c => `Explore ${c.params.level} courses on Lextures, compare subjects, course content, and enrollment options.`,
    changefreq: 'daily', sitemap: true, priority: '0.7', interactive: true,
    parent: '/courses', hub: true,
  },
  {
    path: '/courses/:slug',
    title: c => {
      const slug = c.params.slug ?? 'course'
      return truncateTitle(`${slug} — Lextures`)
    },
    description: COURSES_COPY.pageDescription,
    changefreq: 'weekly',
    sitemap: true,
    priority: '0.8',
    interactive: true,
    parent: '/courses',
    enumerate: () => [],
  },
  {
    path: '/request-information',
    title: 'Request information — Lextures',
    description:
      'Request a Lextures demo or quote for your school, district, or university. Tell us about enrollment size and SSO needs.',
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.7',
    interactive: true,
    jsonLd: c => defaultPageGraph(c, 'Request information'),
    parent: '/',
  },
  ...[
    ['/platform', 'Platform', 'Explore the adaptive learning, assessment, grading, analytics, accessibility, and AI capabilities in Lextures.', '/', 'platform', true],
    ['/platform/adaptive-learning', 'Adaptive learning', 'Build practice that responds to each learner with mastery signals and spaced review.', '/platform', 'platform'],
    ['/platform/assessment', 'Assessment', 'Design quizzes and assessments with reusable question banks and actionable feedback.', '/platform', 'platform'],
    ['/platform/grading', 'Grading', 'Keep grading consistent with rubrics, audit trails, and a unified gradebook.', '/platform', 'platform'],
    ['/platform/analytics', 'Learning analytics', 'Turn progress and misconception signals into timely instructional decisions.', '/platform', 'platform'],
    ['/platform/accessibility', 'Accessible learning', 'Create inclusive learning experiences designed around WCAG 2.2 AA.', '/platform', 'platform'],
    ['/platform/ai', 'AI for learning', 'Use AI as an accountable assistant for course creation, practice, and feedback.', '/platform', 'platform'],
    ['/resources', 'Resources', 'Practical articles, guides, research, templates, and definitions for better learning systems.', '/', 'resources', true],
    ['/guides', 'Guides', 'In-depth guides to assessment, adaptive learning, course design, and LMS operations.', '/resources', 'resources', true],
    ['/research', 'Research', 'Original Lextures research about learning, assessment, and education technology.', '/resources', 'resources', true],
    ['/trust', 'Trust center', 'Review Lextures security, privacy, accessibility, and legal commitments.', '/', 'trust', true],
    ['/compare', 'Compare Lextures', 'Compare Lextures with other approaches to learning management and course delivery.', '/', 'company', true],
    ['/alternatives', 'LMS alternatives', 'Evaluate LMS and course-platform alternatives using transparent, evidence-led criteria.', '/', 'company', true],
    ['/integrations', 'Integrations', 'Connect Lextures with the tools institutions and educators already use.', '/platform', 'platform', true],
  ].map(([path, label, description, parent, navGroup, hub]) => ({
    path: path as string,
    title: `${label} — Lextures`,
    description: description as string,
    changefreq: 'monthly' as const,
    sitemap: true,
    priority: hub ? '0.7' : '0.6',
    interactive: false,
    jsonLd: (c: RenderContext) => defaultPageGraph(c, label as string),
    parent: parent as string,
    navGroup: navGroup as RouteDescriptor['navGroup'],
    hub: Boolean(hub),
  })),
  {
    path: '/glossary', title: 'Learning and assessment glossary — Lextures',
    description: 'Reviewed definitions, examples, sources, and practical decisions for assessment, adaptive learning, and instructional design.',
    changefreq: 'monthly', sitemap: true, priority: '0.7', interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Glossary'), parent: '/resources', hub: true, navGroup: 'resources',
  },
  {
    path: '/glossary/:term',
    title: c => truncateTitle(`${getGlossaryTerm(c.params.term)?.term ?? 'Glossary term'} — definition`),
    description: c => truncateMetaDescription(getGlossaryTerm(c.params.term)?.sentence ?? 'Reviewed learning and assessment definition.'),
    changefreq: 'monthly', sitemap: false, robots: 'noindex,follow', priority: '0.6', interactive: true,
    jsonLd: c => defaultPageGraph(c, getGlossaryTerm(c.params.term)?.term), parent: '/glossary', cluster: 'utility',
    enumerate: () => GLOSSARY_TERMS.map(item => ({ path: `/glossary/${item.slug}`, params: { term: item.slug } })),
  },
  {
    path: '/standards', title: 'Academic standards browser — Lextures', description: 'Browse US academic standards with mastery evidence, assessment approaches, misconceptions, and official source links.',
    changefreq: 'monthly', sitemap: true, priority: '0.7', interactive: false, jsonLd: c => defaultPageGraph(c, 'Standards'), parent: '/resources', hub: true, navGroup: 'resources',
  },
  {
    path: '/standards/:framework', title: c => `${STANDARDS.find(s => s.framework === c.params.framework)?.frameworkName ?? 'Standards'} — Lextures`, description: 'Browse a licence-reviewed academic standards framework with original instructional and assessment guidance.',
    changefreq: 'monthly', sitemap: false, robots: 'noindex,follow', priority: '0.6', interactive: false, jsonLd: c => defaultPageGraph(c, 'Framework'), parent: '/standards', hub: true,
    enumerate: () => [...new Set(STANDARDS.map(s => s.framework))].map(framework => ({ path: `/standards/${framework}`, params: { framework } })),
  },
  {
    path: '/standards/:framework/:grade', title: c => truncateTitle(`${c.params.grade.replaceAll('-', ' ')} ${STANDARDS.find(s => s.framework === c.params.framework)?.frameworkName ?? 'standards'}`), description: 'Browse standards for this grade band with distinct URLs, mastery indicators, assessment approaches, and official sources.',
    changefreq: 'monthly', sitemap: false, robots: 'noindex,follow', priority: '0.6', interactive: false, jsonLd: c => defaultPageGraph(c, 'Grade'), parent: '/standards/:framework', hub: true,
    enumerate: () => [...new Map(STANDARDS.map(s => [`${s.framework}/${s.grade}`, s])).values()].map(s => ({ path: `/standards/${s.framework}/${s.grade}`, params: { framework: s.framework, grade: s.grade } })),
  },
  {
    path: '/standards/:framework/:grade/:code', title: c => truncateTitle(`${c.params.code.toUpperCase()} assessment guidance — Lextures`), description: c => truncateMetaDescription(getStandard(c.params.framework, c.params.grade, c.params.code)?.summary ?? 'Standard assessment guidance.'),
    changefreq: 'monthly', sitemap: true, priority: '0.6', interactive: false, jsonLd: c => defaultPageGraph(c, c.params.code.toUpperCase()), parent: '/standards/:framework/:grade', cluster: 'utility',
    enumerate: () => STANDARDS.map(s => ({ path: `/standards/${s.framework}/${s.grade}/${s.code}`, params: { framework: s.framework, grade: s.grade, code: s.code } })),
  },
  {
    path: '/templates', title: 'Teaching and course-design templates — Lextures', description: 'Preview and download ungated PDF and editable templates for rubrics, assessment blueprints, and accessibility reviews.',
    changefreq: 'monthly', sitemap: true, priority: '0.7', interactive: false, jsonLd: c => defaultPageGraph(c, 'Templates'), parent: '/resources', hub: true, navGroup: 'resources',
  },
  {
    path: '/templates/:slug', title: c => truncateTitle(`${getTemplate(c.params.slug)?.title ?? 'Template'} — Lextures`), description: c => truncateMetaDescription(getTemplate(c.params.slug)?.description ?? 'Downloadable teaching template.'),
    changefreq: 'monthly', sitemap: false, robots: 'noindex,follow', priority: '0.6', interactive: false, jsonLd: c => defaultPageGraph(c, getTemplate(c.params.slug)?.title), parent: '/templates', cluster: 'utility',
    enumerate: () => TEMPLATES.map(item => ({ path: `/templates/${item.slug}`, params: { slug: item.slug } })),
  },
  { path: '/tools', title: 'Free teaching tools — Lextures', description: 'Use private, client-side grade, rubric, reading-level, assessment-blueprint, and accessibility tools without an account.', changefreq: 'monthly', sitemap: true, priority: '0.7', interactive: false, jsonLd: c => defaultPageGraph(c, 'Tools'), parent: '/resources', hub: true, navGroup: 'resources' },
  {
    path: '/tools/:tool', title: c => truncateTitle(`${c.params.tool.replaceAll('-', ' ')} — Lextures`), description: c => truncateMetaDescription(`Use the free ${c.params.tool.replaceAll('-', ' ')} in your browser without an account; inputs stay on your device and results are shareable.`),
    changefreq: 'monthly', sitemap: false, robots: 'noindex,follow', priority: '0.6', interactive: true, jsonLd: c => defaultPageGraph(c, c.params.tool.replaceAll('-', ' ')), parent: '/tools', cluster: 'utility',
    enumerate: () => TOOL_SLUGS.map(tool => ({ path: `/tools/${tool}`, params: { tool } })),
  },
  {
    path: '/compare/lextures-vs-:slug',
    title: c => truncateTitle(`Lextures vs. ${getCompetitor(c.params.slug)?.name || 'LMS'}`),
    description: c => truncateMetaDescription(`Compare Lextures and ${getCompetitor(c.params.slug)?.name || 'another LMS'} on adaptive learning, integrations, hosting, migration, accessibility, and pricing.`),
    changefreq: 'monthly', sitemap: true, priority: '0.7', interactive: false,
    jsonLd: comparisonPageGraph, parent: '/compare', cluster: 'comparison',
    enumerate: () => COMPETITORS.map(item => ({ path: `/compare/lextures-vs-${item.slug}`, params: { slug: item.slug }, lastmod: VERIFIED_AT })),
  },
  {
    path: '/alternatives/:slug',
    title: c => truncateTitle(`8 alternatives to ${getCompetitor(c.params.slug)?.name || 'an LMS'}`),
    description: c => truncateMetaDescription(`Compare eight alternatives to ${getCompetitor(c.params.slug)?.name || 'an LMS'} using transparent criteria for hosting, assessment, integrations, accessibility, and cost.`),
    changefreq: 'monthly', sitemap: true, priority: '0.7', interactive: false,
    jsonLd: comparisonPageGraph, parent: '/alternatives', cluster: 'comparison',
    enumerate: () => COMPETITORS.map(item => ({ path: `/alternatives/${item.slug}`, params: { slug: item.slug }, lastmod: VERIFIED_AT })),
  },
  {
    path: '/integrations/:slug',
    title: c => truncateTitle(`${getIntegration(c.params.slug)?.name || 'Integration'} with Lextures`),
    description: c => truncateMetaDescription(`See supported versions, capabilities, setup effort, limitations, and setup guidance for ${getIntegration(c.params.slug)?.name || 'this Lextures integration'}.`),
    changefreq: 'monthly', sitemap: true, priority: '0.7', interactive: false,
    jsonLd: integrationPageGraph, parent: '/integrations', cluster: 'integrations',
    enumerate: () => INTEGRATIONS.map(item => ({ path: `/integrations/${item.slug}`, params: { slug: item.slug }, lastmod: VERIFIED_AT })),
  },
  {
    path: '/resources/guides',
    title: 'Editorial guides — Lextures',
    description: 'Explore six evidence-led guide clusters about adaptive learning, assessment, grading, standards, LMS selection, and homeschool teaching.',
    changefreq: 'weekly', sitemap: true, priority: '0.7', interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Editorial guides'), parent: '/resources', hub: true,
    navGroup: 'resources',
  },
  {
    path: '/resources/research',
    title: 'Original education research — Lextures',
    description: 'Read ungated Lextures research with preregistered methods, privacy-safe aggregate data, reusable datasets, and visible corrections.',
    changefreq: 'monthly', sitemap: true, priority: '0.8', interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Original research'), parent: '/resources', hub: true,
    navGroup: 'resources', cluster: 'research',
  },
  {
    path: '/resources/research/methodology',
    title: 'Research methodology and privacy — Lextures',
    description: 'How Lextures preregisters analyses, excludes opted-out tenants, de-identifies data, suppresses small cells, and publishes corrections.',
    changefreq: 'yearly', sitemap: true, priority: '0.6', interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Research methodology'), parent: '/resources/research',
    cluster: 'research', relatedTo: ['/trust', '/resources/guides'],
  },
  {
    path: '/blog',
    title: 'Blog — Lextures',
    description:
      'Essays on adaptive learning, assessment design, AI in education, and building better learning systems.',
    changefreq: 'weekly',
    sitemap: true,
    priority: '0.7',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Blog'),
    parent: '/resources', hub: true, navGroup: 'resources',
  },
  {
    path: '/blog/:slug',
    title: c => {
      const post = blogPostMeta.find(p => p.slug === c.params.slug)
      return truncateTitle(post ? `${post.title} — Lextures` : 'Blog — Lextures')
    },
    description: c => {
      const post = blogPostMeta.find(p => p.slug === c.params.slug)
      return truncateMetaDescription(post?.description || 'Lextures blog')
    },
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.6',
    lastmodSource: 'content',
    interactive: false,
    jsonLd: blogPostGraph,
    parent: '/blog', cluster: 'learning-design',
    enumerate: () =>
      blogPostMeta.map(p => ({
        path: `/blog/${p.slug}`,
        title: truncateTitle(`${p.title} — Lextures`),
        description: truncateMetaDescription(p.description || p.title),
        lastmod: p.updated || p.date || undefined,
        params: { slug: p.slug },
        robots: p.noindex ? 'noindex,follow' : undefined,
      })),
  },
  {
    path: '/docs',
    title: 'Documentation — Lextures',
    description:
      'Guides for creating courses, navigating Lextures, self-hosting, and connecting Zapier or Make.com automations.',
    changefreq: 'weekly',
    sitemap: true,
    priority: '0.7',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Docs'),
    parent: '/', hub: true,
  },
  {
    path: '/docs/:category',
    title: c => truncateTitle(`${getHelpCategory(c.params.category)?.title || 'Help'} help — Lextures`),
    description: c => truncateMetaDescription(getHelpCategory(c.params.category)?.description || 'Lextures help center category.'),
    changefreq: 'weekly', sitemap: true, priority: '0.6', interactive: false,
    jsonLd: c => defaultPageGraph(c, getHelpCategory(c.params.category)?.title || 'Help'), parent: '/docs', hub: true,
    enumerate: () => HELP_CATEGORIES.map(category => ({ path: `/docs/${category.id}`, title: truncateTitle(`${category.title} help — Lextures`), description: truncateMetaDescription(category.description), params: { category: category.id } })),
  },
  {
    path: '/docs/:category/:slug',
    title: c => {
      const article = docArticleMeta.find(a => a.slug === c.params.slug && a.category === c.params.category)
      return truncateTitle(article ? `${article.title} — Lextures` : 'Docs — Lextures')
    },
    description: c => {
      const article = docArticleMeta.find(a => a.slug === c.params.slug && a.category === c.params.category)
      return truncateMetaDescription(article?.description || 'Lextures documentation')
    },
    changefreq: 'monthly',
    sitemap: true,
    priority: '0.6',
    lastmodSource: 'content',
    interactive: false,
    jsonLd: docsPostGraph,
    parent: '/docs/:category', cluster: 'help',
    enumerate: () =>
      docArticleMeta.map(a => ({
        path: `/docs/${a.category}/${a.slug}`,
        title: truncateTitle(`${a.title} — Lextures`),
        description: truncateMetaDescription(a.description || a.title),
        lastmod: a.updated || a.date || undefined,
        params: { category: a.category || '', slug: a.slug },
        robots: a.noindex ? 'noindex,follow' : undefined,
      })),
  },
  {
    path: '/internal/content-kit',
    title: 'Answer-first content kit — Lextures',
    description: 'Internal rendered reference for the Lextures answer-first content components and authoring contract.',
    sitemap: false,
    robots: 'noindex,follow',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'Answer-first content kit'),
  },
  {
    path: '/privacy',
    title: 'Privacy Policy — Lextures',
    description: 'How Lextures collects, uses, and protects personal data for learners, instructors, and institutions.',
    changefreq: 'yearly',
    sitemap: true,
    priority: '0.3',
    interactive: false,
    jsonLd: c => legalDocumentGraph(c, 'privacy'),
    parent: '/trust', navGroup: 'trust',
  },
  {
    path: '/privacy/history',
    title: 'Privacy Policy history — Lextures',
    description: 'Historical versions of the Lextures Privacy Policy.',
    changefreq: 'yearly',
    sitemap: true,
    priority: '0.2',
    robots: 'noindex,follow',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'History'),
    parent: '/privacy', navGroup: 'trust',
  },
  {
    path: '/terms',
    title: 'Terms of Service — Lextures',
    description: 'Terms governing use of Lextures software, hosted services, and marketplace courses.',
    changefreq: 'yearly',
    sitemap: true,
    priority: '0.3',
    interactive: false,
    jsonLd: c => legalDocumentGraph(c, 'terms'),
    parent: '/trust', navGroup: 'trust',
  },
  {
    path: '/terms/history',
    title: 'Terms of Service history — Lextures',
    description: 'Historical versions of the Lextures Terms of Service.',
    changefreq: 'yearly',
    sitemap: true,
    priority: '0.2',
    robots: 'noindex,follow',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'History'),
    parent: '/terms', navGroup: 'trust',
  },
  {
    path: '/security',
    title: 'Security — Lextures',
    description:
      'Lextures security practices and responsible disclosure policy for reporting vulnerabilities.',
    changefreq: 'yearly',
    sitemap: true,
    priority: '0.3',
    interactive: false,
    jsonLd: securityGraph,
    parent: '/trust', navGroup: 'trust',
  },
  {
    path: '/accessibility',
    title: 'Accessibility — Lextures',
    description:
      'Lextures accessibility conformance statement and commitment to WCAG 2.2 Level AA.',
    changefreq: 'yearly',
    sitemap: true,
    priority: '0.3',
    interactive: false,
    jsonLd: accessibilityGraph,
    parent: '/trust', navGroup: 'trust',
  },
  {
    path: '/accessibility/vpat',
    title: 'VPAT — Lextures',
    description:
      'Voluntary Product Accessibility Template (VPAT 2.5) for Lextures — accessibility conformance report.',
    changefreq: 'yearly',
    sitemap: true,
    priority: '0.3',
    interactive: false,
    jsonLd: accessibilityGraph,
    parent: '/accessibility', navGroup: 'trust',
  },
  {
    path: '/privacy-rights/california',
    title: 'California privacy rights — Lextures',
    description:
      'Your California privacy rights under the CCPA/CPRA when using Lextures products and services.',
    changefreq: 'yearly',
    sitemap: true,
    priority: '0.3',
    interactive: false,
    jsonLd: c => defaultPageGraph(c, 'California'),
    parent: '/privacy', navGroup: 'trust',
  },
  {
    path: '/404',
    title: 'Page not found — Lextures',
    description: 'The page you requested does not exist on lextures.com.',
    sitemap: false,
    robots: 'noindex,follow',
    interactive: false,
  },
]

/** Localized entries are added here only after human and compliance review. */
const LOCALIZED_ROUTE_MANIFEST: RouteDescriptor[] = []

export const ROUTE_MANIFEST: RouteDescriptor[] = [
  ...ENGLISH_ROUTE_MANIFEST.map(descriptor => ({ ...descriptor, locale: DEFAULT_LOCALE as LocaleCode })),
  ...LOCALIZED_ROUTE_MANIFEST,
]

export function resolveRouteDescriptor(
  pathname: string,
): { descriptor: RouteDescriptor; params: Record<string, string> } | null {
  const clean =
    pathname !== '/' && pathname.endsWith('/') ? pathname.replace(/\/+$/, '') : pathname || '/'

  return matchManifest(clean) ?? matchManifest(stripContentLocalePrefix(clean))
}

function stripContentLocalePrefix(pathname: string): string {
  const first = pathname.split('/').filter(Boolean)[0]
  if (!first || !isValidLocaleCode(first) || first.toLowerCase() === DEFAULT_LOCALE) return pathname
  const rest = pathname.slice(`/${first}`.length) || '/'
  if (rest === '/blog' || rest.startsWith('/blog/') || rest === '/docs' || rest.startsWith('/docs/')) return rest
  return pathname
}

function matchManifest(clean: string): { descriptor: RouteDescriptor; params: Record<string, string> } | null {
  for (const descriptor of ROUTE_MANIFEST) {
    if (!isPublishable(descriptor)) continue
    if (!descriptor.path.includes(':')) {
      if (descriptor.path === clean) {
        return { descriptor, params: {} }
      }
    }
  }
  for (const descriptor of ROUTE_MANIFEST) {
    if (!isPublishable(descriptor)) continue
    if (descriptor.path.includes(':')) {
      const params = matchPattern(descriptor.path, clean)
      if (params) return { descriptor, params }
    }
  }
  return null
}

export function isInteractiveRoute(pathname: string): boolean {
  const resolved = resolveRouteDescriptor(pathname)
  if (!resolved) return false
  return resolved.descriptor.interactive !== false
}

export function resolveTitle(descriptor: RouteDescriptor, renderCtx: RenderContext): string {
  const raw = typeof descriptor.title === 'function' ? descriptor.title(renderCtx) : descriptor.title
  return truncateTitle(raw)
}

export function resolveDescription(descriptor: RouteDescriptor, renderCtx: RenderContext): string {
  const raw =
    typeof descriptor.description === 'function'
      ? descriptor.description(renderCtx)
      : descriptor.description
  return truncateMetaDescription(raw)
}

export type ConcreteRoute = {
  path: string
  descriptor: RouteDescriptor
  params: Record<string, string>
  title: string
  description: string
  lastmod?: string
  robots: string
  canonical: string
}

/**
 * Expand the manifest into concrete paths (static + enumerated dynamic families).
 * Course detail paths are merged in by the generator after the API fetch.
 */
export function enumerateConcreteRoutes(
  extraCoursePaths: Array<{ path: string; title?: string; description?: string; lastmod?: string; robots?: string }> = [],
): ConcreteRoute[] {
  const out: ConcreteRoute[] = []
  const seen = new Set<string>()

  for (const descriptor of ROUTE_MANIFEST) {
    if (!isPublishable(descriptor)) continue
    if (descriptor.path.includes(':')) {
      const entries = descriptor.enumerate?.() ?? []
      for (const entry of entries) {
        if (seen.has(entry.path)) continue
        seen.add(entry.path)
        const params = entry.params ?? matchPattern(descriptor.path, entry.path) ?? {}
        const renderCtx = ctx(entry.path, params)
        out.push({
          path: entry.path,
          descriptor,
          params,
          title: entry.title ?? resolveTitle(descriptor, renderCtx),
          description: entry.description ?? resolveDescription(descriptor, renderCtx),
          lastmod: entry.lastmod,
          robots: descriptor.robots ?? 'index,follow',
          canonical: canonicalUrl(entry.path),
        })
      }
      continue
    }
    if (seen.has(descriptor.path)) continue
    seen.add(descriptor.path)
    const renderCtx = ctx(descriptor.path)
    out.push({
      path: descriptor.path,
      descriptor,
      params: {},
      title: resolveTitle(descriptor, renderCtx),
      description: resolveDescription(descriptor, renderCtx),
      robots: descriptor.robots ?? 'index,follow',
      canonical: canonicalUrl(descriptor.path),
    })
  }

  for (const entry of extraCoursePaths) {
    const resolved = resolveRouteDescriptor(entry.path)
    const courseDescriptor = resolved?.descriptor
    if (courseDescriptor) {
      if (seen.has(entry.path)) continue
      seen.add(entry.path)
      const params = matchPattern(courseDescriptor.path, entry.path) ?? {}
      const renderCtx = ctx(entry.path, params)
      out.push({
        path: entry.path,
        descriptor: courseDescriptor,
        params,
        title: entry.title ?? resolveTitle(courseDescriptor, renderCtx),
        description: entry.description ?? resolveDescription(courseDescriptor, renderCtx),
        lastmod: entry.lastmod,
        robots: entry.robots ?? courseDescriptor.robots ?? 'index,follow',
        canonical: canonicalUrl(entry.path),
      })
    }
  }

  return out
}

/** Build the reciprocal cluster for a concrete route; x-default is always English. */
export function hreflangAlternatesForPath(pathname: string): HreflangAlternate[] {
  const resolved = resolveRouteDescriptor(pathname)
  if (!resolved) return []
  const sourcePath = resolved.descriptor.translationOf ?? pathname
  const variants = ROUTE_MANIFEST.filter(descriptor =>
    isPublishable(descriptor) &&
    (descriptor.path === sourcePath || descriptor.translationOf === sourcePath) &&
    !descriptor.path.includes(':'),
  )
  if (variants.length < 2) return []
  const english = variants.find(v => v.locale === DEFAULT_LOCALE)
  if (!english) return []
  const alternates: HreflangAlternate[] = variants.map(variant => ({
    hreflang: variant.locale,
    href: canonicalUrl(variant.path),
  }))
  alternates.push({ hreflang: 'x-default', href: canonicalUrl(english.path) })
  return alternates
}

export function localeTargetsForPath(pathname: string): Array<{
  locale: LocaleCode
  name: string
  href: string
  exact: boolean
}> {
  const resolved = resolveRouteDescriptor(pathname)
  const sourcePath = resolved?.descriptor.translationOf ?? pathname
  return ROUTE_MANIFEST
    .filter(route => isPublishable(route) && !route.path.includes(':'))
    .filter((route, index, routes) => routes.findIndex(candidate => candidate.locale === route.locale) === index)
    .map(localeHome => {
      const exact = ROUTE_MANIFEST.find(route => isPublishable(route) && (
        (localeHome.locale === DEFAULT_LOCALE && route.path === sourcePath) ||
        (route.locale === localeHome.locale && route.translationOf === sourcePath)
      ))
      const locale = getLocale(localeHome.locale)
      return {
        locale: locale.code as LocaleCode,
        name: locale.name,
        href: exact?.path ?? localizedPath('/', locale.code),
        exact: Boolean(exact),
      }
    })
}

export function validateLocaleManifest(descriptors: RouteDescriptor[] = ROUTE_MANIFEST): string[] {
  const errors: string[] = []
  const byPath = new Map(descriptors.map(route => [route.path, route]))
  for (const route of descriptors) {
    if (!isWellFormedBcp47(route.locale) || !isValidLocaleCode(route.locale)) {
      errors.push(`${route.path}: invalid or unsupported BCP 47 locale "${route.locale}"`)
    }
    if (route.locale !== DEFAULT_LOCALE) {
      if (!route.translationOf) errors.push(`${route.path}: localized route is missing translationOf`)
      const source = route.translationOf ? byPath.get(route.translationOf) : undefined
      if (!source || source.locale !== DEFAULT_LOCALE) errors.push(`${route.path}: translationOf must reference an English route`)
      const expected = localizedPath(route.translationOf || '/', route.locale)
      if (route.path !== expected) errors.push(`${route.path}: locale URL must be ${expected}`)
      if (route.translationStatus === 'complete' && getLocale(route.locale).status !== 'published') {
        errors.push(`${route.path}: complete translation uses a locale that is not published`)
      }
    }
  }
  return errors
}

/** Load and render the page element for a path (SSR + client). */
export async function loadRouteElement(pathname: string): Promise<ReactElement> {
  const resolved = resolveRouteDescriptor(pathname)
  if (!resolved) {
    const NotFound = await getPageLoader('/404')!()
    return <NotFound />
  }
  const loader = getPageLoader(resolved.descriptor.translationOf ?? resolved.descriptor.path)
  if (!loader) {
    const NotFound = await getPageLoader('/404')!()
    return <NotFound />
  }
  const Page = (await loader()) as ComponentType<Record<string, unknown>>
  const props = pagePropsFor(resolved.descriptor.translationOf ?? resolved.descriptor.path, ctx(pathname, resolved.params))
  return <Page {...props} />
}

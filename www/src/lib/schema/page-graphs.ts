/**
 * Per-route JSON-LD composition used by the route manifest (SEO.3).
 */
import type { JsonLdNode } from '../document-head'
import { getAuthor, getActiveAuthors, isAuthorLinkable } from '../authors'

/** Minimal render context — avoids circular import with route-manifest. */
export type SchemaRenderContext = {
  path: string
  origin: string
  params: Record<string, string>
}
import { blogPostMeta, docArticleMeta } from '../../utils/content-meta'
import { LEGAL_VERSIONS } from '../legal-versions'
import { composePageGraph } from './graph'
import { buildArticle } from './article'
import { buildPerson } from './person'
import { buildFaqPage } from './faq'
import { buildHowTo } from './how-to'
import { buildPricingProduct } from './offer'
import { buildCourseItemList, normalizeServerCourseJsonLd, extendCourseNode } from './course'
import {
  buildAccessibilityGraph,
  buildDigitalDocument,
  buildSecurityWebPage,
} from './conformance'
import { PRICING_FAQS } from './pricing-faqs'
import type { PublicMarketplaceCourse, PublicMarketplaceCourseDetail } from '../marketplace-api'
import { getCompetitor, VERIFIED_AT } from '../competitors'
import { getIntegration } from '../integrations'
import { softwareApplicationId, absoluteUrl } from './ids'

export function siteOriginFromCtx(ctx: SchemaRenderContext): string {
  return (ctx.origin || 'https://lextures.com').replace(/\/$/, '')
}

/** Default graph for static marketing pages (site-wide + breadcrumb). */
export function defaultPageGraph(ctx: SchemaRenderContext, leafName?: string): JsonLdNode[] {
  return composePageGraph({
    path: ctx.path,
    leafName,
    siteOrigin: siteOriginFromCtx(ctx),
  })
}

export function homePageGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  // Homepage: site-wide only (no breadcrumb)
  return composePageGraph({
    path: '/',
    siteOrigin: siteOriginFromCtx(ctx),
  })
}

export function aboutPageGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  const people = getActiveAuthors()
    .map(a => buildPerson(a, origin))
    .filter((n): n is JsonLdNode => Boolean(n))
  return composePageGraph({
    path: '/about',
    leafName: 'About',
    siteOrigin: origin,
    pageNodes: people,
  })
}

export function authorsIndexGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  const people = getActiveAuthors()
    .map(a => buildPerson(a, origin))
    .filter((n): n is JsonLdNode => Boolean(n))
  return composePageGraph({
    path: '/authors',
    leafName: 'Authors',
    siteOrigin: origin,
    pageNodes: people,
  })
}

export function authorPageGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  const slug = ctx.params.slug ?? ''
  const author = getAuthor(slug)
  const pageNodes: JsonLdNode[] = []
  if (author && isAuthorLinkable(slug)) {
    const person = buildPerson(author, origin)
    if (person) pageNodes.push(person)
  }
  return composePageGraph({
    path: ctx.path,
    leafName: author?.name,
    siteOrigin: origin,
    pageNodes,
  })
}

export function blogPostGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  const slug = ctx.params.slug ?? ''
  const post = blogPostMeta.find(p => p.slug === slug)
  const pageNodes: JsonLdNode[] = []
  if (post) {
    const author = getAuthor(post.author)
    if (author && isAuthorLinkable(post.author)) {
      const person = buildPerson(author, origin)
      if (person) pageNodes.push(person)
    }
    if (post.reviewedBy) {
      const reviewer = getAuthor(post.reviewedBy)
      if (reviewer && isAuthorLinkable(post.reviewedBy)) {
        const person = buildPerson(reviewer, origin)
        if (person) pageNodes.push(person)
      }
    }
    pageNodes.push(
      buildArticle({
        path: `/blog/${post.slug}`,
        headline: post.title,
        description: post.description,
        datePublished: post.date,
        dateModified: post.updated || post.date,
        authorSlug: post.author,
        reviewedBySlug: post.reviewedBy,
        wordCount: post.wordCount,
        articleSection: 'Education',
        citations: post.citations,
        siteOrigin: origin,
      }),
    )
    const faq = buildFaqPage(`/blog/${post.slug}`, post.faq || [], origin)
    if (faq) pageNodes.push(faq)
  }
  return composePageGraph({
    path: ctx.path,
    leafName: post?.title,
    siteOrigin: origin,
    pageNodes,
  })
}

export function docsPostGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  const slug = ctx.params.slug ?? ''
  const article = docArticleMeta.find(a => a.slug === slug)
  const pageNodes: JsonLdNode[] = []
  if (article) {
    const author = getAuthor(article.author)
    if (author && isAuthorLinkable(article.author)) {
      const person = buildPerson(author, origin)
      if (person) pageNodes.push(person)
    }
    pageNodes.push(
      buildArticle({
        path: `/docs/${article.slug}`,
        headline: article.title,
        description: article.description,
        datePublished: article.date,
        dateModified: article.updated || article.date,
        authorSlug: article.author,
        tech: true,
        articleSection: 'Documentation',
        siteOrigin: origin,
      }),
    )
    const faq = buildFaqPage(`/docs/${article.slug}`, article.faq || [], origin)
    if (faq) pageNodes.push(faq)
    // Procedural create-course guide → HowTo for LLM comprehension
    if (article.slug === 'creating-a-new-course') {
      const howto = buildHowTo({
        path: `/docs/${article.slug}`,
        name: article.title,
        description: article.description,
        steps: [
          {
            name: 'Open the course wizard',
            text: 'Log in as an instructor or admin, open Courses, and click New Course.',
          },
          {
            name: 'Enter course details',
            text: 'Set the course title, code, and section settings in the creation wizard.',
          },
          {
            name: 'Save and publish',
            text: 'Review the summary, create the course, and open it to add modules and content.',
          },
        ],
        siteOrigin: origin,
      })
      if (howto) pageNodes.push(howto)
    }
  }
  return composePageGraph({
    path: ctx.path,
    leafName: article?.title,
    siteOrigin: origin,
    pageNodes,
  })
}

export function pricingPageGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  const faq = buildFaqPage('/pricing', PRICING_FAQS, origin)
  return composePageGraph({
    path: '/pricing',
    leafName: 'Pricing',
    siteOrigin: origin,
    pageNodes: [...buildPricingProduct(origin), ...(faq ? [faq] : [])],
  })
}

const comparisonFaqs = (name: string) => [
  { question: `Is Lextures better than ${name}?`, answer: `Neither is universally better. Evaluate each product against your audience, hosting, interoperability, accessibility, migration, and support requirements.` },
  { question: `Can we migrate from ${name}?`, answer: 'A staged migration may be possible, but portability depends on the source export. Test one representative course before committing to cost or timing.' },
  { question: 'Does Lextures support LTI 1.3?', answer: 'Lextures supports LTI 1.3 deployments. Available LTI Advantage services depend on configuration and should be verified in a pilot.' },
  { question: 'Can Lextures be self-hosted?', answer: 'Yes. Lextures is available under AGPL-3.0 for self-hosting; operating infrastructure and support remain the deployer’s responsibility.' },
]

export function comparisonPageGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx); const competitor = getCompetitor(ctx.params.slug || '')
  const name = competitor?.name || 'another LMS'; const faq = buildFaqPage(ctx.path, comparisonFaqs(name), origin)
  const author = getAuthor('chase-willden'); const person = author ? buildPerson(author, origin) : null
  return composePageGraph({ path: ctx.path, leafName: name, siteOrigin: origin, pageNodes: [...(person ? [person] : []), buildArticle({ path: ctx.path, headline: ctx.path.startsWith('/alternatives/') ? `Alternatives to ${name}` : `Lextures vs. ${name}`, description: `An evidence-led evaluation of Lextures and ${name}.`, datePublished: VERIFIED_AT, dateModified: VERIFIED_AT, authorSlug: 'chase-willden', articleSection: 'LMS evaluation', citations: competitor ? [competitor.docsUrl, competitor.pricingUrl] : [], siteOrigin: origin }), ...(faq ? [faq] : [])] })
}

export function integrationPageGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx); const item = getIntegration(ctx.params.slug || ''); const name = item?.name || 'Integration'
  const faqPairs = [{question:`Is ${name} a native Lextures integration?`,answer:item?.native?'Yes, within the documented versions and enabled platform features. Confirm the exact deployment in a pilot.':'No turnkey native connector is currently documented. Contact Lextures before planning a custom implementation.'},{question:'How long does setup take?',answer:`Expected effort: ${item?.effort || 'Confirm with Lextures'}. Timing depends on approvals, data mapping, testing, and the connected system.`},{question:'Should we test with production learner data?',answer:'No. Begin with synthetic or tightly limited data and expand only after permissions, mapping, errors, and rollback have been verified.'},{question:'Where are setup instructions?',answer:'The linked Lextures help article describes the closest supported setup path and operational checks.'}]
  const faq = buildFaqPage(ctx.path, faqPairs, origin); const howTo = buildHowTo({path:ctx.path,name:`Set up ${name} with Lextures`,description:item?.capability || 'Plan and validate the integration.',steps:[{name:'Confirm support',text:'Confirm the version, protocol, and capability meet the requirement.'},{name:'Run a pilot',text:'Configure minimum permissions and test with synthetic or limited data.'},{name:'Validate and launch',text:'Verify success, failures, monitoring, ownership, and rollback before production.'}],siteOrigin:origin})
  const related: JsonLdNode = {'@type':'SoftwareApplication','@id':`${absoluteUrl(ctx.path,origin)}#integration`,name:`Lextures integration with ${name}`,applicationCategory:'EducationalApplication',isRelatedTo:{'@type':'SoftwareApplication',name},supportingData:item?.helpPath ? absoluteUrl(item.helpPath,origin) : absoluteUrl('/docs',origin),provider:{'@id':softwareApplicationId(origin)}}
  return composePageGraph({path:ctx.path,leafName:name,siteOrigin:origin,pageNodes:[related,...(faq?[faq]:[]),...(howTo?[howTo]:[])]})
}

export function coursesIndexGraph(
  ctx: SchemaRenderContext,
  courses?: PublicMarketplaceCourse[],
): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  const list =
    courses && courses.length >= 3
      ? buildCourseItemList(
          courses.map(c => ({ slug: c.slug, title: c.title })),
          origin,
        )
      : null
  return composePageGraph({
    path: ctx.path,
    leafName: ctx.path === '/courses' ? 'Courses' : 'Course collection',
    siteOrigin: origin,
    pageNodes: list ? [list] : [],
  })
}

export function courseDetailGraph(
  ctx: SchemaRenderContext,
  detail?: PublicMarketplaceCourseDetail | null,
): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  const slug = ctx.params.slug || detail?.course.slug || ''
  const pageNodes: JsonLdNode[] = []
  if (detail?.jsonLd || slug) {
    let course = normalizeServerCourseJsonLd(
      (detail?.jsonLd as Record<string, unknown>) || {
        '@type': 'Course',
        name: detail?.course.title,
        description: detail?.course.description,
      },
      slug,
      origin,
    )
    if (course) {
      course = extendCourseNode(course, {
        educationalLevel: detail?.course.level,
        teaches: detail?.course.category,
      })
      pageNodes.push(course)
    }
  }
  return composePageGraph({
    path: ctx.path,
    leafName: detail?.course.title,
    siteOrigin: origin,
    pageNodes,
  })
}

export function accessibilityGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  return composePageGraph({
    path: ctx.path,
    leafName: ctx.path.includes('vpat') ? 'VPAT' : 'Accessibility',
    siteOrigin: origin,
    pageNodes: buildAccessibilityGraph({
      path: ctx.path,
      includeVpat: true,
      siteOrigin: origin,
    }),
  })
}

export function securityGraph(ctx: SchemaRenderContext): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  return composePageGraph({
    path: '/security',
    leafName: 'Security',
    siteOrigin: origin,
    pageNodes: [buildSecurityWebPage(origin)],
  })
}

export function legalDocumentGraph(
  ctx: SchemaRenderContext,
  kind: 'privacy' | 'terms',
): JsonLdNode[] {
  const origin = siteOriginFromCtx(ctx)
  const version =
    kind === 'privacy'
      ? LEGAL_VERSIONS.privacy_policy.version
      : LEGAL_VERSIONS.terms_of_service.version
  const name = kind === 'privacy' ? 'Privacy Policy' : 'Terms of Service'
  const path = kind === 'privacy' ? '/privacy' : '/terms'
  return composePageGraph({
    path: ctx.path,
    leafName: name,
    siteOrigin: origin,
    pageNodes: [
      buildDigitalDocument({
        path,
        name,
        dateModified: version,
        siteOrigin: origin,
      }),
    ],
  })
}

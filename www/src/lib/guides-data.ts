/**
 * Repository-native guides (`/guides/:slug`). Each guide carries a
 * `publishDate`; future-dated guides are left out of routes, the sitemap,
 * and internal listings until that date in America/Boise.
 * See www/docs/scheduled-publishing.md.
 */
import { filterPublished, publishToday } from './publish-date.ts'

export type GuideFaq = { question: string; answer: string }

export type GuideMeta = {
  slug: string
  path: string
  /** Visible H1. */
  headline: string
  /** <title>, ≤ 60 characters. */
  metaTitle: string
  /** Meta description, ≤ 155 characters. */
  description: string
  /** YYYY-MM-DD, America/Boise. The page goes live on this date. */
  publishDate: string
  /** Date the sourced facts were last re-checked. */
  verifiedOn: string
  authorSlug: string
  briefRef: string
  /** Short label for internal links. */
  linkLabel: string
  citations: string[]
  /** FAQ copy; rendered verbatim on the page and in FAQPage JSON-LD. */
  faqs: GuideFaq[]
}

export const GUIDES: GuideMeta[] = [
  {
    slug: 'canvas-lite-for-homeschool',
    path: '/guides/canvas-lite-for-homeschool',
    headline: 'Canvas Lite for Homeschool: What You Get, the Limits, and a Simpler Option',
    metaTitle: 'Canvas Lite for Homeschool: Features, Limits & Setup',
    description: "Canvas Lite is Instructure's free Canvas for homeschool: what's included, 5-course and 500 MB limits, ID verification, and when to pick something simpler.",
    publishDate: '2026-10-13',
    verifiedOn: '2026-10-12',
    authorSlug: 'chase-willden',
    briefRef: 'seo/briefs/2026-10-12-canvas-lite-homeschool.md',
    linkLabel: 'Canvas Lite for homeschool: features and limits',
    citations: [
      'https://www.instructure.com/try-canvas',
      'https://www.instructure.com/resources/blog/say-hello-canvas-lite',
      'https://www.instructure.com/policies/canvas-lite-terms-conditions',
      'https://support.google.com/edu/classroom/answer/7582372',
      'https://support.google.com/edu/classroom/answer/6025224',
    ],
    faqs: [
      { question: 'Is Canvas free for homeschool?', answer: "Yes. Instructure's Canvas Lite is free for individual educators, and its terms explicitly include homeschool instruction. Sign up at canvaslite.instructure.com and verify your identity with a government ID and a selfie. Your students don't need to verify." },
      { question: 'What are the limits of Canvas Lite?', answer: 'Up to 5 courses, 50 learners per course, and 500 MB of storage per course. Attendance tracking, course analytics, AI tools, admin features, and API access tokens require a full Canvas license.' },
      { question: 'Is Canvas Lite the same as Canvas Free for Teacher?', answer: 'Canvas Lite replaces Free for Teacher. It runs on the same Canvas platform, adds teacher identity verification, and lives on its own domain, separate from school Canvas sites.' },
      { question: 'Does Canvas Lite track attendance or make transcripts?', answer: "Instructure says attendance tracking requires a full Canvas license. Canvas Lite does include grade exports through SpeedGrader. Instructure doesn't list transcripts or report cards as Lite features." },
      { question: 'Canvas Lite or Google Classroom for homeschool?', answer: 'Canvas Lite has more structure (modules, rubrics, SpeedGrader) but caps courses and storage. Google Classroom works with personal accounts but limits under-13 use and guardian summaries. Families who need records may need another tool for either one.' },
    ],
  },
  {
    slug: 'lms-for-homeschool-co-ops',
    path: '/guides/lms-for-homeschool-co-ops',
    headline: 'Choosing an LMS for Your Homeschool Co-op (and When Canvas Is Too Much)',
    metaTitle: 'Homeschool Co-op LMS: Canvas Alternatives Compared',
    description: 'What a homeschool co-op needs from an LMS, what Canvas Lite allows, and how NavEd, CO-OP Link, CooplyHQ and Lextures compare on classes, grading and price.',
    publishDate: '2026-10-15',
    verifiedOn: '2026-10-12',
    authorSlug: 'chase-willden',
    briefRef: 'seo/briefs/2026-10-12-homeschool-coop-canvas-alternative.md',
    linkLabel: 'LMS options for homeschool co-ops',
    citations: [
      'https://www.instructure.com/policies/canvas-lite-terms-conditions',
      'https://www.instructure.com/resources/blog/say-hello-canvas-lite',
      'https://www.instructure.com/try-canvas',
      'https://www.instructure.com/canvas/pricing',
      'https://nav.education/',
      'https://coop-link.com/',
      'https://cooplyhq.com/',
      'https://support.google.com/edu/classroom/answer/7582372',
    ],
    faqs: [
      { question: 'Can a homeschool co-op use Canvas for free?', answer: 'Canvas Lite is free for individual educators in homeschool or "small-group, non-commercial" settings. It\'s capped at 5 courses and 50 learners per course, with no admin or attendance features. Co-ops that charge fees or need a central admin view should read Instructure\'s terms or consider a co-op-focused tool.' },
      { question: "What's the best LMS for a homeschool co-op?", answer: 'It depends on the job. CooplyHQ focuses on registration, fees, and volunteers. CO-OP Link adds class assignments and online grading. NavEd covers records, attendance, and transcripts. Lextures focuses on teaching classes with lessons, adaptive quizzes, and automatic grading.' },
      { question: 'How much does homeschool co-op software cost?', answer: 'As published on 2026-10-12: CooplyHQ is free for up to 15 families, then $149 per year. CO-OP Link is free for up to 25 students, then from $29 per month. NavEd is free for 5 students, then $99 per year for up to 15. Canvas Lite is free within its limits.' },
      { question: 'Can co-op teachers grade work online?', answer: 'Yes. CO-OP Link lets teachers annotate submitted PDFs and photos in the browser. NavEd and Lextures include gradebooks, and Lextures can grade quizzes automatically. Canvas Lite includes SpeedGrader with rubrics.' },
      { question: 'Do we need one tool or two?', answer: 'Many co-ops use one tool for operations (sign-ups, fees, calendar) and another for classes (assignments, grading). Pick a single platform only if it covers both jobs well for your size.' },
    ],
  },
]

/** Guides live on `today` (America/Boise), newest first. */
export function publishedGuides(today: string = publishToday()): GuideMeta[] {
  return filterPublished(GUIDES, today).sort((a, b) => b.publishDate.localeCompare(a.publishDate))
}

export function getPublishedGuide(slug: string, today: string = publishToday()): GuideMeta | undefined {
  return publishedGuides(today).find(g => g.slug === slug)
}

export function isGuideLive(path: string, today: string = publishToday()): boolean {
  return publishedGuides(today).some(g => g.path === path)
}

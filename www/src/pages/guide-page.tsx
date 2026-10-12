import type { ReactNode } from 'react'
import { AnswerBox, ComparisonTable, FAQ } from '../components/content'
import { AudienceCta, MarketingPageShell } from '../components/marketing-page-shell'
import { RelatedContent } from '../components/related-content'
import { getAuthor } from '../lib/authors'
import { getPublishedGuide, isGuideLive, type GuideMeta } from '../lib/guides-data'
import { SITE_LINKS } from '../lib/site-links'
import { NotFoundPage } from './not-found-page'

const TERMS = 'https://www.instructure.com/policies/canvas-lite-terms-conditions'
const LITE_BLOG = 'https://www.instructure.com/resources/blog/say-hello-canvas-lite'
const TRY_CANVAS = 'https://www.instructure.com/try-canvas'
const GC_PERSONAL = 'https://support.google.com/edu/classroom/answer/7582372'
const GC_AGE = 'https://support.google.com/edu/classroom/answer/6025224'

function longDate(iso: string) {
  return new Date(`${iso}T12:00:00Z`).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric', timeZone: 'UTC' })
}

function Src({ href, children }: { href: string; children: ReactNode }) {
  return <a href={href} rel="noopener noreferrer">{children}</a>
}

function GuideShell({ guide, eyebrow, lead, children }: { guide: GuideMeta; eyebrow: string; lead: string; children: ReactNode }) {
  const author = getAuthor(guide.authorSlug)
  return <MarketingPageShell>
    <header className="mx-auto max-w-[960px] px-5 pt-16 md:px-10 xl:px-14">
      <p className="text-sm font-semibold uppercase tracking-wide" style={{ color: '#4fa894' }}>{eyebrow}</p>
      <h1 className="font-display mt-4 font-semibold leading-[1.08] tracking-[-0.02em] text-balance" style={{ color: '#22333b', fontSize: 'clamp(30px,4vw,46px)' }}>{guide.headline}</h1>
      <p className="mt-5 max-w-[680px] text-[18px] leading-[1.6]" style={{ color: '#4a5b5d' }}>{lead}</p>
      <p className="mt-4 text-sm" style={{ color: 'var(--text-soft)' }}>
        By {author ? <a href={`/authors/${author.slug}`}>{author.name}</a> : 'Lextures'} · Published <time dateTime={guide.publishDate}>{longDate(guide.publishDate)}</time> · Verified on <time dateTime={guide.verifiedOn}>{longDate(guide.verifiedOn)}</time>
      </p>
    </header>
    <article className="comparison-article">{children}
      <FAQ items={guide.faqs} />
      <section className="content-sources"><h2>Sources</h2><ul>{guide.citations.map(url => <li key={url}><Src href={url}>{url}</Src> (accessed {guide.verifiedOn})</li>)}</ul></section>
      <RelatedContent path={guide.path} />
    </article>
  </MarketingPageShell>
}

function CanvasLiteGuide({ guide }: { guide: GuideMeta }) {
  const coopLive = isGuideLive('/guides/lms-for-homeschool-co-ops')
  return <GuideShell guide={guide} eyebrow="Homeschool guide" lead="What Instructure's free Canvas Lite includes, what it caps, and how to tell whether it's enough for your family, sourced to Instructure's own pages.">
    <AnswerBox><p>Yes. Canvas Lite is free for individual educators doing homeschool instruction. It includes modules, assignments, quizzes, rubrics, and SpeedGrader. It's capped at 5 courses, 50 learners per course, and 500 MB per course, and the teacher must verify with a government ID and a selfie.</p></AnswerBox>

    <section><h2>What is Canvas Lite?</h2>
      <p>Canvas Lite is Instructure's successor to Canvas Free for Teacher. It runs on its own domain, canvaslite.instructure.com, separate from school Canvas sites, and went live worldwide on September 30, 2026 (<Src href={LITE_BLOG}>Instructure, "Say hello to Canvas Lite"</Src>).</p>
      <p>Instructure's <Src href={TERMS}>Canvas Lite terms</Src> name "individual educators engaged in homeschool instruction or other small-group, non-commercial educational settings" among the eligible users.</p>
    </section>

    <section><h2>Can homeschoolers use Canvas Lite? (eligibility and signup)</h2>
      <h3>Identity verification</h3>
      <p>The teacher verifies their identity with a government ID and a selfie through Veriff. Students aren't verified (<Src href={LITE_BLOG}>Instructure</Src>).</p>
      <h3>Student accounts</h3>
      <p>Under the terms, the teacher is responsible for every linked account, and linked accounts may only go to bona fide participants, such as students and parents (<Src href={TERMS}>Canvas Lite terms</Src>).</p>
      <h3>Non-commercial only</h3>
      <p>The terms describe eligible settings as "small-group, non-commercial." If you teach in a co-op that charges fees, read the <Src href={TERMS}>terms</Src> yourself; this guide isn't legal advice, and Instructure decides eligibility.{coopLive && <> For co-op-specific options, see our guide to <a href="/guides/lms-for-homeschool-co-ops">choosing an LMS for a homeschool co-op</a>.</>}</p>
    </section>

    <section><h2>What's included</h2>
      <p>Per Instructure's <Src href={LITE_BLOG}>launch post</Src> and <Src href={TRY_CANVAS}>try page</Src>, Canvas Lite includes:</p>
      <ul>
        <li>Modules and pages</li>
        <li>File, text, and URL submissions</li>
        <li>Quizzes and rubrics</li>
        <li>SpeedGrader, with annotations, audio and video feedback, and grade exports</li>
        <li>Messaging, discussions, and announcements</li>
        <li>Google Drive, Zoom, and Microsoft Teams integrations</li>
        <li>The Canvas Teacher and Canvas Student mobile apps</li>
      </ul>
    </section>

    <section><h2>The limits that matter for a homeschool family</h2>
      <ComparisonTable summary="Canvas Lite limits and features that need a full Canvas license, per Instructure.">
        <table><caption>Canvas Lite limits, verified {guide.verifiedOn} (<Src href={LITE_BLOG}>Instructure</Src>)</caption>
          <thead><tr><th scope="col">Item</th><th scope="col">Canvas Lite</th></tr></thead>
          <tbody>
            <tr><th scope="row">Courses</th><td>5</td></tr>
            <tr><th scope="row">Learners per course</th><td>50</td></tr>
            <tr><th scope="row">Storage per course</th><td>500 MB</td></tr>
            <tr><th scope="row">Attendance tracking</th><td>Not included (needs a full license)</td></tr>
            <tr><th scope="row">Course analytics</th><td>Not included</td></tr>
            <tr><th scope="row">AI tools</th><td>Not included</td></tr>
            <tr><th scope="row">API access tokens</th><td>Not included</td></tr>
            <tr><th scope="row">Admin features</th><td>Not included</td></tr>
          </tbody>
        </table>
      </ComparisonTable>
      <h3>What 5 courses means in practice</h3>
      <p>Example math only: if one high schooler takes 6 subjects and each subject is its own course, that's already over the 5-course cap. How you split subjects into courses is up to you.</p>
      <h3>500 MB per course</h3>
      <p>Uploaded video fills 500 MB quickly. Linking out to video hosted elsewhere, instead of uploading it, keeps a course under the cap.</p>
      <h3>Records</h3>
      <p>Attendance tracking isn't included in Lite. Instructure's Lite pages don't mention transcripts or report cards, although grade export is available through SpeedGrader. If you need formal records, check what Instructure currently lists before relying on Lite for them.</p>
    </section>

    <section><h2>Canvas Lite vs Google Classroom for homeschool</h2>
      <p>Google Classroom is available to individuals over 13 with personal Google accounts (<Src href={GC_AGE}>Google</Src>). Children with parent-managed accounts can't create or teach classes, and teachers using personal accounts can't invite guardians to email summaries (<Src href={GC_PERSONAL}>Google</Src>).</p>
      <p>Canvas Lite has more course structure (modules, rubrics, SpeedGrader) but caps courses and storage. A fuller three-way comparison is coming in a later guide.</p>
    </section>

    <section><h2>When Canvas Lite is enough, and when it isn't</h2>
      <p><strong>Likely enough:</strong> one or two students, a few courses, and parents who want their kids to get used to Canvas before college. Instructure makes that familiarity point itself on its <Src href={TRY_CANVAS}>try page</Src>.</p>
      <p><strong>Likely not enough:</strong> more than 5 courses, attendance or state record keeping, a multi-child family that also teaches co-op classes, or video-heavy courses.</p>
    </section>

    <AudienceCta title="Try Lextures free for your homeschool" body="Self-hosting Lextures is free under AGPL-3.0. The hosted version at self.lextures.com has optional paid tiers." primaryHref={SITE_LINKS.homeschool} primaryLabel="Try Lextures free for your homeschool" secondaryHref="/compare/lextures-vs-canvas" secondaryLabel="See how Lextures compares to Canvas" />

    <section><h2>A simpler option: Lextures for homeschool</h2>
      <p><a href="/homeschool">Lextures for homeschool</a> is built for self-paced study without a classroom:</p>
      <ul>
        <li><a href="/platform/adaptive-learning">Adaptive quizzes</a> routed by Item Response Theory (IRT)</li>
        <li>Spaced-repetition review on web, iOS, and Android</li>
        <li>Self-paced courses, with a <a href="/platform/grading">gradebook</a> and progress dashboard</li>
        <li>QTI import for existing question banks</li>
        <li>Optional AI question generation using your own provider keys</li>
      </ul>
      <p>Self-hosting is free under AGPL-3.0. The hosted version at self.lextures.com has optional paid tiers; see <a href="/pricing">Lextures pricing</a>. For an institution-level view, read <a href="/compare/lextures-vs-canvas">Lextures vs. Canvas</a> or browse <a href="/alternatives/canvas">all Canvas alternatives</a>.</p>
    </section>

    <AudienceCta title="Try Lextures free for your homeschool" body="Self-host free under AGPL-3.0, or use self.lextures.com with optional paid tiers." primaryHref={SITE_LINKS.homeschool} primaryLabel="Try Lextures free for your homeschool" secondaryHref="/compare/lextures-vs-canvas" secondaryLabel="See how Lextures compares to Canvas" />
  </GuideShell>
}

function CoopGuide({ guide }: { guide: GuideMeta }) {
  const liteLive = isGuideLive('/guides/canvas-lite-for-homeschool')
  return <GuideShell guide={guide} eyebrow="Homeschool co-op guide" lead="Co-op software does two jobs: running the co-op and running the classes. Here's what Canvas Lite allows, and how co-op tools compare on classes, grading, and published price.">
    <AnswerBox><p>Most homeschool co-ops don't need full Canvas. Its free tier, Canvas Lite, is limited to individual educators in "small-group, non-commercial" settings and capped at 5 courses. Choose a co-op tool by job: registration and fees (CooplyHQ), communication plus simple grading (CO-OP Link), records and transcripts (NavEd), or teaching and grading classes (Lextures).</p></AnswerBox>

    <section><h2>Two jobs co-op software does</h2>
      <ul>
        <li><strong>Running the co-op:</strong> families, registration, waitlists, fees, volunteers, calendar, and messaging.</li>
        <li><strong>Running the classes:</strong> lessons, assignments, quizzes, grading, the gradebook, progress reports, and transcripts.</li>
      </ul>
      <p>A quick self-test: if your pain is the sign-up sheet and Venmo, start with job 1. If it's grading and "what's my kid's grade?", start with job 2.</p>
    </section>

    <section><h2>Can a homeschool co-op use Canvas?</h2>
      <h3>Canvas Lite eligibility</h3>
      <p>The <Src href={TERMS}>Canvas Lite terms</Src> cover "individual educators engaged in homeschool instruction or other small-group, non-commercial educational settings." Employees or agents of for-profit businesses, not-for-profits, or government agencies acting in a professional capacity are excluded unless they qualify individually. What that means for a co-op that charges fees depends on your situation; read the terms, since this guide isn't legal advice and Instructure decides eligibility.</p>
      <h3>Caps</h3>
      <p>Canvas Lite allows 5 courses, 50 learners per course, and 500 MB per course. Attendance, admin features, AI tools, course analytics, and API tokens need a full license (<Src href={LITE_BLOG}>Instructure</Src>). Because Lite is offered to individual educators and doesn't include admin features, a co-op with many parent-teachers would likely end up with separate teacher accounts and no shared admin view. That's our inference from Instructure's description, not a statement from Instructure; confirm it with them.</p>
      <h3>A full Canvas license</h3>
      <p>Instructure doesn't publish full Canvas pricing; you request a quote via its <Src href="https://www.instructure.com/canvas/pricing">pricing page</Src>.{liteLive && <> For the complete rundown, see our guide to <a href="/guides/canvas-lite-for-homeschool">Canvas Lite for homeschool</a>.</>}</p>
    </section>

    <section><h2>What a co-op LMS should have (checklist)</h2>
      <ul>
        <li>Parent-teacher dual roles (a parent who teaches one class and has kids in others)</li>
        <li>Per-class teacher access</li>
        <li>A parent view limited to their own kids</li>
        <li>A quick assignment and grading flow</li>
        <li>A gradebook and progress reports</li>
        <li>Class rosters</li>
        <li>Transcripts for high schoolers</li>
        <li>Mobile access</li>
        <li>Data export</li>
        <li>Simple pricing that doesn't charge per family</li>
      </ul>
    </section>

    <section><h2>Homeschool co-op LMS options compared</h2>
      <ComparisonTable summary="Co-op tools differ mainly in which job they're built for. Prices are as each vendor published them.">
        <table><caption>Homeschool co-op tools, prices verified {guide.verifiedOn}</caption>
          <thead><tr><th scope="col">Tool</th><th scope="col">Main job</th><th scope="col">Grading / gradebook</th><th scope="col">Free tier</th><th scope="col">Paid pricing (as published)</th><th scope="col">Mobile</th><th scope="col">Source</th></tr></thead>
          <tbody>
            <tr><th scope="row">Canvas Lite</th><td>Classes</td><td>Quizzes, rubrics, SpeedGrader</td><td>Free: 5 courses, 50 learners/course, 500 MB/course</td><td>Full license: pricing not public</td><td>Canvas Teacher and Student apps</td><td><Src href={LITE_BLOG}>Instructure</Src></td></tr>
            <tr><th scope="row">NavEd</th><td>Records/SIS plus LMS</td><td>Gradebook, report cards, transcripts/GPA</td><td>First 5 students free, forever</td><td>Annual: up to 15 students $99 · up to 40 $299 · up to 100 $699 · up to 250 $2,499 (homepage, 2026-10-12)</td><td>Not listed in our sources</td><td><Src href="https://nav.education/">nav.education</Src></td></tr>
            <tr><th scope="row">CO-OP Link</th><td>Co-op management plus classes</td><td>File submissions, PDF/photo annotation grading, grading queue</td><td>Up to 25 students / 3 classes</td><td>Essentials $29/mo · Growth $79/mo · Pro $149/mo · Enterprise custom</td><td>iOS, Android, web</td><td><Src href="https://coop-link.com/">coop-link.com</Src></td></tr>
            <tr><th scope="row">CooplyHQ</th><td>Co-op operations</td><td>Not listed on its site</td><td>Up to 15 families</td><td>Co-op $149/yr · Co-op Plus $349/yr</td><td>Phone browser (no app-store app yet)</td><td><Src href="https://cooplyhq.com/">cooplyhq.com</Src></td></tr>
            <tr><th scope="row">Google Classroom</th><td>Classes</td><td>Not covered here</td><td>Free with personal Google accounts</td><td>Not covered here</td><td>Not covered here</td><td><Src href={GC_PERSONAL}>Google</Src></td></tr>
            <tr><th scope="row">Lextures</th><td>Classes</td><td>Gradebook, progress dashboard, auto-graded adaptive quizzes</td><td>Self-hosting free (AGPL-3.0)</td><td>Hosted at self.lextures.com with optional paid tiers; see <a href="/pricing">pricing</a></td><td>Web, iOS, Android</td><td><a href="/homeschool">Lextures homeschool</a></td></tr>
          </tbody>
        </table>
      </ComparisonTable>

      <h3>NavEd</h3>
      <p>NavEd describes itself as built for homeschool co-ops, Classical Conversations communities, microschools, and university-model and hybrid schools. It covers records and the classroom: gradebook and report cards, transcripts and GPA, attendance, assignments, lessons, forms and e-signatures, enrollment, calendar, and Stripe payments. Records, transcripts, and attendance are free for the first 5 students; curriculum and payments need a paid plan. Prices above are as listed on the homepage on 2026-10-12 (<Src href="https://nav.education/">NavEd</Src>).</p>
      <h3>CO-OP Link</h3>
      <p>CO-OP Link combines co-op management with classes: messaging, classes and assignments, a shared calendar, volunteer sign-ups, rosters and registration, file submissions, PDF and photo annotation grading, a cross-class grading queue, a general ledger, and PDF reports, on iOS, Android, and web. It's free up to 25 students and 3 classes; Essentials covers 50 students and 10 classes, Growth 125, and Pro 250. AI Assignment Assist comes with Growth and Pro (<Src href="https://coop-link.com/">CO-OP Link</Src>).</p>
      <h3>CooplyHQ</h3>
      <p>CooplyHQ focuses on co-op operations: families, classes and waitlists, calendar and events, volunteers, email and a forum, payments (Stripe's fee only), and a co-op website. It advertises "no per-family fees," imports from Homeschool-Life, and works in a phone browser. Co-op Plus adds online payments and your own domain, and a Founding Co-op plan is $149/yr locked for 3 years (25 spots). Grading and assignments aren't listed on its site (<Src href="https://cooplyhq.com/">CooplyHQ</Src>).</p>
      <h3>Google Classroom</h3>
      <p>Google Classroom is free with personal Google accounts. Children with parent-managed accounts can't create or teach classes, and teachers on personal accounts can't invite guardians to email summaries (<Src href={GC_PERSONAL}>Google</Src>).</p>
      <h3>Lextures</h3>
      <p>See the next section.</p>
    </section>

    <section><h2>Where Lextures fits for co-op classes</h2>
      <p>Lextures is for the class side of a co-op:</p>
      <ul>
        <li>Courses with modules and question banks</li>
        <li><a href="/platform/adaptive-learning">Adaptive quizzes</a> routed by Item Response Theory (IRT), graded automatically</li>
        <li>Spaced-repetition review on web, iOS, and Android</li>
        <li>A <a href="/platform/grading">gradebook</a> and progress dashboard</li>
        <li>QTI import, plus <a href="/platform/assessment">question banks for assessment</a></li>
        <li>Optional AI question generation using your own provider keys</li>
      </ul>
      <p>Self-hosting is free under AGPL-3.0; the hosted version at self.lextures.com has optional paid tiers (<a href="/pricing">pricing</a>). Pair Lextures with a co-op ops tool if you need registration and fee collection.</p>
    </section>

    <AudienceCta title="Set up your co-op's first class in Lextures" body="Self-host free under AGPL-3.0, or use self.lextures.com with optional paid tiers." primaryHref={SITE_LINKS.homeschool} primaryLabel="Set up your co-op's first class in Lextures" secondaryHref="/compare/lextures-vs-canvas" secondaryLabel="Compare Lextures and Canvas" />

    <section><h2>How to choose, by co-op type</h2>
      <ul>
        <li><strong>Small co-op with one or two academic classes:</strong> Canvas Lite (if you're eligible) or Lextures.</li>
        <li><strong>Co-op where operations are the pain:</strong> CooplyHQ or CO-OP Link, plus a class tool if needed.</li>
        <li><strong>Co-op issuing high school transcripts:</strong> NavEd.</li>
        <li><strong>Academic co-op that wants quizzes and auto-grading:</strong> Lextures.</li>
      </ul>
      <p>Looking for the broader institutional list? See <a href="/alternatives/canvas">all Canvas alternatives</a>.</p>
    </section>

    <AudienceCta title="Set up your co-op's first class in Lextures" body="Self-host free under AGPL-3.0, or use self.lextures.com with optional paid tiers." primaryHref={SITE_LINKS.homeschool} primaryLabel="Set up your co-op's first class in Lextures" secondaryHref="/compare/lextures-vs-canvas" secondaryLabel="Compare Lextures and Canvas" />
  </GuideShell>
}

const BODIES: Record<string, (props: { guide: GuideMeta }) => ReactNode> = {
  'canvas-lite-for-homeschool': CanvasLiteGuide,
  'lms-for-homeschool-co-ops': CoopGuide,
}

export function GuidePage({ slug }: { slug: string }) {
  const guide = getPublishedGuide(slug)
  const Body = guide ? BODIES[guide.slug] : undefined
  if (!guide || !Body) return <NotFoundPage />
  return <Body guide={guide} />
}

import {
  AudienceCta,
  AudienceHero,
  CardGrid,
  MarketingPageShell,
} from '../components/marketing-page-shell'
import { SITE_LINKS } from '../lib/site-links'

const FEATURES = [
  {
    title: 'Adaptive quizzes without an instructor',
    body: 'Item Response Theory estimates your ability after each response and routes you to the next question that will teach the system the most — not whatever comes next in a fixed list.',
  },
  {
    title: 'Spaced repetition review',
    body: 'Review sessions queue items due today across your courses. Grade each recall (Again / Hard / Good / Easy) and the scheduler sets the next interval. Available on web, iOS, and Android.',
  },
  {
    title: 'Self-paced courses',
    body: 'Enroll in courses marked self-paced and work on your schedule. What-if grade projection shows how pending assignments would affect your standing.',
  },
  {
    title: 'Build from your own material',
    body: 'Create a course, add modules and question banks, or import QTI packages. Optional AI-assisted question generation when your instance has an OpenRouter key configured.',
  },
]

const USE_CASES = [
  {
    title: 'Professional certification',
    body: 'High-stakes exams where covering the syllabus once is not enough. Adaptive practice targets weak areas; spaced review keeps them retained under pressure.',
  },
  {
    title: 'Language learning',
    body: 'Vocabulary and grammar fade without structured retrieval practice. The review scheduler applies evidence-backed spacing automatically.',
  },
  {
    title: 'Independent study',
    body: 'Working through a textbook or online curriculum on your own timeline — with a gradebook and progress dashboard instead of a reading list alone.',
  },
]

export function SelfLearnerPage() {
  return (
    <MarketingPageShell>
      <AudienceHero
        eyebrow="Self-learner"
        title="An adaptive study system that runs without a classroom"
        lead="Create or enroll in courses, practice with IRT-routed quizzes, and clear spaced-repetition reviews from your phone. Self-host the full stack for free, or sign up at self.lextures.com with optional paid tiers."
        primaryHref={SITE_LINKS.selfLearner}
        primaryLabel="Start studying"
        secondaryHref="/pricing"
        secondaryLabel="Hosted pricing"
      />

      <CardGrid heading="How it works" items={FEATURES} columns={2} />

      <section className="border-b py-16 md:py-20" style={{ borderColor: 'var(--line)' }}>
        <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
          <h2 className="font-display text-[clamp(26px,3vw,34px)] font-semibold leading-tight" style={{ color: 'var(--ink)' }}>
            What a study session looks like
          </h2>
          <p className="mt-4 max-w-[640px] text-[16px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
            Open the dashboard. Review items due today — concepts approaching the edge of your
            predicted recall window. Answer adaptive quiz questions calibrated to your current level.
            After each session, ability estimates update and the next review dates shift forward or
            closer based on how you performed.
          </p>
        </div>
      </section>

      <CardGrid heading="Common use cases" items={USE_CASES} columns={3} />

      <section className="border-t py-12" style={{ borderColor: 'var(--line)' }}>
        <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
          <p className="text-[15px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
            <strong style={{ color: 'var(--ink-nav)' }}>Self-host:</strong> clone the repo and run
            without course or student limits.{' '}
            <strong style={{ color: 'var(--ink-nav)' }}>Hosted:</strong> free tier limits apply
            on self.lextures.com; see pricing for details. Public course catalog and paid enrollment
            require your administrator to enable those platform features.
          </p>
        </div>
      </section>

      <AudienceCta
        title="Try it on self.lextures.com or your own server"
        body="Sign up on self.lextures.com to start immediately, or follow the self-hosting guide to run Lextures on your hardware with full control."
        primaryHref={SITE_LINKS.selfLearner}
        primaryLabel="Open self.lextures.com"
        secondaryHref={SITE_LINKS.github}
        secondaryLabel="View on GitHub"
      />
    </MarketingPageShell>
  )
}

import {
  AudienceCta,
  AudienceHero,
  CardGrid,
  MarketingPageShell,
  TagStrip,
} from '../components/marketing-page-shell'
import { SITE_LINKS } from '../lib/site-links'

export function ParentsPage() {
  return (
    <MarketingPageShell>
      <AudienceHero
        eyebrow="Parents & guardians"
        title="See your child's grades and due dates in one place"
        lead="When your district enables the parent portal, you get read-only access to each linked child's courses, assignment scores, and upcoming work — without logging in as your student."
        primaryHref="/get-started"
        primaryLabel="Try it free"
        secondaryHref="/k-12"
        secondaryLabel="K–12 overview"
      />

      <CardGrid
        heading="What the parent portal actually shows"
        subheading="Parents cannot change grades, submit work, or edit course content. Access is scoped to children linked to your account."
        items={[
          {
            title: 'Multi-child dashboard',
            body: 'Switch between linked children from one account. Each child\'s courses, grades, and due dates stay separate.',
          },
          {
            title: 'Grades and missing work',
            body: 'View posted scores and assignments past due. Optional email alerts can notify you when grades post or work is overdue.',
          },
          {
            title: 'FERPA-aware access',
            body: 'Parent links are provisioned by the school. When a student reaches age 18, parent access to their records can be revoked automatically.',
          },
        ]}
      />

      <section className="border-b py-16 md:py-20" style={{ borderColor: 'var(--line)' }}>
        <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
          <h2 className="font-display text-[clamp(26px,3vw,34px)] font-semibold leading-tight" style={{ color: 'var(--ink)' }}>
            How schools turn it on
          </h2>
          <p className="mt-4 max-w-[640px] text-[16px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
            The parent portal is a platform feature your district administrator enables. Staff link
            parent accounts to students individually or through bulk CSV import. Mobile apps for iOS
            and Android include a parent context when your school configures it.
          </p>
        </div>
      </section>

      <TagStrip
        label="Related capabilities"
        tags={[
          'Read-only grade access',
          'Weekly summary view',
          'Bulk parent–student linking',
          'iOS & Android parent context',
          'Attendance summary (when enabled)',
        ]}
      />

      <AudienceCta
        title="Ask your school if Lextures is in use"
        body="Parents access Lextures through their district or school — not a separate consumer signup. If your school runs Lextures, they can provision your parent account and link it to your children."
        primaryHref={SITE_LINKS.selfLearner}
        primaryLabel="See a sample parent view"
        secondaryHref="/docs"
        secondaryLabel="Read the docs"
      />
    </MarketingPageShell>
  )
}

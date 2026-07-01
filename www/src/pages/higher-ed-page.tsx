import {
  AudienceCta,
  AudienceHero,
  CardGrid,
  MarketingPageShell,
  TagStrip,
} from '../components/marketing-page-shell'
import { SITE_LINKS } from '../lib/site-links'

const ADMIN_FEATURES = [
  {
    title: 'Enterprise identity',
    body: 'SAML 2.0 and OIDC single sign-on through your IdP. SCIM 2.0 provisions users and groups; Clever and ClassLink are available for K–12-style identity when you need them.',
  },
  {
    title: 'Enrollment state machine',
    body: 'Track active, dropped, withdrawn, audit, and incomplete enrollments with a history row for every transition — including deadline enforcement from your academic calendar.',
  },
  {
    title: 'Grade audit trail',
    body: 'Every grade change records the actor, prior value, new value, and reason. Export structured grade data when accreditation or appeals need evidence.',
  },
]

const DEAN_FEATURES = [
  {
    title: 'Course blueprints',
    body: 'Maintain a master course and push updates to child sections: syllabus modules, rubrics, and question bank items sync without each instructor copying files manually.',
  },
  {
    title: 'Standards and outcomes',
    body: 'Map assignments to learning outcomes and export mastery data. Departments see whether sections hit the same objectives, not just average scores.',
  },
  {
    title: 'LTI inside your existing LMS',
    body: 'Run Lextures as an LTI 1.3 tool in Canvas, Moodle, or Blackboard. Grades pass back via AGS; students stay in the LMS they already use.',
  },
]

const INSTRUCTOR_FEATURES = [
  {
    title: 'IRT adaptive quizzing',
    body: 'Item Response Theory routes each student to questions matched to their ability. Enable adaptive delivery per course when a section needs differentiated assessment.',
  },
  {
    title: 'Misconception detection',
    body: 'When many students pick the same wrong answer, instructors see the pattern in the dashboard — a signal for the next lecture, not a surprise on the final.',
  },
  {
    title: 'Canvas import and QTI',
    body: 'Import courses and question banks from Canvas. Bring QTI 2.1 or 3.0 packages from other systems when you migrate content.',
  },
]

export function HigherEdPage() {
  return (
    <MarketingPageShell>
      <AudienceHero
        eyebrow="Higher education"
        title="Assessment and records that hold up under review"
        lead="For registrars, deans, and faculty running multi-section courses: adaptive quizzing, LTI integration with major LMS platforms, enrollment states, and grade audit logs — self-hosted on Postgres or on the hosted demo."
        primaryHref="/get-started"
        primaryLabel="Try the demo"
        secondaryHref={SITE_LINKS.github}
        secondaryLabel="Browse the source"
      />

      <CardGrid
        heading="For administrators and registrars"
        items={ADMIN_FEATURES}
      />

      <CardGrid
        heading="For deans and department chairs"
        items={DEAN_FEATURES}
        columns={3}
      />

      <CardGrid
        heading="For instructors"
        items={INSTRUCTOR_FEATURES}
        columns={3}
      />

      <TagStrip
        label="Standards and protocols"
        tags={[
          'LTI 1.3 provider & consumer',
          'SAML 2.0 · OIDC',
          'SCIM 2.0',
          'AGS grade passback',
          'QTI 2.1 / 3.0',
          'Canvas import',
          'Incomplete grade workflow',
          'What-if grades · grade curving',
        ]}
      />

      <AudienceCta
        title="Walk the flows on the live demo"
        body="The hosted demo includes instructor and student views — gradebook, adaptive quizzes, blueprint sync, and LTI configuration — so your team can evaluate before deploying on your own infrastructure."
        primaryHref={SITE_LINKS.demo}
        primaryLabel="Open demo.lextures.com"
        secondaryHref="/pricing"
        secondaryLabel="Hosted vs self-host pricing"
      />
    </MarketingPageShell>
  )
}

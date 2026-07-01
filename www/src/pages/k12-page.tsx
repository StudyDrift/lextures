import {
  AudienceCta,
  AudienceHero,
  CardGrid,
  MarketingPageShell,
  TagStrip,
} from '../components/marketing-page-shell'
import { SITE_LINKS } from '../lib/site-links'

const TEACHER_FEATURES = [
  {
    title: 'Standards-aligned gradebook',
    body: 'Tag questions and assignments to CCSS, NGSS, or your district framework. Mastery rolls up by standard so you see which objectives need reteaching — not just class averages.',
  },
  {
    title: 'Misconception flags',
    body: 'When most of the class selects the same wrong answer, Lextures surfaces it before the unit test. You walk into the next period knowing which concept to address.',
  },
  {
    title: 'Spaced repetition review',
    body: 'Scheduled review sessions on web and mobile keep earlier units fresh through the year. Students practice retrieval at intervals the engine sets from their last responses.',
  },
]

const ADMIN_FEATURES = [
  {
    title: 'Roster sync from your SIS',
    body: 'Clever and ClassLink SSO connect district identity. OneRoster 1.2 CSV import and SCIM 2.0 HTTP provisioning sync enrollments without manual spreadsheet uploads each Monday.',
  },
  {
    title: 'Accommodations that apply automatically',
    body: 'Extended time, extra attempts, and display accommodations are set once on a student profile and enforced on every quiz — not something teachers toggle before each assessment.',
  },
  {
    title: 'District course blueprints',
    body: 'Push curriculum updates from a master course to every section in the district. Child sections inherit shared content while keeping local additions.',
  },
]

const PARENT_FEATURES = [
  {
    title: 'Parent portal (when enabled)',
    body: 'Districts can turn on read-only parent access: grades, assignments, and due dates for each linked child from one account, with optional email alerts for low grades or missing work.',
  },
  {
    title: 'Multi-child households',
    body: 'Parents switch between linked students without separate logins. Schools provision links individually or through bulk CSV import.',
  },
  {
    title: 'Mobile apps',
    body: 'iOS and Android apps include a parent context when your district configures it — the same grade and calendar data as the web portal.',
  },
]

export function K12Page() {
  return (
    <MarketingPageShell>
      <AudienceHero
        eyebrow="K–12"
        title="Standards, rosters, and accommodations built in"
        lead="For teachers, district administrators, and parents: standards-based grading, automatic accommodations, roster sync, and an optional parent portal — so compliance work happens in the system, not in side spreadsheets."
        primaryHref="/get-started"
        primaryLabel="Try the demo"
        secondaryHref="/parents"
        secondaryLabel="For parents"
      />

      <CardGrid heading="For teachers" items={TEACHER_FEATURES} />

      <CardGrid heading="For school administrators" items={ADMIN_FEATURES} columns={3} />

      <CardGrid heading="For parents" items={PARENT_FEATURES} columns={3} />

      <TagStrip
        label="Standards, identity, and roster protocols"
        tags={[
          'CCSS · NGSS · custom standards',
          'Clever · ClassLink SSO',
          'OneRoster 1.2 CSV',
          'SCIM 2.0',
          'LTI 1.3',
          'Accommodations engine',
          'Parent portal',
          'Daily attendance (when enabled)',
        ]}
      />

      <AudienceCta
        title="See the teacher and parent views"
        body="The demo walks through standards alignment, adaptive quizzes, accommodation profiles, and how a district enables the parent portal for linked families."
        primaryHref={SITE_LINKS.demo}
        primaryLabel="Open demo.lextures.com"
        secondaryHref="/docs"
        secondaryLabel="Read the docs"
      />
    </MarketingPageShell>
  )
}

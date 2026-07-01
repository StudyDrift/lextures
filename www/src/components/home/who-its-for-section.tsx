const AUDIENCES = [
  {
    title: 'Higher ed administrators',
    task: 'Provision users and audit grade changes',
    body: 'SAML 2.0, OIDC, and SCIM 2.0 provisioning connect to your identity stack. Enrollment state history and grade audit events give registrars a defensible record for FERPA requests.',
    href: '/higher-ed',
  },
  {
    title: 'Deans & department chairs',
    task: 'Keep multi-section courses aligned',
    body: 'Course blueprints let a coordinator push syllabus updates, rubrics, and question bank changes to every child section — without emailing ten instructors the same file.',
    href: '/higher-ed',
  },
  {
    title: 'K–12 teachers',
    task: 'See standards mastery, not just points',
    body: 'Map assignments to CCSS, NGSS, or district frameworks. Misconception detection flags the wrong answers most of the class chose before the next lesson.',
    href: '/k-12',
  },
  {
    title: 'School administrators',
    task: 'Sync rosters and enforce accommodations',
    body: 'Clever, ClassLink, and OneRoster CSV bring enrollments in line with your SIS. Accommodations like extended time apply automatically on quizzes — configured once per student.',
    href: '/k-12',
  },
  {
    title: 'Parents',
    task: 'Follow grades without logging in as your child',
    body: 'When your district enables the parent portal, you get read-only access to linked children\'s grades, assignments, and due dates from one account.',
    href: '/parents',
  },
  {
    title: 'Self-learners',
    task: 'Study with adaptive quizzes and spaced review',
    body: 'Create or enroll in self-paced courses, practice with IRT-routed quizzes, and use spaced-repetition review sessions on web, iOS, or Android — no instructor required.',
    href: '/self-learner',
  },
]

export function WhoItsForSection() {
  return (
    <section id="institutions" className="border-b" style={{ borderColor: 'var(--line)' }}>
      <div className="mx-auto max-w-[1200px] px-5 py-16 md:px-10 xl:px-14 xl:py-20">
        <p className="section-label">Who it&apos;s for</p>
        <h2
          className="font-display mt-4 max-w-[520px] text-[clamp(28px,3.5vw,40px)] font-semibold leading-[1.1] tracking-[-0.015em]"
          style={{ color: 'var(--ink)' }}
        >
          Built for the people who run courses
        </h2>

        <div className="mt-11 grid gap-8 sm:grid-cols-2 xl:grid-cols-3 xl:gap-10">
          {AUDIENCES.map(audience => (
            <div key={audience.title}>
              <p className="font-mono text-[11px] uppercase tracking-[0.14em]" style={{ color: 'var(--teal-deep)' }}>
                {audience.title}
              </p>
              <h3
                className="font-display mt-3 text-[22px] font-semibold leading-snug"
                style={{ color: 'var(--ink)' }}
              >
                {audience.task}
              </h3>
              <p className="mt-3 text-[15.5px] leading-[1.6]" style={{ color: 'var(--text-soft)' }}>
                {audience.body}
              </p>
              <a
                href={audience.href}
                className="mt-4 inline-block text-[15px] font-medium no-underline"
                style={{ color: 'var(--teal-deep)' }}
              >
                Learn more →
              </a>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

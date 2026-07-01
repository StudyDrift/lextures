import { ProductScreenshot } from './product-screenshot'

const REPLACEMENTS = [
  {
    instead: 'Adaptive assessment platform',
    inLextures: 'IRT 2PL/3PL quizzing with a course item bank',
  },
  {
    instead: 'Interactive content subscription',
    inLextures: 'Vibe Activities — describe a lesson, publish it to modules',
  },
  {
    instead: 'Gradebook analytics add-on',
    inLextures: 'Grade audit log, student progress, and what-if grades',
  },
  {
    instead: 'Spaced-repetition review app',
    inLextures: 'Review queue built into the course workflow',
  },
  {
    instead: 'Standalone quiz engine',
    inLextures: 'Timed quizzes, attempts, and QTI import in one gradebook',
  },
  {
    instead: 'Enrollment tracking spreadsheets',
    inLextures: 'Enrollment states with a history row for every transition',
  },
] as const

export function VibeActivitiesSection() {
  return (
    <section id="vibe-activities" className="border-b" style={{ borderColor: 'var(--line)' }}>
      <div className="mx-auto max-w-[1200px] px-5 py-16 md:px-10 xl:px-14 xl:py-20">
        <p className="section-label">Interactive content</p>
        <h2
          className="font-display mt-4 max-w-[640px] text-[clamp(28px,3.5vw,40px)] font-semibold leading-[1.1] tracking-[-0.015em]"
          style={{ color: 'var(--ink)' }}
        >
          Vibe Activities replace the tools you stack on top of your LMS
        </h2>
        <p className="mt-4 max-w-[620px] text-[16px] leading-[1.6]" style={{ color: 'var(--text-soft)' }}>
          Describe an interactive lesson in plain language. Lextures generates a sandboxed activity
          you publish straight to a module — the same gradebook, enrollments, and audit trail as
          everything else in the course. No export loop, no second vendor login.
        </p>

        <ProductScreenshot
          src="/assets/screenshots/vibe-activity.png"
          alt="Vibe Activity showing an interactive supply and demand checkpoint with answer choices inside a course module."
          filename="lextures · vibe activity"
          term="Fall 2026"
          className="mt-10"
        />

        <div className="mt-14">
          <h3
            className="font-display max-w-[520px] text-[clamp(22px,2.5vw,28px)] font-semibold leading-[1.15]"
            style={{ color: 'var(--ink)' }}
          >
            One platform instead of a patchwork contract
          </h3>
          <p className="mt-3 max-w-[620px] text-[15.5px] leading-[1.6]" style={{ color: 'var(--text-soft)' }}>
            Universities often pay separately for adaptive testing, interactive authoring, review
            apps, and gradebook extensions. Lextures ships those workflows together — or plugs into
            Canvas, Moodle, and Blackboard over LTI 1.3 when you want to keep your existing shell.
          </p>

          <div
            className="mt-8 overflow-hidden border"
            style={{
              backgroundColor: 'var(--line-card)',
              borderColor: 'var(--line-card)',
              borderRadius: 'var(--radius-card)',
            }}
          >
            <div
              className="hidden border-b px-4 py-3 sm:grid sm:grid-cols-2"
              style={{ backgroundColor: 'var(--panel-sunken)', borderColor: 'var(--line-card)' }}
            >
              <p className="font-mono text-[10px] uppercase tracking-[0.14em]" style={{ color: 'var(--muted)' }}>
                Instead of paying for
              </p>
              <p className="font-mono text-[10px] uppercase tracking-[0.14em]" style={{ color: 'var(--muted)' }}>
                In Lextures
              </p>
            </div>
            <dl className="divide-y" style={{ borderColor: 'var(--line-card)' }}>
              {REPLACEMENTS.map(row => (
                <div
                  key={row.instead}
                  className="grid gap-1 px-4 py-3 sm:grid-cols-2 sm:gap-6"
                  style={{ backgroundColor: 'var(--panel)' }}
                >
                  <dt className="text-[14px] font-medium" style={{ color: 'var(--text-soft)' }}>
                    <span className="font-mono text-[10px] uppercase tracking-[0.14em] sm:hidden" style={{ color: 'var(--muted)' }}>
                      Instead of
                    </span>
                    <span className="mt-0.5 block sm:mt-0">{row.instead}</span>
                  </dt>
                  <dd className="text-[14px] font-medium" style={{ color: 'var(--ink-nav)' }}>
                    <span className="font-mono text-[10px] uppercase tracking-[0.14em] sm:hidden" style={{ color: 'var(--muted)' }}>
                      In Lextures
                    </span>
                    <span className="mt-0.5 block sm:mt-0">{row.inLextures}</span>
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        </div>
      </div>
    </section>
  )
}

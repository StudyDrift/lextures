import { HeroGradebookPanel } from './hero-gradebook-panel'
import { SITE_LINKS } from '../../lib/site-links'

export function HeroSection() {
  return (
    <section className="border-b" style={{ borderColor: 'var(--line)' }}>
      <div className="mx-auto grid max-w-[1200px] items-center gap-11 px-5 py-14 md:px-10 md:py-[60px] xl:grid-cols-[minmax(0,500px)_1fr] xl:gap-[52px] xl:px-14 xl:py-[76px]">
        <div>
          <p className="eyebrow-label">Open-source LMS</p>
          <h1
            className="font-display mt-4 font-semibold tracking-[-0.02em]"
            style={{
              color: 'var(--ink)',
              fontSize: 'clamp(34px, 4.5vw, 62px)',
              lineHeight: 1.06,
            }}
          >
            Every course, cohort, and grade on{' '}
            <em className="not-italic" style={{ color: 'var(--teal-deep)', fontStyle: 'italic' }}>
              one screen.
            </em>
          </h1>
          <p
            className="mt-6 max-w-[500px] text-[19px] leading-[1.6]"
            style={{ color: 'var(--text)' }}
          >
            This is the gradebook — not a mockup of one. Lextures runs courses on Postgres with
            adaptive quizzes, standards-aligned grading, and a grade audit trail your office can
            export.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
            <a href={SITE_LINKS.demo} className="btn-primary" target="_blank" rel="noopener noreferrer">
              Try the demo
            </a>
            <a href="/docs" className="btn-secondary">
              Read the docs →
            </a>
          </div>
          <p
            className="mt-6 font-mono text-[12px] tracking-[0.06em]"
            style={{ color: 'var(--muted)' }}
          >
            SELF-HOSTED · LTI 1.3 · AGPL-3.0
          </p>
        </div>

        <div className="min-w-0 overflow-x-auto pb-1 xl:overflow-visible">
          <HeroGradebookPanel />
        </div>
      </div>
    </section>
  )
}

import { HeroProductPanel } from './hero-product-panel'
import { SITE_LINKS } from '../../lib/site-links'

export function HeroSection() {
  return (
    <section className="border-b" style={{ borderColor: 'var(--line)' }}>
      <div className="mx-auto grid max-w-[1200px] items-center gap-11 px-5 py-14 md:px-10 md:py-[60px] xl:grid-cols-[minmax(0,500px)_1fr] xl:gap-[52px] xl:px-14 xl:py-[76px]">
        <div>
          <h1
            className="font-display font-semibold tracking-[-0.02em]"
            style={{
              color: 'var(--ink)',
              fontSize: 'clamp(34px, 4.5vw, 62px)',
              lineHeight: 1.06,
            }}
          >
            The learning environment that{' '}
            <em className="not-italic" style={{ color: 'var(--teal-deep)', fontStyle: 'italic' }}>
              adapts.
            </em>
          </h1>
          <p
            className="mt-6 max-w-[500px] text-[19px] leading-[1.6]"
            style={{ color: 'var(--text)' }}
          >
            Stop juggling separate contracts for adaptive quizzing, interactive content, spaced
            review, and gradebook add-ons. Lextures brings enrollment, assessments, grading, and
            audit trails into one platform.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
            <a href={SITE_LINKS.demo} className="btn-primary" target="_blank" rel="noopener noreferrer">
              Try the demo
            </a>
            <a href="/docs" className="btn-secondary">
              Read the docs →
            </a>
          </div>
        </div>

        <div className="min-w-0 overflow-x-auto pb-1 xl:overflow-visible">
          <HeroProductPanel />
        </div>
      </div>
    </section>
  )
}

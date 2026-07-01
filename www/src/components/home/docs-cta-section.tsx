import { SITE_LINKS } from '../../lib/site-links'

export function DocsCtaSection() {
  return (
    <section className="border-b" style={{ borderColor: 'var(--line)' }}>
      <div className="mx-auto max-w-[1200px] px-5 py-16 md:px-10 xl:px-14 xl:py-20">
        <div
          className="border px-6 py-10 sm:px-10 sm:py-12"
          style={{
            backgroundColor: 'var(--panel)',
            borderColor: 'var(--line-card)',
            borderRadius: 'var(--radius-card)',
            boxShadow: 'var(--shadow-page)',
          }}
        >
          <p className="section-label">Documentation</p>
          <h2
            className="font-display mt-4 text-[clamp(28px,3.5vw,36px)] font-semibold leading-[1.1] tracking-[-0.015em]"
            style={{ color: 'var(--ink)' }}
          >
            Documentation for administrators and instructors
          </h2>
          <p className="mt-4 max-w-[520px] text-[16px] leading-[1.6]" style={{ color: 'var(--text-soft)' }}>
            LTI setup, course workflows, and integration guides — written for the people who
            configure and run courses.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
            <a href="/docs" className="btn-primary">
              Read the docs
            </a>
            <a
              href={SITE_LINKS.github}
              className="btn-secondary"
              target="_blank"
              rel="noopener noreferrer"
            >
              GitHub →
            </a>
          </div>
        </div>
      </div>
    </section>
  )
}

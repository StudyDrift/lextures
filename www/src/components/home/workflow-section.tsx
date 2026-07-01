import { ProductScreenshot } from './product-screenshot'

export function WorkflowSection() {
  return (
    <section id="workflow" style={{ backgroundColor: 'var(--ink-nav)' }}>
      <div className="mx-auto max-w-[1200px] px-5 py-16 md:px-10 xl:px-14 xl:py-20">
        <p className="font-mono text-[12px] uppercase tracking-[0.18em]" style={{ color: 'var(--muted)' }}>
          One workflow, start to finish
        </p>
        <h2
          className="font-display mt-4 max-w-[640px] text-[clamp(28px,3.5vw,40px)] font-semibold leading-[1.1] tracking-[-0.015em]"
          style={{ color: 'var(--cream)' }}
        >
          Enrollment states and grade changes your office can trace
        </h2>
        <p className="mt-4 max-w-[560px] text-[16px] leading-[1.6]" style={{ color: 'var(--text-soft)' }}>
          When a student withdraws or receives an incomplete, the enrollment state machine records
          the transition. When an instructor adjusts a score, the grade audit log captures the change
          — not a separate spreadsheet.
        </p>

        <ProductScreenshot
          src="/assets/screenshots/gradebook-grid.png"
          alt="Gradebook grid with student rows, assignment columns, and class average statistics."
          filename="lextures · gradebook"
          term="Fall 2026"
          className="mt-10"
        />
      </div>
    </section>
  )
}

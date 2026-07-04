import { WindLines } from './wind-lines'

type Step = { n: string; title: string; desc: string }

const STEPS: Step[] = [
  {
    n: '1',
    title: 'Bring your roster aboard',
    desc: 'Import courses, sections, and students from your SIS in minutes — no spreadsheets.',
  },
  {
    n: '2',
    title: 'Build the voyage',
    desc: 'Assemble adaptive lessons and quizzes with a drag-and-drop editor made for teaching.',
  },
  {
    n: '3',
    title: 'Set sail',
    desc: 'Launch to students and watch progress chart itself in real time on one dashboard.',
  },
]

export function WorkflowSection() {
  return (
    <section
      id="how"
      className="relative mt-[92px] overflow-hidden"
      style={{ backgroundColor: '#22333b', color: '#eef4f0' }}
    >
      <WindLines variant="deep" />

      <div className="relative z-[2] mx-auto max-w-[1180px] px-7 pb-[90px] pt-[84px]">
        <div className="mx-auto max-w-[640px] text-center">
          <span
            className="text-[13px] font-semibold uppercase tracking-[0.06em]"
            style={{ color: '#6ac5b0' }}
          >
            Chart the voyage
          </span>
          <h2
            className="font-display mt-3 font-semibold tracking-[-0.02em]"
            style={{ fontSize: 'clamp(30px, 3.6vw, 44px)', lineHeight: 1.08 }}
          >
            From scattered courses to one clear crossing.
          </h2>
        </div>

        <div className="mt-[52px] grid gap-[22px] md:grid-cols-3">
          {STEPS.map(s => (
            <div
              key={s.n}
              className="min-w-0 rounded-[18px] px-[26px] pb-8 pt-[30px]"
              style={{
                backgroundColor: 'rgba(255,255,255,0.04)',
                border: '1px solid rgba(255,255,255,0.09)',
              }}
            >
              <div className="flex items-center gap-3">
                <span
                  className="font-display flex h-[34px] w-[34px] items-center justify-center rounded-full text-[15px] font-semibold"
                  style={{ border: '1.5px solid var(--coral)', color: 'var(--coral)' }}
                >
                  {s.n}
                </span>
                <span
                  className="h-px flex-1"
                  style={{ background: 'linear-gradient(90deg, rgba(255,255,255,0.2), transparent)' }}
                />
              </div>
              <h3 className="font-display mt-5 text-[22px] font-semibold">{s.title}</h3>
              <p className="mt-2.5 text-[15px] leading-[1.55]" style={{ color: '#b8c6c4' }}>
                {s.desc}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

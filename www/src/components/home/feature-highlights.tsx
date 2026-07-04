type Feature = {
  title: string
  desc: string
  color: string
  tint: string
}

const FEATURES: Feature[] = [
  {
    title: 'Adaptive quizzing',
    desc: 'Questions that adjust in real time to what each learner already knows — nobody bored, nobody lost.',
    color: '#f2684e',
    tint: 'rgba(242,104,78,0.12)',
  },
  {
    title: 'Interactive lessons',
    desc: 'Lessons students actually engage with. Embed questions, media, and checkpoints, not just slides.',
    color: '#f49b44',
    tint: 'rgba(244,155,68,0.14)',
  },
  {
    title: 'Automated grading',
    desc: 'Reclaim hours every week. Feedback and gradebook sync happen the moment work is submitted.',
    color: '#6ac5b0',
    tint: 'rgba(106,197,176,0.16)',
  },
  {
    title: 'One roster, one place',
    desc: 'Consolidate enrollment across every course and section into a single, tidy source of truth.',
    color: '#4fa894',
    tint: 'rgba(79,168,148,0.14)',
  },
]

function SailIcon({ color }: { color: string }) {
  return (
    <svg viewBox="0 0 48 58" width="30" height="36" aria-hidden>
      <line x1="8" y1="4" x2="8" y2="54" stroke="#22333b" strokeWidth="3.4" strokeLinecap="round" />
      <path d="M11,9 C30,11 37,22 33,32 C29,42 20,46 11,47 Z" fill={color} />
    </svg>
  )
}

export function FeatureHighlights() {
  return (
    <section id="features" className="mx-auto max-w-[1180px] px-7 pb-5 pt-[88px]">
      <div className="max-w-[720px]">
        <span
          className="text-[13px] font-semibold uppercase tracking-[0.06em]"
          style={{ color: 'var(--coral)' }}
        >
          Everything on board
        </span>
        <h2
          className="font-display mt-3 font-semibold tracking-[-0.02em]"
          style={{ color: '#22333b', fontSize: 'clamp(30px, 3.6vw, 44px)', lineHeight: 1.08 }}
        >
          Everything you need to keep the class on course.
        </h2>
      </div>

      <div className="mt-11 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
        {FEATURES.map(f => (
          <div
            key={f.title}
            className="min-w-0 rounded-[18px] px-6 pb-7 pt-[26px] transition-all duration-200 hover:-translate-y-1"
            style={{
              backgroundColor: 'var(--panel)',
              border: '1px solid rgba(38,58,60,0.08)',
              boxShadow: '0 10px 26px rgba(34,51,59,0.05)',
            }}
            onMouseEnter={e => {
              e.currentTarget.style.boxShadow = '0 18px 38px rgba(34,51,59,0.09)'
            }}
            onMouseLeave={e => {
              e.currentTarget.style.boxShadow = '0 10px 26px rgba(34,51,59,0.05)'
            }}
          >
            <div
              className="flex h-[52px] w-[52px] items-center justify-center rounded-[13px]"
              style={{ backgroundColor: f.tint }}
            >
              <SailIcon color={f.color} />
            </div>
            <h3
              className="font-display mt-[18px] text-[21px] font-semibold"
              style={{ color: '#22333b' }}
            >
              {f.title}
            </h3>
            <p className="mt-[9px] text-[15px] leading-[1.55]" style={{ color: '#56676a' }}>
              {f.desc}
            </p>
          </div>
        ))}
      </div>
    </section>
  )
}

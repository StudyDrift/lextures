import { WindLines } from './wind-lines'

export function HeroSection() {
  return (
    <section id="top" className="relative overflow-hidden">
      <WindLines variant="hero" />

      <div className="relative z-[2] mx-auto grid max-w-[1180px] items-center gap-10 px-7 pb-[92px] pt-[76px] lg:grid-cols-[1.05fr_0.95fr]">
        <div className="min-w-0" style={{ animation: 'lx-fade-up 0.7s ease both' }}>
          <span
            className="inline-flex items-center gap-2 rounded-full px-3.5 py-[7px] text-[13px] font-semibold uppercase tracking-[0.04em]"
            style={{ color: '#4fa894', backgroundColor: 'rgba(106,197,176,0.14)' }}
          >
            The learning environment that adapts
          </span>
          <h1
            className="font-display mt-[22px] font-semibold tracking-[-0.02em] text-balance"
            style={{ color: '#22333b', fontSize: 'clamp(40px, 5.4vw, 66px)', lineHeight: 1.03 }}
          >
            Set a course for learning that{' '}
            <span className="italic" style={{ color: 'var(--coral)' }}>
              adapts
            </span>{' '}
            to every student.
          </h1>
          <p className="mt-[22px] max-w-[30em] text-[19px] leading-[1.55]" style={{ color: '#4a5b5d' }}>
            One platform for adaptive quizzing, interactive lessons, and automated grading — with
            every course and roster consolidated in a single place.
          </p>
          <div className="mt-8 flex flex-wrap gap-3.5">
            <a href="/get-started" className="btn-primary">
              Start free →
            </a>
            <a href="#how" className="btn-secondary">
              Watch a demo
            </a>
          </div>
          <p
            className="mt-[26px] flex flex-wrap gap-x-4 gap-y-2 text-[14px]"
            style={{ color: '#7b8a8b' }}
          >
            <span>Consolidate</span>
            <span style={{ color: '#c7cdc9' }}>·</span>
            <span>FERPA-ready</span>
            <span style={{ color: '#c7cdc9' }}>·</span>
            <span>Works with your SIS</span>
          </p>
        </div>

        <div className="relative flex min-w-0 min-h-[380px] items-center justify-center">
          <div
            className="absolute h-[360px] w-[360px] rounded-full"
            style={{
              background:
                'radial-gradient(circle at 50% 45%, rgba(106,197,176,0.22), rgba(106,197,176,0) 68%)',
            }}
          />
          <img
            src="/logo.svg"
            alt="Lextures — a ship built of books"
            className="relative w-[min(380px,78%)]"
            style={{
              filter: 'drop-shadow(0 26px 40px rgba(34,51,59,0.16))',
              animation: 'lx-bob 7s ease-in-out infinite',
              transformOrigin: '50% 90%',
            }}
          />
        </div>
      </div>
      <div
        className="h-px"
        style={{ background: 'linear-gradient(90deg, transparent, rgba(38,58,60,0.1), transparent)' }}
      />
    </section>
  )
}

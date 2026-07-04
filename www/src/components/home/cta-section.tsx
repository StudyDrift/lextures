import { WindLines } from './wind-lines'

export function CtaSection() {
  return (
    <section
      id="cta"
      className="relative overflow-hidden"
      style={{ backgroundColor: '#4fa894', color: '#fff' }}
    >
      <WindLines variant="teal" />

      <div className="relative z-[2] mx-auto flex max-w-[1180px] flex-wrap items-center justify-between gap-8 px-7 py-[88px]">
        <div className="min-w-0 max-w-[34em]">
          <h2
            className="font-display font-semibold tracking-[-0.02em]"
            style={{ fontSize: 'clamp(32px, 4vw, 50px)', lineHeight: 1.05 }}
          >
            Ready to set sail with your class?
          </h2>
          <p className="mt-4 text-[18px] leading-[1.55]" style={{ color: 'rgba(255,255,255,0.9)' }}>
            Bring your roster on board and launch your first adaptive course in an afternoon.
          </p>
          <div className="mt-7 flex flex-wrap gap-3.5">
            <a href="/get-started" className="btn-primary">
              Start free →
            </a>
            <a
              href="/request-information"
              className="inline-flex items-center gap-2 rounded-full px-6 py-3.5 text-[16px] font-semibold text-white no-underline transition-colors duration-150"
              style={{ backgroundColor: 'rgba(255,255,255,0.14)', border: '1px solid rgba(255,255,255,0.3)' }}
              onMouseEnter={e => {
                e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.22)'
              }}
              onMouseLeave={e => {
                e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.14)'
              }}
            >
              Talk to us
            </a>
          </div>
        </div>
        <img
          src="/logo.svg"
          alt=""
          aria-hidden
          className="w-[172px]"
          style={{
            filter: 'drop-shadow(0 18px 30px rgba(0,0,0,0.2))',
            animation: 'lx-bob 8s ease-in-out infinite',
            transformOrigin: '50% 90%',
          }}
        />
      </div>
    </section>
  )
}

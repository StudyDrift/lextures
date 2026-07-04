import { WindLines } from './wind-lines'

export function QuoteSection() {
  return (
    <section id="quote" className="relative overflow-hidden" style={{ backgroundColor: '#f1e5c6' }}>
      <WindLines variant="sand" />

      <div className="relative z-[2] mx-auto max-w-[900px] px-7 py-[84px] text-center">
        <div
          className="font-display h-[30px] text-[64px] leading-[0.6]"
          style={{ color: 'var(--coral)' }}
        >
          &ldquo;
        </div>
        <p
          className="font-display mt-2 font-medium italic tracking-[-0.01em] text-balance"
          style={{ color: '#3a3320', fontSize: 'clamp(24px, 3.2vw, 34px)', lineHeight: 1.34 }}
        >
          Lextures turned our scattered courses into one clear voyage. Grading that used to eat my
          weekends now takes minutes.
        </p>
        <div className="mt-7 flex items-center justify-center gap-3">
          <span
            className="font-display flex h-[42px] w-[42px] items-center justify-center rounded-full font-bold text-white"
            style={{ backgroundColor: '#6ac5b0' }}
          >
            M
          </span>
          <div className="text-left">
            <div className="text-[15px] font-bold" style={{ color: '#3a3320' }}>
              Dr. Maya Okonkwo
            </div>
            <div className="text-[14px]" style={{ color: '#8a7c56' }}>
              Learning Design Lead, Northbay College
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

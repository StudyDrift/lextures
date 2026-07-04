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
          Consolidate many tools into one platform — one enrollment record, one gradebook, one place
          to teach.
        </p>
      </div>
    </section>
  )
}

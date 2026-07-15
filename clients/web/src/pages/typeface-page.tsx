import { useState } from 'react'
import { Link } from 'react-router-dom'

const weights = [
  { name: 'Regular', value: 400, use: 'Reading and learning content' },
  { name: 'Medium', value: 500, use: 'Navigation and controls' },
  { name: 'SemiBold', value: 600, use: 'Section headings and emphasis' },
  { name: 'Bold', value: 700, use: 'Titles and brand moments' },
]

const principles = [
  {
    number: '01',
    title: 'Made for reading',
    copy: 'A high x-height and open counters keep lessons comfortable at small sizes and over long sessions.',
  },
  {
    number: '02',
    title: 'Clear at a glance',
    copy: 'Distinct letterforms and sturdy punctuation help labels, grades, dates, and dense course tables scan quickly.',
  },
  {
    number: '03',
    title: 'Warm, not whimsical',
    copy: 'Humanist shapes give the interface a welcoming voice while the rhythm stays disciplined and calm.',
  },
]

export default function TypefacePage() {
  const [sample, setSample] = useState('Learning changes the shape of tomorrow.')
  const [size, setSize] = useState(72)

  return (
    <main className="font-lextures min-h-screen bg-[#f6f1e7] text-[#14262f]">
      <header className="border-b border-[#d8cfbc] px-5 py-5 sm:px-8 lg:px-12">
        <nav className="mx-auto flex max-w-7xl items-center justify-between" aria-label="Typeface page navigation">
          <Link to="/" className="text-xl font-semibold tracking-[-0.02em]">lextures</Link>
          <span className="rounded-full border border-[#cfc5b1] px-3 py-1 text-xs font-medium uppercase tracking-[0.14em] text-[#58656b]">
            Type specimen · 2026
          </span>
        </nav>
      </header>

      <section className="overflow-hidden border-b border-[#d8cfbc] px-5 pb-16 pt-14 sm:px-8 sm:pb-24 sm:pt-20 lg:px-12">
        <div className="mx-auto grid max-w-7xl items-end gap-10 lg:grid-cols-[minmax(0,1fr)_20rem]">
          <div>
            <p className="mb-5 text-sm font-semibold uppercase tracking-[0.18em] text-[#1f7a63]">The Lextures typeface</p>
            <h1 className="max-w-5xl text-[clamp(4.5rem,15vw,12rem)] font-semibold leading-[0.78] tracking-[-0.055em]">
              Learn in<br />your own way.
            </h1>
          </div>
          <p className="max-w-sm text-lg leading-7 text-[#4a5960] lg:pb-2">
            A warm humanist sans designed for the places where learning happens—from a single annotation to an entire course.
          </p>
        </div>
      </section>

      <section className="border-b border-[#d8cfbc] px-5 py-16 sm:px-8 lg:px-12">
        <div className="mx-auto max-w-7xl">
          <div className="mb-12 grid gap-4 sm:grid-cols-2">
            <h2 className="text-3xl font-semibold tracking-[-0.025em]">One family. Four voices.</h2>
            <p className="max-w-lg text-[#58656b] sm:justify-self-end">Lextures moves from quiet body copy to confident display type without changing character.</p>
          </div>
          <div className="divide-y divide-[#d8cfbc] border-y border-[#d8cfbc]">
            {weights.map((weight) => (
              <article key={weight.value} className="grid gap-3 py-7 sm:grid-cols-[8rem_1fr_13rem] sm:items-baseline">
                <div><p className="font-medium">{weight.name}</p><p className="text-sm tabular-nums text-[#768086]">{weight.value}</p></div>
                <p className="text-4xl leading-none tracking-[-0.015em] sm:text-6xl" style={{ fontWeight: weight.value }}>Curiosity grows here.</p>
                <p className="text-sm text-[#657178] sm:text-end">{weight.use}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-[#17313f] px-5 py-20 text-[#f8f1e4] sm:px-8 sm:py-28 lg:px-12">
        <div className="mx-auto max-w-7xl">
          <p className="mb-8 text-sm font-semibold uppercase tracking-[0.18em] text-[#86d1c2]">Reading sample</p>
          <blockquote className="max-w-6xl text-[clamp(2.4rem,6vw,5.75rem)] font-medium leading-[1.05] tracking-[-0.025em]">
            “The beautiful thing about learning is that no one can take it away from you.”
          </blockquote>
          <p className="mt-8 text-base text-[#a9c0c8]">B. B. King</p>
        </div>
      </section>

      <section className="border-b border-[#d8cfbc] px-5 py-16 sm:px-8 sm:py-24 lg:px-12">
        <div className="mx-auto max-w-7xl">
          <div className="flex flex-col gap-5 border-b border-[#d8cfbc] pb-6 sm:flex-row sm:items-end sm:justify-between">
            <div><p className="text-sm font-semibold uppercase tracking-[0.18em] text-[#1f7a63]">Type tester</p><h2 className="mt-2 text-3xl font-semibold tracking-[-0.025em]">Make it yours</h2></div>
            <label className="flex items-center gap-3 text-sm font-medium text-[#58656b]">
              Size <input aria-label="Font size" type="range" min="32" max="120" value={size} onChange={(event) => setSize(Number(event.target.value))} className="accent-[#1f7a63]" />
              <span className="w-12 text-end tabular-nums">{size}px</span>
            </label>
          </div>
          <textarea aria-label="Typeface sample text" value={sample} onChange={(event) => setSample(event.target.value)} className="mt-8 min-h-56 w-full resize-none bg-transparent leading-[1.05] tracking-[-0.02em] outline-none placeholder:text-[#9b968b]" style={{ fontSize: `clamp(2rem, ${size / 16}vw, ${size}px)` }} />
        </div>
      </section>

      <section className="border-b border-[#d8cfbc] px-5 py-16 sm:px-8 sm:py-24 lg:px-12">
        <div className="mx-auto grid max-w-7xl gap-12 lg:grid-cols-[20rem_1fr]">
          <div>
            <p className="text-sm font-semibold uppercase tracking-[0.18em] text-[#1f7a63]">The wind signature</p>
            <h2 className="mt-3 text-3xl font-semibold tracking-[-0.025em]">Drawn from the mark</h2>
            <p className="mt-5 leading-7 text-[#58656b]">Rising crossbars recall open pages. Flag terminals lean into the wind. The details give Lextures a voice without interrupting a lesson.</p>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="flex min-h-64 items-center justify-center rounded-2xl bg-[#ef6449] px-6"><p className="text-[clamp(5rem,14vw,10rem)] font-semibold leading-none tracking-[-0.055em]">Let</p></div>
            <div className="flex min-h-64 items-center justify-center rounded-2xl bg-[#6ec0b1] px-6"><p className="text-[clamp(5rem,14vw,10rem)] font-semibold leading-none tracking-[-0.055em]">AEr7</p></div>
          </div>
        </div>
      </section>

      <section className="border-b border-[#d8cfbc] px-5 py-16 sm:px-8 sm:py-24 lg:px-12">
        <div className="mx-auto grid max-w-7xl gap-12 lg:grid-cols-[18rem_1fr]">
          <div><p className="text-sm font-semibold uppercase tracking-[0.18em] text-[#1f7a63]">Character</p><h2 className="mt-3 text-3xl font-semibold tracking-[-0.025em]">Clear by design</h2></div>
          <div className="space-y-7 overflow-hidden">
            <p className="break-words text-[clamp(2.6rem,7vw,6rem)] font-medium leading-none tracking-[-0.015em]">ABCDEFGHIJKLM<br />NOPQRSTUVWXYZ</p>
            <p className="break-words text-[clamp(2.6rem,7vw,6rem)] leading-none tracking-[-0.015em] text-[#1f7a63]">abcdefghijklm<br />nopqrstuvwxyz</p>
            <p className="text-[clamp(2.6rem,7vw,6rem)] font-medium leading-none tracking-[-0.01em] tabular-nums">0123456789</p>
            <p className="text-3xl tracking-[0.08em] text-[#58656b] sm:text-5xl">&amp; @ ? ! $ % # ( ) [ ] {'{ }'}</p>
          </div>
        </div>
      </section>

      <section className="px-5 py-16 sm:px-8 sm:py-24 lg:px-12">
        <div className="mx-auto max-w-7xl">
          <h2 className="max-w-2xl text-4xl font-semibold tracking-[-0.025em] sm:text-5xl">Built around the learner.</h2>
          <div className="mt-12 grid gap-px overflow-hidden rounded-2xl border border-[#d8cfbc] bg-[#d8cfbc] md:grid-cols-3">
            {principles.map((principle) => (
              <article key={principle.number} className="bg-[#fbf7ef] p-7 sm:p-9">
                <p className="text-sm font-semibold tabular-nums text-[#1f7a63]">{principle.number}</p>
                <h3 className="mt-16 text-2xl font-semibold tracking-[-0.02em]">{principle.title}</h3>
                <p className="mt-3 leading-7 text-[#58656b]">{principle.copy}</p>
              </article>
            ))}
          </div>
          <footer className="mt-20 flex flex-col gap-4 border-t border-[#d8cfbc] pt-6 text-sm text-[#657178] sm:flex-row sm:items-center sm:justify-between">
            <p>Lextures Regular, Medium, SemiBold, and Bold</p><p>Designed for learning · Self-hosted · Open Font License 1.1</p>
          </footer>
        </div>
      </section>
    </main>
  )
}

import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

const previewClass = [
  'min-w-0 text-[15px] leading-[1.65] text-fg-default',
  '[&_h1]:mt-6 [&_h1]:text-2xl [&_h1]:font-bold [&_h1]:text-fg-default [&_h1]:first:mt-0',
  '[&_h2]:mt-8 [&_h2]:text-xl [&_h2]:font-semibold [&_h2]:text-fg-default [&_h2]:first:mt-0',
  '[&_h3]:mt-6 [&_h3]:text-lg [&_h3]:font-semibold [&_h3]:text-fg-default [&_h3]:first:mt-0',
  '[&_p]:mt-3 [&_p:first-child]:mt-0',
  '[&_ul]:mt-3 [&_ul]:list-disc [&_ul]:ps-5',
  '[&_ol]:mt-3 [&_ol]:list-decimal [&_ol]:ps-5',
  '[&_li]:mt-1',
  '[&_a]:font-medium [&_a]:text-accent-fg [&_a]:underline [&_a]:underline-offset-2',
  '[&_strong]:font-semibold',
  '[&_code]:rounded [&_code]:bg-surface-sunken [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[13px]',
  '[&_pre]:mt-3 [&_pre]:overflow-x-auto [&_pre]:rounded-lg [&_pre]:bg-surface-sunken [&_pre]:p-3 [&_pre]:font-mono [&_pre]:text-[13px]',
  '[&_blockquote]:mt-3 [&_blockquote]:border-s-4 [&_blockquote]:border-border-strong [&_blockquote]:ps-4 [&_blockquote]:italic [&_blockquote]:text-fg-muted',
  '[&_table]:mt-3 [&_table]:w-full [&_table]:border-collapse [&_table]:text-sm',
  '[&_th]:border-b [&_th]:border-border-default [&_th]:bg-surface-sunken [&_th]:px-3 [&_th]:py-2 [&_th]:text-start [&_th]:font-semibold',
  '[&_td]:border-b [&_td]:border-border-subtle [&_td]:px-3 [&_td]:py-2',
].join(' ')

const DIRECTIVE_RE =
  /^:::[ \t]*(key-takeaways|answer|definition|comparison-table|steps|faq|callout|stat|sources)(?:[ \t]+([^\n]*))?\r?\n([\s\S]*?)^:::[ \t]*$/gim

type Segment =
  | { type: 'markdown'; text: string }
  | { type: 'directive'; name: string; args: string; content: string }

function splitMarketingMarkdown(source: string): Segment[] {
  const text = source ?? ''
  const segments: Segment[] = []
  const re = new RegExp(DIRECTIVE_RE.source, DIRECTIVE_RE.flags)
  let last = 0
  let match: RegExpExecArray | null
  while ((match = re.exec(text))) {
    if (match.index > last) segments.push({ type: 'markdown', text: text.slice(last, match.index) })
    segments.push({
      type: 'directive',
      name: match[1].toLowerCase(),
      args: (match[2] ?? '').trim(),
      content: match[3],
    })
    last = match.index + match[0].length
  }
  if (last < text.length) segments.push({ type: 'markdown', text: text.slice(last) })
  return segments
}

function Markdown({ source }: { source: string }) {
  const text = source.replace(/^<!--[\s\S]*?-->$/gm, '').trim()
  if (!text) return null
  return <ReactMarkdown remarkPlugins={[remarkGfm]}>{text}</ReactMarkdown>
}

function calloutKind(args: string): 'note' | 'warning' | 'tip' {
  const raw = args.toLowerCase()
  const typed = raw.match(/type\s*=\s*["']?(note|warning|tip)/)
  const kind = typed?.[1] ?? raw.replace(/[{}"]/g, '').trim()
  return kind === 'warning' || kind === 'tip' ? kind : 'note'
}

function faqEntries(content: string): Array<{ question: string; answer: string }> {
  const entries: Array<{ question: string; answer: string }> = []
  for (const line of content.replace(/\r\n/g, '\n').split('\n')) {
    if (line.startsWith('### ') && line.trim().endsWith('?')) {
      entries.push({ question: line.slice(4).trim(), answer: '' })
      continue
    }
    const current = entries[entries.length - 1]
    if (current) current.answer += `${line}\n`
  }
  return entries
}

function Directive({ name, args, content }: { name: string; args: string; content: string }) {
  const card = 'my-4 rounded-xl border border-border-default bg-surface-sunken p-4'
  if (name === 'key-takeaways') {
    return (
      <aside className={card} aria-label="Key takeaways">
        <h2 className="!mt-0 text-base font-semibold">Key takeaways</h2>
        <Markdown source={content} />
      </aside>
    )
  }
  if (name === 'answer') {
    return (
      <section className={card} aria-label="Direct answer">
        <Markdown source={content} />
      </section>
    )
  }
  if (name === 'faq') {
    return (
      <section className={card} aria-label="Frequently asked questions">
        <h2 className="!mt-0 text-base font-semibold">Frequently asked questions</h2>
        <div className="mt-2 space-y-2">
          {faqEntries(content).map((entry) => (
            <details key={entry.question} className="rounded-lg border border-border-subtle bg-surface-raised px-3 py-2" open>
              <summary className="cursor-pointer font-medium text-fg-default">{entry.question}</summary>
              <div className="mt-2"><Markdown source={entry.answer} /></div>
            </details>
          ))}
        </div>
      </section>
    )
  }
  if (name === 'callout') {
    const kind = calloutKind(args)
    const label = kind === 'warning' ? 'Warning' : kind === 'tip' ? 'Tip' : 'Note'
    return (
      <aside className={card} aria-label={label}>
        <p className="text-xs font-semibold uppercase tracking-wide text-fg-muted">{label}</p>
        <Markdown source={content} />
      </aside>
    )
  }
  if (name === 'steps') {
    return <section className={card} aria-label="Steps"><Markdown source={content} /></section>
  }
  if (name === 'sources') {
    return (
      <section className={card} aria-label="Sources">
        <h2 className="!mt-0 text-base font-semibold">Sources</h2>
        <Markdown source={content} />
      </section>
    )
  }
  if (name === 'stat') {
    return (
      <figure className={card}>
        <blockquote className="!mt-0 !border-0 !ps-0 !italic"><Markdown source={content} /></blockquote>
        {args ? <figcaption className="mt-2 text-sm text-fg-muted">{args}</figcaption> : null}
      </figure>
    )
  }
  return <section className={card}><Markdown source={content} /></section>
}

export function ArticlePreview({ title, body, dir = 'auto' }: { title: string; body: string; dir?: 'ltr' | 'rtl' | 'auto' }) {
  const segments = splitMarketingMarkdown(body)
  return (
    <article dir={dir} className={`${previewClass} rounded-xl bg-surface-raised p-5 sm:p-8`}>
      <h1 className="!mt-0 text-2xl font-semibold tracking-tight sm:text-3xl">{title || 'Untitled article'}</h1>
      {segments.map((segment, index) => (
        segment.type === 'markdown'
          ? <Markdown key={index} source={segment.text} />
          : <Directive key={index} name={segment.name} args={segment.args} content={segment.content} />
      ))}
    </article>
  )
}

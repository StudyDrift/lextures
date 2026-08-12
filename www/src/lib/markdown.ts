/**
 * Build-time / SSR markdown → HTML (SEO.4 FR-5).
 * markdown-it stays out of interactive client chunks when only content pages
 * import this module (those pages use interactive:false + static-island).
 */
import MarkdownIt from 'markdown-it'

export { renderMarkdownLite } from './markdown-lite'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: false,
})

const directiveLabels = {
  keyTakeaways: 'Key takeaways', answer: 'Direct answer', sources: 'Sources',
  faq: 'Frequently asked questions', note: 'Note', warning: 'Warning', tip: 'Tip',
} as const

function escape(value: string): string { return md.utils.escapeHtml(value) }

function renderDirectives(source: string): { markdown: string; replacements: Map<string, string> } {
  const replacements = new Map<string, string>()
  let sequence = 0
  const stash = (html: string) => {
    const token = `CONTENTDIRECTIVE${sequence++}TOKEN`
    replacements.set(`<p>${token}</p>`, html)
    replacements.set(token, html)
    return token
  }
  const markdown = source.replace(
    /^:::[ \t]*(key-takeaways|answer|definition|comparison-table|steps|faq|callout|stat|sources)(?:[ \t]+([^\n]*))?\r?\n([\s\S]*?)^:::[ \t]*$/gim,
    (_whole, rawName: string, rawArgs: string, rawContent: string) => {
      const name = rawName.toLowerCase(); const args = String(rawArgs || '').trim(); const content = rawContent.trim()
      if (name === 'key-takeaways') return stash(`<aside class="content-card content-takeaways" aria-labelledby="key-takeaways-${sequence}"><h2 id="key-takeaways-${sequence}">${directiveLabels.keyTakeaways}</h2>${md.render(content)}</aside>`)
      if (name === 'answer') return stash(`<section class="content-card content-answer" aria-label="${directiveLabels.answer}">${md.render(content)}</section>`)
      if (name === 'definition') {
        const term = args.replace(/^term\s*=\s*['"]?|['"]$/g, '')
        return stash(`<dfn class="content-definition" data-definition-term="${escape(term)}"><strong>${escape(term)}</strong>${md.render(content)}</dfn>`)
      }
      if (name === 'comparison-table') return stash(`<section class="content-comparison">${args ? `<p class="content-table-summary">${escape(args.replace(/^summary\s*=\s*['"]?|['"]$/g, ''))}</p>` : ''}<div class="content-table-scroll">${md.render(content)}</div></section>`)
      if (name === 'steps') return stash(`<section class="content-steps" data-how-to>${md.render(content)}</section>`)
      if (name === 'faq') {
        const entries = [...content.matchAll(/^###\s+(.+\?)\s*\r?\n([\s\S]*?)(?=^###\s+|$)/gm)]
        const items = entries.map((entry, index) => `<details class="content-faq-item" open><summary id="faq-q-${sequence}-${index}">${escape(entry[1])}</summary><div aria-labelledby="faq-q-${sequence}-${index}">${md.render(entry[2].trim())}</div></details>`).join('')
        return stash(`<section class="content-faq" data-faq><h2>${directiveLabels.faq}</h2>${items}</section>`)
      }
      if (name === 'callout') {
        const type = /^(note|warning|tip)$/.test(args) ? args : 'note'
        return stash(`<aside class="content-callout content-callout-${type}"><strong>${directiveLabels[type as keyof typeof directiveLabels]}</strong>${md.render(content)}</aside>`)
      }
      if (name === 'stat') return stash(`<figure class="content-stat"><blockquote>${md.renderInline(content)}</blockquote>${args ? `<figcaption>${escape(args)}</figcaption>` : ''}</figure>`)
      return stash(content ? `<section class="content-sources"><h2>${directiveLabels.sources}</h2>${md.render(content)}</section>` : '')
    },
  )
  return { markdown, replacements }
}

const LOCAL_IMAGE_DIMENSIONS: Record<string, [number, number]> = {
  '/docs-course-interface.png': [1280, 720],
  '/docs-create-course-dashboard.png': [1280, 800],
  '/docs-create-course-step1.png': [1280, 800],
  '/docs-create-course-step2.png': [1280, 800],
  '/docs-create-course-step3.png': [1280, 800],
  '/docs-create-course-success.png': [1280, 800],
  '/docs-dashboard.png': [1280, 800],
  '/docs-login.png': [1280, 720],
}

// SEO.4 FR-12/FR-13: build-time markdown output uses generated AVIF/WebP,
// reserves dimensions, and never eagerly transfers below-the-fold docs media.
md.renderer.rules.image = (tokens, idx) => {
  const token = tokens[idx]
  const src = String(token.attrGet('src') || '')
  const alt = md.utils.escapeHtml(token.content || '')
  const dimensions = LOCAL_IMAGE_DIMENSIONS[src]
  if (!dimensions || !src.endsWith('.png')) {
    const width = dimensions ? ` width="${dimensions[0]}" height="${dimensions[1]}"` : ''
    return `<img src="${md.utils.escapeHtml(src)}" alt="${alt}"${width} loading="lazy" decoding="async">`
  }
  const base = md.utils.escapeHtml(src.slice(0, -4))
  const [width, height] = dimensions
  return `<picture><source srcset="${base}.avif" type="image/avif"><source srcset="${base}.webp" type="image/webp"><img src="${base}.png" alt="${alt}" width="${width}" height="${height}" loading="lazy" decoding="async"></picture>`
}

try {
  md.enable(['table'])
} catch {
  /* table rule unavailable */
}

export function slugifyHeading(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
}

/** Render markdown to HTML (tables when supported by markdown-it build). */
export function renderMarkdown(source: string, opts?: { headingIds?: boolean }): string {
  const raw = String(source || '')
  if (!raw.trim()) return ''
  const directives = renderDirectives(raw)
  let html = md.render(directives.markdown)
  for (const [token, replacement] of directives.replacements) html = html.replace(token, replacement)
  html = html.replace(/\[\^(\d+)\]/g, '<sup class="content-citation"><a href="#source-$1" aria-label="Source $1">$1</a></sup>')
  html = html.replace(/<p>\[\^(\d+)\]:\s*<a href="([^"]+)"[^>]*>(.*?)<\/a>(.*?)<\/p>/g, '<p id="source-$1"><span aria-hidden="true">$1. </span><a href="$2" target="_blank" rel="noopener noreferrer">$3</a>$4</p>')

  if (opts?.headingIds !== false) {
    html = html.replace(/<h([2-4])>([\s\S]*?)<\/h\1>/gi, (_m, level, inner) => {
      const text = String(inner)
        .replace(/<[^>]+>/g, '')
        .replace(/\s+/g, ' ')
        .trim()
      const id = slugifyHeading(text)
      return `<h${level} id="${id}" tabindex="-1">${inner}</h${level}>`
    })
  }

  html = html.replace(
    /<a href="(https?:\/\/[^"]+)"/gi,
    '<a href="$1" target="_blank" rel="noopener noreferrer"',
  )

  return html
}

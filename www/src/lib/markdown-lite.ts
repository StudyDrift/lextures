/**
 * Minimal runtime markdown for API-driven course descriptions (interactive
 * routes only). Does not import markdown-it (SEO.4 FR-5).
 */
export function renderMarkdownLite(source: string): string {
  const text = String(source || '').trim()
  if (!text) return ''

  let s = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  s = s.replace(/```[\w]*\n([\s\S]*?)```/g, (_m, code) => `<pre><code>${code.trimEnd()}</code></pre>`)
  s = s.replace(/`([^`]+)`/g, '<code>$1</code>')
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  s = s.replace(/\*([^*]+)\*/g, '<em>$1</em>')
  s = s.replace(
    /\[([^\]]+)\]\((https?:\/\/[^)\s]+|\/[^)\s]*)\)/g,
    (_m, label, href) => {
      const external = String(href).startsWith('http')
      const attrs = external ? ' target="_blank" rel="nofollow ugc noopener"' : ''
      return `<a href="${href}"${attrs}>${label}</a>`
    },
  )
  s = s.replace(/(?:^|\n)((?:[-*] .+(?:\n|$))+)/g, block => {
    const items = block
      .trim()
      .split('\n')
      .map(line => line.replace(/^[-*] /, '').trim())
      .filter(Boolean)
      .map(item => `<li>${item}</li>`)
      .join('')
    return `\n<ul>${items}</ul>\n`
  })
  s = s.replace(/^### (.+)$/gm, '<h3>$1</h3>')
  s = s.replace(/^## (.+)$/gm, '<h2>$1</h2>')
  s = s.replace(/^# (.+)$/gm, '<h1>$1</h1>')
  const parts = s.split(/\n{2,}/)
  return parts
    .map(part => {
      const t = part.trim()
      if (!t) return ''
      if (/^<(h[1-6]|ul|ol|pre|blockquote|p|div)/i.test(t)) return t
      return `<p>${t.replace(/\n/g, '<br />')}</p>`
    })
    .filter(Boolean)
    .join('\n')
}

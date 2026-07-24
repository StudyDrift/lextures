const COLOR_STYLE_PROPS = new Set([
  'color',
  'background',
  'background-color',
  'background-image',
  '-webkit-text-fill-color',
  'caret-color',
])

const COLOR_HTML_ATTRS = ['color', 'bgcolor', 'background'] as const

function stripColorStylesFromDeclaration(styleText: string): string {
  return styleText
    .split(';')
    .map((part) => part.trim())
    .filter(Boolean)
    .filter((part) => {
      const prop = part.split(':')[0]?.trim().toLowerCase()
      return prop ? !COLOR_STYLE_PROPS.has(prop) : false
    })
    .join('; ')
}

/**
 * Removes pasted text/background colors from HTML so content inherits the editor theme.
 * Used by TipTap `transformPastedHTML` and sidebar HTML→Markdown paste.
 */
export function stripPastedHtmlColors(html: string): string {
  if (!html || typeof DOMParser === 'undefined') return html

  const doc = new DOMParser().parseFromString(html, 'text/html')
  const nodes = doc.body.querySelectorAll('*')

  for (const el of nodes) {
    if (!(el instanceof HTMLElement)) continue

    for (const attr of COLOR_HTML_ATTRS) {
      if (el.hasAttribute(attr)) el.removeAttribute(attr)
    }

    const style = el.getAttribute('style')
    if (!style) continue

    const next = stripColorStylesFromDeclaration(style)
    if (next) el.setAttribute('style', next)
    else el.removeAttribute('style')
  }

  return doc.body.innerHTML
}

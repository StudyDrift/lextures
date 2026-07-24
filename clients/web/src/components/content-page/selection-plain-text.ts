/**
 * Plain-text extraction for DOM Ranges used by the content page reader.
 *
 * Prefer capturing text at selection time and storing the string. After React
 * remounts markdown nodes, a live Range can be boundary-adjusted by the browser
 * so `range.toString()` returns only the first block.
 */

const BLOCK_SELECTOR =
  'p,div,li,h1,h2,h3,h4,h5,h6,tr,pre,blockquote,section,article,header,footer,ul,ol,table'

/** Insert newlines around block-level nodes so multi-paragraph copy keeps breaks. */
function normalizeClonedFragment(root: HTMLElement): void {
  for (const br of Array.from(root.querySelectorAll('br'))) {
    br.replaceWith(root.ownerDocument.createTextNode('\n'))
  }
  for (const el of Array.from(root.querySelectorAll(BLOCK_SELECTOR))) {
    if (!el.parentNode) continue
    el.parentNode.insertBefore(root.ownerDocument.createTextNode('\n'), el)
    if (el.nextSibling) {
      el.parentNode.insertBefore(root.ownerDocument.createTextNode('\n'), el.nextSibling)
    } else {
      el.parentNode.appendChild(root.ownerDocument.createTextNode('\n'))
    }
  }
}

/**
 * Serialize a Range to plain text with paragraph breaks.
 * Safe to call while the Range still points at live document nodes.
 */
export function plainTextFromRange(range: Range): string {
  if (range.collapsed) return ''
  try {
    const frag = range.cloneContents()
    const holder = range.commonAncestorContainer.ownerDocument?.createElement('div')
    if (!holder) return range.toString()
    holder.appendChild(frag)
    normalizeClonedFragment(holder)
    return (holder.textContent ?? '').replace(/\n{3,}/g, '\n\n').trim()
  } catch {
    try {
      return range.toString().trim()
    } catch {
      return ''
    }
  }
}

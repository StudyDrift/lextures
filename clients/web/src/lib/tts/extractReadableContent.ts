export type ReadableSentence = {
  text: string
  element: HTMLElement
}

const SKIP_SELECTORS = [
  'nav',
  'aside',
  'header',
  'footer',
  '[aria-hidden="true"]',
  '.sr-only',
  'script',
  'style',
  'code',
  '.katex',
  'math',
  '[data-reader-selection-ui]',
  '[data-read-aloud-controls]',
].join(',')

/** Split plain text into sentence chunks (client-side, no server round-trip). */
export function chunkSentences(text: string): string[] {
  const trimmed = text.replace(/\s+/g, ' ').trim()
  if (!trimmed) return []
  const parts = trimmed.split(/(?<=[.!?…])\s+(?=[A-Z0-9"([])|(?<=[.!?…])\s*$/u)
  return parts.map((p) => p.trim()).filter(Boolean)
}

function isSkippable(el: Element): boolean {
  if (el.closest(SKIP_SELECTORS)) return true
  if (el.closest('[role="navigation"]')) return true
  if (el.closest('[role="banner"]')) return true
  if (el.closest('[role="complementary"]')) return true
  return false
}

function blockText(el: Element): string {
  const tag = el.tagName.toLowerCase()
  if (tag === 'img') {
    const alt = el.getAttribute('alt')?.trim()
    return alt ? alt : ''
  }
  if (tag === 'pre' || tag === 'code' || el.classList.contains('katex')) {
    return tag === 'pre' || tag === 'code' ? 'Code block.' : 'Math equation.'
  }
  return (el.textContent ?? '').replace(/\s+/g, ' ').trim()
}

/**
 * Extract readable sentences from the main content region (role="main" or article).
 */
export function extractReadableContent(root: ParentNode = document): ReadableSentence[] {
  const container =
    root.querySelector<HTMLElement>('[data-content-reader]') ??
    root.querySelector<HTMLElement>('[role="main"]') ??
    root.querySelector<HTMLElement>('article')

  if (!container) return []

  const blockTags = new Set([
    'P',
    'LI',
    'H1',
    'H2',
    'H3',
    'H4',
    'H5',
    'H6',
    'TD',
    'TH',
    'BLOCKQUOTE',
    'FIGCAPTION',
    'DD',
    'PRE',
  ])

  const sentences: ReadableSentence[] = []
  const walker = document.createTreeWalker(container, NodeFilter.SHOW_ELEMENT)
  let node = walker.currentNode as Element | null
  while (node) {
    if (node instanceof HTMLElement && blockTags.has(node.tagName) && !isSkippable(node)) {
      const raw = blockText(node)
      if (raw) {
        for (const chunk of chunkSentences(raw)) {
          sentences.push({ text: chunk, element: node })
        }
      }
    }
    node = walker.nextNode() as Element | null
  }

  if (sentences.length === 0) {
    const fallback = blockText(container)
    if (fallback) {
      for (const chunk of chunkSentences(fallback)) {
        sentences.push({ text: chunk, element: container })
      }
    }
  }

  return sentences
}

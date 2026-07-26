import { Children, isValidElement, type ReactNode } from 'react'

/** Extract ```lex-tool fence body from a react-markdown `pre` child tree. */
export function extractLexToolFenceText(children: ReactNode): string | null {
  for (const child of Children.toArray(children)) {
    if (!isValidElement(child)) continue
    const props = child.props as { className?: string; children?: ReactNode }
    const className = props.className ?? ''
    if (className.includes('language-lex-tool')) {
      const raw = props.children
      if (typeof raw === 'string' || typeof raw === 'number') {
        return String(raw).replace(/\n$/, '')
      }
      return Children.toArray(raw)
        .map((c) => (typeof c === 'string' || typeof c === 'number' ? String(c) : ''))
        .join('')
        .replace(/\n$/, '')
    }
  }
  return null
}

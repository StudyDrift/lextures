import type { SyllabusSection } from '../../lib/courses-api'

/**
 * Invisible boundary between untitled sections so save → re-edit keeps them separate.
 * Only `##` headings otherwise define section breaks; untitled bodies would merge.
 * Stripped for student/reader display via {@link stripEditorSectionBoundaries}.
 */
export const EDITOR_SECTION_BOUNDARY = '<!-- lextures:section -->'

const BOUNDARY_LINE_RE = /^[ \t]*<!--\s*lextures:section\s*-->[ \t]*$/
/** Split before a section heading or an editor boundary marker. */
const SECTION_SPLIT_RE = /\n+(?=## |<!--\s*lextures:section\s*-->)/g
const BOUNDARY_PREFIX_RE = /^<!--\s*lextures:section\s*-->\s*/

/** Remove editor-only section markers before rendering Markdown to students. */
export function stripEditorSectionBoundaries(markdown: string): string {
  if (!markdown.includes('lextures:section')) return markdown
  return markdown
    .split('\n')
    .filter((line) => !BOUNDARY_LINE_RE.test(line))
    .join('\n')
    .replace(/\n{3,}/g, '\n\n')
}

/** Joins syllabus-style sections into one Markdown document (same shape the syllabus viewer uses). */
export function sectionsToMarkdown(sections: SyllabusSection[]): string {
  const parts: string[] = []

  for (const s of sections) {
    const h = s.heading.trim()
    const body = s.markdown.replace(/\s+$/u, '')
    if (!h && !body.trim()) continue

    if (h) {
      parts.push(`## ${h}\n\n${body}`)
      continue
    }

    // Untitled: first block is plain body (back-compat). Later untitled blocks need a
    // marker so re-edit does not merge them into the previous section.
    if (parts.length === 0) {
      parts.push(body)
    } else {
      parts.push(`${EDITOR_SECTION_BOUNDARY}\n\n${body}`)
    }
  }

  return parts.join('\n\n')
}

/**
 * Inverse of {@link sectionsToMarkdown} for loading a stored document into the block editor.
 * Best-effort: content that never used section headings stays a single block; ambiguous `##` in body can split.
 */
export function markdownToSectionsForEditor(markdown: string, newId: () => string): SyllabusSection[] {
  const trimmed = markdown.trim()
  if (!trimmed) return [{ id: newId(), heading: '', markdown: '' }]

  const parts = trimmed.split(SECTION_SPLIT_RE)
  const sections: SyllabusSection[] = []

  for (const part of parts) {
    let chunk = part.trim()
    if (!chunk) continue

    if (BOUNDARY_PREFIX_RE.test(chunk)) {
      chunk = chunk.replace(BOUNDARY_PREFIX_RE, '').trim()
      // Boundary with no body still represents a distinct empty untitled section.
      sections.push({
        id: newId(),
        heading: '',
        markdown: chunk,
      })
      continue
    }

    if (chunk.startsWith('## ')) {
      const firstNewline = chunk.indexOf('\n')
      if (firstNewline === -1) {
        sections.push({
          id: newId(),
          heading: chunk.slice(3).trim(),
          markdown: '',
        })
      } else {
        sections.push({
          id: newId(),
          heading: chunk.slice(3, firstNewline).trim(),
          markdown: chunk.slice(firstNewline + 1).trim(),
        })
      }
    } else {
      sections.push({
        id: newId(),
        heading: '',
        markdown: chunk,
      })
    }
  }

  return sections.length > 0 ? sections : [{ id: newId(), heading: '', markdown: '' }]
}

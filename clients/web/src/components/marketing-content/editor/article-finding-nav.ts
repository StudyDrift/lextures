import type { Editor } from '@tiptap/core'
import type { MarketingArticleAIDraft } from '../../../lib/marketing-content-ai-api'
import type { MarketingArticle, MarketingFinding } from '../../../lib/marketing-content-api'
import { directives, slugify } from './article-editor-utils'

export const METADATA_FINDING_PATHS = new Set([
  'title',
  'description',
  'author',
  'cluster',
  'primaryQuestion',
  'keywords',
  'updated',
])

const DIRECTIVE_BY_RULE: Record<string, string> = {
  'struct.key-takeaways': 'key-takeaways',
  'struct.answer-block': 'answer',
  'struct.faq-count': 'faq',
}

export function directiveIdForFinding(rule: string): string | null {
  return DIRECTIVE_BY_RULE[rule] ?? null
}

export function directiveTemplateForFinding(rule: string): string | null {
  const id = directiveIdForFinding(rule)
  return id ? (directives.find((item) => item.id === id)?.markdown ?? null) : null
}

export function bodyHasDirective(bodyMd: string, rule: string): boolean {
  const id = directiveIdForFinding(rule)
  if (!id) return false
  return new RegExp(`^:::\\s*${id}\\b`, 'm').test(bodyMd)
}

export function markdownLineRange(body: string, line: number): { start: number; end: number; text: string } | null {
  if (line < 1) return null
  const lines = body.split('\n')
  if (line > lines.length) return null
  let start = 0
  for (let index = 0; index < line - 1; index += 1) start += lines[index].length + 1
  const text = lines[line - 1] ?? ''
  return { start, end: start + text.length, text }
}

export function visibleLineSnippet(line: string): string {
  return line
    .replace(/^#{1,6}\s+/, '')
    .replace(/^:::\s*/, '')
    .replace(/^[-*+]\s+/, '')
    .replace(/^\d+\.\s+/, '')
    .replace(/^\|/, '')
    .trim()
}

export function selectTextareaLine(el: HTMLTextAreaElement, line: number): void {
  const range = markdownLineRange(el.value, line)
  el.focus()
  if (!range) return
  el.setSelectionRange(range.start, range.end)
  const parsed = Number.parseFloat(getComputedStyle(el).lineHeight)
  const lineHeight = Number.isFinite(parsed) ? parsed : 20
  el.scrollTop = Math.max(0, (line - 1) * lineHeight - el.clientHeight / 3)
}

export function jumpEditorToMarkdownLine(editor: Editor, bodyMd: string, line: number): boolean {
  const range = markdownLineRange(bodyMd, line)
  const snippet = visibleLineSnippet(range?.text ?? '')
  if (!snippet) {
    editor.chain().focus('start').scrollIntoView().run()
    return false
  }
  const needles = [snippet.slice(0, 80), snippet.slice(0, 24)].filter((value, index, all) => value.length >= 8 && all.indexOf(value) === index)
  const { doc } = editor.state
  let from = -1
  let to = -1
  doc.descendants((node, pos) => {
    if (from >= 0 || !node.isText || !node.text) return
    for (const needle of needles) {
      const idx = node.text.indexOf(needle)
      if (idx >= 0) {
        from = pos + idx
        to = from + needle.length
        return false
      }
    }
  })
  if (from < 0) {
    editor.chain().focus().scrollIntoView().run()
    return false
  }
  editor.chain().focus().setTextSelection({ from, to }).scrollIntoView().run()
  return true
}

export function findingLocationLabel(finding: MarketingFinding): string | null {
  if (finding.path) return finding.path
  if (finding.line && finding.line > 1) return `line ${finding.line}`
  return null
}

export function findingKey(finding: MarketingFinding, index: number): string {
  return `${finding.rule}-${finding.path ?? finding.line ?? index}`
}

type RepairableArticle = Pick<
  MarketingArticle,
  'kind' | 'title' | 'slug' | 'description' | 'bodyMd' | 'primaryQuestion' | 'cluster' | 'pillar' | 'keywords'
>

function draftLooksEmpty(draft: MarketingArticleAIDraft): boolean {
  return !draft.bodyMd.trim() && !draft.title.trim() && !draft.description.trim() && !draft.cluster.trim() && !draft.primaryQuestion.trim()
}

/** Send every finding in one repair request so a single model pass can fix them together. */
export async function solveAllFindings(input: {
  article: RepairableArticle
  findings: MarketingFinding[]
  repair: (article: RepairableArticle, findings: MarketingFinding[]) => Promise<MarketingArticleAIDraft>
  onProgress?: (total: number) => void
}): Promise<{ article: RepairableArticle; applied: number; error?: string }> {
  if (!input.findings.length) return { article: input.article, applied: 0 }
  input.onProgress?.(input.findings.length)
  let draft: MarketingArticleAIDraft
  try {
    draft = await input.repair(input.article, input.findings)
  } catch (error) {
    const detail = error instanceof Error ? error.message : 'Could not solve findings with AI.'
    return { article: input.article, applied: 0, error: detail }
  }
  if (draftLooksEmpty(draft)) {
    return { article: input.article, applied: 0, error: 'AI did not return a usable revision.' }
  }
  return { article: { ...input.article, ...mergeRepairDraft(input.article, draft) }, applied: input.findings.length }
}

/** Walk each finding in order, apply the returned draft, and continue with the updated article. */
export async function solveFindingsSequentially(input: {
  article: RepairableArticle
  findings: MarketingFinding[]
  repair: (article: RepairableArticle, finding: MarketingFinding) => Promise<MarketingArticleAIDraft>
  onProgress?: (index: number, total: number, finding: MarketingFinding) => void
}): Promise<{ article: RepairableArticle; applied: number; error?: string }> {
  let current = input.article
  let applied = 0
  for (let index = 0; index < input.findings.length; index += 1) {
    const finding = input.findings[index]
    input.onProgress?.(index, input.findings.length, finding)
    let draft: MarketingArticleAIDraft
    try {
      draft = await input.repair(current, finding)
    } catch (error) {
      const detail = error instanceof Error ? error.message : 'Could not solve findings with AI.'
      return { article: current, applied, error: applied ? `Stopped after ${applied} of ${input.findings.length} findings. ${detail}` : detail }
    }
    if (draftLooksEmpty(draft)) {
      return { article: current, applied, error: `AI did not return a usable revision for “${finding.message || finding.rule}”.` }
    }
    current = { ...current, ...mergeRepairDraft(current, draft) }
    applied += 1
  }
  return { article: current, applied }
}

/** Merge a repair draft onto the article, keeping current values when the model returns blanks. */
export function mergeRepairDraft(
  article: Pick<MarketingArticle, 'title' | 'slug' | 'description' | 'bodyMd' | 'primaryQuestion' | 'cluster' | 'pillar' | 'keywords'>,
  draft: MarketingArticleAIDraft,
): Partial<MarketingArticle> {
  const autoSlug = !article.slug || article.slug === slugify(article.title)
  const nextTitle = draft.title.trim() || article.title
  return {
    title: nextTitle,
    description: draft.description.trim() || article.description,
    bodyMd: draft.bodyMd.trim() || article.bodyMd,
    primaryQuestion: draft.primaryQuestion.trim() || article.primaryQuestion,
    cluster: draft.cluster.trim() || article.cluster,
    pillar: draft.pillar.trim() || article.pillar,
    keywords: draft.keywords.length ? draft.keywords : article.keywords,
    ...(autoSlug && (draft.slug.trim() || nextTitle) ? { slug: slugify(draft.slug || nextTitle) } : {}),
  }
}

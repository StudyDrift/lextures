import type { Components } from 'react-markdown'
import { forwardRef, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import rehypeKatex from 'rehype-katex'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import { Download } from 'lucide-react'
import 'katex/dist/katex.min.css'
import { FilePreview, FilePreviewBody } from '../file-preview'
import { CourseFileMarkdownImage } from './course-file-markdown-image'
import { normalizeMarkdownLists } from './normalize-markdown-lists'
import { remarkMergeAdjacentLists } from './remark-merge-adjacent-lists'
import type { SyllabusSection } from '../../lib/courses-api'
import type { ResolvedMarkdownTheme } from '../../lib/markdown-theme'
import { resolveMarkdownTheme } from '../../lib/markdown-theme'
import { useReducedData } from '../../context/reduced-data-context'
import { isMathRenderingEnabled } from '../../lib/math'
import { sectionsToMarkdown } from './syllabus-section-markdown'
import { authorizedFetch } from '../../lib/api'
import { createThemedMarkdownComponents } from '../markdown/markdown-themed-components'
import type { PluggableList } from 'unified'

// Matches Lextures course-file content URLs: /api/v1/courses/{code}/files/items/{id}/content
const lexturesCourseFileRe = /^\/api\/v1\/courses\/[^/]+\/files\/items\/[^/]+\/content/

function CourseFileLink({
  href,
  children,
}: {
  href: string
  children: React.ReactNode
  className?: string
  style?: React.CSSProperties
}) {
  const [previewOpen, setPreviewOpen] = useState(false)
  const { filePath, filename } = useMemo(() => {
    try {
      const u = new URL(href, window.location.origin)
      return {
        filePath: u.pathname,
        filename: u.searchParams.get('name') || String(children) || 'file',
      }
    } catch {
      return { filePath: href, filename: String(children) || 'file' }
    }
  }, [href, children])

  async function downloadFile() {
    try {
      const res = await authorizedFetch(filePath)
      if (!res.ok) return
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch { /* noop */ }
  }

  return (
    <div className="not-prose my-4 overflow-hidden rounded-xl border border-slate-200/90 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900/80">
      <div className="flex items-center justify-between gap-3 border-b border-slate-200/80 px-3 py-2 dark:border-neutral-700">
        <button
          type="button"
          onClick={() => setPreviewOpen(true)}
          className="min-w-0 truncate text-left text-sm font-medium text-indigo-700 hover:text-indigo-600 dark:text-indigo-300 dark:hover:text-indigo-200"
        >
          {filename}
        </button>
        <div className="flex shrink-0 items-center gap-1">
          <button
            type="button"
            className="inline-flex items-center rounded-lg px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
            onClick={() => setPreviewOpen(true)}
          >
            Full screen
          </button>
          <button
            type="button"
            className="inline-flex items-center rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
            title={`Download ${filename}`}
            aria-label={`Download ${filename}`}
            onClick={() => void downloadFile()}
          >
            <Download className="h-4 w-4" aria-hidden />
          </button>
        </div>
      </div>
      <div className="min-h-[min(28rem,60vh)] bg-slate-50 dark:bg-neutral-950/60">
        <FilePreviewBody filePath={filePath} filename={filename} mimeType={null} className="h-[min(28rem,60vh)]" />
      </div>
      {previewOpen ? (
        <FilePreview
          open={previewOpen}
          filePath={filePath}
          filename={filename}
          mimeType={null}
          onClose={() => setPreviewOpen(false)}
        />
      ) : null}
    </div>
  )
}

const katexRehypePlugins: PluggableList = [
  [rehypeKatex, { output: 'htmlAndMathml', strict: 'ignore' }],
]

function mathPluginsFor(enabled: boolean) {
  return enabled && isMathRenderingEnabled()
    ? {
        remark: [remarkMath],
        rehype: katexRehypePlugins,
      }
    : { remark: [], rehype: [] as PluggableList }
}

function createMarkdownComponents(
  theme: ResolvedMarkdownTheme,
  opts?: { useCourseFileImages?: boolean },
): Components {
  const o = theme.styleOverrides
  const c = theme.classes
  const base = createThemedMarkdownComponents(theme)
  return {
    ...base,
    a: ({ children, href }) => {
      if (href && lexturesCourseFileRe.test(href)) {
        return <CourseFileLink href={href}>{children}</CourseFileLink>
      }
      if (href?.startsWith('/courses/')) {
        return (
          <Link to={href} className={c.a} style={o.a}>
            {children}
          </Link>
        )
      }
      return (
        <a href={href} className={c.a} style={o.a} target="_blank" rel="noreferrer noopener">
          {children}
        </a>
      )
    },
    img: ({ src, alt }) =>
      opts?.useCourseFileImages ? (
        <CourseFileMarkdownImage
          src={src}
          alt={alt}
          className="lex-content-img max-h-[min(28rem,80vh)] w-auto max-w-full rounded-lg"
        />
      ) : (
        <img
          src={src ?? undefined}
          alt={alt ?? ''}
          className="lex-content-img max-h-[min(28rem,80vh)] w-auto max-w-full rounded-lg"
          loading="lazy"
        />
      ),
  }
}

const defaultResolved = resolveMarkdownTheme('classic', null)

type SyllabusMarkdownViewProps = {
  sections: SyllabusSection[]
  /** From GET course: `markdownThemePreset` + `markdownThemeCustom` */
  theme?: ResolvedMarkdownTheme
  courseCode?: string
}

type MarkdownArticleViewProps = {
  markdown: string
  emptyMessage?: string
  theme?: ResolvedMarkdownTheme
  /** When set, images under `/api/v1/.../course-files/.../content` load with the signed-in session. */
  courseCode?: string
}

/** Renders a single Markdown document with the same styling as the syllabus. */
function markdownLooksLikeMath(src: string): boolean {
  return /\$\$|\\\(|\\\[|\$[^$\s]/.test(src)
}

export const MarkdownArticleView = forwardRef<HTMLDivElement, MarkdownArticleViewProps>(
  function MarkdownArticleView(
    { markdown, emptyMessage = 'No content yet.', theme = defaultResolved, courseCode },
    ref,
  ) {
    const reducedData = useReducedData()
    const src = markdown.trim()
    const hasMath = useMemo(() => markdownLooksLikeMath(src), [src])
    const [userForcedMath, setUserForcedMath] = useState(false)
    const deferMath = reducedData && hasMath && !userForcedMath
    const mathPlugins = useMemo(() => mathPluginsFor(!deferMath), [deferMath])

    if (!src) {
      return (
        <div ref={ref} className={`syllabus-md ${theme.classes.article}`}>
          <p className="text-sm leading-relaxed text-slate-500 dark:text-neutral-400">{emptyMessage}</p>
        </div>
      )
    }
    const components = createMarkdownComponents(theme, { useCourseFileImages: Boolean(courseCode) })
    const normalized = normalizeMarkdownLists(markdown)
    return (
      <div ref={ref} className={`syllabus-md ${theme.classes.article}`}>
        {deferMath ? (
          <div className="mb-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200">
            <span className="font-medium">Math formatting is paused</span> to save data.{' '}
            <button
              type="button"
              className="ms-2 font-semibold text-indigo-600 underline decoration-indigo-300 hover:text-indigo-500 dark:text-indigo-400"
              onClick={() => setUserForcedMath(true)}
            >
              Load math
            </button>
          </div>
        ) : null}
        <ReactMarkdown
          remarkPlugins={[remarkGfm, remarkMergeAdjacentLists, ...mathPlugins.remark]}
          rehypePlugins={mathPlugins.rehype}
          components={components}
        >
          {normalized}
        </ReactMarkdown>
      </div>
    )
  },
)

export function SyllabusMarkdownView({ sections, theme = defaultResolved, courseCode }: SyllabusMarkdownViewProps) {
  const src = sectionsToMarkdown(sections)
  const reducedData = useReducedData()
  const hasMath = useMemo(() => markdownLooksLikeMath(src), [src])
  const [userForcedMath, setUserForcedMath] = useState(false)
  const deferMath = reducedData && hasMath && !userForcedMath
  const mathPlugins = useMemo(() => mathPluginsFor(!deferMath), [deferMath])

  if (!src.trim()) {
    return <p className="text-sm leading-relaxed text-slate-500">No syllabus content yet.</p>
  }
  const components = createMarkdownComponents(theme, { useCourseFileImages: Boolean(courseCode) })
  const normalized = normalizeMarkdownLists(src)
  return (
    <div className={`syllabus-md ${theme.classes.article}`}>
      {deferMath ? (
        <div className="mb-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200">
          <span className="font-medium">Math formatting is paused</span> to save data.{' '}
          <button
            type="button"
            className="ms-2 font-semibold text-indigo-600 underline decoration-indigo-300 hover:text-indigo-500 dark:text-indigo-400"
            onClick={() => setUserForcedMath(true)}
          >
            Load math
          </button>
        </div>
      ) : null}
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMergeAdjacentLists, ...mathPlugins.remark]}
        rehypePlugins={mathPlugins.rehype}
        components={components}
      >
        {normalized}
      </ReactMarkdown>
    </div>
  )
}

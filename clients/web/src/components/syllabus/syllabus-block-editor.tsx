import { FileText, Plus } from 'lucide-react'
import { formatNumber } from '../../lib/format'
import { marked } from 'marked'
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { Editor } from '@tiptap/core'
import {
  generateSyllabusSectionMarkdown,
  uploadCourseFile,
  type SyllabusSection,
} from '../../lib/courses-api'
import { sectionsToMarkdown, markdownToSectionsForEditor } from './syllabus-section-markdown'
import TurndownService from 'turndown'
import {
  BlockCanvas,
  BlockEditorProvider,
  BlockEditorShell,
  BlockFloatingToolbar,
  BlockFrame,
  CaretAnchoredToolbarPortal,
  EditorSidebar,
  MarkdownBodyEditor,
  MarkdownFormatToolbar,
  SidebarSection,
  useBlockEditor,
  type MarkdownEditKind,
} from '../editor/block-editor'
import { EquationEditorProvider, useEquationEditor } from '../editor/equation-editor-context'
import { isEquationEditorEnabled } from '../../lib/math'
import { BookLoader } from '../quiz/book-loader'
import { AltTextEnforcementProvider } from '../editor/block-editor/alt-text-enforcement-context'
import { AltTextWarningBanner } from '../editor/block-editor/alt-text-warning-banner'
import {
  altTextEnforcementFeatureEnabled,
  altTextHardBlockEnabled,
} from '../../lib/platform-features'
import { useSpeechToTextAvailability } from '../../hooks/use-speech-to-text-availability'
import {
  commitTipTapDictationFinal,
  insertTipTapDictationInterim,
  type InterimRange,
} from '../../lib/stt/text-insert'
import { summarizeSectionsAltText } from '../../lib/image-alt-validation'
import { toastMutationError } from '../../lib/lms-toast'
import { InputDialog } from '../input-dialog'

function newLocalId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `local-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

const MAX_SECTION_GENERATE_INSTRUCTIONS = 8000

type SyllabusBlockEditorProps = {
  sections: SyllabusSection[]
  onChange: (next: SyllabusSection[]) => void
  disabled?: boolean
  /** When set, section toolbars can generate Markdown via the course setup model. */
  courseCode?: string
  /** Content page / assignment structure item for equation audit events. */
  structureItemId?: string
  /** Sidebar copy: syllabus vs module page / assignment body. */
  documentVariant?: 'syllabus' | 'page'
  /**
   * With `documentVariant="page"`, replaces the default “Page” sidebar tab (stats + copy actions)
   * and renames that tab to “Settings”.
   */
  pageDocumentPanel?: ReactNode
  /** Syllabus only: require first-visit acceptance from students. */
  requireSyllabusAcceptance?: boolean
  onRequireSyllabusAcceptanceChange?: (next: boolean) => void
}

type ActiveField = { blockId: string; field: 'heading' | 'markdown' }

function SyllabusDocumentPanel({
  sections,
  onChange,
  documentVariant,
  requireSyllabusAcceptance,
  onRequireSyllabusAcceptanceChange,
}: {
  sections: SyllabusSection[]
  onChange: (next: SyllabusSection[]) => void
  documentVariant: 'syllabus' | 'page'
  requireSyllabusAcceptance?: boolean
  onRequireSyllabusAcceptanceChange?: (next: boolean) => void
}) {
  const { disabled } = useBlockEditor()
  const blocks = sections.length
  const chars = sections.reduce((n, s) => n + s.markdown.length + s.heading.length, 0)

  const [markdownCopiedFlash, setMarkdownCopiedFlash] = useState(0)
  const [htmlCopiedFlash, setHtmlCopiedFlash] = useState(0)
  const [pastedFlash, setPastedFlash] = useState(0)

  const copyMarkdown = useCallback(async () => {
    const text = sectionsToMarkdown(sections)
    try {
      await navigator.clipboard.writeText(text)
      setMarkdownCopiedFlash((n) => n + 1)
    } catch {
      /* ignore */
    }
  }, [sections, setMarkdownCopiedFlash])

  const copyHtml = useCallback(async () => {
    const md = sectionsToMarkdown(sections)
    const html = marked.parse(md, { async: false }) as string
    try {
      await navigator.clipboard.writeText(html)
      setHtmlCopiedFlash((n) => n + 1)
    } catch {
      /* ignore */
    }
  }, [sections, setHtmlCopiedFlash])

  const pasteFromClipboard = useCallback(async () => {
    try {
      const items = await navigator.clipboard.read()
      let markdown = ''

      for (const item of items) {
        if (item.types.includes('text/html')) {
          const blob = await item.getType('text/html')
          const html = await blob.text()
          const turndownService = new TurndownService({
            headingStyle: 'atx',
            hr: '---',
            bulletListMarker: '-',
            codeBlockStyle: 'fenced',
          })
          markdown = turndownService.turndown(html)
          break
        } else if (item.types.includes('text/plain')) {
          const blob = await item.getType('text/plain')
          markdown = await blob.text()
          break
        }
      }

      if (markdown) {
        const nextSections = markdownToSectionsForEditor(markdown, newLocalId)
        onChange(nextSections)
        setPastedFlash((n) => n + 1)
      }
    } catch {
      // Fallback to readText if read() is not supported or fails
      try {
        const text = await navigator.clipboard.readText()
        if (text) {
          const nextSections = markdownToSectionsForEditor(text, newLocalId)
          onChange(nextSections)
          setPastedFlash((n) => n + 1)
        }
      } catch {
        /* ignore */
      }
    }
  }, [onChange, setPastedFlash])

  return (
    <div className="space-y-4">
      <p className="text-[13px] leading-relaxed text-slate-600 dark:text-neutral-300">
        {documentVariant === 'page'
          ? 'Build this page from sections. Each section has an optional title and Markdown body, matching what students see when they open it.'
          : 'The syllabus is built from sections. Each section has an optional title and Markdown body, matching what students see on the course page.'}
      </p>
      {documentVariant === 'syllabus' &&
        requireSyllabusAcceptance !== undefined &&
        onRequireSyllabusAcceptanceChange && (
          <label className="flex cursor-pointer items-start gap-2.5 rounded-lg border border-slate-100 bg-slate-50/80 px-3 py-2.5 dark:border-neutral-700 dark:bg-neutral-800/50">
            <input
              type="checkbox"
              className="mt-0.5 h-4 w-4 shrink-0 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-900"
              checked={requireSyllabusAcceptance}
              disabled={disabled}
              onChange={(e) => onRequireSyllabusAcceptanceChange(e.target.checked)}
            />
            <span className="text-[13px] leading-snug text-slate-700 dark:text-neutral-200">
              Require students to review and accept this syllabus the first time they open the course.
            </span>
          </label>
        )}
      <dl className="space-y-0 text-[13px]">
        <div className="flex justify-between gap-3 border-b border-slate-100 py-2.5 dark:border-neutral-700">
          <dt className="text-slate-500 dark:text-neutral-400">Sections</dt>
          <dd className="font-medium text-slate-900 dark:text-neutral-100">{blocks}</dd>
        </div>
        <div className="flex justify-between gap-3 border-b border-slate-100 py-2.5 dark:border-neutral-700">
          <dt className="text-slate-500 dark:text-neutral-400">Characters</dt>
          <dd className="font-medium text-slate-900 dark:text-neutral-100">{formatNumber(chars)}</dd>
        </div>
      </dl>
      <div className="border-t border-slate-100 pt-3 dark:border-neutral-700">
        <h3 className="text-[13px] font-bold text-slate-900 dark:text-neutral-100">Actions</h3>
        <div className="mt-2 flex flex-col gap-1" aria-live="polite">
          <div className="flex items-center justify-between gap-3">
            <button
              type="button"
              onClick={copyMarkdown}
              className="min-w-0 flex-1 text-start text-[13px] text-slate-600 underline-offset-2 transition-[background-color,color,border-color] hover:text-indigo-600 hover:underline dark:text-neutral-300 dark:hover:text-indigo-400"
            >
              Copy as Markdown
            </button>
            <span className="pointer-events-none flex h-5 min-w-[3.25rem] shrink-0 items-center justify-end text-[13px]">
              {markdownCopiedFlash > 0 ? (
                <span
                  key={markdownCopiedFlash}
                  className="copy-action-copied-fade font-medium text-emerald-600 dark:text-emerald-400"
                  onAnimationEnd={() => setMarkdownCopiedFlash(0)}
                >
                  Copied
                </span>
              ) : null}
            </span>
          </div>
          <div className="flex items-center justify-between gap-3">
            <button
              type="button"
              onClick={copyHtml}
              className="min-w-0 flex-1 text-start text-[13px] text-slate-600 underline-offset-2 transition-[background-color,color,border-color] hover:text-indigo-600 hover:underline dark:text-neutral-300 dark:hover:text-indigo-400"
            >
              Copy as HTML
            </button>
            <span className="pointer-events-none flex h-5 min-w-[3.25rem] shrink-0 items-center justify-end text-[13px]">
              {htmlCopiedFlash > 0 ? (
                <span
                  key={htmlCopiedFlash}
                  className="copy-action-copied-fade font-medium text-emerald-600 dark:text-emerald-400"
                  onAnimationEnd={() => setHtmlCopiedFlash(0)}
                >
                  Copied
                </span>
              ) : null}
            </span>
          </div>
          <div className="mt-1 border-t border-slate-100 pt-1 dark:border-neutral-700/50">
            <div className="flex items-center justify-between gap-3">
              <button
                type="button"
                onClick={pasteFromClipboard}
                disabled={disabled}
                className="min-w-0 flex-1 text-start text-[13px] text-slate-600 underline-offset-2 transition-[background-color,color,border-color] hover:text-indigo-600 hover:underline disabled:no-underline disabled:opacity-40 dark:text-neutral-300 dark:hover:text-indigo-400"
              >
                Paste from Clipboard
              </button>
              <span className="pointer-events-none flex h-5 min-w-[3.25rem] shrink-0 items-center justify-end text-[13px]">
                {pastedFlash > 0 ? (
                  <span
                    key={pastedFlash}
                    className="copy-action-copied-fade font-medium text-emerald-600 dark:text-emerald-400"
                    onAnimationEnd={() => setPastedFlash(0)}
                  >
                    Pasted
                  </span>
                ) : null}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function SyllabusBlockPanel({
  section,
  index,
  updateAt,
}: {
  section: SyllabusSection
  index: number
  updateAt: (index: number, patch: Partial<SyllabusSection>) => void
}) {
  const { disabled } = useBlockEditor()
  const words = section.markdown.trim()
    ? section.markdown.trim().split(/\s+/).length
    : 0

  return (
    <div>
      <div className="mb-4 flex items-start gap-2 border-b border-slate-100 pb-4 dark:border-neutral-700">
        <span className="mt-0.5 flex h-8 w-8 items-center justify-center rounded border border-slate-200 bg-slate-50 text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
          <FileText className="h-4 w-4" aria-hidden />
        </span>
        <div className="min-w-0">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Section</h3>
          <p className="text-xs text-slate-500 dark:text-neutral-400">Optional heading plus Markdown content.</p>
        </div>
      </div>
      <SidebarSection title="Content" defaultOpen>
        <div>
          <label htmlFor={`syllabus-heading-${section.id}`} className="mb-1.5 block text-xs font-medium text-slate-600 dark:text-neutral-300">
            Heading
          </label>
          <input
            id={`syllabus-heading-${section.id}`}
            type="text"
            value={section.heading}
            onChange={(e) => updateAt(index, { heading: e.target.value })}
            disabled={disabled}
            placeholder="Optional"
            className="w-full rounded-md border border-slate-200 bg-white px-2.5 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-indigo-500 dark:focus:ring-indigo-500"
          />
        </div>
      </SidebarSection>
      <SidebarSection title="Markdown" defaultOpen>
        <p className="text-xs leading-relaxed text-slate-600 dark:text-neutral-300">
          Formatting is visual in the editor; stored content is Markdown for reliable rendering on the course page.
          Type @ to insert a link to a content page or assignment (the @ is not kept in the text).
        </p>
        <p className="text-xs text-slate-500 dark:text-neutral-400">
          ~{formatNumber(words)} word{words === 1 ? '' : 's'} ·{' '}
          {formatNumber(section.markdown.length)} characters
        </p>
      </SidebarSection>
    </div>
  )
}

function SyllabusSidebar({
  sections,
  onChange,
  updateAt,
  documentVariant,
  pageDocumentPanel,
  requireSyllabusAcceptance,
  onRequireSyllabusAcceptanceChange,
}: {
  sections: SyllabusSection[]
  onChange: (next: SyllabusSection[]) => void
  updateAt: (index: number, patch: Partial<SyllabusSection>) => void
  documentVariant: 'syllabus' | 'page'
  pageDocumentPanel?: ReactNode
  requireSyllabusAcceptance?: boolean
  onRequireSyllabusAcceptanceChange?: (next: boolean) => void
}) {
  const { selectedId } = useBlockEditor()
  const index = selectedId ? sections.findIndex((s) => s.id === selectedId) : -1
  const section = index >= 0 ? sections[index] : null

  const usePageSettings = documentVariant === 'page' && pageDocumentPanel != null

  return (
    <EditorSidebar
      documentLabel={usePageSettings ? 'Settings' : documentVariant === 'page' ? 'Page' : 'Syllabus'}
      blockLabel="Section"
      documentPanel={
        usePageSettings ? (
          pageDocumentPanel
        ) : (
          <SyllabusDocumentPanel
            sections={sections}
            onChange={onChange}
            documentVariant={documentVariant}
            requireSyllabusAcceptance={requireSyllabusAcceptance}
            onRequireSyllabusAcceptanceChange={onRequireSyllabusAcceptanceChange}
          />
        )
      }
      blockPanel={
        section ? (
          <SyllabusBlockPanel
            section={section}
            index={index}
            updateAt={updateAt}
          />
        ) : null
      }
      blockDisabled={!section}
      blockDisabledMessage="Click a section in the editor to change its settings here."
    />
  )
}

function BlockInsertionRow({ onAdd, disabled }: { onAdd: () => void; disabled?: boolean }) {
  return (
    <div className="relative py-6" onClick={(e) => e.stopPropagation()}>
      <div className="relative flex items-center justify-center">
        <div className="absolute inset-x-0 top-1/2 h-px bg-slate-300/80 dark:bg-neutral-600" aria-hidden />
        <button
          type="button"
          disabled={disabled}
          onClick={onAdd}
          className="relative z-10 flex h-9 w-9 items-center justify-center rounded-full border border-slate-300 bg-[#f0f0f0] text-slate-600 shadow-sm transition-[background-color,color,border-color] hover:border-slate-400 hover:bg-white hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:border-neutral-500 dark:hover:bg-neutral-700 dark:hover:text-neutral-50"
          aria-label="Add section"
        >
          <Plus className="h-5 w-5" strokeWidth={2} aria-hidden />
        </button>
      </div>
      <button
        type="button"
        disabled={disabled}
        onClick={onAdd}
        className="mt-4 w-full border-0 bg-transparent py-1 text-start text-[13px] text-slate-400 transition-[background-color,color,border-color] hover:text-slate-600 disabled:cursor-not-allowed disabled:opacity-50 dark:text-neutral-500 dark:hover:text-neutral-300"
      >
        Type / to add a section
      </button>
    </div>
  )
}

type SyllabusBlockEditorInnerProps = SyllabusBlockEditorProps

function SyllabusBlockEditorInner({
  sections,
  onChange,
  disabled,
  courseCode,
  documentVariant = 'syllabus',
  pageDocumentPanel,
  requireSyllabusAcceptance,
  onRequireSyllabusAcceptanceChange,
}: SyllabusBlockEditorInnerProps) {
  const { t } = useTranslation('common')
  const { selectedId, setSelectedId } = useBlockEditor()
  const equationEditor = useEquationEditor()
  const [activeField, setActiveField] = useState<ActiveField | null>(null)
  const editorRefs = useRef<Record<string, Editor | null>>({})
  const [generateSectionId, setGenerateSectionId] = useState<string | null>(null)
  const [generateInstructions, setGenerateInstructions] = useState('')
  const [generateSubmittingId, setGenerateSubmittingId] = useState<string | null>(null)
  const [generateError, setGenerateError] = useState<string | null>(null)
  const generateInputRef = useRef<HTMLInputElement>(null)
  const toolbarImageInputRef = useRef<HTMLInputElement>(null)
  const pendingToolbarImageSectionRef = useRef<string | null>(null)
  const mathToolbarAnchorRef = useRef<HTMLButtonElement | null>(null)
  const equationEditorEnabled = isEquationEditorEnabled()
  const altTextOn = altTextEnforcementFeatureEnabled()
  const altTextHardBlock = altTextHardBlockEnabled()
  const sttAvailability = useSpeechToTextAvailability(courseCode)
  const dictationInterimRef = useRef<Record<string, InterimRange>>({})
  const [linkDialogOpen, setLinkDialogOpen] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')
  const [linkSectionId, setLinkSectionId] = useState<string | null>(null)
  const altCoverage = useMemo(
    () => (altTextOn ? summarizeSectionsAltText(sections) : { withAlt: 0, total: 0, missing: [] }),
    [altTextOn, sections],
  )

  const imageInsertAttrs = useCallback(
    (path: string, fileName: string) => {
      if (altTextOn) {
        return { src: path, alt: '', decorative: false, altPending: true }
      }
      return {
        src: path,
        alt: (fileName || 'Image').replace(/[[\]]/g, '').slice(0, 200),
      }
    },
    [altTextOn],
  )

  const [editorVersion, setEditorVersion] = useState(0)
  const handleEditorChange = useCallback((sectionId: string, editor: Editor | null) => {
    editorRefs.current[sectionId] = editor
    setEditorVersion((v) => v + 1)
  }, [])

  const insertImagesIntoSection = useCallback(
    (sectionId: string, files: File[]) => {
      const editor = editorRefs.current[sectionId]
      if (!editor || !courseCode) return
      void (async () => {
        let pos = editor.state.selection.from
        for (const file of files) {
          if (!file.type.startsWith('image/')) continue
          try {
            const path = await uploadCourseFile(courseCode, file).then((r) => r.contentPath)
            editor
              .chain()
              .focus()
              .insertContentAt(pos, {
                type: 'image',
                attrs: imageInsertAttrs(path, file.name),
              })
              .run()
            pos = editor.state.selection.to
          } catch {
            /* ignore */
          }
        }
      })()
    },
    [courseCode, imageInsertAttrs],
  )

  /** Ignore stale field state when another block is selected (no sync effect). */
  const activeFieldResolved = useMemo((): ActiveField | null => {
    if (!activeField || !selectedId) return null
    if (activeField.blockId !== selectedId) return null
    return activeField
  }, [activeField, selectedId])

  const showMarkdownToolbar = useMemo(() => {
    const markdownFocused =
      activeFieldResolved?.field === 'markdown' && activeFieldResolved.blockId === selectedId
    const generateOpenForSelection =
      generateSectionId != null && generateSectionId === selectedId
    return Boolean(markdownFocused || generateOpenForSelection)
  }, [activeFieldResolved, selectedId, generateSectionId])

  useEffect(() => {
    if (generateSectionId && generateInputRef.current) {
      generateInputRef.current.focus()
    }
  }, [generateSectionId])

  useEffect(() => {
    if (generateSectionId != null && selectedId !== generateSectionId) {
      setGenerateSectionId(null)
      setGenerateInstructions('')
      setGenerateError(null)
    }
  }, [selectedId, generateSectionId])

  function updateAt(index: number, patch: Partial<SyllabusSection>) {
    const next = sections.map((s, i) => (i === index ? { ...s, ...patch } : s))
    onChange(next)
  }

  function removeAt(index: number) {
    if (sections.length <= 1) return
    onChange(sections.filter((_, i) => i !== index))
  }

  function move(index: number, dir: -1 | 1) {
    const j = index + dir
    if (j < 0 || j >= sections.length) return
    const next = [...sections]
    const t = next[index]!
    next[index] = next[j]!
    next[j] = t
    onChange(next)
  }

  function addSection() {
    onChange([...sections, { id: newLocalId(), heading: '', markdown: '' }])
  }

  function applyMarkdownForSection(sectionId: string, kind: MarkdownEditKind) {
    const editor = editorRefs.current[sectionId]
    if (!editor) return
    const chain = editor.chain().focus()

    switch (kind) {
      case 'bold':
        chain.toggleBold().run()
        break
      case 'italic':
        chain.toggleItalic().run()
        break
      case 'inlineCode':
        chain.toggleCode().run()
        break
      case 'codeBlock':
        chain.toggleCodeBlock().run()
        break
      case 'bulletList':
        chain.toggleBulletList().run()
        break
      case 'orderedList':
        chain.toggleOrderedList().run()
        break
      case 'link': {
        const prev = editor.getAttributes('link').href as string | undefined
        setLinkSectionId(sectionId)
        setLinkUrl(prev ?? t('dialogs.linkUrl.placeholder'))
        setLinkDialogOpen(true)
        break
      }
      default:
        break
    }
  }

  function toggleGeneratePanel(sectionId: string) {
    if (generateSectionId === sectionId) {
      setGenerateSectionId(null)
      setGenerateInstructions('')
      setGenerateError(null)
      return
    }
    setSelectedId(sectionId)
    setActiveField({ blockId: sectionId, field: 'markdown' })
    setGenerateSectionId(sectionId)
    setGenerateInstructions('')
    setGenerateError(null)
  }

  async function submitGenerate(section: SyllabusSection, index: number) {
    const text = generateInstructions.trim()
    if (!text || !courseCode) return
    setGenerateError(null)
    setGenerateSubmittingId(section.id)
    try {
      const { markdown } = await generateSyllabusSectionMarkdown(courseCode, {
        instructions: text,
        sectionHeading: section.heading,
        existingMarkdown: section.markdown,
      })
      updateAt(index, { markdown })
      setGenerateSectionId(null)
      setGenerateInstructions('')
    } catch (e) {
      setGenerateError(e instanceof Error ? e.message : 'Generation failed.')
    } finally {
      setGenerateSubmittingId(null)
    }
  }

  const caretAnchoredSectionIndex = useMemo(() => {
    if (!showMarkdownToolbar || !selectedId) return -1
    return sections.findIndex((s) => s.id === selectedId)
  }, [showMarkdownToolbar, selectedId, sections])

  const caretAnchoredEditor = useMemo(() => {
    if (caretAnchoredSectionIndex < 0 || !selectedId) return null
    void editorVersion
    return editorRefs.current[selectedId] ?? null
  }, [caretAnchoredSectionIndex, selectedId, editorVersion])

  function renderGeneratePanel(
    section: SyllabusSection,
    index: number,
    placement: 'inline' | 'anchored' = 'inline',
  ) {
    if (!courseCode) return null

    const inputClassName = [
      'w-full py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:outline-none disabled:opacity-60 dark:text-neutral-100 dark:placeholder:text-neutral-500',
      placement === 'anchored'
        ? 'border-0 bg-transparent px-2 focus:ring-0'
        : 'rounded-md border border-slate-200 bg-white px-2.5 focus:border-indigo-400 focus:ring-1 focus:ring-indigo-400 dark:border-neutral-600 dark:bg-neutral-950 dark:focus:border-indigo-500 dark:focus:ring-indigo-500',
      generateSubmittingId === section.id ? (placement === 'anchored' ? 'ps-2 pe-14' : 'ps-2.5 pe-14') : placement === 'anchored' ? 'px-2' : 'px-2.5',
    ].join(' ')

    const panel = (
      <>
        <label htmlFor={`section-generate-input-${section.id}`} className="sr-only">
          Instructions for generated section content
        </label>
        <div className="relative">
          <input
            ref={generateInputRef}
            id={`section-generate-input-${section.id}`}
            type="text"
            value={generateInstructions}
            maxLength={MAX_SECTION_GENERATE_INSTRUCTIONS}
            disabled={disabled || generateSubmittingId === section.id}
            onChange={(e) => setGenerateInstructions(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                void submitGenerate(section, index)
              }
              if (e.key === 'Escape') {
                e.stopPropagation()
                setGenerateSectionId(null)
                setGenerateInstructions('')
                setGenerateError(null)
              }
            }}
            placeholder={
              generateSubmittingId === section.id
                ? 'Generating…'
                : 'Describe what this section should say… (Enter to generate)'
            }
            className={inputClassName}
            aria-busy={generateSubmittingId === section.id ? true : undefined}
          />
          {generateSubmittingId === section.id ? (
            <div
              className="pointer-events-none absolute inset-y-0 end-1.5 flex items-center justify-end pe-[5px]"
              role="status"
              aria-live="polite"
              aria-label="Generating section"
            >
              <div className="translate-y-[10px]">
                <div className="inline-flex shrink-0 origin-center scale-[0.32]">
                  <BookLoader />
                </div>
              </div>
            </div>
          ) : null}
        </div>
        {generateError ? (
          <p className={`text-xs text-rose-600 dark:text-rose-400 ${placement === 'anchored' ? 'px-2 pb-1.5 pt-0.5' : 'mt-1.5'}`}>
            {generateError}
          </p>
        ) : null}
      </>
    )

    if (placement === 'anchored') {
      return (
        <div
          id={`section-generate-${section.id}`}
          data-generate-anchor
          className="border-t border-slate-200 py-1 dark:border-neutral-600"
          onClick={(e) => e.stopPropagation()}
        >
          {panel}
        </div>
      )
    }

    return (
      <div
        id={`section-generate-${section.id}`}
        data-generate-anchor
        className="mb-3 rounded-lg border border-slate-200 bg-slate-50/80 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900/40"
        onClick={(e) => e.stopPropagation()}
      >
        {panel}
      </div>
    )
  }

  function renderToolbar(section: SyllabusSection, index: number, embedded = false) {
    const showMarkdownTools = showMarkdownToolbar && selectedId === section.id
    const label = showMarkdownTools ? 'Markdown' : 'Section'
    const genBusy = generateSubmittingId === section.id

    return (
      <BlockFloatingToolbar
        embedded={embedded}
        icon={<FileText className="h-4 w-4" />}
        label={label}
        onMoveUp={() => move(index, -1)}
        onMoveDown={() => move(index, 1)}
        moveUpDisabled={index === 0}
        moveDownDisabled={index === sections.length - 1}
        onRemove={() => removeAt(index)}
        removeLabel="Remove section"
        disabled={disabled}
      >
        {showMarkdownTools && (
          <>
            <MarkdownFormatToolbar
              disabled={disabled || genBusy}
              onApply={(kind) => applyMarkdownForSection(section.id, kind)}
              mathInsert={
                equationEditorEnabled && equationEditor
                  ? {
                      onOpen: () => {
                        setSelectedId(section.id)
                        setActiveField({ blockId: section.id, field: 'markdown' })
                        const ed = editorRefs.current[section.id]
                        ed?.chain().focus().run()
                        if (ed) equationEditor.openInsert(ed)
                      },
                      registerMathAnchor: (node) => {
                        mathToolbarAnchorRef.current = node
                      },
                    }
                  : undefined
              }
              courseImage={
                courseCode
                  ? {
                      onPickClick: () => {
                        pendingToolbarImageSectionRef.current = section.id
                        setSelectedId(section.id)
                        setActiveField({ blockId: section.id, field: 'markdown' })
                        editorRefs.current[section.id]?.chain().focus().run()
                        requestAnimationFrame(() => toolbarImageInputRef.current?.click())
                      },
                      onFiles: (files) => {
                        setSelectedId(section.id)
                        setActiveField({ blockId: section.id, field: 'markdown' })
                        editorRefs.current[section.id]?.chain().focus().run()
                        insertImagesIntoSection(section.id, files)
                      },
                    }
                  : undefined
              }
              dictation={
                sttAvailability.enabled
                  ? {
                      language: sttAvailability.language,
                      accommodationTooltip: sttAvailability.accommodationTooltip,
                      onInterimResult: (text) => {
                        const editor = editorRefs.current[section.id]
                        if (!editor) return
                        dictationInterimRef.current[section.id] = insertTipTapDictationInterim(
                          editor,
                          text,
                          dictationInterimRef.current[section.id] ?? null,
                        )
                      },
                      onFinalResult: (text) => {
                        const editor = editorRefs.current[section.id]
                        if (!editor) return
                        commitTipTapDictationFinal(
                          editor,
                          text,
                          dictationInterimRef.current[section.id] ?? null,
                        )
                        dictationInterimRef.current[section.id] = null
                      },
                    }
                  : undefined
              }
            />
            {courseCode ? (
              <>
                <span className="mx-0.5 h-5 w-px shrink-0 bg-slate-200 dark:bg-neutral-600" aria-hidden />
                <button
                  type="button"
                  disabled={disabled || genBusy}
                  aria-expanded={generateSectionId === section.id}
                  aria-controls={`section-generate-${section.id}`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => toggleGeneratePanel(section.id)}
                  className="shrink-0 rounded px-2 py-1 text-xs font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-neutral-200 dark:hover:bg-neutral-700"
                >
                  Generate
                </button>
              </>
            ) : null}
          </>
        )}
      </BlockFloatingToolbar>
    )
  }

  return (
    <AltTextEnforcementProvider
      value={{
        enabled: altTextOn,
        hardBlock: altTextHardBlock,
        courseCode,
        onAiUnavailable: () => {
          toastMutationError('AI suggestion unavailable — please enter alt text manually.')
        },
      }}
    >
    <InputDialog
      open={linkDialogOpen}
      title={t('dialogs.linkUrl.title')}
      label={t('dialogs.linkUrl.label')}
      placeholder={t('dialogs.linkUrl.placeholder')}
      value={linkUrl}
      onValueChange={setLinkUrl}
      onConfirm={(url) => {
        const trimmed = url.trim()
        if (trimmed && linkSectionId) {
          const linkEditor = editorRefs.current[linkSectionId]
          linkEditor?.chain().focus().toggleLink({ href: trimmed }).run()
        }
        setLinkDialogOpen(false)
        setLinkSectionId(null)
      }}
      onClose={() => {
        setLinkDialogOpen(false)
        setLinkSectionId(null)
      }}
    />
    <BlockEditorShell
      sidebar={
        <SyllabusSidebar
          sections={sections}
          onChange={onChange}
          updateAt={updateAt}
          documentVariant={documentVariant}
          pageDocumentPanel={pageDocumentPanel}
          requireSyllabusAcceptance={requireSyllabusAcceptance}
          onRequireSyllabusAcceptanceChange={onRequireSyllabusAcceptanceChange}
        />
      }
    >
      <BlockCanvas className="pt-10">
        <input
          ref={toolbarImageInputRef}
          type="file"
          accept="image/png,image/jpeg,image/jpg,image/gif,image/webp"
          multiple
          className="sr-only"
          aria-hidden
          tabIndex={-1}
          onChange={(e) => {
            const sid = pendingToolbarImageSectionRef.current ?? selectedId
            pendingToolbarImageSectionRef.current = null
            const list = e.target.files
            e.target.value = ''
            if (!sid || !list?.length) return
            insertImagesIntoSection(sid, [...list])
          }}
        />
        {altTextOn ? (
          <AltTextWarningBanner coverage={altCoverage} hardBlock={altTextHardBlock} />
        ) : null}
        {caretAnchoredSectionIndex >= 0 ? (
          <CaretAnchoredToolbarPortal editor={caretAnchoredEditor} enabled>
            {generateSectionId === selectedId ? (
              <div className="flex w-max max-w-[calc(100vw-2rem)] flex-col overflow-hidden rounded-lg border border-slate-200 bg-white shadow-md shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40">
                {renderToolbar(sections[caretAnchoredSectionIndex]!, caretAnchoredSectionIndex, true)}
                {renderGeneratePanel(
                  sections[caretAnchoredSectionIndex]!,
                  caretAnchoredSectionIndex,
                  'anchored',
                )}
              </div>
            ) : (
              renderToolbar(sections[caretAnchoredSectionIndex]!, caretAnchoredSectionIndex)
            )}
          </CaretAnchoredToolbarPortal>
        ) : null}
        {sections.map((section, index) => {
          const caretAnchored = caretAnchoredSectionIndex === index
          return (
          <BlockFrame
            key={section.id}
            blockId={section.id}
            toolbar={caretAnchored ? undefined : renderToolbar(section, index)}
            inlineToolbar={!caretAnchored}
          >
            <div className="pb-8 pt-0.5">
              {generateSectionId === section.id && courseCode && !caretAnchored
                ? renderGeneratePanel(section, index)
                : null}
              <label className="sr-only" htmlFor={`canvas-heading-${section.id}`}>
                Section heading (optional)
              </label>
              <input
                id={`canvas-heading-${section.id}`}
                type="text"
                value={section.heading}
                onChange={(e) => updateAt(index, { heading: e.target.value })}
                onFocus={() => setActiveField({ blockId: section.id, field: 'heading' })}
                onBlur={(e) => {
                  const next = e.relatedTarget as HTMLElement | null
                  if (next?.closest('[data-toolbar-anchor]')) return
                  requestAnimationFrame(() => {
                    if (document.activeElement === e.currentTarget) return
                    setActiveField((prev) =>
                      prev?.blockId === section.id && prev.field === 'heading' ? null : prev,
                    )
                  })
                }}
                disabled={disabled}
                placeholder="Section heading (optional)"
                className="mb-1 w-full border-0 border-b border-dashed border-transparent bg-transparent pb-2 text-2xl font-semibold tracking-tight text-slate-900 placeholder:text-slate-400 focus:border-slate-300 focus:outline-none focus:ring-0 disabled:opacity-60 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-neutral-600"
              />
              <label className="sr-only" htmlFor={`canvas-md-${section.id}`}>
                Section body (Markdown)
              </label>
              <div id={`canvas-md-${section.id}`}>
                <MarkdownBodyEditor
                  sectionId={section.id}
                  value={section.markdown}
                  onChange={(markdown) => updateAt(index, { markdown })}
                  onFocus={() => setActiveField({ blockId: section.id, field: 'markdown' })}
                  onBlur={(e) => {
                    const next = e.relatedTarget as HTMLElement | null
                    if (next?.closest('[data-toolbar-anchor]')) return
                    if (next?.closest('[data-generate-anchor]')) return
                    requestAnimationFrame(() => {
                      setActiveField((prev) =>
                        prev?.blockId === section.id && prev.field === 'markdown' ? null : prev,
                      )
                    })
                  }}
                  disabled={disabled}
                  placeholder="Write this section in Markdown…"
                  onEditorChange={handleEditorChange}
                  courseCode={courseCode}
                  uploadCourseImage={
                    courseCode ? (file) => uploadCourseFile(courseCode, file).then((r) => r.contentPath) : undefined
                  }
                  onEquationSlash={
                    equationEditorEnabled && equationEditor
                      ? () => {
                          setSelectedId(section.id)
                          setActiveField({ blockId: section.id, field: 'markdown' })
                          const ed = editorRefs.current[section.id]
                          if (ed) equationEditor.openInsert(ed)
                        }
                      : undefined
                  }
                />
              </div>
            </div>
          </BlockFrame>
          )
        })}

        <BlockInsertionRow onAdd={addSection} disabled={disabled} />
      </BlockCanvas>
    </BlockEditorShell>
    </AltTextEnforcementProvider>
  )
}

export function SyllabusBlockEditor(props: SyllabusBlockEditorProps) {
  const validBlockIds = useMemo(() => props.sections.map((s) => s.id), [props.sections])

  return (
    <BlockEditorProvider disabled={props.disabled} validBlockIds={validBlockIds}>
      <EquationEditorProvider
        disabled={props.disabled}
        courseCode={props.courseCode}
        structureItemId={props.structureItemId}
      >
        <SyllabusBlockEditorInner {...props} />
      </EquationEditorProvider>
    </BlockEditorProvider>
  )
}

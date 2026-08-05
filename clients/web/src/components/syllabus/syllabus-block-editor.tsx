import { FileText, Plus } from 'lucide-react'
import { formatNumber } from '../../lib/format'
import { marked } from 'marked'
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type MutableRefObject,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'
import type { Editor } from '@tiptap/core'
import {
  createContentToolInstance,
  defaultContentToolConfig,
  deleteContentToolInstance,
  duplicateContentToolInstance,
  fetchContentToolsCatalog,
  fetchContentToolsInstances,
  fetchContentToolsSettings,
  patchContentToolInstance,
  uploadCourseFile,
  generateSyllabusSectionMarkdown,
  type ContentToolHostKind,
  type ContentToolInstance,
  type ContentToolManifest,
  type ContentToolsCatalogTool,
  type SyllabusSection,
} from '../../lib/courses-api'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import {
  ContentToolAuthoringProvider,
  type ContentToolAuthoringContextValue,
} from '../content-tools/authoring/content-tool-authoring-context'
import { AddSectionDropdown } from '../content-tools/authoring/add-section-dropdown'
import { ToolsDropdown } from '../content-tools/authoring/tools-dropdown'
import { ToolConfigPanel } from '../content-tools/authoring/tool-config-panel'
import { ToolPreviewModal } from '../content-tools/authoring/tool-preview-modal'
import { ToolDeleteDialog } from '../content-tools/authoring/tool-delete-dialog'
import {
  parseFencePayload,
  serializeLexToolFenceBlock,
} from '../../lib/content-tools/lex-tool-fence'
import { sectionsToMarkdown, markdownToSectionsForEditor } from './syllabus-section-markdown'
import { isSectionHeadingEnterToContentKey } from './section-heading-enter'
import TurndownService from 'turndown'
import { gfm } from 'turndown-plugin-gfm'
import { stripPastedHtmlColors } from '../editor/block-editor/strip-pasted-html-colors'
import {
  BlockCanvas,
  BlockEditorProvider,
  BlockEditorShell,
  BlockFloatingToolbar,
  BlockFrame,
  EditorSidebar,
  MarkdownBodyEditor,
  MarkdownFormatToolbar,
  SidebarSection,
  useBlockEditor,
  type MarkdownEditKind,
} from '../editor/block-editor'
import {
  MarkdownImageUploadModal,
  type CourseFileInsertItem,
} from '../editor/block-editor/markdown-image-upload-modal'
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
import { useConfirm } from '../use-confirm'

function newLocalId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `local-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

const MAX_SECTION_GENERATE_INSTRUCTIONS = 8000

const LEX_TOOL_FENCE_RE = /```lex-tool\s*\n([\s\S]*?)```/g

/** Instance IDs embedded in a section (TipTap nodes + markdown fences). */
function collectContentToolInstanceIds(
  section: SyllabusSection,
  editor: Editor | null | undefined,
): Set<string> {
  const ids = new Set<string>()
  if (editor) {
    editor.state.doc.descendants((node) => {
      if (node.type.name !== 'content_tool_block') return
      const id = String(node.attrs.instanceId ?? '').trim()
      if (id) ids.add(id)
    })
  }
  const md = section.markdown ?? ''
  LEX_TOOL_FENCE_RE.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = LEX_TOOL_FENCE_RE.exec(md)) !== null) {
    const payload = parseFencePayload(match[1] ?? '')
    if (payload?.instanceId) ids.add(payload.instanceId)
  }
  return ids
}

/**
 * Host document save should call `flush()` first so unsaved content-tool
 * config drafts (e.g. inline questions) are persisted with the page.
 */
export type ContentToolsFlushHandle = {
  flush: () => Promise<void>
}

type SyllabusBlockEditorProps = {
  sections: SyllabusSection[]
  onChange: (next: SyllabusSection[]) => void
  disabled?: boolean
  /** When set, section toolbars can generate Markdown via the course setup model. */
  courseCode?: string
  /** Content page / assignment structure item for equation audit events. */
  structureItemId?: string
  /** CT.2 — host surface for Content Tool instances. */
  hostKind?: ContentToolHostKind
  /**
   * Bump to re-fetch content-tool instances (e.g. after Build with AI creates tools
   * outside this editor’s insert path).
   */
  instancesReloadKey?: number
  /** Sidebar copy: syllabus vs module page / assignment body. */
  documentVariant?: 'syllabus' | 'page'
  /**
   * With `documentVariant="page"`, replaces the default “Page” sidebar stats/description
   * and renames that tab to “Settings”. Clipboard Actions (copy markdown/HTML, paste)
   * are still appended below this panel so quizzes and assignments match content pages.
   */
  pageDocumentPanel?: ReactNode
  /** Syllabus only: require first-visit acceptance from students. */
  requireSyllabusAcceptance?: boolean
  onRequireSyllabusAcceptanceChange?: (next: boolean) => void
  /**
   * Optional handle so host Save can persist all pending content-tool configs
   * (open panel + drafts from tools edited then closed).
   */
  contentToolsFlushRef?: MutableRefObject<ContentToolsFlushHandle | null>
}

type ActiveField = { blockId: string; field: 'heading' | 'markdown' }

/** Copy / paste document body actions shared by content pages, quizzes, and assignments. */
function DocumentClipboardActions({
  sections,
  onChange,
  className = 'border-t border-slate-100 pt-3 dark:border-neutral-700',
}: {
  sections: SyllabusSection[]
  onChange: (next: SyllabusSection[]) => void
  className?: string
}) {
  const { disabled } = useBlockEditor()
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
  }, [sections])

  const copyHtml = useCallback(async () => {
    const md = sectionsToMarkdown(sections)
    const html = marked.parse(md, { async: false }) as string
    try {
      await navigator.clipboard.writeText(html)
      setHtmlCopiedFlash((n) => n + 1)
    } catch {
      /* ignore */
    }
  }, [sections])

  const pasteFromClipboard = useCallback(async () => {
    try {
      const items = await navigator.clipboard.read()
      let markdown = ''

      for (const item of items) {
        if (item.types.includes('text/html')) {
          const blob = await item.getType('text/html')
          const html = stripPastedHtmlColors(await blob.text())
          const turndownService = new TurndownService({
            headingStyle: 'atx',
            hr: '---',
            bulletListMarker: '-',
            codeBlockStyle: 'fenced',
          })
          turndownService.use(gfm)
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
  }, [onChange])

  return (
    <div className={className}>
      <h3 className="text-[13px] font-bold text-slate-900 dark:text-neutral-100">Actions</h3>
      <div className="mt-2 flex flex-col gap-1" aria-live="polite">
        <div className="flex items-center justify-between gap-3">
          <button
            type="button"
            onClick={() => void copyMarkdown()}
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
            onClick={() => void copyHtml()}
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
              onClick={() => void pasteFromClipboard()}
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
  )
}

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

  return (
    <div data-focus-anchor="syllabus.section" className="space-y-4">
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
      <DocumentClipboardActions sections={sections} onChange={onChange} />
    </div>
  )
}

function SyllabusBlockPanel({
  section,
  index,
  total,
  updateAt,
}: {
  section: SyllabusSection
  index: number
  total: number
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
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            Section {index + 1} of {total}
          </h3>
          <p className="truncate text-xs text-slate-500 dark:text-neutral-400">
            {section.heading.trim() || 'Untitled section'}
          </p>
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
  configureInstance,
  courseCode,
  draftConfig,
  onDraftChange,
  onConfigSaved,
  onConfigClose,
  onManifestLoaded,
  manifests,
  toolConfigFlushRef,
}: {
  sections: SyllabusSection[]
  onChange: (next: SyllabusSection[]) => void
  updateAt: (index: number, patch: Partial<SyllabusSection>) => void
  documentVariant: 'syllabus' | 'page'
  pageDocumentPanel?: ReactNode
  requireSyllabusAcceptance?: boolean
  onRequireSyllabusAcceptanceChange?: (next: boolean) => void
  configureInstance: ContentToolInstance | null
  courseCode?: string
  draftConfig?: Record<string, unknown>
  onDraftChange: (instanceId: string, config: Record<string, unknown>) => void
  onConfigSaved: (instance: ContentToolInstance) => void
  onConfigClose: () => void
  onManifestLoaded: (manifest: ContentToolManifest) => void
  manifests: Record<string, ContentToolManifest>
  toolConfigFlushRef: MutableRefObject<(() => Promise<void>) | null>
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
          <div className="space-y-4">
            {pageDocumentPanel}
            <DocumentClipboardActions
              sections={sections}
              onChange={onChange}
              className="border-t border-slate-100 pt-4 dark:border-neutral-700"
            />
          </div>
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
        configureInstance && courseCode ? (
          <ToolConfigPanel
            courseCode={courseCode}
            instance={configureInstance}
            draftConfig={draftConfig}
            manifestCache={manifests[configureInstance.toolId] ?? null}
            onManifestLoaded={onManifestLoaded}
            onDraftChange={onDraftChange}
            onSaved={onConfigSaved}
            onClose={onConfigClose}
            flushRef={toolConfigFlushRef}
          />
        ) : section ? (
          <SyllabusBlockPanel
            section={section}
            index={index}
            total={sections.length}
            updateAt={updateAt}
          />
        ) : null
      }
      blockDisabled={!section && !configureInstance}
      blockDisabledMessage="Click any section to edit its settings here."
    />
  )
}

type SectionInsertProps = {
  disabled?: boolean
  onAddContent: () => void
  /** When set with content tools on, the control becomes Content | Tool. */
  contentTools?: {
    tools: ContentToolsCatalogTool[]
    loading?: boolean
    emptyCatalog?: boolean
    atMaxInstances?: boolean
    settingsHref?: string
    onAddTool: (toolId: string) => void
  }
}

/** Equal gap above/below the insert control. Fixed height so revealing the button never reflows. */
function SectionDivider({ disabled, onAddContent, contentTools }: SectionInsertProps) {
  return (
    <div
      className="group/divider relative flex items-center justify-center py-2"
      onClick={(e) => e.stopPropagation()}
    >
      <span
        className="absolute inset-x-0 top-1/2 h-px bg-slate-200 opacity-0 motion-safe:transition-opacity group-hover/divider:opacity-100 group-focus-within/divider:opacity-100 dark:bg-neutral-700"
        aria-hidden
      />
      {contentTools ? (
        <AddSectionDropdown
          variant="divider"
          label="Add section here"
          disabled={disabled}
          onAddContent={onAddContent}
          onAddTool={contentTools.onAddTool}
          tools={contentTools.tools}
          loading={contentTools.loading}
          emptyCatalog={contentTools.emptyCatalog}
          atMaxInstances={contentTools.atMaxInstances}
          settingsHref={contentTools.settingsHref}
        />
      ) : (
        <button
          type="button"
          disabled={disabled}
          onClick={onAddContent}
          className="pointer-events-none relative z-10 flex h-7 items-center gap-1.5 rounded-full border border-slate-200 bg-white px-3 text-xs font-medium text-slate-600 opacity-0 shadow-sm hover:text-slate-900 disabled:cursor-not-allowed motion-safe:transition-opacity group-hover/divider:pointer-events-auto group-hover/divider:opacity-100 group-focus-within/divider:pointer-events-auto group-focus-within/divider:opacity-100 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:text-neutral-50"
        >
          <Plus className="h-3.5 w-3.5" aria-hidden />
          Add section here
        </button>
      )}
    </div>
  )
}

function BlockInsertionRow({ disabled, onAddContent, contentTools }: SectionInsertProps) {
  return (
    <div className="pt-3" onClick={(e) => e.stopPropagation()}>
      {contentTools ? (
        <AddSectionDropdown
          variant="row"
          label="Add a section"
          disabled={disabled}
          onAddContent={onAddContent}
          onAddTool={contentTools.onAddTool}
          tools={contentTools.tools}
          loading={contentTools.loading}
          emptyCatalog={contentTools.emptyCatalog}
          atMaxInstances={contentTools.atMaxInstances}
          settingsHref={contentTools.settingsHref}
        />
      ) : (
        <button
          type="button"
          disabled={disabled}
          onClick={onAddContent}
          className="flex w-full items-center justify-center gap-2 rounded-xl border border-dashed border-slate-300 px-4 py-4 text-sm font-medium text-slate-600 motion-safe:transition-[background-color,color,border-color] hover:border-indigo-400 hover:bg-white hover:text-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:border-indigo-500 dark:hover:bg-neutral-900 dark:hover:text-indigo-400"
        >
          <Plus className="h-4 w-4" aria-hidden />
          Add a section
        </button>
      )}
    </div>
  )
}

type SyllabusBlockEditorInnerProps = SyllabusBlockEditorProps

function SyllabusBlockEditorInner({
  sections,
  onChange,
  disabled,
  courseCode,
  structureItemId,
  hostKind: hostKindProp,
  instancesReloadKey = 0,
  documentVariant = 'syllabus',
  pageDocumentPanel,
  requireSyllabusAcceptance,
  onRequireSyllabusAcceptanceChange,
  contentToolsFlushRef,
}: SyllabusBlockEditorInnerProps) {
  const { t } = useTranslation('common')
  const { selectedId, setSelectedId } = useBlockEditor()
  const { confirm, ConfirmDialogHost } = useConfirm()
  const equationEditor = useEquationEditor()
  const { contentToolsEnabled } = useCourseNavFeatures()
  const hostKind: ContentToolHostKind =
    hostKindProp ?? (documentVariant === 'syllabus' ? 'syllabus' : 'content_page')
  const [activeField, setActiveField] = useState<ActiveField | null>(null)
  const editorRefs = useRef<Record<string, Editor | null>>({})
  const [generateSectionId, setGenerateSectionId] = useState<string | null>(null)
  const [generateInstructions, setGenerateInstructions] = useState('')
  const [generateSubmittingId, setGenerateSubmittingId] = useState<string | null>(null)
  const [generateError, setGenerateError] = useState<string | null>(null)
  const [ctCatalog, setCtCatalog] = useState<ContentToolsCatalogTool[]>([])
  const [ctCatalogLoading, setCtCatalogLoading] = useState(false)
  const [ctInstances, setCtInstances] = useState<Record<string, ContentToolInstance>>({})
  const [ctManifests, setCtManifests] = useState<Record<string, ContentToolManifest>>({})
  const [ctMaxInstances, setCtMaxInstances] = useState(50)
  const [configureInstanceId, setConfigureInstanceId] = useState<string | null>(null)
  const [previewInstanceId, setPreviewInstanceId] = useState<string | null>(null)
  const [deleteInstanceId, setDeleteInstanceId] = useState<string | null>(null)
  const [ctInsertError, setCtInsertError] = useState<string | null>(null)
  /** Unsaved config drafts keyed by instance id (open panel + closed-without-save). */
  const toolDraftsRef = useRef<Record<string, Record<string, unknown>>>({})
  const [toolDraftsVersion, setToolDraftsVersion] = useState(0)
  const toolConfigFlushRef = useRef<(() => Promise<void>) | null>(null)
  const ctInstancesRef = useRef(ctInstances)
  ctInstancesRef.current = ctInstances
  const configureInstanceIdRef = useRef(configureInstanceId)
  configureInstanceIdRef.current = configureInstanceId
  const courseCodeRef = useRef(courseCode)
  courseCodeRef.current = courseCode
  const generateInputRef = useRef<HTMLInputElement>(null)
  const pendingToolbarImageSectionRef = useRef<string | null>(null)
  const [toolbarImageModalOpen, setToolbarImageModalOpen] = useState(false)
  const [toolbarImageInitialFiles, setToolbarImageInitialFiles] = useState<File[]>([])
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

  const handleEditorChange = useCallback((sectionId: string, editor: Editor | null) => {
    editorRefs.current[sectionId] = editor
  }, [])

  const focusSectionContent = useCallback(
    (sectionId: string) => {
      setSelectedId(sectionId)
      setActiveField({ blockId: sectionId, field: 'markdown' })
      // TipTap may not be registered on the first paint after mount; retry once.
      const focus = () =>
        editorRefs.current[sectionId]?.chain().focus('end', { scrollIntoView: false }).run()
      focus()
      if (!editorRefs.current[sectionId]) {
        requestAnimationFrame(focus)
      }
    },
    [setSelectedId],
  )

  const openToolbarImageModal = useCallback((sectionId: string, files?: File[]) => {
    pendingToolbarImageSectionRef.current = sectionId
    setSelectedId(sectionId)
    setActiveField({ blockId: sectionId, field: 'markdown' })
    editorRefs.current[sectionId]?.chain().focus().run()
    setToolbarImageInitialFiles(files ?? [])
    setToolbarImageModalOpen(true)
  }, [setSelectedId])

  const insertCourseFilesIntoSection = useCallback(
    (sectionId: string, items: CourseFileInsertItem[]) => {
      const editor = editorRefs.current[sectionId]
      if (!editor) return
      let pos = editor.state.selection.from
      for (const item of items) {
        if (item.mimeType.toLowerCase().startsWith('image/')) {
          editor
            .chain()
            .focus()
            .insertContentAt(pos, {
              type: 'image',
              attrs: imageInsertAttrs(item.contentPath, item.displayName),
            })
            .run()
        } else {
          const label = item.displayName.replace(/[[\]]/g, '') || 'File'
          editor
            .chain()
            .focus()
            .insertContentAt(pos, {
              type: 'text',
              text: label,
              marks: [{ type: 'link', attrs: { href: item.contentPath } }],
            })
            .run()
        }
        pos = editor.state.selection.to
      }
    },
    [imageInsertAttrs],
  )

  const handleToolbarImageInsert = useCallback(
    async (items: CourseFileInsertItem[]) => {
      const sid = pendingToolbarImageSectionRef.current ?? selectedId
      pendingToolbarImageSectionRef.current = null
      if (!sid) throw new Error('No section selected.')
      insertCourseFilesIntoSection(sid, items)
    },
    [insertCourseFilesIntoSection, selectedId],
  )

  useEffect(() => {
    if (!courseCode || !contentToolsEnabled) {
      setCtCatalog([])
      setCtInstances({})
      return
    }
    let cancelled = false
    setCtCatalogLoading(true)
    void Promise.all([
      fetchContentToolsCatalog(courseCode),
      fetchContentToolsSettings(courseCode).catch(() => null),
      fetchContentToolsInstances(courseCode, {
        ...(structureItemId ? { itemId: structureItemId } : {}),
        ...(hostKind ? { hostKind } : {}),
      }).catch(() => [] as ContentToolInstance[]),
    ])
      .then(([catalog, settings, instances]) => {
        if (cancelled) return
        setCtCatalog(catalog)
        if (settings) setCtMaxInstances(settings.maxInstancesPerItem)
        const map: Record<string, ContentToolInstance> = {}
        for (const inst of instances) map[inst.id] = inst
        setCtInstances(map)
      })
      .catch(() => {
        if (!cancelled) setCtCatalog([])
      })
      .finally(() => {
        if (!cancelled) setCtCatalogLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [courseCode, contentToolsEnabled, structureItemId, hostKind, instancesReloadKey])

  const activeInstanceCount = Object.values(ctInstances).filter((i) => i.status === 'active').length
  const atMaxInstances = activeInstanceCount >= ctMaxInstances

  const insertContentToolAtEditor = useCallback(
    async (toolId: string, sectionId: string) => {
      if (!courseCode) return
      setCtInsertError(null)
      const editor = editorRefs.current[sectionId]
      if (!editor) return
      try {
        const created = await createContentToolInstance(courseCode, {
          toolId,
          hostKind,
          structureItemId: hostKind === 'syllabus' ? null : structureItemId ?? null,
          sectionKey: hostKind === 'syllabus' ? sectionId : sectionId,
          config: defaultContentToolConfig(toolId),
        })
        setCtInstances((prev) => ({ ...prev, [created.id]: created }))
        const pos = editor.state.selection.from
        editor
          .chain()
          .focus()
          .insertContentAt(pos, {
            type: 'content_tool_block',
            attrs: {
              instanceId: created.id,
              toolId: created.toolId,
              courseCode,
            },
          })
          .run()
        setConfigureInstanceId(created.id)
        setSelectedId(sectionId)
      } catch (err) {
        setCtInsertError(err instanceof Error ? err.message : String(err))
        toastMutationError(err instanceof Error ? err.message : 'Could not insert tool.')
      }
    },
    [courseCode, hostKind, structureItemId, setSelectedId],
  )

  const removeToolBlockFromEditors = useCallback((instanceId: string) => {
    for (const editor of Object.values(editorRefs.current)) {
      if (!editor) continue
      const { state } = editor
      let tr = state.tr
      let modified = false
      state.doc.descendants((node, pos) => {
        if (node.type.name !== 'content_tool_block') return
        if (String(node.attrs.instanceId) !== instanceId) return
        tr = tr.delete(pos, pos + node.nodeSize)
        modified = true
      })
      if (modified) editor.view.dispatch(tr)
    }
  }, [])

  const authoringValue = useMemo((): ContentToolAuthoringContextValue | null => {
    if (!courseCode || !contentToolsEnabled) return null
    return {
      courseCode,
      instances: ctInstances,
      catalog: ctCatalog,
      manifests: ctManifests,
      onConfigure: (instanceId) => setConfigureInstanceId(instanceId),
      onPreview: (instanceId) => setPreviewInstanceId(instanceId),
      onDuplicate: (instanceId) => {
        void (async () => {
          try {
            const dup = await duplicateContentToolInstance(courseCode, instanceId)
            setCtInstances((prev) => ({ ...prev, [dup.id]: dup }))
            const sectionId = selectedId ?? sections[0]?.id
            const editor = sectionId ? editorRefs.current[sectionId] : null
            if (editor) {
              const pos = editor.state.selection.from
              editor
                .chain()
                .focus()
                .insertContentAt(pos, {
                  type: 'content_tool_block',
                  attrs: {
                    instanceId: dup.id,
                    toolId: dup.toolId,
                    courseCode,
                  },
                })
                .run()
            }
            setConfigureInstanceId(dup.id)
          } catch (err) {
            // Fallback: clone via create when duplicate route is not yet available.
            const src = ctInstances[instanceId]
            if (!src) {
              toastMutationError(err instanceof Error ? err.message : 'Duplicate failed.')
              return
            }
            try {
              const created = await createContentToolInstance(courseCode, {
                toolId: src.toolId,
                hostKind,
                structureItemId: hostKind === 'syllabus' ? null : structureItemId ?? null,
                sectionKey: selectedId ?? src.sectionKey ?? null,
                config: { ...src.config },
                title: src.title,
              })
              setCtInstances((prev) => ({ ...prev, [created.id]: created }))
              const sectionId = selectedId ?? sections[0]?.id
              const editor = sectionId ? editorRefs.current[sectionId] : null
              if (editor) {
                editor
                  .chain()
                  .focus()
                  .insertContentAt(editor.state.selection.from, {
                    type: 'content_tool_block',
                    attrs: {
                      instanceId: created.id,
                      toolId: created.toolId,
                      courseCode,
                    },
                  })
                  .run()
              }
              setConfigureInstanceId(created.id)
            } catch (err2) {
              toastMutationError(err2 instanceof Error ? err2.message : 'Duplicate failed.')
            }
          }
        })()
      },
      onDelete: (instanceId) => setDeleteInstanceId(instanceId),
      upsertInstance: (instance) => {
        setCtInstances((prev) => ({ ...prev, [instance.id]: instance }))
      },
      removeInstance: (instanceId) => {
        setCtInstances((prev) => {
          const next = { ...prev }
          delete next[instanceId]
          return next
        })
      },
      cacheManifest: (manifest) => {
        setCtManifests((prev) => ({ ...prev, [manifest.id]: manifest }))
      },
      getHostMarkdown: () => sectionsToMarkdown(sections),
    }
  }, [
    courseCode,
    contentToolsEnabled,
    ctInstances,
    ctCatalog,
    ctManifests,
    selectedId,
    sections,
    hostKind,
    structureItemId,
  ])

  const configureInstance = configureInstanceId ? ctInstances[configureInstanceId] ?? null : null
  const previewInstance = previewInstanceId ? ctInstances[previewInstanceId] ?? null : null
  // toolDraftsVersion forces re-read of toolDraftsRef when drafts change.
  const configureDraftConfig = useMemo(() => {
    if (!configureInstanceId) return undefined
    return toolDraftsRef.current[configureInstanceId]
  }, [configureInstanceId, toolDraftsVersion])

  const handleToolDraftChange = useCallback((instanceId: string, config: Record<string, unknown>) => {
    toolDraftsRef.current[instanceId] = config
    setToolDraftsVersion((v) => v + 1)
  }, [])

  const handleToolConfigSaved = useCallback((inst: ContentToolInstance) => {
    setCtInstances((prev) => ({ ...prev, [inst.id]: inst }))
    delete toolDraftsRef.current[inst.id]
    setToolDraftsVersion((v) => v + 1)
  }, [])

  /**
   * Persist every pending content-tool config draft. Host document save awaits this
   * so page Save does not discard open inline-question / tool edits.
   */
  const flushPendingContentTools = useCallback(async () => {
    const code = courseCodeRef.current
    if (!code) return

    // Open panel first (shows field errors in the sidebar when validation fails).
    if (toolConfigFlushRef.current) {
      await toolConfigFlushRef.current()
    }

    const drafts = { ...toolDraftsRef.current }
    const openId = configureInstanceIdRef.current
    for (const [id, config] of Object.entries(drafts)) {
      // Open panel flush already handled this id when dirty.
      if (id === openId) continue
      const saved = ctInstancesRef.current[id]?.config ?? {}
      try {
        if (JSON.stringify(config) === JSON.stringify(saved)) {
          delete toolDraftsRef.current[id]
          continue
        }
      } catch {
        /* fall through and PATCH */
      }
      const updated = await patchContentToolInstance(code, id, { config })
      setCtInstances((prev) => ({ ...prev, [id]: updated }))
      delete toolDraftsRef.current[id]
    }
    setToolDraftsVersion((v) => v + 1)
  }, [])

  // Publish flush handle for host Save buttons.
  useEffect(() => {
    if (!contentToolsFlushRef) return
    contentToolsFlushRef.current = { flush: flushPendingContentTools }
    return () => {
      contentToolsFlushRef.current = null
    }
  }, [contentToolsFlushRef, flushPendingContentTools])

  // Drop stale configure/preview ids when the instance map no longer has them
  // (e.g. section delete cleaned up tools, or external refresh).
  useEffect(() => {
    if (configureInstanceId && !ctInstances[configureInstanceId]) {
      setConfigureInstanceId(null)
    }
  }, [configureInstanceId, ctInstances])
  useEffect(() => {
    if (previewInstanceId && !ctInstances[previewInstanceId]) {
      setPreviewInstanceId(null)
    }
  }, [previewInstanceId, ctInstances])

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

  async function removeAt(index: number) {
    const section = sections[index]
    if (!section || sections.length <= 1) return

    // Only interrupt when there is written work to lose.
    const hasContent = Boolean(section.heading.trim() || section.markdown.trim())
    if (hasContent) {
      const name = section.heading.trim()
      const ok = await confirm({
        title: name ? `Delete “${name}”?` : `Delete section ${index + 1}?`,
        description: 'This removes the heading and everything written in this section.',
        confirmLabel: 'Delete section',
        variant: 'danger',
      })
      if (!ok) return
    }

    // Tools hosted in this section (editor nodes, fences, or sectionKey).
    const toolIds = collectContentToolInstanceIds(section, editorRefs.current[section.id])
    for (const inst of Object.values(ctInstances)) {
      if (inst.sectionKey === section.id) toolIds.add(inst.id)
    }

    onChange(sections.filter((_, i) => i !== index))
    setSelectedId(null)

    // Dismiss tool config/preview/delete UI for removed tools (or any open
    // config while deleting the selected section — avoids a stale sidebar).
    const wasSelected = selectedId === section.id
    if (
      configureInstanceId &&
      (toolIds.has(configureInstanceId) || wasSelected)
    ) {
      setConfigureInstanceId(null)
    }
    if (previewInstanceId && toolIds.has(previewInstanceId)) {
      setPreviewInstanceId(null)
    }
    if (deleteInstanceId && toolIds.has(deleteInstanceId)) {
      setDeleteInstanceId(null)
    }

    // Best-effort: drop local instances and permanently delete on the server so
    // removed sections do not leave tools counting toward maxInstances.
    if (toolIds.size > 0) {
      setCtInstances((prev) => {
        const next = { ...prev }
        for (const id of toolIds) delete next[id]
        return next
      })
      if (courseCode) {
        for (const id of toolIds) {
          void deleteContentToolInstance(courseCode, id, { permanent: true }).catch(() => {
            /* section already removed; cleanup is best-effort */
          })
        }
      }
    }
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

  /** Inserts at `index`; pass `sections.length` to append. */
  function addSectionAt(index: number) {
    const section: SyllabusSection = { id: newLocalId(), heading: '', markdown: '' }
    const next = [...sections]
    next.splice(index, 0, section)
    onChange(next)
    setSelectedId(section.id)
  }

  /** New section seeded with a content tool fence (Content Tools add-section flow). */
  async function addToolSectionAt(index: number, toolId: string) {
    if (!courseCode) return
    setCtInsertError(null)
    try {
      const sectionId = newLocalId()
      const created = await createContentToolInstance(courseCode, {
        toolId,
        hostKind,
        structureItemId: hostKind === 'syllabus' ? null : structureItemId ?? null,
        sectionKey: sectionId,
        config: defaultContentToolConfig(toolId),
      })
      setCtInstances((prev) => ({ ...prev, [created.id]: created }))
      const section: SyllabusSection = {
        id: sectionId,
        heading: '',
        markdown: serializeLexToolFenceBlock({
          instanceId: created.id,
          toolId: created.toolId,
          v: 1,
        }),
      }
      const next = [...sections]
      next.splice(index, 0, section)
      onChange(next)
      setSelectedId(section.id)
      setActiveField({ blockId: section.id, field: 'markdown' })
      setConfigureInstanceId(created.id)
    } catch (err) {
      setCtInsertError(err instanceof Error ? err.message : String(err))
      toastMutationError(err instanceof Error ? err.message : 'Could not insert tool.')
    }
  }

  const sectionInsertToolsBase =
    courseCode && contentToolsEnabled
      ? {
          tools: ctCatalog,
          loading: ctCatalogLoading,
          emptyCatalog: !ctCatalogLoading && ctCatalog.length === 0,
          atMaxInstances,
          settingsHref: `/courses/${encodeURIComponent(courseCode)}/settings`,
        }
      : null

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
      case 'table':
        chain.insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
        break
      default: {
        const _exhaustive: never = kind
        void _exhaustive
        break
      }
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

  function renderGeneratePanel(section: SyllabusSection, index: number) {
    if (!courseCode) return null

    const inputClassName = [
      'w-full py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:outline-none disabled:opacity-60 dark:text-neutral-100 dark:placeholder:text-neutral-500',
      'rounded-md border border-slate-200 bg-white px-2.5 focus:border-indigo-400 focus:ring-1 focus:ring-indigo-400 dark:border-neutral-600 dark:bg-neutral-950 dark:focus:border-indigo-500 dark:focus:ring-indigo-500',
      generateSubmittingId === section.id ? 'ps-2.5 pe-14' : 'px-2.5',
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
          <p className="mt-1.5 text-xs text-rose-600 dark:text-rose-400">{generateError}</p>
        ) : null}
      </>
    )

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

  function renderFormatToolbar(section: SyllabusSection, embedded = false) {
    const genBusy = generateSubmittingId === section.id

    return (
      <BlockFloatingToolbar embedded={embedded}>
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
                    openToolbarImageModal(section.id)
                  },
                  onFiles: (files) => {
                    openToolbarImageModal(section.id, files)
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
        {courseCode && contentToolsEnabled ? (
          <>
            <span className="mx-0.5 h-5 w-px shrink-0 bg-slate-200 dark:bg-neutral-600" aria-hidden />
            <ToolsDropdown
              tools={ctCatalog}
              loading={ctCatalogLoading}
              emptyCatalog={!ctCatalogLoading && ctCatalog.length === 0}
              atMaxInstances={atMaxInstances}
              disabled={disabled || genBusy}
              settingsHref={`/courses/${encodeURIComponent(courseCode)}/settings`}
              onSelect={(toolId) => {
                setSelectedId(section.id)
                setActiveField({ blockId: section.id, field: 'markdown' })
                void insertContentToolAtEditor(toolId, section.id)
              }}
            />
          </>
        ) : null}
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
      </BlockFloatingToolbar>
    )
  }

  const editorTree = (
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
          configureInstance={configureInstance}
          courseCode={courseCode}
          draftConfig={configureDraftConfig}
          onDraftChange={handleToolDraftChange}
          manifests={ctManifests}
          onManifestLoaded={(m) => setCtManifests((prev) => ({ ...prev, [m.id]: m }))}
          onConfigSaved={handleToolConfigSaved}
          onConfigClose={() => setConfigureInstanceId(null)}
          toolConfigFlushRef={toolConfigFlushRef}
        />
      }
    >
      <BlockCanvas>
        {courseCode ? (
          <MarkdownImageUploadModal
            open={toolbarImageModalOpen}
            onClose={() => {
              setToolbarImageModalOpen(false)
              setToolbarImageInitialFiles([])
              pendingToolbarImageSectionRef.current = null
            }}
            courseCode={courseCode}
            initialFiles={toolbarImageInitialFiles}
            onInsert={handleToolbarImageInsert}
          />
        ) : null}
        {altTextOn ? (
          <AltTextWarningBanner coverage={altCoverage} hardBlock={altTextHardBlock} />
        ) : null}
        {sections.map((section, index) => {
          const writing = showMarkdownToolbar && selectedId === section.id
          return (
          <div key={section.id}>
            {index > 0 && (
              <SectionDivider
                disabled={disabled}
                onAddContent={() => addSectionAt(index)}
                contentTools={
                  sectionInsertToolsBase
                    ? {
                        ...sectionInsertToolsBase,
                        onAddTool: (toolId) => {
                          void addToolSectionAt(index, toolId)
                        },
                      }
                    : undefined
                }
              />
            )}
            <BlockFrame
              blockId={section.id}
              positionLabel={`Section ${index + 1} of ${sections.length}`}
              onMoveUp={() => move(index, -1)}
              onMoveDown={() => move(index, 1)}
              moveUpDisabled={index === 0}
              moveDownDisabled={index === sections.length - 1}
              onRemove={() => void removeAt(index)}
              removeDisabledReason={
                sections.length <= 1 ? 'You need at least one section' : undefined
              }
              toolbar={writing ? renderFormatToolbar(section, true) : undefined}
            >
            <div>
              {generateSectionId === section.id && courseCode
                ? renderGeneratePanel(section, index)
                : null}
              {ctInsertError && selectedId === section.id ? (
                <p className="mb-2 text-xs text-rose-600 dark:text-rose-400" role="alert">
                  {ctInsertError}
                </p>
              ) : null}
              <label className="sr-only" htmlFor={`canvas-heading-${section.id}`}>
                Section heading (optional)
              </label>
              <input
                id={`canvas-heading-${section.id}`}
                type="text"
                value={section.heading}
                onChange={(e) => updateAt(index, { heading: e.target.value })}
                onFocus={() => setActiveField({ blockId: section.id, field: 'heading' })}
                onKeyDown={(e) => {
                  if (!isSectionHeadingEnterToContentKey(e)) return
                  e.preventDefault()
                  focusSectionContent(section.id)
                }}
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
                  contentToolsEnabled={contentToolsEnabled}
                  contentToolsCatalog={ctCatalog}
                  structureItemId={structureItemId}
                  hostKind={hostKind}
                  onInsertContentTool={(toolId) => {
                    setSelectedId(section.id)
                    setActiveField({ blockId: section.id, field: 'markdown' })
                    void insertContentToolAtEditor(toolId, section.id)
                  }}
                />
              </div>
            </div>
            </BlockFrame>
          </div>
          )
        })}

        <BlockInsertionRow
          disabled={disabled}
          onAddContent={() => addSectionAt(sections.length)}
          contentTools={
            sectionInsertToolsBase
              ? {
                  ...sectionInsertToolsBase,
                  onAddTool: (toolId) => {
                    void addToolSectionAt(sections.length, toolId)
                  },
                }
              : undefined
          }
        />
      </BlockCanvas>
    </BlockEditorShell>
    {courseCode ? (
      <>
        <ToolPreviewModal
          open={Boolean(previewInstance)}
          courseCode={courseCode}
          instance={previewInstance}
          onClose={() => setPreviewInstanceId(null)}
        />
        <ToolDeleteDialog
          open={Boolean(deleteInstanceId)}
          courseCode={courseCode}
          instanceId={deleteInstanceId}
          onClose={() => setDeleteInstanceId(null)}
          onArchive={async () => {
            if (!deleteInstanceId) return
            await patchContentToolInstance(courseCode, deleteInstanceId, { status: 'archived' })
            setCtInstances((prev) => {
              const cur = prev[deleteInstanceId]
              if (!cur) return prev
              return { ...prev, [deleteInstanceId]: { ...cur, status: 'archived' } }
            })
            removeToolBlockFromEditors(deleteInstanceId)
            if (configureInstanceId === deleteInstanceId) setConfigureInstanceId(null)
            setDeleteInstanceId(null)
          }}
          onDeletePermanently={async () => {
            if (!deleteInstanceId) return
            await deleteContentToolInstance(courseCode, deleteInstanceId, { permanent: true })
            setCtInstances((prev) => {
              const next = { ...prev }
              delete next[deleteInstanceId]
              return next
            })
            removeToolBlockFromEditors(deleteInstanceId)
            if (configureInstanceId === deleteInstanceId) setConfigureInstanceId(null)
            setDeleteInstanceId(null)
          }}
        />
      </>
    ) : null}
    {ConfirmDialogHost}
    </AltTextEnforcementProvider>
  )

  if (authoringValue) {
    return (
      <ContentToolAuthoringProvider value={authoringValue}>{editorTree}</ContentToolAuthoringProvider>
    )
  }
  return editorTree
}

/** CC.8 syllabus.section entity anchors attach per section row. */
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

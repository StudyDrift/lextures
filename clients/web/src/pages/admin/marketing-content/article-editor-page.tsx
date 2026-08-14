import { useCallback, useEffect, useRef, useState } from 'react'
import type { Editor } from '@tiptap/core'
import { ArrowLeft, Braces, ChevronDown, Eye, FileText, History, Keyboard, PanelRight, Save, Sparkles } from 'lucide-react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { MarkdownBodyEditor } from '../../../components/editor/block-editor'
import { ArticleMetadataPanel } from '../../../components/marketing-content/editor/article-metadata-panel'
import { ArticlePreview } from '../../../components/marketing-content/editor/article-preview'
import { TranslationSourcePane } from '../../../components/marketing-content/editor/translation-source-pane'
import { directives, slugify } from '../../../components/marketing-content/editor/article-editor-utils'
import { METADATA_FINDING_PATHS, findingKey, jumpEditorToMarkdownLine, selectTextareaLine, solveFindingsSequentially } from '../../../components/marketing-content/editor/article-finding-nav'
import { ArticleFindingsBar } from '../../../components/marketing-content/editor/article-findings-bar'
import { BuildArticleWithAiModal } from '../../../components/marketing-content/editor/build-article-with-ai-modal'
import { RevisionDrawer } from '../../../components/marketing-content/editor/revision-drawer'
import { Badge, Button, Checkbox, Dialog, EmptyState, InlineAlert, Input, Menu, SegmentedControl, Skeleton, Textarea, type MenuItem } from '../../../components/ui'
import { usePermissions } from '../../../context/use-permissions'
import { generateMarketingArticle, repairMarketingArticle, type MarketingArticleAIDraft } from '../../../lib/marketing-content-ai-api'
import { createMarketingArticle, createMarketingPreviewToken, getMarketingArticle, lintMarketingArticle, lintMetadataFromArticle, listMarketingAuthors, listMarketingCategories, listMarketingKnownPaths, MarketingConflictError, restoreMarketingRevision, transitionMarketingArticle, updateMarketingArticle, type MarketingArticle, type MarketingArticleWrite, type MarketingFinding } from '../../../lib/marketing-content-api'
import { createMarketingTranslation, listMarketingLocales, listMarketingTranslations, markMarketingTranslationSynced, type MarketingLocale, type MarketingTranslationLink } from '../../../lib/marketing-content-i18n-api'
import { PERM_MARKETING_CONTENT_AUTHOR, PERM_MARKETING_CONTENT_PUBLISH, PERM_MARKETING_CONTENT_REVIEW } from '../../../lib/rbac-api'
import { resolveMarketingPreviewUrl } from '../../../lib/marketing-site'

const emptyArticle = (kind: 'blog' | 'doc'): MarketingArticle => ({
  id: '', kind, slug: '', path: '', locale: 'en', title: '', description: '', bodyMd: '', status: 'draft', authorSlug: '', reviewerSlug: null,
  primaryQuestion: '', cluster: '', pillar: '', verifiedAgainst: '', keywords: [], relatedTo: [], roles: [], segments: [], citations: [], heroMediaId: null,
  noindex: false, canonicalOverride: null, updatedAt: '', revisionNo: 0,
})

type View = 'write' | 'split' | 'preview' | 'details'

function EditorMenu({ label, items, variant = 'ghost', icon, placement = 'bottom-start' }: { label: string; items: MenuItem[]; variant?: 'secondary' | 'ghost'; icon?: React.ReactNode; placement?: 'bottom-start' | 'bottom-end' }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLButtonElement>(null)
  return <><Button ref={ref} size="sm" variant={variant} className="min-h-6" aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen((value) => !value)}>{icon}{label}<ChevronDown aria-hidden className="h-3.5 w-3.5 opacity-70" /></Button><Menu open={open} onOpenChange={setOpen} anchorRef={ref} items={items} placement={placement} /></>
}

function writePayload(article: MarketingArticle): MarketingArticleWrite {
  return {
    kind: article.kind, slug: article.slug, locale: article.locale, categoryId: article.categoryId, title: article.title,
    description: article.description, bodyMd: article.bodyMd, authorSlug: article.authorSlug, reviewerSlug: article.reviewerSlug,
    reviewDueOn: article.reviewDueOn,
    primaryQuestion: article.primaryQuestion, cluster: article.cluster, pillar: article.pillar, verifiedAgainst: article.verifiedAgainst,
    keywords: article.keywords, relatedTo: article.relatedTo, roles: article.roles, segments: article.segments, citations: article.citations,
    heroMediaId: article.heroMediaId, noindex: article.noindex, canonicalOverride: article.canonicalOverride,
  }
}

type EditorPaneProps = {
  article: MarketingArticle
  canAuthor: boolean
  simple: boolean
  titleHighlight?: boolean
  titleError?: string
  onTitleChange: (title: string) => void
  onBodyChange: (bodyMd: string) => void
  onBlur: () => void
  onEditorChange: (sectionId: string, editor: Editor | null) => void
}

// Keep this component at module scope. Defining it inside ArticleEditorPage gives
// it a new component identity on every keystroke, which remounts TipTap while
// PureEditorContent is completing its own mount update.
function EditorPane({
  article,
  canAuthor,
  simple,
  titleHighlight,
  titleError,
  onTitleChange,
  onBodyChange,
  onBlur,
  onEditorChange,
}: EditorPaneProps) {
  const path = article.path || `${article.locale && article.locale !== 'en' ? `/${article.locale}` : ''}${article.kind === 'blog' ? '/blog/' : '/docs/…/'}${article.slug || '…'}`
  return <section className="min-w-0 overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-sm" aria-label="Article body editor">
    <div className="border-b border-border-subtle px-5 pb-4 pt-6 sm:px-9">
      <Input
        id="article-title"
        value={article.title}
        onChange={(event) => onTitleChange(event.target.value)}
        onBlur={onBlur}
        disabled={!canAuthor}
        aria-label="Article title"
        aria-invalid={titleError ? true : undefined}
        placeholder="Untitled article"
        className={`w-full border-0 bg-transparent p-0 text-2xl font-semibold tracking-tight text-fg-default shadow-none placeholder:text-fg-subtle disabled:opacity-60 sm:text-3xl ${titleHighlight ? 'rounded-lg ring-2 ring-accent-solid ring-offset-2 ring-offset-surface-raised' : ''}`}
      />
      {titleError ? <p className="mt-2 text-xs font-medium text-danger-fg">{titleError}</p> : null}
      <p className="mt-2 font-mono text-xs text-fg-muted">{path}</p>
    </div>
    <div className="px-5 py-5 sm:px-9">
      {simple
        ? <><label className="sr-only" htmlFor="article-markdown">Article body, markdown</label><Textarea id="article-markdown" dir="ltr" aria-describedby="article-findings" className="min-h-[52vh] resize-y border-0 bg-transparent p-0 font-mono text-sm shadow-none" value={article.bodyMd} onChange={(e) => onBodyChange(e.target.value)} onBlur={onBlur} /></>
        : <div className="min-h-[52vh]"><MarkdownBodyEditor sectionId="marketing-article" value={article.bodyMd} onChange={onBodyChange} onBlur={onBlur} disabled={!canAuthor} placeholder="Write the article… type / for blocks" onEditorChange={onEditorChange} /></div>}
    </div>
  </section>
}

export default function ArticleEditorPage() {
  const { articleId = '' } = useParams()
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const isNew = articleId === 'new'
  const { allows } = usePermissions()
  const canAuthor = allows(PERM_MARKETING_CONTENT_AUTHOR)
  const canReview = allows(PERM_MARKETING_CONTENT_REVIEW)
  const canPublish = allows(PERM_MARKETING_CONTENT_PUBLISH)
  const [article, setArticle] = useState(() => emptyArticle(params.get('kind') === 'doc' ? 'doc' : 'blog'))
  const [loading, setLoading] = useState(!isNew)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [lastSaved, setLastSaved] = useState<Date | null>(null)
  const [saveError, setSaveError] = useState('')
  const [findings, setFindings] = useState<MarketingFinding[]>([])
  const [score, setScore] = useState<number | null>(null)
  const [validating, setValidating] = useState(false)
  const [view, setView] = useState<View>('write')
  const [simple, setSimple] = useState(false)
  const [metadataOpen, setMetadataOpen] = useState(true)
  const [findingsOpen, setFindingsOpen] = useState(false)
  const [highlightField, setHighlightField] = useState<string | null>(null)
  const [revisionsOpen, setRevisionsOpen] = useState(false)
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
  const [buildAiOpen, setBuildAiOpen] = useState(false)
  const [solvingFindings, setSolvingFindings] = useState(false)
  const [solvingFindingKey, setSolvingFindingKey] = useState<string | null>(null)
  const [solveProgress, setSolveProgress] = useState('')
  const [solveError, setSolveError] = useState('')
  const [conflict, setConflict] = useState<MarketingConflictError['detail'] | null>(null)
  const [recovery, setRecovery] = useState<MarketingArticle | null>(null)
  const [transition, setTransition] = useState<string | null>(null)
  const [actionNote, setActionNote] = useState('')
  const [lintOverride, setLintOverride] = useState(false)
  const [scheduledFor, setScheduledFor] = useState('')
  const [categories, setCategories] = useState<Array<{ id: string; title: string; slug: string }>>([])
  const [authors, setAuthors] = useState<Array<{ slug: string; name: string }>>([])
  const [knownPaths, setKnownPaths] = useState<string[]>([])
  const [translations, setTranslations] = useState<MarketingTranslationLink[]>([])
  const [locales, setLocales] = useState<MarketingLocale[]>([])
  const [localesEnabled, setLocalesEnabled] = useState(false)
  const [source, setSource] = useState<MarketingArticle | null>(null)
  const [markingSynced, setMarkingSynced] = useState(false)
  const editorRef = useRef<Editor | null>(null)
  const articleRef = useRef(article)
  const dirtyRef = useRef(dirty)
  const openedFindingsRef = useRef(false)
  useEffect(() => { articleRef.current = article }, [article])
  useEffect(() => { dirtyRef.current = dirty }, [dirty])

  const patch = useCallback((value: Partial<MarketingArticle>) => {
    setArticle((old) => ({ ...old, ...value }))
    setDirty(true)
    setSaveError('')
  }, [])
  const patchMetadata = useCallback((value: Partial<MarketingArticle>) => {
    setArticle((old) => {
      const autoSlug = !old.slug || old.slug === slugify(old.title)
      return { ...old, ...value, ...(value.title !== undefined && autoSlug ? { slug: slugify(value.title) } : {}) }
    })
    setDirty(true)
  }, [])
  const changeTitle = useCallback((title: string) => patchMetadata({ title }), [patchMetadata])
  const changeBody = useCallback((bodyMd: string) => patch({ bodyMd }), [patch])
  const registerEditor = useCallback((_sectionId: string, editor: Editor | null) => {
    editorRef.current = editor
  }, [])

  useEffect(() => {
    void Promise.all([listMarketingCategories(), listMarketingAuthors(), listMarketingKnownPaths()]).then(([c, a, p]) => { setCategories(c ?? []); setAuthors(a ?? []); setKnownPaths(p.items ?? []); if (isNew && a?.[0]) setArticle((old) => ({ ...old, authorSlug: old.authorSlug || a[0].slug })) }).catch(() => undefined)
  }, [isNew])
  useEffect(() => {
    if (isNew) return
    const controller = new AbortController()
    void getMarketingArticle(articleId, controller.signal).then((value) => { setArticle(value); setLoading(false); const cached = sessionStorage.getItem(`mc:draft:${articleId}`); if (cached) { try { const cachedArticle = JSON.parse(cached) as MarketingArticle; if (cachedArticle.revisionNo === value.revisionNo) setRecovery(cachedArticle) } catch { /* ignore invalid recovery */ } } }).catch((e) => { setSaveError(String(e)); setLoading(false) })
    return () => controller.abort()
  }, [articleId, isNew])
  useEffect(() => {
    if (isNew || !article.id) return
    void Promise.all([listMarketingTranslations(article.id), listMarketingLocales()]).then(([group, loc]) => {
      setTranslations(group.items ?? [])
      setLocales(loc.items ?? [])
      setLocalesEnabled(Boolean(loc.localesEnabled))
    }).catch(() => undefined)
  }, [article.id, isNew, article.revisionNo])
  useEffect(() => {
    if (!article.sourceArticleId) { setSource(null); return }
    const controller = new AbortController()
    void getMarketingArticle(article.sourceArticleId, controller.signal).then(setSource).catch(() => setSource(null))
    return () => controller.abort()
  }, [article.sourceArticleId])

  useEffect(() => {
    if (loading || solvingFindings) return
    if (!isNew && dirty) sessionStorage.setItem(`mc:draft:${articleId}`, JSON.stringify(article))
    const timer = window.setTimeout(() => {
      setValidating(true)
      void lintMarketingArticle({ kind: article.kind, bodyMd: article.bodyMd, metadata: lintMetadataFromArticle(article) }).then((report) => {
        const next = report.findings ?? []
        setFindings(next)
        setScore(report.score)
        if (next.length && !openedFindingsRef.current) {
          openedFindingsRef.current = true
          setFindingsOpen(true)
        }
      }).catch(() => undefined).finally(() => setValidating(false))
    }, dirty ? 800 : 0)
    return () => window.clearTimeout(timer)
  }, [article, articleId, dirty, isNew, loading, solvingFindings])

  const saveArticle = useCallback(async () => {
    if (!dirtyRef.current || saving) return articleRef.current
    const current = articleRef.current
    if (!current.title.trim() || !current.slug.trim() || !current.authorSlug || (current.kind === 'doc' && !current.categoryId)) { setSaveError('Title, slug, author, and help-article category are required.'); return null }
    setSaving(true); setSaveError('')
    try {
      const saved = isNew ? await createMarketingArticle(writePayload(current)) : await updateMarketingArticle(articleId, current.revisionNo, writePayload(current))
      setArticle(saved); setDirty(false); setLastSaved(new Date()); sessionStorage.removeItem(`mc:draft:${saved.id}`)
      if (isNew) navigate(`/admin/marketing-content/${saved.id}`, { replace: true })
      return saved
    } catch (e) {
      if (e instanceof MarketingConflictError) setConflict(e.detail)
      else setSaveError(e instanceof Error ? e.message : 'Save failed.')
      return null
    } finally { setSaving(false) }
  }, [articleId, isNew, navigate, saving])

  useEffect(() => { if (!dirty || isNew) return; const timer = window.setInterval(() => void saveArticle(), 30000); return () => window.clearInterval(timer) }, [dirty, isNew, saveArticle])
  useEffect(() => { const warn = (e: BeforeUnloadEvent) => { if (dirtyRef.current) e.preventDefault() }; window.addEventListener('beforeunload', warn); return () => window.removeEventListener('beforeunload', warn) }, [])
  useEffect(() => {
    const keys = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') { e.preventDefault(); void saveArticle() }
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'p') { e.preventDefault(); setView((old) => old === 'preview' ? 'write' : 'preview') }
    }
    window.addEventListener('keydown', keys); return () => window.removeEventListener('keydown', keys)
  }, [saveArticle])

  const blocking = findings.filter((v) => v.severity === 'error')
  const doTransition = async () => {
    if (!transition) return
    const saved = dirty ? await saveArticle() : article
    if (!saved?.id) return
    try { const next = await transitionMarketingArticle(saved.id, transition, saved.revisionNo, { note: actionNote, lintOverride, ...(transition === 'schedule' && scheduledFor ? { scheduledFor: new Date(scheduledFor).toISOString() } : {}) }); setArticle((old) => ({ ...old, ...next })); setTransition(null); setActionNote(''); setLintOverride(false); setScheduledFor('') } catch (e) { setSaveError(String(e)); setTransition(null) }
  }
  const openPreview = async () => {
    const saved = dirty ? await saveArticle() : article
    if (!saved?.id) return
    try { const preview = await createMarketingPreviewToken(saved.id); window.open(resolveMarketingPreviewUrl(preview.url, saved.path), '_blank', 'noopener') } catch { setView('preview') }
  }
  const solveFindingsWithAI = useCallback(async () => {
    const snapshot = findings
    if (!snapshot.length || solvingFindings) return
    setSolvingFindings(true)
    setSolveError('')
    setSolveProgress('')
    setFindingsOpen(true)
    try {
      const result = await solveFindingsSequentially({
        article: articleRef.current,
        findings: snapshot,
        onProgress: (index, total, finding) => {
          setSolvingFindingKey(findingKey(finding, index))
          setSolveProgress(`Solving ${index + 1} of ${total}: ${finding.message || finding.rule}`)
        },
        repair: (current, finding) => repairMarketingArticle({
          kind: current.kind,
          existingTitle: current.title,
          existingBodyMd: current.bodyMd,
          description: current.description,
          primaryQuestion: current.primaryQuestion,
          cluster: current.cluster,
          pillar: current.pillar,
          keywords: current.keywords,
          knownPaths,
          findings: [{
            rule: finding.rule,
            severity: finding.severity,
            message: finding.message,
            line: finding.line,
            path: finding.path,
          }],
        }),
      })
      articleRef.current = { ...articleRef.current, ...result.article }
      setArticle((old) => ({ ...old, ...result.article }))
      if (result.applied) {
        setDirty(true)
        setSaveError('')
      }
      if (result.error) setSolveError(result.error)
    } finally {
      setSolvingFindings(false)
      setSolvingFindingKey(null)
      setSolveProgress('')
    }
  }, [findings, knownPaths, solvingFindings])
  const applyAIDraft = (draft: MarketingArticleAIDraft) => {
    setArticle((old) => {
      const autoSlug = !old.slug || old.slug === slugify(old.title)
      return {
        ...old,
        title: draft.title || old.title,
        description: draft.description,
        bodyMd: draft.bodyMd,
        primaryQuestion: draft.primaryQuestion,
        cluster: draft.cluster,
        pillar: draft.pillar,
        keywords: draft.keywords,
        ...(autoSlug && (draft.slug || draft.title) ? { slug: slugify(draft.slug || draft.title) } : {}),
      }
    })
    setDirty(true)
    setSaveError('')
  }
  const insertDirective = (markdown: string) => {
    if (simple || !editorRef.current) patch({ bodyMd: `${article.bodyMd}${article.bodyMd.endsWith('\n') || !article.bodyMd ? '' : '\n\n'}${markdown}` })
    else editorRef.current.commands.insertContent(markdown)
  }
  const revealFinding = (finding: MarketingFinding) => {
    const path = finding.path
    if (path && METADATA_FINDING_PATHS.has(path)) {
      setHighlightField(path)
      window.setTimeout(() => setHighlightField((current) => (current === path ? null : current)), 2500)
      if (path === 'title') {
        setView((current) => (current === 'preview' || current === 'details' ? 'write' : current))
        window.setTimeout(() => document.getElementById('article-title')?.focus(), 50)
        return
      }
      setMetadataOpen(true)
      if (window.matchMedia('(max-width: 767px)').matches) setView('details')
      return
    }
    setView((current) => (current === 'preview' || current === 'details' ? 'write' : current))
    const line = finding.line && finding.line > 0 ? finding.line : 1
    window.setTimeout(() => {
      if (simple) {
        const textarea = document.getElementById('article-markdown') as HTMLTextAreaElement | null
        if (textarea) selectTextareaLine(textarea, line)
        return
      }
      if (editorRef.current) jumpEditorToMarkdownLine(editorRef.current, articleRef.current.bodyMd, line)
    }, 50)
  }
  const restore = async (no: number) => { const next = await restoreMarketingRevision(article.id, no, article.revisionNo); setArticle(next); setDirty(false); setRevisionsOpen(false) }
  const addTranslation = async (locale: string) => {
    const saved = dirty ? await saveArticle() : article
    if (!saved?.id) return
    try {
      const created = await createMarketingTranslation(saved.id, locale)
      navigate(`/admin/marketing-content/${created.id}`)
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Could not create translation.')
    }
  }
  const markSynced = async () => {
    if (!article.id) return
    setMarkingSynced(true)
    try {
      const next = await markMarketingTranslationSynced(article.id)
      setArticle((old) => ({ ...old, ...next, stale: false }))
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Could not mark synced.')
    } finally { setMarkingSynced(false) }
  }

  if (loading) return <div className="space-y-3 p-6"><Skeleton className="h-16 w-full" /><Skeleton className="h-[60vh] w-full" /></div>
  if (!canAuthor && isNew) return <EmptyState icon={FileText} title={'You cannot create articles'} body="Author permission is required." />

  const status = article.liveStatus || article.status
  const statusTone = status === 'published' ? 'success' : status === 'scheduled' ? 'info' : status === 'in_review' ? 'warning' : 'neutral'
  const saveState = saving ? 'Saving…' : dirty ? 'Unsaved changes' : lastSaved ? `Saved ${lastSaved.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}` : article.updatedAt ? `Saved ${new Date(article.updatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}` : 'Not saved'
  const titleFinding = findings.find((finding) => finding.path === 'title')
  // The 'details' view is mobile-only, and 'split' is desktop-only; map each to
  // the nearest equivalent so a viewport change never leaves an empty canvas.
  const desktopSplit = view === 'split' || view === 'preview'
  const showDesktopEditor = view !== 'preview'
  const railOpen = metadataOpen || view === 'details'
  const editorPane = <EditorPane article={article} canAuthor={canAuthor} simple={simple} titleHighlight={highlightField === 'title'} titleError={titleFinding?.severity === 'error' ? titleFinding.message : undefined} onTitleChange={changeTitle} onBodyChange={changeBody} onBlur={() => void saveArticle()} onEditorChange={registerEditor} />
  const previewPane = <div className="min-w-0 overflow-auto rounded-2xl border border-border-default bg-surface-raised shadow-sm"><ArticlePreview title={article.title} body={article.bodyMd} dir={article.locale.startsWith('ar') ? 'rtl' : 'ltr'} /></div>
  const metadataPane = <ArticleMetadataPanel article={article} onChange={patchMetadata} categories={categories} authors={authors} knownPaths={knownPaths} isNew={isNew} findings={findings} highlightField={highlightField} canFillWithAI={canAuthor} />
  const missingLocales = locales.filter((loc) => loc.enabled && loc.code !== article.locale && !translations.some((row) => row.locale === loc.code))
  const translationItems: MenuItem[] = [
    ...translations.filter((row) => row.id !== article.id).map((row) => ({ id: row.id, label: `${row.locale.toUpperCase()} · ${row.status}${row.stale ? ' · stale' : ''}`, onSelect: () => navigate(`/admin/marketing-content/${row.id}`) })),
    ...(canAuthor && localesEnabled ? missingLocales.map((loc) => ({ id: `add-${loc.code}`, label: `Add ${loc.label} translation`, onSelect: () => void addTranslation(loc.code) })) : []),
  ]

  return <main className="min-w-0 pb-16">
    <div className="sticky top-0 z-20 border-b border-border-default bg-surface-raised/95 backdrop-blur supports-[backdrop-filter]:bg-surface-raised/80">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2 px-4 py-2.5 sm:px-6">
        <Link to="/admin/marketing-content" aria-label="Back to Marketing Content" className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-fg-muted hover:bg-surface-sunken hover:text-fg-default"><ArrowLeft className="h-4 w-4" /></Link>
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2">
            <h1 className="min-w-0 truncate text-sm font-semibold text-fg-default">{article.title || 'New article'}</h1>
            <Badge tone={statusTone}>{status}</Badge>
            <Badge tone="neutral">{(article.locale || 'en').toUpperCase()}</Badge>
            {article.stale ? <Badge tone="warning">Stale</Badge> : null}
          </div>
          <p className="truncate text-xs text-fg-muted"><span className="hidden sm:inline">{article.kind === 'blog' ? 'Blog post' : 'Help article'} · </span><span role="status">{saveState}</span></p>
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="secondary" className="min-h-6" onClick={() => void openPreview()}><Eye className="h-4 w-4" /> <span className="hidden sm:inline">Preview</span></Button>
          <Button size="sm" className="min-h-6" loading={saving} disabled={!dirty} onClick={() => void saveArticle()}><Save className="h-4 w-4" /> Save</Button>
          {!isNew ? <EditorMenu label="Actions" variant="secondary" placement="bottom-end" items={[
            ...(canAuthor ? [{ id: 'submit', label: 'Submit for review', onSelect: () => setTransition('submit') }] : []),
            ...(canReview ? [{ id: 'approve', label: 'Approve', onSelect: () => setTransition('approve') }, { id: 'request_changes', label: 'Request changes', onSelect: () => setTransition('request_changes') }] : []),
            ...(canPublish ? [{ id: 'publish', label: 'Publish', onSelect: () => setTransition('publish') }, { id: 'schedule', label: 'Schedule', onSelect: () => setTransition('schedule') }, { id: 'unpublish', label: 'Unpublish', onSelect: () => setTransition('unpublish') }, { id: 'archive', label: 'Archive', onSelect: () => setTransition('archive') }] : []),
          ]} /> : null}
          {!isNew && localesEnabled ? <EditorMenu label="Translations" variant="ghost" placement="bottom-end" items={translationItems.length ? translationItems : [{ id: 'none', label: 'This article is only available in this locale', disabled: true }]} /> : null}
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2 border-t border-border-subtle px-4 py-2 sm:px-6" role="toolbar" aria-label="Article editor tools">
        <SegmentedControl<View>
          size="sm"
          className="hidden md:inline-flex"
          value={view === 'details' ? 'write' : view}
          onChange={setView}
          options={[{ value: 'write', label: 'Write' }, { value: 'split', label: 'Split' }, { value: 'preview', label: 'Preview' }]}
        />
        <SegmentedControl<View>
          size="sm"
          className="md:hidden"
          value={view === 'split' ? 'write' : view}
          onChange={setView}
          options={[{ value: 'write', label: 'Write' }, { value: 'preview', label: 'Preview' }, { value: 'details', label: 'Details' }]}
        />
        <span aria-hidden className="mx-1 hidden h-5 w-px bg-border-default sm:block" />
        {canAuthor ? <Button size="sm" variant="ghost" className="min-h-6" onClick={() => setBuildAiOpen(true)}><Sparkles className="h-4 w-4" /> Build with AI</Button> : null}
        <EditorMenu label="Insert block" items={directives.map((v) => ({ id: v.id, label: v.label, onSelect: () => insertDirective(v.markdown) }))} />
        <Button size="sm" variant="ghost" className="min-h-6" aria-pressed={simple} onClick={() => setSimple((v) => !v)}><Braces className="h-4 w-4" /> <span className="hidden lg:inline">{simple ? 'Visual editor' : 'Markdown source'}</span></Button>
        <Button size="sm" variant="ghost" className="min-h-6" disabled={isNew} onClick={() => setRevisionsOpen(true)}><History className="h-4 w-4" /> <span className="hidden lg:inline">Revisions</span></Button>
        <Button size="sm" variant="ghost" className="min-h-6" onClick={() => setShortcutsOpen(true)}><Keyboard className="h-4 w-4" /> <span className="hidden lg:inline">Shortcuts</span></Button>
        <div className="ms-auto">
          <Button size="sm" variant="ghost" className="hidden min-h-6 md:inline-flex" aria-pressed={metadataOpen} onClick={() => setMetadataOpen((v) => !v)}><PanelRight className="h-4 w-4" /> Details</Button>
        </div>
      </div>
    </div>

    <div className="px-4 pt-4 sm:px-6">
      {saveError ? <InlineAlert tone="danger" className="mb-3"><span className="flex flex-wrap items-center gap-2"><strong>Could not save.</strong> {saveError}<Button size="sm" variant="secondary" onClick={() => void saveArticle()}>Retry</Button></span></InlineAlert> : null}
      {recovery ? <InlineAlert tone="warning" className="mb-3"><span className="flex flex-wrap items-center gap-2"><strong>Unsaved browser draft found.</strong><Button size="sm" onClick={() => { setArticle(recovery); setRecovery(null); setDirty(true) }}>Recover</Button><Button size="sm" variant="secondary" onClick={() => { sessionStorage.removeItem(`mc:draft:${articleId}`); setRecovery(null) }}>Discard</Button></span></InlineAlert> : null}
    </div>

    {/* Mobile: one pane at a time. */}
    <div className="px-4 pb-4 md:hidden">
      {view === 'details' ? <div className="rounded-2xl border border-border-default bg-surface-raised p-3">{metadataPane}</div> : view === 'preview' ? previewPane : editorPane}
    </div>

    {/* Desktop: canvas + metadata rail. */}
    <div className={`hidden gap-4 px-4 pb-4 md:grid sm:px-6 ${railOpen ? 'md:grid-cols-[minmax(0,1fr)_20rem] xl:grid-cols-[minmax(0,1fr)_22rem]' : 'md:grid-cols-1'}`}>
      <div className={`grid min-w-0 gap-4 ${desktopSplit && showDesktopEditor ? 'xl:grid-cols-2' : 'grid-cols-1'}`}>
        {source && showDesktopEditor ? <TranslationSourcePane source={source} stale={Boolean(article.stale)} canAuthor={canAuthor} marking={markingSynced} onMarkSynced={() => void markSynced()} /> : null}
        {showDesktopEditor ? editorPane : null}
        {desktopSplit ? previewPane : null}
      </div>
      {railOpen ? <aside aria-label="Article metadata" className="min-w-0"><div className="sticky top-[7rem] max-h-[calc(100dvh-9.5rem)] overflow-y-auto pb-2">{metadataPane}</div></aside> : null}
    </div>

    <ArticleFindingsBar
      findings={findings}
      score={score}
      validating={validating}
      open={findingsOpen}
      onOpenChange={setFindingsOpen}
      bodyMd={article.bodyMd}
      onSelectFinding={revealFinding}
      onInsertTemplate={insertDirective}
      canSolve={canAuthor}
      solving={solvingFindings}
      solvingFindingKey={solvingFindingKey}
      solveProgress={solveProgress}
      solveError={solveError}
      onSolveWithAI={() => void solveFindingsWithAI()}
    />

    <RevisionDrawer open={revisionsOpen} articleId={article.id} currentBody={article.bodyMd} onClose={() => setRevisionsOpen(false)} onRestore={restore} />
    <BuildArticleWithAiModal
      open={buildAiOpen}
      kind={article.kind}
      existingTitle={article.title}
      existingBodyMd={article.bodyMd}
      onClose={() => setBuildAiOpen(false)}
      onBuild={(prompt) => generateMarketingArticle({ prompt, kind: article.kind, existingTitle: article.title, existingBodyMd: article.bodyMd })}
      onBuilt={applyAIDraft}
    />
    <Dialog open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} title={'Keyboard shortcuts'} closeLabel="Close shortcuts"><dl className="grid grid-cols-[1fr_auto] gap-2 text-sm"><dt>Save</dt><dd>Ctrl/Cmd+S</dd><dt>Toggle preview</dt><dd>Ctrl/Cmd+Shift+P</dd><dt>Insert a block</dt><dd>Type / in the editor</dd><dt>Bold, italic and link</dt><dd>Use the shared editor toolbar</dd><dt>Move focus</dt><dd>Tab</dd></dl></Dialog>
    <Dialog open={Boolean(conflict)} onClose={() => undefined} hideClose closeOnBackdrop={false} closeOnEscape={false} title={'Someone else saved this article'} description={`A newer revision was saved${conflict?.updatedAt ? ` at ${new Date(conflict.updatedAt).toLocaleString()}` : ''}. Your local work is still safe.`} footer={<><Button variant="secondary" onClick={() => void navigator.clipboard.writeText(article.bodyMd)}>Keep mine (copy to clipboard)</Button><Button variant="secondary" onClick={() => window.open(`/admin/marketing-content/${article.id}`, '_blank', 'noopener')}>View their version</Button><Button onClick={() => window.location.reload()}>Reload</Button></>}></Dialog>
    <Dialog open={Boolean(transition)} onClose={() => setTransition(null)} title={`${transition?.replaceAll('_', ' ') ?? ''} article`} description={blocking.length && (transition === 'publish' || transition === 'schedule') ? `${blocking.length} blocking finding(s) must be resolved before publishing.` : 'This action is recorded in the article history.'} footer={<><Button variant="secondary" onClick={() => setTransition(null)}>Cancel</Button><Button disabled={Boolean((transition === 'schedule' && !scheduledFor) || (blocking.length && (transition === 'publish' || transition === 'schedule') && (!lintOverride || actionNote.trim().length < 20)))} onClick={() => void doTransition()}>Confirm</Button></>}><div className="space-y-3">{transition === 'schedule' ? <label className="block text-sm font-medium">Publish date and time<Input className="mt-1" type="datetime-local" value={scheduledFor} onChange={(e) => setScheduledFor(e.target.value)} /></label> : null}<label className="block text-sm font-medium">Change note<Textarea className="mt-1" value={actionNote} onChange={(e) => setActionNote(e.target.value)} /></label>{blocking.length && (transition === 'publish' || transition === 'schedule') && canPublish ? <Checkbox checked={lintOverride} onChange={(e) => setLintOverride(e.target.checked)} label="Override validation" description="Requires a justification of at least 20 characters in the change note." /> : null}</div></Dialog>
  </main>
}

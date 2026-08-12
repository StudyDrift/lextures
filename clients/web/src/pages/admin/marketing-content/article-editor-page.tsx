import { useCallback, useEffect, useRef, useState } from 'react'
import type { Editor } from '@tiptap/core'
import { ArrowLeft, Braces, ChevronDown, Eye, FileText, History, Keyboard, PanelRight, Save } from 'lucide-react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { MarkdownBodyEditor } from '../../../components/editor/block-editor'
import { ArticleMetadataPanel } from '../../../components/marketing-content/editor/article-metadata-panel'
import { ArticlePreview } from '../../../components/marketing-content/editor/article-preview'
import { TranslationSourcePane } from '../../../components/marketing-content/editor/translation-source-pane'
import { directives, slugify } from '../../../components/marketing-content/editor/article-editor-utils'
import { RevisionDrawer } from '../../../components/marketing-content/editor/revision-drawer'
import { Badge, Button, Checkbox, Dialog, EmptyState, InlineAlert, Input, Menu, SegmentedControl, Skeleton, Textarea, type MenuItem } from '../../../components/ui'
import { usePermissions } from '../../../context/use-permissions'
import { createMarketingArticle, createMarketingPreviewToken, getMarketingArticle, lintMarketingArticle, listMarketingAuthors, listMarketingCategories, listMarketingKnownPaths, MarketingConflictError, restoreMarketingRevision, transitionMarketingArticle, updateMarketingArticle, type MarketingArticle, type MarketingArticleWrite, type MarketingFinding } from '../../../lib/marketing-content-api'
import { createMarketingTranslation, listMarketingLocales, listMarketingTranslations, markMarketingTranslationSynced, type MarketingLocale, type MarketingTranslationLink } from '../../../lib/marketing-content-i18n-api'
import { PERM_MARKETING_CONTENT_AUTHOR, PERM_MARKETING_CONTENT_PUBLISH, PERM_MARKETING_CONTENT_REVIEW } from '../../../lib/rbac-api'

const emptyArticle = (kind: 'blog' | 'doc'): MarketingArticle => ({
  id: '', kind, slug: '', path: '', locale: 'en', title: '', description: '', bodyMd: '', status: 'draft', authorSlug: '', reviewerSlug: null,
  primaryQuestion: '', cluster: '', pillar: '', verifiedAgainst: '', keywords: [], relatedTo: [], roles: [], segments: [], citations: [], heroMediaId: null,
  noindex: false, canonicalOverride: null, updatedAt: '', revisionNo: 0,
})

type View = 'write' | 'split' | 'preview' | 'details'

function EditorMenu({ label, items, variant = 'ghost', icon }: { label: string; items: MenuItem[]; variant?: 'secondary' | 'ghost'; icon?: React.ReactNode }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLButtonElement>(null)
  return <><Button ref={ref} size="sm" variant={variant} aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen((value) => !value)}>{icon}{label}<ChevronDown aria-hidden className="h-3.5 w-3.5 opacity-70" /></Button><Menu open={open} onOpenChange={setOpen} anchorRef={ref} items={items} /></>
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
  onTitleChange,
  onBodyChange,
  onBlur,
  onEditorChange,
}: EditorPaneProps) {
  const path = article.path || `${article.locale && article.locale !== 'en' ? `/${article.locale}` : ''}${article.kind === 'blog' ? '/blog/' : '/docs/…/'}${article.slug || '…'}`
  return <section className="min-w-0 overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-sm" aria-label="Article body editor">
    <div className="border-b border-border-subtle px-5 pb-4 pt-6 sm:px-9">
      <Input
        value={article.title}
        onChange={(event) => onTitleChange(event.target.value)}
        onBlur={onBlur}
        disabled={!canAuthor}
        aria-label="Article title"
        placeholder="Untitled article"
        className="w-full border-0 bg-transparent p-0 text-2xl font-semibold tracking-tight text-fg-default shadow-none placeholder:text-fg-subtle disabled:opacity-60 sm:text-3xl"
      />
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
  const [revisionsOpen, setRevisionsOpen] = useState(false)
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
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
    if (!dirty || !article.bodyMd) return
    if (!isNew) sessionStorage.setItem(`mc:draft:${articleId}`, JSON.stringify(article))
    const timer = window.setTimeout(() => {
      setValidating(true)
      void lintMarketingArticle({ kind: article.kind, bodyMd: article.bodyMd, metadata: writePayload(article) }).then((report) => { setFindings(report.findings ?? []); setScore(report.score) }).catch(() => undefined).finally(() => setValidating(false))
    }, 800)
    return () => window.clearTimeout(timer)
  }, [article, articleId, dirty, isNew])

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
  const warnings = findings.filter((v) => v.severity !== 'error')
  const doTransition = async () => {
    if (!transition) return
    const saved = dirty ? await saveArticle() : article
    if (!saved?.id) return
    try { const next = await transitionMarketingArticle(saved.id, transition, saved.revisionNo, { note: actionNote, lintOverride, ...(transition === 'schedule' && scheduledFor ? { scheduledFor: new Date(scheduledFor).toISOString() } : {}) }); setArticle((old) => ({ ...old, ...next })); setTransition(null); setActionNote(''); setLintOverride(false); setScheduledFor('') } catch (e) { setSaveError(String(e)); setTransition(null) }
  }
  const openPreview = async () => {
    const saved = dirty ? await saveArticle() : article
    if (!saved?.id) return
    try { const preview = await createMarketingPreviewToken(saved.id); window.open(preview.url, '_blank', 'noopener') } catch { setView('preview') }
  }
  const insertDirective = (markdown: string) => {
    if (simple || !editorRef.current) patch({ bodyMd: `${article.bodyMd}${article.bodyMd.endsWith('\n') || !article.bodyMd ? '' : '\n\n'}${markdown}` })
    else editorRef.current.commands.insertContent(markdown)
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
  const scoreTone = score == null ? 'text-fg-muted' : score >= 80 ? 'text-success-fg' : score >= 50 ? 'text-warning-fg' : 'text-danger-fg'
  const scoreBar = score == null ? 'bg-border-strong' : score >= 80 ? 'bg-success-fg' : score >= 50 ? 'bg-warning-fg' : 'bg-danger-fg'
  // The 'details' view is mobile-only, and 'split' is desktop-only; map each to
  // the nearest equivalent so a viewport change never leaves an empty canvas.
  const desktopSplit = view === 'split' || view === 'preview'
  const showDesktopEditor = view !== 'preview'
  const railOpen = metadataOpen || view === 'details'
  const editorPane = <EditorPane article={article} canAuthor={canAuthor} simple={simple} onTitleChange={changeTitle} onBodyChange={changeBody} onBlur={() => void saveArticle()} onEditorChange={registerEditor} />
  const previewPane = <div className="min-w-0 overflow-auto rounded-2xl border border-border-default bg-surface-raised shadow-sm"><ArticlePreview title={article.title} body={article.bodyMd} dir={article.locale.startsWith('ar') ? 'rtl' : 'ltr'} /></div>
  const metadataPane = <ArticleMetadataPanel article={article} onChange={patchMetadata} categories={categories} authors={authors} knownPaths={knownPaths} isNew={isNew} />
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
          <Button size="sm" variant="secondary" onClick={() => void openPreview()}><Eye className="h-4 w-4" /> <span className="hidden sm:inline">Preview</span></Button>
          <Button size="sm" loading={saving} disabled={!dirty} onClick={() => void saveArticle()}><Save className="h-4 w-4" /> Save</Button>
          {!isNew ? <EditorMenu label="Actions" variant="secondary" items={[
            ...(canAuthor ? [{ id: 'submit', label: 'Submit for review', onSelect: () => setTransition('submit') }] : []),
            ...(canReview ? [{ id: 'approve', label: 'Approve', onSelect: () => setTransition('approve') }, { id: 'request_changes', label: 'Request changes', onSelect: () => setTransition('request_changes') }] : []),
            ...(canPublish ? [{ id: 'publish', label: 'Publish', onSelect: () => setTransition('publish') }, { id: 'schedule', label: 'Schedule', onSelect: () => setTransition('schedule') }, { id: 'unpublish', label: 'Unpublish', onSelect: () => setTransition('unpublish') }, { id: 'archive', label: 'Archive', onSelect: () => setTransition('archive') }] : []),
          ]} /> : null}
          {!isNew && localesEnabled ? <EditorMenu label="Translations" variant="ghost" items={translationItems.length ? translationItems : [{ id: 'none', label: 'This article is only available in this locale', disabled: true }]} /> : null}
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
        <EditorMenu label="Insert block" items={directives.map((v) => ({ id: v.id, label: v.label, onSelect: () => insertDirective(v.markdown) }))} />
        <Button size="sm" variant="ghost" aria-pressed={simple} onClick={() => setSimple((v) => !v)}><Braces className="h-4 w-4" /> <span className="hidden lg:inline">{simple ? 'Visual editor' : 'Markdown source'}</span></Button>
        <Button size="sm" variant="ghost" disabled={isNew} onClick={() => setRevisionsOpen(true)}><History className="h-4 w-4" /> <span className="hidden lg:inline">Revisions</span></Button>
        <Button size="sm" variant="ghost" onClick={() => setShortcutsOpen(true)}><Keyboard className="h-4 w-4" /> <span className="hidden lg:inline">Shortcuts</span></Button>
        <div className="ms-auto">
          <Button size="sm" variant="ghost" className="hidden md:inline-flex" aria-pressed={metadataOpen} onClick={() => setMetadataOpen((v) => !v)}><PanelRight className="h-4 w-4" /> Details</Button>
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

    <section id="article-findings" aria-live="polite" className="sticky bottom-0 z-20 border-t border-border-default bg-surface-raised/95 backdrop-blur supports-[backdrop-filter]:bg-surface-raised/80">
      <Button type="button" variant="ghost" aria-expanded={findingsOpen} onClick={() => setFindingsOpen((v) => !v)} className="flex h-auto w-full items-center justify-start gap-3 rounded-none px-4 py-2.5 text-start sm:px-6">
        <span className="text-xs font-medium text-fg-muted">Quality</span>
        <span aria-hidden className="h-1.5 w-24 overflow-hidden rounded-full bg-surface-sunken sm:w-40"><span className={`block h-full rounded-full motion-safe:transition-[width] ${scoreBar}`} style={{ width: `${Math.max(0, Math.min(100, score ?? 0))}%` }} /></span>
        <span className={`text-sm font-semibold tabular-nums ${scoreTone}`}>{validating ? 'checking…' : score == null ? '—' : Math.round(score)}</span>
        <span className="hidden text-xs text-fg-muted sm:inline">/ 100 · publish floor 80</span>
        <span className="ms-auto flex items-center gap-2">
          {blocking.length ? <Badge tone="danger">{blocking.length} blocking</Badge> : null}
          {warnings.length ? <Badge tone="warning">{warnings.length} warning{warnings.length === 1 ? '' : 's'}</Badge> : null}
          {!findings.length ? <span className="text-xs text-fg-muted">{score == null ? 'Not checked yet' : 'No findings'}</span> : null}
          <ChevronDown aria-hidden className={`h-4 w-4 text-fg-muted motion-safe:transition-transform ${findingsOpen ? '' : 'rotate-180'}`} />
        </span>
      </Button>
      {findingsOpen ? <div className="max-h-64 overflow-y-auto border-t border-border-subtle px-4 py-3 sm:px-6">
        {findings.length
          ? <ul className="space-y-1.5 text-sm">{findings.map((finding, index) => <li key={`${finding.rule}-${index}`} className="flex flex-wrap items-baseline gap-x-2">
            <Badge tone={finding.severity === 'error' ? 'danger' : 'warning'}>{finding.severity === 'error' ? 'Error' : 'Warning'}</Badge>
            {finding.line ? <span className="font-mono text-xs text-fg-muted">line {finding.line}</span> : null}
            <span className="text-fg-default">{finding.message}</span>
            <span className="font-mono text-xs text-fg-subtle">{finding.rule}</span>
          </li>)}</ul>
          : <p className="text-sm text-fg-muted">Nothing to fix. Findings appear here as you write.</p>}
      </div> : null}
    </section>

    <RevisionDrawer open={revisionsOpen} articleId={article.id} currentBody={article.bodyMd} onClose={() => setRevisionsOpen(false)} onRestore={restore} />
    <Dialog open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} title={'Keyboard shortcuts'} closeLabel="Close shortcuts"><dl className="grid grid-cols-[1fr_auto] gap-2 text-sm"><dt>Save</dt><dd>Ctrl/Cmd+S</dd><dt>Toggle preview</dt><dd>Ctrl/Cmd+Shift+P</dd><dt>Insert a block</dt><dd>Type / in the editor</dd><dt>Bold, italic and link</dt><dd>Use the shared editor toolbar</dd><dt>Move focus</dt><dd>Tab</dd></dl></Dialog>
    <Dialog open={Boolean(conflict)} onClose={() => undefined} hideClose closeOnBackdrop={false} closeOnEscape={false} title={'Someone else saved this article'} description={`A newer revision was saved${conflict?.updatedAt ? ` at ${new Date(conflict.updatedAt).toLocaleString()}` : ''}. Your local work is still safe.`} footer={<><Button variant="secondary" onClick={() => void navigator.clipboard.writeText(article.bodyMd)}>Keep mine (copy to clipboard)</Button><Button variant="secondary" onClick={() => window.open(`/admin/marketing-content/${article.id}`, '_blank', 'noopener')}>View their version</Button><Button onClick={() => window.location.reload()}>Reload</Button></>}></Dialog>
    <Dialog open={Boolean(transition)} onClose={() => setTransition(null)} title={`${transition?.replaceAll('_', ' ') ?? ''} article`} description={blocking.length && (transition === 'publish' || transition === 'schedule') ? `${blocking.length} blocking finding(s) must be resolved before publishing.` : 'This action is recorded in the article history.'} footer={<><Button variant="secondary" onClick={() => setTransition(null)}>Cancel</Button><Button disabled={Boolean((transition === 'schedule' && !scheduledFor) || (blocking.length && (transition === 'publish' || transition === 'schedule') && (!lintOverride || actionNote.trim().length < 20)))} onClick={() => void doTransition()}>Confirm</Button></>}><div className="space-y-3">{transition === 'schedule' ? <label className="block text-sm font-medium">Publish date and time<Input className="mt-1" type="datetime-local" value={scheduledFor} onChange={(e) => setScheduledFor(e.target.value)} /></label> : null}<label className="block text-sm font-medium">Change note<Textarea className="mt-1" value={actionNote} onChange={(e) => setActionNote(e.target.value)} /></label>{blocking.length && (transition === 'publish' || transition === 'schedule') && canPublish ? <Checkbox checked={lintOverride} onChange={(e) => setLintOverride(e.target.checked)} label="Override validation" description="Requires a justification of at least 20 characters in the change note." /> : null}</div></Dialog>
  </main>
}

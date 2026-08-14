import { useEffect, useState, type ReactNode } from 'react'
import { Sparkles } from 'lucide-react'
import { generateMarketingArticleMetadata } from '../../../lib/marketing-content-ai-api'
import type { MarketingArticle, MarketingFinding } from '../../../lib/marketing-content-api'
import { Badge, Button, Checkbox, Disclosure, Field, InlineAlert, Input, Select, Textarea } from '../../ui'
import { commaList, slugify } from './article-editor-utils'

type Option = { id?: string; slug?: string; title?: string; name?: string }
type Props = {
  article: MarketingArticle
  onChange: (patch: Partial<MarketingArticle>) => void
  categories: Option[] | null
  authors: Option[] | null
  knownPaths: string[] | null
  isNew: boolean
  findings?: MarketingFinding[]
  highlightField?: string | null
  canFillWithAI?: boolean
}

function fieldNotice(findings: MarketingFinding[] | undefined, path: string): { error?: string; warning?: string } {
  const finding = findings?.find((item) => item.path === path)
  if (!finding?.message) return {}
  return finding.severity === 'error' ? { error: finding.message } : { warning: finding.message }
}

function fieldHighlight(path: string, highlightField?: string | null): string {
  return highlightField === path ? 'rounded-xl ring-2 ring-accent-solid ring-offset-2 ring-offset-surface-raised' : ''
}

function Group({ title, action, children, collapsible = false, defaultOpen = true }: { title: string; action?: ReactNode; children: ReactNode; collapsible?: boolean; defaultOpen?: boolean }) {
  if (collapsible) {
    return (
      <Disclosure
        title={<span className="text-xs font-semibold uppercase tracking-wide text-fg-muted">{title}</span>}
        defaultOpen={defaultOpen}
      >
        <div className="space-y-4">{children}</div>
      </Disclosure>
    )
  }
  return (
    <section className="rounded-xl border border-border-default bg-surface-raised">
      {action ? (
        <div className="flex items-start justify-between gap-2 px-4 pb-2 pt-3">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-fg-muted">{title}</h3>
          {action}
        </div>
      ) : (
        <h3 className="px-4 pb-2 pt-3 text-xs font-semibold uppercase tracking-wide text-fg-muted">{title}</h3>
      )}
      <div className="space-y-4 px-4 pb-4">{children}</div>
    </section>
  )
}

export function ArticleMetadataPanel({ article, onChange, categories, authors, knownPaths, isNew, findings, highlightField, canFillWithAI = false }: Props) {
  const categoryOptions = categories ?? []
  const authorOptions = authors ?? []
  const pathOptions = knownPaths ?? []
  const description = article.description ?? ''
  const [fillBusy, setFillBusy] = useState(false)
  const [fillError, setFillError] = useState('')
  useEffect(() => {
    if (!highlightField || highlightField === 'title') return
    const root = document.querySelector<HTMLElement>(`[data-metadata-field="${highlightField}"]`)
    if (!root) return
    root.scrollIntoView({ block: 'center', behavior: 'smooth' })
    root.querySelector<HTMLElement>('input, textarea, select')?.focus()
  }, [highlightField])
  const list = (key: 'keywords' | 'relatedTo' | 'roles' | 'segments' | 'citations') => ({
    // Older/imported rows can contain SQL NULL arrays, which Go encodes as
    // JSON null despite the generated client contract declaring string[].
    value: (article[key] ?? []).join(', '),
    onChange: (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => onChange({ [key]: commaList(event.target.value) }),
  })
  const chips = (key: 'keywords' | 'roles' | 'segments') => {
    const values = (article[key] ?? []).filter(Boolean)
    if (!values.length) return null
    return <div className="flex flex-wrap gap-1">{values.slice(0, 8).map((value) => <Badge key={value} tone="neutral">{value}</Badge>)}{values.length > 8 ? <Badge tone="neutral">+{values.length - 8}</Badge> : null}</div>
  }

  async function fillWithAI() {
    if (fillBusy) return
    if (!article.title.trim() && !article.bodyMd.trim()) {
      setFillError('Add a title or article body first.')
      return
    }
    setFillBusy(true)
    setFillError('')
    try {
      const draft = await generateMarketingArticleMetadata({
        kind: article.kind,
        existingTitle: article.title,
        existingBodyMd: article.bodyMd,
      })
      const nextSlug = slugify(draft.slug)
      const autoSlug = !article.publishedAt && (!article.slug || article.slug === slugify(article.title))
      if (!draft.description.trim() && !(autoSlug && nextSlug)) {
        setFillError('No metadata was generated. Add more article content and try again.')
        return
      }
      onChange({
        ...(draft.description.trim() ? { description: draft.description } : {}),
        ...(autoSlug && nextSlug ? { slug: nextSlug } : {}),
      })
    } catch (error) {
      setFillError(error instanceof Error ? error.message : 'Could not fill metadata.')
    } finally {
      setFillBusy(false)
    }
  }

  return <div className="space-y-3">
    <Group title={'Essentials'} action={canFillWithAI ? (
      <Button size="sm" variant="ghost" className="-me-1.5 min-h-6 shrink-0" loading={fillBusy} disabled={fillBusy} onClick={() => void fillWithAI()}>
        <Sparkles className="h-3.5 w-3.5" aria-hidden />
        Fill out with AI
      </Button>
    ) : undefined}>
      {fillError ? <InlineAlert tone="danger">{fillError}</InlineAlert> : null}
      <Field label="Kind" description={isNew ? undefined : 'Locked after creation.'}>
        <Select value={article.kind} disabled={!isNew} onChange={(e) => onChange({ kind: e.target.value as MarketingArticle['kind'] })}><option value="blog">Blog post</option><option value="doc">Help article</option></Select>
      </Field>
      <Field label="Locale" description="Locked after creation. Translations are separate articles.">
        <Input value={article.locale || 'en'} disabled readOnly />
      </Field>
      <Field label="Slug" required description={<span className="font-mono">{article.path || `${article.locale && article.locale !== 'en' ? `/${article.locale}` : ''}${article.kind === 'blog' ? '/blog/' : '/docs/…/'}${article.slug || '…'}`}</span>} warning={article.publishedAt ? 'Changing this creates a redirect.' : undefined}>
        <Input value={article.slug} required pattern="[a-z0-9]+(?:-[a-z0-9]+)*" onChange={(e) => onChange({ slug: e.target.value })} placeholder="my-article" />
      </Field>
      {article.kind === 'doc' ? <Field label="Category" required>
        <Select value={article.categoryId ?? ''} required onChange={(e) => onChange({ categoryId: e.target.value || null })}><option value="">Choose a category</option>{categoryOptions.map((v) => <option key={v.id} value={v.id}>{v.title}</option>)}</Select>
      </Field> : null}
      <Field data-metadata-field="description" htmlFor="article-meta-description" className={fieldHighlight('description', highlightField)} label="Description" description="Shown in search results and social cards." warning={fieldNotice(findings, 'description').warning ?? (description.length > 155 ? 'Close to the 160-character limit.' : undefined)} error={fieldNotice(findings, 'description').error}>
        <Textarea rows={3} maxLength={160} value={description} onChange={(e) => onChange({ description: e.target.value })} placeholder="One or two sentences that summarise the article." />
      </Field>
      <p className="-mt-2 text-end text-xs tabular-nums text-fg-muted">{description.length}/160</p>
    </Group>

    <Group title={'Search & audience'}>
      <Field data-metadata-field="primaryQuestion" htmlFor="article-meta-primaryQuestion" className={fieldHighlight('primaryQuestion', highlightField)} label="Primary question" description="The question this article answers." {...fieldNotice(findings, 'primaryQuestion')}>
        <Input value={article.primaryQuestion ?? ''} onChange={(e) => onChange({ primaryQuestion: e.target.value })} placeholder="How do I…?" />
      </Field>
      <Field data-metadata-field="keywords" htmlFor="article-meta-keywords" className={fieldHighlight('keywords', highlightField)} label="Keywords" description="Comma separated." {...fieldNotice(findings, 'keywords')}>
        <Input {...list('keywords')} placeholder="courses, navigation" />
      </Field>
      {chips('keywords')}
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
        <Field data-metadata-field="cluster" htmlFor="article-meta-cluster" className={fieldHighlight('cluster', highlightField)} label="Cluster" {...fieldNotice(findings, 'cluster')}><Input value={article.cluster} onChange={(e) => onChange({ cluster: e.target.value })} /></Field>
        <Field label="Pillar"><Input value={article.pillar} onChange={(e) => onChange({ pillar: e.target.value })} /></Field>
      </div>
      <Field label="Related paths" description="Comma separated site paths.">
        <><Input {...list('relatedTo')} list="marketing-known-paths" placeholder="/docs/getting-started" /><datalist id="marketing-known-paths">{pathOptions.map((path) => <option key={path} value={path} />)}</datalist></>
      </Field>
      <Field label="Roles"><Input {...list('roles')} placeholder="teacher, administrator" /></Field>
      {chips('roles')}
      <Field label="Segments"><Input {...list('segments')} placeholder="k12, higher-ed" /></Field>
      {chips('segments')}
    </Group>

    <Group title={'People & review'}>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
        <Field data-metadata-field="author" htmlFor="article-meta-author" className={fieldHighlight('author', highlightField)} label="Author" required {...fieldNotice(findings, 'author')}>
          <Select value={article.authorSlug} required onChange={(e) => onChange({ authorSlug: e.target.value })}><option value="">Choose an author</option>{authorOptions.map((v) => <option key={v.slug} value={v.slug}>{v.name}</option>)}</Select>
        </Field>
        <Field label="Reviewer">
          <Select value={article.reviewerSlug ?? ''} onChange={(e) => onChange({ reviewerSlug: e.target.value || null })}><option value="">No reviewer</option>{authorOptions.map((v) => <option key={v.slug} value={v.slug}>{v.name}</option>)}</Select>
        </Field>
      </div>
      <Field label="Review due date">
        <Input type="date" value={article.reviewDueOn?.slice(0, 10) ?? ''} onChange={(e) => onChange({ reviewDueOn: e.target.value ? new Date(`${e.target.value}T12:00:00Z`).toISOString() : null })} />
      </Field>
      <Field label="Verified against" description="Product version or release this content was checked against.">
        <Input value={article.verifiedAgainst} onChange={(e) => onChange({ verifiedAgainst: e.target.value })} />
      </Field>
    </Group>

    <Group title={'Sources & media'}>
      <Field label="Citations" description="Comma separated URLs or references.">
        <Textarea rows={3} {...list('citations')} />
      </Field>
      <Field label="Hero media ID">
        <Input value={article.heroMediaId ?? ''} onChange={(e) => onChange({ heroMediaId: e.target.value || null })} />
      </Field>
    </Group>

    <Group title={'Advanced'} collapsible defaultOpen={false}>
      <Checkbox checked={article.noindex} onChange={(e) => onChange({ noindex: e.target.checked })} label="Exclude from search engines" />
      <Field label="Canonical override" description="Point search engines at a different URL.">
        <Input type="url" value={article.canonicalOverride ?? ''} onChange={(e) => onChange({ canonicalOverride: e.target.value || null })} />
      </Field>
    </Group>
  </div>
}

import { type ReactNode } from 'react'
import type { MarketingArticle } from '../../../lib/marketing-content-api'
import { Badge, Checkbox, Disclosure, Field, Input, Select, Textarea } from '../../ui'
import { commaList } from './article-editor-utils'

type Option = { id?: string; slug?: string; title?: string; name?: string }
type Props = {
  article: MarketingArticle
  onChange: (patch: Partial<MarketingArticle>) => void
  categories: Option[] | null
  authors: Option[] | null
  knownPaths: string[] | null
  isNew: boolean
}

function Group({ title, children, collapsible = false, defaultOpen = true }: { title: string; children: ReactNode; collapsible?: boolean; defaultOpen?: boolean }) {
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
      <h3 className="px-4 pb-2 pt-3 text-xs font-semibold uppercase tracking-wide text-fg-muted">{title}</h3>
      <div className="space-y-4 px-4 pb-4">{children}</div>
    </section>
  )
}

export function ArticleMetadataPanel({ article, onChange, categories, authors, knownPaths, isNew }: Props) {
  const categoryOptions = categories ?? []
  const authorOptions = authors ?? []
  const pathOptions = knownPaths ?? []
  const description = article.description ?? ''
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

  return <div className="space-y-3">
    <Group title={'Essentials'}>
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
      <Field label="Description" required description="Shown in search results and social cards. Required to publish." warning={description.length > 155 ? 'Close to the 160-character limit.' : undefined}>
        <Textarea rows={3} maxLength={160} value={description} required onChange={(e) => onChange({ description: e.target.value })} placeholder="One or two sentences that summarise the article." />
      </Field>
      <p className="-mt-2 text-end text-xs tabular-nums text-fg-muted">{description.length}/160</p>
    </Group>

    <Group title={'Search & audience'}>
      <Field label="Primary question" required description="The question this article answers. Required to publish.">
        <Input value={article.primaryQuestion ?? ''} required onChange={(e) => onChange({ primaryQuestion: e.target.value })} placeholder="How do I…?" />
      </Field>
      <Field label="Keywords" required description="Comma separated. At least one is required to publish.">
        <Input {...list('keywords')} required placeholder="courses, navigation" />
      </Field>
      {chips('keywords')}
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
        <Field label="Cluster" required description="Required to publish."><Input value={article.cluster} required onChange={(e) => onChange({ cluster: e.target.value })} /></Field>
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
        <Field label="Author" required>
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

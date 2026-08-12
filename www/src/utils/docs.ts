import { contentSource, type ContentArticle } from '../lib/content-source'

export type DocArticle = ContentArticle
export type DocArticleMeta = Omit<DocArticle, 'content' | 'html'>
export const allArticles: DocArticle[] = contentSource.listArticles().filter(a => a.kind === 'doc').sort((a, b) => b.date.localeCompare(a.date))
export const allArticleMeta: DocArticleMeta[] = allArticles.map(({ content: _content, html: _html, ...meta }) => meta)
export function getArticle(slug: string) { return allArticles.find(article => article.slug === slug) }
export function getCategorizedArticle(category: string, slug: string) { return allArticles.find(article => article.category === category && article.slug === slug) }
export function articlePath(article: Pick<DocArticle, 'category' | 'slug'>) { return `/docs/${article.category}/${article.slug}` }
export function formatDate(iso: string) { return iso ? new Date(`${iso}T00:00:00`).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' }) : '' }
export function articlesByAuthor(authorSlug: string) { return allArticles.filter(article => article.author === authorSlug) }
export function articlesMetaByAuthor(authorSlug: string) { return allArticleMeta.filter(article => article.author === authorSlug) }

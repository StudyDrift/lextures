import { contentSource, type ContentArticle } from '../lib/content-source'

export type BlogPost = ContentArticle
export type BlogPostMeta = Omit<BlogPost, 'content' | 'html'>
export const allPosts: BlogPost[] = contentSource.listArticles().filter(a => a.kind === 'blog').sort((a, b) => b.date.localeCompare(a.date))
export const allPostMeta: BlogPostMeta[] = allPosts.map(({ content: _content, html: _html, ...meta }) => meta)
export function getPost(slug: string) { return allPosts.find(post => post.slug === slug) }
export function formatDate(iso: string) { return iso ? new Date(`${iso}T00:00:00`).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' }) : '' }
export function postWordCount(post: BlogPost) { return post.content.replace(/```[\s\S]*?```/g, ' ').replace(/[#>*_`[\]()!-]/g, ' ').split(/\s+/).filter(Boolean).length }
export function postsByAuthor(authorSlug: string) { return allPosts.filter(post => post.author === authorSlug) }
export function postsMetaByAuthor(authorSlug: string) { return allPostMeta.filter(post => post.author === authorSlug) }
